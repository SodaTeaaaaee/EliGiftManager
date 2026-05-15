package app

import (
	"crypto/sha256"
	"encoding/hex"
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

func (s *ProjectionHashService) ComputeHash(waveID uint) string {
	h := sha256.New()

	rules, _ := s.ruleRepo.ListByWave(waveID)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	for _, r := range rules {
		fmt.Fprintf(h, "R:%d:%d:%d:%s:%d:%t;", r.ID, r.ProductID, r.ContributionQuantity, r.RuleKind, r.Priority, r.Active)
	}

	lines, _ := s.fulfillRepo.ListByWave(waveID)
	sort.Slice(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })
	for _, l := range lines {
		fmt.Fprintf(h, "F:%d:%d:%d:%s;", l.ID, l.Quantity, l.ProductID, l.GeneratedBy)
	}

	adjs, _ := s.adjRepo.ListByWave(waveID)
	sort.Slice(adjs, func(i, j int) bool { return adjs[i].ID < adjs[j].ID })
	for _, a := range adjs {
		fmt.Fprintf(h, "A:%d:%s:%d;", a.ID, a.AdjustmentKind, a.QuantityDelta)
	}

	return hex.EncodeToString(h.Sum(nil))
}
