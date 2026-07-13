package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// AdjustmentBatchUseCase records a batch of adjustments in a single call, reporting
// per-entry success/failure so a single bad entry does not abort the rest (plan 5.3).
type AdjustmentBatchUseCase interface {
	BatchRecordAdjustments(ctx context.Context, input dto.BatchRecordAdjustmentsInput) (dto.BatchRecordAdjustmentsResult, error)
}

type adjustmentBatchUseCase struct {
	adjustmentUC AdjustmentUseCase
}

// NewAdjustmentBatchUseCase wires the batch use case on top of the existing
// single-record AdjustmentUseCase so validation/persistence logic is not duplicated.
func NewAdjustmentBatchUseCase(adjustmentUC AdjustmentUseCase) AdjustmentBatchUseCase {
	return &adjustmentBatchUseCase{adjustmentUC: adjustmentUC}
}

func (uc *adjustmentBatchUseCase) BatchRecordAdjustments(ctx context.Context, input dto.BatchRecordAdjustmentsInput) (dto.BatchRecordAdjustmentsResult, error) {
	result := dto.BatchRecordAdjustmentsResult{
		Results: make([]dto.BatchAdjustmentItemResult, 0, len(input.Entries)),
	}

	for i, entry := range input.Entries {
		adj, err := uc.adjustmentUC.RecordAdjustment(ctx, entry)
		if err != nil {
			result.Results = append(result.Results, dto.BatchAdjustmentItemResult{
				Index:   i,
				Success: false,
				Error:   err.Error(),
			})
			result.FailureCount++
			continue
		}

		itemDTO := adjustmentToDTO(*adj)
		result.Results = append(result.Results, dto.BatchAdjustmentItemResult{
			Index:      i,
			Success:    true,
			Adjustment: &itemDTO,
		})
		result.SuccessCount++
	}

	return result, nil
}

// adjustmentToDTO maps a domain FulfillmentAdjustment to its DTO representation.
// Kept local to the batch use case to avoid reaching into the controller layer
// (package main) for the equivalent mapper.
func adjustmentToDTO(a domain.FulfillmentAdjustment) dto.FulfillmentAdjustmentDTO {
	return dto.FulfillmentAdjustmentDTO{
		ID:                        a.ID,
		WaveID:                    a.WaveID,
		TargetKind:                a.TargetKind,
		FulfillmentLineID:         a.FulfillmentLineID,
		WaveParticipantSnapshotID: a.WaveParticipantSnapshotID,
		AdjustmentKind:            a.AdjustmentKind,
		QuantityDelta:             a.QuantityDelta,
		FromProductID:             a.FromProductID,
		ToProductID:               a.ToProductID,
		ReasonCode:                a.ReasonCode,
		OperatorID:                a.OperatorID,
		Note:                      a.Note,
		EvidenceRef:               a.EvidenceRef,
		CreatedAt:                 a.CreatedAt,
		UpdatedAt:                 a.UpdatedAt,
	}
}
