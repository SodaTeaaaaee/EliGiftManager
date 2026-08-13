package controller

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// AllocationPolicyController exposes allocation-policy Wails bindings.
type AllocationPolicyController struct {
	uc                  app.AllocationPolicyUseCase
	ruleRepo            domain.AllocationPolicyRuleRepository
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
	gdb                 *gorm.DB
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
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)

	return &AllocationPolicyController{
		uc:                  app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo),
		ruleRepo:            ruleRepo,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc)),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo),
		snapshotSvc:         snapshotSvc,
		gdb:                 gdb,
	}
}

// ReconcileWave idempotently rebuilds policy-driven fulfillment lines for the wave,
// replaying any recorded adjustments.
func (c *AllocationPolicyController) ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		return nil, err
	}

	var result *dto.ReconcileResultDTO
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		ruleRepo := repos.RuleRepo
		fulfillRepo := repos.FulfillRepo
		waveRepo := repos.WaveRepo
		adjustmentRepo := repos.AdjustmentRepo
		demandRepo := repos.DemandRepo
		assignmentRepo := repos.AssignmentRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		allocationUC := app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		reconcileResult, reconcileErr := allocationUC.ReconcileWave(ctx, waveID)
		if reconcileErr != nil {
			return reconcileErr
		}
		result = reconcileResult

		postSnapshot, snapErr := snapshotSvc.CaptureSnapshot(ctx, waveID)
		if snapErr != nil {
			return snapErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, waveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  waveID,
			CommandKind:             domain.CmdReconcileWave,
			CommandSummary:          fmt.Sprintf("reconcile wave %d (%d created, %d deleted)", waveID, result.Created, result.Deleted),
			PatchPayload:            fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
			InversePatchPayload:     fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
			CheckpointHint:          true,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateAllocationPolicyRule creates a new allocation policy rule.
func (c *AllocationPolicyController) CreateAllocationPolicyRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return nil, err
	}

	var rule *dto.AllocationPolicyRuleDTO
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		ruleRepo := repos.RuleRepo
		fulfillRepo := repos.FulfillRepo
		waveRepo := repos.WaveRepo
		adjustmentRepo := repos.AdjustmentRepo
		demandRepo := repos.DemandRepo
		assignmentRepo := repos.AssignmentRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		allocationUC := app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		createdRule, createErr := allocationUC.CreateRule(ctx, input)
		if createErr != nil {
			return createErr
		}
		rule = createdRule

		domainRule, findErr := ruleRepo.FindByID(ctx, createdRule.ID)
		if findErr != nil {
			return findErr
		}
		patchPayload, patchErr := app.BuildRuleRestorePatch("restore_rule", domainRule)
		if patchErr != nil {
			return patchErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdCreateRule,
			CommandSummary:          fmt.Sprintf("create allocation rule %d for wave %d", createdRule.ID, input.WaveID),
			PatchPayload:            patchPayload,
			InversePatchPayload:     fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, createdRule.ID),
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// UpdateAllocationPolicyRule updates an existing allocation policy rule.
func (c *AllocationPolicyController) UpdateAllocationPolicyRule(input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error) {
	ctx := appContext
	oldRule, err := c.ruleRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if oldRule == nil {
		return nil, fmt.Errorf("allocation rule %d not found", input.ID)
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, oldRule.WaveID)
	if err != nil {
		return nil, err
	}
	oldPayload, err := app.BuildRuleUpdatePatch(oldRule)
	if err != nil {
		return nil, err
	}

	var rule *dto.AllocationPolicyRuleDTO
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		ruleRepo := repos.RuleRepo
		fulfillRepo := repos.FulfillRepo
		waveRepo := repos.WaveRepo
		adjustmentRepo := repos.AdjustmentRepo
		demandRepo := repos.DemandRepo
		assignmentRepo := repos.AssignmentRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		allocationUC := app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		updatedRule, updateErr := allocationUC.UpdateRule(ctx, input)
		if updateErr != nil {
			return updateErr
		}
		rule = updatedRule

		domainRule, findErr := ruleRepo.FindByID(ctx, updatedRule.ID)
		if findErr != nil {
			return findErr
		}
		newPayload, patchErr := app.BuildRuleUpdatePatch(domainRule)
		if patchErr != nil {
			return patchErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, updatedRule.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  updatedRule.WaveID,
			CommandKind:             domain.CmdUpdateRule,
			CommandSummary:          fmt.Sprintf("update allocation rule %d for wave %d", updatedRule.ID, updatedRule.WaveID),
			PatchPayload:            newPayload,
			InversePatchPayload:     oldPayload,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// DeleteAllocationPolicyRule deletes an allocation policy rule by ID.
func (c *AllocationPolicyController) DeleteAllocationPolicyRule(ruleID uint) error {
	ctx := appContext
	rule, err := c.ruleRepo.FindByID(ctx, ruleID)
	if err != nil {
		return err
	}
	if rule == nil {
		return fmt.Errorf("allocation rule %d not found", ruleID)
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, rule.WaveID)
	if err != nil {
		return err
	}
	inversePayload, err := app.BuildRuleRestorePatch("restore_rule", rule)
	if err != nil {
		return err
	}

	return c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		ruleRepo := repos.RuleRepo
		fulfillRepo := repos.FulfillRepo
		waveRepo := repos.WaveRepo
		adjustmentRepo := repos.AdjustmentRepo
		demandRepo := repos.DemandRepo
		assignmentRepo := repos.AssignmentRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		allocationUC := app.NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjustmentRepo, demandRepo, assignmentRepo, productRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		if deleteErr := allocationUC.DeleteRule(ctx, ruleID); deleteErr != nil {
			return deleteErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, rule.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  rule.WaveID,
			CommandKind:             domain.CmdDeleteRule,
			CommandSummary:          fmt.Sprintf("delete allocation rule %d from wave %d", ruleID, rule.WaveID),
			PatchPayload:            fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID),
			InversePatchPayload:     inversePayload,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
}

// ListAllocationPolicyRules lists all allocation policy rules for a wave.
func (c *AllocationPolicyController) ListAllocationPolicyRules(waveID uint) ([]dto.AllocationPolicyRuleDTO, error) {
	ctx := appContext
	return c.uc.ListRulesByWave(ctx, waveID)
}
