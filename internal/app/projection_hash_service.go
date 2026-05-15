package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ProjectionHashService struct {
	fulfillRepo domain.FulfillmentLineRepository
	ruleRepo    domain.AllocationPolicyRuleRepository
	adjRepo     domain.FulfillmentAdjustmentRepository
}

func NewProjectionHashService(
	fulfillRepo domain.FulfillmentLineRepository,
	ruleRepo domain.AllocationPolicyRuleRepository,
	adjRepo domain.FulfillmentAdjustmentRepository,
) *ProjectionHashService {
	return &ProjectionHashService{
		fulfillRepo: fulfillRepo,
		ruleRepo:    ruleRepo,
		adjRepo:     adjRepo,
	}
}

// ptrVal safely dereferences a *uint; returns 0 for nil.
func ptrVal(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// ComputeHash returns a stable SHA-256 digest of the wave's semantic projection
// state.  IDs are excluded from the hash inputs because they change after a
// restore-snapshot cycle; instead rows are sorted by stable semantic keys.
func (s *ProjectionHashService) ComputeHash(waveID uint) string {
	h := sha256.New()

	// Rules — sort by (ProductID, RuleKind, Priority) — stable regardless of row ID.
	rules, _ := s.ruleRepo.ListByWave(waveID)
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].ProductID != rules[j].ProductID {
			return rules[i].ProductID < rules[j].ProductID
		}
		if rules[i].RuleKind != rules[j].RuleKind {
			return rules[i].RuleKind < rules[j].RuleKind
		}
		return rules[i].Priority < rules[j].Priority
	})
	for _, r := range rules {
		selectorJSON, _ := json.Marshal(r.SelectorPayload)
		fmt.Fprintf(h, "R:%d:%d:%s:%d:%t:%s;",
			r.ProductID, r.ContributionQuantity, r.RuleKind, r.Priority, r.Active, selectorJSON)
	}

	// Fulfillment lines — sort by (WaveParticipantSnapshotID, ProductID, DemandLineID).
	lines, _ := s.fulfillRepo.ListByWave(waveID)
	sort.Slice(lines, func(i, j int) bool {
		pi := ptrVal(lines[i].WaveParticipantSnapshotID)
		pj := ptrVal(lines[j].WaveParticipantSnapshotID)
		if pi != pj {
			return pi < pj
		}
		ppi := ptrVal(lines[i].ProductID)
		ppj := ptrVal(lines[j].ProductID)
		if ppi != ppj {
			return ppi < ppj
		}
		return ptrVal(lines[i].DemandLineID) < ptrVal(lines[j].DemandLineID)
	})
	for _, l := range lines {
		fmt.Fprintf(h, "F:%d:%d:%d:%s:%s;",
			ptrVal(l.WaveParticipantSnapshotID), ptrVal(l.ProductID),
			l.Quantity, l.GeneratedBy, l.AllocationState)
	}

	// Adjustments — sort by (AdjustmentKind, QuantityDelta, FulfillmentLineID).
	adjs, _ := s.adjRepo.ListByWave(waveID)
	sort.Slice(adjs, func(i, j int) bool {
		if adjs[i].AdjustmentKind != adjs[j].AdjustmentKind {
			return adjs[i].AdjustmentKind < adjs[j].AdjustmentKind
		}
		if adjs[i].QuantityDelta != adjs[j].QuantityDelta {
			return adjs[i].QuantityDelta < adjs[j].QuantityDelta
		}
		return ptrVal(adjs[i].FulfillmentLineID) < ptrVal(adjs[j].FulfillmentLineID)
	})
	for _, a := range adjs {
		fmt.Fprintf(h, "A:%s:%d:%s:%d;",
			a.AdjustmentKind, a.QuantityDelta, a.TargetKind, ptrVal(a.FulfillmentLineID))
	}

	return hex.EncodeToString(h.Sum(nil))
}
