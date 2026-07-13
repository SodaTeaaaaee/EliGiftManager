package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestBatchRecordAdjustmentsReissueAcceptedAndReplays(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()
	batchUC := NewAdjustmentBatchUseCase(s.uc)

	input := dto.BatchRecordAdjustmentsInput{
		Entries: []dto.RecordAdjustmentInput{
			{
				WaveID:                    10,
				TargetKind:                "participant",
				WaveParticipantSnapshotID: uintPtr(100),
				AdjustmentKind:            string(domain.AdjustmentKindReissue),
				QuantityDelta:             3,
				ReasonCode:                "lost_in_transit",
				OperatorID:                "op-1",
				Note:                      "reissue after carrier loss",
				EvidenceRef:               "ref-900",
			},
		},
	}

	result, err := batchUC.BatchRecordAdjustments(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SuccessCount != 1 || result.FailureCount != 0 {
		t.Fatalf("expected 1 success/0 failure, got success=%d failure=%d", result.SuccessCount, result.FailureCount)
	}
	if len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("expected result[0].Success = true, got %+v", result.Results)
	}
	if result.Results[0].Adjustment == nil || result.Results[0].Adjustment.AdjustmentKind != string(domain.AdjustmentKindReissue) {
		t.Fatalf("expected recorded AdjustmentKind = reissue, got %+v", result.Results[0].Adjustment)
	}

	// Replay: reissue should behave as a positive quantity delta on the target line,
	// symmetric with add/compensation (decision #15).
	adjustments, err := s.adjRepo.ListByWave(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error listing adjustments: %v", err)
	}
	baseline := []domain.FulfillmentLine{
		{ID: 1, WaveID: 10, WaveParticipantSnapshotID: uintPtr(100), Quantity: 2},
	}
	replayed, failures := ReplayAdjustments(baseline, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no replay failures, got %+v", failures)
	}
	if replayed[0].Quantity != 5 {
		t.Errorf("Quantity after reissue replay = %d, want 5", replayed[0].Quantity)
	}
}

func TestBatchRecordAdjustmentsRejectsReissueOnFulfillmentLineTarget(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()
	batchUC := NewAdjustmentBatchUseCase(s.uc)

	input := dto.BatchRecordAdjustmentsInput{
		Entries: []dto.RecordAdjustmentInput{
			{
				WaveID:            10,
				TargetKind:        "fulfillment_line",
				FulfillmentLineID: uintPtr(1),
				AdjustmentKind:    string(domain.AdjustmentKindReissue),
				QuantityDelta:     1,
				ReasonCode:        "test",
				OperatorID:        "op-1",
			},
		},
	}

	result, err := batchUC.BatchRecordAdjustments(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if result.SuccessCount != 0 || result.FailureCount != 1 {
		t.Fatalf("expected reissue on fulfillment_line target to fail, got success=%d failure=%d", result.SuccessCount, result.FailureCount)
	}
	if result.Results[0].Error == "" {
		t.Error("expected a non-empty error message on the failed entry")
	}
}

func TestBatchRecordAdjustmentsPartialSuccess(t *testing.T) {
	t.Parallel()
	s := newAdjustmentTestSetup()
	batchUC := NewAdjustmentBatchUseCase(s.uc)

	valid := validAdjustmentInput()
	invalid := validAdjustmentInput()
	invalid.AdjustmentKind = "not_a_real_kind"

	input := dto.BatchRecordAdjustmentsInput{
		Entries: []dto.RecordAdjustmentInput{valid, invalid},
	}

	result, err := batchUC.BatchRecordAdjustments(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if !result.Results[0].Success || result.Results[0].Adjustment == nil {
		t.Errorf("expected entry 0 to succeed, got %+v", result.Results[0])
	}
	if result.Results[1].Success || result.Results[1].Error == "" {
		t.Errorf("expected entry 1 to fail with an error message, got %+v", result.Results[1])
	}
	if len(s.adjRepo.records) != 1 {
		t.Errorf("expected only the valid entry to persist, got %d records", len(s.adjRepo.records))
	}
}
