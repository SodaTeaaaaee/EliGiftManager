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
	fulfillRepo    domain.FulfillmentLineRepository
	ruleRepo       domain.AllocationPolicyRuleRepository
	adjRepo        domain.FulfillmentAdjustmentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	waveRepo       domain.WaveRepository
	productRepo    domain.ProductRepository
	closureRepo    domain.ChannelClosureDecisionRepository
}

func NewProjectionHashService(
	fulfillRepo domain.FulfillmentLineRepository,
	ruleRepo domain.AllocationPolicyRuleRepository,
	adjRepo domain.FulfillmentAdjustmentRepository,
	extraRepos ...interface{},
) *ProjectionHashService {
	svc := &ProjectionHashService{
		fulfillRepo: fulfillRepo,
		ruleRepo:    ruleRepo,
		adjRepo:     adjRepo,
	}
	for _, repo := range extraRepos {
		switch r := repo.(type) {
		case domain.WaveDemandAssignmentRepository:
			svc.assignmentRepo = r
		case domain.WaveRepository:
			svc.waveRepo = r
		case domain.ProductRepository:
			svc.productRepo = r
		case domain.ChannelClosureDecisionRepository:
			svc.closureRepo = r
		}
	}
	return svc
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

	// Demand assignments — sort by DemandDocumentID.
	if s.assignmentRepo != nil {
		assignments, _ := s.assignmentRepo.ListByWave(waveID)
		sort.Slice(assignments, func(i, j int) bool {
			return assignments[i].DemandDocumentID < assignments[j].DemandDocumentID
		})
		for _, a := range assignments {
			fmt.Fprintf(h, "D:%d:%s;", a.DemandDocumentID, a.AcceptedBy)
		}
	}

	// Wave participant snapshots — sort by (CustomerProfileID, SnapshotType, IdentityPlatform, IdentityValue).
	if s.waveRepo != nil {
		participants, _ := s.waveRepo.ListParticipantsByWave(waveID)
		sort.Slice(participants, func(i, j int) bool {
			if participants[i].CustomerProfileID != participants[j].CustomerProfileID {
				return participants[i].CustomerProfileID < participants[j].CustomerProfileID
			}
			if participants[i].SnapshotType != participants[j].SnapshotType {
				return participants[i].SnapshotType < participants[j].SnapshotType
			}
			if participants[i].IdentityPlatform != participants[j].IdentityPlatform {
				return participants[i].IdentityPlatform < participants[j].IdentityPlatform
			}
			return participants[i].IdentityValue < participants[j].IdentityValue
		})
		for _, p := range participants {
			fmt.Fprintf(h, "P:%d:%s:%s:%s:%s;",
				p.CustomerProfileID, p.SnapshotType, p.IdentityPlatform, p.IdentityValue, p.GiftLevel)
		}
	}

	// Wave-scoped product snapshots — sort by (SupplierPlatform, FactorySKU).
	if s.productRepo != nil {
		products, _ := s.productRepo.ListByWave(waveID)
		sort.Slice(products, func(i, j int) bool {
			if products[i].SupplierPlatform != products[j].SupplierPlatform {
				return products[i].SupplierPlatform < products[j].SupplierPlatform
			}
			return products[i].FactorySKU < products[j].FactorySKU
		})
		for _, p := range products {
			fmt.Fprintf(h, "W:%d:%s:%s:%s;",
				ptrVal(p.ProductMasterID), p.SupplierPlatform, p.FactorySKU, p.Name)
		}
	}

	// Manual closure decisions — sort by (FulfillmentLineID, DecisionKind, IntegrationProfileID).
	if s.closureRepo != nil {
		decisions, _ := s.closureRepo.ListByWave(waveID)
		sort.Slice(decisions, func(i, j int) bool {
			if decisions[i].FulfillmentLineID != decisions[j].FulfillmentLineID {
				return decisions[i].FulfillmentLineID < decisions[j].FulfillmentLineID
			}
			if decisions[i].DecisionKind != decisions[j].DecisionKind {
				return decisions[i].DecisionKind < decisions[j].DecisionKind
			}
			return decisions[i].IntegrationProfileID < decisions[j].IntegrationProfileID
		})
		for _, d := range decisions {
			fmt.Fprintf(h, "C:%d:%d:%s:%s;",
				d.FulfillmentLineID, d.IntegrationProfileID, d.DecisionKind, d.ReasonCode)
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}
