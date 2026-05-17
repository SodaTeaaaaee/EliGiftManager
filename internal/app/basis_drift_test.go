package app

import (
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock supplier order repo for drift tests ──
// mockSupplierRepoForDrift only implements ListByWave; all other methods panic.

type mockSupplierRepoForDrift struct {
	mu         sync.Mutex
	orders     []domain.SupplierOrder
	orderLines map[uint][]domain.SupplierOrderLine // orderID → lines
	lastID     uint
	waveID     uint // orders are scoped to this wave
}

func newMockSupplierRepoForDrift(waveID uint) *mockSupplierRepoForDrift {
	return &mockSupplierRepoForDrift{
		waveID:     waveID,
		orderLines: make(map[uint][]domain.SupplierOrderLine),
	}
}

func (m *mockSupplierRepoForDrift) nextID() uint { m.lastID++; return m.lastID }

func (m *mockSupplierRepoForDrift) add(o domain.SupplierOrder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o.ID == 0 {
		o.ID = m.nextID()
	}
	m.orders = append(m.orders, o)
}

// addLine attaches a SupplierOrderLine to an existing order by orderID.
func (m *mockSupplierRepoForDrift) addLine(orderID uint, line domain.SupplierOrderLine) {
	m.mu.Lock()
	defer m.mu.Unlock()
	line.SupplierOrderID = orderID
	m.orderLines[orderID] = append(m.orderLines[orderID], line)
}

func (m *mockSupplierRepoForDrift) ListByWave(waveID uint) ([]domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if waveID != m.waveID {
		return nil, nil
	}
	out := make([]domain.SupplierOrder, len(m.orders))
	copy(out, m.orders)
	return out, nil
}

// Stubs — unused by drift tests
func (m *mockSupplierRepoForDrift) Create(order *domain.SupplierOrder) error { panic("not implemented") }
func (m *mockSupplierRepoForDrift) FindByID(id uint) (*domain.SupplierOrder, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForDrift) List() ([]domain.SupplierOrder, error) { panic("not implemented") }
func (m *mockSupplierRepoForDrift) DeleteDraftsByWave(waveID uint) error  { panic("not implemented") }
func (m *mockSupplierRepoForDrift) CreateLine(line *domain.SupplierOrderLine) error {
	panic("not implemented")
}
func (m *mockSupplierRepoForDrift) ListLinesByOrder(orderID uint) ([]domain.SupplierOrderLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lines := m.orderLines[orderID]
	out := make([]domain.SupplierOrderLine, len(lines))
	copy(out, lines)
	return out, nil
}
func (m *mockSupplierRepoForDrift) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForDrift) DeleteLinesByOrder(orderID uint) error { panic("not implemented") }
func (m *mockSupplierRepoForDrift) Update(order *domain.SupplierOrder) error { panic("not implemented") }
func (m *mockSupplierRepoForDrift) AtomicCreateSupplierOrder(order *domain.SupplierOrder, lines []*domain.SupplierOrderLine, pin *domain.BasisPinParam) error {
	panic("not implemented")
}

// ── test setup ──

type driftTestSetup struct {
	supplierRepo    *mockSupplierRepoForDrift
	shipmentRepo    *mockShipmentRepo
	channelSyncRepo *mockChannelSyncRepo
	fulfillRepo     *mockFulfillRepo
	uc              BasisDriftDetectionUseCase
	waveID          uint
}

func newDriftTestSetup() *driftTestSetup {
	const waveID uint = 1
	sr := newMockSupplierRepoForDrift(waveID)
	sh := newMockShipmentRepo()
	cs := newMockChannelSyncRepo()
	fr := newMockFulfillRepo()
	return &driftTestSetup{
		supplierRepo:    sr,
		shipmentRepo:    sh,
		channelSyncRepo: cs,
		fulfillRepo:     fr,
		uc:              NewBasisDriftDetectionUseCase(sr, sh, cs, fr),
		waveID:          waveID,
	}
}

// addSupplierOrder adds a supplier order scoped to the test wave and returns its ID.
func (d *driftTestSetup) addSupplierOrder(nodeID, storedHash string) uint {
	d.supplierRepo.mu.Lock()
	id := d.supplierRepo.nextID()
	d.supplierRepo.mu.Unlock()
	d.supplierRepo.add(domain.SupplierOrder{
		ID:                  id,
		WaveID:              d.waveID,
		BasisHistoryNodeID:  nodeID,
		BasisProjectionHash: storedHash,
		Status:              "submitted",
	})
	return id
}

// addSupplierOrderWithStatus adds a supplier order with an explicit status and returns its ID.
func (d *driftTestSetup) addSupplierOrderWithStatus(nodeID, storedHash, status string) uint {
	d.supplierRepo.mu.Lock()
	id := d.supplierRepo.nextID()
	d.supplierRepo.mu.Unlock()
	d.supplierRepo.add(domain.SupplierOrder{
		ID:                  id,
		WaveID:              d.waveID,
		BasisHistoryNodeID:  nodeID,
		BasisProjectionHash: storedHash,
		Status:              status,
	})
	return id
}

// addFulfillmentLine adds a fulfillment line to the wave and returns its ID.
func (d *driftTestSetup) addFulfillmentLine() uint {
	fl := domain.FulfillmentLine{WaveID: d.waveID}
	if err := d.fulfillRepo.Create(&fl); err != nil {
		panic("driftTestSetup.addFulfillmentLine: " + err.Error())
	}
	return fl.ID
}

// addShipment adds a shipment scoped to the test wave via supplierOrderID 1.
func (d *driftTestSetup) addShipment(nodeID, storedHash string) {
	const soID uint = 1
	d.shipmentRepo.supplierOrderWave[soID] = d.waveID
	d.shipmentRepo.shipments[soID] = &domain.Shipment{
		ID:                  soID,
		SupplierOrderID:     soID,
		BasisHistoryNodeID:  nodeID,
		BasisProjectionHash: storedHash,
	}
}

// addSyncJob adds a channel sync job scoped to the test wave.
func (d *driftTestSetup) addSyncJob(nodeID, storedHash string) {
	job := &domain.ChannelSyncJob{
		WaveID:              d.waveID,
		Direction:           "push_tracking",
		Status:              "success",
		BasisHistoryNodeID:  nodeID,
		BasisProjectionHash: storedHash,
	}
	if err := d.channelSyncRepo.CreateJob(job); err != nil {
		panic("driftTestSetup.addSyncJob: " + err.Error())
	}
}

// ── tests ──

func TestBasisDriftNoExternalObjects(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals, got %d", len(signals))
	}
}

func TestBasisDriftObjectWithoutBasisNodeSkipped(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// All three object types with empty BasisHistoryNodeID
	d.addSupplierOrder("", "some-hash")
	d.addShipment("", "some-hash")
	d.addSyncJob("", "some-hash")

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "current-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals (all skipped), got %d", len(signals))
	}
}

func TestBasisDriftProjectionHashMismatch(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Supplier order with a stored hash that differs from the current projection hash
	d.addSupplierOrder("node-abc", "hash-old")

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "hash-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	s := signals[0]
	if s.BasisDriftStatus != "drifted" {
		t.Errorf("BasisDriftStatus = %q, want %q", s.BasisDriftStatus, "drifted")
	}
	if len(s.DriftReasonCodes) != 1 || s.DriftReasonCodes[0] != "projection_changed" {
		t.Errorf("DriftReasonCodes = %v, want [projection_changed]", s.DriftReasonCodes)
	}
	if s.BasisKind != "supplier_order_basis" {
		t.Errorf("BasisKind = %q, want %q", s.BasisKind, "supplier_order_basis")
	}
}

func TestBasisDriftEmptyProjectionHash(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Shipment has a node ID but no stored hash — basis infra not yet populating hashes
	d.addShipment("node-xyz", "")

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "current-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	s := signals[0]
	if s.BasisDriftStatus != "drifted" {
		t.Errorf("BasisDriftStatus = %q, want %q", s.BasisDriftStatus, "drifted")
	}
	if len(s.DriftReasonCodes) != 1 || s.DriftReasonCodes[0] != "external_basis_stale" {
		t.Errorf("DriftReasonCodes = %v, want [external_basis_stale]", s.DriftReasonCodes)
	}
	if s.BasisKind != "shipment_basis" {
		t.Errorf("BasisKind = %q, want %q", s.BasisKind, "shipment_basis")
	}
}

func TestBasisDriftAllInSync(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	const hash = "hash-v42"
	d.addSyncJob("node-sync-1", hash)

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	s := signals[0]
	if s.BasisDriftStatus != "in_sync" {
		t.Errorf("BasisDriftStatus = %q, want %q", s.BasisDriftStatus, "in_sync")
	}
	if s.ReviewRequirement != "none" {
		t.Errorf("ReviewRequirement = %q, want %q", s.ReviewRequirement, "none")
	}
	if s.DriftReasonCodes != nil {
		t.Errorf("DriftReasonCodes = %v, want nil", s.DriftReasonCodes)
	}
	if s.BasisKind != "channel_sync_basis" {
		t.Errorf("BasisKind = %q, want %q", s.BasisKind, "channel_sync_basis")
	}
}

func TestBasisDriftCurrentHashUnavailable(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Object has nodeID + storedHash, but currentHash is "" (Phase 9 not active)
	// This is the active production path today
	d.addSupplierOrder("node-abc", "hash-stored")

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	s := signals[0]
	if s.BasisDriftStatus != "drifted" {
		t.Errorf("BasisDriftStatus = %q, want %q", s.BasisDriftStatus, "drifted")
	}
	if s.ReviewRequirement != "recommended" {
		t.Errorf("ReviewRequirement = %q, want %q", s.ReviewRequirement, "recommended")
	}
	if len(s.DriftReasonCodes) != 1 || s.DriftReasonCodes[0] != "projection_hash_unavailable" {
		t.Errorf("DriftReasonCodes = %v, want [projection_hash_unavailable]", s.DriftReasonCodes)
	}
}

// TestBasisDriftTargetDeletedTriggersRequired verifies that a submitted supplier order
// whose line references a fulfillment line that no longer exists in the wave emits
// a "required" signal with reason code "target_deleted".
func TestBasisDriftTargetDeletedTriggersRequired(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Add a submitted order (no basis node — structural check is independent of hash state).
	orderID := d.addSupplierOrderWithStatus("", "", "submitted")

	// Attach a line referencing fulfillment line ID 999, which does NOT exist in the wave.
	d.supplierRepo.addLine(orderID, domain.SupplierOrderLine{
		FulfillmentLineID: 999,
	})

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "hash-current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the structural signal.
	var found *dto.BasisDriftSignalDTO
	for i := range signals {
		if signals[i].BasisKind == "supplier_order" {
			found = &signals[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a supplier_order structural signal, got signals: %v", signals)
	}
	if found.ReviewRequirement != "required" {
		t.Errorf("ReviewRequirement = %q, want %q", found.ReviewRequirement, "required")
	}
	if len(found.DriftReasonCodes) != 1 || found.DriftReasonCodes[0] != "target_deleted" {
		t.Errorf("DriftReasonCodes = %v, want [target_deleted]", found.DriftReasonCodes)
	}
	if found.BasisDriftStatus != "drifted" {
		t.Errorf("BasisDriftStatus = %q, want %q", found.BasisDriftStatus, "drifted")
	}
}

// TestBasisDriftDraftOrdersSkipped verifies that draft supplier orders with orphaned
// line references do NOT trigger a "required" structural signal, because drafts are
// rebuilt on re-export and carry no structural commitment.
func TestBasisDriftDraftOrdersSkipped(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Add a draft order with a line referencing a non-existent fulfillment line.
	orderID := d.addSupplierOrderWithStatus("", "", "draft")
	d.supplierRepo.addLine(orderID, domain.SupplierOrderLine{
		FulfillmentLineID: 999,
	})

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "hash-current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range signals {
		if s.BasisKind == "supplier_order" && s.ReviewRequirement == "required" {
			t.Errorf("draft order should not produce a required signal, got: %+v", s)
		}
	}
}

// TestBasisDriftNoStructuralIssueStaysRecommended verifies that a hash mismatch with
// all fulfillment line references intact stays at "recommended", not "required".
func TestBasisDriftNoStructuralIssueStaysRecommended(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	// Add a real fulfillment line in the wave.
	flID := d.addFulfillmentLine()

	// Add a submitted order with a line referencing that valid fulfillment line,
	// but with a stored hash that differs from the current hash.
	orderID := d.addSupplierOrderWithStatus("node-abc", "hash-old", "submitted")
	d.supplierRepo.addLine(orderID, domain.SupplierOrderLine{
		FulfillmentLineID: flID,
	})

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, "hash-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range signals {
		if s.ReviewRequirement == "required" {
			t.Errorf("expected no required signals when all references are valid, got: %+v", s)
		}
	}

	// Confirm the hash-mismatch signal is still present as recommended.
	var hashSignal *dto.BasisDriftSignalDTO
	for i := range signals {
		if signals[i].BasisKind == "supplier_order_basis" {
			hashSignal = &signals[i]
			break
		}
	}
	if hashSignal == nil {
		t.Fatal("expected a supplier_order_basis hash-mismatch signal")
	}
	if hashSignal.ReviewRequirement != "recommended" {
		t.Errorf("ReviewRequirement = %q, want %q", hashSignal.ReviewRequirement, "recommended")
	}
}

// TestBasisDriftInSyncPlusRequiredCannotOccur verifies the invariant: even when all
// hash-based signals are in_sync, a structural target_deleted signal still surfaces
// with ReviewRequirement "required". The two layers are independent.
func TestBasisDriftInSyncPlusRequiredCannotOccur(t *testing.T) {
	t.Parallel()
	d := newDriftTestSetup()

	const hash = "hash-v1"

	// Add a submitted order whose hash is in sync, but whose line references a
	// fulfillment line that no longer exists — structural unsafety despite hash match.
	orderID := d.addSupplierOrderWithStatus("node-abc", hash, "submitted")
	d.supplierRepo.addLine(orderID, domain.SupplierOrderLine{
		FulfillmentLineID: 999, // does not exist in wave
	})

	signals, err := d.uc.DetectWaveBasisDrift(d.waveID, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var hasInSync, hasRequired bool
	for _, s := range signals {
		if s.BasisDriftStatus == "in_sync" {
			hasInSync = true
		}
		if s.ReviewRequirement == "required" {
			hasRequired = true
		}
	}

	// Hash layer sees in_sync; structural layer sees required — both must be present.
	if !hasInSync {
		t.Error("expected at least one in_sync signal from hash layer")
	}
	if !hasRequired {
		t.Error("expected at least one required signal from structural layer")
	}
}
