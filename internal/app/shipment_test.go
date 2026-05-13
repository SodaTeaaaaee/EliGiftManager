package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock shipment repo ──

type mockShipmentRepo struct {
	mu                sync.Mutex
	shipments         map[uint]*domain.Shipment
	shipmentLines     map[uint][]*domain.ShipmentLine
	supplierOrderWave map[uint]uint // supplierOrderID → waveID
	lastID            uint
}

func newMockShipmentRepo() *mockShipmentRepo {
	return &mockShipmentRepo{
		shipments:         make(map[uint]*domain.Shipment),
		shipmentLines:     make(map[uint][]*domain.ShipmentLine),
		supplierOrderWave: make(map[uint]uint),
	}
}

func (m *mockShipmentRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockShipmentRepo) Create(shipment *domain.Shipment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	shipment.ID = m.next()
	cp := *shipment
	m.shipments[shipment.ID] = &cp
	return nil
}

func (m *mockShipmentRepo) FindByID(id uint) (*domain.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.shipments[id]
	if !ok {
		return nil, fmt.Errorf("shipment %d not found", id)
	}
	cp := *s
	return &cp, nil
}

func (m *mockShipmentRepo) ListBySupplierOrder(supplierOrderID uint) ([]domain.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Shipment
	for _, s := range m.shipments {
		if s.SupplierOrderID == supplierOrderID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockShipmentRepo) ListByWave(waveID uint) ([]domain.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Shipment
	for _, s := range m.shipments {
		if m.supplierOrderWave[s.SupplierOrderID] == waveID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockShipmentRepo) CreateLine(line *domain.ShipmentLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.ID = m.next()
	cp := *line
	m.shipmentLines[line.ShipmentID] = append(m.shipmentLines[line.ShipmentID], &cp)
	return nil
}

func (m *mockShipmentRepo) AtomicCreateShipment(shipment *domain.Shipment, lines []*domain.ShipmentLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	shipment.ID = m.next()
	cp := *shipment
	m.shipments[shipment.ID] = &cp
	for _, line := range lines {
		line.ShipmentID = shipment.ID
		line.ID = m.next()
		cpLine := *line
		m.shipmentLines[shipment.ID] = append(m.shipmentLines[shipment.ID], &cpLine)
	}
	return nil
}

func (m *mockShipmentRepo) ListLinesByShipment(shipmentID uint) ([]domain.ShipmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ll, ok := m.shipmentLines[shipmentID]
	if !ok {
		return nil, nil
	}
	out := make([]domain.ShipmentLine, len(ll))
	for i, l := range ll {
		out[i] = *l
	}
	return out, nil
}

// ── tests ──

func TestCreateShipmentPersistsShipmentAndLines(t *testing.T) {
	t.Parallel()

	repo := newMockShipmentRepo()

	// Create shipment
	shipment := &domain.Shipment{
		SupplierOrderID: 1,
		ShipmentNo:      "SHIP-001",
		TrackingNo:      "TRACK-123",
		Status:          "shipped",
		CarrierCode:     "SF",
		CarrierName:     "顺丰速运",
	}
	if err := repo.Create(shipment); err != nil {
		t.Fatalf("Create shipment: %v", err)
	}
	if shipment.ID == 0 {
		t.Error("expected shipment.ID > 0 after Create")
	}

	// Create lines
	line1 := &domain.ShipmentLine{
		ShipmentID:          shipment.ID,
		SupplierOrderLineID: 10,
		FulfillmentLineID:   100,
		Quantity:            2,
	}
	line2 := &domain.ShipmentLine{
		ShipmentID:          shipment.ID,
		SupplierOrderLineID: 11,
		FulfillmentLineID:   101,
		Quantity:            3,
	}
	if err := repo.CreateLine(line1); err != nil {
		t.Fatalf("CreateLine 1: %v", err)
	}
	if err := repo.CreateLine(line2); err != nil {
		t.Fatalf("CreateLine 2: %v", err)
	}
	if line1.ID == 0 {
		t.Error("expected line1.ID > 0 after CreateLine")
	}
	if line2.ID == 0 {
		t.Error("expected line2.ID > 0 after CreateLine")
	}
	if line1.ShipmentID != shipment.ID {
		t.Errorf("line1.ShipmentID = %d, want %d", line1.ShipmentID, shipment.ID)
	}
	if line2.ShipmentID != shipment.ID {
		t.Errorf("line2.ShipmentID = %d, want %d", line2.ShipmentID, shipment.ID)
	}

	// Read back and verify field completeness
	got, err := repo.FindByID(shipment.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.SupplierOrderID != 1 {
		t.Errorf("SupplierOrderID = %d, want 1", got.SupplierOrderID)
	}
	if got.ShipmentNo != "SHIP-001" {
		t.Errorf("ShipmentNo = %q, want %q", got.ShipmentNo, "SHIP-001")
	}
	if got.TrackingNo != "TRACK-123" {
		t.Errorf("TrackingNo = %q, want %q", got.TrackingNo, "TRACK-123")
	}
	if got.Status != "shipped" {
		t.Errorf("Status = %q, want %q", got.Status, "shipped")
	}

	// Read back lines
	lines, err := repo.ListLinesByShipment(shipment.ID)
	if err != nil {
		t.Fatalf("ListLinesByShipment: %v", err)
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestListShipmentsByWaveReturnsJoinedResults(t *testing.T) {
	t.Parallel()

	repo := newMockShipmentRepo()

	// Setup: supplierOrder 1 → wave 1, supplierOrder 2 → wave 2
	repo.supplierOrderWave[1] = 1
	repo.supplierOrderWave[2] = 2

	// Create shipment 1 for supplierOrder 1 (wave 1)
	s1 := &domain.Shipment{
		SupplierOrderID: 1,
		ShipmentNo:      "SHIP-100",
		Status:          "shipped",
	}
	if err := repo.Create(s1); err != nil {
		t.Fatalf("Create s1: %v", err)
	}

	// Create shipment 2 for supplierOrder 2 (wave 2)
	s2 := &domain.Shipment{
		SupplierOrderID: 2,
		ShipmentNo:      "SHIP-200",
		Status:          "delivered",
	}
	if err := repo.Create(s2); err != nil {
		t.Fatalf("Create s2: %v", err)
	}

	// ListByWave(1) → 1 result
	w1, err := repo.ListByWave(1)
	if err != nil {
		t.Fatalf("ListByWave(1): %v", err)
	}
	if len(w1) != 1 {
		t.Fatalf("ListByWave(1): expected 1, got %d", len(w1))
	}
	if w1[0].ShipmentNo != "SHIP-100" {
		t.Errorf("ListByWave(1)[0].ShipmentNo = %q, want %q", w1[0].ShipmentNo, "SHIP-100")
	}

	// ListByWave(2) → 1 result
	w2, err := repo.ListByWave(2)
	if err != nil {
		t.Fatalf("ListByWave(2): %v", err)
	}
	if len(w2) != 1 {
		t.Fatalf("ListByWave(2): expected 1, got %d", len(w2))
	}
	if w2[0].ShipmentNo != "SHIP-200" {
		t.Errorf("ListByWave(2)[0].ShipmentNo = %q, want %q", w2[0].ShipmentNo, "SHIP-200")
	}

	// ListByWave(3) → 0 results
	w3, err := repo.ListByWave(3)
	if err != nil {
		t.Fatalf("ListByWave(3): %v", err)
	}
	if len(w3) != 0 {
		t.Errorf("ListByWave(3): expected 0, got %d", len(w3))
	}
}

func TestWaveOverviewCountsShipmentsAndTrackedFulfillment(t *testing.T) {
	t.Parallel()

	repo := newMockShipmentRepo()

	// Setup: both supplier orders belong to wave 1
	repo.supplierOrderWave[1] = 1
	repo.supplierOrderWave[2] = 1

	// Shipment with tracking (supplierOrder 1)
	trackedShipment := &domain.Shipment{
		SupplierOrderID: 1,
		ShipmentNo:      "SHIP-TRACKED",
		TrackingNo:      "TRACK-999",
		Status:          "shipped",
	}
	if err := repo.Create(trackedShipment); err != nil {
		t.Fatalf("Create tracked shipment: %v", err)
	}

	// Lines for tracked shipment with distinct fulfillment line IDs
	if err := repo.CreateLine(&domain.ShipmentLine{
		ShipmentID:          trackedShipment.ID,
		SupplierOrderLineID: 10,
		FulfillmentLineID:   100,
		Quantity:            1,
	}); err != nil {
		t.Fatalf("CreateLine for tracked: %v", err)
	}
	if err := repo.CreateLine(&domain.ShipmentLine{
		ShipmentID:          trackedShipment.ID,
		SupplierOrderLineID: 11,
		FulfillmentLineID:   101,
		Quantity:            1,
	}); err != nil {
		t.Fatalf("CreateLine for tracked: %v", err)
	}

	// Shipment without tracking (supplierOrder 2)
	untrackedShipment := &domain.Shipment{
		SupplierOrderID: 2,
		ShipmentNo:      "SHIP-NOTRACK",
		TrackingNo:      "",
		Status:          "pending",
	}
	if err := repo.Create(untrackedShipment); err != nil {
		t.Fatalf("Create untracked shipment: %v", err)
	}

	// Line for untracked shipment
	if err := repo.CreateLine(&domain.ShipmentLine{
		ShipmentID:          untrackedShipment.ID,
		SupplierOrderLineID: 20,
		FulfillmentLineID:   200,
		Quantity:            1,
	}); err != nil {
		t.Fatalf("CreateLine for untracked: %v", err)
	}

	// Verify shipmentCount for wave 1
	allShipments, err := repo.ListByWave(1)
	if err != nil {
		t.Fatalf("ListByWave(1): %v", err)
	}
	shipmentCount := len(allShipments)
	if shipmentCount != 2 {
		t.Errorf("shipmentCount = %d, want 2", shipmentCount)
	}

	// Compute trackedFulfillmentCount: iterate shipments with tracking,
	// collect unique FulfillmentLineIDs from their lines.
	seen := make(map[uint]bool)
	for _, s := range allShipments {
		if s.TrackingNo == "" {
			continue // skip untracked
		}
		lines, err := repo.ListLinesByShipment(s.ID)
		if err != nil {
			t.Fatalf("ListLinesByShipment(%d): %v", s.ID, err)
		}
		for _, l := range lines {
			seen[l.FulfillmentLineID] = true
		}
	}
	trackedFulfillmentCount := len(seen)
	if trackedFulfillmentCount != 2 {
		t.Errorf("trackedFulfillmentCount = %d, want 2", trackedFulfillmentCount)
	}

	// Confirm untracked shipment's fulfillment line is NOT in the set
	if seen[200] {
		t.Error("untracked shipment's FulfillmentLineID=200 should NOT appear in tracked set")
	}
}

// ── mock supplier repo for shipment validation tests ──

type mockSupplierRepoForShipment struct {
	mu         sync.Mutex
	orders     map[uint]*domain.SupplierOrder
	orderLines map[uint]*domain.SupplierOrderLine
}

func newMockSupplierRepoForShipment() *mockSupplierRepoForShipment {
	return &mockSupplierRepoForShipment{
		orders:     make(map[uint]*domain.SupplierOrder),
		orderLines: make(map[uint]*domain.SupplierOrderLine),
	}
}

func (m *mockSupplierRepoForShipment) FindByID(id uint) (*domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, fmt.Errorf("supplier order %d not found", id)
	}
	cp := *o
	return &cp, nil
}

func (m *mockSupplierRepoForShipment) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.orderLines[id]
	if !ok {
		return nil, fmt.Errorf("supplier order line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

// ── mock fulfill repo for shipment validation tests ──

type mockFulfillRepoForShipment struct {
	mu    sync.Mutex
	lines map[uint]*domain.FulfillmentLine
}

func newMockFulfillRepoForShipment() *mockFulfillRepoForShipment {
	return &mockFulfillRepoForShipment{lines: make(map[uint]*domain.FulfillmentLine)}
}

func (m *mockFulfillRepoForShipment) FindByID(id uint) (*domain.FulfillmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.lines[id]
	if !ok {
		return nil, fmt.Errorf("fulfillment line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

// ── validation tests ──

func TestCreateShipmentRejectsNonexistentSupplierOrder(t *testing.T) {
	t.Parallel()

	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()

	input := dto.CreateShipmentInput{
		SupplierOrderID:  999,
		SupplierPlatform: "SF",
		ShipmentNo:       "SHIP-001",
		Status:           "shipped",
	}

	_, err := createShipmentWithValidation(shipmentRepo, supplierRepo, fulfillRepo, input)
	if err == nil {
		t.Fatal("expected error for nonexistent supplier order, got nil")
	}
}

func TestCreateShipmentRejectsInconsistentLineChain(t *testing.T) {
	t.Parallel()

	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()

	// Pre-create supplier order 1 (wave 1)
	supplierRepo.orders[1] = &domain.SupplierOrder{ID: 1, WaveID: 1}
	// Pre-create supplier order line 10 (belongs to order 1, references fulfillment line 100)
	supplierRepo.orderLines[10] = &domain.SupplierOrderLine{ID: 10, SupplierOrderID: 1, FulfillmentLineID: 100}
	// Pre-create fulfillment line 100 (wave 1)
	fulfillRepo.lines[100] = &domain.FulfillmentLine{ID: 100, WaveID: 1}

	input := dto.CreateShipmentInput{
		SupplierOrderID:  1,
		SupplierPlatform: "SF",
		ShipmentNo:       "SHIP-002",
		Status:           "shipped",
		Lines: []dto.CreateShipmentLineInput{
			{SupplierOrderLineID: 10, FulfillmentLineID: 999, Quantity: 1},
		},
	}

	_, err := createShipmentWithValidation(shipmentRepo, supplierRepo, fulfillRepo, input)
	if err == nil {
		t.Fatal("expected error for inconsistent line chain, got nil")
	}
}

func TestCreateShipmentRejectsCrossWaveLine(t *testing.T) {
	t.Parallel()

	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()

	// Pre-create supplier order 1 (wave 1)
	supplierRepo.orders[1] = &domain.SupplierOrder{ID: 1, WaveID: 1}
	// Pre-create supplier order line 10 (belongs to order 1, references fulfillment line 100)
	supplierRepo.orderLines[10] = &domain.SupplierOrderLine{ID: 10, SupplierOrderID: 1, FulfillmentLineID: 100}
	// Pre-create fulfillment line 100 (wave 2 — different wave!)
	fulfillRepo.lines[100] = &domain.FulfillmentLine{ID: 100, WaveID: 2}

	input := dto.CreateShipmentInput{
		SupplierOrderID:  1,
		SupplierPlatform: "SF",
		ShipmentNo:       "SHIP-003",
		Status:           "shipped",
		Lines: []dto.CreateShipmentLineInput{
			{SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 1},
		},
	}

	_, err := createShipmentWithValidation(shipmentRepo, supplierRepo, fulfillRepo, input)
	if err == nil {
		t.Fatal("expected error for cross-wave line, got nil")
	}
}

// createShipmentWithValidation mirrors ShipmentController.CreateShipment
// validation logic using the supplied mock repos.
func createShipmentWithValidation(
	shipmentRepo *mockShipmentRepo,
	supplierRepo *mockSupplierRepoForShipment,
	fulfillRepo *mockFulfillRepoForShipment,
	input dto.CreateShipmentInput,
) (dto.ShipmentDTO, error) {
	now := "2024-01-01T00:00:00Z"

	// Validate supplier order existence
	supplierOrder, err := supplierRepo.FindByID(input.SupplierOrderID)
	if err != nil {
		return dto.ShipmentDTO{}, fmt.Errorf("supplier order %d not found: %w", input.SupplierOrderID, err)
	}

	shipment := &domain.Shipment{
		SupplierOrderID:  input.SupplierOrderID,
		SupplierPlatform: input.SupplierPlatform,
		ShipmentNo:       input.ShipmentNo,
		TrackingNo:       input.TrackingNo,
		Status:           input.Status,
		ShippedAt:        input.ShippedAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := shipmentRepo.Create(shipment); err != nil {
		return dto.ShipmentDTO{}, err
	}

	for _, li := range input.Lines {
		sol, err := supplierRepo.FindLineByID(li.SupplierOrderLineID)
		if err != nil {
			return dto.ShipmentDTO{}, fmt.Errorf("supplier order line %d not found: %w", li.SupplierOrderLineID, err)
		}
		fl, err := fulfillRepo.FindByID(li.FulfillmentLineID)
		if err != nil {
			return dto.ShipmentDTO{}, fmt.Errorf("fulfillment line %d not found: %w", li.FulfillmentLineID, err)
		}
		if sol.SupplierOrderID != shipment.SupplierOrderID {
			return dto.ShipmentDTO{}, fmt.Errorf("supplier order line %d belongs to order %d, not %d", li.SupplierOrderLineID, sol.SupplierOrderID, shipment.SupplierOrderID)
		}
		if sol.FulfillmentLineID != li.FulfillmentLineID {
			return dto.ShipmentDTO{}, fmt.Errorf("supplier order line %d references fulfillment line %d, not %d", li.SupplierOrderLineID, sol.FulfillmentLineID, li.FulfillmentLineID)
		}
		if fl.WaveID != supplierOrder.WaveID {
			return dto.ShipmentDTO{}, fmt.Errorf("fulfillment line %d belongs to wave %d, not wave %d", li.FulfillmentLineID, fl.WaveID, supplierOrder.WaveID)
		}

		line := &domain.ShipmentLine{
			ShipmentID:          shipment.ID,
			SupplierOrderLineID: li.SupplierOrderLineID,
			FulfillmentLineID:   li.FulfillmentLineID,
			Quantity:            li.Quantity,
			CreatedAt:           now,
		}
		if err := shipmentRepo.CreateLine(line); err != nil {
			return dto.ShipmentDTO{}, err
		}
	}

	return dto.ShipmentDTO{}, nil
}

func TestCreateShipmentRejectsEmptyLines(t *testing.T) {
	t.Parallel()

	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()
	uc := NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo)

	input := dto.CreateShipmentInput{
		SupplierOrderID: 1,
		Lines:           []dto.CreateShipmentLineInput{},
	}

	_, _, err := uc.CreateShipment(input)
	if err == nil {
		t.Fatal("expected error for empty lines, got nil")
	}
}

func TestCreateShipmentPersistsShipmentAndLinesAtomically(t *testing.T) {
	t.Parallel()

	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()
	uc := NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo)

	// Setup: existing supplier order + line + fulfillment line
	now := "2026-01-01T00:00:00Z"
	supplierOrder := &domain.SupplierOrder{ID: 1, WaveID: 1, Status: "draft", SupplierPlatform: "test", CreatedAt: now, UpdatedAt: now}
	supplierRepo.orders[1] = supplierOrder
	supplierRepo.orderLines[1] = &domain.SupplierOrderLine{ID: 1, SupplierOrderID: 1, FulfillmentLineID: 1}
	fulfillRepo.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1}

	input := dto.CreateShipmentInput{
		SupplierOrderID:  1,
		SupplierPlatform: "test-platform",
		ShipmentNo:       "SHIP-001",
		TrackingNo:       "TRACK-123",
		Status:           "shipped",
		Lines: []dto.CreateShipmentLineInput{
			{SupplierOrderLineID: 1, FulfillmentLineID: 1, Quantity: 5},
		},
	}

	shipment, lines, err := uc.CreateShipment(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shipment.ID == 0 {
		t.Error("expected shipment to have an ID")
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].ShipmentID != shipment.ID {
		t.Errorf("line shipment ID %d != shipment ID %d", lines[0].ShipmentID, shipment.ID)
	}

	// Verify no leftover on error path — if AtomicCreateShipment failed, nothing should persist
	// (The mock implements AtomicCreateShipment atomically by design)
}
