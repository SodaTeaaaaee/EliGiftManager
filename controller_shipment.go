package main

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// ShipmentController exposes shipment-management Wails bindings.
type ShipmentController struct {
	shipmentUC          app.ShipmentUseCase
	shipmentRepo        domain.ShipmentRepository
	gdb                 *gorm.DB
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
}

func NewShipmentController() *ShipmentController {
	gdb := db.GetDB()
	shipmentRepo := infra.NewShipmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyPinRepo := infra.NewHistoryPinRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)

	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)

	return &ShipmentController{
		shipmentUC:          app.NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp),
		shipmentRepo:        shipmentRepo,
		gdb:                 gdb,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo),
		snapshotSvc:         snapshotSvc,
	}
}

// CreateShipment creates a shipment with its lines.
func (c *ShipmentController) CreateShipment(input dto.CreateShipmentInput) (dto.ShipmentDTO, error) {
	preSnapshot, err := c.snapshotSvc.CaptureSnapshotForSupplierOrder(input.SupplierOrderID)
	if err != nil {
		return dto.ShipmentDTO{}, err
	}

	var shipment *domain.Shipment
	var lines []domain.ShipmentLine
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		shipmentRepo := infra.NewShipmentRepository(tx)
		supplierRepo := infra.NewSupplierOrderRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		closureDecisionRepo := infra.NewClosureDecisionRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyPinRepo := infra.NewHistoryPinRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		shipmentUC := app.NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		createdShipment, createdLines, createErr := shipmentUC.CreateShipment(input)
		if createErr != nil {
			return createErr
		}
		shipment = createdShipment
		lines = createdLines

		supplierOrder, findErr := supplierRepo.FindByID(input.SupplierOrderID)
		if findErr != nil {
			return findErr
		}

		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  supplierOrder.WaveID,
			CommandKind:             domain.CmdCreateShipment,
			CommandSummary:          fmt.Sprintf("create shipment %d for wave %d", shipment.ID, supplierOrder.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHashSvc.ComputeHash(supplierOrder.WaveID),
		})
		return recordErr
	})
	if err != nil {
		return dto.ShipmentDTO{}, err
	}

	result := domainToShipmentDTO(shipment)
	result.Lines = make([]dto.ShipmentLineDTO, len(lines))
	for i, l := range lines {
		result.Lines[i] = domainToShipmentLineDTO(&l)
	}
	return result, nil
}

// ListShipmentsByWave lists all shipments for a given wave.
func (c *ShipmentController) ListShipmentsByWave(waveID uint) ([]dto.ShipmentDTO, error) {
	shipments, err := c.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ShipmentDTO, len(shipments))
	for i, s := range shipments {
		shipmentDTO := domainToShipmentDTO(&s)
		lines, err := c.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return nil, err
		}
		shipmentDTO.Lines = make([]dto.ShipmentLineDTO, len(lines))
		for j, l := range lines {
			shipmentDTO.Lines[j] = domainToShipmentLineDTO(&l)
		}
		result[i] = shipmentDTO
	}
	return result, nil
}

// domainToShipmentDTO converts a domain Shipment to a DTO.
func domainToShipmentDTO(s *domain.Shipment) dto.ShipmentDTO {
	if s == nil {
		return dto.ShipmentDTO{}
	}
	return dto.ShipmentDTO{
		ID:                   s.ID,
		SupplierOrderID:      s.SupplierOrderID,
		SupplierPlatform:     s.SupplierPlatform,
		ShipmentNo:           s.ShipmentNo,
		ExternalShipmentNo:   s.ExternalShipmentNo,
		CarrierCode:          s.CarrierCode,
		CarrierName:          s.CarrierName,
		TrackingNo:           s.TrackingNo,
		Status:               s.Status,
		ShippedAt:            s.ShippedAt,
		BasisHistoryNodeID:   s.BasisHistoryNodeID,
		BasisProjectionHash:  s.BasisProjectionHash,
		BasisPayloadSnapshot: s.BasisPayloadSnapshot,
		ExtraData:            s.ExtraData,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// domainToShipmentLineDTO converts a domain ShipmentLine to a DTO.
func domainToShipmentLineDTO(l *domain.ShipmentLine) dto.ShipmentLineDTO {
	if l == nil {
		return dto.ShipmentLineDTO{}
	}
	return dto.ShipmentLineDTO{
		ID:                  l.ID,
		ShipmentID:          l.ShipmentID,
		SupplierOrderLineID: l.SupplierOrderLineID,
		FulfillmentLineID:   l.FulfillmentLineID,
		Quantity:            l.Quantity,
		CreatedAt:           l.CreatedAt,
	}
}
