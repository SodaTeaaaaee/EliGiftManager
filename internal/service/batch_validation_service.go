package service

import (
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

func ValidateBatch(db *gorm.DB, waveNo string) (model.BatchValidationResult, error) {
	result := model.BatchValidationResult{BatchName: strings.TrimSpace(waveNo), MissingMembers: make([]model.BatchValidationMissingMember, 0)}
	if db == nil {
		return result, fmt.Errorf("wave validation failed: database connection is required")
	}
	if result.BatchName == "" {
		return result, fmt.Errorf("wave validation failed: wave number is required")
	}
	var wave model.Wave
	if err := db.Where("wave_no = ?", result.BatchName).First(&wave).Error; err != nil {
		return result, fmt.Errorf("wave validation failed: query wave failed: %w", err)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		var records []model.DispatchRecord
		if err := tx.Preload("Member").Where("wave_id = ?", wave.ID).Find(&records).Error; err != nil {
			return fmt.Errorf("query dispatch records failed: %w", err)
		}
		result.TotalRecords = len(records)
		missingMembers := make(map[uint]struct{})
		for _, record := range records {
			activeAddress, err := GetActiveAddress(tx, record.MemberID)
			if err != nil {
				return err
			}
			if activeAddress != nil {
				updates := map[string]any{"member_address_id": activeAddress.ID}
				if record.Status == "" || record.Status == model.DispatchStatusPendingAddress {
					updates["status"] = model.DispatchStatusPending
				}
				if err := tx.Model(&model.DispatchRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("bind address for record %d failed: %w", record.ID, err)
				}
				result.BoundAddressRecords++
				continue
			}
			updates := map[string]any{"member_address_id": nil, "status": model.DispatchStatusPendingAddress}
			if err := tx.Model(&model.DispatchRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("mark missing address for record %d failed: %w", record.ID, err)
			}
			result.PendingAddressRecords++
			if _, exists := missingMembers[record.MemberID]; exists {
				continue
			}
			latestNickname, err := getLatestMemberNickname(tx, record.MemberID)
			if err != nil {
				return err
			}
			missingMembers[record.MemberID] = struct{}{}
			result.MissingMembers = append(result.MissingMembers, model.BatchValidationMissingMember{MemberID: record.Member.ID, Platform: record.Member.Platform, PlatformUID: record.Member.PlatformUID, LatestNickname: extractNicknameValue(latestNickname)})
		}
		return nil
	})
	if err != nil {
		return model.BatchValidationResult{}, fmt.Errorf("wave validation failed: %w", err)
	}
	return result, nil
}

func extractNicknameValue(nickname *model.MemberNickname) string {
	if nickname == nil {
		return ""
	}
	return nickname.Nickname
}
