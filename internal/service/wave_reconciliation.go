package service

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// ReconcileWave 根据 ProductTag 和 WaveMember 重新计算整个波次的 DispatchRecord。
// WaveMember 自包含快照字段（Platform/GiftLevel），不再需要 JOIN members。
// user tag 通过 WaveMemberID 直接定位；level tag 通过 wm.Platform + wm.GiftLevel 匹配。
// 比对期望状态与实际状态，通过 INSERT/UPDATE/DELETE 抹平差异。幂等。
func ReconcileWave(db *gorm.DB, waveID uint) (int, error) {
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
			if tag.TagType == model.TagTypeUser {
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
					var records []model.DispatchRecord
					if err := tx.Where("wave_id = ? AND member_id = ? AND product_id = ?", waveID, memberID, productID).
						Limit(1).Find(&records).Error; err != nil {
						return fmt.Errorf("lookup dispatch record (member=%d, product=%d): %w", memberID, productID, err)
					}
					var record model.DispatchRecord
					if len(records) == 0 {
						record = model.DispatchRecord{
							WaveID:    waveID,
							MemberID:  memberID,
							ProductID: productID,
							Quantity:  expectedQty,
							Status:    model.DispatchStatusDraft,
						}
						if createErr := tx.Create(&record).Error; createErr != nil {
							return fmt.Errorf("create dispatch record (member=%d, product=%d): %w", memberID, productID, createErr)
						}
					} else {
						record = records[0]
						if record.Quantity != expectedQty {
							if updateErr := tx.Model(&record).Update("quantity", expectedQty).Error; updateErr != nil {
								return fmt.Errorf("update dispatch quantity (id=%d): %w", record.ID, updateErr)
							}
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
