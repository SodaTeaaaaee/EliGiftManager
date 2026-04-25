package service

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestValidateBatchBindsActiveAddressAndMarksMissingMembers(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t)
	wave := model.Wave{WaveNo: "TASK-TEST-001", Name: "test dispatch task", Status: "draft"}
	memberWithAddress := model.Member{Platform: "douyin", PlatformUID: "uid-with-address", ExtraData: "{}"}
	memberWithoutAddress := model.Member{Platform: "kuaishou", PlatformUID: "uid-without-address", ExtraData: "{}"}
	product := model.Product{Platform: "douyin", Factory: "factory", FactorySKU: "sku-001", Name: "gift", ExtraData: "{}"}
	for _, record := range []any{&wave, &memberWithAddress, &memberWithoutAddress, &product} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to seed test record: %v", err)
		}
	}
	if err := db.Create(&model.MemberNickname{MemberID: memberWithAddress.ID, Nickname: "has-address"}).Error; err != nil {
		t.Fatalf("failed to seed nickname history: %v", err)
	}
	if err := db.Create(&model.MemberNickname{MemberID: memberWithoutAddress.ID, Nickname: "missing-address"}).Error; err != nil {
		t.Fatalf("failed to seed nickname history: %v", err)
	}
	activeAddress := model.MemberAddress{MemberID: memberWithAddress.ID, RecipientName: "Alice", Phone: "13800000000", Address: "Shanghai", IsDeleted: false}
	if err := db.Create(&activeAddress).Error; err != nil {
		t.Fatalf("failed to seed active address: %v", err)
	}
	dispatchRecords := []model.DispatchRecord{
		{WaveID: wave.ID, MemberID: memberWithAddress.ID, ProductID: product.ID, Quantity: 1, Status: model.DispatchStatusPendingAddress},
		{WaveID: wave.ID, MemberID: memberWithoutAddress.ID, ProductID: product.ID, Quantity: 2, Status: model.DispatchStatusPending},
		{WaveID: wave.ID, MemberID: memberWithoutAddress.ID, ProductID: product.ID, Quantity: 1, Status: model.DispatchStatusPending},
	}
	for _, record := range dispatchRecords {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("failed to seed dispatch record: %v", err)
		}
	}
	result, err := ValidateBatch(db, wave.WaveNo)
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
	if missingMember.LatestNickname != "missing-address" {
		t.Fatalf("expected missing member nickname, got %q", missingMember.LatestNickname)
	}
	var updatedRecords []model.DispatchRecord
	if err := db.Where("wave_id = ?", wave.ID).Order("id ASC").Find(&updatedRecords).Error; err != nil {
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
