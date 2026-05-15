package main

import (
	"encoding/json"
	"fmt"
	"log"

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
	snapshotSvc         *app.WaveSnapshotService
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
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)

	return &AllocationPolicyController{
		uc:                  app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo),
		ruleRepo:            ruleRepo,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo),
		snapshotSvc:         snapshotSvc,
	}
}

// ReconcileWave idempotently rebuilds policy-driven fulfillment lines for the wave,
// replaying any recorded adjustments.
func (c *AllocationPolicyController) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	preSnapshot, _ := c.snapshotSvc.CaptureSnapshot(waveID)

	result, err := c.uc.ReconcileWave(waveID)
	if err != nil {
		return nil, err
	}

	postSnapshot, _ := c.snapshotSvc.CaptureSnapshot(waveID)
	if _, err := c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         domain.CmdReconcileWave,
		CommandSummary:      fmt.Sprintf("reconcile wave %d (%d created, %d deleted)", waveID, result.Created, result.Deleted),
		PatchPayload:        fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
		InversePatchPayload: fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
		CheckpointHint:      true,
		ProjectionHash:      c.projHashSvc.ComputeHash(waveID),
	}); err != nil {
		log.Printf("WARNING: history recording failed for reconcile_wave wave %d: %v", waveID, err)
	}

	return result, nil
}

// CreateAllocationPolicyRule creates a new allocation policy rule.
func (c *AllocationPolicyController) CreateAllocationPolicyRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	rule, err := c.uc.CreateRule(input)
	if err != nil {
		return nil, err
	}

	ruleData, _ := json.Marshal(rule)
	if _, err := c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              input.WaveID,
		CommandKind:         domain.CmdCreateRule,
		CommandSummary:      fmt.Sprintf("create allocation rule %d for wave %d", rule.ID, input.WaveID),
		PatchPayload:        fmt.Sprintf(`{"op":"restore_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, input.WaveID, ruleData),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID),
		ProjectionHash:      c.projHashSvc.ComputeHash(input.WaveID),
	}); err != nil {
		log.Printf("WARNING: history recording failed for create_rule wave %d rule %d: %v", input.WaveID, rule.ID, err)
	}
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
	if _, err := c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              rule.WaveID,
		CommandKind:         domain.CmdUpdateRule,
		CommandSummary:      fmt.Sprintf("update allocation rule %d for wave %d", rule.ID, rule.WaveID),
		PatchPayload:        fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, rule.WaveID, newData),
		InversePatchPayload: fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, rule.WaveID, oldData),
		ProjectionHash:      c.projHashSvc.ComputeHash(rule.WaveID),
	}); err != nil {
		log.Printf("WARNING: history recording failed for update_rule wave %d rule %d: %v", rule.WaveID, rule.ID, err)
	}
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
		if _, err := c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
			WaveID:              rule.WaveID,
			CommandKind:         domain.CmdDeleteRule,
			CommandSummary:      fmt.Sprintf("delete allocation rule %d from wave %d", ruleID, rule.WaveID),
			PatchPayload:        fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID),
			InversePatchPayload: fmt.Sprintf(`{"op":"restore_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, rule.WaveID, ruleData),
			ProjectionHash:      c.projHashSvc.ComputeHash(rule.WaveID),
		}); err != nil {
			log.Printf("WARNING: history recording failed for delete_rule wave %d rule %d: %v", rule.WaveID, ruleID, err)
		}
	}
	return nil
}

// ListAllocationPolicyRules lists all allocation policy rules for a wave.
func (c *AllocationPolicyController) ListAllocationPolicyRules(waveID uint) ([]dto.AllocationPolicyRuleDTO, error) {
	return c.uc.ListRulesByWave(waveID)
}

