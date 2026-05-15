package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type adjustmentUseCase struct {
	adjustmentRepo domain.FulfillmentAdjustmentRepository
	fulfillRepo    domain.FulfillmentLineRepository
	waveRepo       domain.WaveRepository
}

func NewAdjustmentUseCase(
	adjustmentRepo domain.FulfillmentAdjustmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	waveRepo domain.WaveRepository,
) AdjustmentUseCase {
	return &adjustmentUseCase{
		adjustmentRepo: adjustmentRepo,
		fulfillRepo:    fulfillRepo,
		waveRepo:       waveRepo,
	}
}

func (uc *adjustmentUseCase) RecordAdjustment(input dto.RecordAdjustmentInput) (*domain.FulfillmentAdjustment, error) {
	// Default TargetKind for backward compatibility
	targetKind := input.TargetKind
	if targetKind == "" {
		targetKind = "fulfillment_line"
	}

	// Validate adjustment kind
	switch input.AdjustmentKind {
	case "add_send", "reduce_send", "remove", "supplement":
		// valid
	default:
		return nil, fmt.Errorf("invalid adjustment kind: %q", input.AdjustmentKind)
	}

	// Validate target kind and enforce kind-target constraints
	switch targetKind {
	case "fulfillment_line":
		// supplement is only allowed with participant target
		if input.AdjustmentKind == "supplement" {
			return nil, fmt.Errorf("adjustment kind %q requires target_kind \"participant\"", input.AdjustmentKind)
		}
		if input.FulfillmentLineID == nil || *input.FulfillmentLineID == 0 {
			return nil, fmt.Errorf("fulfillment_line_id is required when target_kind is \"fulfillment_line\"")
		}
		// Validate fulfillment line exists and belongs to the wave
		line, err := uc.fulfillRepo.FindByID(*input.FulfillmentLineID)
		if err != nil {
			return nil, fmt.Errorf("fulfillment line %d lookup failed: %w", *input.FulfillmentLineID, err)
		}
		if line == nil {
			return nil, fmt.Errorf("fulfillment line %d not found", *input.FulfillmentLineID)
		}
		if line.WaveID != input.WaveID {
			return nil, fmt.Errorf("fulfillment line %d does not belong to wave %d", *input.FulfillmentLineID, input.WaveID)
		}

	case "participant":
		// add_send/reduce_send/remove require fulfillment_line target
		switch input.AdjustmentKind {
		case "add_send", "reduce_send", "remove":
			return nil, fmt.Errorf("adjustment kind %q requires target_kind \"fulfillment_line\"", input.AdjustmentKind)
		}
		if input.WaveParticipantSnapshotID == nil || *input.WaveParticipantSnapshotID == 0 {
			return nil, fmt.Errorf("wave_participant_snapshot_id is required when target_kind is \"participant\"")
		}
		// Validate participant snapshot exists and belongs to the wave.
		// Use WaveRepository.ListParticipantsByWave to verify ownership.
		participants, err := uc.waveRepo.ListParticipantsByWave(input.WaveID)
		if err != nil {
			return nil, fmt.Errorf("participant snapshot lookup failed: %w", err)
		}
		found := false
		for _, p := range participants {
			if p.ID == *input.WaveParticipantSnapshotID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("wave participant snapshot %d not found in wave %d", *input.WaveParticipantSnapshotID, input.WaveID)
		}

	default:
		return nil, fmt.Errorf("invalid target_kind: %q", targetKind)
	}

	adj := &domain.FulfillmentAdjustment{
		WaveID:                    input.WaveID,
		TargetKind:                targetKind,
		FulfillmentLineID:         input.FulfillmentLineID,
		WaveParticipantSnapshotID: input.WaveParticipantSnapshotID,
		AdjustmentKind:            input.AdjustmentKind,
		QuantityDelta:             input.QuantityDelta,
		ReasonCode:                input.ReasonCode,
		OperatorID:                input.OperatorID,
		Note:                      input.Note,
		EvidenceRef:               input.EvidenceRef,
	}

	if err := uc.adjustmentRepo.Create(adj); err != nil {
		return nil, err
	}
	return adj, nil
}

func (uc *adjustmentUseCase) ListAdjustmentsByWave(waveID uint) ([]dto.FulfillmentAdjustmentDTO, error) {
	adjs, err := uc.adjustmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.FulfillmentAdjustmentDTO, len(adjs))
	for i, a := range adjs {
		out[i] = dto.FulfillmentAdjustmentDTO{
			ID:                        a.ID,
			WaveID:                    a.WaveID,
			TargetKind:                a.TargetKind,
			FulfillmentLineID:         a.FulfillmentLineID,
			WaveParticipantSnapshotID: a.WaveParticipantSnapshotID,
			AdjustmentKind:            a.AdjustmentKind,
			QuantityDelta:             a.QuantityDelta,
			ReasonCode:                a.ReasonCode,
			OperatorID:                a.OperatorID,
			Note:                      a.Note,
			EvidenceRef:               a.EvidenceRef,
			CreatedAt:                 a.CreatedAt,
			UpdatedAt:                 a.UpdatedAt,
		}
	}
	return out, nil
}
