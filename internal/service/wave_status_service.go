package service

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// RecomputeWaveStatus 统一计算并更新波次状态。
// 四个分支：
//   - 无 DispatchRecord → "draft"
//   - 有缺地址（NULL / 已删除地址 / 地址字段为空 / status=pending_address）→ "pending_address"
//   - 全部 exported → "exported"
//   - 其余 → "pending"
func RecomputeWaveStatus(tx *gorm.DB, waveID uint) error {
	var total int64
	if err := tx.Model(&model.DispatchRecord{}).
		Where("wave_id = ?", waveID).
		Count(&total).Error; err != nil {
		return err
	}

	if total == 0 {
		return tx.Model(&model.Wave{}).
			Where("id = ?", waveID).
			Update("status", "draft").Error
	}

	var missingAddr int64
	if err := tx.Model(&model.DispatchRecord{}).
		Joins("LEFT JOIN member_addresses ON dispatch_records.member_address_id = member_addresses.id").
		Where("dispatch_records.wave_id = ?", waveID).
		Where(`dispatch_records.status = ?
			OR dispatch_records.member_address_id IS NULL
			OR member_addresses.is_deleted = ?
			OR member_addresses.recipient_name = ''
			OR member_addresses.phone = ''
			OR member_addresses.address = ''`,
			model.DispatchStatusPendingAddress, true).
		Count(&missingAddr).Error; err != nil {
		return err
	}

	if missingAddr > 0 {
		return tx.Model(&model.Wave{}).
			Where("id = ?", waveID).
			Update("status", model.DispatchStatusPendingAddress).Error
	}

	var exported int64
	if err := tx.Model(&model.DispatchRecord{}).
		Where("wave_id = ? AND status = ?",
			waveID, model.DispatchStatusExported).
		Count(&exported).Error; err != nil {
		return err
	}

	if exported == total {
		return tx.Model(&model.Wave{}).
			Where("id = ?", waveID).
			Update("status", model.DispatchStatusExported).Error
	}

	return tx.Model(&model.Wave{}).
		Where("id = ?", waveID).
		Update("status", model.DispatchStatusPending).Error
}

// InvalidateWaveExports marks exported dispatch records in the wave back to
// pending when the underlying allocation set has changed.
func InvalidateWaveExports(tx *gorm.DB, waveID uint) error {
	return tx.Model(&model.DispatchRecord{}).
		Where("wave_id = ? AND status = ?", waveID, model.DispatchStatusExported).
		Update("status", model.DispatchStatusPending).Error
}
