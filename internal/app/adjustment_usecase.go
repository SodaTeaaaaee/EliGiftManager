package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type adjustmentUseCase struct {
	adjustmentRepo domain.FulfillmentAdjustmentRepository
	fulfillRepo    domain.FulfillmentLineRepository
}

func NewAdjustmentUseCase(
	adjustmentRepo domain.FulfillmentAdjustmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
) AdjustmentUseCase {
	return &adjustmentUseCase{
		adjustmentRepo: adjustmentRepo,
		fulfillRepo:    fulfillRepo,
	}
}

func (uc *adjustmentUseCase) RecordAdjustment(input dto.RecordAdjustmentInput) (*domain.FulfillmentAdjustment, error) {
	// Validate fulfillment line exists and belongs to the wave
	line, err := uc.fulfillRepo.FindByID(input.FulfillmentLineID)
	if err != nil {
		return nil, fmt.Errorf("fulfillment line %d not found: %w", input.FulfillmentLineID, err)
	}
	if line.WaveID != input.WaveID {
		return nil, fmt.Errorf("fulfillment line %d does not belong to wave %d", input.FulfillmentLineID, input.WaveID)
	}

	// Validate adjustment kind
	switch input.AdjustmentKind {
	case "add_send", "reduce_send", "replace", "remove", "supplement":
		// valid
	default:
		return nil, fmt.Errorf("invalid adjustment kind: %q", input.AdjustmentKind)
	}

	adj := &domain.FulfillmentAdjustment{
		WaveID:            input.WaveID,
		FulfillmentLineID: input.FulfillmentLineID,
		AdjustmentKind:    input.AdjustmentKind,
		QuantityDelta:     input.QuantityDelta,
		ReasonCode:        input.ReasonCode,
		OperatorID:        input.OperatorID,
		Note:              input.Note,
		EvidenceRef:       input.EvidenceRef,
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
			ID:                a.ID,
			WaveID:            a.WaveID,
			FulfillmentLineID: a.FulfillmentLineID,
			AdjustmentKind:    a.AdjustmentKind,
			QuantityDelta:     a.QuantityDelta,
			ReasonCode:        a.ReasonCode,
			OperatorID:        a.OperatorID,
			Note:              a.Note,
			EvidenceRef:       a.EvidenceRef,
			CreatedAt:         a.CreatedAt,
			UpdatedAt:         a.UpdatedAt,
		}
	}
	return out, nil
}
