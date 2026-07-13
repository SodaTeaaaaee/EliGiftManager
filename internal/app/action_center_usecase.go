package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ActionCenterUseCase aggregates the cross-wave action-center summary consumed by
// the Home page and the nav badges: per-wave blocked-bucket counts (with deep-link
// filter params), the inbox pending-intake count, and per-nav-item badge counts.
// Read-only aggregation — it issues no writes.
type ActionCenterUseCase interface {
	GetActionCenterSummary(ctx context.Context) (dto.ActionCenterSummaryDTO, error)
}

type actionCenterUseCase struct {
	waveRepo   domain.WaveRepository
	demandRepo domain.DemandDocumentRepository
	overviewUC WaveOverviewQueryUseCase
}

// NewActionCenterUseCase constructs an ActionCenterUseCase. overviewUC supplies the
// per-wave state counts (address/mapping/channel-sync/closure/drift), which it in
// turn assembles via WaveOverviewProjectionUseCase → BasisDriftDetectionUseCase — so
// drift counts here are always the real basis-comparison signal, never a heuristic.
func NewActionCenterUseCase(
	waveRepo domain.WaveRepository,
	demandRepo domain.DemandDocumentRepository,
	overviewUC WaveOverviewQueryUseCase,
) ActionCenterUseCase {
	return &actionCenterUseCase{
		waveRepo:   waveRepo,
		demandRepo: demandRepo,
		overviewUC: overviewUC,
	}
}

func (uc *actionCenterUseCase) GetActionCenterSummary(ctx context.Context) (dto.ActionCenterSummaryDTO, error) {
	waves, err := uc.waveRepo.List(ctx)
	if err != nil {
		return dto.ActionCenterSummaryDTO{}, err
	}

	waveSummaries := make([]dto.ActionCenterWaveSummaryDTO, 0, len(waves))
	totalBlockedCount := 0
	wavesNeedingAttention := 0

	for _, w := range waves {
		overview, err := uc.overviewUC.GetWaveOverview(ctx, w.ID)
		if err != nil {
			return dto.ActionCenterSummaryDTO{}, err
		}
		// Closed waves are done — nothing actionable, so they never produce cards.
		if overview.ProjectedLifecycleStage == "closed" {
			continue
		}

		buckets := buildActionCenterBuckets(overview)
		if len(buckets) == 0 {
			continue
		}

		waveTotal := 0
		for _, b := range buckets {
			waveTotal += b.Count
		}
		totalBlockedCount += waveTotal
		wavesNeedingAttention++

		waveSummaries = append(waveSummaries, dto.ActionCenterWaveSummaryDTO{
			WaveID:            w.ID,
			WaveNo:            w.WaveNo,
			WaveName:          w.Name,
			Buckets:           buckets,
			TotalBlockedCount: waveTotal,
		})
	}

	inboxPendingCount, err := uc.countInboxPendingIntake(ctx)
	if err != nil {
		return dto.ActionCenterSummaryDTO{}, err
	}

	navBadges := []dto.ActionCenterNavBadgeDTO{
		{NavKey: "home", Count: totalBlockedCount + inboxPendingCount},
		{NavKey: "waves", Count: wavesNeedingAttention},
		{NavKey: "inbox", Count: inboxPendingCount},
		// No backend signal defined yet for these nav items — report 0 honestly
		// rather than fabricate a heuristic count. Future units may extend this.
		{NavKey: "customers", Count: 0},
		{NavKey: "products", Count: 0},
		{NavKey: "integrations", Count: 0},
	}

	return dto.ActionCenterSummaryDTO{
		Waves:                   waveSummaries,
		InboxPendingIntakeCount: inboxPendingCount,
		NavBadges:               navBadges,
	}, nil
}

// countInboxPendingIntake counts DemandDocuments that have at least one DemandLine
// still sitting at RoutingDisposition "pending_intake" (imported, not yet triaged).
// A document with even one pending line is "待分诊" — this mirrors the disposition
// bucketing entitlementRoutingUseCase.GetWaveRoutingStats uses, just scoped globally
// across all documents (assigned or not) instead of a single wave.
func (uc *actionCenterUseCase) countInboxPendingIntake(ctx context.Context) (int, error) {
	docs, err := uc.demandRepo.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, doc := range docs {
		lines, err := uc.demandRepo.ListLinesByDocument(ctx, doc.ID)
		if err != nil {
			return 0, err
		}
		for _, line := range lines {
			if line.RoutingDisposition == string(domain.RoutingDispositionPendingIntake) {
				count++
				break
			}
		}
	}
	return count, nil
}

// buildActionCenterBuckets derives the six blocked buckets for a single wave from its
// already-projected WaveOverviewDTO. Only non-zero buckets are returned.
func buildActionCenterBuckets(overview dto.WaveOverviewDTO) []dto.ActionCenterWaveBucketDTO {
	waveID := overview.Wave.ID
	buckets := make([]dto.ActionCenterWaveBucketDTO, 0, 6)

	if overview.AddressMissingCount > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "missing_address",
			Count:      overview.AddressMissingCount,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "lines", AddressState: "missing"},
		})
	}
	if overview.AcceptedWaitingForInput > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "waiting_input",
			Count:      overview.AcceptedWaitingForInput,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "intake"},
		})
	}
	if overview.MappingBlockedCount > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "mapping_blocked",
			Count:      overview.MappingBlockedCount,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "allocation"},
		})
	}
	if overview.ChannelSyncFailedCount > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "channel_sync_failed",
			Count:      overview.ChannelSyncFailedCount,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "closure", ChannelSyncState: "failed"},
		})
	}
	if awaiting := awaitingManualClosureCount(overview); awaiting > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "awaiting_manual_closure",
			Count:      awaiting,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "closure"},
		})
	}
	if driftCount := driftNeedsReviewCount(overview); driftCount > 0 {
		buckets = append(buckets, dto.ActionCenterWaveBucketDTO{
			WaveID:     waveID,
			BucketKind: "drift_needs_review",
			Count:      driftCount,
			Filter:     dto.ActionCenterBucketFilterDTO{StepKey: "wave_overview", Drift: "drifted"},
		})
	}
	return buckets
}

// awaitingManualClosureCount counts manual-closure candidate fulfillment lines that
// have not yet received a closure decision. Both inputs come from the real overview
// aggregation (buildClosureCandidates / ProjectWaveOverview in
// wave_overview_query_usecase.go and wave_overview_projection_usecase.go) — not a
// heuristic.
func awaitingManualClosureCount(overview dto.WaveOverviewDTO) int {
	remaining := overview.ManualClosureCandidateCount - overview.ManualClosureDecisionCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// driftNeedsReviewCount counts basis drift signals whose ReviewRequirement is
// "recommended" or "required". overview.BasisDriftSignals is populated by the real
// BasisDriftDetectionUseCase (internal/app/basis_drift_usecase.go), threaded through
// WaveOverviewProjectionUseCase.ProjectWaveOverview — never a heuristic. This is the
// signal plan 5.1 says must replace the old heuristic drift count.
func driftNeedsReviewCount(overview dto.WaveOverviewDTO) int {
	count := 0
	for _, s := range overview.BasisDriftSignals {
		if s.ReviewRequirement == "recommended" || s.ReviewRequirement == "required" {
			count++
		}
	}
	return count
}
