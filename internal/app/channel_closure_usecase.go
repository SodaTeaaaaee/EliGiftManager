package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type channelClosureUseCase struct {
	profileRepo  domain.IntegrationProfileRepository
	shipmentRepo domain.ShipmentRepository
	fulfillRepo  domain.FulfillmentLineRepository
	demandRepo   domain.DemandDocumentRepository
	channelSyncUC ChannelSyncUseCase
}

func NewChannelClosureUseCase(
	profileRepo domain.IntegrationProfileRepository,
	shipmentRepo domain.ShipmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	demandRepo domain.DemandDocumentRepository,
	channelSyncUC ChannelSyncUseCase,
) ChannelClosureUseCase {
	return &channelClosureUseCase{
		profileRepo:  profileRepo,
		shipmentRepo: shipmentRepo,
		fulfillRepo:  fulfillRepo,
		demandRepo:   demandRepo,
		channelSyncUC: channelSyncUC,
	}
}

func (uc *channelClosureUseCase) PlanChannelClosure(input dto.PlanChannelClosureInput) (*dto.PlanChannelClosureResult, error) {
	profile, err := uc.profileRepo.FindByID(input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	// Candidates must be verified BEFORE any decision branch.
	// If this wave/profile has no execution objects, the closure plan
	// does not apply — regardless of tracking_sync_mode.
	candidates, err := uc.planCandidates(input.WaveID, profile)
	if err != nil {
		return nil, fmt.Errorf("cannot plan channel sync candidates: %w", err)
	}

	result := &dto.PlanChannelClosureResult{
		IntegrationProfileID: profile.ID,
		TrackingSyncMode:     profile.TrackingSyncMode,
		ClosurePolicy:        profile.ClosurePolicy,
	}

	switch profile.TrackingSyncMode {
	case "api_push", "document_export":
		result.Decision = dto.ClosureDecisionCreateJob

		lowLevelInput := dto.CreateChannelSyncJobInput{
			WaveID:               input.WaveID,
			IntegrationProfileID: profile.ID,
			Direction:            "push_tracking",
			Items:                candidates,
		}
		job, items, err := uc.channelSyncUC.CreateChannelSyncJob(lowLevelInput)
		if err != nil {
			return nil, fmt.Errorf("create channel sync job: %w", err)
		}
		result.Job = domainJobToDTO(job)
		result.Items = domainItemsToDTOs(items)

	case "manual_confirmation":
		if !profile.AllowsManualClosure {
			return nil, fmt.Errorf("profile %q has tracking_sync_mode=manual_confirmation but allows_manual_closure=false", profile.ProfileKey)
		}
		result.Decision = dto.ClosureDecisionManualClosure
		result.Items = candidateInputsToDTOs(candidates)

	case "unsupported":
		result.Decision = dto.ClosureDecisionUnsupported
		result.Items = candidateInputsToDTOs(candidates)

	default:
		return nil, fmt.Errorf("unknown tracking_sync_mode %q for profile %q", profile.TrackingSyncMode, profile.ProfileKey)
	}

	return result, nil
}

func (uc *channelClosureUseCase) planCandidates(waveID uint, profile *domain.IntegrationProfile) ([]dto.CreateChannelSyncItemInput, error) {
	shipments, err := uc.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, fmt.Errorf("list shipments: %w", err)
	}

	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment lines: %w", err)
	}
	flMap := make(map[uint]*domain.FulfillmentLine, len(fulfillLines))
	for i := range fulfillLines {
		flMap[fulfillLines[i].ID] = &fulfillLines[i]
	}

	docCache := make(map[uint]*domain.DemandDocument)

	var candidates []dto.CreateChannelSyncItemInput
	for _, s := range shipments {
		sLines, err := uc.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return nil, fmt.Errorf("list shipment lines for shipment %d: %w", s.ID, err)
		}
		for _, sl := range sLines {
			fl := flMap[sl.FulfillmentLineID]
			if fl == nil {
				continue
			}

			if fl.DemandDocumentID == nil {
				continue
			}
			docID := *fl.DemandDocumentID
			doc, ok := docCache[docID]
			if !ok {
				d, err := uc.demandRepo.FindByID(docID)
				if err != nil {
					return nil, fmt.Errorf("fulfillment line %d references demand document %d which was not found: %w", fl.ID, docID, err)
				}
				docCache[docID] = d
				doc = d
			}
			if doc.IntegrationProfileID == nil || *doc.IntegrationProfileID != profile.ID {
				continue
			}

			candidate := dto.CreateChannelSyncItemInput{
				FulfillmentLineID:  sl.FulfillmentLineID,
				ShipmentID:         s.ID,
				TrackingNo:         s.TrackingNo,
				CarrierCode:        s.CarrierCode,
				ExternalDocumentNo: doc.SourceDocumentNo,
			}

			if fl.DemandLineID != nil {
				dl, err := uc.demandRepo.FindLineByID(*fl.DemandLineID)
				if err != nil {
					return nil, fmt.Errorf("fulfillment line %d references demand line %d which was not found: %w", fl.ID, *fl.DemandLineID, err)
				}
				candidate.ExternalLineNo = fmt.Sprintf("%d", dl.SourceLineNo)
			}

			if profile.RequiresExternalOrderNo && candidate.ExternalDocumentNo == "" {
				return nil, fmt.Errorf("profile %q requires_external_order_no but fulfillment line %d has no derivable external_document_no", profile.ProfileKey, fl.ID)
			}

			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no sync candidates found for wave %d and profile %q", waveID, profile.ProfileKey)
	}

	return candidates, nil
}

func domainJobToDTO(job *domain.ChannelSyncJob) *dto.ChannelSyncJobDTO {
	if job == nil {
		return nil
	}
	return &dto.ChannelSyncJobDTO{
		ID:                   job.ID,
		WaveID:               job.WaveID,
		IntegrationProfileID: job.IntegrationProfileID,
		Direction:            job.Direction,
		Status:               job.Status,
		BasisHistoryNodeID:   job.BasisHistoryNodeID,
		BasisProjectionHash:  job.BasisProjectionHash,
		BasisPayloadSnapshot: job.BasisPayloadSnapshot,
		RequestPayload:       job.RequestPayload,
		ResponsePayload:      job.ResponsePayload,
		ErrorMessage:         job.ErrorMessage,
		StartedAt:            job.StartedAt,
		FinishedAt:           job.FinishedAt,
		CreatedAt:            job.CreatedAt,
		UpdatedAt:            job.UpdatedAt,
	}
}

func domainItemsToDTOs(items []domain.ChannelSyncItem) []dto.ChannelSyncItemDTO {
	out := make([]dto.ChannelSyncItemDTO, len(items))
	for i, it := range items {
		out[i] = dto.ChannelSyncItemDTO{
			ID:                 it.ID,
			ChannelSyncJobID:   it.ChannelSyncJobID,
			FulfillmentLineID:  it.FulfillmentLineID,
			ShipmentID:         it.ShipmentID,
			ExternalDocumentNo: it.ExternalDocumentNo,
			ExternalLineNo:     it.ExternalLineNo,
			TrackingNo:         it.TrackingNo,
			CarrierCode:        it.CarrierCode,
			Status:             it.Status,
			ErrorMessage:       it.ErrorMessage,
			CreatedAt:          it.CreatedAt,
			UpdatedAt:          it.UpdatedAt,
		}
	}
	return out
}

func candidateInputsToDTOs(candidates []dto.CreateChannelSyncItemInput) []dto.ChannelSyncItemDTO {
	out := make([]dto.ChannelSyncItemDTO, len(candidates))
	for i, c := range candidates {
		out[i] = dto.ChannelSyncItemDTO{
			FulfillmentLineID:  c.FulfillmentLineID,
			ShipmentID:         c.ShipmentID,
			ExternalDocumentNo: c.ExternalDocumentNo,
			ExternalLineNo:     c.ExternalLineNo,
			TrackingNo:         c.TrackingNo,
			CarrierCode:        c.CarrierCode,
			// ID and ChannelSyncJobID are zero — these are planned, not persisted.
		}
	}
	return out
}
