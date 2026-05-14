package app

import (
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock supplier order repo for drift tests ──
// mockSupplierRepoForDrift only implements ListByWave; all other methods panic.

type mockSupplierRepoForDrift struct {
	mu     sync.Mutex
	orders []domain.SupplierOrder
	waveID uint // orders are scoped to this wave
}

func newMockSupplierRepoForDrift(waveID uint) *mockSupplierRepoForDrift {
	return &mockSupplierRepoForDrift{waveID: waveID}
}

func (m *mockSupplierRepoForDrift) add(o domain.SupplierOrder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders = append(m.orders, o)
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
	panic("not implemented")
}
func (m *mockSupplierRepoForDrift) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForDrift) DeleteLinesByOrder(orderID uint) error { panic("not implemented") }

// ── test setup ──

type driftTestSetup struct {
	supplierRepo    *mockSupplierRepoForDrift
	shipmentRepo    *mockShipmentRepo
	channelSyncRepo *mockChannelSyncRepo
	uc              BasisDriftDetectionUseCase
	waveID          uint
}

func newDriftTestSetup() *driftTestSetup {
	const waveID uint = 1
	sr := newMockSupplierRepoForDrift(waveID)
	sh := newMockShipmentRepo()
	cs := newMockChannelSyncRepo()
	return &driftTestSetup{
		supplierRepo:    sr,
		shipmentRepo:    sh,
		channelSyncRepo: cs,
		uc:              NewBasisDriftDetectionUseCase(sr, sh, cs),
		waveID:          waveID,
	}
}

// addSupplierOrder adds a supplier order scoped to the test wave.
func (d *driftTestSetup) addSupplierOrder(nodeID, storedHash string) {
	d.supplierRepo.add(domain.SupplierOrder{
		WaveID:              d.waveID,
		BasisHistoryNodeID:  nodeID,
		BasisProjectionHash: storedHash,
	})
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
