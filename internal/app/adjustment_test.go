package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock FulfillmentAdjustmentRepository ──

type mockAdjustmentRepo struct {
	mu      sync.Mutex
	records []domain.FulfillmentAdjustment
	lastID  uint
	failOn  string // "create" to simulate Create error
}

func newMockAdjustmentRepo() *mockAdjustmentRepo {
	return &mockAdjustmentRepo{}
}

func (m *mockAdjustmentRepo) Create(adj *domain.FulfillmentAdjustment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn == "create" {
		return fmt.Errorf("mock: create failed")
	}
	m.lastID++
	adj.ID = m.lastID
	adj.CreatedAt = "2024-01-01T00:00:00Z"
	adj.UpdatedAt = "2024-01-01T00:00:00Z"
	cp := *adj
	m.records = append(m.records, cp)
	return nil
}

func (m *mockAdjustmentRepo) ListByWave(waveID uint) ([]domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FulfillmentAdjustment
	for _, r := range m.records {
		if r.WaveID == waveID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockAdjustmentRepo) ListByFulfillmentLine(fulfillmentLineID uint) ([]domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FulfillmentAdjustment
	for _, r := range m.records {
		if r.FulfillmentLineID != nil && *r.FulfillmentLineID == fulfillmentLineID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockAdjustmentRepo) FindByID(id uint) (*domain.FulfillmentAdjustment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *mockAdjustmentRepo) Delete(id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.records {
		if r.ID == id {
			m.records = append(m.records[:i], m.records[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockAdjustmentRepo) DeleteByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []domain.FulfillmentAdjustment
	for _, r := range m.records {
		if r.WaveID != waveID {
			kept = append(kept, r)
		}
	}
	m.records = kept
	return nil
}

// ── mock FulfillmentLineRepository (adjustment tests) ──

type mockFulfillRepoForAdjustment struct {
	mu    sync.Mutex
	lines map[uint]*domain.FulfillmentLine
}

func newMockFulfillRepoForAdjustment() *mockFulfillRepoForAdjustment {
	return &mockFulfillRepoForAdjustment{lines: make(map[uint]*domain.FulfillmentLine)}
}

func (m *mockFulfillRepoForAdjustment) Create(line *domain.FulfillmentLine) error {
	panic("not implemented")
}

func (m *mockFulfillRepoForAdjustment) FindByID(id uint) (*domain.FulfillmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.lines[id]
	if !ok {
		return nil, fmt.Errorf("fulfillment line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

func (m *mockFulfillRepoForAdjustment) ListByWave(waveID uint) ([]domain.FulfillmentLine, error) {
	panic("not implemented")
}

func (m *mockFulfillRepoForAdjustment) DeleteByWaveAndGeneratedBy(waveID uint, generatedBy string) error {
	panic("not implemented")
}
func (m *mockFulfillRepoForAdjustment) ReplaceByWaveAndGeneratedBy(_ uint, _ string, _ []domain.FulfillmentLine) error {
	panic("not implemented")
}

func (m *mockFulfillRepoForAdjustment) DeleteByWave(waveID uint) error {
	panic("not implemented")
}

// ── mock WaveRepository (adjustment tests) ──

type mockWaveRepoForAdjustment struct {
	mu           sync.Mutex
	participants map[uint][]domain.WaveParticipantSnapshot // waveID -> snapshots
}

func newMockWaveRepoForAdjustment() *mockWaveRepoForAdjustment {
	return &mockWaveRepoForAdjustment{
		participants: make(map[uint][]domain.WaveParticipantSnapshot),
	}
}

func (m *mockWaveRepoForAdjustment) Create(wave *domain.Wave) error {
	panic("not implemented")
}

func (m *mockWaveRepoForAdjustment) FindByID(id uint) (*domain.Wave, error) {
	panic("not implemented")
}

func (m *mockWaveRepoForAdjustment) FindByWaveNo(waveNo string) (*domain.Wave, error) {
	panic("not implemented")
}

func (m *mockWaveRepoForAdjustment) List() ([]domain.Wave, error) {
	panic("not implemented")
}

func (m *mockWaveRepoForAdjustment) AddParticipant(snap *domain.WaveParticipantSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.participants[snap.WaveID] = append(m.participants[snap.WaveID], *snap)
	return nil
}

func (m *mockWaveRepoForAdjustment) ListParticipantsByWave(waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.participants[waveID], nil
}

func (m *mockWaveRepoForAdjustment) DeleteParticipantsByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.participants, waveID)
	return nil
}

// ── helpers ──

type adjustmentTestSetup struct {
	adjRepo     *mockAdjustmentRepo
	fulfillRepo *mockFulfillRepoForAdjustment
	waveRepo    *mockWaveRepoForAdjustment
	uc          AdjustmentUseCase
}

func newAdjustmentTestSetup() *adjustmentTestSetup {
	ar := newMockAdjustmentRepo()
	fr := newMockFulfillRepoForAdjustment()
	wr := newMockWaveRepoForAdjustment()
	fr.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 10}
	wr.participants[10] = []domain.WaveParticipantSnapshot{
		{ID: 100, WaveID: 10, CustomerProfileID: 1, DisplayName: "Test Participant"},
	}
	return &adjustmentTestSetup{
		adjRepo:     ar,
		fulfillRepo: fr,
		waveRepo:    wr,
		uc:          NewAdjustmentUseCase(ar, fr, wr),
	}
}

func validAdjustmentInput() dto.RecordAdjustmentInput {
	return dto.RecordAdjustmentInput{
		WaveID:            10,
		TargetKind:        "fulfillment_line",
		FulfillmentLineID: uintPtr(1),
		AdjustmentKind:    "add",
		QuantityDelta:     2,
		ReasonCode:        "restock",
		OperatorID:        "op-1",
		Note:              "extra unit",
		EvidenceRef:       "ref-001",
	}
}

// ── tests ──

func TestRecordAdjustmentSuccess(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	adj, err := s.uc.RecordAdjustment(validAdjustmentInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adj.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if adj.WaveID != 10 {
		t.Errorf("WaveID = %d, want 10", adj.WaveID)
	}
	if adj.FulfillmentLineID == nil || *adj.FulfillmentLineID != 1 {
		t.Errorf("FulfillmentLineID = %v, want 1", adj.FulfillmentLineID)
	}
	if adj.AdjustmentKind != "add" {
		t.Errorf("AdjustmentKind = %q, want add", adj.AdjustmentKind)
	}
	if adj.QuantityDelta != 2 {
		t.Errorf("QuantityDelta = %d, want 2", adj.QuantityDelta)
	}
	if adj.TargetKind != "fulfillment_line" {
		t.Errorf("TargetKind = %q, want fulfillment_line", adj.TargetKind)
	}
	if len(s.adjRepo.records) != 1 {
		t.Errorf("expected 1 persisted record, got %d", len(s.adjRepo.records))
	}
}

func TestRecordAdjustmentDefaultTargetKind(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.TargetKind = "" // should default to "fulfillment_line"

	adj, err := s.uc.RecordAdjustment(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adj.TargetKind != "fulfillment_line" {
		t.Errorf("TargetKind = %q, want fulfillment_line", adj.TargetKind)
	}
}

func TestRecordAdjustmentFulfillmentLineNotFound(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.FulfillmentLineID = uintPtr(999)

	_, err := s.uc.RecordAdjustment(input)
	if err == nil {
		t.Fatal("expected error for non-existent fulfillment line, got nil")
	}
}

func TestRecordAdjustmentWaveMismatch(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.WaveID = 99 // line 1 belongs to wave 10, not 99

	_, err := s.uc.RecordAdjustment(input)
	if err == nil {
		t.Fatal("expected error for wave mismatch, got nil")
	}
}

func TestRecordAdjustmentInvalidKind(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.AdjustmentKind = "invalid_kind"

	_, err := s.uc.RecordAdjustment(input)
	if err == nil {
		t.Fatal("expected error for invalid adjustment kind, got nil")
	}
}

func TestListAdjustmentsByWave(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	// Add a second fulfillment line in a different wave
	s.fulfillRepo.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 20}

	// Record two adjustments in wave 10
	for i := 0; i < 2; i++ {
		if _, err := s.uc.RecordAdjustment(validAdjustmentInput()); err != nil {
			t.Fatalf("setup: unexpected error: %v", err)
		}
	}
	// Record one adjustment in wave 20
	other := validAdjustmentInput()
	other.WaveID = 20
	other.FulfillmentLineID = uintPtr(2)
	if _, err := s.uc.RecordAdjustment(other); err != nil {
		t.Fatalf("setup: unexpected error: %v", err)
	}

	results, err := s.uc.ListAdjustmentsByWave(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 adjustments for wave 10, got %d", len(results))
	}
	for _, r := range results {
		if r.WaveID != 10 {
			t.Errorf("got WaveID = %d, want 10", r.WaveID)
		}
	}

	// Wave 20 should have exactly 1
	results20, err := s.uc.ListAdjustmentsByWave(20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results20) != 1 {
		t.Errorf("expected 1 adjustment for wave 20, got %d", len(results20))
	}
}

func TestRecordSupplementAdjustmentWithParticipantTarget(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := dto.RecordAdjustmentInput{
		WaveID:                    10,
		TargetKind:                "participant",
		WaveParticipantSnapshotID: uintPtr(100),
		AdjustmentKind:            "compensation",
		QuantityDelta:             1,
		ReasonCode:                "bonus",
		OperatorID:                "op-1",
		Note:                      "compensation for participant",
		EvidenceRef:               "ref-002",
	}

	adj, err := s.uc.RecordAdjustment(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adj.TargetKind != "participant" {
		t.Errorf("TargetKind = %q, want participant", adj.TargetKind)
	}
	if adj.WaveParticipantSnapshotID == nil || *adj.WaveParticipantSnapshotID != 100 {
		t.Errorf("WaveParticipantSnapshotID = %v, want 100", adj.WaveParticipantSnapshotID)
	}
	if adj.AdjustmentKind != "compensation" {
		t.Errorf("AdjustmentKind = %q, want compensation", adj.AdjustmentKind)
	}
}

func TestRecordSupplementAdjustmentRejectsFulfillmentLineTarget(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.AdjustmentKind = "compensation"
	input.TargetKind = "fulfillment_line"

	_, err := s.uc.RecordAdjustment(input)
	if err == nil {
		t.Fatal("expected error: compensation should require participant target, got nil")
	}
}

func TestRecordAddSendRejectsParticipantTarget(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := dto.RecordAdjustmentInput{
		WaveID:                    10,
		TargetKind:                "participant",
		WaveParticipantSnapshotID: uintPtr(100),
		AdjustmentKind:            "add",
		QuantityDelta:             1,
		ReasonCode:                "test",
		OperatorID:                "op-1",
	}

	_, err := s.uc.RecordAdjustment(input)
	if err == nil {
		t.Fatal("expected error: add should require fulfillment_line target, got nil")
	}
}
