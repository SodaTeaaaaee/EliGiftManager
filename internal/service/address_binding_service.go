package service

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// BindDefaultAddresses binds the preferred address of each member to all
// dispatch records in the given wave that currently have no address set.
// It uses GetPreferredAddress which gives priority to the default address,
// falling back to the latest non-deleted address.
// Members without any usable address are skipped.
// Returns the count of updated records and the count of skipped records.
func BindDefaultAddresses(db *gorm.DB, waveID uint) (updated int, skipped int, err error) {
	if db == nil {
		return 0, 0, fmt.Errorf("bind default addresses failed: database connection is required")
	}
	if waveID == 0 {
		return 0, 0, fmt.Errorf("bind default addresses failed: wave ID is required")
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// 1. Query records that need address binding: NULL address OR bound to a
		//    deleted address (D12).
		var records []model.DispatchRecord
		if err := tx.
			Where("wave_id = ? AND (member_address_id IS NULL OR member_address_id IN (SELECT id FROM member_addresses WHERE is_deleted = ?))", waveID, true).
			Find(&records).Error; err != nil {
			return fmt.Errorf("bind default addresses failed: query records: %w", err)
		}

		if len(records) == 0 {
			return nil
		}

		// Cache of memberID -> default address ID to avoid duplicate queries.
		defaultAddrCache := make(map[uint]*uint)

		for _, record := range records {
			addrID, ok := defaultAddrCache[record.MemberID]
			if !ok {
				// Look up the member's preferred address (default first, then latest).
				addr, err := GetPreferredAddress(tx, record.MemberID)
				if err != nil {
					return fmt.Errorf("bind default addresses failed: get preferred address for member %d: %w", record.MemberID, err)
				}
				if addr == nil {
					// No usable address found – cache nil and skip.
					defaultAddrCache[record.MemberID] = nil
					skipped++
					continue
				}
				defaultAddrCache[record.MemberID] = &addr.ID
				addrID = &addr.ID
			}

			if addrID == nil {
				skipped++
				continue
			}

			// Update the record's MemberAddressID and set status back to pending.
			if err := tx.Model(&model.DispatchRecord{}).
				Where("id = ?", record.ID).
				Updates(map[string]interface{}{
					"member_address_id": *addrID,
					"status":            model.DispatchStatusPending,
				}).Error; err != nil {
				return fmt.Errorf("bind default addresses failed: update record %d: %w", record.ID, err)
			}
			updated++
		}

		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	return updated, skipped, nil
}
