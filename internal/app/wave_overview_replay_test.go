package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TestWaveOverviewReplayFailuresDetected verifies that when a wave has an
// adjustment targeting a non-existent fulfillment line, GetWaveOverview
// surfaces ReplayHealthy=false and a non-zero ReplayFailureCount, and that
// "replay_failures_detected" appears in BlockingIssues.
func TestWaveOverviewReplayFailuresDetected(t *testing.T) {
	t.Parallel()

	waveID := uint(1)

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	adjRepo := newMockAdjustmentRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	// Create the wave.
	wave := &domain.Wave{Name: "replay-failure-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	waveID = wave.ID

	// Create one real fulfillment line so the wave has FulfillmentCount > 0.
	if err := fulfillRepo.Create(&domain.FulfillmentLine{
		WaveID:          waveID,
		Quantity:        1,
		AllocationState: "ready",
	}); err != nil {
		t.Fatalf("Create fulfillment line: %v", err)
	}

	// Create an adjustment that targets a non-existent fulfillment line ID (999).
	// This will produce an orphaned_line failure during replay.
	nonExistentLineID := uint(999)
	adj := &domain.FulfillmentAdjustment{
		WaveID:            waveID,
		TargetKind:        "fulfillment_line",
		FulfillmentLineID: &nonExistentLineID,
		AdjustmentKind:    "add",
		QuantityDelta:     1,
	}
	if err := adjRepo.Create(adj); err != nil {
		t.Fatalf("Create adjustment: %v", err)
	}

	// Wire projection use case with fulfillRepo and adjRepo so replay runs.
	projUC := NewWaveOverviewProjectionUseCase(
		syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{},
		fulfillRepo, adjRepo,
	)

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo,
		shipmentRepo, staticProductRepo{}, profileRepo,
		newMockHistoryScopeRepo(), newMockHistoryNodeRepo(),
		projUC,
		adjRepo,
	)

	overview, err := queryUC.GetWaveOverview(waveID)
	if err != nil {
		t.Fatalf("GetWaveOverview: %v", err)
	}

	if overview.ReplayHealthy {
		t.Error("ReplayHealthy = true, want false (orphaned adjustment present)")
	}
	if overview.ReplayFailureCount != 1 {
		t.Errorf("ReplayFailureCount = %d, want 1", overview.ReplayFailureCount)
	}

	foundIssue := false
	for _, issue := range overview.BlockingIssues {
		if issue == "replay_failures_detected" {
			foundIssue = true
			break
		}
	}
	if !foundIssue {
		t.Errorf("BlockingIssues = %v, want to contain \"replay_failures_detected\"", overview.BlockingIssues)
	}
}

// TestWaveOverviewReplayHealthyWhenNoFailures verifies that a wave with
// adjustments that all resolve cleanly reports ReplayHealthy=true.
func TestWaveOverviewReplayHealthyWhenNoFailures(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	adjRepo := newMockAdjustmentRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	wave := &domain.Wave{Name: "replay-healthy-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	waveID := wave.ID

	// Create a real fulfillment line.
	fl := &domain.FulfillmentLine{
		WaveID:          waveID,
		Quantity:        5,
		AllocationState: "ready",
	}
	if err := fulfillRepo.Create(fl); err != nil {
		t.Fatalf("Create fulfillment line: %v", err)
	}

	// Create an adjustment targeting the real line — replay should succeed.
	adj := &domain.FulfillmentAdjustment{
		WaveID:            waveID,
		TargetKind:        "fulfillment_line",
		FulfillmentLineID: &fl.ID,
		AdjustmentKind:    "add",
		QuantityDelta:     2,
	}
	if err := adjRepo.Create(adj); err != nil {
		t.Fatalf("Create adjustment: %v", err)
	}

	projUC := NewWaveOverviewProjectionUseCase(
		syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{},
		fulfillRepo, adjRepo,
	)

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo,
		shipmentRepo, staticProductRepo{}, profileRepo,
		newMockHistoryScopeRepo(), newMockHistoryNodeRepo(),
		projUC,
		adjRepo,
	)

	overview, err := queryUC.GetWaveOverview(waveID)
	if err != nil {
		t.Fatalf("GetWaveOverview: %v", err)
	}

	if !overview.ReplayHealthy {
		t.Error("ReplayHealthy = false, want true (all adjustments resolve)")
	}
	if overview.ReplayFailureCount != 0 {
		t.Errorf("ReplayFailureCount = %d, want 0", overview.ReplayFailureCount)
	}

	for _, issue := range overview.BlockingIssues {
		if issue == "replay_failures_detected" {
			t.Errorf("BlockingIssues contains \"replay_failures_detected\" unexpectedly: %v", overview.BlockingIssues)
			break
		}
	}
}

// TestWaveOverviewReplayHealthyWhenNoAdjustments verifies that a wave with
// no adjustments at all reports ReplayHealthy=true (zero failures).
func TestWaveOverviewReplayHealthyWhenNoAdjustments(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	adjRepo := newMockAdjustmentRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	wave := &domain.Wave{Name: "replay-no-adj-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	projUC := NewWaveOverviewProjectionUseCase(
		syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{},
		fulfillRepo, adjRepo,
	)

	queryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo,
		shipmentRepo, staticProductRepo{}, profileRepo,
		newMockHistoryScopeRepo(), newMockHistoryNodeRepo(),
		projUC,
		adjRepo,
	)

	overview, err := queryUC.GetWaveOverview(wave.ID)
	if err != nil {
		t.Fatalf("GetWaveOverview: %v", err)
	}

	if !overview.ReplayHealthy {
		t.Error("ReplayHealthy = false, want true (no adjustments)")
	}
	if overview.ReplayFailureCount != 0 {
		t.Errorf("ReplayFailureCount = %d, want 0", overview.ReplayFailureCount)
	}
}
