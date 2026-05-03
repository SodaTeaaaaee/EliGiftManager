package main

import (
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type MemberController struct {
	db *gorm.DB
}

// ListMembers ...
func (c *MemberController) ListMembers(page, pageSize int, keyword, platform string) (MemberListPayload, error) {
	db := c.db
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
	db := c.db
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

// ListWaveMembers returns members associated with a specific wave.
func (c *MemberController) ListWaveMembers(waveID uint) ([]MemberItem, error) {
	db := c.db

	// 1. Load wave_members with snapshot fields + related Member.Addresses
	var waveMembers []model.WaveMember
	if err := db.
		Preload("Member.Addresses", func(d *gorm.DB) *gorm.DB { return d.Order("is_default DESC, created_at DESC") }).
		Preload("Member.Nicknames", func(d *gorm.DB) *gorm.DB { return d.Order("created_at DESC") }).
		Where("wave_id = ?", waveID).
		Order("id ASC").
		Find(&waveMembers).Error; err != nil {
		return nil, fmt.Errorf("list wave members failed: %w", err)
	}

	// 2. Build MemberItems from wave_member snapshot fields + Member.Addresses
	items := make([]MemberItem, 0, len(waveMembers))
	for _, wm := range waveMembers {
		item := MemberItem{
			ID:             wm.ID,
			MemberID:       wm.MemberID,
			Platform:       wm.Platform,
			PlatformUID:    wm.PlatformUID,
			LatestNickname: wm.LatestNickname,
			GiftLevel:      wm.GiftLevel,
			ExtraData:      wm.Member.ExtraData,
			AddressCount:   len(wm.Member.Addresses),
			Addresses:      wm.Member.Addresses,
			Nicknames:      wm.Member.Nicknames,
		}
		for _, address := range wm.Member.Addresses {
			if address.IsDeleted {
				continue
			}
			item.ActiveAddressCount++
			if item.LatestAddress == "" || address.IsDefault {
				item.LatestRecipient = address.RecipientName
				item.LatestPhone = address.Phone
				item.LatestAddress = address.Address
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// AddMemberAddress creates a new address for a member.
func (c *MemberController) AddMemberAddress(memberID uint, recipientName, phone, address string) (model.MemberAddress, error) {
	db := c.db
	recipientName = strings.TrimSpace(recipientName)
	phone = strings.TrimSpace(phone)
	address = strings.TrimSpace(address)
	if recipientName == "" || phone == "" || address == "" {
		return model.MemberAddress{}, fmt.Errorf("recipient name, phone, and address are required")
	}
	var count int64
	if err := db.Model(&model.Member{}).Where("id = ?", memberID).Count(&count).Error; err != nil {
		return model.MemberAddress{}, fmt.Errorf("add member address failed: %w", err)
	}
	if count == 0 {
		return model.MemberAddress{}, fmt.Errorf("member not found")
	}
	addr := model.MemberAddress{MemberID: memberID, RecipientName: recipientName, Phone: phone, Address: address, IsDefault: false, IsDeleted: false}
	if err := db.Create(&addr).Error; err != nil {
		return model.MemberAddress{}, fmt.Errorf("add member address failed: %w", err)
	}
	return addr, nil
}

// UpdateMemberAddress updates an existing active address.
func (c *MemberController) UpdateMemberAddress(addressID uint, recipientName, phone, address string) error {
	db := c.db
	recipientName = strings.TrimSpace(recipientName)
	phone = strings.TrimSpace(phone)
	address = strings.TrimSpace(address)
	if recipientName == "" || phone == "" || address == "" {
		return fmt.Errorf("recipient name, phone, and address are required")
	}
	result := db.Model(&model.MemberAddress{}).Where("id = ? AND is_deleted = ?", addressID, false).Updates(map[string]any{
		"recipient_name": recipientName,
		"phone":          phone,
		"address":        address,
	})
	if result.Error != nil {
		return fmt.Errorf("update member address failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("address not found")
	}
	return nil
}

// DeleteMemberAddress soft-deletes an address. If it was the default, unsets
// the default flag for the member first.
func (c *MemberController) DeleteMemberAddress(addressID uint) error {
	db := c.db
	return db.Transaction(func(tx *gorm.DB) error {
		var addr model.MemberAddress
		if err := tx.Where("id = ? AND is_deleted = ?", addressID, false).First(&addr).Error; err != nil {
			return fmt.Errorf("address not found")
		}
		if addr.IsDefault {
			if err := tx.Model(&model.MemberAddress{}).Where("member_id = ?", addr.MemberID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&addr).Update("is_deleted", true).Error
	})
}

func (c *MemberController) RemoveMemberFromWave(waveID, waveMemberID uint) error {
	db := c.db
	return db.Transaction(func(tx *gorm.DB) error {
		// Find wave_member record to get the underlying member_id
		var wm model.WaveMember
		if err := tx.Where("id = ? AND wave_id = ?", waveMemberID, waveID).First(&wm).Error; err != nil {
			return fmt.Errorf("member not found in this wave")
		}
		// Delete user tags referencing this wave member (manual FK cascade).
		if err := tx.Where("wave_member_id = ? AND tag_type = ?", wm.ID, model.TagTypeUser).Delete(&model.ProductTag{}).Error; err != nil {
			return fmt.Errorf("clean user tags failed: %w", err)
		}
		// Clean up associated dispatch records by wave_id + member_id
		if err := tx.Where("wave_id = ? AND member_id = ?", waveID, wm.MemberID).Delete(&model.DispatchRecord{}).Error; err != nil {
			return fmt.Errorf("clean dispatch records failed: %w", err)
		}
		// Delete the wave_member record
		if err := tx.Delete(&wm).Error; err != nil {
			return fmt.Errorf("remove member from wave failed: %w", err)
		}
		return nil
	})
}
