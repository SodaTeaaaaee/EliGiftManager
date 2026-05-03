package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WaveController struct {
	db     *gorm.DB
	appCtx context.Context
}

func (c *WaveController) SetContext(ctx context.Context) { c.appCtx = ctx }

func (c *WaveController) CreateWave(name string) (model.Wave, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Wave{}, fmt.Errorf("create wave failed: name is required")
	}
	db := c.db
	var err error
	wave := model.Wave{Name: name, Status: model.WaveStatusDraft}
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
	db := c.db
	return db.Transaction(func(tx *gorm.DB) error {
		// Manually cascade: SQLite FK constraints may be stale from previous
		// migrations, so explicitly delete dependents before the wave itself.
		if err := tx.Where("wave_id = ?", waveID).Delete(&model.DispatchRecord{}).Error; err != nil {
			return fmt.Errorf("delete wave failed: clean dispatch records: %w", err)
		}
		if err := tx.Where("wave_id = ?", waveID).Delete(&model.WaveMember{}).Error; err != nil {
			return fmt.Errorf("delete wave failed: clean wave members: %w", err)
		}
		// Unlink products (no FK, just set wave_id to NULL).
		if err := tx.Model(&model.Product{}).Where("wave_id = ?", waveID).Update("wave_id", nil).Error; err != nil {
			return fmt.Errorf("delete wave failed: unlink products: %w", err)
		}
		result := tx.Delete(&model.Wave{}, waveID)
		if result.Error != nil {
			return fmt.Errorf("delete wave failed: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("delete wave failed: wave not found")
		}
		return nil
	})
}

func (c *WaveController) ListWaves(status string) ([]WaveItem, error) {
	db := c.db
	return queryWaves(db, 100, status)
}

func (c *WaveController) ImportToWave(waveID uint, csvPath string, templateID uint) error {
	db := c.db
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
		if templateMeta.Format == model.TemplateFormatZIP {
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
	default:
		err = fmt.Errorf("template type %q cannot import", template.Type)
	}
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}
	return nil
}

func (c *WaveController) ImportDispatchWave(waveID uint, csvPath string, importTemplateID uint, setDefault bool) error {
	db := c.db
	var template model.TemplateConfig
	if err := db.First(&template, importTemplateID).Error; err != nil {
		return fmt.Errorf("import dispatch wave failed: template not found: %w", err)
	}
	_, err := service.ImportDispatchWave(db, waveID, csvPath, template, setDefault)
	if err != nil {
		return fmt.Errorf("import dispatch wave failed: %w", err)
	}
	return nil
}

func (c *WaveController) ListDispatchRecords(waveID uint) ([]DispatchRecordItem, error) {
	db := c.db
	return queryDispatchRecords(db, waveID, 500)
}

// ReconcileWave delegates to the service-layer reconciler.  It is a thin
// controller wrapper so the Wails binding surface stays stable.
func (c *WaveController) ReconcileWave(waveID uint) (int, error) {
	return service.ReconcileWave(c.db, waveID)
}

func (c *WaveController) AllocateByTags(waveID uint) (int, error) {
	return c.ReconcileWave(waveID)
}

func (c *WaveController) SetDispatchAddress(waveID, memberID, addressID uint) error {
	db := c.db
	return db.Model(&model.DispatchRecord{}).Where("wave_id = ? AND member_id = ?", waveID, memberID).Update("member_address_id", addressID).Error
}

func (c *WaveController) UpdateDispatchQuantity(dispatchID uint, quantity int) error {
	db := c.db
	var record model.DispatchRecord
	if err := db.First(&record, dispatchID).Error; err != nil {
		return fmt.Errorf("dispatch record not found: %w", err)
	}
	// Look up wave_member by (waveID, memberID) for the new user-tag logic.
	var wm model.WaveMember
	if err := db.Where("wave_id = ? AND member_id = ?", record.WaveID, record.MemberID).First(&wm).Error; err != nil {
		return fmt.Errorf("member (id=%d) not found in wave (id=%d): %w", record.MemberID, record.WaveID, err)
	}
	return c.syncUserTagForTargetQuantity(record.WaveID, wm.ID, record.ProductID, quantity)
}

// SyncUserTagForTargetQuantity is a Wails-bound wrapper that accepts memberID for
// frontend compatibility.  It resolves the wave_member internally and delegates
// to the waveMemberID-based engine.
func (c *WaveController) SyncUserTagForTargetQuantity(waveID, memberID, productID uint, targetQty int) error {
	db := c.db
	var wm model.WaveMember
	if err := db.Where("wave_id = ? AND member_id = ?", waveID, memberID).First(&wm).Error; err != nil {
		return fmt.Errorf("member (id=%d) not found in wave (id=%d): %w", memberID, waveID, err)
	}
	return c.syncUserTagForTargetQuantity(waveID, wm.ID, productID, targetQty)
}

// syncUserTagForTargetQuantity is the waveMemberID-based engine for adjusting a
// single product's user-tag quantity to reach targetQty, accounting for level-tag
// base quantity.  It upserts or deletes the user tag on (product_id, wave_member_id,
// tag_type), then triggers ReconcileWave.
func (c *WaveController) syncUserTagForTargetQuantity(waveID, waveMemberID, productID uint, targetQty int) error {
	db := c.db

	// 1. Verify wave_member exists and belongs to this wave.
	var wm model.WaveMember
	if err := db.Where("id = ? AND wave_id = ?", waveMemberID, waveID).First(&wm).Error; err != nil {
		return fmt.Errorf("wave member (id=%d) not found in wave (id=%d): %w", waveMemberID, waveID, err)
	}

	// 2. Verify product belongs to this wave.
	var product model.Product
	if err := db.Where("id = ? AND wave_id = ?", productID, waveID).First(&product).Error; err != nil {
		return fmt.Errorf("product (id=%d) not found in wave (id=%d): %w", productID, waveID, err)
	}

	// 3. Calculate baseQty from level tags matching this member's platform + gift_level.
	var levelTags []model.ProductTag
	db.Where("product_id = ? AND platform = ? AND tag_type = ?", productID, wm.Platform, model.TagTypeLevel).Find(&levelTags)

	baseQty := 0
	for _, tag := range levelTags {
		if tag.TagName == wm.GiftLevel {
			baseQty += tag.Quantity
		}
	}

	neededUserQty := targetQty - baseQty

	// 4. Upsert or delete user tag.
	// neededUserQty == 0 → no override needed, remove user tag.
	// neededUserQty > 0 or < 0 → upsert (allows negative user tags to reduce allocation).
	if neededUserQty == 0 {
		db.Where("product_id = ? AND wave_member_id = ? AND tag_type = ?",
			productID, waveMemberID, model.TagTypeUser).Delete(&model.ProductTag{})
	} else {
		userTag := model.ProductTag{
			ProductID:    productID,
			Platform:     wm.Platform,
			TagName:      wm.PlatformUID,
			TagType:      model.TagTypeUser,
			Quantity:     neededUserQty,
			WaveMemberID: &waveMemberID,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "product_id"}, {Name: "wave_member_id"}, {Name: "tag_type"}},
			DoUpdates: clause.AssignmentColumns([]string{"quantity", "platform", "tag_name", "updated_at"}),
		}).Create(&userTag).Error; err != nil {
			return fmt.Errorf("upsert user tag failed: %w", err)
		}
	}

	_, err := service.ReconcileWave(c.db, waveID)
	return err
}

// ReallocateWave is a deprecated wrapper kept for Wails binding compatibility.
// New code should call ReconcileWave directly; tag operations auto-trigger it.
func (c *WaveController) ReallocateWave(waveID uint) error {
	_, err := c.ReconcileWave(waveID)
	return err
}

func (c *WaveController) AddDispatchToMember(waveID, memberID, productID uint, quantity int) error {
	if quantity < 1 {
		quantity = 1
	}
	db := c.db
	var wm model.WaveMember
	if err := db.Where("wave_id = ? AND member_id = ?", waveID, memberID).First(&wm).Error; err != nil {
		return fmt.Errorf("member (id=%d) not found in wave (id=%d): %w", memberID, waveID, err)
	}
	return c.syncUserTagForTargetQuantity(waveID, wm.ID, productID, quantity)
}

func (c *WaveController) RemoveDispatchFromMember(dispatchID uint) error {
	db := c.db
	var record model.DispatchRecord
	if err := db.First(&record, dispatchID).Error; err != nil {
		return fmt.Errorf("dispatch record not found: %w", err)
	}
	var wm model.WaveMember
	if err := db.Where("wave_id = ? AND member_id = ?", record.WaveID, record.MemberID).First(&wm).Error; err != nil {
		return fmt.Errorf("member (id=%d) not found in wave (id=%d): %w", record.MemberID, record.WaveID, err)
	}
	return c.syncUserTagForTargetQuantity(record.WaveID, wm.ID, record.ProductID, 0)
}

func (c *WaveController) RemoveProductFromWave(waveID, productID uint) error {
	db := c.db
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
	db := c.db
	updated, skipped, err := service.BindDefaultAddresses(db, waveID)
	if err != nil {
		return nil, fmt.Errorf("bind default addresses failed: %w", err)
	}
	return map[string]int64{"updated": int64(updated), "skipped": int64(skipped)}, nil
}

func (c *WaveController) ExportOrderCSV(waveID uint, exportTemplateID uint) (string, error) {
	db := c.db

	var wave model.Wave
	if err := db.First(&wave, waveID).Error; err != nil {
		return "", fmt.Errorf("export order CSV failed: wave not found: %w", err)
	}

	var template model.TemplateConfig
	if err := db.First(&template, exportTemplateID).Error; err != nil {
		return "", fmt.Errorf("export order CSV failed: export template not found: %w", err)
	}

	path := filepath.Join(os.TempDir(), fmt.Sprintf("eligift-factory-order-%d-%s.csv", waveID, time.Now().Format("20060102150405")))
	if c.appCtx != nil {
		selected, dialogErr := wailsruntime.SaveFileDialog(c.appCtx, wailsruntime.SaveDialogOptions{DefaultFilename: filepath.Base(path)})
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
	db := c.db

	total, missing, err := service.ExportWavePreview(db, waveID)
	if err != nil {
		return nil, fmt.Errorf("preview export failed: %w", err)
	}

	return map[string]int64{"totalRecords": int64(total), "missingAddressCount": int64(missing)}, nil
}
