package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// validRoutingDispositions is the canonical set of allowed routing_disposition values.
var validRoutingDispositions = map[string]bool{
	"pending_intake":      true,
	"accepted":            true,
	"deferred":            true,
	"excluded_manual":     true,
	"excluded_duplicate":  true,
	"excluded_revoked":    true,
}

// validRecipientInputStates is the canonical set of allowed recipient_input_state values.
var validRecipientInputStates = map[string]bool{
	"not_required":        true,
	"waiting_for_input":   true,
	"partially_collected": true,
	"ready":               true,
	"waived":              true,
	"expired":             true,
}

type entitlementRoutingUseCase struct {
	demandRepo     domain.DemandDocumentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
}

// NewEntitlementRoutingUseCase constructs an EntitlementRoutingUseCase.
func NewEntitlementRoutingUseCase(
	demandRepo domain.DemandDocumentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
) EntitlementRoutingUseCase {
	return &entitlementRoutingUseCase{
		demandRepo:     demandRepo,
		assignmentRepo: assignmentRepo,
	}
}

// UpdateDemandLineRouting validates and applies routing field updates to a single demand line.
func (uc *entitlementRoutingUseCase) UpdateDemandLineRouting(input dto.UpdateDemandLineRoutingInput) error {
	if _, err := uc.demandRepo.FindLineByID(input.DemandLineID); err != nil {
		return fmt.Errorf("demand line %d not found: %w", input.DemandLineID, err)
	}
	if !validRoutingDispositions[input.RoutingDisposition] {
		return fmt.Errorf("invalid routing_disposition %q: must be one of pending_intake, accepted, deferred, excluded_manual, excluded_duplicate, excluded_revoked", input.RoutingDisposition)
	}
	if !validRecipientInputStates[input.RecipientInputState] {
		return fmt.Errorf("invalid recipient_input_state %q: must be one of not_required, waiting_for_input, partially_collected, ready, waived, expired", input.RecipientInputState)
	}
	return uc.demandRepo.UpdateLineRoutingFields(
		input.DemandLineID,
		input.RoutingDisposition,
		input.RecipientInputState,
		input.RoutingReasonCode,
	)
}

// BatchUpdateDemandLineRouting applies routing updates to multiple demand lines.
// Per-line errors are collected; the call itself succeeds as long as it can iterate.
func (uc *entitlementRoutingUseCase) BatchUpdateDemandLineRouting(input dto.BatchUpdateDemandLineRoutingInput) (*dto.BatchUpdateDemandLineRoutingResult, error) {
	result := &dto.BatchUpdateDemandLineRoutingResult{
		Errors: []dto.DemandLineRoutingError{},
	}
	for _, upd := range input.Updates {
		if err := uc.UpdateDemandLineRouting(upd); err != nil {
			result.Errors = append(result.Errors, dto.DemandLineRoutingError{
				DemandLineID: upd.DemandLineID,
				Reason:       err.Error(),
			})
		} else {
			result.UpdatedCount++
		}
	}
	return result, nil
}

// GetWaveRoutingStats aggregates routing disposition counts across all demand lines
// belonging to demand documents assigned to the given wave.
func (uc *entitlementRoutingUseCase) GetWaveRoutingStats(waveID uint) (*dto.WaveRoutingStatsDTO, error) {
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return nil, fmt.Errorf("list demand documents for wave %d: %w", waveID, err)
	}

	stats := &dto.WaveRoutingStatsDTO{}
	for _, doc := range docs {
		lines, err := uc.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return nil, fmt.Errorf("list lines for demand document %d: %w", doc.ID, err)
		}
		for _, line := range lines {
			stats.TotalLines++
			switch line.RoutingDisposition {
			case "accepted":
				switch line.RecipientInputState {
				case "ready", "not_required", "waived":
					stats.AcceptedReadyCount++
				case "waiting_for_input":
					stats.AcceptedWaitingCount++
				case "partially_collected":
					stats.AcceptedPartialCount++
				default:
					stats.AcceptedReadyCount++
				}
			case "deferred":
				stats.DeferredCount++
			case "excluded_manual":
				stats.ExcludedManualCount++
			case "excluded_duplicate":
				stats.ExcludedDuplicateCount++
			case "excluded_revoked":
				stats.ExcludedRevokedCount++
			default:
				// pending_intake or any unrecognized value
				stats.PendingIntakeCount++
			}
		}
	}
	return stats, nil
}
