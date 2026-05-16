package app

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type waveOverviewProjectionUseCase struct {
	channelSyncRepo domain.ChannelSyncRepository
	closureRepo     domain.ChannelClosureDecisionRepository
	driftUC         BasisDriftDetectionUseCase
	historyHeadUC   HistoryHeadQueryUseCase
}

func NewWaveOverviewProjectionUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	closureRepo domain.ChannelClosureDecisionRepository,
	driftUC BasisDriftDetectionUseCase,
	historyHeadUC HistoryHeadQueryUseCase,
) WaveOverviewProjectionUseCase {
	return &waveOverviewProjectionUseCase{
		channelSyncRepo: channelSyncRepo,
		closureRepo:     closureRepo,
		driftUC:         driftUC,
		historyHeadUC:   historyHeadUC,
	}
}

func (uc *waveOverviewProjectionUseCase) ProjectWaveOverview(base dto.WaveOverviewDTO) (dto.WaveOverviewDTO, error) {
	waveID := base.Wave.ID

	jobs, err := uc.channelSyncRepo.ListJobsByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	var pendingCount, runningCount, successCount, partialSuccessCount, failedCount int
	for _, j := range jobs {
		switch j.Status {
		case "pending":
			pendingCount++
		case "running":
			runningCount++
		case "success":
			successCount++
		case "partial_success":
			partialSuccessCount++
		case "failed":
			failedCount++
		}
	}

	decisions, err := uc.closureRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	var unsupportedCount, skippedCount, completedManuallyCount int
	for _, d := range decisions {
		switch d.DecisionKind {
		case "mark_sync_unsupported":
			unsupportedCount++
		case "mark_sync_skipped":
			skippedCount++
		case "mark_sync_completed_manually":
			completedManuallyCount++
		}
	}

	// Derive projected lifecycle stage from observable state.
	// This is the single authoritative stage aggregation point.
	projectedStage := deriveStage(base)

	hasUncoveredFailures, err := uc.hasUncoveredFailedItems(jobs, decisions)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	activeCount := pendingCount + runningCount + partialSuccessCount
	if activeCount > 0 {
		projectedStage = "syncing_back"
	} else if hasUncoveredFailures {
		projectedStage = "awaiting_manual_closure"
	} else if len(jobs) > 0 {
		// No active jobs, no uncovered failures, sync jobs exist → closed.
		projectedStage = "closed"
	}

	base.ChannelSyncJobCount = len(jobs)
	base.ChannelSyncPendingCount = pendingCount
	base.ChannelSyncRunningCount = runningCount
	base.ChannelSyncSuccessCount = successCount
	base.ChannelSyncPartialSuccessCount = partialSuccessCount
	base.ChannelSyncFailedCount = failedCount
	base.ManualClosureDecisionCount = len(decisions)
	base.ManualUnsupportedCount = unsupportedCount
	base.ManualSkippedCount = skippedCount
	base.ManualCompletedCount = completedManuallyCount
	base.ProjectedLifecycleStage = projectedStage

	// Basis drift detection
	projHash, err := uc.historyHeadUC.GetCurrentProjectionHash(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	signals, err := uc.driftUC.DetectWaveBasisDrift(waveID, projHash)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	base.BasisDriftSignals = signals
	for _, s := range signals {
		if s.BasisDriftStatus == "drifted" {
			base.HasDriftedBasis = true
		}
		if s.ReviewRequirement == "required" {
			base.HasRequiredReviewBasis = true
		}
	}

	return base, nil
}

func (uc *waveOverviewProjectionUseCase) hasUncoveredFailedItems(
	jobs []domain.ChannelSyncJob,
	decisions []domain.ChannelClosureDecisionRecord,
) (bool, error) {
	coveredLines := make(map[uint]struct{}, len(decisions))
	for _, d := range decisions {
		coveredLines[d.FulfillmentLineID] = struct{}{}
	}

	for _, j := range jobs {
		if j.Status != "failed" && j.Status != "partial_success" {
			continue
		}
		items, err := uc.channelSyncRepo.ListItemsByJob(j.ID)
		if err != nil {
			return false, err
		}
		for _, item := range items {
			if item.Status == "failed" {
				if _, covered := coveredLines[item.FulfillmentLineID]; !covered {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// deriveStage computes the authoritative lifecycle stage from observable state.
// This is the single source of truth for the projected stage — frontend pages
// must not independently derive stage labels.
func deriveStage(base dto.WaveOverviewDTO) string {
	if base.DemandCount == 0 {
		return "intake"
	}
	if base.FulfillmentCount == 0 {
		return "allocation"
	}
	if base.SupplierOrderCount == 0 {
		return "review"
	}
	if base.ShipmentCount == 0 {
		return "execution"
	}
	// Sync-level stages (syncing_back, awaiting_manual_closure, closed) are
	// overlaid later by ProjectWaveOverview based on channel sync state.
	// Fall back to execution: shipments exist, so the wave is in or past execution.
	return "execution"
}
