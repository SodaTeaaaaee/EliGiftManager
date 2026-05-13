package app

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── In-memory mock repositories ──

type mockDemandRepo struct {
	mu    sync.Mutex
	docs  map[uint]*domain.DemandDocument
	lines map[uint][]*domain.DemandLine
	lastID uint
}

func newMockDemandRepo() *mockDemandRepo {
	return &mockDemandRepo{
		docs:  make(map[uint]*domain.DemandDocument),
		lines: make(map[uint][]*domain.DemandLine),
	}
}

func (m *mockDemandRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockDemandRepo) Create(doc *domain.DemandDocument) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc.ID = m.next()
	cp := *doc
	m.docs[doc.ID] = &cp
	return nil
}

func (m *mockDemandRepo) FindByID(id uint) (*domain.DemandDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.docs[id]
	if !ok {
		return nil, fmt.Errorf("demand document %d not found", id)
	}
	cp := *d
	return &cp, nil
}

func (m *mockDemandRepo) List() ([]domain.DemandDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.DemandDocument, 0, len(m.docs))
	for _, d := range m.docs {
		out = append(out, *d)
	}
	return out, nil
}

func (m *mockDemandRepo) CreateLine(line *domain.DemandLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.ID = m.next()
	cp := *line
	m.lines[line.DemandDocumentID] = append(m.lines[line.DemandDocumentID], &cp)
	return nil
}

func (m *mockDemandRepo) ListLinesByDocument(docID uint) ([]domain.DemandLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ll, ok := m.lines[docID]
	if !ok {
		return nil, nil
	}
	out := make([]domain.DemandLine, len(ll))
	for i, l := range ll {
		out[i] = *l
	}
	return out, nil
}

// ── mock wave repo ──

type mockWaveRepo struct {
	mu     sync.Mutex
	waves  map[uint]*domain.Wave
	lastID uint
}

func newMockWaveRepo() *mockWaveRepo {
	return &mockWaveRepo{waves: make(map[uint]*domain.Wave)}
}

func (m *mockWaveRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockWaveRepo) Create(wave *domain.Wave) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	wave.ID = m.next()
	cp := *wave
	m.waves[wave.ID] = &cp
	return nil
}

func (m *mockWaveRepo) FindByID(id uint) (*domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.waves[id]
	if !ok {
		return nil, fmt.Errorf("wave %d not found", id)
	}
	cp := *w
	return &cp, nil
}

func (m *mockWaveRepo) FindByWaveNo(waveNo string) (*domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.waves {
		if w.WaveNo == waveNo {
			cp := *w
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("wave %q not found", waveNo)
}

func (m *mockWaveRepo) List() ([]domain.Wave, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Wave, 0, len(m.waves))
	for _, w := range m.waves {
		out = append(out, *w)
	}
	return out, nil
}

func (m *mockWaveRepo) AddParticipant(snap *domain.WaveParticipantSnapshot) error {
	// no-op for tests
	return nil
}

func (m *mockWaveRepo) ListParticipantsByWave(waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	return nil, nil
}

// ── mock fulfill repo ──

type mockFulfillRepo struct {
	mu     sync.Mutex
	lines  map[uint]*domain.FulfillmentLine
	lastID uint
}

func newMockFulfillRepo() *mockFulfillRepo {
	return &mockFulfillRepo{lines: make(map[uint]*domain.FulfillmentLine)}
}

func (m *mockFulfillRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockFulfillRepo) Create(line *domain.FulfillmentLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.ID = m.next()
	cp := *line
	m.lines[line.ID] = &cp
	return nil
}

func (m *mockFulfillRepo) FindByID(id uint) (*domain.FulfillmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.lines[id]
	if !ok {
		return nil, fmt.Errorf("fulfillment line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

func (m *mockFulfillRepo) ListByWave(waveID uint) ([]domain.FulfillmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FulfillmentLine
	for _, l := range m.lines {
		if l.WaveID == waveID {
			out = append(out, *l)
		}
	}
	return out, nil
}

// ── mock rule repo ──

type mockRuleRepo struct{}

func newMockRuleRepo() *mockRuleRepo { return &mockRuleRepo{} }

func (m *mockRuleRepo) Create(rule *domain.AllocationPolicyRule) error { return nil }
func (m *mockRuleRepo) FindByID(id uint) (*domain.AllocationPolicyRule, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockRuleRepo) ListByWave(waveID uint) ([]domain.AllocationPolicyRule, error) {
	return nil, nil
}

// ── mock supplier repo ──

type mockSupplierRepo struct {
	mu        sync.Mutex
	orders    map[uint]*domain.SupplierOrder
	orderLines map[uint][]*domain.SupplierOrderLine
	lastID    uint
}

func newMockSupplierRepo() *mockSupplierRepo {
	return &mockSupplierRepo{
		orders:     make(map[uint]*domain.SupplierOrder),
		orderLines: make(map[uint][]*domain.SupplierOrderLine),
	}
}

func (m *mockSupplierRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockSupplierRepo) Create(order *domain.SupplierOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	order.ID = m.next()
	cp := *order
	m.orders[order.ID] = &cp
	return nil
}

func (m *mockSupplierRepo) FindByID(id uint) (*domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, fmt.Errorf("supplier order %d not found", id)
	}
	cp := *o
	return &cp, nil
}

func (m *mockSupplierRepo) List() ([]domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.SupplierOrder, 0, len(m.orders))
	for _, o := range m.orders {
		out = append(out, *o)
	}
	return out, nil
}

func (m *mockSupplierRepo) ListByWave(waveID uint) ([]domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.SupplierOrder
	for _, o := range m.orders {
		if o.WaveID == waveID {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (m *mockSupplierRepo) CreateLine(line *domain.SupplierOrderLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.ID = m.next()
	cp := *line
	m.orderLines[line.SupplierOrderID] = append(m.orderLines[line.SupplierOrderID], &cp)
	return nil
}

func (m *mockSupplierRepo) ListLinesByOrder(orderID uint) ([]domain.SupplierOrderLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ll, ok := m.orderLines[orderID]
	if !ok {
		return nil, nil
	}
	out := make([]domain.SupplierOrderLine, len(ll))
	for i, l := range ll {
		out[i] = *l
	}
	return out, nil
}

// ── Tests ──

func TestImportDemand(t *testing.T) {
	t.Parallel()

	repo := newMockDemandRepo()
	uc := NewDemandIntakeUseCase(repo)

	doc := &domain.DemandDocument{
		Kind:             "retail_order",
		CaptureMode:      "manual_entry",
		SourceChannel:    "test",
		SourceDocumentNo: "TEST-001",
	}
	lines := []*domain.DemandLine{
		{
			LineType:           "entitlement_rule",
			RoutingDisposition: "accepted",
			RequestedQuantity:  5,
		},
		{
			LineType:           "entitlement_rule",
			RoutingDisposition: "deferred",
			RequestedQuantity:  3,
		},
	}

	err := uc.ImportDemand(doc, lines)
	if err != nil {
		t.Fatalf("ImportDemand failed: %v", err)
	}

	if doc.ID == 0 {
		t.Error("expected doc.ID to be set after create")
	}
	if doc.Kind != "retail_order" {
		t.Errorf("expected Kind=retail_order, got %q", doc.Kind)
	}

	// Verify lines persisted
	persistedLines, err := repo.ListLinesByDocument(doc.ID)
	if err != nil {
		t.Fatalf("ListLinesByDocument failed: %v", err)
	}
	if len(persistedLines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(persistedLines))
	}
	for i, l := range persistedLines {
		if l.DemandDocumentID != doc.ID {
			t.Errorf("line %d: expected DemandDocumentID=%d, got %d", i, doc.ID, l.DemandDocumentID)
		}
	}
	if persistedLines[0].RequestedQuantity != 5 {
		t.Errorf("line 0: expected quantity 5, got %d", persistedLines[0].RequestedQuantity)
	}
}

func TestCreateWaveGeneratesWaveNo(t *testing.T) {
	t.Parallel()

	repo := newMockWaveRepo()
	uc := NewWaveUseCase(repo)

	wave := &domain.Wave{Name: "测试波次"}
	err := uc.CreateWave(wave)
	if err != nil {
		t.Fatalf("CreateWave failed: %v", err)
	}

	if wave.ID == 0 {
		t.Error("expected wave.ID to be set")
	}
	if wave.WaveNo == "" {
		t.Error("expected WaveNo to be generated")
	}
	if !strings.HasPrefix(wave.WaveNo, "WAVE-") {
		t.Errorf("expected WaveNo to start with 'WAVE-', got %q", wave.WaveNo)
	}
	if wave.LifecycleStage != "intake" {
		t.Errorf("expected lifecycleStage 'intake', got %q", wave.LifecycleStage)
	}

	// Create a second wave, should get sequential number
	wave2 := &domain.Wave{Name: "波次2"}
	err = uc.CreateWave(wave2)
	if err != nil {
		t.Fatalf("CreateWave 2 failed: %v", err)
	}
	if wave.WaveNo == wave2.WaveNo {
		t.Errorf("expected different WaveNo, both got %q", wave.WaveNo)
	}
}

func TestApplyRulesDemandDriven(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	ruleRepo := newMockRuleRepo()
	fulfillRepo := newMockFulfillRepo()

	// Setup: create a demand document with accepted + deferred lines
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:             "retail_order",
		CaptureMode:      "manual_entry",
		SourceChannel:    "test",
		SourceDocumentNo: "TEST-ALLOC",
	}
	lines := []*domain.DemandLine{
		{RoutingDisposition: "accepted", RequestedQuantity: 10, LineType: "sku_order"},
		{RoutingDisposition: "deferred", RequestedQuantity: 5, LineType: "sku_order"},
		{RoutingDisposition: "accepted", RequestedQuantity: 3, LineType: "sku_order"},
	}
	if err := demandUC.ImportDemand(doc, lines); err != nil {
		t.Fatalf("setup ImportDemand failed: %v", err)
	}

	allocUC := NewAllocationUseCase(demandRepo, ruleRepo, fulfillRepo)
	allocLines, err := allocUC.ApplyRules(1)
	if err != nil {
		t.Fatalf("ApplyRules failed: %v", err)
	}

	// Only accepted lines (2 out of 3) should produce fulfillment lines
	if len(allocLines) != 2 {
		t.Fatalf("expected 2 fulfillment lines (only accepted lines), got %d", len(allocLines))
	}

	for i, fl := range allocLines {
		if fl.AllocationState != "allocated" {
			t.Errorf("fulfillment line %d: expected state 'allocated', got %q", i, fl.AllocationState)
		}
		if fl.WaveID != 1 {
			t.Errorf("fulfillment line %d: expected WaveID=1, got %d", i, fl.WaveID)
		}
		if fl.ID == 0 {
			t.Errorf("fulfillment line %d: expected ID to be set", i)
		}
	}

	// Verify quantities match accepted lines
	expectedQuantities := []int{10, 3}
	for i, fl := range allocLines {
		if fl.Quantity != expectedQuantities[i] {
			t.Errorf("fulfillment line %d: expected quantity %d, got %d", i, expectedQuantities[i], fl.Quantity)
		}
	}
}

func TestExportSupplierOrder(t *testing.T) {
	t.Parallel()

	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	// Setup: create fulfillment lines
	waveID := uint(42)
	for i := 0; i < 3; i++ {
		err := fulfillRepo.Create(&domain.FulfillmentLine{
			WaveID:          waveID,
			Quantity:        10 + i,
			AllocationState: "allocated",
		})
		if err != nil {
			t.Fatalf("setup fulfill Create failed: %v", err)
		}
	}

	exportUC := NewExportUseCase(supplierRepo, fulfillRepo)
	order, err := exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		t.Fatalf("ExportSupplierOrder failed: %v", err)
	}

	if order.ID == 0 {
		t.Error("expected SupplierOrder.ID to be set")
	}
	if order.WaveID != waveID {
		t.Errorf("expected WaveID=%d, got %d", waveID, order.WaveID)
	}
	if order.Status != "draft" {
		t.Errorf("expected status 'draft', got %q", order.Status)
	}

	// Verify order lines
	orderLines, err := supplierRepo.ListLinesByOrder(order.ID)
	if err != nil {
		t.Fatalf("ListLinesByOrder failed: %v", err)
	}
	if len(orderLines) != 3 {
		t.Fatalf("expected 3 order lines, got %d", len(orderLines))
	}
	for i, ol := range orderLines {
		if ol.Status != "draft" {
			t.Errorf("order line %d: expected status 'draft', got %q", i, ol.Status)
		}
		if ol.SupplierOrderID != order.ID {
			t.Errorf("order line %d: expected SupplierOrderID=%d, got %d", i, order.ID, ol.SupplierOrderID)
		}
	}
}

func TestFullVerticalSlice(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	waveRepo := newMockWaveRepo()
	ruleRepo := newMockRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	// Step 1: Demand Intake
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:             "retail_order",
		CaptureMode:      "manual_entry",
		SourceChannel:    "test",
		SourceDocumentNo: "VS-001",
	}
	demandLines := []*domain.DemandLine{
		{RoutingDisposition: "accepted", RequestedQuantity: 7, LineType: "sku_order", ExternalTitle: "Widget A"},
		{RoutingDisposition: "accepted", RequestedQuantity: 2, LineType: "sku_order", ExternalTitle: "Widget B"},
	}
	if err := demandUC.ImportDemand(doc, demandLines); err != nil {
		t.Fatalf("Step 1 ImportDemand failed: %v", err)
	}

	// Step 2: Create Wave
	waveUC := NewWaveUseCase(waveRepo)
	wave := &domain.Wave{Name: "纵切面测试波次"}
	if err := waveUC.CreateWave(wave); err != nil {
		t.Fatalf("Step 2 CreateWave failed: %v", err)
	}
	if wave.WaveNo == "" || wave.ID == 0 {
		t.Fatal("Step 2 wave not properly created")
	}

	// Step 3: Apply Allocation Rules
	allocUC := NewAllocationUseCase(demandRepo, ruleRepo, fulfillRepo)
	allocLines, err := allocUC.ApplyRules(wave.ID)
	if err != nil {
		t.Fatalf("Step 3 ApplyRules failed: %v", err)
	}
	if len(allocLines) != 2 {
		t.Fatalf("Step 3: expected 2 fulfillment lines, got %d", len(allocLines))
	}

	// Step 4: Export Supplier Order
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo)
	order, err := exportUC.ExportSupplierOrder(wave.ID)
	if err != nil {
		t.Fatalf("Step 4 ExportSupplierOrder failed: %v", err)
	}
	if order.Status != "draft" {
		t.Errorf("Step 4: expected draft order, got %q", order.Status)
	}

	// Verify order lines link back to fulfillment lines
	orderLines, _ := supplierRepo.ListLinesByOrder(order.ID)
	if len(orderLines) != 2 {
		t.Errorf("Step 4: expected 2 order lines, got %d", len(orderLines))
	}
}
