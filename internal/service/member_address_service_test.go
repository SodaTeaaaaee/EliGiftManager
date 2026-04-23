package service

import (
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
