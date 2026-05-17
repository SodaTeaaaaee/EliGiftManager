package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type channelClosureUseCase struct {
	profileRepo        domain.IntegrationProfileRepository
	shipmentRepo       domain.ShipmentRepository
	fulfillRepo        domain.FulfillmentLineRepository
	demandRepo         domain.DemandDocumentRepository
	channelSyncUC      ChannelSyncUseCase
	carrierMappingRepo domain.CarrierMappingRepository
}

func NewChannelClosureUseCase(
	profileRepo domain.IntegrationProfileRepository,
	shipmentRepo domain.ShipmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	demandRepo domain.DemandDocumentRepository,
	channelSyncUC ChannelSyncUseCase,
	carrierMappingRepo domain.CarrierMappingRepository,
) ChannelClosureUseCase {
	return &channelClosureUseCase{
		profileRepo:        profileRepo,
		shipmentRepo:       shipmentRepo,
		fulfillRepo:        fulfillRepo,
		demandRepo:         demandRepo,
		channelSyncUC:      channelSyncUC,
		carrierMappingRepo: carrierMappingRepo,
	}
}

func (uc *channelClosureUseCase) PlanChannelClosure(input dto.PlanChannelClosureInput) (*dto.PlanChannelClosureResult, error) {
	// Resolve the effective profile view for this wave.
	// We first attempt to load a bound snapshot from any demand document in this wave
	// that references the requested profile — this ensures closure planning uses the
	// profile state that was active when the wave was assembled, not the current live state.
	effectiveProfile, err := uc.resolveEffectiveProfileForWave(input.WaveID, input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	// Candidates must be verified BEFORE any decision branch.
	// If this wave/profile has no execution objects, the closure plan
	// does not apply — regardless of tracking_sync_mode.
	candidates, err := uc.planCandidates(input.WaveID, effectiveProfile)
	if err != nil {
		return nil, fmt.Errorf("cannot plan channel sync candidates: %w", err)
	}

	result := &dto.PlanChannelClosureResult{
		IntegrationProfileID: effectiveProfile.ProfileID,
		TrackingSyncMode:     effectiveProfile.TrackingSyncMode,
		ClosurePolicy:        effectiveProfile.ClosurePolicy,
	}

	switch effectiveProfile.TrackingSyncMode {
	case "api_push", "document_export":
		result.Decision = dto.ClosureDecisionCreateJob

		lowLevelInput := dto.CreateChannelSyncJobInput{
			WaveID:               input.WaveID,
			IntegrationProfileID: effectiveProfile.ProfileID,
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
		if !effectiveProfile.AllowsManualClosure {
			return nil, fmt.Errorf("profile %q has tracking_sync_mode=manual_confirmation but allows_manual_closure=false", effectiveProfile.ProfileKey)
		}
		result.Decision = dto.ClosureDecisionManualClosure
		result.Items = candidateInputsToDTOs(candidates)

	case "unsupported":
		result.Decision = dto.ClosureDecisionUnsupported
		result.Items = candidateInputsToDTOs(candidates)

	default:
		return nil, fmt.Errorf("unknown tracking_sync_mode %q for profile %q", effectiveProfile.TrackingSyncMode, effectiveProfile.ProfileKey)
	}

	return result, nil
}

// resolveEffectiveProfileForWave returns the bound snapshot from the first demand document
// in the wave that references profileID. Falls back to a live profile lookup when no
// snapshot is stored (backward compatibility for pre-binding data).
func (uc *channelClosureUseCase) resolveEffectiveProfileForWave(waveID uint, profileID uint) (*dto.BoundProfileSnapshot, error) {
	// Walk fulfillment lines to find a demand document with a bound snapshot for this profile.
	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err == nil {
		docCache := make(map[uint]*domain.DemandDocument)
		for _, fl := range fulfillLines {
			if fl.DemandDocumentID == nil {
				continue
			}
			docID := *fl.DemandDocumentID
			if _, seen := docCache[docID]; seen {
				continue
			}
			doc, docErr := uc.demandRepo.FindByID(docID)
			if docErr != nil {
				continue
			}
			docCache[docID] = doc
			if doc.IntegrationProfileID == nil || *doc.IntegrationProfileID != profileID {
				continue
			}
			if doc.BoundProfileSnapshot != "" {
				snap, parseErr := ParseProfileSnapshot(doc.BoundProfileSnapshot)
				if parseErr == nil && snap != nil {
					return snap, nil
				}
			}
		}
	}

	// Fallback: live profile lookup.
	profile, err := uc.profileRepo.FindByID(profileID)
	if err != nil {
		return nil, err
	}
	return &dto.BoundProfileSnapshot{
		ProfileID:               profile.ID,
		ProfileKey:              profile.ProfileKey,
		TrackingSyncMode:        profile.TrackingSyncMode,
		ClosurePolicy:           profile.ClosurePolicy,
		AllowsManualClosure:     profile.AllowsManualClosure,
		RequiresCarrierMapping:  profile.RequiresCarrierMapping,
		RequiresExternalOrderNo: profile.RequiresExternalOrderNo,
		SupportsPartialShipment: profile.SupportsPartialShipment,
		ConnectorKey:            profile.ConnectorKey,
		SupportsAPIExport:       profile.SupportsAPIExport,
	}, nil
}

func (uc *channelClosureUseCase) planCandidates(waveID uint, profile *dto.BoundProfileSnapshot) ([]dto.CreateChannelSyncItemInput, error) {
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
			if doc.IntegrationProfileID == nil || *doc.IntegrationProfileID != profile.ProfileID {
				continue
			}

			// Translate carrier code when the profile requires a mapping.
			// Raw shipment carrier codes are internal identifiers; external channels
			// expect the mapped external code. Reject the candidate if no mapping exists,
			// because sending an unmapped code would silently corrupt the sync payload.
			carrierCode := s.CarrierCode
			if profile.RequiresCarrierMapping {
				mapping, mappingErr := uc.carrierMappingRepo.FindByProfileAndInternal(profile.ProfileID, carrierCode)
				if mappingErr != nil || mapping == nil {
					return nil, fmt.Errorf("profile %q requires_carrier_mapping but no mapping found for carrier %q (fulfillment line %d)", profile.ProfileKey, carrierCode, fl.ID)
				}
				carrierCode = mapping.ExternalCarrierCode
			}

			candidate := dto.CreateChannelSyncItemInput{
				FulfillmentLineID:  sl.FulfillmentLineID,
				ShipmentID:         s.ID,
				TrackingNo:         s.TrackingNo,
				CarrierCode:        carrierCode,
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
