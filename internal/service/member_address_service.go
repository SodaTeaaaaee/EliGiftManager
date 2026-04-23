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
		return nil, fmt.Errorf("获取会员有效地址失败: 数据库连接不能为空")
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
		return nil, fmt.Errorf("获取会员 %d 的有效地址失败: %w", memberID, err)
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
		return nil, fmt.Errorf("查询会员 %d 的最新昵称失败: %w", memberID, err)
	}

	return &nickname, nil
}
