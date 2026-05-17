package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type waveOverviewQueryUseCase struct {
	waveRepo         domain.WaveRepository
	fulfillRepo      domain.FulfillmentLineRepository
	supplierRepo     domain.SupplierOrderRepository
	assignmentRepo   domain.WaveDemandAssignmentRepository
	demandRepo       domain.DemandDocumentRepository
	shipmentRepo     domain.ShipmentRepository
	productRepo      domain.ProductRepository
	profileRepo      domain.IntegrationProfileRepository
	historyScopeRepo domain.HistoryScopeRepository
	historyNodeRepo  domain.HistoryNodeRepository
	adjustmentRepo   domain.FulfillmentAdjustmentRepository
	overviewProjUC   WaveOverviewProjectionUseCase
}

type overviewClosureCandidates struct {
	AutoCandidateCount   int
	ManualCandidateCount int
}

func NewWaveOverviewQueryUseCase(
	waveRepo domain.WaveRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	supplierRepo domain.SupplierOrderRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
	demandRepo domain.DemandDocumentRepository,
	shipmentRepo domain.ShipmentRepository,
	productRepo domain.ProductRepository,
	profileRepo domain.IntegrationProfileRepository,
	historyScopeRepo domain.HistoryScopeRepository,
	historyNodeRepo domain.HistoryNodeRepository,
	overviewProjUC WaveOverviewProjectionUseCase,
	adjustmentRepo ...domain.FulfillmentAdjustmentRepository,
) WaveOverviewQueryUseCase {
	var adjRepo domain.FulfillmentAdjustmentRepository
	if len(adjustmentRepo) > 0 {
		adjRepo = adjustmentRepo[0]
	}
	return &waveOverviewQueryUseCase{
		waveRepo:         waveRepo,
		fulfillRepo:      fulfillRepo,
		supplierRepo:     supplierRepo,
		assignmentRepo:   assignmentRepo,
		demandRepo:       demandRepo,
		shipmentRepo:     shipmentRepo,
		productRepo:      productRepo,
		profileRepo:      profileRepo,
		historyScopeRepo: historyScopeRepo,
		historyNodeRepo:  historyNodeRepo,
		adjustmentRepo:   adjRepo,
		overviewProjUC:   overviewProjUC,
	}
}

func (uc *waveOverviewQueryUseCase) BuildBaseOverview(waveID uint) (dto.WaveOverviewDTO, error) {
	w, err := uc.waveRepo.FindByID(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	supplierOrders, err := uc.supplierRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	waveProducts, err := uc.productRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	productMasterToWaveProduct := make(map[uint]uint, len(waveProducts))
	for _, wp := range waveProducts {
		if wp.ProductMasterID != nil {
			productMasterToWaveProduct[*wp.ProductMasterID] = wp.ID
		}
	}

	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	var (
		demandCount             int
		acceptedReadyOrNotReq   int
		acceptedWaitingForInput int
		deferredCount           int
		excludedManualCount     int
		excludedDuplicateCount  int
		excludedRevokedCount    int
		mappingBlockedCount     int
	)
	for _, doc := range docs {
		lines, err := uc.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, line := range lines {
			switch line.RoutingDisposition {
			case "accepted":
				demandCount++
				if line.RecipientInputState == "ready" || line.RecipientInputState == "not_required" {
					acceptedReadyOrNotReq++
					if doc.Kind == "retail_order" && line.ProductMasterID != nil {
						if _, ok := productMasterToWaveProduct[*line.ProductMasterID]; !ok {
							mappingBlockedCount++
						}
					}
				} else if line.RecipientInputState == "waiting_for_input" || line.RecipientInputState == "partially_collected" {
					acceptedWaitingForInput++
				}
			case "deferred":
				deferredCount++
			case "excluded_manual":
				excludedManualCount++
			case "excluded_duplicate":
				excludedDuplicateCount++
			case "excluded_revoked":
				excludedRevokedCount++
			}
		}
	}

	shipments, err := uc.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	shipmentCount := len(shipments)

	trackedFulfillmentCount := 0
	trackedSet := make(map[uint]bool)
	for _, s := range shipments {
		if s.TrackingNo == "" {
			continue
		}
		lines, err := uc.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, l := range lines {
			if !trackedSet[l.FulfillmentLineID] {
				trackedSet[l.FulfillmentLineID] = true
				trackedFulfillmentCount++
			}
		}
	}

	// Fulfillment state breakdown — single pass over already-fetched lines
	var (
		fulfillDraftCount        int
		fulfillReadyCount        int
		addressMissingCount      int
		addressReadyCount        int
		addressInvalidCount      int
		supplierNotSubmittedCount int
		supplierSubmittedCount   int
		supplierShippedCount     int
	)
	for _, fl := range fulfillLines {
		switch fl.AllocationState {
		case "draft":
			fulfillDraftCount++
		case "ready":
			fulfillReadyCount++
		}
		switch fl.AddressState {
		case "missing":
			addressMissingCount++
		case "ready":
			addressReadyCount++
		case "invalid":
			addressInvalidCount++
		}
		switch fl.SupplierState {
		case "not_submitted":
			supplierNotSubmittedCount++
		case "submitted", "accepted", "producing", "partially_shipped":
			supplierSubmittedCount++
		case "shipped":
			supplierShippedCount++
		}
	}

	// Adjustment summary
	var (
		adjustmentCount        int
		adjustmentAddCount     int
		adjustmentReduceCount  int
		adjustmentReplaceCount int
		adjustmentRemoveCount  int
	)
	if uc.adjustmentRepo != nil {
		adjustments, adjErr := uc.adjustmentRepo.ListByWave(waveID)
		if adjErr != nil {
			return dto.WaveOverviewDTO{}, adjErr
		}
		adjustmentCount = len(adjustments)
		for _, adj := range adjustments {
			switch adj.AdjustmentKind {
			case "add":
				adjustmentAddCount++
			case "reduce":
				adjustmentReduceCount++
			case "replace":
				adjustmentReplaceCount++
			case "remove":
				adjustmentRemoveCount++
			}
		}
	}

	// Next-step guidance
	suggestedNextStep, nextStepReason := buildNextStepGuidance(
		demandCount, len(fulfillLines), len(supplierOrders), shipmentCount, 0,
	)

	// Blocking issues
	blockingIssues := buildBlockingIssues(addressMissingCount, false, false, mappingBlockedCount, true)

	return dto.WaveOverviewDTO{
		Wave:                       toWaveDTO(w),
		DemandCount:                demandCount,
		FulfillmentCount:           len(fulfillLines),
		SupplierOrderCount:         len(supplierOrders),
		ShipmentCount:              shipmentCount,
		TrackedFulfillmentCount:    trackedFulfillmentCount,
		AcceptedReadyOrNotRequired: acceptedReadyOrNotReq,
		AcceptedWaitingForInput:    acceptedWaitingForInput,
		DeferredCount:              deferredCount,
		ExcludedManualCount:        excludedManualCount,
		ExcludedDuplicateCount:     excludedDuplicateCount,
		ExcludedRevokedCount:       excludedRevokedCount,
		MappingBlockedCount:        mappingBlockedCount,
		// Fulfillment breakdown
		FulfillmentDraftCount:     fulfillDraftCount,
		FulfillmentReadyCount:     fulfillReadyCount,
		AddressMissingCount:       addressMissingCount,
		AddressReadyCount:         addressReadyCount,
		AddressInvalidCount:       addressInvalidCount,
		SupplierNotSubmittedCount: supplierNotSubmittedCount,
		SupplierSubmittedCount:    supplierSubmittedCount,
		SupplierShippedCount:      supplierShippedCount,
		// Adjustment summary
		AdjustmentCount:        adjustmentCount,
		AdjustmentAddCount:     adjustmentAddCount,
		AdjustmentReduceCount:  adjustmentReduceCount,
		AdjustmentReplaceCount: adjustmentReplaceCount,
		AdjustmentRemoveCount:  adjustmentRemoveCount,
		// Next-step guidance
		SuggestedNextStep: suggestedNextStep,
		NextStepReason:    nextStepReason,
		BlockingIssues:    blockingIssues,
	}, nil
}

func (uc *waveOverviewQueryUseCase) GetWaveOverview(waveID uint) (dto.WaveOverviewDTO, error) {
	base, err := uc.BuildBaseOverview(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	candidates, err := uc.buildClosureCandidates(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	base.AutoClosureCandidateCount = candidates.AutoCandidateCount
	base.ManualClosureCandidateCount = candidates.ManualCandidateCount
	projected, err := uc.overviewProjUC.ProjectWaveOverview(base)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	// Re-compute next-step guidance and blocking issues now that projection has
	// filled in HasDriftedBasis, HasRequiredReviewBasis and ChannelSyncPendingCount.
	projected.SuggestedNextStep, projected.NextStepReason = buildNextStepGuidance(
		projected.DemandCount, projected.FulfillmentCount, projected.SupplierOrderCount,
		projected.ShipmentCount, projected.ChannelSyncPendingCount,
	)
	projected.BlockingIssues = buildBlockingIssues(
		projected.AddressMissingCount,
		projected.HasDriftedBasis,
		projected.HasRequiredReviewBasis,
		projected.MappingBlockedCount,
		projected.ReplayHealthy,
	)
	return projected, nil
}

func (uc *waveOverviewQueryUseCase) GetWaveWorkspaceSnapshot(waveID uint) (dto.WaveWorkspaceSnapshotDTO, error) {
	overview, err := uc.GetWaveOverview(waveID)
	if err != nil {
		return dto.WaveWorkspaceSnapshotDTO{}, err
	}

	fulfillmentRows, err := uc.ListWaveFulfillmentRows(waveID)
	if err != nil {
		return dto.WaveWorkspaceSnapshotDTO{}, err
	}

	recentHistory, err := uc.ListRecentHistory(waveID, 10)
	if err != nil {
		return dto.WaveWorkspaceSnapshotDTO{}, err
	}
	historyHeadNodeID, historyHeadProjectionHash, err := uc.resolveHistoryHead(waveID)
	if err != nil {
		return dto.WaveWorkspaceSnapshotDTO{}, err
	}
	stepStates := buildWorkspaceStepStates(overview, fulfillmentRows)
	guidance := buildWorkspaceGuidance(overview)
	basisSummary := dto.WaveWorkspaceBasisSummaryDTO{
		HasDriftedBasis:   overview.HasDriftedBasis,
		HasRequiredReview: overview.HasRequiredReviewBasis,
	}
	for _, signal := range overview.BasisDriftSignals {
		if signal.BasisDriftStatus == "drifted" {
			basisSummary.DriftedCount++
		}
		if signal.ReviewRequirement == "required" {
			basisSummary.RequiredReviewCount++
		}
	}

	return dto.WaveWorkspaceSnapshotDTO{
		Wave:                      overview.Wave,
		ProjectedLifecycleStage:   overview.ProjectedLifecycleStage,
		Overview:                  overview,
		StepStates:                stepStates,
		Guidance:                  guidance,
		BasisSummary:              basisSummary,
		HistoryHeadNodeID:         historyHeadNodeID,
		HistoryHeadProjectionHash: historyHeadProjectionHash,
		RecentHistory:             recentHistory,
	}, nil
}

func (uc *waveOverviewQueryUseCase) ListRecentHistory(waveID uint, limit int) ([]dto.HistoryNodeDTO, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	if uc.historyScopeRepo == nil || uc.historyNodeRepo == nil {
		return []dto.HistoryNodeDTO{}, nil
	}

	scope, err := uc.historyScopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return nil, err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return []dto.HistoryNodeDTO{}, nil
	}

	nodes, err := uc.historyNodeRepo.ListByScopeRecent(scope.ID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]dto.HistoryNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		if n.CommandKind == domain.CmdSystemBaseline {
			continue
		}
		result = append(result, dto.HistoryNodeDTO{
			ID:                   n.ID,
			ParentNodeID:         n.ParentNodeID,
			PreferredRedoChildID: n.PreferredRedoChildID,
			CommandKind:          n.CommandKind,
			CommandSummary:       n.CommandSummary,
			ProjectionHash:       n.ProjectionHash,
			CheckpointHint:       n.CheckpointHint,
			CreatedAt:            n.CreatedAt,
			CreatedBy:            n.CreatedBy,
		})
	}
	return result, nil
}

func (uc *waveOverviewQueryUseCase) resolveHistoryHead(waveID uint) (uint, string, error) {
	if uc.historyScopeRepo == nil || uc.historyNodeRepo == nil {
		return 0, "", nil
	}

	scope, err := uc.historyScopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return 0, "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return 0, "", nil
	}

	node, err := uc.historyNodeRepo.FindByID(scope.CurrentHeadNodeID)
	if err != nil {
		return 0, "", err
	}
	if node == nil {
		return scope.CurrentHeadNodeID, "", nil
	}
	return node.ID, node.ProjectionHash, nil
}

func (uc *waveOverviewQueryUseCase) ListDashboardRows() ([]dto.WaveDashboardRowDTO, error) {
	waves, err := uc.waveRepo.List()
	if err != nil {
		return nil, err
	}
	rows := make([]dto.WaveDashboardRowDTO, 0, len(waves))
	for _, w := range waves {
		projected, err := uc.GetWaveOverview(w.ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, dto.WaveDashboardRowDTO{
			ID:                      w.ID,
			WaveNo:                  w.WaveNo,
			Name:                    w.Name,
			CreatedAt:               w.CreatedAt,
			ProjectedLifecycleStage: projected.ProjectedLifecycleStage,
		})
	}
	return rows, nil
}

func (uc *waveOverviewQueryUseCase) ListWaveFulfillmentRows(waveID uint) ([]dto.WaveFulfillmentRowDTO, error) {
	lines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	participants, err := uc.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return nil, err
	}
	waveProducts, err := uc.productRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	overview, err := uc.GetWaveOverview(waveID)
	if err != nil {
		return nil, err
	}

	participantMap := make(map[uint]domain.WaveParticipantSnapshot, len(participants))
	for _, p := range participants {
		participantMap[p.ID] = p
	}
	productMap := make(map[uint]domain.Product, len(waveProducts))
	for _, p := range waveProducts {
		productMap[p.ID] = p
	}

	reviewRequirement := "none"
	reviewReasonSummary := ""
	if overview.HasRequiredReviewBasis {
		reviewRequirement = "required"
		reviewReasonSummary = "basis review required"
	} else if overview.HasDriftedBasis {
		reviewRequirement = "recommended"
		reviewReasonSummary = "basis drift detected"
	}

	rows := make([]dto.WaveFulfillmentRowDTO, 0, len(lines))
	for _, line := range lines {
		row := dto.WaveFulfillmentRowDTO{
			FulfillmentLineID:         line.ID,
			WaveID:                    line.WaveID,
			WaveParticipantSnapshotID: line.WaveParticipantSnapshotID,
			CustomerProfileID:         line.CustomerProfileID,
			ProductID:                 line.ProductID,
			DemandDocumentID:          line.DemandDocumentID,
			DemandLineID:              line.DemandLineID,
			Quantity:                  line.Quantity,
			AllocationState:           line.AllocationState,
			AddressState:              line.AddressState,
			SupplierState:             line.SupplierState,
			ChannelSyncState:          line.ChannelSyncState,
			LineReason:                line.LineReason,
			GeneratedBy:               line.GeneratedBy,
			BasisDriftStatus:          basisDriftStatusFromOverview(overview),
			ReviewRequirement:         reviewRequirement,
			ReviewReasonSummary:       reviewReasonSummary,
		}
		if line.WaveParticipantSnapshotID != nil {
			if p, ok := participantMap[*line.WaveParticipantSnapshotID]; ok {
				row.ParticipantType = p.SnapshotType
				row.ParticipantDisplay = participantDisplay(p)
				row.ParticipantBadge = p.GiftLevel
			}
		}
		if line.ProductID != nil {
			if p, ok := productMap[*line.ProductID]; ok {
				row.ProductDisplay = fmt.Sprintf("%s (%s)", p.Name, p.FactorySKU)
			}
		}
		if line.DemandDocumentID != nil {
			doc, docErr := uc.demandRepo.FindByID(*line.DemandDocumentID)
			if docErr == nil && doc != nil {
				row.DemandKind = doc.Kind
				row.DemandSourceSummary = fmt.Sprintf("%s · %s", doc.SourceChannel, doc.SourceDocumentNo)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (uc *waveOverviewQueryUseCase) ListWaveParticipantRows(waveID uint) ([]dto.WaveParticipantRowDTO, error) {
	participants, err := uc.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return nil, err
	}
	lines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	lineCounts := make(map[uint]int)
	readyCounts := make(map[uint]int)
	for _, line := range lines {
		if line.WaveParticipantSnapshotID == nil {
			continue
		}
		lineCounts[*line.WaveParticipantSnapshotID]++
		if line.AllocationState == "ready" {
			readyCounts[*line.WaveParticipantSnapshotID]++
		}
	}

	rows := make([]dto.WaveParticipantRowDTO, 0, len(participants))
	for _, p := range participants {
		rows = append(rows, dto.WaveParticipantRowDTO{
			WaveParticipantSnapshotID: p.ID,
			WaveID:                    p.WaveID,
			CustomerProfileID:         p.CustomerProfileID,
			SnapshotType:              p.SnapshotType,
			DisplayName:               participantDisplay(p),
			IdentityPlatform:          p.IdentityPlatform,
			IdentityValue:             p.IdentityValue,
			GiftLevel:                 p.GiftLevel,
			SourceSummary:             p.SourceDocumentRefs,
			FulfillmentLineCount:      lineCounts[p.ID],
			ReadyFulfillmentCount:     readyCounts[p.ID],
		})
	}
	return rows, nil
}

func (uc *waveOverviewQueryUseCase) buildClosureCandidates(waveID uint) (overviewClosureCandidates, error) {
	shipments, err := uc.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return overviewClosureCandidates{}, err
	}
	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return overviewClosureCandidates{}, err
	}
	flMap := make(map[uint]*domain.FulfillmentLine, len(fulfillLines))
	for i := range fulfillLines {
		fl := fulfillLines[i]
		flMap[fl.ID] = &fl
	}

	docCache := make(map[uint]*domain.DemandDocument)
	profileCache := make(map[uint]*domain.IntegrationProfile)
	seenAuto := make(map[uint]struct{})
	seenManual := make(map[uint]struct{})

	for _, s := range shipments {
		shipLines, err := uc.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return overviewClosureCandidates{}, err
		}
		for _, sl := range shipLines {
			fl := flMap[sl.FulfillmentLineID]
			if fl == nil || fl.DemandDocumentID == nil {
				continue
			}
			docID := *fl.DemandDocumentID
			doc, ok := docCache[docID]
			if !ok {
				doc, err = uc.demandRepo.FindByID(docID)
				if err != nil {
					return overviewClosureCandidates{}, err
				}
				docCache[docID] = doc
			}
			if doc.IntegrationProfileID == nil {
				continue
			}
			profileID := *doc.IntegrationProfileID
			profile, ok := profileCache[profileID]
			if !ok {
				profile, err = uc.profileRepo.FindByID(profileID)
				if err != nil {
					return overviewClosureCandidates{}, err
				}
				profileCache[profileID] = profile
			}
			switch profile.TrackingSyncMode {
			case "api_push", "document_export":
				seenAuto[fl.ID] = struct{}{}
			case "manual_confirmation", "unsupported":
				seenManual[fl.ID] = struct{}{}
			}
		}
	}

	return overviewClosureCandidates{
		AutoCandidateCount:   len(seenAuto),
		ManualCandidateCount: len(seenManual),
	}, nil
}

func toWaveDTO(w *domain.Wave) dto.WaveDTO {
	if w == nil {
		return dto.WaveDTO{}
	}
	return dto.WaveDTO{
		ID:               w.ID,
		WaveNo:           w.WaveNo,
		Name:             w.Name,
		WaveType:         w.WaveType,
		LifecycleStage:   w.LifecycleStage,
		ProgressSnapshot: w.ProgressSnapshot,
		Notes:            w.Notes,
		LevelTags:        w.LevelTags,
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
}

func participantDisplay(p domain.WaveParticipantSnapshot) string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	if p.IdentityValue != "" {
		return p.IdentityValue
	}
	return fmt.Sprintf("participant #%d", p.ID)
}

func buildNextStepGuidance(demandCount, fulfillCount, supplierOrderCount, shipmentCount, channelSyncPendingCount int) (step, reason string) {
	switch {
	case demandCount == 0:
		return "demand_intake", "no_demands_assigned"
	case fulfillCount == 0:
		return "membership_allocation", "no_fulfillment_lines"
	case supplierOrderCount == 0:
		return "supplier_execution", "not_exported"
	case shipmentCount == 0:
		return "shipment_intake", "no_shipments"
	case channelSyncPendingCount > 0:
		return "channel_sync", "pending_sync"
	default:
		return "wave_overview", "all_steps_progressed"
	}
}

func buildBlockingIssues(addressMissingCount int, hasDriftedBasis, hasRequiredReviewBasis bool, mappingBlockedCount int, replayHealthy bool) []string {
	issues := make([]string, 0, 5)
	if addressMissingCount > 0 {
		issues = append(issues, "address_missing")
	}
	if hasDriftedBasis {
		issues = append(issues, "basis_drifted")
	}
	if hasRequiredReviewBasis {
		issues = append(issues, "review_required")
	}
	if mappingBlockedCount > 0 {
		issues = append(issues, "mapping_blocked")
	}
	if !replayHealthy {
		issues = append(issues, "replay_failures_detected")
	}
	return issues
}

func basisDriftStatusFromOverview(overview dto.WaveOverviewDTO) string {
	if overview.HasDriftedBasis {
		return "drifted"
	}
	return "in_sync"
}

func buildWorkspaceStepStates(overview dto.WaveOverviewDTO, rows []dto.WaveFulfillmentRowDTO) []dto.WaveStepStateDTO {
	return []dto.WaveStepStateDTO{
		{StepKey: "demand_intake", Status: "active", PrimaryCount: overview.DemandCount, SecondaryCount: overview.AcceptedWaitingForInput},
		{StepKey: "membership_allocation", Status: "available", PrimaryCount: overview.FulfillmentCount},
		{StepKey: "demand_mapping", Status: "available", PrimaryCount: overview.MappingBlockedCount},
		{StepKey: "wave_overview", Status: "current", PrimaryCount: len(rows)},
		{StepKey: "adjustment_review", Status: "available", PrimaryCount: overview.ManualClosureCandidateCount},
		{StepKey: "supplier_execution", Status: "available", PrimaryCount: overview.SupplierOrderCount},
		{StepKey: "shipment_intake", Status: "available", PrimaryCount: overview.ShipmentCount},
		{StepKey: "channel_sync", Status: "available", PrimaryCount: overview.ChannelSyncJobCount, SecondaryCount: overview.ChannelSyncFailedCount},
	}
}

func buildWorkspaceGuidance(overview dto.WaveOverviewDTO) []dto.WaveWorkspaceGuidanceDTO {
	guidance := make([]dto.WaveWorkspaceGuidanceDTO, 0, 5)
	if overview.AcceptedWaitingForInput > 0 {
		guidance = append(guidance, dto.WaveWorkspaceGuidanceDTO{
			Code:          "waiting_input",
			Severity:      "warning",
			TargetStepKey: "demand_intake",
			Count:         overview.AcceptedWaitingForInput,
		})
	}
	if overview.MappingBlockedCount > 0 {
		guidance = append(guidance, dto.WaveWorkspaceGuidanceDTO{
			Code:          "mapping_blocked",
			Severity:      "warning",
			TargetStepKey: "demand_mapping",
			Count:         overview.MappingBlockedCount,
		})
	}
	if overview.HasRequiredReviewBasis {
		guidance = append(guidance, dto.WaveWorkspaceGuidanceDTO{
			Code:          "basis_review_required",
			Severity:      "error",
			TargetStepKey: "wave_overview",
			Count:         1,
		})
	} else if overview.HasDriftedBasis {
		guidance = append(guidance, dto.WaveWorkspaceGuidanceDTO{
			Code:          "basis_drift_detected",
			Severity:      "warning",
			TargetStepKey: "wave_overview",
			Count:         1,
		})
	}
	if !overview.ReplayHealthy {
		guidance = append(guidance, dto.WaveWorkspaceGuidanceDTO{
			Code:          "replay_failures",
			Severity:      "warning",
			TargetStepKey: "wave_overview",
			Count:         overview.ReplayFailureCount,
		})
	}
	return guidance
}
