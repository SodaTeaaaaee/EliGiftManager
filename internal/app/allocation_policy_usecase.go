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
}

func NewAllocationPolicyUseCase(
	ruleRepo domain.AllocationPolicyRuleRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	waveRepo domain.WaveRepository,
	adjustmentRepo domain.FulfillmentAdjustmentRepository,
) AllocationPolicyUseCase {
	return &allocationPolicyUseCase{
		ruleRepo:       ruleRepo,
		fulfillRepo:    fulfillRepo,
		waveRepo:       waveRepo,
		adjustmentRepo: adjustmentRepo,
	}
}

func (uc *allocationPolicyUseCase) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	// Step 1: Idempotent delete of old policy-driven lines.
	if err := uc.fulfillRepo.DeleteByWaveAndGeneratedBy(waveID, "allocation_policy_driven"); err != nil {
		return nil, err
	}

	// Step 2: Load active rules (sorted by Priority ASC via repo).
	rules, err := uc.ruleRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Step 3: Load all participants for this wave.
	participants, err := uc.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Steps 4-6: Evaluate rules → accumulate contribution map.
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

	// Step 7: Generate FulfillmentLines from contribution map.
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

	// Step 9: Load adjustments for replay.
	adjustments, err := uc.adjustmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Step 10: Replay adjustments on in-memory lines BEFORE persisting.
	// Design decision: replay before Create so we don't need a repo Update method.
	// Participant-targeted adjustments resolve via WaveParticipantSnapshotID (already set).
	var failures []ReplayFailure
	replayedCount := len(adjustments)
	if len(adjustments) > 0 {
		newLines, failures = ReplayAdjustments(newLines, adjustments)
	}

	// Step 8/11: Persist final lines (post-replay quantities).
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

	// Step 12: Build result DTO.
	failureDTOs := make([]dto.ReplayFailureDTO, 0, len(failures))
	for _, f := range failures {
		failureDTOs = append(failureDTOs, dto.ReplayFailureDTO{
			AdjustmentID: f.AdjustmentID,
			Reason:       f.Reason,
		})
	}

	return &dto.ReconcileResultDTO{
		Created:       created,
		Deleted:       0, // DeleteByWaveAndGeneratedBy doesn't return count; first version omits
		ReplayedCount: replayedCount,
		Failures:      failureDTOs,
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
