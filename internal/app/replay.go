package app

import "github.com/SodaTeaaaaee/EliGiftManager/internal/domain"

// ReplayFailure records a single adjustment that could not be applied during replay.
type ReplayFailure struct {
	AdjustmentID uint
	Reason       string // "orphaned_line" | "ambiguous_target"
}

// ReplayAdjustments applies a chronologically-ordered slice of adjustments onto
// the given baselines. Adjustments that cannot be matched to a target line are
// collected as failures without interrupting the remaining replay.
//
// Caller guarantees: adjustments are sorted by CreatedAt ascending.
func ReplayAdjustments(
	baselines []domain.FulfillmentLine,
	adjustments []domain.FulfillmentAdjustment,
) ([]domain.FulfillmentLine, []ReplayFailure) {
	if len(adjustments) == 0 {
		return baselines, nil
	}

	// Build index: line ID → slice index for O(1) lookup.
	idIndex := make(map[uint]int, len(baselines))
	for i := range baselines {
		idIndex[baselines[i].ID] = i
	}

	var failures []ReplayFailure

	for _, adj := range adjustments {
		idx, ok := resolveTarget(baselines, idIndex, adj)
		if !ok {
			failures = append(failures, ReplayFailure{
				AdjustmentID: adj.ID,
				Reason:       resolveFailureReason(baselines, adj),
			})
			continue
		}
		applyAdjustment(&baselines[idx], adj)
	}

	// Final clamp: quantity must be >= 0.
	for i := range baselines {
		if baselines[i].Quantity < 0 {
			baselines[i].Quantity = 0
		}
	}

	return baselines, failures
}

// resolveTarget finds the index of the target baseline for the given adjustment.
// Returns (index, true) on success, or (0, false) when the target cannot be
// unambiguously resolved.
func resolveTarget(
	baselines []domain.FulfillmentLine,
	idIndex map[uint]int,
	adj domain.FulfillmentAdjustment,
) (int, bool) {
	switch adj.TargetKind {
	case "fulfillment_line":
		if adj.FulfillmentLineID == nil {
			return 0, false
		}
		idx, found := idIndex[*adj.FulfillmentLineID]
		return idx, found

	case "participant":
		if adj.WaveParticipantSnapshotID == nil {
			return 0, false
		}
		targetID := *adj.WaveParticipantSnapshotID
		matchIdx := -1
		matchCount := 0
		for i := range baselines {
			if baselines[i].WaveParticipantSnapshotID != nil &&
				*baselines[i].WaveParticipantSnapshotID == targetID {
				matchIdx = i
				matchCount++
			}
		}
		if matchCount == 1 {
			return matchIdx, true
		}
		return 0, false

	default:
		return 0, false
	}
}

// resolveFailureReason determines the appropriate failure reason string when
// target resolution fails.
func resolveFailureReason(baselines []domain.FulfillmentLine, adj domain.FulfillmentAdjustment) string {
	if adj.TargetKind == "participant" && adj.WaveParticipantSnapshotID != nil {
		targetID := *adj.WaveParticipantSnapshotID
		count := 0
		for i := range baselines {
			if baselines[i].WaveParticipantSnapshotID != nil &&
				*baselines[i].WaveParticipantSnapshotID == targetID {
				count++
			}
		}
		if count > 1 {
			return "ambiguous_target"
		}
	}
	return "orphaned_line"
}

// applyAdjustment mutates the target line's Quantity based on the adjustment kind.
func applyAdjustment(line *domain.FulfillmentLine, adj domain.FulfillmentAdjustment) {
	switch adj.AdjustmentKind {
	case "add", "compensation", "reduce":
		line.Quantity += adj.QuantityDelta
	case "remove":
		line.Quantity = 0
	case "replace":
		// TODO: replace changes the product, not quantity.
		// First version does not handle product replacement; no-op on quantity.
	}
}
