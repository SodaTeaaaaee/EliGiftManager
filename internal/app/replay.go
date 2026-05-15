package app

import "github.com/SodaTeaaaaee/EliGiftManager/internal/domain"

// ReplayFailure records a single adjustment that could not be applied during replay.
type ReplayFailure struct {
	AdjustmentID uint
	Reason       string // "orphaned_line" | "ambiguous_target"
}

// ReplayMode controls how failures are handled during replay.
type ReplayMode int

const (
	// ReplayMarkAndContinue skips failed adjustments and continues with the rest.
	ReplayMarkAndContinue ReplayMode = iota
	// ReplayHaltOnFirstFailure stops replay on the first failed adjustment.
	ReplayHaltOnFirstFailure
)

// LineHint stores the stable identity of a fulfillment line for target resolution
// after reconcile rebuild (where line IDs change).
type LineHint struct {
	WaveParticipantSnapshotID uint
	ProductID                 uint
}

// ReplayOptions configures replay behaviour. Zero value = mark-and-continue + no hints.
type ReplayOptions struct {
	Mode      ReplayMode
	LineHints map[uint]LineHint // oldLineID → stable identity; nil if not needed
}

// ReplayAdjustments applies a chronologically-ordered slice of adjustments onto
// the given baselines.
//
// Caller guarantees: adjustments are sorted by CreatedAt ascending.
func ReplayAdjustments(
	baselines []domain.FulfillmentLine,
	adjustments []domain.FulfillmentAdjustment,
	opts ...ReplayOptions,
) ([]domain.FulfillmentLine, []ReplayFailure) {
	if len(adjustments) == 0 {
		return baselines, nil
	}

	var opt ReplayOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Build index: line ID → slice index for O(1) lookup.
	idIndex := make(map[uint]int, len(baselines))
	for i := range baselines {
		idIndex[baselines[i].ID] = i
	}

	var failures []ReplayFailure

	for _, adj := range adjustments {
		idx, ok := resolveTarget(baselines, idIndex, adj, opt.LineHints)
		if !ok {
			failures = append(failures, ReplayFailure{
				AdjustmentID: adj.ID,
				Reason:       resolveFailureReason(baselines, adj, opt.LineHints),
			})
			if opt.Mode == ReplayHaltOnFirstFailure {
				break
			}
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
	lineHints map[uint]LineHint,
) (int, bool) {
	switch adj.TargetKind {
	case "fulfillment_line":
		if adj.FulfillmentLineID == nil {
			return 0, false
		}
		idx, found := idIndex[*adj.FulfillmentLineID]
		if found {
			return idx, true
		}
		// Stable target fallback: old line ID not in baselines (post-rebuild).
		// Use LineHints to resolve via (WaveParticipantSnapshotID, ProductID).
		return stableTargetFallback(baselines, *adj.FulfillmentLineID, lineHints)

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

// stableTargetFallback resolves an orphaned fulfillment_line target using
// (WaveParticipantSnapshotID, ProductID) from LineHints. Returns (index, true) if
// exactly one baseline matches; (0, false) otherwise.
func stableTargetFallback(
	baselines []domain.FulfillmentLine,
	oldLineID uint,
	lineHints map[uint]LineHint,
) (int, bool) {
	if lineHints == nil {
		return 0, false
	}
	hint, ok := lineHints[oldLineID]
	if !ok {
		return 0, false
	}

	matchIdx := -1
	matchCount := 0
	for i := range baselines {
		if baselines[i].WaveParticipantSnapshotID != nil &&
			*baselines[i].WaveParticipantSnapshotID == hint.WaveParticipantSnapshotID &&
			baselines[i].ProductID != nil &&
			*baselines[i].ProductID == hint.ProductID {
			matchIdx = i
			matchCount++
		}
	}
	if matchCount == 1 {
		return matchIdx, true
	}
	return 0, false
}

// resolveFailureReason determines the appropriate failure reason string when
// target resolution fails.
func resolveFailureReason(
	baselines []domain.FulfillmentLine,
	adj domain.FulfillmentAdjustment,
	lineHints map[uint]LineHint,
) string {
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

	// For fulfillment_line targets with hints, check if the hint matched multiple baselines.
	if adj.TargetKind == "fulfillment_line" && adj.FulfillmentLineID != nil && lineHints != nil {
		hint, ok := lineHints[*adj.FulfillmentLineID]
		if ok {
			count := 0
			for i := range baselines {
				if baselines[i].WaveParticipantSnapshotID != nil &&
					*baselines[i].WaveParticipantSnapshotID == hint.WaveParticipantSnapshotID &&
					baselines[i].ProductID != nil &&
					*baselines[i].ProductID == hint.ProductID {
					count++
				}
			}
			if count > 1 {
				return "ambiguous_target"
			}
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
