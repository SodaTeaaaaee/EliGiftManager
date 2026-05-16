package app

import (
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
