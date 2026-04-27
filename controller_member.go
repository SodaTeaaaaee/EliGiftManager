package main

import (
	"fmt"
	"strings"

	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type MemberController struct{}

func (c *MemberController) db() *gorm.DB { return dbpkg.GetDB() }

// ListMembers ...
func (c *MemberController) ListMembers(page, pageSize int, keyword, platform string) (MemberListPayload, error) {
	db := c.db()
	if db == nil {
		return MemberListPayload{}, fmt.Errorf("database not available")
	}
	page, pageSize = normalizePagination(page, pageSize)

	countQuery := db.Model(&model.Member{})
	if platform = strings.TrimSpace(platform); platform != "" {
		countQuery = countQuery.Where("platform = ?", platform)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		sub := db.Model(&model.MemberNickname{}).Select("member_id").Where("nickname LIKE ?", like)
		countQuery = countQuery.Where("platform_uid LIKE ? OR id IN (?)", like, sub)
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return MemberListPayload{}, err
	}

	query := db.Model(&model.Member{}).
		Preload("Nicknames", func(d *gorm.DB) *gorm.DB { return d.Order("created_at DESC") }).
		Preload("Addresses", func(d *gorm.DB) *gorm.DB { return d.Order("is_default DESC, created_at DESC") })
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		sub := db.Model(&model.MemberNickname{}).Select("member_id").Where("nickname LIKE ?", like)
		query = query.Where("platform_uid LIKE ? OR id IN (?)", like, sub)
	}
	var members []model.Member
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&members).Error; err != nil {
		return MemberListPayload{}, err
	}
	items, err := buildMemberItems(db, members)
	if err != nil {
		return MemberListPayload{}, err
	}
	platforms, err := queryMemberPlatforms(db)
	if err != nil {
		return MemberListPayload{}, err
	}
	return MemberListPayload{Items: items, Total: total, Platforms: platforms}, nil
}

func (c *MemberController) SetDefaultAddress(memberID, addressID uint) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", memberID).Update("is_default", false).Error; err != nil {
			return err
		}
		r := tx.Model(&model.MemberAddress{}).Where("id = ? AND member_id = ? AND is_deleted = ?", addressID, memberID, false).Update("is_default", true)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return fmt.Errorf("address not found")
		}
		return nil
	})
}
