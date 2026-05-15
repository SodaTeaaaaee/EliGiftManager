package app

import (
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/selector"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- AllocationPolicy ----

type allocationPolicyUseCase struct {
	ruleRepo       domain.AllocationPolicyRuleRepository
	fulfillRepo    domain.FulfillmentLineRepository
	waveRepo       domain.WaveRepository
	adjustmentRepo domain.FulfillmentAdjustmentRepository
	demandRepo     domain.DemandDocumentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
}

func NewAllocationPolicyUseCase(
	ruleRepo domain.AllocationPolicyRuleRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	waveRepo domain.WaveRepository,
	adjustmentRepo domain.FulfillmentAdjustmentRepository,
	demandRepo domain.DemandDocumentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
) AllocationPolicyUseCase {
	return &allocationPolicyUseCase{
		ruleRepo:       ruleRepo,
		fulfillRepo:    fulfillRepo,
		waveRepo:       waveRepo,
		adjustmentRepo: adjustmentRepo,
		demandRepo:     demandRepo,
		assignmentRepo: assignmentRepo,
	}
}

func (uc *allocationPolicyUseCase) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	// ---- Phase 1: Load all inputs (no mutations yet) ----

	// Load old policy-driven lines BEFORE deletion — needed for stable target hints.
	oldLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Build LineHints: oldLineID → (WaveParticipantSnapshotID, ProductID) for stable
	// target resolution after ID-breaking rebuild.
	lineHints := make(map[uint]LineHint)
	for _, ol := range oldLines {
		if ol.GeneratedBy == "allocation_policy_driven" &&
			ol.WaveParticipantSnapshotID != nil &&
			ol.ProductID != nil {
			lineHints[ol.ID] = LineHint{
				WaveParticipantSnapshotID: *ol.WaveParticipantSnapshotID,
				ProductID:                 *ol.ProductID,
			}
		}
	}

	// Load active rules (sorted by Priority ASC via repo).
	rules, err := uc.ruleRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Load all participants for this wave.
	allParticipants, err := uc.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Filter to eligible participants: only those whose backing DemandLines
	// include at least one accepted + ready line enter fulfillment generation.
	// (docs/04-workflows-and-state/03:122-127)
	// When demandRepo/assignmentRepo are nil (e.g. in unit tests), all participants are eligible.
	var participants []domain.WaveParticipantSnapshot
	if uc.assignmentRepo != nil && uc.demandRepo != nil {
		eligibleProfileIDs := make(map[uint]bool)
		docs, _ := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
		for _, doc := range docs {
			if doc.CustomerProfileID == nil {
				continue
			}
			lines, _ := uc.demandRepo.ListLinesByDocument(doc.ID)
			for _, line := range lines {
				if line.RoutingDisposition == "accepted" &&
					(line.RecipientInputState == "ready" || line.RecipientInputState == "not_required") {
					eligibleProfileIDs[*doc.CustomerProfileID] = true
					break
				}
			}
		}
		participants = make([]domain.WaveParticipantSnapshot, 0, len(allParticipants))
		for _, p := range allParticipants {
			if eligibleProfileIDs[p.CustomerProfileID] {
				participants = append(participants, p)
			}
		}
	} else {
		participants = allParticipants
	}

	// Load adjustments for replay (sorted by created_at ASC via repo).
	adjustments, err := uc.adjustmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	// ---- Phase 2: In-memory computation (no DB writes) ----

	// Evaluate rules → accumulate contribution map.
	// contributionMap[participantID][productID] = summed quantity
	type contribKey struct {
		participantIdx int
		productID      uint
	}
	contributionMap := make(map[contribKey]int)

	for ruleIdx := range rules {
		rule := &rules[ruleIdx]
		if !rule.Active {
			continue
		}
		matched := selector.MatchSelector(rule.SelectorPayload, participants)
		for _, p := range matched {
			key := contribKey{participantIdx: findParticipantIdx(participants, p.ID), productID: rule.ProductID}
			contributionMap[key] += rule.ContributionQuantity
		}
	}

	// Generate FulfillmentLines from contribution map.
	now := time.Now().Format(time.RFC3339)
	var newLines []domain.FulfillmentLine

	for key, sum := range contributionMap {
		quantity := sum
		if quantity <= 0 {
			continue
		}
		p := &participants[key.participantIdx]
		productID := key.productID
		participantID := p.ID
		customerProfileID := p.CustomerProfileID

		fl := domain.FulfillmentLine{
			WaveID:                    waveID,
			ProductID:                 &productID,
			WaveParticipantSnapshotID: &participantID,
			CustomerProfileID:         &customerProfileID,
			Quantity:                  quantity,
			AllocationState:           "ready",
			AddressState:              "missing",
			SupplierState:             "not_submitted",
			ChannelSyncState:          "not_required",
			LineReason:                "entitlement",
			GeneratedBy:               "allocation_policy_driven",
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}
		newLines = append(newLines, fl)
	}

	// Replay adjustments on in-memory lines BEFORE persisting.
	// Use halt-on-first-failure mode: if any adjustment fails to resolve,
	// abort the entire reconcile to preserve the old data.
	var failures []ReplayFailure
	if len(adjustments) > 0 {
		newLines, failures = ReplayAdjustments(newLines, adjustments, ReplayOptions{
			Mode:      ReplayHaltOnFirstFailure,
			LineHints: lineHints,
		})
	}

	// If replay produced failures, do NOT delete old data — return failures immediately.
	if len(failures) > 0 {
		failureDTOs := make([]dto.ReplayFailureDTO, 0, len(failures))
		for _, f := range failures {
			failureDTOs = append(failureDTOs, dto.ReplayFailureDTO{
				AdjustmentID: f.AdjustmentID,
				Reason:       f.Reason,
			})
		}
		return &dto.ReconcileResultDTO{
			Created:       0,
			Deleted:       0,
			ReplayedCount: 0,
			Failures:      failureDTOs,
		}, nil
	}

	// ---- Phase 3: Persist (only reached when replay fully succeeded) ----

	// Delete old policy-driven lines.
	if err := uc.fulfillRepo.DeleteByWaveAndGeneratedBy(waveID, "allocation_policy_driven"); err != nil {
		return nil, err
	}

	// Persist final lines (post-replay quantities).
	created := 0
	for i := range newLines {
		if newLines[i].Quantity <= 0 {
			continue
		}
		if err := uc.fulfillRepo.Create(&newLines[i]); err != nil {
			return nil, err
		}
		created++
	}

	return &dto.ReconcileResultDTO{
		Created:       created,
		Deleted:       0, // DeleteByWaveAndGeneratedBy doesn't return count; first version omits
		ReplayedCount: len(adjustments) - len(failures),
		Failures:      []dto.ReplayFailureDTO{},
	}, nil
}

func (uc *allocationPolicyUseCase) CreateRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	now := time.Now().Format(time.RFC3339)
	rule := &domain.AllocationPolicyRule{
		WaveID:               input.WaveID,
		ProductID:            input.ProductID,
		SelectorPayload:      input.SelectorPayload,
		ProductTargetRef:     input.ProductTargetRef,
		ContributionQuantity: input.ContributionQuantity,
		RuleKind:             input.RuleKind,
		Priority:             input.Priority,
		Active:               input.Active,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := uc.ruleRepo.Create(rule); err != nil {
		return nil, err
	}
	d := ruleToDTO(rule)
	return &d, nil
}

func (uc *allocationPolicyUseCase) UpdateRule(input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	rule, err := uc.ruleRepo.FindByID(input.ID)
	if err != nil {
		return nil, err
	}

	if input.ProductID != nil {
		rule.ProductID = *input.ProductID
	}
	if input.SelectorPayload != nil {
		rule.SelectorPayload = *input.SelectorPayload
	}
	if input.ProductTargetRef != nil {
		rule.ProductTargetRef = *input.ProductTargetRef
	}
	if input.ContributionQuantity != nil {
		rule.ContributionQuantity = *input.ContributionQuantity
	}
	if input.RuleKind != nil {
		rule.RuleKind = *input.RuleKind
	}
	if input.Priority != nil {
		rule.Priority = *input.Priority
	}
	if input.Active != nil {
		rule.Active = *input.Active
	}
	rule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := uc.ruleRepo.Update(rule); err != nil {
		return nil, err
	}
	d := ruleToDTO(rule)
	return &d, nil
}

func (uc *allocationPolicyUseCase) DeleteRule(ruleID uint) error {
	return uc.ruleRepo.Delete(ruleID)
}

func (uc *allocationPolicyUseCase) ListRulesByWave(waveID uint) ([]dto.AllocationPolicyRuleDTO, error) {
	rules, err := uc.ruleRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.AllocationPolicyRuleDTO, len(rules))
	for i := range rules {
		result[i] = ruleToDTO(&rules[i])
	}
	return result, nil
}

// ---- helpers ----

func ruleToDTO(r *domain.AllocationPolicyRule) dto.AllocationPolicyRuleDTO {
	return dto.AllocationPolicyRuleDTO{
		ID:                   r.ID,
		WaveID:               r.WaveID,
		ProductID:            r.ProductID,
		SelectorPayload:      r.SelectorPayload,
		ProductTargetRef:     r.ProductTargetRef,
		ContributionQuantity: r.ContributionQuantity,
		RuleKind:             r.RuleKind,
		Priority:             r.Priority,
		Active:               r.Active,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func findParticipantIdx(participants []domain.WaveParticipantSnapshot, id uint) int {
	for i := range participants {
		if participants[i].ID == id {
			return i
		}
	}
	return -1
}
