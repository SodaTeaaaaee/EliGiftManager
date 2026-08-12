package app

import (
	"context"
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

func (uc *channelClosureUseCase) PlanChannelClosure(ctx context.Context, input dto.PlanChannelClosureInput) (*dto.PlanChannelClosureResult, error) {
	// Resolve the effective profile view for this wave.
	// We first attempt to load a bound snapshot from any demand document in this wave
	// that references the requested profile — this ensures closure planning uses the
	// profile state that was active when the wave was assembled, not the current live state.
	effectiveProfile, err := uc.resolveEffectiveProfileForWave(ctx, input.WaveID, input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	// Candidates must be verified BEFORE any decision branch.
	// If this wave/profile has no execution objects, the closure plan
	// does not apply — regardless of tracking_sync_mode.
	candidates, err := uc.planCandidates(ctx, input.WaveID, effectiveProfile)
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
		job, items, err := uc.channelSyncUC.CreateChannelSyncJob(ctx, lowLevelInput)
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

// resolveEffectiveProfileForWave returns the single consistent bound snapshot for
// all demand documents in the wave that reference profileID. It falls back to a
// live profile only when none of those documents has a snapshot.
func (uc *channelClosureUseCase) resolveEffectiveProfileForWave(ctx context.Context, waveID uint, profileID uint) (*dto.BoundProfileSnapshot, error) {
	fulfillLines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment lines for wave %d: %w", waveID, err)
	}
	docCache := make(map[uint]struct{})
	var effective *dto.BoundProfileSnapshot
	for _, fl := range fulfillLines {
		if fl.DemandDocumentID == nil {
			continue
		}
		docID := *fl.DemandDocumentID
		if _, seen := docCache[docID]; seen {
			continue
		}
		docCache[docID] = struct{}{}
		doc, docErr := uc.demandRepo.FindByID(ctx, docID)
		if docErr != nil {
			return nil, fmt.Errorf("load demand document %d for wave profile snapshot: %w", docID, docErr)
		}
		if doc == nil || doc.IntegrationProfileID == nil || *doc.IntegrationProfileID != profileID || doc.BoundProfileSnapshot == "" {
			continue
		}
		snap, parseErr := ParseProfileSnapshot(doc.BoundProfileSnapshot)
		if parseErr != nil {
			return nil, fmt.Errorf("parse bound profile snapshot for demand document %d: %w", docID, parseErr)
		}
		if snap == nil {
			return nil, fmt.Errorf("demand document %d has an empty bound profile snapshot", docID)
		}
		if snap.ProfileID != profileID {
			return nil, fmt.Errorf("demand document %d snapshot profile %d does not match requested profile %d", docID, snap.ProfileID, profileID)
		}
		if effective != nil && *effective != *snap {
			return nil, fmt.Errorf("wave %d has inconsistent bound snapshots for profile %d", waveID, profileID)
		}
		copy := *snap
		effective = &copy
	}
	if effective != nil {
		return effective, nil
	}

	// Fallback: live profile lookup.
	profile, err := uc.profileRepo.FindByID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return &dto.BoundProfileSnapshot{
		ProfileID:                      profile.ID,
		ProfileKey:                     profile.ProfileKey,
		SourceSurface:                  profile.SourceSurface,
		TrackingSyncMode:               profile.TrackingSyncMode,
		ClosurePolicy:                  profile.ClosurePolicy,
		AllowsManualClosure:            profile.AllowsManualClosure,
		RequiresCarrierMapping:         profile.RequiresCarrierMapping,
		RequiresExternalOrderNo:        profile.RequiresExternalOrderNo,
		SupportsPartialShipment:        profile.SupportsPartialShipment,
		ConnectorKey:                   profile.ConnectorKey,
		FactorySupplierPlatform:        profile.FactorySupplierPlatform,
		SupportsAPIExport:              profile.SupportsAPIExport,
		SupportsExportSupplierOrder:    profile.SupportsExportSupplierOrder,
		SupportsImportProductCatalog:   profile.SupportsImportProductCatalog,
		SupportsImportSupplierShipment: profile.SupportsImportSupplierShipment,
	}, nil
}

func (uc *channelClosureUseCase) planCandidates(ctx context.Context, waveID uint, profile *dto.BoundProfileSnapshot) ([]dto.CreateChannelSyncItemInput, error) {
	shipments, err := uc.shipmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("list shipments: %w", err)
	}

	fulfillLines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
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
		sLines, err := uc.shipmentRepo.ListLinesByShipment(ctx, s.ID)
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
				d, err := uc.demandRepo.FindByID(ctx, docID)
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
				mapping, mappingErr := uc.carrierMappingRepo.FindByProfileAndInternal(ctx, profile.ProfileID, carrierCode)
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
				dl, err := uc.demandRepo.FindLineByID(ctx, *fl.DemandLineID)
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
