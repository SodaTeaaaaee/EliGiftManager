package app

import (
	"context"
	"os"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys = ON;")

	// Migrate models
	err = db.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerMergeRecord{},
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.FulfillmentLine{},
		&persistence.WaveDemandAssignment{},
		&persistence.MergeSuggestion{},
	)
	if err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	return db
}

func setupUsecase(t *testing.T, db *gorm.DB) (*CustomerProfileUseCase, ProfileMergeUseCase) {
	t.Helper()

	// Set env to mock dev mode so resolveDataDir resolves to package relative path
	t.Setenv("devserver", "1")
	t.Cleanup(func() {
		os.RemoveAll("data")
	})

	profileRepo := infra.NewCustomerMergeProfileRepository(db)
	addressRepo := infra.NewCustomerMergeAddressRepository(db)
	demandRepo := infra.NewCustomerMergeDemandRepository(db)
	mergeRepo := infra.NewCustomerMergeRecordRepository(db)

	settingsSvc := service.NewSettingsService()

	suggestionRepo := infra.NewMergeSuggestionRepository(db)
	cpUseCase := NewCustomerProfileUseCase(profileRepo, addressRepo, settingsSvc, suggestionRepo)
	mergeUseCase := NewProfileMergeUseCase(profileRepo, addressRepo, demandRepo, mergeRepo)

	return cpUseCase, mergeUseCase
}

func TestCustomerProfileUseCase_CRUD(t *testing.T) {
	db := setupTestDB(t)
	uc, _ := setupUsecase(t, db)

	// Test Create
	input := dto.CreateCustomerProfileInput{
		DisplayName: "Alice Smith",
		ProfileType: "member",
		ExtraData:   `{"level":"gold"}`,
	}
	profile, err := uc.CreateCustomerProfile(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}
	if profile.DisplayName != "Alice Smith" {
		t.Errorf("expected DisplayName = Alice Smith, got %s", profile.DisplayName)
	}
	if profile.ProfileType != "member" {
		t.Errorf("expected ProfileType = member, got %s", profile.ProfileType)
	}

	// Test Get
	got, err := uc.GetCustomerProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("GetCustomerProfile failed: %v", err)
	}
	if got.ID != profile.ID || got.DisplayName != "Alice Smith" {
		t.Errorf("GetCustomerProfile returned wrong profile: %+v", got)
	}

	// Test Update
	updateInput := dto.UpdateCustomerProfileInput{
		ID:          profile.ID,
		DisplayName: "Alice Smith Updated",
		ProfileType: "buyer",
		ExtraData:   `{"level":"gold","updated":true}`,
	}
	updated, err := uc.UpdateCustomerProfile(context.Background(), updateInput)
	if err != nil {
		t.Fatalf("UpdateCustomerProfile failed: %v", err)
	}
	if updated.DisplayName != "Alice Smith Updated" {
		t.Errorf("expected updated DisplayName = Alice Smith Updated, got %s", updated.DisplayName)
	}

	// Test List
	list, err := uc.ListCustomerProfiles(context.Background(), "", "", false)
	if err != nil {
		t.Fatalf("ListCustomerProfiles failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 profile, got %d", len(list))
	}
	if list[0].DisplayName != "Alice Smith Updated" {
		t.Errorf("expected displayName Alice Smith Updated, got %s", list[0].DisplayName)
	}

	// Test Delete
	err = uc.DeleteCustomerProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("DeleteCustomerProfile failed: %v", err)
	}

	// Fetching should fail or return deleted profile depending on soft-delete
	_, err = uc.GetCustomerProfile(context.Background(), profile.ID)
	if err == nil {
		t.Error("expected error retrieving soft-deleted profile, but got nil")
	}
}

func TestCustomerProfileUseCase_Identities(t *testing.T) {
	db := setupTestDB(t)
	uc, _ := setupUsecase(t, db)

	// Create profile
	profile, err := uc.CreateCustomerProfile(context.Background(), dto.CreateCustomerProfileInput{
		DisplayName: "Bob Jones",
		ProfileType: "buyer",
	})
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}

	// Add Identity
	identInput := dto.CreateCustomerIdentityInput{
		CustomerProfileID: profile.ID,
		IdentityPlatform:  "patreon",
		IdentityValue:     "patreon_user_bob",
		IdentityType:      "platform_uid",
		IsPrimary:         true,
	}
	ident, err := uc.AddCustomerIdentity(context.Background(), identInput)
	if err != nil {
		t.Fatalf("AddCustomerIdentity failed: %v", err)
	}
	if ident.IdentityPlatform != "patreon" || ident.IdentityValue != "patreon_user_bob" {
		t.Errorf("AddCustomerIdentity returned wrong details: %+v", ident)
	}

	// Reload and verify
	p, err := uc.GetCustomerProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("GetCustomerProfile failed: %v", err)
	}
	if len(p.Identities) != 1 {
		t.Errorf("expected 1 identity, got %d", len(p.Identities))
	}
	if p.Identities[0].IdentityValue != "patreon_user_bob" {
		t.Errorf("expected IdentityValue patreon_user_bob, got %s", p.Identities[0].IdentityValue)
	}

	// Delete Identity
	err = uc.DeleteCustomerIdentity(context.Background(), ident.ID)
	if err != nil {
		t.Fatalf("DeleteCustomerIdentity failed: %v", err)
	}

	// Verify identity is deleted
	p, err = uc.GetCustomerProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("GetCustomerProfile failed: %v", err)
	}
	if len(p.Identities) != 0 {
		t.Errorf("expected 0 identities after delete, got %d", len(p.Identities))
	}
}

func TestCustomerProfileUseCase_AutoMergeSuggestions(t *testing.T) {
	db := setupTestDB(t)
	uc, mergeUC := setupUsecase(t, db)

	// Configure settings
	err := uc.SaveSettings(context.Background(), dto.SystemSettingsDTO{
		AutoMergeCrossPlatform: true,
		AutoMergeByEmail:       true,
		AutoMergeByPhone:       true,
	})
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	// Create profile 1 (Older, target)
	p1, err := uc.CreateCustomerProfile(context.Background(), dto.CreateCustomerProfileInput{
		DisplayName: "P1 Target",
		ProfileType: "mixed",
	})
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}

	// Create profile 2 (Newer, source)
	p2, err := uc.CreateCustomerProfile(context.Background(), dto.CreateCustomerProfileInput{
		DisplayName: "P2 Source",
		ProfileType: "buyer",
	})
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}

	// 1. Test AutoMerge By Email
	// Add identical emails to both profiles
	_, err = uc.AddCustomerIdentity(context.Background(), dto.CreateCustomerIdentityInput{
		CustomerProfileID: p1.ID,
		IdentityPlatform:  "patreon",
		IdentityValue:     "user@example.com",
		IdentityType:      "email",
	})
	if err != nil {
		t.Fatalf("AddCustomerIdentity for p1 failed: %v", err)
	}

	_, err = uc.AddCustomerIdentity(context.Background(), dto.CreateCustomerIdentityInput{
		CustomerProfileID: p2.ID,
		IdentityPlatform:  "gumroad",
		IdentityValue:     "user@example.com",
		IdentityType:      "email",
	})
	if err != nil {
		t.Fatalf("AddCustomerIdentity for p2 failed: %v", err)
	}

	// Auto-merge triggers on AddCustomerIdentity automatically!
	// Retrieve suggestions
	suggestions, err := uc.GetMergeSuggestions(context.Background())
	if err != nil {
		t.Fatalf("GetMergeSuggestions failed: %v", err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}

	s := suggestions[0]
	if s.SourceProfileID != p2.ID || s.TargetProfileID != p1.ID {
		t.Errorf("expected source=%d, target=%d, got source=%d, target=%d", p2.ID, p1.ID, s.SourceProfileID, s.TargetProfileID)
	}

	// Perform merge using profileMergeUseCase
	mergeRes, err := mergeUC.MergeProfiles(context.Background(), dto.MergeProfilesInput{
		SourceProfileID: s.SourceProfileID,
		TargetProfileID: s.TargetProfileID,
	})
	if err != nil {
		t.Fatalf("MergeProfiles failed: %v", err)
	}
	if mergeRes.MigratedIdentityCount != 1 {
		t.Errorf("expected 1 migrated identity, got %d", mergeRes.MigratedIdentityCount)
	}

	// Verify source profile was soft deleted
	_, err = uc.profileRepo.FindByID(context.Background(), p2.ID)
	if err == nil {
		t.Error("expected source profile to be deleted")
	}

	// Verify target profile now has both identities
	targetProfile, err := uc.GetCustomerProfile(context.Background(), p1.ID)
	if err != nil {
		t.Fatalf("GetCustomerProfile failed: %v", err)
	}
	if len(targetProfile.Identities) != 2 {
		t.Errorf("expected 2 identities for target profile, got %d", len(targetProfile.Identities))
	}
}

func TestCustomerProfileUseCase_AutoMergeSuggestionsByPhone(t *testing.T) {
	db := setupTestDB(t)
	uc, _ := setupUsecase(t, db)

	// Configure settings
	err := uc.SaveSettings(context.Background(), dto.SystemSettingsDTO{
		AutoMergeCrossPlatform: true,
		AutoMergeByEmail:       false,
		AutoMergeByPhone:       true,
	})
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	// Create profile 1 (Target)
	p1, err := uc.CreateCustomerProfile(context.Background(), dto.CreateCustomerProfileInput{
		DisplayName: "P1 Target",
		ProfileType: "mixed",
	})
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}

	// Create profile 2 (Source)
	p2, err := uc.CreateCustomerProfile(context.Background(), dto.CreateCustomerProfileInput{
		DisplayName: "P2 Source",
		ProfileType: "buyer",
	})
	if err != nil {
		t.Fatalf("CreateCustomerProfile failed: %v", err)
	}

	// Add identical phones to both profiles using the direct GORM DB or address repository
	addrRepo := infra.NewAddressRepository(db)
	err = addrRepo.Create(context.Background(), &domain.CustomerAddress{
		CustomerProfileID: p1.ID,
		Label:             "Home",
		RecipientName:     "P1 Recipient",
		Phone:             "+8612345678901",
		AddressLine1:      "Test Road 1",
	})
	if err != nil {
		t.Fatalf("create address 1 failed: %v", err)
	}

	err = addrRepo.Create(context.Background(), &domain.CustomerAddress{
		CustomerProfileID: p2.ID,
		Label:             "Work",
		RecipientName:     "P2 Recipient",
		Phone:             "+8612345678901",
		AddressLine1:      "Test Road 2",
	})
	if err != nil {
		t.Fatalf("create address 2 failed: %v", err)
	}

	// Retrieve suggestions (which triggers DetectMergeSuggestions)
	suggestions, err := uc.GetMergeSuggestions(context.Background())
	if err != nil {
		t.Fatalf("GetMergeSuggestions failed: %v", err)
	}

	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}

	s := suggestions[0]
	if s.SourceProfileID != p2.ID || s.TargetProfileID != p1.ID {
		t.Errorf("expected source=%d, target=%d, got source=%d, target=%d", p2.ID, p1.ID, s.SourceProfileID, s.TargetProfileID)
	}

	// Test DismissMergeSuggestion
	err = uc.DismissMergeSuggestion(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("DismissMergeSuggestion failed: %v", err)
	}

	// Retrieve suggestions again - should be empty as it is dismissed
	suggestions2, err := uc.GetMergeSuggestions(context.Background())
	if err != nil {
		t.Fatalf("GetMergeSuggestions 2 failed: %v", err)
	}
	if len(suggestions2) != 0 {
		t.Errorf("expected 0 pending suggestions after dismissal, got %d", len(suggestions2))
	}
}
