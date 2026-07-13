package app

import (
	"context"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type failingProductRepo struct{}

func (failingProductRepo) Create(_ context.Context, _ *domain.Product) error {
	panic("not implemented")
}

func (failingProductRepo) FindByID(_ context.Context, _ uint) (*domain.Product, error) {
	panic("not implemented")
}

func (failingProductRepo) FindByWaveAndID(_ context.Context, _ uint, _ uint) (*domain.Product, error) {
	panic("not implemented")
}

func (failingProductRepo) ListByWave(_ context.Context, _ uint) ([]domain.Product, error) {
	return nil, errTest("product repo unavailable")
}

func (failingProductRepo) FindByWaveAndSKU(_ context.Context, _ uint, _ string, _ string) (*domain.Product, error) {
	panic("not implemented")
}
func (failingProductRepo) DeleteByWave(_ context.Context, _ uint) error { panic("not implemented") }

type staticProductRepo struct {
	products []domain.Product
}

func (s staticProductRepo) Create(_ context.Context, _ *domain.Product) error {
	panic("not implemented")
}

func (s staticProductRepo) FindByID(_ context.Context, _ uint) (*domain.Product, error) {
	panic("not implemented")
}

func (s staticProductRepo) FindByWaveAndID(_ context.Context, _ uint, _ uint) (*domain.Product, error) {
	panic("not implemented")
}

func (s staticProductRepo) ListByWave(_ context.Context, _ uint) ([]domain.Product, error) {
	return s.products, nil
}

func (s staticProductRepo) FindByWaveAndSKU(_ context.Context, _ uint, _ string, _ string) (*domain.Product, error) {
	panic("not implemented")
}
func (s staticProductRepo) DeleteByWave(_ context.Context, _ uint) error { panic("not implemented") }

func errTest(msg string) error { return &testErr{msg: msg} }

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }

func TestBuildBaseOverviewPropagatesProductRepoError(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	profileRepo.profiles[1] = &domain.IntegrationProfile{
		ID:               1,
		TrackingSyncMode: "api_push",
		ClosurePolicy:    "close_after_sync",
	}
	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		failingProductRepo{}, profileRepo, newMockHistoryScopeRepo(), newMockHistoryNodeRepo(), NewWaveOverviewProjectionUseCase(newMockChannelSyncRepo(), newMockClosureDecisionRepo(), noopDriftUC{}, noopHistoryHeadUC{}),
	)

	wave := &domain.Wave{Name: "overview-error"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(context.Background(), wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	_, err := queryUC.BuildBaseOverview(context.Background(), wave.ID)
	if err == nil {
		t.Fatal("expected BuildBaseOverview to fail when product repo fails, got nil")
	}
	if !strings.Contains(err.Error(), "product repo unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDashboardRowsUsesProjectedStage(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()
	profileRepo.profiles[1] = &domain.IntegrationProfile{
		ID:               1,
		TrackingSyncMode: "api_push",
		ClosurePolicy:    "close_after_sync",
	}

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		staticProductRepo{}, profileRepo, newMockHistoryScopeRepo(), newMockHistoryNodeRepo(), NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{}),
	)

	wave := &domain.Wave{Name: "dashboard-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(context.Background(), wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	profileID := uint(1)
	doc := &domain.DemandDocument{
		Kind:              "retail_order",
		CaptureMode:       "manual_entry",
		SourceChannel:     "test",
		SourceDocumentNo:  "DB-001",
		CustomerProfileID: &profileID,
	}
	if err := NewDemandIntakeUseCase(demandRepo).ImportDemand(context.Background(), doc, []*domain.DemandLine{{
		RoutingDisposition:  "accepted",
		RecipientInputState: "ready",
		RequestedQuantity:   1,
		LineType:            "sku_order",
	}}); err != nil {
		t.Fatalf("ImportDemand: %v", err)
	}
	if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{WaveID: wave.ID, DemandDocumentID: doc.ID}); err != nil {
		t.Fatalf("assignment: %v", err)
	}
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: wave.ID, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})
	if _, err := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil, nil).MapDemandToFulfillment(context.Background(), wave.ID); err != nil {
		t.Fatalf("MapDemandToFulfillment: %v", err)
	}

	rows, err := queryUC.ListDashboardRows(context.Background())
	if err != nil {
		t.Fatalf("ListDashboardRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ProjectedLifecycleStage != "review" {
		t.Fatalf("ProjectedLifecycleStage = %q, want review", rows[0].ProjectedLifecycleStage)
	}
}

func TestBuildClosureCandidatesDocumentlessLineUsesManualClosurePolicy(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	fulfillRepo := newMockFulfillRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	profileRepo.profiles[1] = &domain.IntegrationProfile{
		ID:               1,
		TrackingSyncMode: "document_export",
		ClosurePolicy:    "close_after_manual_confirmation",
	}

	profileID := uint(1)
	customerID := uint(10)
	doc := &domain.DemandDocument{CustomerProfileID: &customerID, IntegrationProfileID: &profileID}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create demand document: %v", err)
	}
	if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: doc.ID}); err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	line := &domain.FulfillmentLine{WaveID: 1, CustomerProfileID: &customerID, Quantity: 1}
	if err := fulfillRepo.Create(context.Background(), line); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}
	shipmentRepo.shipments[1] = &domain.Shipment{ID: 1, SupplierOrderID: 1}
	shipmentRepo.supplierOrderWave[1] = 1
	shipmentRepo.shipmentLines[1] = []*domain.ShipmentLine{{ID: 1, ShipmentID: 1, FulfillmentLineID: line.ID, Quantity: 1}}

	queryUC := &waveOverviewQueryUseCase{
		fulfillRepo:    fulfillRepo,
		assignmentRepo: assignmentRepo,
		demandRepo:     demandRepo,
		shipmentRepo:   shipmentRepo,
		profileRepo:    profileRepo,
	}
	candidates, err := queryUC.buildClosureCandidates(context.Background(), 1)
	if err != nil {
		t.Fatalf("buildClosureCandidates: %v", err)
	}
	if candidates.AutoCandidateCount != 0 || candidates.ManualCandidateCount != 1 {
		t.Fatalf("closure candidates = %+v, want 0 auto and 1 manual", candidates)
	}
}

func TestClosureCandidateClassificationFallsBackToTrackingModeForAutoPolicy(t *testing.T) {
	t.Parallel()

	profile := &domain.IntegrationProfile{
		TrackingSyncMode: "document_export",
		ClosurePolicy:    "close_after_sync",
	}
	if closureCandidateIsManual(profile) {
		t.Fatal("document_export with close_after_sync classified as manual, want auto")
	}
}

func TestGetWaveWorkspaceSnapshotIncludesRecentHistoryAndHead(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		staticProductRepo{}, profileRepo, scopeRepo, nodeRepo, NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{}),
	)

	wave := &domain.Wave{Name: "history-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(context.Background(), wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	scope, err := scopeRepo.FindOrCreate(context.Background(), "wave", "1")
	if err != nil {
		t.Fatalf("FindOrCreate scope: %v", err)
	}
	head := &domain.HistoryNode{
		HistoryScopeID: scope.ID,
		CommandKind:    "create_rule",
		CommandSummary: "create rule",
		ProjectionHash: "hash-123",
		CreatedBy:      "tester",
	}
	if err := nodeRepo.Create(context.Background(), head); err != nil {
		t.Fatalf("Create history node: %v", err)
	}
	if err := scopeRepo.UpdateHead(context.Background(), scope.ID, head.ID); err != nil {
		t.Fatalf("UpdateHead: %v", err)
	}

	snapshot, err := queryUC.GetWaveWorkspaceSnapshot(context.Background(), wave.ID)
	if err != nil {
		t.Fatalf("GetWaveWorkspaceSnapshot: %v", err)
	}
	if snapshot.HistoryHeadNodeID != head.ID {
		t.Fatalf("HistoryHeadNodeID = %d, want %d", snapshot.HistoryHeadNodeID, head.ID)
	}
	if snapshot.HistoryHeadProjectionHash != "hash-123" {
		t.Fatalf("HistoryHeadProjectionHash = %q, want %q", snapshot.HistoryHeadProjectionHash, "hash-123")
	}
	if len(snapshot.RecentHistory) != 1 {
		t.Fatalf("RecentHistory len = %d, want 1", len(snapshot.RecentHistory))
	}
	if snapshot.RecentHistory[0].ParentNodeID != 0 {
		t.Fatalf("ParentNodeID = %d, want 0", snapshot.RecentHistory[0].ParentNodeID)
	}
}

func TestListRecentHistorySkipsSystemBaseline(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		staticProductRepo{}, profileRepo, scopeRepo, nodeRepo, NewWaveOverviewProjectionUseCase(newMockChannelSyncRepo(), newMockClosureDecisionRepo(), noopDriftUC{}, noopHistoryHeadUC{}),
	)

	scope, err := scopeRepo.FindOrCreate(context.Background(), "wave", "1")
	if err != nil {
		t.Fatalf("FindOrCreate scope: %v", err)
	}
	baseline := &domain.HistoryNode{
		HistoryScopeID: scope.ID,
		CommandKind:    domain.CmdSystemBaseline,
		CommandSummary: "system baseline",
	}
	if err := nodeRepo.Create(context.Background(), baseline); err != nil {
		t.Fatalf("Create baseline: %v", err)
	}
	if err := scopeRepo.UpdateHead(context.Background(), scope.ID, baseline.ID); err != nil {
		t.Fatalf("UpdateHead baseline: %v", err)
	}
	node := &domain.HistoryNode{
		HistoryScopeID: scope.ID,
		ParentNodeID:   baseline.ID,
		CommandKind:    "create_rule",
		CommandSummary: "create rule",
	}
	if err := nodeRepo.Create(context.Background(), node); err != nil {
		t.Fatalf("Create node: %v", err)
	}
	if err := scopeRepo.UpdateHead(context.Background(), scope.ID, node.ID); err != nil {
		t.Fatalf("UpdateHead node: %v", err)
	}

	items, err := queryUC.ListRecentHistory(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListRecentHistory: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListRecentHistory len = %d, want 1", len(items))
	}
	if items[0].CommandKind != "create_rule" {
		t.Fatalf("CommandKind = %q, want create_rule", items[0].CommandKind)
	}
}

var _ dto.WaveOverviewDTO
