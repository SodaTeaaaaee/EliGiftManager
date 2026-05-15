package main

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	db "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type AdjustmentController struct {
	adjustmentUC        app.AdjustmentUseCase
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
}

func NewAdjustmentController() *AdjustmentController {
	gdb := db.GetDB()
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)
	return &AdjustmentController{
		adjustmentUC:        app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo),
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo),
	}
}

func (c *AdjustmentController) RecordAdjustment(input dto.RecordAdjustmentInput) (dto.FulfillmentAdjustmentDTO, error) {
	adj, err := c.adjustmentUC.RecordAdjustment(input)
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}

	_, _ = c.historyRecordingSvc.RecordNode(app.RecordNodeInput{
		WaveID:              adj.WaveID,
		CommandKind:         domain.CmdRecordAdjustment,
		CommandSummary:      fmt.Sprintf("record adjustment %d (%s) for wave %d", adj.ID, adj.AdjustmentKind, adj.WaveID),
		PatchPayload:        buildAdjustmentPatch("record_adjustment", adj),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_adjustment","adjustment_id":%d}`, adj.ID),
		ProjectionHash:      c.projHashSvc.ComputeHash(adj.WaveID),
	})

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

func buildAdjustmentPatch(op string, adj *domain.FulfillmentAdjustment) string {
	data, _ := json.Marshal(adj)
	return fmt.Sprintf(`{"op":%q,"adjustment_id":%d,"wave_id":%d,"data":%s}`, op, adj.ID, adj.WaveID, data)
}

