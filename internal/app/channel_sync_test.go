package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock channel sync repo ──

type mockChannelSyncRepo struct {
	mu              sync.Mutex
	jobs            map[uint]*domain.ChannelSyncJob
	jobItems        map[uint][]*domain.ChannelSyncItem
	lastID          uint
	failOnItemIndex int
}

func newMockChannelSyncRepo() *mockChannelSyncRepo {
	return &mockChannelSyncRepo{
		jobs:            make(map[uint]*domain.ChannelSyncJob),
		jobItems:        make(map[uint][]*domain.ChannelSyncItem),
		failOnItemIndex: -1,
	}
}

func (m *mockChannelSyncRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockChannelSyncRepo) CreateJob(job *domain.ChannelSyncJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job.ID = m.next()
	cp := *job
	m.jobs[job.ID] = &cp
	return nil
}

func (m *mockChannelSyncRepo) FindJobByID(id uint) (*domain.ChannelSyncJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("channel sync job %d not found", id)
	}
	cp := *j
	return &cp, nil
}

func (m *mockChannelSyncRepo) SaveJob(job *domain.ChannelSyncJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *job
	m.jobs[job.ID] = &cp
	return nil
}

func (m *mockChannelSyncRepo) SaveItem(item *domain.ChannelSyncItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *item
	for i, existing := range m.jobItems[item.ChannelSyncJobID] {
		if existing.ID == item.ID {
			m.jobItems[item.ChannelSyncJobID][i] = &cp
			return nil
		}
	}
	return fmt.Errorf("item %d not found in job %d", item.ID, item.ChannelSyncJobID)
}

func (m *mockChannelSyncRepo) ListJobsByWave(waveID uint) ([]domain.ChannelSyncJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ChannelSyncJob
	for _, j := range m.jobs {
		if j.WaveID == waveID {
			out = append(out, *j)
		}
	}
	return out, nil
}

func (m *mockChannelSyncRepo) CreateItem(item *domain.ChannelSyncItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item.ID = m.next()
	cp := *item
	m.jobItems[item.ChannelSyncJobID] = append(m.jobItems[item.ChannelSyncJobID], &cp)
	return nil
}

func (m *mockChannelSyncRepo) AtomicCreateChannelSync(job *domain.ChannelSyncJob, items []*domain.ChannelSyncItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobID := m.next()
	itemIDs := make([]uint, len(items))
	for i := range items {
		itemIDs[i] = m.next()
	}

	pendingJob := *job
	pendingJob.ID = jobID
	pendingItems := make([]*domain.ChannelSyncItem, len(items))
	for i, item := range items {
		if m.failOnItemIndex >= 0 && i == m.failOnItemIndex {
			return fmt.Errorf("mock: fail on item index %d", i)
		}
		cp := *item
		cp.ChannelSyncJobID = jobID
		cp.ID = itemIDs[i]
		pendingItems[i] = &cp
	}

	m.jobs[jobID] = &pendingJob
	m.jobItems[jobID] = pendingItems

	job.ID = jobID
	for i := range items {
		items[i].ChannelSyncJobID = jobID
		items[i].ID = itemIDs[i]
	}
	return nil
}

func (m *mockChannelSyncRepo) ListItemsByJob(jobID uint) ([]domain.ChannelSyncItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ll, ok := m.jobItems[jobID]
	if !ok {
		return nil, nil
	}
	out := make([]domain.ChannelSyncItem, len(ll))
	for i, it := range ll {
		out[i] = *it
	}
	return out, nil
}

// ── mock shipment repo ──

type mockShipmentRepoForSync struct {
	mu                sync.Mutex
	shipments         map[uint]*domain.Shipment
	lines             map[uint][]domain.ShipmentLine
	lastID            uint
	supplierOrderWave map[uint]uint // supplierOrderID → waveID
}

func newMockShipmentRepoForSync() *mockShipmentRepoForSync {
	return &mockShipmentRepoForSync{
		shipments:         make(map[uint]*domain.Shipment),
		lines:             make(map[uint][]domain.ShipmentLine),
		supplierOrderWave: make(map[uint]uint),
	}
}

func (m *mockShipmentRepoForSync) setSupplierOrderWave(supplierOrderID, waveID uint) {
	m.supplierOrderWave[supplierOrderID] = waveID
}

func (m *mockShipmentRepoForSync) FindByID(id uint) (*domain.Shipment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.shipments[id]
	if !ok {
		return nil, fmt.Errorf("shipment %d not found", id)
	}
	cp := *s
	return &cp, nil
}

func (m *mockShipmentRepoForSync) ListLinesByShipment(shipmentID uint) ([]domain.ShipmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ll, ok := m.lines[shipmentID]
	if !ok {
		return nil, nil
	}
	out := make([]domain.ShipmentLine, len(ll))
	copy(out, ll)
	return out, nil
}

func (m *mockShipmentRepoForSync) add(s *domain.Shipment) {
	m.lastID++
	s.ID = m.lastID
	cp := *s
	m.shipments[s.ID] = &cp
}

func (m *mockShipmentRepoForSync) addLine(line domain.ShipmentLine) {
	m.lines[line.ShipmentID] = append(m.lines[line.ShipmentID], line)
}

// Stubs
func (m *mockShipmentRepoForSync) Create(shipment *domain.Shipment) error          { panic("not implemented") }
func (m *mockShipmentRepoForSync) ListBySupplierOrder(supplierOrderID uint) ([]domain.Shipment, error) {
	panic("not implemented")
}
func (m *mockShipmentRepoForSync) ListByWave(waveID uint) ([]domain.Shipment, error) {
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
func (m *mockShipmentRepoForSync) CreateLine(line *domain.ShipmentLine) error { panic("not implemented") }
func (m *mockShipmentRepoForSync) AtomicCreateShipment(shipment *domain.Shipment, lines []*domain.ShipmentLine) error {
	panic("not implemented")
}

// ── mock supplier order repo ──

type mockSupplierRepoForSync struct {
	mu     sync.Mutex
	orders map[uint]*domain.SupplierOrder
}

func newMockSupplierRepoForSync() *mockSupplierRepoForSync {
	return &mockSupplierRepoForSync{
		orders: make(map[uint]*domain.SupplierOrder),
	}
}

func (m *mockSupplierRepoForSync) FindByID(id uint) (*domain.SupplierOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, fmt.Errorf("supplier order %d not found", id)
	}
	cp := *o
	return &cp, nil
}

// Stubs
func (m *mockSupplierRepoForSync) Create(order *domain.SupplierOrder) error          { panic("not implemented") }
func (m *mockSupplierRepoForSync) List() ([]domain.SupplierOrder, error)              { panic("not implemented") }
func (m *mockSupplierRepoForSync) ListByWave(waveID uint) ([]domain.SupplierOrder, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForSync) DeleteDraftsByWave(waveID uint) error { panic("not implemented") }
func (m *mockSupplierRepoForSync) CreateLine(line *domain.SupplierOrderLine) error { panic("not implemented") }
func (m *mockSupplierRepoForSync) ListLinesByOrder(orderID uint) ([]domain.SupplierOrderLine, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForSync) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	panic("not implemented")
}
func (m *mockSupplierRepoForSync) DeleteLinesByOrder(orderID uint) error { panic("not implemented") }

// ── mock fulfill repo ──

type mockFulfillRepoForSync struct {
	mu    sync.Mutex
	lines map[uint]*domain.FulfillmentLine
}

func newMockFulfillRepoForSync() *mockFulfillRepoForSync {
	return &mockFulfillRepoForSync{lines: make(map[uint]*domain.FulfillmentLine)}
}

func (m *mockFulfillRepoForSync) FindByID(id uint) (*domain.FulfillmentLine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.lines[id]
	if !ok {
		return nil, fmt.Errorf("fulfillment line %d not found", id)
	}
	cp := *l
	return &cp, nil
}

// Stubs
func (m *mockFulfillRepoForSync) Create(line *domain.FulfillmentLine) error { panic("not implemented") }
func (m *mockFulfillRepoForSync) ListByWave(waveID uint) ([]domain.FulfillmentLine, error) {
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
func (m *mockFulfillRepoForSync) DeleteByWaveAndGeneratedBy(waveID uint, generatedBy string) error {
	panic("not implemented")
}

// ── helper: build valid setup ──

type syncTestSetup struct {
	channelSync *mockChannelSyncRepo
	shipment    *mockShipmentRepoForSync
	supplier    *mockSupplierRepoForSync
	fulfill     *mockFulfillRepoForSync
	uc          ChannelSyncUseCase

	waveID          uint
	shipmentID      uint
	fulfillLineID   uint
	supplierOrderID uint
}

func newSyncTestSetup() *syncTestSetup {
	cs := newMockChannelSyncRepo()
	sh := newMockShipmentRepoForSync()
	su := newMockSupplierRepoForSync()
	fl := newMockFulfillRepoForSync()

	// Pre-create: supplier order 1 (wave 1) -> shipment 1 -> shipment line (fulfillLine 1)
	so := &domain.SupplierOrder{ID: 1, WaveID: 1}
	su.orders[1] = so
	sh.setSupplierOrderWave(1, 1)
	sh.add(&domain.Shipment{SupplierOrderID: 1})
	sh.addLine(domain.ShipmentLine{ShipmentID: 1, FulfillmentLineID: 1})
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1}

	return &syncTestSetup{
		channelSync:     cs,
		shipment:        sh,
		supplier:        su,
		fulfill:         fl,
		uc:              NewChannelSyncUseCase(cs, sh, su, fl),
		waveID:          1,
		shipmentID:      1,
		fulfillLineID:   1,
		supplierOrderID: 1,
	}
}

func (s *syncTestSetup) validInput() dto.CreateChannelSyncJobInput {
	return dto.CreateChannelSyncJobInput{
		WaveID:               s.waveID,
		IntegrationProfileID: 5,
		Direction:            "push_tracking",
		Items: []dto.CreateChannelSyncItemInput{
			{
				FulfillmentLineID: s.fulfillLineID,
				ShipmentID:        s.shipmentID,
				TrackingNo:        "TRACK-001",
				CarrierCode:       "SF",
			},
		},
	}
}

// ── tests ──

func TestCreateChannelSyncJobPersistsJobAndItems(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	job, items, err := s.uc.CreateChannelSyncJob(s.validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ID == 0 {
		t.Error("expected job.ID > 0")
	}
	if job.Status != "pending" {
		t.Errorf("job.Status = %q, want %q", job.Status, "pending")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ChannelSyncJobID != job.ID {
		t.Errorf("item ChannelSyncJobID = %d, want %d", items[0].ChannelSyncJobID, job.ID)
	}
}

func TestCreateChannelSyncJobRejectsEmptyItems(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	input := s.validInput()
	input.Items = nil

	job, items, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for empty items, got nil")
	}
	if job != nil || items != nil {
		t.Error("expected nil returns on error")
	}
}

func TestCreateChannelSyncJobRejectsZeroIntegrationProfileID(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	input := s.validInput()
	input.IntegrationProfileID = 0

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for zero integration_profile_id, got nil")
	}
}

func TestCreateChannelSyncJobRejectsInvalidDirection(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	tests := []struct {
		name      string
		direction string
	}{
		{"empty", ""},
		{"unknown", "pull_status"},
		{"typo", "push_trackingg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := s.validInput()
			input.Direction = tt.direction
			_, _, err := s.uc.CreateChannelSyncJob(input)
			if err == nil {
				t.Errorf("expected error for direction %q, got nil", tt.direction)
			}
		})
	}
}

func TestCreateChannelSyncJobRejectsNonexistentShipment(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	input := s.validInput()
	input.Items[0].ShipmentID = 999

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for nonexistent shipment, got nil")
	}
}

func TestCreateChannelSyncJobRejectsNonexistentFulfillmentLine(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	input := s.validInput()
	input.Items[0].FulfillmentLineID = 999

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for nonexistent fulfillment line, got nil")
	}
}

func TestCreateChannelSyncJobRejectsShipmentOutsideWave(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	// Add a shipment linked to supplier order in wave 2
	s.supplier.orders[2] = &domain.SupplierOrder{ID: 2, WaveID: 2}
	s.shipment.setSupplierOrderWave(2, 2)
	s.shipment.add(&domain.Shipment{SupplierOrderID: 2})
	s.shipment.addLine(domain.ShipmentLine{ShipmentID: 2, FulfillmentLineID: 1})

	input := s.validInput()
	input.Items[0].ShipmentID = 2

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for shipment outside wave, got nil")
	}
}

func TestCreateChannelSyncJobRejectsFulfillmentLineOutsideWave(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	// Add a fulfillment line in wave 2, and a shipment in wave 1 covering it
	s.fulfill.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 2}
	s.shipment.addLine(domain.ShipmentLine{ShipmentID: 1, FulfillmentLineID: 2})

	input := s.validInput()
	input.Items[0].FulfillmentLineID = 2

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for fulfillment line outside wave, got nil")
	}
}

func TestCreateChannelSyncJobRejectsUnlinkedShipmentAndFulfillmentLine(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	// fulfillment line 2 exists in wave 1 but isn't covered by any shipment line of shipment 1
	s.fulfill.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 1}

	input := s.validInput()
	input.Items[0].FulfillmentLineID = 2

	_, _, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error for unlinked shipment/fulfillment, got nil")
	}
}

func TestCreateChannelSyncJobRollsBackWhenItemPersistenceFails(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()
	s.channelSync.failOnItemIndex = 1

	input := s.validInput()
	input.Items = append(input.Items, dto.CreateChannelSyncItemInput{
		FulfillmentLineID: s.fulfillLineID,
		ShipmentID:        s.shipmentID,
		TrackingNo:        "TRACK-002",
		CarrierCode:       "SF",
	})

	job, items, err := s.uc.CreateChannelSyncJob(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if job != nil || items != nil {
		t.Error("expected nil returns on error")
	}
	if len(s.channelSync.jobs) != 0 {
		t.Errorf("expected 0 jobs after rollback, got %d", len(s.channelSync.jobs))
	}
	if len(s.channelSync.jobItems) != 0 {
		t.Errorf("expected 0 item maps after rollback, got %d", len(s.channelSync.jobItems))
	}
}

func TestListChannelSyncJobsByWaveReturnsCorrectSets(t *testing.T) {
	t.Parallel()
	s := newSyncTestSetup()

	j1 := &domain.ChannelSyncJob{WaveID: 1, Direction: "push_tracking", Status: "pending"}
	if err := s.channelSync.CreateJob(j1); err != nil {
		t.Fatalf("CreateJob 1: %v", err)
	}
	j2 := &domain.ChannelSyncJob{WaveID: 2, Direction: "push_tracking", Status: "pending"}
	if err := s.channelSync.CreateJob(j2); err != nil {
		t.Fatalf("CreateJob 2: %v", err)
	}

	w1, _ := s.channelSync.ListJobsByWave(1)
	if len(w1) != 1 {
		t.Errorf("expected 1 job for wave 1, got %d", len(w1))
	}
	w3, _ := s.channelSync.ListJobsByWave(3)
	if len(w3) != 0 {
		t.Errorf("expected 0 jobs for wave 3, got %d", len(w3))
	}
}
