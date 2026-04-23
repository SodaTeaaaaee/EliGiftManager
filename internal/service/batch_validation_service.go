package service

import (
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// ValidateBatch 执行批次导出前的地址预校验，并回写地址绑定与状态。
func ValidateBatch(db *gorm.DB, batchName string) (model.BatchValidationResult, error) {
	result := model.BatchValidationResult{
		BatchName:      strings.TrimSpace(batchName),
		MissingMembers: make([]model.BatchValidationMissingMember, 0),
	}

	if db == nil {
		return result, fmt.Errorf("批次预校验失败: 数据库连接不能为空")
	}
	if result.BatchName == "" {
		return result, fmt.Errorf("批次预校验失败: 批次号不能为空")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var records []model.DispatchRecord
		if err := tx.
			Preload("Member").
			Where("batch_name = ?", result.BatchName).
			Find(&records).
			Error; err != nil {
			return fmt.Errorf("查询批次记录失败: %w", err)
		}

		result.TotalRecords = len(records)
		missingMembers := make(map[uint]struct{})

		for _, record := range records {
			activeAddress, err := GetActiveAddress(tx, record.MemberID)
			if err != nil {
				return err
			}

			if activeAddress != nil {
				updates := map[string]interface{}{
					"member_address_id": activeAddress.ID,
				}
				if record.Status == "" || record.Status == model.DispatchStatusPendingAddress {
					updates["status"] = model.DispatchStatusPending
				}

				if err := tx.Model(&model.DispatchRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("更新记录 %d 的地址绑定失败: %w", record.ID, err)
				}

				result.BoundAddressRecords++
				continue
			}

			updates := map[string]interface{}{
				"member_address_id": nil,
				"status":            model.DispatchStatusPendingAddress,
			}
			if err := tx.Model(&model.DispatchRecord{}).Where("id = ?", record.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("更新记录 %d 的缺地址状态失败: %w", record.ID, err)
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
			result.MissingMembers = append(result.MissingMembers, model.BatchValidationMissingMember{
				MemberID:       record.Member.ID,
				Platform:       record.Member.Platform,
				PlatformUID:    record.Member.PlatformUID,
				LatestNickname: extractNicknameValue(latestNickname),
			})
		}

		return nil
	})
	if err != nil {
		return model.BatchValidationResult{}, fmt.Errorf("批次预校验失败: %w", err)
	}

	return result, nil
}

func extractNicknameValue(nickname *model.MemberNickname) string {
	if nickname == nil {
		return ""
	}

	return nickname.Nickname
}
