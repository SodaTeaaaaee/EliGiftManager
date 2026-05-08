package service

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestBindDefaultAddresses_UsesNonDefault(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	// Create a member with only a non-default (but active) address.
	member := model.Member{
		Platform:    "test",
		PlatformUID: "uid-nondefault-binding",
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	nonDefaultAddr := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "NonDefault User",
		Phone:         "13900000001",
		Address:       "123 NonDefault St",
		IsDefault:     false,
		IsDeleted:     false,
	}
	if err := db.Create(&nonDefaultAddr).Error; err != nil {
		t.Fatalf("create non-default address: %v", err)
	}

	// Create a wave and a product.
	wave := model.Wave{WaveNo: "TASK-20260101-001", Name: "Binding Wave", Status: "draft"}
	p := model.Product{Platform: "test", Factory: "F", FactorySKU: "sku-nondef", Name: "Product"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	// Create a dispatch record with no address set.
	rec := model.DispatchRecord{
		WaveID:    wave.ID,
		MemberID:  member.ID,
		ProductID: p.ID,
		Quantity:  1,
		Status:    model.DispatchStatusPendingAddress,
	}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("create dispatch record: %v", err)
	}

	// Bind — should pick up the non-default address via GetPreferredAddress fallback.
	updated, skipped, err := BindDefaultAddresses(db, wave.ID)
	if err != nil {
		t.Fatalf("BindDefaultAddresses error: %v", err)
	}
	if updated != 1 {
		t.Errorf("expected updated=1, got %d", updated)
	}
	if skipped != 0 {
		t.Errorf("expected skipped=0, got %d", skipped)
	}

	// Verify the record was bound.
	var updatedRec model.DispatchRecord
	if err := db.First(&updatedRec, rec.ID).Error; err != nil {
		t.Fatalf("re-fetch dispatch record: %v", err)
	}
	if updatedRec.MemberAddressID == nil {
		t.Fatal("expected member_address_id to be set, got nil")
	}
	if *updatedRec.MemberAddressID != nonDefaultAddr.ID {
		t.Errorf("expected address ID %d, got %d", nonDefaultAddr.ID, *updatedRec.MemberAddressID)
	}
	if updatedRec.Status != model.DispatchStatusPending {
		t.Errorf("expected status=%s, got %s", model.DispatchStatusPending, updatedRec.Status)
	}
}

func TestBindDefaultAddresses_KeepsExistingBinding(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	// Create a member with one address.
	member := model.Member{
		Platform:    "test",
		PlatformUID: "uid-keep-existing",
		ExtraData:   "{}",
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	addr := model.MemberAddress{
		MemberID:      member.ID,
		RecipientName: "Existing User",
		Phone:         "13900000002",
		Address:       "456 Existing St",
		IsDefault:     false,
		IsDeleted:     false,
	}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("create address: %v", err)
	}

	// Create a wave and two products.
	wave := model.Wave{WaveNo: "TASK-20260101-002", Name: "Keep Existing Wave", Status: "draft"}
	p1 := model.Product{Platform: "test", Factory: "F", FactorySKU: "sku-existing-1", Name: "Product 1"}
	p2 := model.Product{Platform: "test", Factory: "F", FactorySKU: "sku-existing-2", Name: "Product 2"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("create product 1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("create product 2: %v", err)
	}

	// Record with an existing binding — should NOT be touched.
	boundRec := model.DispatchRecord{
		WaveID:          wave.ID,
		MemberID:        member.ID,
		ProductID:       p1.ID,
		MemberAddressID: &addr.ID,
		Quantity:        1,
		Status:          model.DispatchStatusPending,
	}
	if err := db.Create(&boundRec).Error; err != nil {
		t.Fatalf("create bound dispatch record: %v", err)
	}

	// Record without an address — should get bound.
	unboundRec := model.DispatchRecord{
		WaveID:    wave.ID,
		MemberID:  member.ID,
		ProductID: p2.ID,
		Quantity:  1,
		Status:    model.DispatchStatusPendingAddress,
	}
	if err := db.Create(&unboundRec).Error; err != nil {
		t.Fatalf("create unbound dispatch record: %v", err)
	}

	updated, skipped, err := BindDefaultAddresses(db, wave.ID)
	if err != nil {
		t.Fatalf("BindDefaultAddresses error: %v", err)
	}
	if updated != 1 {
		t.Errorf("expected updated=1, got %d", updated)
	}
	if skipped != 0 {
		t.Errorf("expected skipped=0, got %d", skipped)
	}

	// Verify the already-bound record is unchanged.
	var reBound model.DispatchRecord
	if err := db.First(&reBound, boundRec.ID).Error; err != nil {
		t.Fatalf("re-fetch bound dispatch record: %v", err)
	}
	if reBound.MemberAddressID == nil {
		t.Fatal("existing binding should not be cleared")
	}
	if *reBound.MemberAddressID != addr.ID {
		t.Errorf("existing binding address ID should be %d, got %d", addr.ID, *reBound.MemberAddressID)
	}

	// Verify the unbound record got the address.
	var reUnbound model.DispatchRecord
	if err := db.First(&reUnbound, unboundRec.ID).Error; err != nil {
		t.Fatalf("re-fetch unbound dispatch record: %v", err)
	}
	if reUnbound.MemberAddressID == nil {
		t.Fatal("unbound record should now have an address")
	}
	if *reUnbound.MemberAddressID != addr.ID {
		t.Errorf("unbound record address ID should be %d, got %d", addr.ID, *reUnbound.MemberAddressID)
	}
}
