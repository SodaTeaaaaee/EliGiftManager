package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// noopDriftUC is a no-op BasisDriftDetectionUseCase that always returns empty signals.
// Used so existing projection tests are unaffected by drift detection.
type noopDriftUC struct{}

func (noopDriftUC) DetectWaveBasisDrift(_ uint, _ string) ([]dto.BasisDriftSignalDTO, error) {
	return nil, nil
}

// noopHistoryHeadUC is a no-op HistoryHeadQueryUseCase that always returns empty values.
type noopHistoryHeadUC struct{}

func (noopHistoryHeadUC) GetCurrentProjectionHash(_ uint) (string, error) { return "", nil }
func (noopHistoryHeadUC) GetCurrentHeadNodeIDAndHash(_ uint) (uint, string, error) {
	return 0, "", nil
}

// ── test setup ──

type projTestSetup struct {
	syncRepo    *mockChannelSyncRepo
	closureRepo *mockClosureDecisionRepo
	uc          WaveOverviewProjectionUseCase
}

func newProjTestSetup() *projTestSetup {
	sr := newMockChannelSyncRepo()
	cr := newMockClosureDecisionRepo()
	return &projTestSetup{
		syncRepo:    sr,
		closureRepo: cr,
		uc:          NewWaveOverviewProjectionUseCase(sr, cr, noopDriftUC{}, noopHistoryHeadUC{}),
	}
}

func (p *projTestSetup) addJob(waveID uint, status string) uint {
	job := &domain.ChannelSyncJob{
		WaveID:    waveID,
		Direction: "push_tracking",
		Status:    status,
	}
	if err := p.syncRepo.CreateJob(job); err != nil {
		panic("projTestSetup.addJob: " + err.Error())
	}
	return job.ID
}

func (p *projTestSetup) addItem(jobID uint, fulfillmentLineID uint, status string) {
	item := &domain.ChannelSyncItem{
		ChannelSyncJobID:  jobID,
		FulfillmentLineID: fulfillmentLineID,
		Status:            status,
	}
	if err := p.syncRepo.CreateItem(item); err != nil {
		panic("projTestSetup.addItem: " + err.Error())
	}
}

func (p *projTestSetup) addDecision(waveID uint, fulfillmentLineID uint, kind string) {
	record := &domain.ChannelClosureDecisionRecord{
		WaveID:            waveID,
		FulfillmentLineID: fulfillmentLineID,
		DecisionKind:      kind,
		OperatorID:        "op-test",
	}
	if err := p.closureRepo.Create(record); err != nil {
		panic("projTestSetup.addDecision: " + err.Error())
	}
}

func baseOverview(waveID uint, stage string) dto.WaveOverviewDTO {
	return dto.WaveOverviewDTO{
		Wave: dto.WaveDTO{
			ID:             waveID,
			LifecycleStage: stage,
		},
		DemandCount:             5,
		FulfillmentCount:        3,
		SupplierOrderCount:      2,
		ShipmentCount:           1,
		TrackedFulfillmentCount: 1,
	}
}

func baseOverviewWithCandidates(waveID uint, stage string, autoCandidates int, manualCandidates int) dto.WaveOverviewDTO {
	base := baseOverview(waveID, stage)
	base.AutoClosureCandidateCount = autoCandidates
	base.ManualClosureCandidateCount = manualCandidates
	return base
}

// ── tests ──

func TestProjectWaveOverviewNoJobsNoDecisions(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "execution"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ChannelSyncJobCount != 0 {
		t.Errorf("ChannelSyncJobCount = %d, want 0", result.ChannelSyncJobCount)
	}
	if result.ManualClosureDecisionCount != 0 {
		t.Errorf("ManualClosureDecisionCount = %d, want 0", result.ManualClosureDecisionCount)
	}
	// No sync jobs — projected stage derived from observable facts alone
	if result.ProjectedLifecycleStage != "execution" {
		t.Errorf("ProjectedLifecycleStage = %q, want %q", result.ProjectedLifecycleStage, "execution")
	}
	// Base fields must be preserved
	if result.DemandCount != 5 {
		t.Errorf("DemandCount = %d, want 5", result.DemandCount)
	}
}

func TestProjectWaveOverviewActivePendingJobsSetSyncingBack(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()
	p.addJob(1, "pending")
	p.addJob(1, "success")

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "execution"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ChannelSyncJobCount != 2 {
		t.Errorf("ChannelSyncJobCount = %d, want 2", result.ChannelSyncJobCount)
	}
	if result.ChannelSyncPendingCount != 1 {
		t.Errorf("ChannelSyncPendingCount = %d, want 1", result.ChannelSyncPendingCount)
	}
	if result.ChannelSyncSuccessCount != 1 {
		t.Errorf("ChannelSyncSuccessCount = %d, want 1", result.ChannelSyncSuccessCount)
	}
	if result.ProjectedLifecycleStage != "syncing_back" {
		t.Errorf("ProjectedLifecycleStage = %q, want syncing_back", result.ProjectedLifecycleStage)
	}
}

func TestProjectWaveOverviewDecisionsWithoutManualCompletedSetsAwaitingManualClosure(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()
	// No active jobs, but failed job with failed items exists
	p.addJob(1, "success")
	failedJobID := p.addJob(1, "failed")
	p.addItem(failedJobID, 100, "failed")
	p.addItem(failedJobID, 101, "failed")
	// No decisions yet — user hasn't acted on the failures

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "execution"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ChannelSyncFailedCount != 1 {
		t.Errorf("ChannelSyncFailedCount = %d, want 1", result.ChannelSyncFailedCount)
	}
	if result.ManualClosureDecisionCount != 0 {
		t.Errorf("ManualClosureDecisionCount = %d, want 0", result.ManualClosureDecisionCount)
	}
	if result.ProjectedLifecycleStage != "awaiting_manual_closure" {
		t.Errorf("ProjectedLifecycleStage = %q, want awaiting_manual_closure", result.ProjectedLifecycleStage)
	}
}

func TestProjectWaveOverviewPartialDecisionsCoverageStillAwaiting(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()
	// Failed job with 2 failed items
	p.addJob(1, "success")
	failedJobID := p.addJob(1, "failed")
	p.addItem(failedJobID, 100, "failed")
	p.addItem(failedJobID, 101, "failed")
	// Only 1 of 2 failed items has a decision — still awaiting
	p.addDecision(1, 100, "mark_sync_unsupported")

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "execution"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ManualClosureDecisionCount != 1 {
		t.Errorf("ManualClosureDecisionCount = %d, want 1", result.ManualClosureDecisionCount)
	}
	if result.ProjectedLifecycleStage != "awaiting_manual_closure" {
		t.Errorf("ProjectedLifecycleStage = %q, want awaiting_manual_closure", result.ProjectedLifecycleStage)
	}
}

func TestProjectWaveOverviewAllFailedItemsCoveredWithJobsSetsClosed(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()
	// Failed job with 2 failed items, both covered by decisions
	p.addJob(1, "success")
	failedJobID := p.addJob(1, "failed")
	p.addItem(failedJobID, 100, "failed")
	p.addItem(failedJobID, 101, "failed")
	p.addDecision(1, 100, "mark_sync_unsupported")
	p.addDecision(1, 101, "mark_sync_completed_manually")

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "execution"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ManualCompletedCount != 1 {
		t.Errorf("ManualCompletedCount = %d, want 1", result.ManualCompletedCount)
	}
	if result.ManualUnsupportedCount != 1 {
		t.Errorf("ManualUnsupportedCount = %d, want 1", result.ManualUnsupportedCount)
	}
	// All failed items covered + sync jobs exist → closed
	if result.ProjectedLifecycleStage != "closed" {
		t.Errorf("ProjectedLifecycleStage = %q, want closed", result.ProjectedLifecycleStage)
	}
}

func TestProjectWaveOverviewManualClosureCandidatesWithoutJobsAwaitingManualClosure(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()

	result, err := p.uc.ProjectWaveOverview(baseOverviewWithCandidates(1, "execution", 0, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ManualClosureDecisionCount != 0 {
		t.Errorf("ManualClosureDecisionCount = %d, want 0", result.ManualClosureDecisionCount)
	}
	if result.ProjectedLifecycleStage != "awaiting_manual_closure" {
		t.Errorf("ProjectedLifecycleStage = %q, want awaiting_manual_closure", result.ProjectedLifecycleStage)
	}
}

func TestProjectWaveOverviewManualClosureCandidatesCoveredWithoutJobsSetsClosed(t *testing.T) {
	t.Parallel()
	p := newProjTestSetup()
	p.addDecision(1, 100, "mark_sync_completed_manually")
	p.addDecision(1, 101, "mark_sync_skipped")

	result, err := p.uc.ProjectWaveOverview(baseOverviewWithCandidates(1, "execution", 0, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ManualClosureDecisionCount != 2 {
		t.Errorf("ManualClosureDecisionCount = %d, want 2", result.ManualClosureDecisionCount)
	}
	if result.ProjectedLifecycleStage != "closed" {
		t.Errorf("ProjectedLifecycleStage = %q, want closed", result.ProjectedLifecycleStage)
	}
}
