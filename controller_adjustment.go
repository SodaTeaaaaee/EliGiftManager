package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	db "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type AdjustmentController struct {
	adjustmentUC app.AdjustmentUseCase
}

func NewAdjustmentController() *AdjustmentController {
	gdb := db.GetDB()
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	return &AdjustmentController{
		adjustmentUC: app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo),
	}
}

func (c *AdjustmentController) RecordAdjustment(input dto.RecordAdjustmentInput) (dto.FulfillmentAdjustmentDTO, error) {
	adj, err := c.adjustmentUC.RecordAdjustment(input)
	if err != nil {
		return dto.FulfillmentAdjustmentDTO{}, err
	}
	return dto.FulfillmentAdjustmentDTO{
		ID:                adj.ID,
		WaveID:            adj.WaveID,
		FulfillmentLineID: adj.FulfillmentLineID,
		AdjustmentKind:    adj.AdjustmentKind,
		QuantityDelta:     adj.QuantityDelta,
		ReasonCode:        adj.ReasonCode,
		OperatorID:        adj.OperatorID,
		Note:              adj.Note,
		EvidenceRef:       adj.EvidenceRef,
		CreatedAt:         adj.CreatedAt,
		UpdatedAt:         adj.UpdatedAt,
	}, nil
}

func (c *AdjustmentController) ListAdjustmentsByWave(waveID uint) ([]dto.FulfillmentAdjustmentDTO, error) {
	return c.adjustmentUC.ListAdjustmentsByWave(waveID)
}
