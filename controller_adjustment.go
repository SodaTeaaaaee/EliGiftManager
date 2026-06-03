package main

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	db "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

type AdjustmentController struct {
	adjustmentUC        app.AdjustmentUseCase
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
	gdb                 *gorm.DB
}

func NewAdjustmentController() *AdjustmentController {
	gdb := db.GetDB()
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
	return &AdjustmentController{
		adjustmentUC:        app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo),
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc)),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo),
		snapshotSvc:         snapshotSvc,
		gdb:                 gdb,
	}
}

func (c *AdjustmentController) RecordAdjustment(input dto.RecordAdjustmentInput) (dto.FulfillmentAdjustmentDTO, error) {
	ctx := appContext
	preSnapshot, err := c.captureBaselineSnapshot(input.WaveID)
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}

	var adj *domain.FulfillmentAdjustment
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		adjustmentRepo := repos.AdjustmentRepo
		fulfillRepo := repos.FulfillRepo
		waveRepo := repos.WaveRepo
		ruleRepo := repos.RuleRepo
		assignmentRepo := repos.AssignmentRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint
		productRepo := repos.ProductRepo

		adjustmentUC := app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		recordedAdj, recordErr := adjustmentUC.RecordAdjustment(ctx, input)
		if recordErr != nil {
			return recordErr
		}
		adj = recordedAdj

		patchPayload, patchErr := app.BuildAdjustmentPatch("record_adjustment", adj)
		if patchErr != nil {
			return patchErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, adj.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, historyErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  adj.WaveID,
			CommandKind:             domain.CmdRecordAdjustment,
			CommandSummary:          fmt.Sprintf("record adjustment %d (%s) for wave %d", adj.ID, adj.AdjustmentKind, adj.WaveID),
			PatchPayload:            patchPayload,
			InversePatchPayload:     fmt.Sprintf(`{"op":"delete_adjustment","adjustment_id":%d}`, adj.ID),
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return historyErr
	})
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}

	return domainToFulfillmentAdjustmentDTO(*adj), nil
}

func domainToFulfillmentAdjustmentDTO(adj domain.FulfillmentAdjustment) dto.FulfillmentAdjustmentDTO {
	return dto.FulfillmentAdjustmentDTO{
		ID:                        adj.ID,
		WaveID:                    adj.WaveID,
		TargetKind:                adj.TargetKind,
		FulfillmentLineID:         adj.FulfillmentLineID,
		WaveParticipantSnapshotID: adj.WaveParticipantSnapshotID,
		AdjustmentKind:            adj.AdjustmentKind,
		QuantityDelta:             adj.QuantityDelta,
		FromProductID:             adj.FromProductID,
		ToProductID:               adj.ToProductID,
		ReasonCode:                adj.ReasonCode,
		OperatorID:                adj.OperatorID,
		Note:                      adj.Note,
		EvidenceRef:               adj.EvidenceRef,
		CreatedAt:                 adj.CreatedAt,
		UpdatedAt:                 adj.UpdatedAt,
	}
}

func (c *AdjustmentController) ListAdjustmentsByWave(waveID uint) ([]dto.FulfillmentAdjustmentDTO, error) {
	ctx := appContext
	return c.adjustmentUC.ListAdjustmentsByWave(ctx, waveID)
}

func (c *AdjustmentController) captureBaselineSnapshot(waveID uint) (string, error) {
	ctx := appContext
	if waveID == 0 {
		return "", nil
	}
	return c.snapshotSvc.CaptureSnapshot(ctx, waveID)
}
