package service

import (
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// GetActiveAddress 返回指定会员最新一条且未被删除的地址记录。
func GetActiveAddress(db *gorm.DB, memberID uint) (*model.MemberAddress, error) {
	if db == nil {
		return nil, fmt.Errorf("get active address failed: database connection is required")
	}

	var address model.MemberAddress
	err := db.
		Where("member_id = ? AND is_deleted = ?", memberID, false).
		Order("created_at DESC").
		Order("id DESC").
		First(&address).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active address for member %d failed: %w", memberID, err)
	}

	return &address, nil
}

// GetPreferredAddress returns the member's preferred address: default active address first,
// falling back to the latest non-deleted address. Returns nil if no active address exists.
func GetPreferredAddress(db *gorm.DB, memberID uint) (*model.MemberAddress, error) {
	if db == nil {
		return nil, fmt.Errorf("get preferred address failed: database connection is required")
	}
	// Priority 1: active default address.
	var addr model.MemberAddress
	err := db.Where("member_id = ? AND is_default = ? AND is_deleted = ?", memberID, true, false).
		First(&addr).Error
	if err == nil {
		return &addr, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get preferred address for member %d failed: %w", memberID, err)
	}
	// Priority 2: latest active non-deleted address.
	return GetActiveAddress(db, memberID)
}

func getLatestMemberNickname(db *gorm.DB, memberID uint) (*model.MemberNickname, error) {
	var nickname model.MemberNickname
	err := db.
		Where("member_id = ?", memberID).
		Order("created_at DESC").
		Order("id DESC").
		First(&nickname).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest nickname for member %d failed: %w", memberID, err)
	}

	return &nickname, nil
}
