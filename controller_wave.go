package main

import (
	"encoding/json"
	"errors"
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
	"gorm.io/gorm/clause"
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

// ReconcileWave 根据 ProductTag 和 WaveMember 重新计算整个波次的 DispatchRecord。
// WaveMember 自包含快照字段（Platform/GiftLevel），不再需要 JOIN members。
// user tag 通过 WaveMemberID 直接定位；level tag 通过 wm.Platform + wm.GiftLevel 匹配。
// 比对期望状态与实际状态，通过 INSERT/UPDATE/DELETE 抹平差异。幂等。
func (c *WaveController) ReconcileWave(waveID uint) (int, error) {
	db := c.db()
	if db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// 1. Load WaveMembers (self-contained snapshot, no Preload needed).
	var waveMembers []model.WaveMember
	if err := db.Where("wave_id = ?", waveID).Find(&waveMembers).Error; err != nil {
		return 0, fmt.Errorf("load wave members failed: %w", err)
	}

	// Build wmID -> memberID lookup (DispatchRecord still uses MemberID).
	wmToMember := make(map[uint]uint, len(waveMembers))
	for _, wm := range waveMembers {
		wmToMember[wm.ID] = wm.MemberID
	}

	// 2. Load products with tags for this wave.
	var products []model.Product
	if err := db.Where("wave_id = ?", waveID).Preload("Tags").Find(&products).Error; err != nil {
		return 0, fmt.Errorf("load products failed: %w", err)
	}

	// 3. Calculate expected state: productID -> memberID -> expectedQuantity.
	expectedState := make(map[uint]map[uint]int)
	for _, p := range products {
		expectedState[p.ID] = make(map[uint]int)
	}

	for _, p := range products {
		for _, tag := range p.Tags {
			if tag.TagType == "user" {
				// User tag: match via WaveMemberID directly.
				if tag.WaveMemberID == nil {
					continue
				}
				memberID, ok := wmToMember[*tag.WaveMemberID]
				if !ok {
					continue // wave member not in current wave (stale tag).
				}
				expectedState[p.ID][memberID] += tag.Quantity
			} else {
				// Level tag: match wave_members by platform + gift_level (simple string compare).
				for _, wm := range waveMembers {
					if wm.Platform == tag.Platform && wm.GiftLevel == tag.TagName {
						expectedState[p.ID][wm.MemberID] += tag.Quantity
					}
				}
			}
		}
	}

	allocatedCount := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		var validDispatchIDs []uint

		for productID, memberMap := range expectedState {
			for memberID, expectedQty := range memberMap {
				if expectedQty > 0 {
					var record model.DispatchRecord
					err := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?", waveID, memberID, productID).
						First(&record).Error
					if errors.Is(err, gorm.ErrRecordNotFound) {
						record = model.DispatchRecord{
							WaveID:    waveID,
							MemberID:  memberID,
							ProductID: productID,
							Quantity:  expectedQty,
							Status:    "draft",
						}
						if createErr := tx.Create(&record).Error; createErr != nil {
							return fmt.Errorf("create dispatch record (member=%d, product=%d): %w", memberID, productID, createErr)
						}
					} else if err != nil {
						return fmt.Errorf("lookup dispatch record (member=%d, product=%d): %w", memberID, productID, err)
					} else if record.Quantity != expectedQty {
						if updateErr := tx.Model(&record).Update("quantity", expectedQty).Error; updateErr != nil {
							return fmt.Errorf("update dispatch quantity (id=%d): %w", record.ID, updateErr)
						}
					}
					allocatedCount++
					validDispatchIDs = append(validDispatchIDs, record.ID)
				}
			}
		}

		// GC: delete DispatchRecords not in expected state.
		if len(validDispatchIDs) > 0 {
			if err := tx.Where("wave_id = ? AND id NOT IN ?", waveID, validDispatchIDs).Delete(&model.DispatchRecord{}).Error; err != nil {
				return fmt.Errorf("cleanup orphaned dispatch records failed: %w", err)
			}
		} else {
			if err := tx.Where("wave_id = ?", waveID).Delete(&model.DispatchRecord{}).Error; err != nil {
				return fmt.Errorf("clear all dispatch records failed: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("reconcile wave failed: %w", err)
	}
	return allocatedCount, nil
}

func (c *WaveController) AllocateByTags(waveID uint) (int, error) {
	return c.ReconcileWave(waveID)
}

func (c *WaveController) SetDispatchAddress(waveID, memberID, addressID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("set dispatch address failed: database not available")
	}
	return db.Model(&model.DispatchRecord{}).Where("wave_id = ? AND member_id = ?", waveID, memberID).Update("member_address_id", addressID).Error
}

func (c *WaveController) UpdateDispatchQuantity(dispatchID uint, quantity int) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
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
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
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
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}

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
	db.Where("product_id = ? AND platform = ? AND tag_type = 'level'", productID, wm.Platform).Find(&levelTags)

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
		db.Where("product_id = ? AND wave_member_id = ? AND tag_type = 'user'",
			productID, waveMemberID).Delete(&model.ProductTag{})
	} else {
		userTag := model.ProductTag{
			ProductID:    productID,
			Platform:     wm.Platform,
			TagName:      wm.PlatformUID,
			TagType:      "user",
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

	_, err := c.ReconcileWave(waveID)
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
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	var wm model.WaveMember
	if err := db.Where("wave_id = ? AND member_id = ?", waveID, memberID).First(&wm).Error; err != nil {
		return fmt.Errorf("member (id=%d) not found in wave (id=%d): %w", memberID, waveID, err)
	}
	return c.syncUserTagForTargetQuantity(waveID, wm.ID, productID, quantity)
}

func (c *WaveController) RemoveDispatchFromMember(dispatchID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
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
