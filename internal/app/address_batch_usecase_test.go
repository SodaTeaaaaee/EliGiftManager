package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func setupAddressBatchUseCase(t *testing.T, db *gorm.DB) (AddressBatchUseCase, domain.CustomerAddressRepository, domain.FulfillmentLineRepository) {
	t.Helper()

	addressRepo := infra.NewAddressRepository(db)
	fulfillmentRepo := infra.NewFulfillmentRepository(db)

	return NewAddressBatchUseCase(addressRepo, fulfillmentRepo), addressRepo, fulfillmentRepo
}

func createTestProfile(t *testing.T, db *gorm.DB, name string) uint {
	t.Helper()
	profile := persistence.CustomerProfile{
		DisplayName: name,
		ProfileType: persistence.ProfileType("member"),
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}
	return profile.ID
}

func createTestWave(t *testing.T, db *gorm.DB, waveNo string) uint {
	t.Helper()
	wave := persistence.Wave{
		WaveNo: waveNo,
		Name:   "Test Wave",
	}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("failed to create test wave: %v", err)
	}
	return wave.ID
}

func TestAddressBatchUseCase_BatchBindAddressToLines(t *testing.T) {
	db := setupTestDB(t)
	uc, addressRepo, fulfillmentRepo := setupAddressBatchUseCase(t, db)
	ctx := context.Background()

	profileID := createTestProfile(t, db, "Alice")
	waveID := createTestWave(t, db, "W-001")

	addr := &domain.CustomerAddress{
		CustomerProfileID: profileID,
		Label:             "Home",
		RecipientName:     "Alice",
		ValidationStatus:  "valid",
	}
	if err := addressRepo.Create(ctx, addr); err != nil {
		t.Fatalf("failed to create address: %v", err)
	}

	line1 := &domain.FulfillmentLine{
		WaveID:            waveID,
		CustomerProfileID: &profileID,
		Quantity:          1,
		AddressState:      string(domain.AddressStateMissing),
		LineReason:        string(domain.LineReasonEntitlement),
	}
	if err := fulfillmentRepo.Create(ctx, line1); err != nil {
		t.Fatalf("failed to create line1: %v", err)
	}

	entries := []dto.BindAddressEntry{
		{FulfillmentLineID: line1.ID, CustomerAddressID: addr.ID},
		{FulfillmentLineID: 9999, CustomerAddressID: addr.ID}, // non-existent line -> failure entry
	}

	results, err := uc.BatchBindAddressToLines(ctx, entries)
	if err != nil {
		t.Fatalf("BatchBindAddressToLines returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("expected first entry to succeed, got error: %s", results[0].ErrorMessage)
	}
	if results[0].CustomerAddressID == nil || *results[0].CustomerAddressID != addr.ID {
		t.Errorf("expected first entry to be bound to address %d", addr.ID)
	}

	if results[1].Success {
		t.Errorf("expected second entry to fail (non-existent line)")
	}
	if results[1].ErrorMessage == "" {
		t.Errorf("expected error message on failed entry")
	}

	updatedLine, err := fulfillmentRepo.FindByID(ctx, line1.ID)
	if err != nil {
		t.Fatalf("failed to reload line1: %v", err)
	}
	if updatedLine.CustomerAddressID == nil || *updatedLine.CustomerAddressID != addr.ID {
		t.Errorf("expected line1 to have address bound in DB")
	}
	if updatedLine.AddressState != string(domain.AddressStateReady) {
		t.Errorf("expected line1 address state to become ready, got %q", updatedLine.AddressState)
	}
}

func TestAddressBatchUseCase_BindDefaultAddressesForWave(t *testing.T) {
	db := setupTestDB(t)
	uc, addressRepo, fulfillmentRepo := setupAddressBatchUseCase(t, db)
	ctx := context.Background()

	profileID := createTestProfile(t, db, "Bob")
	waveID := createTestWave(t, db, "W-002")

	defaultAddr := &domain.CustomerAddress{
		CustomerProfileID: profileID,
		Label:             "Default",
		RecipientName:     "Bob",
		ValidationStatus:  "valid",
		IsDefault:         true,
	}
	if err := addressRepo.Create(ctx, defaultAddr); err != nil {
		t.Fatalf("failed to create default address: %v", err)
	}

	otherAddr := &domain.CustomerAddress{
		CustomerProfileID: profileID,
		Label:             "Other",
		RecipientName:     "Bob",
		ValidationStatus:  "valid",
	}
	if err := addressRepo.Create(ctx, otherAddr); err != nil {
		t.Fatalf("failed to create other address: %v", err)
	}

	// Line missing an address -> should get bound to default.
	missingLine := &domain.FulfillmentLine{
		WaveID:            waveID,
		CustomerProfileID: &profileID,
		Quantity:          1,
		AddressState:      string(domain.AddressStateMissing),
		LineReason:        string(domain.LineReasonEntitlement),
	}
	if err := fulfillmentRepo.Create(ctx, missingLine); err != nil {
		t.Fatalf("failed to create missing line: %v", err)
	}

	// Line already bound (ready) -> should be left untouched.
	readyLine := &domain.FulfillmentLine{
		WaveID:            waveID,
		CustomerProfileID: &profileID,
		CustomerAddressID: &otherAddr.ID,
		Quantity:          1,
		AddressState:      string(domain.AddressStateReady),
		LineReason:        string(domain.LineReasonEntitlement),
	}
	if err := fulfillmentRepo.Create(ctx, readyLine); err != nil {
		t.Fatalf("failed to create ready line: %v", err)
	}

	// Line with no customer profile -> should surface as a failure result, not a bind.
	noProfileLine := &domain.FulfillmentLine{
		WaveID:       waveID,
		Quantity:     1,
		AddressState: string(domain.AddressStateMissing),
		LineReason:   string(domain.LineReasonEntitlement),
	}
	if err := fulfillmentRepo.Create(ctx, noProfileLine); err != nil {
		t.Fatalf("failed to create no-profile line: %v", err)
	}

	results, err := uc.BindDefaultAddressesForWave(ctx, waveID)
	if err != nil {
		t.Fatalf("BindDefaultAddressesForWave returned error: %v", err)
	}

	// Only the two "missing" lines should produce results; the already-bound ready line is skipped.
	if len(results) != 2 {
		t.Fatalf("expected 2 results (missing lines only), got %d", len(results))
	}

	var missingResult, noProfileResult *dto.AddressBatchItemResult
	for i := range results {
		switch results[i].FulfillmentLineID {
		case missingLine.ID:
			missingResult = &results[i]
		case noProfileLine.ID:
			noProfileResult = &results[i]
		}
	}

	if missingResult == nil || !missingResult.Success {
		t.Fatalf("expected missing line to be bound successfully, got %+v", missingResult)
	}
	if missingResult.CustomerAddressID == nil || *missingResult.CustomerAddressID != defaultAddr.ID {
		t.Errorf("expected missing line bound to default address %d, got %+v", defaultAddr.ID, missingResult.CustomerAddressID)
	}

	if noProfileResult == nil || noProfileResult.Success {
		t.Fatalf("expected no-profile line to fail, got %+v", noProfileResult)
	}

	updatedMissing, err := fulfillmentRepo.FindByID(ctx, missingLine.ID)
	if err != nil {
		t.Fatalf("failed to reload missing line: %v", err)
	}
	if updatedMissing.CustomerAddressID == nil || *updatedMissing.CustomerAddressID != defaultAddr.ID {
		t.Errorf("expected missing line to be bound to default address in DB")
	}

	updatedReady, err := fulfillmentRepo.FindByID(ctx, readyLine.ID)
	if err != nil {
		t.Fatalf("failed to reload ready line: %v", err)
	}
	if updatedReady.CustomerAddressID == nil || *updatedReady.CustomerAddressID != otherAddr.ID {
		t.Errorf("expected ready line to remain bound to its original address, unchanged")
	}
}
