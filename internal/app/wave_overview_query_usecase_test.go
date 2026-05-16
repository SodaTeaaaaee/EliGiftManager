package app

import (
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type failingProductRepo struct{}

func (failingProductRepo) Create(*domain.Product) error { panic("not implemented") }
func (failingProductRepo) FindByID(uint) (*domain.Product, error) {
	panic("not implemented")
}
func (failingProductRepo) FindByWaveAndID(uint, uint) (*domain.Product, error) {
	panic("not implemented")
}
func (failingProductRepo) ListByWave(uint) ([]domain.Product, error) {
	return nil, errTest("product repo unavailable")
}
func (failingProductRepo) FindByWaveAndSKU(uint, string, string) (*domain.Product, error) {
	panic("not implemented")
}
func (failingProductRepo) DeleteByWave(uint) error { panic("not implemented") }

type staticProductRepo struct {
	products []domain.Product
}

func (s staticProductRepo) Create(*domain.Product) error { panic("not implemented") }
func (s staticProductRepo) FindByID(uint) (*domain.Product, error) {
	panic("not implemented")
}
func (s staticProductRepo) FindByWaveAndID(uint, uint) (*domain.Product, error) {
	panic("not implemented")
}
func (s staticProductRepo) ListByWave(uint) ([]domain.Product, error) { return s.products, nil }
func (s staticProductRepo) FindByWaveAndSKU(uint, string, string) (*domain.Product, error) {
	panic("not implemented")
}
func (s staticProductRepo) DeleteByWave(uint) error { panic("not implemented") }

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
		failingProductRepo{}, profileRepo, NewWaveOverviewProjectionUseCase(newMockChannelSyncRepo(), newMockClosureDecisionRepo(), noopDriftUC{}, noopHistoryHeadUC{}),
	)

	wave := &domain.Wave{Name: "overview-error"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	_, err := queryUC.BuildBaseOverview(wave.ID)
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
		staticProductRepo{}, profileRepo, NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{}),
	)

	wave := &domain.Wave{Name: "dashboard-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(wave); err != nil {
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
	if err := NewDemandIntakeUseCase(demandRepo).ImportDemand(doc, []*domain.DemandLine{{
		RoutingDisposition:  "accepted",
		RecipientInputState: "ready",
		RequestedQuantity:   1,
		LineType:            "sku_order",
	}}); err != nil {
		t.Fatalf("ImportDemand: %v", err)
	}
	if err := assignmentRepo.Create(&domain.WaveDemandAssignment{WaveID: wave.ID, DemandDocumentID: doc.ID}); err != nil {
		t.Fatalf("assignment: %v", err)
	}
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: wave.ID, CustomerProfileID: profileID, SnapshotType: "buyer"},
	})
	if _, err := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil).MapDemandToFulfillment(wave.ID); err != nil {
		t.Fatalf("MapDemandToFulfillment: %v", err)
	}

	rows, err := queryUC.ListDashboardRows()
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

var _ dto.WaveOverviewDTO
