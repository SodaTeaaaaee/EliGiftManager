package service

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestValidateBatchBindsActiveAddressAndMarksMissingMembers(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	memberWithAddress := model.Member{
		Platform:    "抖音",
		PlatformUID: "uid-with-address",
		ExtraData:   "{}",
	}
	memberWithoutAddress := model.Member{
		Platform:    "快手",
		PlatformUID: "uid-without-address",
		ExtraData:   "{}",
	}
	product := model.Product{
		Factory:    "华东工厂",
		FactorySKU: "sku-001",
		Name:       "礼盒",
		ExtraData:  "{}",
	}

	for _, record := range []interface{}{&memberWithAddress, &memberWithoutAddress, &product} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to seed test record: %v", err)
		}
	}

	if err := db.Create(&model.MemberNickname{MemberID: memberWithAddress.ID, Nickname: "有地址会员"}).Error; err != nil {
		t.Fatalf("failed to seed nickname history: %v", err)
	}
	if err := db.Create(&model.MemberNickname{MemberID: memberWithoutAddress.ID, Nickname: "缺地址会员"}).Error; err != nil {
		t.Fatalf("failed to seed nickname history: %v", err)
	}

	activeAddress := model.MemberAddress{
		MemberID:      memberWithAddress.ID,
		RecipientName: "张三",
		Phone:         "13800000000",
		Address:       "上海市浦东新区",
		IsDeleted:     false,
	}
	if err := db.Create(&activeAddress).Error; err != nil {
		t.Fatalf("failed to seed active address: %v", err)
	}

	dispatchRecords := []model.DispatchRecord{
		{
			BatchName: "batch-validate",
			MemberID:  memberWithAddress.ID,
			ProductID: product.ID,
			Quantity:  1,
			Status:    model.DispatchStatusPendingAddress,
		},
		{
			BatchName: "batch-validate",
			MemberID:  memberWithoutAddress.ID,
			ProductID: product.ID,
			Quantity:  2,
			Status:    model.DispatchStatusPending,
		},
		{
			BatchName: "batch-validate",
			MemberID:  memberWithoutAddress.ID,
			ProductID: product.ID,
			Quantity:  1,
			Status:    model.DispatchStatusPending,
		},
	}
	for _, record := range dispatchRecords {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("failed to seed dispatch record: %v", err)
		}
	}

	result, err := ValidateBatch(db, "batch-validate")
	if err != nil {
		t.Fatalf("ValidateBatch returned unexpected error: %v", err)
	}

	if result.TotalRecords != 3 {
		t.Fatalf("expected TotalRecords to be 3, got %d", result.TotalRecords)
	}
	if result.BoundAddressRecords != 1 {
		t.Fatalf("expected BoundAddressRecords to be 1, got %d", result.BoundAddressRecords)
	}
	if result.PendingAddressRecords != 2 {
		t.Fatalf("expected PendingAddressRecords to be 2, got %d", result.PendingAddressRecords)
	}
	if len(result.MissingMembers) != 1 {
		t.Fatalf("expected 1 unique missing member, got %d", len(result.MissingMembers))
	}

	missingMember := result.MissingMembers[0]
	if missingMember.MemberID != memberWithoutAddress.ID {
		t.Fatalf("expected missing member id to be %d, got %d", memberWithoutAddress.ID, missingMember.MemberID)
	}
	if missingMember.LatestNickname != "缺地址会员" {
		t.Fatalf("expected missing member nickname to be 缺地址会员, got %q", missingMember.LatestNickname)
	}

	var updatedRecords []model.DispatchRecord
	if err := db.Where("batch_name = ?", "batch-validate").Order("id ASC").Find(&updatedRecords).Error; err != nil {
		t.Fatalf("failed to query updated dispatch records: %v", err)
	}

	if updatedRecords[0].MemberAddressID == nil || *updatedRecords[0].MemberAddressID != activeAddress.ID {
		t.Fatalf("expected first record to bind active address %d, got %+v", activeAddress.ID, updatedRecords[0].MemberAddressID)
	}
	if updatedRecords[0].Status != model.DispatchStatusPending {
		t.Fatalf("expected first record status to recover to pending, got %q", updatedRecords[0].Status)
	}
	if updatedRecords[1].MemberAddressID != nil || updatedRecords[2].MemberAddressID != nil {
		t.Fatal("expected missing-address records to keep MemberAddressID nil")
	}
	if updatedRecords[1].Status != model.DispatchStatusPendingAddress || updatedRecords[2].Status != model.DispatchStatusPendingAddress {
		t.Fatalf("expected missing-address records to become pending_address, got %q and %q", updatedRecords[1].Status, updatedRecords[2].Status)
	}
}
