package service

import (
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestCreateFakeAddresses_NoMembers(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	result, err := CreateFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalMembers != 0 {
		t.Errorf("expected TotalMembers=0, got %d", result.TotalMembers)
	}
	if result.Created != 0 {
		t.Errorf("expected Created=0, got %d", result.Created)
	}
}

func TestCreateFakeAddresses_OnlyMissingMembers(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	// Create two members.
	m1 := model.Member{Platform: "test", PlatformUID: "uid-1"}
	m2 := model.Member{Platform: "test", PlatformUID: "uid-2"}
	if err := db.Create(&m1).Error; err != nil {
		t.Fatalf("create member 1: %v", err)
	}
	if err := db.Create(&m2).Error; err != nil {
		t.Fatalf("create member 2: %v", err)
	}

	// Give m1 an existing address, leave m2 without one.
	realAddr := model.MemberAddress{
		MemberID:      m1.ID,
		RecipientName: "Real Recipient",
		Phone:         "12345678901",
		Address:       "123 Real St",
		IsDefault:     true,
	}
	if err := db.Create(&realAddr).Error; err != nil {
		t.Fatalf("create real address: %v", err)
	}

	result, err := CreateFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalMembers != 2 {
		t.Errorf("expected TotalMembers=2, got %d", result.TotalMembers)
	}
	if result.Created != 1 {
		t.Errorf("expected Created=1, got %d", result.Created)
	}
	if result.SkippedHasAddress != 1 {
		t.Errorf("expected SkippedHasAddress=1, got %d", result.SkippedHasAddress)
	}

	// Verify m2 got a test address.
	var addrs []model.MemberAddress
	if err := db.Where("member_id = ? AND is_deleted = ?", m2.ID, false).Find(&addrs).Error; err != nil {
		t.Fatalf("query m2 addresses: %v", err)
	}
	if len(addrs) != 1 {
		t.Fatalf("expected 1 address for m2, got %d", len(addrs))
	}
	if !strings.HasPrefix(addrs[0].RecipientName, testAddressMarker) {
		t.Errorf("m2 address not marked as test: recipient=%s", addrs[0].RecipientName)
	}
}

func TestCreateFakeAddresses_FormatCorrect(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-3"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	_, err := CreateFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var addr model.MemberAddress
	if err := db.Where("member_id = ? AND is_deleted = ?", m.ID, false).First(&addr).Error; err != nil {
		t.Fatalf("query fake address: %v", err)
	}

	if !addr.IsDefault {
		t.Error("expected is_default=true")
	}
	if addr.IsDeleted {
		t.Error("expected is_deleted=false")
	}
	if !strings.HasPrefix(addr.RecipientName, testAddressMarker) {
		t.Errorf("recipient_name missing marker: %s", addr.RecipientName)
	}
	if !strings.HasPrefix(addr.Address, testAddressMarker) {
		t.Errorf("address missing marker: %s", addr.Address)
	}
	if len(addr.Phone) != 11 {
		t.Errorf("expected phone length 11, got %d (%s)", len(addr.Phone), addr.Phone)
	}
	if addr.Phone[0] != '1' {
		t.Errorf("expected phone to start with '1', got %s", addr.Phone)
	}
}

func TestCreateFakeAddresses_Idempotent(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-4"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// First call: should create one address.
	result1, err := CreateFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if result1.Created != 1 {
		t.Errorf("first call expected Created=1, got %d", result1.Created)
	}

	// Second call: member already has an address, should create nothing.
	result2, err := CreateFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if result2.Created != 0 {
		t.Errorf("second call expected Created=0, got %d", result2.Created)
	}
	if result2.TotalMembers != 1 {
		t.Errorf("second call expected TotalMembers=1, got %d", result2.TotalMembers)
	}
	if result2.SkippedHasAddress != 1 {
		t.Errorf("second call expected SkippedHasAddress=1, got %d", result2.SkippedHasAddress)
	}
}

func TestDeleteFakeAddresses_OnlyTestAddresses(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-5"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// Create a real address.
	realAddr := model.MemberAddress{
		MemberID:      m.ID,
		RecipientName: "Real Recipient",
		Phone:         "12345678901",
		Address:       "Real Address",
		IsDefault:     true,
	}
	if err := db.Create(&realAddr).Error; err != nil {
		t.Fatalf("create real address: %v", err)
	}

	// Create a fake test address directly.
	fakeAddr := buildFakeAddress(m.ID)
	if err := db.Create(&fakeAddr).Error; err != nil {
		t.Fatalf("create fake address: %v", err)
	}

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeletedAddresses != 1 {
		t.Errorf("expected DeletedAddresses=1, got %d", result.DeletedAddresses)
	}

	// Verify real address is untouched.
	var realCheck model.MemberAddress
	if err := db.First(&realCheck, realAddr.ID).Error; err != nil {
		t.Fatalf("real address should still exist: %v", err)
	}
	if realCheck.IsDeleted {
		t.Error("real address should not be deleted")
	}

	// Verify fake address is soft-deleted.
	var fakeCheck model.MemberAddress
	if err := db.First(&fakeCheck, fakeAddr.ID).Error; err != nil {
		t.Fatalf("fake address record should still exist: %v", err)
	}
	if !fakeCheck.IsDeleted {
		t.Error("fake address should be soft-deleted")
	}
}

func TestDeleteFakeAddresses_ClearsDispatchRecords(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-6"}
	p := model.Product{Platform: "test", Factory: "f", FactorySKU: "sku", Name: "Product"}
	w := model.Wave{WaveNo: "W001", Name: "Test Wave", Status: model.DispatchStatusPending}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}

	// Create fake address and bind a dispatch record to it.
	fakeAddr := buildFakeAddress(m.ID)
	if err := db.Create(&fakeAddr).Error; err != nil {
		t.Fatalf("create fake address: %v", err)
	}

	rec := model.DispatchRecord{
		WaveID:          w.ID,
		MemberID:        m.ID,
		ProductID:       p.ID,
		MemberAddressID: &fakeAddr.ID,
		Status:          model.DispatchStatusPending,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("create dispatch record: %v", err)
	}

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClearedDispatchRecords != 1 {
		t.Errorf("expected ClearedDispatchRecords=1, got %d", result.ClearedDispatchRecords)
	}

	// Re-fetch the dispatch record.
	var updatedRec model.DispatchRecord
	if err := db.First(&updatedRec, rec.ID).Error; err != nil {
		t.Fatalf("re-fetch dispatch record: %v", err)
	}
	if updatedRec.MemberAddressID != nil {
		t.Error("expected member_address_id to be nil after deletion")
	}
	if updatedRec.Status != model.DispatchStatusPendingAddress {
		t.Errorf("expected status=%s, got %s", model.DispatchStatusPendingAddress, updatedRec.Status)
	}
}

func TestDeleteFakeAddresses_ResetsWavesStatus(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-7"}
	p := model.Product{Platform: "test", Factory: "f", FactorySKU: "sku", Name: "Product"}
	w := model.Wave{WaveNo: "W002", Name: "Test Wave 2", Status: model.DispatchStatusPending}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}

	fakeAddr := buildFakeAddress(m.ID)
	if err := db.Create(&fakeAddr).Error; err != nil {
		t.Fatalf("create fake address: %v", err)
	}

	rec := model.DispatchRecord{
		WaveID:          w.ID,
		MemberID:        m.ID,
		ProductID:       p.ID,
		MemberAddressID: &fakeAddr.ID,
		Status:          model.DispatchStatusPending,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("create dispatch record: %v", err)
	}

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UpdatedWaves != 1 {
		t.Errorf("expected UpdatedWaves=1, got %d", result.UpdatedWaves)
	}

	// Verify wave status reset.
	var updatedWave model.Wave
	if err := db.First(&updatedWave, w.ID).Error; err != nil {
		t.Fatalf("re-fetch wave: %v", err)
	}
	if updatedWave.Status != model.DispatchStatusPendingAddress {
		t.Errorf("expected wave status=%s, got %s", model.DispatchStatusPendingAddress, updatedWave.Status)
	}
}

func TestDeleteFakeAddresses_KeepsRealBindings(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-8"}
	p1 := model.Product{Platform: "test", Factory: "f", FactorySKU: "sku-real", Name: "Real Product"}
	p2 := model.Product{Platform: "test", Factory: "f", FactorySKU: "sku-fake", Name: "Fake Product"}
	w := model.Wave{WaveNo: "W003", Name: "Test Wave 3"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("create product 1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("create product 2: %v", err)
	}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}

	// Real address bound to a dispatch.
	realAddr := model.MemberAddress{
		MemberID:      m.ID,
		RecipientName: "Real",
		Phone:         "12345678901",
		Address:       "Real",
		IsDefault:     true,
	}
	if err := db.Create(&realAddr).Error; err != nil {
		t.Fatalf("create real address: %v", err)
	}
	realRec := model.DispatchRecord{
		WaveID:          w.ID,
		MemberID:        m.ID,
		ProductID:       p1.ID,
		MemberAddressID: &realAddr.ID,
		Status:          model.DispatchStatusPending,
	}
	if err := db.Create(&realRec).Error; err != nil {
		t.Fatalf("create real dispatch: %v", err)
	}

	// Fake address bound to another dispatch.
	fakeAddr := buildFakeAddress(m.ID)
	if err := db.Create(&fakeAddr).Error; err != nil {
		t.Fatalf("create fake address: %v", err)
	}
	fakeRec := model.DispatchRecord{
		WaveID:          w.ID,
		MemberID:        m.ID,
		ProductID:       p2.ID,
		MemberAddressID: &fakeAddr.ID,
		Status:          model.DispatchStatusPending,
	}
	if err := db.Create(&fakeRec).Error; err != nil {
		t.Fatalf("create fake dispatch: %v", err)
	}

	_, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Real dispatch record should still have its address.
	var realRecCheck model.DispatchRecord
	if err := db.First(&realRecCheck, realRec.ID).Error; err != nil {
		t.Fatalf("re-fetch real dispatch: %v", err)
	}
	if realRecCheck.MemberAddressID == nil {
		t.Error("real dispatch should still have member_address_id")
	}
	if *realRecCheck.MemberAddressID != realAddr.ID {
		t.Errorf("real dispatch address ID mismatch: got %v, want %d", realRecCheck.MemberAddressID, realAddr.ID)
	}
}

func TestDeleteFakeAddresses_Idempotent(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeletedAddresses != 0 {
		t.Errorf("expected DeletedAddresses=0, got %d", result.DeletedAddresses)
	}
	if result.ClearedDispatchRecords != 0 {
		t.Errorf("expected ClearedDispatchRecords=0, got %d", result.ClearedDispatchRecords)
	}
	if result.UpdatedWaves != 0 {
		t.Errorf("expected UpdatedWaves=0, got %d", result.UpdatedWaves)
	}
	if result.AffectedMembers != 0 {
		t.Errorf("expected AffectedMembers=0, got %d", result.AffectedMembers)
	}
}

func TestDeleteFakeAddresses_TextChangedStillMatches(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-10"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// Create a test address via buildFakeAddress (sets IsTestAddress=true).
	fakeAddr := buildFakeAddress(m.ID)
	if err := db.Create(&fakeAddr).Error; err != nil {
		t.Fatalf("create fake address: %v", err)
	}

	// Manually change recipient_name and address text to remove the marker.
	if err := db.Model(&fakeAddr).Updates(map[string]any{
		"recipient_name": "张三",
		"address":        "上海市浦东新区",
	}).Error; err != nil {
		t.Fatalf("update fake address text: %v", err)
	}

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeletedAddresses != 1 {
		t.Errorf("expected DeletedAddresses=1 (matched by is_test_address), got %d", result.DeletedAddresses)
	}

	// Verify the address was soft-deleted.
	var check model.MemberAddress
	if err := db.First(&check, fakeAddr.ID).Error; err != nil {
		t.Fatalf("fake address record should still exist: %v", err)
	}
	if !check.IsDeleted {
		t.Error("address should be soft-deleted (matched by is_test_address despite text change)")
	}
}

func TestDeleteFakeAddresses_RealAddressWithMarkerText(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	m := model.Member{Platform: "test", PlatformUID: "uid-11"}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	// Create a real address whose text happens to contain the marker string,
	// but IsTestAddress=false.
	realAddr := model.MemberAddress{
		MemberID:      m.ID,
		IsTestAddress: false,
		RecipientName: testAddressMarker + " coincidence",
		Phone:         "12345678901",
		Address:       testAddressMarker + " coincidence",
		IsDefault:     true,
	}
	if err := db.Create(&realAddr).Error; err != nil {
		t.Fatalf("create real address with marker text: %v", err)
	}

	result, err := DeleteFakeAddressesForAllMembers(db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeletedAddresses != 0 {
		t.Errorf("expected DeletedAddresses=0 (real address has marker text but is_test_address=false), got %d", result.DeletedAddresses)
	}

	// Verify the real address is untouched.
	var check model.MemberAddress
	if err := db.First(&check, realAddr.ID).Error; err != nil {
		t.Fatalf("real address should still exist: %v", err)
	}
	if check.IsDeleted {
		t.Error("real address should not be deleted despite marker text in fields")
	}
}
