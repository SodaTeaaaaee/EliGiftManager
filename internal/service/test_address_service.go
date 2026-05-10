package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

const testAddressMarker = "__ELIGIFT_TEST_ADDRESS__"

// CreateFakeAddressesResult reports how many fake addresses were created.
type CreateFakeAddressesResult struct {
	TotalMembers      int64 `json:"totalMembers"`
	Created           int64 `json:"created"`
	SkippedHasAddress int64 `json:"skippedHasAddress"`
}

// DeleteFakeAddressesResult reports how many test addresses and related records were cleaned up.
type DeleteFakeAddressesResult struct {
	DeletedAddresses       int64 `json:"deletedAddresses"`
	ClearedDispatchRecords int64 `json:"clearedDispatchRecords"`
	UpdatedWaves           int64 `json:"updatedWaves"`
	AffectedMembers        int64 `json:"affectedMembers"`
}

// CreateFakeAddressesForAllMembers generates a fake address for every member that does not
// currently have a valid (non-deleted) address. The fake address is marked is_default=true
// so that BindDefaultAddresses can pick it up. All work runs inside a single transaction.
func CreateFakeAddressesForAllMembers(db *gorm.DB) (CreateFakeAddressesResult, error) {
	if db == nil {
		return CreateFakeAddressesResult{}, fmt.Errorf("create fake addresses failed: database connection is required")
	}

	var result CreateFakeAddressesResult

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Member{}).Count(&result.TotalMembers).Error; err != nil {
			return fmt.Errorf("create fake addresses failed: count all members: %w", err)
		}

		var memberIDs []uint
		if err := tx.Raw(
			"SELECT id FROM members WHERE id NOT IN (SELECT member_id FROM member_addresses WHERE is_deleted = ?)",
			false,
		).Scan(&memberIDs).Error; err != nil {
			return fmt.Errorf("create fake addresses failed: query members without address: %w", err)
		}

		if len(memberIDs) == 0 {
			result.SkippedHasAddress = result.TotalMembers
			return nil
		}

		addresses := make([]model.MemberAddress, 0, len(memberIDs))
		for _, mid := range memberIDs {
			addresses = append(addresses, buildFakeAddress(mid))
		}

		result.Created = int64(len(addresses))
		result.SkippedHasAddress = result.TotalMembers - result.Created

		if err := tx.CreateInBatches(addresses, 100).Error; err != nil {
			return fmt.Errorf("create fake addresses failed: batch insert: %w", err)
		}

		return nil
	})

	if err != nil {
		return CreateFakeAddressesResult{}, err
	}

	return result, nil
}

// DeleteFakeAddressesForAllMembers removes all test addresses (identified by is_test_address=true),
// clears dispatch record bindings that reference them, and resets affected wave statuses back to
// affected wave statuses back to pending_address. All work runs inside a single transaction.
func DeleteFakeAddressesForAllMembers(db *gorm.DB) (DeleteFakeAddressesResult, error) {
	if db == nil {
		return DeleteFakeAddressesResult{}, fmt.Errorf("delete fake addresses failed: database connection is required")
	}

	var result DeleteFakeAddressesResult

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Find all test addresses by is_test_address flag.
		var addresses []model.MemberAddress
		if err := tx.
			Where("is_deleted = ? AND is_test_address = ?", false, true).
			Find(&addresses).Error; err != nil {
			return fmt.Errorf("delete fake addresses failed: query test addresses: %w", err)
		}

		if len(addresses) == 0 {
			return nil
		}

		addressIDs := make([]uint, 0, len(addresses))
		memberIDSet := make(map[uint]struct{})
		for _, addr := range addresses {
			addressIDs = append(addressIDs, addr.ID)
			memberIDSet[addr.MemberID] = struct{}{}
		}

		// 2. Find dispatch records that reference these addresses.
		var records []model.DispatchRecord
		if err := tx.Where("member_address_id IN ?", addressIDs).Find(&records).Error; err != nil {
			return fmt.Errorf("delete fake addresses failed: query dispatch records: %w", err)
		}

		waveIDSet := make(map[uint]struct{})
		if len(records) > 0 {
			recordIDs := make([]uint, 0, len(records))
			for _, rec := range records {
				recordIDs = append(recordIDs, rec.ID)
				waveIDSet[rec.WaveID] = struct{}{}
			}

			// Clear dispatch record bindings.
			if err := tx.Model(&model.DispatchRecord{}).
				Where("id IN ?", recordIDs).
				Updates(map[string]any{
					"member_address_id": nil,
					"status":            model.DispatchStatusPendingAddress,
				}).Error; err != nil {
				return fmt.Errorf("delete fake addresses failed: clear dispatch records: %w", err)
			}
			result.ClearedDispatchRecords = int64(len(recordIDs))
		}

		// 3. Recompute affected wave statuses via unified entry point (D14).
		if len(waveIDSet) > 0 {
			for wid := range waveIDSet {
				if err := RecomputeWaveStatus(tx, wid); err != nil {
					return fmt.Errorf("delete fake addresses failed: recompute wave %d status: %w", wid, err)
				}
			}
			result.UpdatedWaves = int64(len(waveIDSet))
		}

		// 4. Soft-delete the test addresses.
		if err := tx.Model(&model.MemberAddress{}).
			Where("id IN ?", addressIDs).
			Updates(map[string]any{
				"is_deleted": true,
				"is_default": false,
			}).Error; err != nil {
			return fmt.Errorf("delete fake addresses failed: soft-delete addresses: %w", err)
		}
		result.DeletedAddresses = int64(len(addressIDs))
		result.AffectedMembers = int64(len(memberIDSet))

		return nil
	})

	if err != nil {
		return DeleteFakeAddressesResult{}, err
	}

	return result, nil
}

// generateFakePhone produces an 11-digit phone number starting with "1".
func generateFakePhone() string {
	var b strings.Builder
	b.WriteByte('1')
	for range 10 {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			b.WriteByte('0')
			continue
		}
		b.WriteString(strconv.Itoa(int(n.Int64())))
	}
	return b.String()
}

// buildFakeAddress constructs a MemberAddress with the test marker and a fake phone.
func buildFakeAddress(memberID uint) model.MemberAddress {
	return model.MemberAddress{
		MemberID:      memberID,
		IsDefault:     true,
		IsDeleted:     false,
		IsTestAddress: true,
		RecipientName: fmt.Sprintf("%s member-%d", testAddressMarker, memberID),
		Phone:         generateFakePhone(),
		Address:       fmt.Sprintf("%s generated for member-%d", testAddressMarker, memberID),
	}
}
