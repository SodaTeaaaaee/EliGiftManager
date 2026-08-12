package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func TestListCustomerProfileOriginsActiveStableSortAndEmpty(t *testing.T) {
	ctx, uc, profiles, origins, _ := newCustomerProfileOriginAPIFixture(t)
	profile := createCustomerProfileOriginAPIProfile(t, ctx, profiles, "active")
	emptyProfile := createCustomerProfileOriginAPIProfile(t, ctx, profiles, "empty")

	integrationID := uint(41)
	documentID := uint(82)
	lastSeenAt := time.Date(2026, time.July, 15, 12, 30, 0, 0, time.UTC)
	first := &domain.CustomerProfileOrigin{
		CustomerProfileID:          profile.ID,
		OriginKind:                 domain.CustomerOriginKindRetailOrder,
		SourceIntegrationProfileID: &integrationID,
		SourceDocumentID:           &documentID,
		ExternalRef:                "order-z",
		LastSeenAt:                 &lastSeenAt,
	}
	second := &domain.CustomerProfileOrigin{
		CustomerProfileID: profile.ID,
		OriginKind:        "manual",
		ExternalRef:       "origin-a",
	}
	if err := origins.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := origins.Create(ctx, second); err != nil {
		t.Fatal(err)
	}

	got, err := uc.ListCustomerProfileOrigins(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("origins are not stably sorted by id: %+v", got)
	}
	if got[0].CustomerProfileID != profile.ID || got[0].OriginKind != first.OriginKind ||
		got[0].SourceIntegrationProfileID == nil || *got[0].SourceIntegrationProfileID != integrationID ||
		got[0].SourceDocumentID == nil || *got[0].SourceDocumentID != documentID ||
		got[0].ExternalRef != first.ExternalRef || got[0].LastSeenAt == nil || !got[0].LastSeenAt.Equal(lastSeenAt) ||
		got[0].CreatedAt.IsZero() {
		t.Fatalf("origin DTO does not preserve the public fields: %+v", got[0])
	}

	empty, err := uc.ListCustomerProfileOrigins(ctx, emptyProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty origin history must be an allocated empty list, got %#v", empty)
	}
}

func TestListCustomerProfileOriginsNonexistentProfile(t *testing.T) {
	ctx, uc, _, _, _ := newCustomerProfileOriginAPIFixture(t)

	if _, err := uc.ListCustomerProfileOrigins(ctx, 999999); err == nil {
		t.Fatal("expected nonexistent profile lookup to fail")
	}
}

func TestListCustomerProfileOriginsMergedProfileIncludesCurrentAndLedgerOrigins(t *testing.T) {
	ctx, uc, profiles, origins, db := newCustomerProfileOriginAPIFixture(t)
	source := createCustomerProfileOriginAPIProfile(t, ctx, profiles, "merged source")
	target := createCustomerProfileOriginAPIProfile(t, ctx, profiles, "active target")

	moved := &domain.CustomerProfileOrigin{CustomerProfileID: source.ID, OriginKind: "manual", ExternalRef: "moved"}
	current := &domain.CustomerProfileOrigin{CustomerProfileID: source.ID, OriginKind: "manual", ExternalRef: "current"}
	if err := origins.Create(ctx, moved); err != nil {
		t.Fatal(err)
	}
	if err := origins.Create(ctx, current); err != nil {
		t.Fatal(err)
	}
	moved.CustomerProfileID = target.ID
	if err := origins.Update(ctx, moved); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	record := &persistence.CustomerMergeRecord{
		SourceProfileID: source.ID,
		TargetProfileID: target.ID,
		Payload:         "{}",
		OperationKey:    "origin-api-merge",
		Status:          domain.MergeRecordStatusCompleted,
		RowVersion:      1,
		CompletedAt:     &now,
		CreatedAt:       now,
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&persistence.MergeMovedEntity{
		MergeRecordID:   record.ID,
		EntityType:      domain.MergeEntityOrigin,
		EntityID:        moved.ID,
		FromProfileID:   source.ID,
		ToProfileID:     target.ID,
		MoveOrder:       1,
		MutationKind:    "merge_reassign",
		SnapshotVersion: 1,
		CreatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&persistence.CustomerProfile{}).Where("id = ?", source.ID).Updates(map[string]any{
		"status":                 domain.CustomerProfileStatusMerged,
		"merged_into_profile_id": target.ID,
		"row_version":            gorm.Expr("row_version + 1"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&persistence.CustomerProfile{}, source.ID).Error; err != nil {
		t.Fatal(err)
	}

	got, err := uc.ListCustomerProfileOrigins(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != moved.ID || got[1].ID != current.ID {
		t.Fatalf("merged history must include current and active-ledger origins in id order: %+v", got)
	}
	if got[0].CustomerProfileID != target.ID || got[1].CustomerProfileID != source.ID {
		t.Fatalf("origin DTOs must report their current owners: %+v", got)
	}

	targetOrigins, err := uc.ListCustomerProfileOrigins(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetOrigins) != 1 || targetOrigins[0].ID != moved.ID {
		t.Fatalf("active profile reads must remain limited to currently owned origins: %+v", targetOrigins)
	}
}

func newCustomerProfileOriginAPIFixture(t *testing.T) (context.Context, *CustomerProfileUseCase, domain.CustomerProfileRepository, domain.CustomerProfileOriginRepository, *gorm.DB) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "customer-profile-origin-api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			t.Errorf("get SQL database: %v", dbErr)
			return
		}
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("close SQL database: %v", closeErr)
		}
	})

	profiles := infra.NewProfileRepository(db)
	origins := infra.NewCustomerProfileOriginRepository(db)
	uc := WithCustomerProfileOrigins(NewCustomerProfileUseCase(
		profiles,
		infra.NewAddressRepository(db),
		nil,
		infra.NewMergeSuggestionRepository(db),
	), origins)
	return context.Background(), uc, profiles, origins, db
}

func createCustomerProfileOriginAPIProfile(t *testing.T, ctx context.Context, profiles domain.CustomerProfileRepository, displayName string) domain.CustomerProfile {
	t.Helper()
	profile := domain.CustomerProfile{
		DisplayName: displayName,
		ProfileType: "member",
		Status:      domain.CustomerProfileStatusActive,
		RowVersion:  1,
	}
	if err := profiles.Create(ctx, &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}
