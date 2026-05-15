package main

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// AllocationPolicyController exposes allocation-policy Wails bindings.
type AllocationPolicyController struct {
	uc                  app.AllocationPolicyUseCase
	ruleRepo            domain.AllocationPolicyRuleRepository
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
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
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)
	snapshotSvc := app.NewWaveSnapshotService(ruleRepo, adjustmentRepo, assignmentRepo)

	return &AllocationPolicyController{
		uc:                  app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo),
		ruleRepo:            ruleRepo,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo),
	}
}

// ReconcileWave idempotently rebuilds policy-driven fulfillment lines for the wave,
// replaying any recorded adjustments.
func (c *AllocationPolicyController) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	return c.uc.ReconcileWave(waveID)
}

// CreateAllocationPolicyRule creates a new allocation policy rule.
func (c *AllocationPolicyController) CreateAllocationPolicyRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	rule, err := c.uc.CreateRule(input)
	if err != nil {
		return nil, err
	}

	ruleData, _ := json.Marshal(rule)
	_, _ = c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              input.WaveID,
		CommandKind:         domain.CmdCreateRule,
		CommandSummary:      fmt.Sprintf("create allocation rule %d for wave %d", rule.ID, input.WaveID),
		PatchPayload:        fmt.Sprintf(`{"op":"restore_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, input.WaveID, ruleData),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID),
		ProjectionHash:      c.projHashSvc.ComputeHash(input.WaveID),
	})
	return rule, nil
}

// UpdateAllocationPolicyRule updates an existing allocation policy rule.
func (c *AllocationPolicyController) UpdateAllocationPolicyRule(input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	oldRule, _ := c.ruleRepo.FindByID(input.ID)
	var oldData []byte
	if oldRule != nil {
		oldData, _ = json.Marshal(oldRule)
	}

	rule, err := c.uc.UpdateRule(input)
	if err != nil {
		return nil, err
	}

	newData, _ := json.Marshal(rule)
	_, _ = c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              rule.WaveID,
		CommandKind:         domain.CmdUpdateRule,
		CommandSummary:      fmt.Sprintf("update allocation rule %d for wave %d", rule.ID, rule.WaveID),
		PatchPayload:        fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, rule.WaveID, newData),
		InversePatchPayload: fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, rule.WaveID, oldData),
		ProjectionHash:      c.projHashSvc.ComputeHash(rule.WaveID),
	})
	return rule, nil
}

// DeleteAllocationPolicyRule deletes an allocation policy rule by ID.
func (c *AllocationPolicyController) DeleteAllocationPolicyRule(ruleID uint) error {
	rule, err := c.ruleRepo.FindByID(ruleID)
	if err != nil {
		return err
	}

	if err := c.uc.DeleteRule(ruleID); err != nil {
		return err
	}

	if rule != nil {
		ruleData, _ := json.Marshal(rule)
		_, _ = c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
			WaveID:              rule.WaveID,
			CommandKind:         domain.CmdDeleteRule,
			CommandSummary:      fmt.Sprintf("delete allocation rule %d from wave %d", ruleID, rule.WaveID),
			PatchPayload:        fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID),
			InversePatchPayload: fmt.Sprintf(`{"op":"restore_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, rule.WaveID, ruleData),
			ProjectionHash:      c.projHashSvc.ComputeHash(rule.WaveID),
		})
	}
	return nil
}

// ListAllocationPolicyRules lists all allocation policy rules for a wave.
func (c *AllocationPolicyController) ListAllocationPolicyRules(waveID uint) ([]dto.AllocationPolicyRuleDTO, error) {
	return c.uc.ListRulesByWave(waveID)
}

