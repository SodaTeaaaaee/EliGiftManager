package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type waveOverviewQueryUseCase struct {
	waveRepo       domain.WaveRepository
	fulfillRepo    domain.FulfillmentLineRepository
	supplierRepo   domain.SupplierOrderRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	demandRepo     domain.DemandDocumentRepository
	shipmentRepo   domain.ShipmentRepository
	productRepo    domain.ProductRepository
	profileRepo    domain.IntegrationProfileRepository
	overviewProjUC WaveOverviewProjectionUseCase
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
	overviewProjUC WaveOverviewProjectionUseCase,
) WaveOverviewQueryUseCase {
	return &waveOverviewQueryUseCase{
		waveRepo:       waveRepo,
		fulfillRepo:    fulfillRepo,
		supplierRepo:   supplierRepo,
		assignmentRepo: assignmentRepo,
		demandRepo:     demandRepo,
		shipmentRepo:   shipmentRepo,
		productRepo:    productRepo,
		profileRepo:    profileRepo,
		overviewProjUC: overviewProjUC,
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
	return uc.overviewProjUC.ProjectWaveOverview(base)
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

	historyRows := []dto.HistoryNodeDTO{}
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
		Wave:                    overview.Wave,
		ProjectedLifecycleStage: overview.ProjectedLifecycleStage,
		Overview:                overview,
		StepStates:              stepStates,
		Guidance:                guidance,
		BasisSummary:            basisSummary,
		RecentHistory:           historyRows,
	}, nil
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
	guidance := make([]dto.WaveWorkspaceGuidanceDTO, 0, 4)
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
	return guidance
}
