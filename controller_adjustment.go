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
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)
	return &AdjustmentController{
		adjustmentUC:        app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo),
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo),
		snapshotSvc:         snapshotSvc,
		gdb:                 gdb,
	}
}

func (c *AdjustmentController) RecordAdjustment(input dto.RecordAdjustmentInput) (dto.FulfillmentAdjustmentDTO, error) {
	preSnapshot, err := c.captureBaselineSnapshot(input.WaveID)
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}

	var adj *domain.FulfillmentAdjustment
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		adjustmentUC := app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo)

		recordedAdj, recordErr := adjustmentUC.RecordAdjustment(input)
		if recordErr != nil {
			return recordErr
		}
		adj = recordedAdj

		patchPayload, patchErr := app.BuildAdjustmentPatch("record_adjustment", adj)
		if patchErr != nil {
			return patchErr
		}

		_, historyErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                 adj.WaveID,
			CommandKind:            domain.CmdRecordAdjustment,
			CommandSummary:         fmt.Sprintf("record adjustment %d (%s) for wave %d", adj.ID, adj.AdjustmentKind, adj.WaveID),
			PatchPayload:           patchPayload,
			InversePatchPayload:    fmt.Sprintf(`{"op":"delete_adjustment","adjustment_id":%d}`, adj.ID),
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:         projHashSvc.ComputeHash(adj.WaveID),
		})
		return historyErr
	})
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}

	return dto.FulfillmentAdjustmentDTO{
		ID:                        adj.ID,
		WaveID:                    adj.WaveID,
		TargetKind:                adj.TargetKind,
		FulfillmentLineID:         adj.FulfillmentLineID,
		WaveParticipantSnapshotID: adj.WaveParticipantSnapshotID,
		AdjustmentKind:            adj.AdjustmentKind,
		QuantityDelta:             adj.QuantityDelta,
		ReasonCode:                adj.ReasonCode,
		OperatorID:                adj.OperatorID,
		Note:                      adj.Note,
		EvidenceRef:               adj.EvidenceRef,
		CreatedAt:                 adj.CreatedAt,
		UpdatedAt:                 adj.UpdatedAt,
	}, nil
}

func (c *AdjustmentController) ListAdjustmentsByWave(waveID uint) ([]dto.FulfillmentAdjustmentDTO, error) {
	return c.adjustmentUC.ListAdjustmentsByWave(waveID)
}

func (c *AdjustmentController) captureBaselineSnapshot(waveID uint) (string, error) {
	if waveID == 0 {
		return "", nil
	}
	return c.snapshotSvc.CaptureSnapshot(waveID)
}
