package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// WaveController exposes wave-management Wails bindings.
type WaveController struct {
	waveUC       app.WaveUseCase
	allocationUC app.AllocationUseCase
	fulfillRepo  domain.FulfillmentLineRepository
	supplierRepo domain.SupplierOrderRepository
}

func NewWaveController() *WaveController {
	gdb := db.GetDB()
	waveRepo := infra.NewWaveRepository(gdb)
	demandRepo := infra.NewDemandRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)

	return &WaveController{
		waveUC:       app.NewWaveUseCase(waveRepo),
		allocationUC: app.NewAllocationUseCase(demandRepo, ruleRepo, fulfillRepo),
		fulfillRepo:  fulfillRepo,
		supplierRepo: supplierRepo,
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

	return dto.WaveOverviewDTO{
		Wave:               domainToWaveDTO(w),
		DemandCount:        0, // DemandDocument not yet directly linked to wave
		FulfillmentCount:   len(fulfillLines),
		SupplierOrderCount: len(supplierOrders),
	}, nil
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
		ID:                       fl.ID,
		WaveID:                   fl.WaveID,
		CustomerProfileID:        fl.CustomerProfileID,
		WaveParticipantSnapshotID: fl.WaveParticipantSnapshotID,
		ProductID:                fl.ProductID,
		DemandDocumentID:         fl.DemandDocumentID,
		DemandLineID:             fl.DemandLineID,
		CustomerAddressID:        fl.CustomerAddressID,
		Quantity:                 fl.Quantity,
		AllocationState:          fl.AllocationState,
		AddressState:             fl.AddressState,
		SupplierState:            fl.SupplierState,
		ChannelSyncState:         fl.ChannelSyncState,
		LineReason:               fl.LineReason,
		ExtraData:                fl.ExtraData,
		CreatedAt:                fl.CreatedAt,
		UpdatedAt:                fl.UpdatedAt,
	}
}
