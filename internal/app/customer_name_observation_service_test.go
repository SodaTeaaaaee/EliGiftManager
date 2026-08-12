package app

import (
	"context"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCustomerNameServiceTest(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&persistence.CustomerProfile{}, &persistence.CustomerNameObservation{}, &persistence.CustomerNameEvent{}); err != nil {
		t.Fatal(err)
	}
	seedEnabledFeaturePolicyForFocusedDB(t, db)
	profile := &domain.CustomerProfile{DisplayName: "seed", ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := infra.NewProfileRepository(db).Create(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	return db, profile.ID
}

func observeNameInTx(t *testing.T, db *gorm.DB, input ObserveCustomerNameInput) *ObserveCustomerNameResult {
	t.Helper()
	var result *ObserveCustomerNameResult
	err := db.Transaction(func(tx *gorm.DB) error {
		service := NewCustomerNameObservationService(
			infra.NewProfileRepository(tx),
			infra.NewCustomerNameObservationRepository(tx),
			infra.NewCustomerNameEventRepository(tx),
		)
		var err error
		result, err = service.Observe(context.Background(), input)
		return err
	})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	return result
}

func activeNameEpisodes(t *testing.T, db *gorm.DB, profileID uint) []domain.CustomerNameObservation {
	t.Helper()
	rows, err := infra.NewCustomerNameObservationRepository(db).ListByProfile(context.Background(), profileID)
	if err != nil {
		t.Fatal(err)
	}
	result := make([]domain.CustomerNameObservation, 0, len(rows))
	for _, row := range rows {
		if row.IsActive {
			result = append(result, row)
		}
	}
	return result
}

func TestCustomerNameObservationEpisodes_AAB(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"A", "A", "B"} {
		observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: name, NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "aab-" + name + string(rune('0'+i)), ObservedAt: base.Add(time.Duration(i) * time.Hour)})
	}
	episodes := activeNameEpisodes(t, db, profileID)
	if len(episodes) != 2 || episodes[0].Name != "A" || episodes[0].ObservationCount != 2 || episodes[1].Name != "B" {
		t.Fatalf("episodes = %+v", episodes)
	}
	profile, _ := infra.NewProfileRepository(db).FindByID(context.Background(), profileID)
	if profile.DisplayName != "B" {
		t.Fatalf("DisplayName=%q, want B", profile.DisplayName)
	}
}

func TestCustomerNameObservationEpisodes_ABA(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"A", "B", "A"} {
		observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: name, NameKind: domain.CustomerNameKindTrustedNickname, SourceEventKey: "aba-" + string(rune('0'+i)), ObservedAt: base.Add(time.Duration(i) * time.Hour)})
	}
	episodes := activeNameEpisodes(t, db, profileID)
	if len(episodes) != 3 || episodes[0].Name != "A" || episodes[1].Name != "B" || episodes[2].Name != "A" {
		t.Fatalf("episodes = %+v", episodes)
	}
}

func TestCustomerNameObservationReplayIsNoOp(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	input := ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "A", NameKind: domain.CustomerNameKindTrustedNickname, SourceEventKey: "replay-1", ObservedAt: time.Now().UTC()}
	first := observeNameInTx(t, db, input)
	second := observeNameInTx(t, db, input)
	if !first.Created || second.Created {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	var eventCount int64
	if err := db.Model(&persistence.CustomerNameEvent{}).Count(&eventCount).Error; err != nil {
		t.Fatal(err)
	}
	if eventCount != 2 { // observed + projection
		t.Fatalf("event count=%d, want 2", eventCount)
	}
}

func TestCustomerNameObservationOutOfOrderRebuildsEpisodesWithoutOverwritingLatest(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "A", NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "oo-1", ObservedAt: base})
	observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "A", NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "oo-3", ObservedAt: base.Add(2 * time.Hour)})
	observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "B", NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "oo-2", ObservedAt: base.Add(time.Hour)})
	episodes := activeNameEpisodes(t, db, profileID)
	if len(episodes) != 3 || episodes[0].Name != "A" || episodes[1].Name != "B" || episodes[2].Name != "A" {
		t.Fatalf("episodes = %+v", episodes)
	}
	profile, _ := infra.NewProfileRepository(db).FindByID(context.Background(), profileID)
	if profile.DisplayName != "A" {
		t.Fatalf("out-of-order event overwrote latest display name: %q", profile.DisplayName)
	}
}

func TestCustomerNameObservationPinAndUnpin(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "A", NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "pin-a", ObservedAt: base})
	if err := db.Transaction(func(tx *gorm.DB) error {
		return NewCustomerNameObservationService(infra.NewProfileRepository(tx), infra.NewCustomerNameObservationRepository(tx), infra.NewCustomerNameEventRepository(tx)).Pin(context.Background(), profileID, "Manual", "operator", "pin-event", base.Add(time.Hour))
	}); err != nil {
		t.Fatal(err)
	}
	observeNameInTx(t, db, ObserveCustomerNameInput{CustomerProfileID: profileID, Name: "B", NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "pin-b", ObservedAt: base.Add(2 * time.Hour)})
	profile, _ := infra.NewProfileRepository(db).FindByID(context.Background(), profileID)
	if profile.DisplayName != "Manual" || profile.DisplayNameMode != domain.DisplayNameModePinned {
		t.Fatalf("pinned profile = %+v", profile)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return NewCustomerNameObservationService(infra.NewProfileRepository(tx), infra.NewCustomerNameObservationRepository(tx), infra.NewCustomerNameEventRepository(tx)).Unpin(context.Background(), profileID, "operator", "unpin-event", base.Add(3*time.Hour))
	}); err != nil {
		t.Fatal(err)
	}
	profile, _ = infra.NewProfileRepository(db).FindByID(context.Background(), profileID)
	if profile.DisplayName != "B" || profile.DisplayNameMode != domain.DisplayNameModeAuto {
		t.Fatalf("unpinned profile = %+v", profile)
	}
}

func TestCustomerNameObservationRebuildsAlreadyImportedLegacyRawPayload(t *testing.T) {
	db, profileID := setupCustomerNameServiceTest(t)
	base := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	legacyKey := "legacy:member_nicknames:1:2025-12-01"
	legacyObservation := persistence.CustomerNameObservation{
		CustomerProfileID: profileID, Name: "Legacy A", NormalizedName: "legacy a",
		SourceEventKey: legacyKey, EpisodeKey: "legacy:members:1", ObservationCount: 1,
		NameKind: domain.CustomerNameKindTrustedNickname, Authority: "legacy_member_nicknames",
		ObservedAt: &base, FirstSeenAt: &base, LastSeenAt: &base, IsActive: true,
	}
	if err := db.Create(&legacyObservation).Error; err != nil {
		t.Fatal(err)
	}
	legacyEvent := persistence.CustomerNameEvent{
		EventKey: legacyKey, CustomerProfileID: profileID, ObservationID: &legacyObservation.ID,
		EventKind: customerNameEventObserved, NewName: "Legacy A", ReasonCode: "legacy_import",
		ActorRef: "legacy_customer_migration", Payload: `{"id":1,"nickname":"Legacy A"}`, CreatedAt: base,
	}
	if err := db.Create(&legacyEvent).Error; err != nil {
		t.Fatal(err)
	}
	observeNameInTx(t, db, ObserveCustomerNameInput{
		CustomerProfileID: profileID, Name: "Current B", NameKind: domain.CustomerNameKindTrustedNickname,
		SourceEventKey: "live-after-legacy", ObservedAt: base.Add(time.Hour),
	})
	episodes := activeNameEpisodes(t, db, profileID)
	if len(episodes) != 2 || episodes[0].Name != "Legacy A" || episodes[1].Name != "Current B" {
		t.Fatalf("legacy-compatible episodes = %+v", episodes)
	}
}
