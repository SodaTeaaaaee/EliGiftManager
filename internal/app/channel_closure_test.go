package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock integration profile repo ──

type mockProfileRepo struct {
	mu       sync.Mutex
	profiles map[uint]*domain.IntegrationProfile
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{profiles: make(map[uint]*domain.IntegrationProfile)}
}

func (m *mockProfileRepo) FindByID(id uint) (*domain.IntegrationProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[id]
	if !ok {
		return nil, fmt.Errorf("integration profile %d not found", id)
	}
	cp := *p
	return &cp, nil
}

func (m *mockProfileRepo) FindByProfileKey(key string) (*domain.IntegrationProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.profiles {
		if p.ProfileKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("integration profile %q not found", key)
}

func (m *mockProfileRepo) Create(profile *domain.IntegrationProfile) error { panic("not implemented") }
func (m *mockProfileRepo) List() ([]domain.IntegrationProfile, error)       { panic("not implemented") }

// ── mock demand repo for closure ──

type mockDemandRepoForClosure struct {
	mu        sync.Mutex
	docs      map[uint]*domain.DemandDocument
	linesByID map[uint]*domain.DemandLine
}

func newMockDemandRepoForClosure() *mockDemandRepoForClosure {
	return &mockDemandRepoForClosure{
		docs:      make(map[uint]*domain.DemandDocument),
		linesByID: make(map[uint]*domain.DemandLine),
	}
}

func (m *mockDemandRepoForClosure) FindByID(id uint) (*domain.DemandDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.docs[id]
	if !ok {
		return nil, fmt.Errorf("demand document %d not found", id)
	}
	cp := *d
	return &cp, nil
}

func (m *mockDemandRepoForClosure) FindLineByID(id uint) (*domain.DemandLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.linesByID[id]
	if !ok {
		return nil, fmt.Errorf("demand line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

func (m *mockDemandRepoForClosure) Create(doc *domain.DemandDocument) error               { panic("not implemented") }
func (m *mockDemandRepoForClosure) List() ([]domain.DemandDocument, error)                 { panic("not implemented") }
func (m *mockDemandRepoForClosure) CreateLine(line *domain.DemandLine) error               { panic("not implemented") }
func (m *mockDemandRepoForClosure) ListLinesByDocument(docID uint) ([]domain.DemandLine, error) {
	panic("not implemented")
}

// ── helper setup ──

type closureTestSetup struct {
	profile     *mockProfileRepo
	shipment    *mockShipmentRepoForSync
	fulfill     *mockFulfillRepoForSync
	demand      *mockDemandRepoForClosure
	channelSync *mockChannelSyncRepo
	supplier    *mockSupplierRepoForSync
	uc          ChannelClosureUseCase
}

func newClosureTestSetup() *closureTestSetup {
	pr := newMockProfileRepo()
	sh := newMockShipmentRepoForSync()
	fl := newMockFulfillRepoForSync()
	dm := newMockDemandRepoForClosure()
	cs := newMockChannelSyncRepo()
	su := newMockSupplierRepoForSync()

	// Default profile: api_push, allows_manual_closure
	pr.profiles[1] = &domain.IntegrationProfile{
		ID:                      1,
		ProfileKey:              "test.profile",
		TrackingSyncMode:        "api_push",
		ClosurePolicy:           "close_after_sync",
		RequiresExternalOrderNo: false,
		AllowsManualClosure:     true,
	}

	// Supplier order in wave 1 (required by low-level CreateChannelSyncJob validation)
	su.orders[1] = &domain.SupplierOrder{ID: 1, WaveID: 1}
	sh.setSupplierOrderWave(1, 1)

	// Default shipment + fulfillment + demand wiring
	sh.add(&domain.Shipment{ID: 1, SupplierOrderID: 1, TrackingNo: "TRACK-001", CarrierCode: "SF"})
	sh.addLine(domain.ShipmentLine{ID: 1, ShipmentID: 1, FulfillmentLineID: 1})
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10), DemandLineID: uintPtr(100)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, SourceDocumentNo: "EXT-ORDER-1", IntegrationProfileID: uintPtr(1)}
	dm.linesByID[100] = &domain.DemandLine{ID: 100, SourceLineNo: 3}

	lowLevelUC := NewChannelSyncUseCase(cs, sh, su, fl)

	return &closureTestSetup{
		profile:     pr,
		shipment:    sh,
		fulfill:     fl,
		demand:      dm,
		channelSync: cs,
		supplier:    su,
		uc:          NewChannelClosureUseCase(pr, sh, fl, dm, lowLevelUC),
	}
}

func uintPtr(v uint) *uint { return &v }

// ── create_job branch tests ──

func TestPlanChannelClosureAPIPushCreatesJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != dto.ClosureDecisionCreateJob {
		t.Errorf("decision = %q, want create_job", result.Decision)
	}
	if result.Job == nil {
		t.Fatal("expected non-nil Job")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].ExternalDocumentNo != "EXT-ORDER-1" {
		t.Errorf("ExternalDocumentNo = %q, want %q", result.Items[0].ExternalDocumentNo, "EXT-ORDER-1")
	}
	if result.Items[0].ID == 0 {
		t.Error("expected persisted item ID > 0 for create_job")
	}
}

func TestPlanChannelClosureDocumentExportCreatesJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "document_export"

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != dto.ClosureDecisionCreateJob {
		t.Errorf("decision = %q, want create_job", result.Decision)
	}
	if result.Job == nil {
		t.Fatal("expected non-nil Job for document_export")
	}
}

// ── manual_confirmation branch tests ──

func TestPlanChannelClosureManualConfirmationReturnsItemsWithoutJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "manual_confirmation"

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != dto.ClosureDecisionManualClosure {
		t.Errorf("decision = %q, want manual_closure", result.Decision)
	}
	if result.Job != nil {
		t.Error("expected nil Job for manual_confirmation")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 planned item, got %d", len(result.Items))
	}
	if result.Items[0].FulfillmentLineID != 1 {
		t.Errorf("item FulfillmentLineID = %d, want 1", result.Items[0].FulfillmentLineID)
	}
	if result.Items[0].ShipmentID != 1 {
		t.Errorf("item ShipmentID = %d, want 1", result.Items[0].ShipmentID)
	}
	if result.Items[0].ExternalDocumentNo != "EXT-ORDER-1" {
		t.Errorf("item ExternalDocumentNo = %q, want %q", result.Items[0].ExternalDocumentNo, "EXT-ORDER-1")
	}
	// Planned candidates have zero IDs (not persisted)
	if result.Items[0].ID != 0 {
		t.Errorf("item ID = %d, want 0 (planned, not persisted)", result.Items[0].ID)
	}
	// Verify no job was created
	if len(s.channelSync.jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(s.channelSync.jobs))
	}
}

func TestPlanChannelClosureRejectsManualConfirmationWithoutAllowsManualClosure(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "manual_confirmation"
	s.profile.profiles[1].AllowsManualClosure = false

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error for manual_confirmation with allows_manual_closure=false, got nil")
	}
}

// ── unsupported branch tests ──

func TestPlanChannelClosureUnsupportedReturnsItemsWithoutJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "unsupported"

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision != dto.ClosureDecisionUnsupported {
		t.Errorf("decision = %q, want unsupported", result.Decision)
	}
	if result.Job != nil {
		t.Error("expected nil Job for unsupported")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 planned item, got %d", len(result.Items))
	}
	if result.Items[0].FulfillmentLineID != 1 {
		t.Errorf("item FulfillmentLineID = %d, want 1", result.Items[0].FulfillmentLineID)
	}
}

// ── candidate existence gates (applies to ALL branches) ──

func TestPlanChannelClosureRejectsNoCandidatesForCreateJob(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.shipment.shipments = make(map[uint]*domain.Shipment)

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error for no candidates, got nil")
	}
}

func TestPlanChannelClosureManualConfirmationRejectsNoCandidates(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "manual_confirmation"
	s.shipment.shipments = make(map[uint]*domain.Shipment)

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error for manual_confirmation with no candidates, got nil")
	}
}

func TestPlanChannelClosureUnsupportedRejectsNoCandidates(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].TrackingSyncMode = "unsupported"
	s.shipment.shipments = make(map[uint]*domain.Shipment)

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error for unsupported with no candidates, got nil")
	}
}

// ── other rejection tests ──

func TestPlanChannelClosureRejectsMissingProfile(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 999})
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestPlanChannelClosureRejectsMissingExternalOrderNo(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()
	s.profile.profiles[1].RequiresExternalOrderNo = true
	s.demand.docs[10].SourceDocumentNo = ""

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error for missing external_order_no, got nil")
	}
}

// ── mixed wave tests ──

func TestPlanChannelClosureOnlyIncludesCandidatesFromRequestedProfile(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	s.profile.profiles[2] = &domain.IntegrationProfile{
		ID:                  2,
		ProfileKey:          "other.profile",
		TrackingSyncMode:    "api_push",
		ClosurePolicy:       "close_after_sync",
		AllowsManualClosure: true,
	}
	s.demand.docs[20] = &domain.DemandDocument{
		ID:                   20,
		SourceDocumentNo:     "EXT-ORDER-2",
		IntegrationProfileID: uintPtr(2),
	}
	s.fulfill.lines[2] = &domain.FulfillmentLine{
		ID:               2,
		WaveID:           1,
		DemandDocumentID: uintPtr(20),
	}
	s.shipment.addLine(domain.ShipmentLine{
		ID:                 2,
		ShipmentID:         1,
		FulfillmentLineID:  2,
	})

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (profile A only), got %d", len(result.Items))
	}
	if result.Items[0].ExternalDocumentNo != "EXT-ORDER-1" {
		t.Errorf("got ExternalDocumentNo = %q, want %q", result.Items[0].ExternalDocumentNo, "EXT-ORDER-1")
	}
}

func TestPlanChannelClosureSkipsCandidatesWithoutDemandDocument(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	s.fulfill.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 1}
	s.shipment.addLine(domain.ShipmentLine{ID: 2, ShipmentID: 1, FulfillmentLineID: 2})

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestPlanChannelClosureSkipsCandidatesWithoutIntegrationProfileID(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	s.demand.docs[20] = &domain.DemandDocument{ID: 20, SourceDocumentNo: "NO-PROFILE"}
	s.fulfill.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 1, DemandDocumentID: uintPtr(20)}
	s.shipment.addLine(domain.ShipmentLine{ID: 2, ShipmentID: 1, FulfillmentLineID: 2})

	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestPlanChannelClosureReturnsNoCandidatesAfterProfileFiltering(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	s.demand.docs[10].IntegrationProfileID = uintPtr(2)

	_, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err == nil {
		t.Fatal("expected error after profile filtering removes all candidates, got nil")
	}
}

// ── ListByWave filtering tests ──

func TestPlanChannelClosureListByWaveFilteringMatters(t *testing.T) {
	t.Parallel()
	s := newClosureTestSetup()

	// Create a shipment in wave 2 with the same profile's demand doc
	s.supplier.orders[2] = &domain.SupplierOrder{ID: 2, WaveID: 2}
	s.shipment.setSupplierOrderWave(2, 2)
	s.shipment.add(&domain.Shipment{ID: 2, SupplierOrderID: 2, TrackingNo: "TRACK-W2", CarrierCode: "SF"})
	s.shipment.addLine(domain.ShipmentLine{ID: 3, ShipmentID: 2, FulfillmentLineID: 3})
	// Fulfillment line 3 is in wave 2, but we're querying wave 1
	s.fulfill.lines[3] = &domain.FulfillmentLine{ID: 3, WaveID: 2, DemandDocumentID: uintPtr(10)}

	// PlanChannelClosure for wave 1 should only see the wave-1 shipment
	result, err := s.uc.PlanChannelClosure(dto.PlanChannelClosureInput{WaveID: 1, IntegrationProfileID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item from wave 1, got %d", len(result.Items))
	}
	// The item must be from shipment 1 (wave 1), NOT shipment 2 (wave 2)
	if result.Items[0].ShipmentID != 1 {
		t.Errorf("ShipmentID = %d, want 1 (should not include shipment from wave 2)", result.Items[0].ShipmentID)
	}
	if result.Items[0].TrackingNo != "TRACK-001" {
		t.Errorf("TrackingNo = %q, want %q", result.Items[0].TrackingNo, "TRACK-001")
	}
}
