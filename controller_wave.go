package main

import (
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// WaveController exposes wave-management Wails bindings.
type WaveController struct {
	waveUC         app.WaveUseCase
	allocationUC   app.AllocationUseCase
	fulfillRepo    domain.FulfillmentLineRepository
	supplierRepo   domain.SupplierOrderRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	demandRepo     domain.DemandDocumentRepository
	shipmentRepo   domain.ShipmentRepository
	overviewProjUC app.WaveOverviewProjectionUseCase
	undoRedoUC     app.UndoRedoUseCase
}

func NewWaveController() *WaveController {
	gdb := db.GetDB()
	waveRepo := infra.NewWaveRepository(gdb)
	demandRepo := infra.NewDemandRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	shipmentRepo := infra.NewShipmentRepository(gdb)
	channelSyncRepo := infra.NewChannelSyncRepository(gdb)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)

	basisDriftUC := app.NewBasisDriftDetectionUseCase(supplierRepo, shipmentRepo, channelSyncRepo)
	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)

	return &WaveController{
		waveUC:         app.NewWaveUseCase(waveRepo),
		allocationUC:   app.NewAllocationUseCase(demandRepo, ruleRepo, fulfillRepo, assignmentRepo),
		fulfillRepo:    fulfillRepo,
		supplierRepo:   supplierRepo,
		assignmentRepo: assignmentRepo,
		demandRepo:     demandRepo,
		shipmentRepo:   shipmentRepo,
		overviewProjUC: app.NewWaveOverviewProjectionUseCase(channelSyncRepo, closureDecisionRepo, basisDriftUC, historyHeadUC),
		undoRedoUC:     app.NewUndoRedoUseCase(historyScopeRepo, historyNodeRepo),
	}
}

// CreateWave creates a new wave.
func (c *WaveController) CreateWave(input dto.CreateWaveInput) (dto.WaveDTO, error) {
	wave := domain.Wave{
		Name: input.Name,
	}
	if err := c.waveUC.CreateWave(&wave); err != nil {
		return dto.WaveDTO{}, err
	}
	return domainToWaveDTO(&wave), nil
}

// ListWaves lists all waves.
func (c *WaveController) ListWaves() ([]dto.WaveDTO, error) {
	waves, err := c.waveUC.ListWaves()
	if err != nil {
		return nil, err
	}
	result := make([]dto.WaveDTO, len(waves))
	for i, w := range waves {
		result[i] = domainToWaveDTO(&w)
	}
	return result, nil
}

// GetWave returns a single wave by ID.
func (c *WaveController) GetWave(id uint) (dto.WaveDTO, error) {
	w, err := c.waveUC.GetWave(id)
	if err != nil {
		return dto.WaveDTO{}, err
	}
	return domainToWaveDTO(w), nil
}

// GetWaveOverview returns aggregated wave overview data.
func (c *WaveController) GetWaveOverview(waveID uint) (dto.WaveOverviewDTO, error) {
	w, err := c.waveUC.GetWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	fulfillLines, err := c.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	supplierOrders, err := c.supplierRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	// Count accepted demand lines for this wave via assignments
	docs, err := c.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	demandCount := 0
	for _, doc := range docs {
		lines, err := c.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, line := range lines {
			if line.RoutingDisposition == "accepted" {
				demandCount++
			}
		}
	}

	// Collect shipment stats
	shipments, err := c.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	shipmentCount := len(shipments)

	trackedFulfillmentCount := 0
	trackedSet := make(map[uint]bool)
	for _, s := range shipments {
		if s.TrackingNo == "" {
			continue
		}
		lines, err := c.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, l := range lines {
			if !trackedSet[l.FulfillmentLineID] {
				trackedSet[l.FulfillmentLineID] = true
				trackedFulfillmentCount++
			}
		}
	}

	base := dto.WaveOverviewDTO{
		Wave:                    domainToWaveDTO(w),
		DemandCount:             demandCount,
		FulfillmentCount:        len(fulfillLines),
		SupplierOrderCount:      len(supplierOrders),
		ShipmentCount:           shipmentCount,
		TrackedFulfillmentCount: trackedFulfillmentCount,
	}
	return c.overviewProjUC.ProjectWaveOverview(base)
}

// AssignDemandToWave assigns a demand document to a wave.
func (c *WaveController) AssignDemandToWave(waveID uint, demandDocumentID uint) error {
	// Validate wave existence
	if _, err := c.waveUC.GetWave(waveID); err != nil {
		return err
	}
	// Validate demand document existence
	if _, err := c.demandRepo.FindByID(demandDocumentID); err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	assignment := &domain.WaveDemandAssignment{
		WaveID:           waveID,
		DemandDocumentID: demandDocumentID,
		AcceptedAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	return c.assignmentRepo.Create(assignment)
}

// ApplyAllocationRules applies allocation policy rules to the given wave
// and returns the generated FulfillmentLines.
func (c *WaveController) ApplyAllocationRules(waveID uint) ([]dto.FulfillmentLineDTO, error) {
	lines, err := c.allocationUC.ApplyRules(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.FulfillmentLineDTO, len(lines))
	for i := range lines {
		result[i] = domainToFulfillmentLineDTO(&lines[i])
	}
	return result, nil
}

// domainToWaveDTO converts a domain Wave to a DTO.
func domainToWaveDTO(w *domain.Wave) dto.WaveDTO {
	if w == nil {
		return dto.WaveDTO{}
	}
	return dto.WaveDTO{
		ID:               w.ID,
		WaveNo:           w.WaveNo,
		Name:             w.Name,
		WaveType:         w.WaveType,
		LifecycleStage:   w.LifecycleStage,
		ProgressSnapshot: w.ProgressSnapshot,
		Notes:            w.Notes,
		LevelTags:        w.LevelTags,
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
}

// domainToFulfillmentLineDTO converts a domain FulfillmentLine to a DTO.
func domainToFulfillmentLineDTO(fl *domain.FulfillmentLine) dto.FulfillmentLineDTO {
	if fl == nil {
		return dto.FulfillmentLineDTO{}
	}
	return dto.FulfillmentLineDTO{
		ID:                        fl.ID,
		WaveID:                    fl.WaveID,
		CustomerProfileID:         fl.CustomerProfileID,
		WaveParticipantSnapshotID: fl.WaveParticipantSnapshotID,
		ProductID:                 fl.ProductID,
		DemandDocumentID:          fl.DemandDocumentID,
		DemandLineID:              fl.DemandLineID,
		CustomerAddressID:         fl.CustomerAddressID,
		Quantity:                  fl.Quantity,
		AllocationState:           fl.AllocationState,
		AddressState:              fl.AddressState,
		SupplierState:             fl.SupplierState,
		ChannelSyncState:          fl.ChannelSyncState,
		LineReason:                fl.LineReason,
		GeneratedBy:               fl.GeneratedBy,
		ExtraData:                 fl.ExtraData,
		CreatedAt:                 fl.CreatedAt,
		UpdatedAt:                 fl.UpdatedAt,
	}
}

// UndoWaveAction undoes the last action for the given wave.
func (c *WaveController) UndoWaveAction(waveID uint) (string, error) {
	return c.undoRedoUC.Undo(waveID)
}

// RedoWaveAction redoes the last undone action for the given wave.
func (c *WaveController) RedoWaveAction(waveID uint) (string, error) {
	return c.undoRedoUC.Redo(waveID)
}
