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
