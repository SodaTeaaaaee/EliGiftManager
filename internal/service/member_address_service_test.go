package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestGetActiveAddressReturnsLatestNonDeletedAddress(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	member := model.Member{
		Platform:    "抖音",
		PlatformUID: "uid-100",
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	baseTime := time.Now().Add(-3 * time.Hour)
	addresses := []model.MemberAddress{
		{
			MemberID:      member.ID,
			RecipientName: "张三",
			Phone:         "13800000001",
			Address:       "旧地址",
			IsDeleted:     false,
			CreatedAt:     baseTime,
		},
		{
			MemberID:      member.ID,
			RecipientName: "李四",
			Phone:         "13800000002",
			Address:       "有效新地址",
			IsDeleted:     false,
			CreatedAt:     baseTime.Add(time.Hour),
		},
		{
			MemberID:      member.ID,
			RecipientName: "王五",
			Phone:         "13800000003",
			Address:       "已删除最新地址",
			IsDeleted:     true,
			CreatedAt:     baseTime.Add(2 * time.Hour),
		},
	}
	for _, address := range addresses {
		if err := db.Create(&address).Error; err != nil {
			t.Fatalf("failed to create member address: %v", err)
		}
	}

	activeAddress, err := GetActiveAddress(db, member.ID)
	if err != nil {
		t.Fatalf("GetActiveAddress returned unexpected error: %v", err)
	}
	if activeAddress == nil {
		t.Fatal("expected active address, got nil")
	}
	if activeAddress.Address != "有效新地址" {
		t.Fatalf("expected latest active address to be 有效新地址, got %q", activeAddress.Address)
	}
}

// --- GetPreferredAddress tests ---

func TestGetPreferredAddress_PrefersDefault(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	member := model.Member{
		Platform:    "抖音",
		PlatformUID: fmt.Sprintf("uid-prefers-default-%d", time.Now().UnixNano()),
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	// Create a non-default address first.
	nonDefault := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "非默认",
		Phone:         "13800000001",
		Address:       "非默认地址",
		IsDefault:     false,
		CreatedAt:     time.Now(),
	}
	if err := db.Create(&nonDefault).Error; err != nil {
		t.Fatalf("failed to create non-default address: %v", err)
	}

	// Create a default address second (later).
	defaultAddr := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "默认",
		Phone:         "13800000002",
		Address:       "默认地址",
		IsDefault:     true,
		CreatedAt:     time.Now().Add(time.Second),
	}
	if err := db.Create(&defaultAddr).Error; err != nil {
		t.Fatalf("failed to create default address: %v", err)
	}

	addr, err := GetPreferredAddress(db, member.ID)
	if err != nil {
		t.Fatalf("GetPreferredAddress returned unexpected error: %v", err)
	}
	if addr == nil {
		t.Fatal("expected preferred address, got nil")
	}
	if addr.ID != defaultAddr.ID {
		t.Fatalf("expected default address (id=%d), got id=%d", defaultAddr.ID, addr.ID)
	}
}

func TestGetPreferredAddress_FallsBackToLatest(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	member := model.Member{
		Platform:    "抖音",
		PlatformUID: fmt.Sprintf("uid-fallback-%d", time.Now().UnixNano()),
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	// No default addresses at all — create two non-default addresses.
	baseTime := time.Now()
	older := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "旧的",
		Phone:         "13800000001",
		Address:       "旧地址",
		IsDefault:     false,
		CreatedAt:     baseTime,
	}
	newer := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "新的",
		Phone:         "13800000002",
		Address:       "新地址",
		IsDefault:     false,
		CreatedAt:     baseTime.Add(time.Hour),
	}
	if err := db.Create(&older).Error; err != nil {
		t.Fatalf("failed to create older address: %v", err)
	}
	if err := db.Create(&newer).Error; err != nil {
		t.Fatalf("failed to create newer address: %v", err)
	}

	addr, err := GetPreferredAddress(db, member.ID)
	if err != nil {
		t.Fatalf("GetPreferredAddress returned unexpected error: %v", err)
	}
	if addr == nil {
		t.Fatal("expected preferred address, got nil")
	}
	if addr.ID != newer.ID {
		t.Fatalf("expected latest address (id=%d), got id=%d", newer.ID, addr.ID)
	}
}

func TestGetPreferredAddress_IgnoresDeleted(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	member := model.Member{
		Platform:    "抖音",
		PlatformUID: fmt.Sprintf("uid-deleted-%d", time.Now().UnixNano()),
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	// Create a default address but mark it deleted.
	deletedDefault := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "已删默认",
		Phone:         "13800000001",
		Address:       "已删除的默认地址",
		IsDefault:     true,
		IsDeleted:     true,
		CreatedAt:     time.Now(),
	}
	if err := db.Create(&deletedDefault).Error; err != nil {
		t.Fatalf("failed to create deleted default address: %v", err)
	}

	// Create a non-default active address as fallback.
	activeAddr := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "有效非默认",
		Phone:         "13800000002",
		Address:       "有效地址",
		IsDefault:     false,
		CreatedAt:     time.Now().Add(time.Second),
	}
	if err := db.Create(&activeAddr).Error; err != nil {
		t.Fatalf("failed to create active address: %v", err)
	}

	addr, err := GetPreferredAddress(db, member.ID)
	if err != nil {
		t.Fatalf("GetPreferredAddress returned unexpected error: %v", err)
	}
	if addr == nil {
		t.Fatal("expected preferred address, got nil")
	}
	if addr.ID != activeAddr.ID {
		t.Fatalf("expected fallback active address (id=%d), got id=%d", activeAddr.ID, addr.ID)
	}
}

func TestGetPreferredAddress_NilWhenNone(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	member := model.Member{
		Platform:    "抖音",
		PlatformUID: fmt.Sprintf("uid-none-%d", time.Now().UnixNano()),
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	// Member has no addresses at all.
	addr, err := GetPreferredAddress(db, member.ID)
	if err != nil {
		t.Fatalf("GetPreferredAddress returned unexpected error: %v", err)
	}
	if addr != nil {
		t.Fatalf("expected nil address, got id=%d address=%q", addr.ID, addr.Address)
	}
}
