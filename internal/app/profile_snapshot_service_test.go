package app

import (
	"encoding/json"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── CaptureProfileSnapshot / ParseProfileSnapshot round-trip ──

func TestProfileSnapshotCaptureAndParse(t *testing.T) {
	t.Parallel()
	profile := &domain.IntegrationProfile{
		ID:                      42,
		ProfileKey:              "test.key",
		TrackingSyncMode:        "api_push",
		ClosurePolicy:           "close_after_sync",
		AllowsManualClosure:     true,
		RequiresCarrierMapping:  false,
		RequiresExternalOrderNo: true,
		SupportsPartialShipment: false,
		ConnectorKey:            "connector_a",
		SupportsAPIExport:       true,
	}

	raw := CaptureProfileSnapshot(profile)
	if raw == "" {
		t.Fatal("CaptureProfileSnapshot returned empty string")
	}

	// Verify it is valid JSON
	var check map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &check); err != nil {
		t.Fatalf("CaptureProfileSnapshot output is not valid JSON: %v", err)
	}

	snap, err := ParseProfileSnapshot(raw)
	if err != nil {
		t.Fatalf("ParseProfileSnapshot error: %v", err)
	}
	if snap == nil {
		t.Fatal("ParseProfileSnapshot returned nil")
	}

	if snap.ProfileID != 42 {
		t.Errorf("ProfileID = %d, want 42", snap.ProfileID)
	}
	if snap.ProfileKey != "test.key" {
		t.Errorf("ProfileKey = %q, want %q", snap.ProfileKey, "test.key")
	}
	if snap.TrackingSyncMode != "api_push" {
		t.Errorf("TrackingSyncMode = %q, want %q", snap.TrackingSyncMode, "api_push")
	}
	if snap.ClosurePolicy != "close_after_sync" {
		t.Errorf("ClosurePolicy = %q, want %q", snap.ClosurePolicy, "close_after_sync")
	}
	if !snap.AllowsManualClosure {
		t.Error("AllowsManualClosure = false, want true")
	}
	if snap.RequiresCarrierMapping {
		t.Error("RequiresCarrierMapping = true, want false")
	}
	if !snap.RequiresExternalOrderNo {
		t.Error("RequiresExternalOrderNo = false, want true")
	}
	if snap.ConnectorKey != "connector_a" {
		t.Errorf("ConnectorKey = %q, want %q", snap.ConnectorKey, "connector_a")
	}
	if !snap.SupportsAPIExport {
		t.Error("SupportsAPIExport = false, want true")
	}
}

func TestParseProfileSnapshotEmptyReturnsNil(t *testing.T) {
	t.Parallel()
	snap, err := ParseProfileSnapshot("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap != nil {
		t.Errorf("expected nil for empty input, got %+v", snap)
	}
}

func TestParseProfileSnapshotInvalidJSONReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ParseProfileSnapshot("{not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ── ResolveEffectiveProfile ──

// mockProfileRepoForSnapshot is a minimal read-only mock for ResolveEffectiveProfile tests.
type mockProfileRepoForSnapshot struct {
	profiles map[uint]*domain.IntegrationProfile
}

func (m *mockProfileRepoForSnapshot) FindByID(id uint) (*domain.IntegrationProfile, error) {
	p, ok := m.profiles[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}
func (m *mockProfileRepoForSnapshot) FindByProfileKey(_ string) (*domain.IntegrationProfile, error) {
	panic("not implemented")
}
func (m *mockProfileRepoForSnapshot) Create(_ *domain.IntegrationProfile) error {
	panic("not implemented")
}
func (m *mockProfileRepoForSnapshot) List() ([]domain.IntegrationProfile, error) {
	panic("not implemented")
}
func (m *mockProfileRepoForSnapshot) Update(_ *domain.IntegrationProfile) error {
	panic("not implemented")
}
func (m *mockProfileRepoForSnapshot) Delete(_ uint) error { panic("not implemented") }

func TestResolveEffectiveProfileUsesBoundSnapshot(t *testing.T) {
	t.Parallel()

	liveProfile := &domain.IntegrationProfile{
		ID:               1,
		ProfileKey:       "live.profile",
		TrackingSyncMode: "manual_confirmation", // different from snapshot
	}
	profileRepo := &mockProfileRepoForSnapshot{
		profiles: map[uint]*domain.IntegrationProfile{1: liveProfile},
	}

	// Build a snapshot with different values than the live profile
	snapData := dto.BoundProfileSnapshot{
		ProfileID:        1,
		ProfileKey:       "bound.profile",
		TrackingSyncMode: "api_push",
	}
	snapJSON, _ := json.Marshal(snapData)

	profileID := uint(1)
	doc := &domain.DemandDocument{
		ID:                   10,
		IntegrationProfileID: &profileID,
		BoundProfileSnapshot: string(snapJSON),
	}

	result, err := ResolveEffectiveProfile(doc, profileRepo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Must use the bound snapshot, not the live profile
	if result.TrackingSyncMode != "api_push" {
		t.Errorf("TrackingSyncMode = %q, want %q (bound snapshot)", result.TrackingSyncMode, "api_push")
	}
	if result.ProfileKey != "bound.profile" {
		t.Errorf("ProfileKey = %q, want %q (bound snapshot)", result.ProfileKey, "bound.profile")
	}
}

func TestResolveEffectiveProfileFallsBackToLive(t *testing.T) {
	t.Parallel()

	liveProfile := &domain.IntegrationProfile{
		ID:               1,
		ProfileKey:       "live.profile",
		TrackingSyncMode: "manual_confirmation",
		ClosurePolicy:    "close_after_sync",
	}
	profileRepo := &mockProfileRepoForSnapshot{
		profiles: map[uint]*domain.IntegrationProfile{1: liveProfile},
	}

	profileID := uint(1)
	doc := &domain.DemandDocument{
		ID:                   10,
		IntegrationProfileID: &profileID,
		BoundProfileSnapshot: "", // no snapshot stored
	}

	result, err := ResolveEffectiveProfile(doc, profileRepo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Must fall back to live profile values
	if result.TrackingSyncMode != "manual_confirmation" {
		t.Errorf("TrackingSyncMode = %q, want %q (live fallback)", result.TrackingSyncMode, "manual_confirmation")
	}
	if result.ProfileKey != "live.profile" {
		t.Errorf("ProfileKey = %q, want %q (live fallback)", result.ProfileKey, "live.profile")
	}
}

func TestResolveEffectiveProfileNoProfileIDReturnsNil(t *testing.T) {
	t.Parallel()

	profileRepo := &mockProfileRepoForSnapshot{profiles: map[uint]*domain.IntegrationProfile{}}
	doc := &domain.DemandDocument{
		ID:                   10,
		IntegrationProfileID: nil,
		BoundProfileSnapshot: "",
	}

	result, err := ResolveEffectiveProfile(doc, profileRepo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for doc with no IntegrationProfileID, got %+v", result)
	}
}

// ── PlanChannelClosure uses bound snapshot ──

func TestPlanChannelClosureUsesBoundSnapshotOverLiveProfile(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	// Live profile says manual_confirmation — but the bound snapshot says api_push.
	// The closure use case must prefer the bound snapshot.
	s.profile.profiles[1].TrackingSyncMode = "manual_confirmation"

	snapData := dto.BoundProfileSnapshot{
		ProfileID:           1,
		ProfileKey:          "test.profile",
		TrackingSyncMode:    "api_push",
		ClosurePolicy:       "close_after_sync",
		AllowsManualClosure: true,
	}
	snapJSON, _ := json.Marshal(snapData)
	s.demand.docs[10].BoundProfileSnapshot = string(snapJSON)

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Bound snapshot says api_push → should create a job, not manual_closure
	if result.Decision != dto.ClosureDecisionCreateJob {
		t.Errorf("decision = %q, want create_job (bound snapshot should override live profile)", result.Decision)
	}
	if result.Job == nil {
		t.Error("expected non-nil Job when bound snapshot says api_push")
	}
}
