package app

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type waveOverviewProjectionUseCase struct {
	channelSyncRepo domain.ChannelSyncRepository
	closureRepo     domain.ChannelClosureDecisionRepository
	driftUC         BasisDriftDetectionUseCase
}

func NewWaveOverviewProjectionUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	closureRepo domain.ChannelClosureDecisionRepository,
	driftUC BasisDriftDetectionUseCase,
) WaveOverviewProjectionUseCase {
	return &waveOverviewProjectionUseCase{
		channelSyncRepo: channelSyncRepo,
		closureRepo:     closureRepo,
		driftUC:         driftUC,
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

	activeCount := pendingCount + runningCount + partialSuccessCount
	projectedStage := base.Wave.LifecycleStage
	if activeCount > 0 {
		projectedStage = "syncing_back"
	} else if failedCount > 0 {
		hasUncoveredFailures, err := uc.hasUncoveredFailedItems(jobs, decisions)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		if hasUncoveredFailures {
			projectedStage = "awaiting_manual_closure"
		}
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
	// Phase 9: replace "" with computed projection hash from HistoryNode
	signals, err := uc.driftUC.DetectWaveBasisDrift(waveID, "")
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	base.BasisDriftSignals = signals
	for _, s := range signals {
		if s.BasisDriftStatus == "drifted" {
			base.HasDriftedBasis = true
		}
		// Phase 9: "required" level needs adjustment target mismatch detection
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
