package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type WaveController struct{}

func (c *WaveController) db() *gorm.DB { return dbpkg.GetDB() }

func (c *WaveController) CreateWave(name string) (model.Wave, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Wave{}, fmt.Errorf("create wave failed: name is required")
	}
	db := c.db()
	if db == nil {
		return model.Wave{}, fmt.Errorf("create wave failed: database not available")
	}
	var err error
	wave := model.Wave{Name: name, Status: "draft"}
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Transaction(func(tx *gorm.DB) error {
			prefix := time.Now().Format("TASK-20060102")
			var count int64
			if err := tx.Model(&model.Wave{}).Where("wave_no LIKE ?", prefix+"-%").Count(&count).Error; err != nil {
				return err
			}
			wave.WaveNo = fmt.Sprintf("%s-%03d", prefix, count+1)
			return tx.Create(&wave).Error
		})
		if err == nil {
			return wave, nil
		}
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") || attempt == maxRetries-1 {
			return model.Wave{}, fmt.Errorf("create wave failed: %w", err)
		}
	}
	return model.Wave{}, fmt.Errorf("create wave failed: max retries exceeded")
}

func (c *WaveController) DeleteWave(waveID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("delete wave failed: database not available")
	}
	result := db.Delete(&model.Wave{}, waveID)
	if result.Error != nil {
		return fmt.Errorf("delete wave failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete wave failed: wave not found")
	}
	return nil
}

func (c *WaveController) ListWaves(status string) ([]WaveItem, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("list waves failed: database not available")
	}
	return queryWaves(db, 100, status)
}

func (c *WaveController) ImportToWave(waveID uint, csvPath string, templateID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("import failed: database not available")
	}
	var err error
	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return fmt.Errorf("import failed: wave not found: %w", err)
	}
	var template model.TemplateConfig
	if err := db.First(&template, templateID).Error; err != nil {
		return fmt.Errorf("import failed: template not found: %w", err)
	}
	switch template.Type {
	case model.TemplateTypeImportMember:
		_, err = service.ImportMembersFromCSV(db, csvPath, template)
	case model.TemplateTypeImportProduct:
		var templateMeta struct {
			Format   string `json:"format"`
			ImageDir string `json:"imageDir"`
		}
		json.Unmarshal([]byte(template.MappingRules), &templateMeta)

		var products []model.Product
		if templateMeta.Format == "zip" {
			var extractDir string
			products, extractDir, err = service.ParseProductZIP(csvPath, template)
			if extractDir != "" {
				defer os.RemoveAll(extractDir)
			}
			if err == nil {
				err = db.Transaction(func(tx *gorm.DB) error {
					for i := range products {
						if products[i].ExtraData == "" {
							products[i].ExtraData = "{}"
						}
						products[i].WaveID = &wave.ID
						if delErr := tx.Where("platform = ? AND factory_sku = ?", products[i].Platform, products[i].FactorySKU).Delete(&model.Product{}).Error; delErr != nil {
							return delErr
						}
					}
					if len(products) > 0 {
						if createErr := tx.CreateInBatches(&products, 100).Error; createErr != nil {
							return createErr
						}
					}
					return nil
				})
				if err == nil {
					_, err = service.ProcessCoverImages(db, extractDir, "")
				}
			}
		} else {
			products, err = service.ParseProductCSV(csvPath, template)
			if err == nil {
				err = db.Transaction(func(tx *gorm.DB) error {
					for i := range products {
						products[i].Platform = template.Platform
						if products[i].ExtraData == "" {
							products[i].ExtraData = "{}"
						}
						products[i].WaveID = &wave.ID
						if delErr := tx.Where("platform = ? AND factory_sku = ?", products[i].Platform, products[i].FactorySKU).Delete(&model.Product{}).Error; delErr != nil {
							return delErr
						}
					}
					if len(products) > 0 {
						if createErr := tx.CreateInBatches(&products, 100).Error; createErr != nil {
							return createErr
						}
					}
					return nil
				})
			}
		}
	case model.TemplateTypeImportDispatchRecord:
		var records []model.DispatchRecord
		records, err = service.ParseDispatchRecordCSV(csvPath, template)
		if err == nil {
			err = db.Transaction(func(tx *gorm.DB) error {
				for i := range records {
					records[i].WaveID = wave.ID
					if err := tx.Create(&records[i]).Error; err != nil {
						return err
					}
				}
				return nil
			})
		}
	default:
		err = fmt.Errorf("template type %q cannot import", template.Type)
	}
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}
	return nil
}

func (c *WaveController) ImportDispatchWave(waveID uint, csvPath string, importTemplateID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("import dispatch wave failed: database not available")
	}
	var template model.TemplateConfig
	if err := db.First(&template, importTemplateID).Error; err != nil {
		return fmt.Errorf("import dispatch wave failed: template not found: %w", err)
	}
	_, err := service.ImportDispatchWave(db, waveID, csvPath, template)
	if err != nil {
		return fmt.Errorf("import dispatch wave failed: %w", err)
	}
	return nil
}

func (c *WaveController) ListDispatchRecords(waveID uint) ([]DispatchRecordItem, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return queryDispatchRecords(db, waveID, 500)
}

func (c *WaveController) AllocateByTags(waveID uint) (int, error) {
	db := c.db()
	if db == nil {
		return 0, fmt.Errorf("allocate by tags failed: database not available")
	}

	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return 0, fmt.Errorf("allocate by tags failed: wave not found: %w", err)
	}

	type levelTagEntry struct {
		Platform string `json:"platform"`
		TagName  string `json:"tagName"`
	}
	var levelTags []levelTagEntry
	if err := json.Unmarshal([]byte(wave.LevelTags), &levelTags); err != nil || len(levelTags) == 0 {
		return 0, fmt.Errorf("allocate by tags failed: wave has no level tags — import member data first")
	}

	allocatedCount := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, lt := range levelTags {
			var members []model.Member
			if err := tx.Where("platform = ? AND extra_data LIKE ?",
				lt.Platform, fmt.Sprintf(`%%"giftLevel":%%%s%%`, lt.TagName)).
				Find(&members).Error; err != nil {
				return fmt.Errorf("query members for %s/%s failed: %w", lt.Platform, lt.TagName, err)
			}
			var tags []model.ProductTag
			if err := tx.Where("platform = ? AND tag_name = ?", lt.Platform, lt.TagName).
				Find(&tags).Error; err != nil {
				return fmt.Errorf("lookup product tags for %s/%s failed: %w", lt.Platform, lt.TagName, err)
			}
			if len(tags) == 0 {
				continue
			}
			for _, member := range members {
				for _, tag := range tags {
					var cnt int64
					if err := tx.Model(&model.DispatchRecord{}).
						Where("wave_id = ? AND member_id = ? AND product_id = ?", waveID, member.ID, tag.ProductID).
						Count(&cnt).Error; err != nil {
						return err
					}
					if cnt > 0 {
						continue
					}
					record := model.DispatchRecord{WaveID: waveID, MemberID: member.ID, ProductID: tag.ProductID, Quantity: 1, Status: "draft"}
					if err := tx.Create(&record).Error; err != nil {
						return err
					}
					allocatedCount++
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("allocate by tags failed: %w", err)
	}
	return allocatedCount, nil
}

func (c *WaveController) BindDefaultAddresses(waveID uint) (map[string]int64, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("bind default addresses failed: database not available")
	}
	updated, skipped, err := service.BindDefaultAddresses(db, waveID)
	if err != nil {
		return nil, fmt.Errorf("bind default addresses failed: %w", err)
	}
	return map[string]int64{"updated": int64(updated), "skipped": int64(skipped)}, nil
}

func (c *WaveController) ExportOrderCSV(waveID uint, exportTemplateID uint) (string, error) {
	db := c.db()
	if db == nil {
		return "", fmt.Errorf("export order CSV failed: database not available")
	}

	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return "", fmt.Errorf("export order CSV failed: wave not found: %w", err)
	}

	var template model.TemplateConfig
	if err := db.First(&template, exportTemplateID).Error; err != nil {
		return "", fmt.Errorf("export order CSV failed: export template not found: %w", err)
	}

	path := filepath.Join(os.TempDir(), fmt.Sprintf("eligift-factory-order-%d-%s.csv", waveID, time.Now().Format("20060102150405")))
	if controllerCtx != nil {
		selected, dialogErr := wailsruntime.SaveFileDialog(controllerCtx, wailsruntime.SaveDialogOptions{DefaultFilename: filepath.Base(path)})
		if dialogErr != nil {
			return "", fmt.Errorf("export order CSV failed: %w", dialogErr)
		}
		if selected == "" {
			return "", fmt.Errorf("export canceled")
		}
		path = selected
	}

	if err := service.ExportOrderCSV(db, waveID, path, template); err != nil {
		return "", fmt.Errorf("export order CSV failed: %w", err)
	}

	return path, nil
}

func (c *WaveController) PreviewExport(waveID uint) (map[string]int64, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("preview export failed: database not available")
	}

	total, missing, err := service.ExportWavePreview(db, waveID)
	if err != nil {
		return nil, fmt.Errorf("preview export failed: %w", err)
	}

	return map[string]int64{"totalRecords": int64(total), "missingAddressCount": int64(missing)}, nil
}
