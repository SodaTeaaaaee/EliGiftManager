package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func uint_ptr(v uint) *uint { return &v }

func TestReplayAdjustments_EmptyAdjustments(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 5},
		{ID: 2, Quantity: 3},
	}
	result, failures := ReplayAdjustments(baselines, nil)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %d", len(failures))
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 baselines, got %d", len(result))
	}
	if result[0].Quantity != 5 || result[1].Quantity != 3 {
		t.Fatalf("baselines should be unchanged, got %d and %d", result[0].Quantity, result[1].Quantity)
	}
}

func TestReplayAdjustments_AddReduceCompensation(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 10, Quantity: 10},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(10),
			AdjustmentKind:    "add",
			QuantityDelta:     3,
		},
		{
			ID:                2,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(10),
			AdjustmentKind:    "reduce",
			QuantityDelta:     -2,
		},
		{
			ID:                3,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(10),
			AdjustmentKind:    "compensation",
			QuantityDelta:     1,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	// 10 + 3 - 2 + 1 = 12
	if result[0].Quantity != 12 {
		t.Fatalf("expected quantity 12, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_OrphanedLine_FulfillmentLineTarget(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 5},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                99,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(999), // does not exist
			AdjustmentKind:    "add",
			QuantityDelta:     2,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].AdjustmentID != 99 {
		t.Fatalf("expected adjustment ID 99, got %d", failures[0].AdjustmentID)
	}
	if failures[0].Reason != "orphaned_line" {
		t.Fatalf("expected reason orphaned_line, got %s", failures[0].Reason)
	}
	// Baseline unchanged.
	if result[0].Quantity != 5 {
		t.Fatalf("expected quantity unchanged at 5, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_OrphanedLine_ParticipantTarget(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, WaveParticipantSnapshotID: uint_ptr(100), Quantity: 5},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                        50,
			TargetKind:                "participant",
			WaveParticipantSnapshotID: uint_ptr(999), // no match
			AdjustmentKind:            "add",
			QuantityDelta:             1,
		},
	}

	_, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].Reason != "orphaned_line" {
		t.Fatalf("expected orphaned_line, got %s", failures[0].Reason)
	}
}

func TestReplayAdjustments_AmbiguousTarget(t *testing.T) {
	// Two lines share the same WaveParticipantSnapshotID.
	baselines := []domain.FulfillmentLine{
		{ID: 1, WaveParticipantSnapshotID: uint_ptr(200), Quantity: 3},
		{ID: 2, WaveParticipantSnapshotID: uint_ptr(200), Quantity: 7},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                        60,
			TargetKind:                "participant",
			WaveParticipantSnapshotID: uint_ptr(200),
			AdjustmentKind:            "add",
			QuantityDelta:             2,
		},
	}

	_, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].AdjustmentID != 60 {
		t.Fatalf("expected adjustment ID 60, got %d", failures[0].AdjustmentID)
	}
	if failures[0].Reason != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target, got %s", failures[0].Reason)
	}
}

func TestReplayAdjustments_QuantityClampToZero(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 2},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1),
			AdjustmentKind:    "reduce",
			QuantityDelta:     -10, // would make it -8
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if result[0].Quantity != 0 {
		t.Fatalf("expected quantity clamped to 0, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_RemoveSetsQuantityToZero(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 5, Quantity: 42},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(5),
			AdjustmentKind:    "remove",
			QuantityDelta:     0,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if result[0].Quantity != 0 {
		t.Fatalf("expected quantity 0 after remove, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_ParticipantTarget_SingleMatch(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, WaveParticipantSnapshotID: uint_ptr(300), Quantity: 10},
		{ID: 2, WaveParticipantSnapshotID: uint_ptr(400), Quantity: 5},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                        70,
			TargetKind:                "participant",
			WaveParticipantSnapshotID: uint_ptr(300),
			AdjustmentKind:            "add",
			QuantityDelta:             4,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if result[0].Quantity != 14 {
		t.Fatalf("expected quantity 14, got %d", result[0].Quantity)
	}
	// Second line untouched.
	if result[1].Quantity != 5 {
		t.Fatalf("expected second line unchanged at 5, got %d", result[1].Quantity)
	}
}

func TestReplayAdjustments_ReplaceIsNoOp(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 8},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1),
			AdjustmentKind:    "replace",
			QuantityDelta:     0,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if result[0].Quantity != 8 {
		t.Fatalf("expected quantity unchanged at 8, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_FailureDoesNotInterruptSubsequent(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 5},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                10,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(999), // orphaned
			AdjustmentKind:    "add",
			QuantityDelta:     100,
		},
		{
			ID:                11,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1), // valid
			AdjustmentKind:    "add",
			QuantityDelta:     2,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].AdjustmentID != 10 {
		t.Fatalf("expected failed adjustment 10, got %d", failures[0].AdjustmentID)
	}
	// Second adjustment still applied.
	if result[0].Quantity != 7 {
		t.Fatalf("expected quantity 7 (5+2), got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_HaltOnFirstFailure(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 5},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                10,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(999), // orphaned — triggers halt
			AdjustmentKind:    "add",
			QuantityDelta:     100,
		},
		{
			ID:                11,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1), // valid but should NOT be applied
			AdjustmentKind:    "add",
			QuantityDelta:     2,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments, ReplayOptions{
		Mode: ReplayHaltOnFirstFailure,
	})
	if len(failures) != 1 {
		t.Fatalf("expected exactly 1 failure (halted), got %d", len(failures))
	}
	if failures[0].AdjustmentID != 10 {
		t.Fatalf("expected failed adjustment 10, got %d", failures[0].AdjustmentID)
	}
	// Second adjustment must NOT have been applied.
	if result[0].Quantity != 5 {
		t.Fatalf("expected quantity unchanged at 5 (halted before second adj), got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_StableTargetFallback_UniqueMatch(t *testing.T) {
	// Simulate post-rebuild baselines: new IDs, but same (participant, product).
	pid := uint(300)
	prodID := uint(42)
	baselines := []domain.FulfillmentLine{
		{ID: 1000, WaveParticipantSnapshotID: &pid, ProductID: &prodID, Quantity: 10},
	}
	// Adjustment was recorded against old line ID 50 — not in new baselines.
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(50), // old ID, not in baselines
			AdjustmentKind:    "add",
			QuantityDelta:     3,
		},
	}
	hints := map[uint]LineHint{
		50: {WaveParticipantSnapshotID: 300, ProductID: 42},
	}

	result, failures := ReplayAdjustments(baselines, adjustments, ReplayOptions{
		LineHints: hints,
	})
	if len(failures) != 0 {
		t.Fatalf("expected no failures (stable fallback), got %v", failures)
	}
	if result[0].Quantity != 13 {
		t.Fatalf("expected quantity 13 (10+3), got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_StableTargetFallback_NoHint_Orphaned(t *testing.T) {
	pid := uint(300)
	prodID := uint(42)
	baselines := []domain.FulfillmentLine{
		{ID: 1000, WaveParticipantSnapshotID: &pid, ProductID: &prodID, Quantity: 10},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(50),
			AdjustmentKind:    "add",
			QuantityDelta:     3,
		},
	}
	// No hints provided — fallback not possible.
	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].Reason != "orphaned_line" {
		t.Fatalf("expected orphaned_line, got %s", failures[0].Reason)
	}
	if result[0].Quantity != 10 {
		t.Fatalf("expected quantity unchanged at 10, got %d", result[0].Quantity)
	}
}

func TestReplayAdjustments_StableTargetFallback_Ambiguous(t *testing.T) {
	// Two new lines share the same (participant, product) — ambiguous fallback.
	pid := uint(300)
	prodID := uint(42)
	baselines := []domain.FulfillmentLine{
		{ID: 1000, WaveParticipantSnapshotID: &pid, ProductID: &prodID, Quantity: 5},
		{ID: 1001, WaveParticipantSnapshotID: &pid, ProductID: &prodID, Quantity: 8},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(50),
			AdjustmentKind:    "add",
			QuantityDelta:     3,
		},
	}
	hints := map[uint]LineHint{
		50: {WaveParticipantSnapshotID: 300, ProductID: 42},
	}

	_, failures := ReplayAdjustments(baselines, adjustments, ReplayOptions{
		LineHints: hints,
	})
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(failures))
	}
	if failures[0].Reason != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target, got %s", failures[0].Reason)
	}
}

func TestReplayAdjustments_ASCOrder_Semantic(t *testing.T) {
	// Verify that adjustments applied in ASC chronological order produce correct results.
	// The "add" must happen before the "remove" — if order were reversed,
	// remove would zero out first, then add would bump to 3.
	baselines := []domain.FulfillmentLine{
		{ID: 1, Quantity: 10},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                1,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1),
			AdjustmentKind:    "add",
			QuantityDelta:     5,
			CreatedAt:         "2026-01-01T00:00:00Z", // earlier
		},
		{
			ID:                2,
			TargetKind:        "fulfillment_line",
			FulfillmentLineID: uint_ptr(1),
			AdjustmentKind:    "remove",
			QuantityDelta:     0,
			CreatedAt:         "2026-01-02T00:00:00Z", // later
		},
	}
	// In ASC order: add(+5) → remove(=0). Result = 0.
	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if result[0].Quantity != 0 {
		t.Fatalf("expected quantity 0 (add then remove), got %d", result[0].Quantity)
	}
}
