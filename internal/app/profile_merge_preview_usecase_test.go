package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProfileMergePreviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func TestProfileMergePreviewUseCase_SurfacesConflictAndDuplicateIdentity(t *testing.T) {
	db := setupProfileMergePreviewTestDB(t)
	ctx := context.Background()

	profileRepo := infra.NewProfileRepository(db)
	addressRepo := infra.NewAddressRepository(db)
	uc := NewProfileMergePreviewUseCase(profileRepo, addressRepo)

	src := &domain.CustomerProfile{DisplayName: "Alice (alt)", ProfileType: "buyer"}
	if err := profileRepo.Create(ctx, src); err != nil {
		t.Fatalf("create source profile: %v", err)
	}
	tgt := &domain.CustomerProfile{DisplayName: "Alice", ProfileType: "member"}
	if err := profileRepo.Create(ctx, tgt); err != nil {
		t.Fatalf("create target profile: %v", err)
	}

	// Shared identity on both sides -> must be flagged as a duplicate.
	sharedIdentity := &domain.CustomerIdentity{
		CustomerProfileID: src.ID,
		IdentityPlatform:  "taobao",
		IdentityValue:     "uid-123",
		IdentityType:      "platform_uid",
	}
	if err := profileRepo.CreateIdentity(ctx, sharedIdentity); err != nil {
		t.Fatalf("create source identity: %v", err)
	}
	tgtIdentity := &domain.CustomerIdentity{
		CustomerProfileID: tgt.ID,
		IdentityPlatform:  "taobao",
		IdentityValue:     "uid-123",
		IdentityType:      "platform_uid",
	}
	if err := profileRepo.CreateIdentity(ctx, tgtIdentity); err != nil {
		t.Fatalf("create target identity: %v", err)
	}

	// A source-only address that should show up on the source side.
	srcAddr := &domain.CustomerAddress{
		CustomerProfileID: src.ID,
		RecipientName:     "Alice A",
		Country:           "CN",
		AddressLine1:      "1 Test Rd",
	}
	if err := addressRepo.Create(ctx, srcAddr); err != nil {
		t.Fatalf("create source address: %v", err)
	}

	result, err := uc.PreviewMergeProfiles(ctx, src.ID, tgt.ID)
	if err != nil {
		t.Fatalf("PreviewMergeProfiles: %v", err)
	}

	if result.Source.ProfileID != src.ID || result.Target.ProfileID != tgt.ID {
		t.Fatalf("unexpected profile ids: source=%d target=%d", result.Source.ProfileID, result.Target.ProfileID)
	}
	if len(result.Source.Identities) != 1 || len(result.Target.Identities) != 1 {
		t.Fatalf("expected 1 identity per side, got source=%d target=%d", len(result.Source.Identities), len(result.Target.Identities))
	}
	if len(result.Source.Addresses) != 1 {
		t.Fatalf("expected 1 address on source side, got %d", len(result.Source.Addresses))
	}
	if len(result.Target.Addresses) != 0 {
		t.Fatalf("expected 0 addresses on target side, got %d", len(result.Target.Addresses))
	}

	// DisplayName differs ("Alice (alt)" vs "Alice") -> must appear as a conflict.
	foundDisplayNameConflict := false
	for _, c := range result.Conflicts {
		if c.Field == "displayName" {
			foundDisplayNameConflict = true
			if c.SourceValue != "Alice (alt)" || c.TargetValue != "Alice" {
				t.Fatalf("unexpected displayName conflict values: %+v", c)
			}
		}
	}
	if !foundDisplayNameConflict {
		t.Fatal("expected a displayName conflict to be surfaced")
	}

	if len(result.DuplicateIdentityValues) != 1 || result.DuplicateIdentityValues[0] != "taobao::uid-123" {
		t.Fatalf("expected duplicate identity taobao::uid-123 to be flagged, got %v", result.DuplicateIdentityValues)
	}

	if result.MovedIdentityCount != 1 {
		t.Fatalf("expected MovedIdentityCount=1, got %d", result.MovedIdentityCount)
	}
	if result.MovedAddressCount != 1 {
		t.Fatalf("expected MovedAddressCount=1, got %d", result.MovedAddressCount)
	}
}

func TestProfileMergePreviewUseCase_RejectsSelfMerge(t *testing.T) {
	db := setupProfileMergePreviewTestDB(t)
	ctx := context.Background()

	profileRepo := infra.NewProfileRepository(db)
	addressRepo := infra.NewAddressRepository(db)
	uc := NewProfileMergePreviewUseCase(profileRepo, addressRepo)

	p := &domain.CustomerProfile{DisplayName: "Solo", ProfileType: "member"}
	if err := profileRepo.Create(ctx, p); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	if _, err := uc.PreviewMergeProfiles(ctx, p.ID, p.ID); err == nil {
		t.Fatal("expected error when previewing a merge of a profile into itself")
	}
}
