package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
)

func TestCustomerProfileNameTimelineAABAndABA(t *testing.T) {
	gdb := setupTestDB(t)
	uc, _ := setupUsecase(t, gdb)
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	service := NewCustomerNameObservationService(infra.NewProfileRepository(gdb),
		infra.NewCustomerNameObservationRepository(gdb), infra.NewCustomerNameEventRepository(gdb))

	aab, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "seed-aab", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"A", "A", "B"} {
		if _, err := service.Observe(ctx, ObserveCustomerNameInput{CustomerProfileID: aab.ID, Name: name,
			NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "api-aab-" + string(rune('0'+i)),
			Authority: "membership", ObservedAt: base.Add(time.Duration(i) * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	aabTimeline, err := uc.ListCustomerNameObservations(ctx, aab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aabTimeline) != 2 || aabTimeline[0].Value != "A" || aabTimeline[0].Count != 2 ||
		aabTimeline[1].Value != "B" || !aabTimeline[1].IsDisplayNameSource || aabTimeline[0].OriginProfileID != aab.ID {
		t.Fatalf("unexpected AAB timeline: %+v", aabTimeline)
	}

	aba, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "seed-aba", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"A", "B", "A"} {
		if _, err := service.Observe(ctx, ObserveCustomerNameInput{CustomerProfileID: aba.ID, Name: name,
			NameKind: domain.CustomerNameKindTrustedNickname, SourceEventKey: "api-aba-" + string(rune('0'+i)),
			Authority: "trusted_source", ObservedAt: base.Add(time.Duration(i) * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	abaTimeline, err := uc.ListCustomerNameObservations(ctx, aba.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(abaTimeline) != 3 || abaTimeline[0].Value != "A" || abaTimeline[1].Value != "B" ||
		abaTimeline[2].Value != "A" || !abaTimeline[2].IsDisplayNameSource {
		t.Fatalf("unexpected ABA timeline: %+v", abaTimeline)
	}
}

func TestCustomerProfileHistoricalNameSearchExplainsMatch(t *testing.T) {
	gdb := setupTestDB(t)
	uc, _ := setupUsecase(t, gdb)
	ctx := context.Background()
	profile, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "seed", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewCustomerNameObservationService(infra.NewProfileRepository(gdb),
		infra.NewCustomerNameObservationRepository(gdb), infra.NewCustomerNameEventRepository(gdb))
	base := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"Old Alias", "Current Alias"} {
		if _, err := service.Observe(ctx, ObserveCustomerNameInput{CustomerProfileID: profile.ID, Name: name,
			NameKind: domain.CustomerNameKindTrustedNickname, SourceEventKey: "search-name-" + string(rune('0'+i)),
			ObservedAt: base.Add(time.Duration(i) * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	profiles, err := uc.ListCustomerProfiles(ctx, "old alias", "", false)
	if err != nil || len(profiles) != 1 || profiles[0].MatchedHistoricalName != "Old Alias" {
		t.Fatalf("historical list search: profiles=%+v err=%v", profiles, err)
	}
	pageRepo := infra.NewListPaginationRepository(gdb)
	pageUC := NewListPaginationUseCase(pageRepo, nil, nil, nil, nil, nil)
	page, err := pageUC.ListCustomerProfilesPage(ctx, dto.CustomerProfilePageFilterInput{Keyword: "oldalias", Limit: 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].MatchedHistoricalName != "Old Alias" {
		t.Fatalf("historical page search: page=%+v err=%v", page, err)
	}
}

func TestCustomerProfilePinUnpinCASAndManualUpdateHistory(t *testing.T) {
	gdb := setupTestDB(t)
	uc, _ := setupUsecase(t, gdb)
	ctx := context.Background()
	profile, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "seed", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	service := NewCustomerNameObservationService(infra.NewProfileRepository(gdb),
		infra.NewCustomerNameObservationRepository(gdb), infra.NewCustomerNameEventRepository(gdb))
	if _, err := service.Observe(ctx, ObserveCustomerNameInput{CustomerProfileID: profile.ID, Name: "Auto Name",
		NameKind: domain.CustomerNameKindStableIdentityNickname, SourceEventKey: "pin-api-auto", ObservedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	profile, err = uc.GetCustomerProfile(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	pinInput := dto.PinCustomerDisplayNameInput{ProfileID: profile.ID, Name: "Manual Name",
		ExpectedRowVersion: profile.RowVersion, ActorRef: "tester", IdempotencyKey: "pin-api-1"}
	pinned, err := uc.PinCustomerDisplayName(ctx, pinInput)
	if err != nil || pinned.DisplayName != "Manual Name" || pinned.DisplayNameMode != domain.DisplayNameModePinned ||
		pinned.RowVersion != profile.RowVersion+1 {
		t.Fatalf("pin result=%+v err=%v", pinned, err)
	}
	if retried, err := uc.PinCustomerDisplayName(ctx, pinInput); err != nil || retried.RowVersion != pinned.RowVersion {
		t.Fatalf("idempotent pin retry=%+v err=%v", retried, err)
	}
	stale := pinInput
	stale.Name = "Other"
	stale.IdempotencyKey = "pin-api-stale"
	if _, err := uc.PinCustomerDisplayName(ctx, stale); !errors.Is(err, ErrCustomerProfileStale) {
		t.Fatalf("expected explicit stale pin error, got %v", err)
	}
	unpinned, err := uc.UnpinCustomerDisplayName(ctx, dto.UnpinCustomerDisplayNameInput{ProfileID: profile.ID,
		ExpectedRowVersion: pinned.RowVersion, ActorRef: "tester", IdempotencyKey: "unpin-api-1"})
	if err != nil || unpinned.DisplayName != "Auto Name" || unpinned.DisplayNameMode != domain.DisplayNameModeAuto ||
		unpinned.RowVersion != pinned.RowVersion+1 {
		t.Fatalf("unpin result=%+v err=%v", unpinned, err)
	}

	manual, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "Before", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := uc.UpdateCustomerProfile(ctx, dto.UpdateCustomerProfileInput{ID: manual.ID, DisplayName: "After",
		ProfileType: "buyer", ExtraData: `{"edited":true}`, ExpectedRowVersion: manual.RowVersion,
		ActorRef: "tester", IdempotencyKey: "manual-update-api"})
	if err != nil || updated.DisplayName != "After" || updated.DisplayNameMode != domain.DisplayNameModePinned ||
		updated.RowVersion != manual.RowVersion+2 {
		t.Fatalf("manual update result=%+v err=%v", updated, err)
	}
	timeline, err := uc.ListCustomerNameObservations(ctx, manual.ID)
	if err != nil || len(timeline) != 1 || timeline[0].Kind != domain.CustomerNameKindManual ||
		timeline[0].Value != "After" || !timeline[0].IsDisplayNameSource {
		t.Fatalf("manual update did not create history: timeline=%+v err=%v", timeline, err)
	}
}

func TestMergedCustomerProfileIsReadableAndRejectsNameWrites(t *testing.T) {
	gdb := setupTestDB(t)
	uc, _ := setupUsecase(t, gdb)
	ctx := context.Background()
	target, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "Target", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	source, err := uc.CreateCustomerProfile(ctx, dto.CreateCustomerProfileInput{DisplayName: "Source", ProfileType: "member"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := gdb.Model(&persistence.CustomerProfile{}).Where("id = ?", source.ID).Updates(map[string]any{
		"status": domain.CustomerProfileStatusMerged, "merged_into_profile_id": target.ID,
		"row_version": source.RowVersion + 1, "deleted_at": now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	merged, err := uc.GetCustomerProfile(ctx, source.ID)
	if err != nil || merged.Status != domain.CustomerProfileStatusMerged || merged.MergedIntoProfileID == nil || *merged.MergedIntoProfileID != target.ID {
		t.Fatalf("merged read result=%+v err=%v", merged, err)
	}
	if _, err := uc.PinCustomerDisplayName(ctx, dto.PinCustomerDisplayNameInput{ProfileID: source.ID, Name: "No",
		ExpectedRowVersion: merged.RowVersion, IdempotencyKey: "merged-pin"}); !errors.Is(err, ErrCustomerProfileMerged) {
		t.Fatalf("merged pin should be rejected, got %v", err)
	}
	if _, err := uc.UnpinCustomerDisplayName(ctx, dto.UnpinCustomerDisplayNameInput{ProfileID: source.ID,
		ExpectedRowVersion: merged.RowVersion, IdempotencyKey: "merged-unpin"}); !errors.Is(err, ErrCustomerProfileMerged) {
		t.Fatalf("merged unpin should be rejected, got %v", err)
	}
	if _, err := uc.UpdateCustomerProfile(ctx, dto.UpdateCustomerProfileInput{ID: source.ID, DisplayName: "No",
		ProfileType: source.ProfileType, ExpectedRowVersion: merged.RowVersion}); !errors.Is(err, ErrCustomerProfileMerged) {
		t.Fatalf("merged update should be rejected, got %v", err)
	}
}

func TestNicknameIdentityCannotEnterStableResolver(t *testing.T) {
	gdb := setupTestDB(t)
	repo := infra.NewProfileRepository(gdb)
	_, err := NewIdentityResolutionService(repo).ResolveStableProfile(context.Background(), StableIdentityResolutionInput{
		Namespace: "legacy", IdentityPlatform: "legacy", IdentityValue: "nickname-only", IdentityType: "username",
	})
	if err == nil || !strings.Contains(err.Error(), "not a stable resolver key") {
		t.Fatalf("nickname identity entered stable resolver: %v", err)
	}
	var count int64
	if err := gdb.Model(&persistence.CustomerProfile{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("nickname resolver created %d profiles", count)
	}
}
