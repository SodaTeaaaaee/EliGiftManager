package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// AllocationPolicyController exposes allocation-policy Wails bindings.
type AllocationPolicyController struct {
	uc app.AllocationPolicyUseCase
}

func NewAllocationPolicyController() *AllocationPolicyController {
	gdb := database.GetDB()
	ruleRepo := infra.NewRuleRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	demandRepo := infra.NewDemandRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)

	return &AllocationPolicyController{
		uc: app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo),
	}
}

// ReconcileWave idempotently rebuilds policy-driven fulfillment lines for the wave,
// replaying any recorded adjustments.
func (c *AllocationPolicyController) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	return c.uc.ReconcileWave(waveID)
}

// CreateAllocationPolicyRule creates a new allocation policy rule.
func (c *AllocationPolicyController) CreateAllocationPolicyRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	return c.uc.CreateRule(input)
}

// UpdateAllocationPolicyRule updates an existing allocation policy rule.
func (c *AllocationPolicyController) UpdateAllocationPolicyRule(input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	return c.uc.UpdateRule(input)
}

// DeleteAllocationPolicyRule deletes an allocation policy rule by ID.
func (c *AllocationPolicyController) DeleteAllocationPolicyRule(ruleID uint) error {
	return c.uc.DeleteRule(ruleID)
}

// ListAllocationPolicyRules lists all allocation policy rules for a wave.
func (c *AllocationPolicyController) ListAllocationPolicyRules(waveID uint) ([]dto.AllocationPolicyRuleDTO, error) {
	return c.uc.ListRulesByWave(waveID)
}
