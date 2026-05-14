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
		if r.FulfillmentLineID == fulfillmentLineID {
			out = append(out, r)
		}
	}
	return out, nil
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

// ── helpers ──

type adjustmentTestSetup struct {
	adjRepo     *mockAdjustmentRepo
	fulfillRepo *mockFulfillRepoForAdjustment
	uc          AdjustmentUseCase
}

func newAdjustmentTestSetup() *adjustmentTestSetup {
	ar := newMockAdjustmentRepo()
	fr := newMockFulfillRepoForAdjustment()
	fr.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 10}
	return &adjustmentTestSetup{
		adjRepo:     ar,
		fulfillRepo: fr,
		uc:          NewAdjustmentUseCase(ar, fr),
	}
}

func validAdjustmentInput() dto.RecordAdjustmentInput {
	return dto.RecordAdjustmentInput{
		WaveID:            10,
		FulfillmentLineID: 1,
		AdjustmentKind:    "add_send",
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
	if adj.FulfillmentLineID != 1 {
		t.Errorf("FulfillmentLineID = %d, want 1", adj.FulfillmentLineID)
	}
	if adj.AdjustmentKind != "add_send" {
		t.Errorf("AdjustmentKind = %q, want add_send", adj.AdjustmentKind)
	}
	if adj.QuantityDelta != 2 {
		t.Errorf("QuantityDelta = %d, want 2", adj.QuantityDelta)
	}
	if len(s.adjRepo.records) != 1 {
		t.Errorf("expected 1 persisted record, got %d", len(s.adjRepo.records))
	}
}

func TestRecordAdjustmentFulfillmentLineNotFound(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()

	input := validAdjustmentInput()
	input.FulfillmentLineID = 999

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
	other.FulfillmentLineID = 2
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
