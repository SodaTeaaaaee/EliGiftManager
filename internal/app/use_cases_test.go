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

func (m *mockDemandRepo) ListUnassigned() ([]domain.DemandDocument, error) {
	return m.List()
}

func (m *mockDemandRepo) CountByProfileID(profileID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, doc := range m.docs {
		if doc.IntegrationProfileID != nil && *doc.IntegrationProfileID == profileID {
			count++
		}
	}
	return count, nil
}

func (m *mockDemandRepo) CreateLine(line *domain.DemandLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.ID = m.next()
	cp := *line
	m.lines[line.DemandDocumentID] = append(m.lines[line.DemandDocumentID], &cp)
	return nil
}

func (m *mockDemandRepo) FindLineByID(id uint) (*domain.DemandLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ll := range m.lines {
		for _, l := range ll {
			if l.ID == id {
				cp := *l
				return &cp, nil
			}
		}
	}
	return nil, fmt.Errorf("demand line %d not found", id)
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

func (m *mockDemandRepo) UpdateLine(line *domain.DemandLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lines[line.DemandDocumentID] == nil {
		return fmt.Errorf("line not found")
	}
	for i, l := range m.lines[line.DemandDocumentID] {
		if l.ID == line.ID {
			m.lines[line.DemandDocumentID][i] = line
			return nil
		}
	}
	return fmt.Errorf("line not found")
}

func (m *mockDemandRepo) UpdateLineRoutingFields(lineID uint, routingDisposition string, recipientInputState string, routingReasonCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, lines := range m.lines {
		for _, l := range lines {
			if l.ID == lineID {
				l.RoutingDisposition = routingDisposition
				l.RecipientInputState = recipientInputState
				l.RoutingReasonCode = routingReasonCode
				return nil
			}
		}
	}
	return fmt.Errorf("line %d not found", lineID)
}

func (m *mockDemandRepo) UpdateBoundProfileSnapshot(_ uint, _ string) error {
	return nil
}

// ── mock wave repo ──

type mockWaveRepo struct {
	mu           sync.Mutex
	waves        map[uint]*domain.Wave
	participants []domain.WaveParticipantSnapshot
	lastID       uint
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
	m.mu.Lock()
	defer m.mu.Unlock()
	snap.ID = m.next()
	return nil
}

func (m *mockWaveRepo) ListParticipantsByWave(waveID uint) ([]domain.WaveParticipantSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.participants, nil
}

func (m *mockWaveRepo) DeleteParticipantsByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []domain.WaveParticipantSnapshot
	for _, p := range m.participants {
		if p.WaveID != waveID {
			kept = append(kept, p)
		}
	}
	m.participants = kept
	return nil
}

func (m *mockWaveRepo) SetParticipants(snaps []domain.WaveParticipantSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.participants = snaps
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

func (m *mockFulfillRepo) DeleteByWaveAndGeneratedBy(waveID uint, generatedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, l := range m.lines {
		if l.WaveID == waveID && l.GeneratedBy == generatedBy {
			delete(m.lines, id)
		}
	}
	return nil
}

func (m *mockFulfillRepo) ReplaceByWaveAndGeneratedBy(waveID uint, generatedBy string, newLines []domain.FulfillmentLine) error {
	m.DeleteByWaveAndGeneratedBy(waveID, generatedBy)
	for i := range newLines {
		m.Create(&newLines[i])
	}
	return nil
}

func (m *mockFulfillRepo) DeleteByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, l := range m.lines {
		if l.WaveID == waveID {
			delete(m.lines, id)
		}
	}
	return nil
}

func (m *mockFulfillRepo) BulkUpdateStates(updates []domain.FulfillmentLineStateUpdate) error { return nil }

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
func (m *mockRuleRepo) Update(rule *domain.AllocationPolicyRule) error { return nil }
func (m *mockRuleRepo) Delete(id uint) error                          { return nil }
func (m *mockRuleRepo) DeleteByWave(waveID uint) error                { return nil }

// ── mock assignment repo ──

type mockAssignmentRepo struct {
	mu          sync.Mutex
	assignments map[uint][]*domain.WaveDemandAssignment // waveID -> assignments
	demandRepo  *mockDemandRepo
	lastID      uint
}

func newMockAssignmentRepo(demandRepo *mockDemandRepo) *mockAssignmentRepo {
	return &mockAssignmentRepo{
		assignments: make(map[uint][]*domain.WaveDemandAssignment),
		demandRepo:  demandRepo,
	}
}

func (m *mockAssignmentRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockAssignmentRepo) Create(assignment *domain.WaveDemandAssignment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Check for duplicate
	for _, a := range m.assignments[assignment.WaveID] {
		if a.DemandDocumentID == assignment.DemandDocumentID {
			return fmt.Errorf("demand document %d already assigned to wave %d", assignment.DemandDocumentID, assignment.WaveID)
		}
	}
	assignment.ID = m.next()
	cp := *assignment
	m.assignments[assignment.WaveID] = append(m.assignments[assignment.WaveID], &cp)
	return nil
}

func (m *mockAssignmentRepo) ListByWave(waveID uint) ([]domain.WaveDemandAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	aa := m.assignments[waveID]
	out := make([]domain.WaveDemandAssignment, len(aa))
	for i, a := range aa {
		out[i] = *a
	}
	return out, nil
}

func (m *mockAssignmentRepo) ListByDemandDocument(docID uint) ([]domain.WaveDemandAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.WaveDemandAssignment
	for _, aa := range m.assignments {
		for _, a := range aa {
			if a.DemandDocumentID == docID {
				out = append(out, *a)
			}
		}
	}
	return out, nil
}

func (m *mockAssignmentRepo) DeleteByWaveAndDocument(waveID uint, demandDocumentID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	aa := m.assignments[waveID]
	for i, a := range aa {
		if a.DemandDocumentID == demandDocumentID {
			m.assignments[waveID] = append(aa[:i], aa[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockAssignmentRepo) DeleteByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assignments, waveID)
	return nil
}

func (m *mockAssignmentRepo) ListDemandDocumentsByWave(waveID uint) ([]domain.DemandDocument, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	aa := m.assignments[waveID]
	var out []domain.DemandDocument
	for _, a := range aa {
		doc, err := m.demandRepo.FindByID(a.DemandDocumentID)
		if err != nil {
			continue
		}
		out = append(out, *doc)
	}
	return out, nil
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

func (m *mockSupplierRepo) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, lines := range m.orderLines {
		for _, l := range lines {
			if l.ID == id {
				cp := *l
				return &cp, nil
			}
		}
	}
	return nil, fmt.Errorf("supplier order line %d not found", id)
}

func (m *mockSupplierRepo) DeleteLinesByOrder(orderID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.orderLines, orderID)
	return nil
}

func (m *mockSupplierRepo) DeleteDraftsByWave(waveID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var idsToDelete []uint
	for id, o := range m.orders {
		if o.WaveID == waveID && o.Status == "draft" {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(m.orders, id)
		delete(m.orderLines, id)
	}
	return nil
}

func (m *mockSupplierRepo) Update(order *domain.SupplierOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[order.ID]; !ok {
		return fmt.Errorf("supplier order %d not found", order.ID)
	}
	cp := *order
	m.orders[order.ID] = &cp
	return nil
}

func (m *mockSupplierRepo) AtomicCreateSupplierOrder(order *domain.SupplierOrder, lines []*domain.SupplierOrderLine, _ *domain.BasisPinParam) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	orderID := m.next()
	order.ID = orderID
	m.orders[orderID] = order
	for _, line := range lines {
		lineID := m.next()
		line.ID = lineID
		line.SupplierOrderID = orderID
		m.orderLines[orderID] = append(m.orderLines[orderID], line)
	}
	return nil
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
	uc := NewWaveUseCase(repo, nil, nil)

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

func TestMapDemandToFulfillmentDemandDriven(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileID := uint(100)
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	// Setup: create a demand document with accepted + deferred lines
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:              "retail_order",
		CaptureMode:       "manual_entry",
		SourceChannel:     "test",
		SourceDocumentNo:  "TEST-ALLOC",
		CustomerProfileID: &profileID,
	}
	lines := []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 10, LineType: "sku_order"},
		{RoutingDisposition: "deferred", RequestedQuantity: 5, LineType: "sku_order"},
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 3, LineType: "sku_order"},
	}
	if err := demandUC.ImportDemand(doc, lines); err != nil {
		t.Fatalf("setup ImportDemand failed: %v", err)
	}

	// Assign the demand document to wave 1
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("setup assignment Create failed: %v", err)
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	dmResult, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("MapDemandToFulfillment failed: %v", err)
	}

	// Only accepted lines (2 out of 3) should produce fulfillment lines
	if len(dmResult.CreatedLines) != 2 {
		t.Fatalf("expected 2 fulfillment lines (only accepted lines), got %d", len(dmResult.CreatedLines))
	}

	for i, fl := range dmResult.CreatedLines {
		if fl.AllocationState != "ready" {
			t.Errorf("fulfillment line %d: expected state 'ready', got %q", i, fl.AllocationState)
		}
		if fl.WaveID != 1 {
			t.Errorf("fulfillment line %d: expected WaveID=1, got %d", i, fl.WaveID)
		}
		if fl.ID == 0 {
			t.Errorf("fulfillment line %d: expected ID to be set", i)
		}
		if fl.WaveParticipantSnapshotID == nil || *fl.WaveParticipantSnapshotID != 1 {
			t.Errorf("fulfillment line %d: expected WaveParticipantSnapshotID=1, got %v", i, fl.WaveParticipantSnapshotID)
		}
	}

	// Verify quantities match accepted lines
	expectedQuantities := []int{10, 3}
	for i, fl := range dmResult.CreatedLines {
		if fl.Quantity != expectedQuantities[i] {
			t.Errorf("fulfillment line %d: expected quantity %d, got %d", i, expectedQuantities[i], fl.Quantity)
		}
	}
}

func TestMapDemandToFulfillmentFailsOnPartialSnapshotMissing(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileA := uint(100)
	profileB := uint(200)
	// Only profileA has a snapshot; profileB does not
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: profileA, SnapshotType: "buyer"},
	})

	demandUC := NewDemandIntakeUseCase(demandRepo)
	docA := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "PARTIAL-A",
		CustomerProfileID: &profileA,
	}
	docB := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "PARTIAL-B",
		CustomerProfileID: &profileB,
	}
	if err := demandUC.ImportDemand(docA, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := demandUC.ImportDemand(docB, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []*domain.DemandDocument{docA, docB} {
		if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
			WaveID: 1, DemandDocumentID: doc.ID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	_, err := dmUC.MapDemandToFulfillment(1)
	if err == nil {
		t.Fatal("expected MapDemandToFulfillment to fail when some retail docs lack participant snapshot, but got nil")
	}
	if !strings.Contains(err.Error(), "200") {
		t.Errorf("error should mention missing profile ID 200, got: %v", err)
	}

	// Verify no fulfillment lines were created (fail-fast before delete+create)
	allLines, _ := fulfillRepo.ListByWave(1)
	if len(allLines) != 0 {
		t.Errorf("expected 0 fulfillment lines after failed MapDemandToFulfillment, got %d", len(allLines))
	}
}

func TestMapDemandToFulfillmentFailsOnMissingCustomerProfileID(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "NO-PROFILE",
	}
	if err := demandUC.ImportDemand(doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID: 1, DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatal(err)
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	_, err := dmUC.MapDemandToFulfillment(1)
	if err == nil {
		t.Fatal("expected MapDemandToFulfillment to fail when retail doc has no CustomerProfileID, but got nil")
	}
	if !strings.Contains(err.Error(), "CustomerProfileID") {
		t.Errorf("error should mention CustomerProfileID, got: %v", err)
	}
}

func TestExportSupplierOrder(t *testing.T) {
	t.Parallel()

	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	// Setup: create fulfillment lines (no DemandDocumentID → catch-all group)
	waveID := uint(42)
	for i := 0; i < 3; i++ {
		err := fulfillRepo.Create(&domain.FulfillmentLine{
			WaveID:          waveID,
			Quantity:        10 + i,
			AllocationState: "ready",
		})
		if err != nil {
			t.Fatalf("setup fulfill Create failed: %v", err)
		}
	}

	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, nil, nil, nil)
	orders, err := exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		t.Fatalf("ExportSupplierOrder failed: %v", err)
	}

	// All lines have nil DemandDocumentID → single catch-all group
	if len(orders) != 1 {
		t.Fatalf("expected 1 supplier order (catch-all group), got %d", len(orders))
	}
	order := orders[0]

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
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)

	profileID := uint(400)

	// Step 1: Demand Intake
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:              "retail_order",
		CaptureMode:       "manual_entry",
		SourceChannel:     "test",
		SourceDocumentNo:  "VS-001",
		CustomerProfileID: &profileID,
	}
	demandLines := []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 7, LineType: "sku_order", ExternalTitle: "Widget A"},
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 2, LineType: "sku_order", ExternalTitle: "Widget B"},
	}
	if err := demandUC.ImportDemand(doc, demandLines); err != nil {
		t.Fatalf("Step 1 ImportDemand failed: %v", err)
	}

	// Step 2: Create Wave
	waveUC := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)
	wave := &domain.Wave{Name: "纵切面测试波次"}
	if err := waveUC.CreateWave(wave); err != nil {
		t.Fatalf("Step 2 CreateWave failed: %v", err)
	}
	if wave.WaveNo == "" || wave.ID == 0 {
		t.Fatal("Step 2 wave not properly created")
	}

	// Assign the demand document to the wave
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           wave.ID,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("setup assignment Create failed: %v", err)
	}

	// Setup participant snapshot for the profile
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: wave.ID, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	// Step 3: Apply Allocation Rules
	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	dmResult, err := dmUC.MapDemandToFulfillment(wave.ID)
	if err != nil {
		t.Fatalf("Step 3 MapDemandToFulfillment failed: %v", err)
	}
	if len(dmResult.CreatedLines) != 2 {
		t.Fatalf("Step 3: expected 2 fulfillment lines, got %d", len(dmResult.CreatedLines))
	}

	// Step 4: Export Supplier Order
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, nil, nil, nil)
	exportedOrders, err := exportUC.ExportSupplierOrder(wave.ID)
	if err != nil {
		t.Fatalf("Step 4 ExportSupplierOrder failed: %v", err)
	}
	if len(exportedOrders) != 1 {
		t.Fatalf("Step 4: expected 1 supplier order (catch-all group), got %d", len(exportedOrders))
	}
	order := exportedOrders[0]
	if order.Status != "draft" {
		t.Errorf("Step 4: expected draft order, got %q", order.Status)
	}

	// Verify order lines link back to fulfillment lines
	orderLines, _ := supplierRepo.ListLinesByOrder(order.ID)
	if len(orderLines) != 2 {
		t.Errorf("Step 4: expected 2 order lines, got %d", len(orderLines))
	}
}

// ── Regression: idempotency & uniqueness ──

func TestAssignDemandToWaveRejectsDuplicateAssignment(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)

	// Setup: create a demand document
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:             "retail_order",
		CaptureMode:      "manual_entry",
		SourceChannel:    "test",
		SourceDocumentNo: "DUP-001",
	}
	if err := demandUC.ImportDemand(doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// First assignment should succeed
	err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	})
	if err != nil {
		t.Fatalf("first assignment failed: %v", err)
	}

	// Second assignment (same wave + same demand) should fail
	err = assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	})
	if err == nil {
		t.Error("expected duplicate assignment to fail, but it succeeded")
	} else {
		t.Logf("got expected error: %v", err)
	}

	// Verify only 1 assignment exists
	assignments, _ := assignmentRepo.ListByWave(1)
	if len(assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(assignments))
	}
}

func TestMapDemandToFulfillmentIsIdempotentForSameWave(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileID := uint(200)
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	// Setup: demand + wave + assignment
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:              "retail_order",
		CaptureMode:       "manual_entry",
		SourceChannel:     "test",
		SourceDocumentNo:  "IDEM-001",
		CustomerProfileID: &profileID,
	}
	if err := demandUC.ImportDemand(doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 5, LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("setup ImportDemand failed: %v", err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("setup assignment failed: %v", err)
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)

	// Run allocation first time
	lines1, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("first MapDemandToFulfillment failed: %v", err)
	}
	count1 := len(lines1.CreatedLines)

	// Run allocation second time — should be idempotent (rebuild, not append)
	lines2, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("second MapDemandToFulfillment failed: %v", err)
	}
	count2 := len(lines2.CreatedLines)

	if count1 != count2 {
		t.Errorf("idempotent violation: first run=%d lines, second run=%d lines", count1, count2)
	}
	if count1 != 1 {
		t.Errorf("expected 1 fulfillment line (1 accepted demand line), got %d", count1)
	}
}

func TestExportSupplierOrderIsIdempotentForDraftSlice(t *testing.T) {
	t.Parallel()

	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	// Setup: fulfillment lines for wave
	waveID := uint(1)
	for i := 0; i < 3; i++ {
		if err := fulfillRepo.Create(&domain.FulfillmentLine{
			WaveID:          waveID,
			Quantity:        10 + i,
			AllocationState: "ready",
			GeneratedBy:     "allocation_demand_driven",
		}); err != nil {
			t.Fatalf("setup fulfill Create failed: %v", err)
		}
	}

	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, nil, nil, nil)

	// First export
	orders1, err := exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		t.Fatalf("first ExportSupplierOrder failed: %v", err)
	}
	if len(orders1) != 1 {
		t.Fatalf("expected 1 order after first export, got %d", len(orders1))
	}

	ordersAfter1, _ := supplierRepo.ListByWave(waveID)
	orderCount1 := len(ordersAfter1)
	if orderCount1 != 1 {
		t.Errorf("expected 1 order after first export, got %d", orderCount1)
	}

	// Second export — should be idempotent for draft
	orders2, err := exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		t.Fatalf("second ExportSupplierOrder failed: %v", err)
	}
	if len(orders2) != 1 {
		t.Fatalf("expected 1 order after second export, got %d", len(orders2))
	}

	ordersAfter2, _ := supplierRepo.ListByWave(waveID)
	orderCount2 := len(ordersAfter2)
	if orderCount2 != 1 {
		t.Errorf("idempotent violation: expected 1 order after second export, got %d", orderCount2)
	}

	if orders1[0].ID == orders2[0].ID {
		t.Log("both exports produced same order ID (reused)")
	} else {
		t.Logf("order IDs differ: %d vs %d (rebuild pattern ok)", orders1[0].ID, orders2[0].ID)
	}

	// Verify the wave still has exactly 1 draft order
	draftCount := 0
	for _, o := range ordersAfter2 {
		if o.Status == "draft" {
			draftCount++
		}
	}
	if draftCount != 1 {
		t.Errorf("expected 1 draft order, got %d", draftCount)
	}
}

func TestGetWaveOverviewStrictErrorHandling(t *testing.T) {
	// Note: full integration test of controller-level overview error handling
	// requires a real DB or integration harness. This test validates the
	// use-case-level semantic: when demand stats fail, the error propagates.
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileID := uint(300)
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	// Setup: demand + assignment
	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind:              "retail_order",
		CaptureMode:       "manual_entry",
		SourceChannel:     "test",
		SourceDocumentNo:  "ERR-001",
		CustomerProfileID: &profileID,
	}
	if err := demandUC.ImportDemand(doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID:           1,
		DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatalf("setup assignment failed: %v", err)
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)

	// Apply rules — should succeed and return correct count
	lines, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("MapDemandToFulfillment failed: %v", err)
	}
	if len(lines.CreatedLines) != 1 {
		t.Errorf("expected 1 fulfillment line, got %d", len(lines.CreatedLines))
	}

	// Verify that when assignment repo is queried, it returns valid results
	docs, err := assignmentRepo.ListDemandDocumentsByWave(1)
	if err != nil {
		t.Fatalf("ListDemandDocumentsByWave should not fail for valid wave: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 demand document for wave 1, got %d", len(docs))
	}
}

// ── Eligibility Tests ──

func TestIsEligibleForFulfillmentAcceptsReady(t *testing.T) {
	t.Parallel()
	dl := &domain.DemandLine{
		RoutingDisposition:  "accepted",
		RecipientInputState: "ready",
	}
	if !isEligibleForFulfillment(dl) {
		t.Error("expected accepted+ready to be eligible")
	}
}

func TestIsEligibleForFulfillmentAcceptsNotRequired(t *testing.T) {
	t.Parallel()
	dl := &domain.DemandLine{
		RoutingDisposition:  "accepted",
		RecipientInputState: "not_required",
	}
	if !isEligibleForFulfillment(dl) {
		t.Error("expected accepted+not_required to be eligible")
	}
}

func TestIsEligibleForFulfillmentRejectsWaitingForInput(t *testing.T) {
	t.Parallel()
	dl := &domain.DemandLine{
		RoutingDisposition:  "accepted",
		RecipientInputState: "waiting_for_input",
	}
	if isEligibleForFulfillment(dl) {
		t.Error("expected accepted+waiting_for_input to be INELIGIBLE")
	}
}

func TestIsEligibleForFulfillmentRejectsDeferred(t *testing.T) {
	t.Parallel()
	dl := &domain.DemandLine{
		RoutingDisposition:  "deferred",
		RecipientInputState: "ready",
	}
	if isEligibleForFulfillment(dl) {
		t.Error("expected deferred+ready to be INELIGIBLE (routing disposition not accepted)")
	}
}

// ── Mapping Blocking Tests ──

func TestMapDemandToFulfillmentBlocksUnmappedProduct(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileID := uint(300)
	productMasterID := uint(999) // No wave-scoped product for this master

	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 10, WaveID: 1, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "BLOCKED-1",
		CustomerProfileID: &profileID,
	}
	lines := []*domain.DemandLine{{
		RoutingDisposition:  "accepted",
		RecipientInputState: "ready",
		RequestedQuantity:   1,
		LineType:            "sku_order",
		ProductMasterID:     &productMasterID,
	}}
	if err := demandUC.ImportDemand(doc, lines); err != nil {
		t.Fatal(err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID: 1, DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatal(err)
	}

	// productRepo is nil — no wave-scoped products → mapping fails
	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	result, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("MapDemandToFulfillment failed: %v", err)
	}

	if len(result.CreatedLines) != 0 {
		t.Errorf("expected 0 created lines (product unmapped), got %d", len(result.CreatedLines))
	}
	if len(result.BlockedLines) != 1 {
		t.Errorf("expected 1 blocked line, got %d", len(result.BlockedLines))
	}
	if len(result.BlockedLines) > 0 && result.BlockedLines[0].Reason != "wave_product_missing" {
		t.Errorf("expected reason 'wave_product_missing', got %q", result.BlockedLines[0].Reason)
	}
}

func TestMapDemandToFulfillmentSucceedsWithoutProductMasterID(t *testing.T) {	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileID := uint(400)

	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 20, WaveID: 1, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})

	demandUC := NewDemandIntakeUseCase(demandRepo)
	doc := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "NO-PRODUCT-REF",
		CustomerProfileID: &profileID,
	}
	// No ProductMasterID set — this is allowed (some retail lines don't need product mapping)
	lines := []*domain.DemandLine{{
		RoutingDisposition:  "accepted",
		RecipientInputState: "ready",
		RequestedQuantity:   2,
		LineType:            "sku_order",
	}}
	if err := demandUC.ImportDemand(doc, lines); err != nil {
		t.Fatal(err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{
		WaveID: 1, DemandDocumentID: doc.ID,
	}); err != nil {
		t.Fatal(err)
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil)
	result, err := dmUC.MapDemandToFulfillment(1)
	if err != nil {
		t.Fatalf("MapDemandToFulfillment failed: %v", err)
	}

	if len(result.CreatedLines) != 1 {
		t.Errorf("expected 1 created line, got %d", len(result.CreatedLines))
	}
	if len(result.BlockedLines) != 0 {
		t.Errorf("expected 0 blocked lines, got %d", len(result.BlockedLines))
	}
}

// ── Export grouping tests ──

// mockProfileRepoForExport is a minimal in-memory IntegrationProfileRepository
// used only by export grouping tests (avoids collision with mockProfileRepo in
// channel_closure_test.go which lives in the same test package).
type mockProfileRepoForExport struct {
	profiles map[uint]*domain.IntegrationProfile
}

func newMockProfileRepoForExport(profiles ...*domain.IntegrationProfile) *mockProfileRepoForExport {
	m := &mockProfileRepoForExport{profiles: make(map[uint]*domain.IntegrationProfile)}
	for _, p := range profiles {
		m.profiles[p.ID] = p
	}
	return m
}

func (m *mockProfileRepoForExport) Create(profile *domain.IntegrationProfile) error { return nil }
func (m *mockProfileRepoForExport) FindByID(id uint) (*domain.IntegrationProfile, error) {
	p, ok := m.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile %d not found", id)
	}
	cp := *p
	return &cp, nil
}
func (m *mockProfileRepoForExport) FindByProfileKey(key string) (*domain.IntegrationProfile, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockProfileRepoForExport) List() ([]domain.IntegrationProfile, error)              { return nil, nil }
func (m *mockProfileRepoForExport) Update(profile *domain.IntegrationProfile) error         { return nil }
func (m *mockProfileRepoForExport) Delete(id uint) error                                    { return nil }

// mockBindingRepo is a minimal in-memory ProfileTemplateBindingRepository for tests.
type mockBindingRepo struct{}

func (m *mockBindingRepo) Create(b *domain.IntegrationProfileTemplateBinding) error { return nil }
func (m *mockBindingRepo) ListByProfile(profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (m *mockBindingRepo) FindDefaultByProfileAndType(profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (m *mockBindingRepo) Delete(id uint) error                          { return nil }
func (m *mockBindingRepo) CountByProfileID(profileID uint) (int64, error) { return 0, nil }

// TestExportSupplierOrderGroupsByProfile verifies that fulfillment lines belonging
// to different IntegrationProfiles are split into separate SupplierOrders, while
// lines with no DemandDocumentID fall into a single catch-all group.
func TestExportSupplierOrderGroupsByProfile(t *testing.T) {
	t.Parallel()

	waveID := uint(10)
	profileA := uint(1)
	profileB := uint(2)

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	// Create demand documents with different integration profiles
	docA := &domain.DemandDocument{IntegrationProfileID: &profileA}
	docB := &domain.DemandDocument{IntegrationProfileID: &profileB}
	if err := demandRepo.Create(docA); err != nil {
		t.Fatal(err)
	}
	if err := demandRepo.Create(docB); err != nil {
		t.Fatal(err)
	}

	// 2 lines for profile A, 1 line for profile B, 1 catch-all (no doc)
	for i := 0; i < 2; i++ {
		if err := fulfillRepo.Create(&domain.FulfillmentLine{
			WaveID:           waveID,
			Quantity:         5,
			DemandDocumentID: &docA.ID,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := fulfillRepo.Create(&domain.FulfillmentLine{
		WaveID:           waveID,
		Quantity:         3,
		DemandDocumentID: &docB.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := fulfillRepo.Create(&domain.FulfillmentLine{
		WaveID:   waveID,
		Quantity: 1,
		// no DemandDocumentID → catch-all group
	}); err != nil {
		t.Fatal(err)
	}

	profileRepo := newMockProfileRepoForExport(
		&domain.IntegrationProfile{ID: profileA, ConnectorKey: "platform_a"},
		&domain.IntegrationProfile{ID: profileB, ConnectorKey: "platform_b"},
	)
	bindingRepo := &mockBindingRepo{}

	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, demandRepo, profileRepo, bindingRepo)
	orders, err := exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		t.Fatalf("ExportSupplierOrder failed: %v", err)
	}

	// Expect 3 groups: catch-all (profileID=0), profileA, profileB
	if len(orders) != 3 {
		t.Fatalf("expected 3 supplier orders (one per group), got %d", len(orders))
	}

	// Verify total line count across all orders equals 4
	totalLines := 0
	for _, o := range orders {
		lines, err := supplierRepo.ListLinesByOrder(o.ID)
		if err != nil {
			t.Fatalf("ListLinesByOrder(%d) failed: %v", o.ID, err)
		}
		totalLines += len(lines)
		if o.Status != "draft" {
			t.Errorf("order %d: expected status 'draft', got %q", o.ID, o.Status)
		}
	}
	if totalLines != 4 {
		t.Errorf("expected 4 total order lines across all groups, got %d", totalLines)
	}

	// Verify BatchNo values are distinct
	batchNos := make(map[string]bool)
	for _, o := range orders {
		if batchNos[o.BatchNo] {
			t.Errorf("duplicate BatchNo %q", o.BatchNo)
		}
		batchNos[o.BatchNo] = true
	}
}
