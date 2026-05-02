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

	// Load all ProductTags for products belonging to this wave.
	var allTags []model.ProductTag
	if err := db.Model(&model.ProductTag{}).
		Joins("JOIN products ON products.id = product_tags.product_id").
		Where("products.wave_id = ?", waveID).
		Find(&allTags).Error; err != nil {
		return 0, fmt.Errorf("allocate by tags failed: load product tags: %w", err)
	}
	if len(allTags) == 0 {
		return 0, fmt.Errorf("allocate by tags failed: no product tags found for this wave")
	}

	// Group tags by product ID.
	tagsByProduct := make(map[uint][]model.ProductTag)
	for _, tag := range allTags {
		tagsByProduct[tag.ProductID] = append(tagsByProduct[tag.ProductID], tag)
	}

	allocatedCount := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for productID, productTagList := range tagsByProduct {
			// Aggregate net quantity per member for this product across all its tags.
			memberQty := make(map[uint]int)

			for _, tag := range productTagList {
				switch tag.TagType {
				case "user":
					var member model.Member
					if err := tx.Where("platform = ? AND platform_uid = ?",
						tag.Platform, tag.TagName).First(&member).Error; err != nil {
						if err == gorm.ErrRecordNotFound {
							continue
						}
						return fmt.Errorf("lookup member for user tag %s/%s: %w", tag.Platform, tag.TagName, err)
					}
					memberQty[member.ID] += tag.Quantity
				default: // "level" or any unrecognised TagType falls back to level matching
					var members []model.Member
					if err := tx.Where("platform = ? AND extra_data LIKE ?",
						tag.Platform, fmt.Sprintf(`%%"giftLevel":%%%s%%`, tag.TagName)).
						Find(&members).Error; err != nil {
						return fmt.Errorf("query members for level tag %s/%s: %w", tag.Platform, tag.TagName, err)
					}
					for _, m := range members {
						memberQty[m.ID] += tag.Quantity
					}
				}
			}

			for memberID, netQty := range memberQty {
				if netQty > 0 {
					var record model.DispatchRecord
					result := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?",
						waveID, memberID, productID).
						FirstOrCreate(&record, model.DispatchRecord{
							WaveID:    waveID,
							MemberID:  memberID,
							ProductID: productID,
							Quantity:  netQty,
							Status:    "draft",
						})
					if result.Error != nil {
						return fmt.Errorf("upsert dispatch record: %w", result.Error)
					}
					if result.RowsAffected > 0 {
						allocatedCount++
					}
					if record.Quantity != netQty {
						if err := tx.Model(&record).Update("quantity", netQty).Error; err != nil {
							return fmt.Errorf("update dispatch quantity: %w", err)
						}
					}
				} else {
					if err := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?",
						waveID, memberID, productID).
						Delete(&model.DispatchRecord{}).Error; err != nil {
						return fmt.Errorf("delete dispatch record: %w", err)
					}
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

func (c *WaveController) AllocateSingleTag(waveID uint, platform, tagName, tagType string) (int, error) {
	db := c.db()
	if db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Load ProductTags matching this specific tag, only for products in the wave.
	var tags []model.ProductTag
	if err := db.Model(&model.ProductTag{}).
		Joins("JOIN products ON products.id = product_tags.product_id").
		Where("products.wave_id = ? AND product_tags.platform = ? AND product_tags.tag_name = ? AND product_tags.tag_type = ?",
			waveID, platform, tagName, tagType).
		Find(&tags).Error; err != nil {
		return 0, fmt.Errorf("allocate single tag failed: %w", err)
	}
	if len(tags) == 0 {
		return 0, nil
	}

	allocated := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, tag := range tags {
			switch tag.TagType {
			case "user":
				var member model.Member
				if err := tx.Where("platform = ? AND platform_uid = ?",
					tag.Platform, tag.TagName).First(&member).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						continue
					}
					return err
				}
				var record model.DispatchRecord
				result := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?",
					waveID, member.ID, tag.ProductID).
					FirstOrCreate(&record, model.DispatchRecord{
						WaveID: waveID, MemberID: member.ID, ProductID: tag.ProductID,
						Quantity: tag.Quantity, Status: "draft",
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					allocated++
				}
				if record.Quantity != tag.Quantity {
					if err := tx.Model(&record).Update("quantity", tag.Quantity).Error; err != nil {
						return err
					}
				}
			default: // "level"
				var members []model.Member
				if err := tx.Where("platform = ? AND extra_data LIKE ?",
					tag.Platform, fmt.Sprintf(`%%"giftLevel":%%%s%%`, tag.TagName)).
					Find(&members).Error; err != nil {
					return err
				}
				for _, member := range members {
					var record model.DispatchRecord
					result := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?",
						waveID, member.ID, tag.ProductID).
						FirstOrCreate(&record, model.DispatchRecord{
							WaveID: waveID, MemberID: member.ID, ProductID: tag.ProductID,
							Quantity: tag.Quantity, Status: "draft",
						})
					if result.Error != nil {
						return result.Error
					}
					if result.RowsAffected > 0 {
						allocated++
					}
					if record.Quantity != tag.Quantity {
						if err := tx.Model(&record).Update("quantity", tag.Quantity).Error; err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	})

	return allocated, err
}

func (c *WaveController) SetDispatchAddress(waveID, memberID, addressID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("set dispatch address failed: database not available")
	}
	return db.Model(&model.DispatchRecord{}).Where("wave_id = ? AND member_id = ?", waveID, memberID).Update("member_address_id", addressID).Error
}

func (c *WaveController) RemoveSingleTag(waveID uint, platform, tagName string) (int, error) {
	db := c.db()
	if db == nil {
		return 0, fmt.Errorf("database not available")
	}

	var tags []model.ProductTag
	if err := db.Where("platform = ? AND tag_name = ?", platform, tagName).Find(&tags).Error; err != nil {
		return 0, err
	}

	if len(tags) == 0 {
		return 0, nil
	}

	var productIDs []uint
	for _, t := range tags {
		productIDs = append(productIDs, t.ProductID)
	}

	result := db.Where("wave_id = ? AND product_id IN ?", waveID, productIDs).Delete(&model.DispatchRecord{})
	return int(result.RowsAffected), result.Error
}

func (c *WaveController) UpdateDispatchQuantity(dispatchID uint, quantity int) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	r := db.Model(&model.DispatchRecord{}).Where("id = ?", dispatchID).Update("quantity", quantity)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return fmt.Errorf("dispatch record not found")
	}
	return nil
}

func (c *WaveController) ReallocateWave(waveID uint) error {
	_, err := c.AllocateByTags(waveID)
	return err
}

func (c *WaveController) AddDispatchToMember(waveID, memberID, productID uint, quantity int) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	if quantity < 1 {
		quantity = 1
	}
	var cnt int64
	db.Model(&model.DispatchRecord{}).Where("wave_id = ? AND member_id = ? AND product_id = ?", waveID, memberID, productID).Count(&cnt)
	if cnt > 0 {
		return fmt.Errorf("this product is already assigned to this member")
	}
	return db.Create(&model.DispatchRecord{WaveID: waveID, MemberID: memberID, ProductID: productID, Quantity: quantity, Status: "draft"}).Error
}

func (c *WaveController) RemoveDispatchFromMember(dispatchID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	r := db.Delete(&model.DispatchRecord{}, dispatchID)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return fmt.Errorf("dispatch record not found")
	}
	return nil
}

func (c *WaveController) RemoveProductFromWave(waveID, productID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var product model.Product
		if err := tx.Where("id = ? AND wave_id = ?", productID, waveID).First(&product).Error; err != nil {
			return fmt.Errorf("product not found in this wave: %w", err)
		}
		if err := tx.Model(&product).Update("wave_id", nil).Error; err != nil {
			return fmt.Errorf("remove product from wave failed: %w", err)
		}
		if err := tx.Where("wave_id = ? AND product_id = ?", waveID, productID).Delete(&model.DispatchRecord{}).Error; err != nil {
			return fmt.Errorf("clean dispatch records failed: %w", err)
		}
		return nil
	})
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
