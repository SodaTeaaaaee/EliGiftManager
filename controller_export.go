package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ExportController exposes supplier-order-export Wails bindings.
type ExportController struct {
	exportUC     app.ExportUseCase
	supplierRepo domain.SupplierOrderRepository
}

func NewExportController() *ExportController {
	gdb := db.GetDB()
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyPinRepo := infra.NewHistoryPinRepository(gdb)

	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)

	return &ExportController{
		exportUC:     app.NewExportUseCase(supplierRepo, fulfillRepo, basisStamp),
		supplierRepo: supplierRepo,
	}
}

// ExportSupplierOrder exports a supplier order from the given wave.
func (c *ExportController) ExportSupplierOrder(waveID uint) (dto.SupplierOrderDTO, error) {
	so, err := c.exportUC.ExportSupplierOrder(waveID)
	if err != nil {
		return dto.SupplierOrderDTO{}, err
	}
	return domainToSupplierOrderDTO(so), nil
}

// ListSupplierOrders lists all supplier orders.
func (c *ExportController) ListSupplierOrders() ([]dto.SupplierOrderDTO, error) {
	orders, err := c.supplierRepo.List()
	if err != nil {
		return nil, err
	}
	result := make([]dto.SupplierOrderDTO, len(orders))
	for i, order := range orders {
		result[i] = domainToSupplierOrderDTO(&order)
	}
	return result, nil
}

// GetSupplierOrderByWave returns the most recent supplier order for the given wave, or empty DTO if none.
func (c *ExportController) GetSupplierOrderByWave(waveID uint) (dto.SupplierOrderDTO, error) {
	orders, err := c.supplierRepo.ListByWave(waveID)
	if err != nil {
		return dto.SupplierOrderDTO{}, err
	}
	if len(orders) == 0 {
		return dto.SupplierOrderDTO{}, nil
	}
	var latest *domain.SupplierOrder
	for i := range orders {
		if latest == nil || orders[i].ID > latest.ID {
			latest = &orders[i]
		}
	}
	return domainToSupplierOrderDTO(latest), nil
}

// ListLinesBySupplierOrder returns all lines for the given supplier order.
func (c *ExportController) ListLinesBySupplierOrder(orderID uint) ([]dto.SupplierOrderLineDTO, error) {
	lines, err := c.supplierRepo.ListLinesByOrder(orderID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.SupplierOrderLineDTO, len(lines))
	for i, line := range lines {
		result[i] = domainToSupplierOrderLineDTO(&line)
	}
	return result, nil
}

// domainToSupplierOrderLineDTO converts a domain SupplierOrderLine to a DTO.
func domainToSupplierOrderLineDTO(line *domain.SupplierOrderLine) dto.SupplierOrderLineDTO {
	if line == nil {
		return dto.SupplierOrderLineDTO{}
	}
	v1 := line.SupplierLineNo
	supplierLineNo := &v1
	v2 := line.AcceptedQuantity
	acceptedQuantity := &v2
	return dto.SupplierOrderLineDTO{
		ID:                line.ID,
		SupplierOrderID:   line.SupplierOrderID,
		FulfillmentLineID: line.FulfillmentLineID,
		SupplierLineNo:    supplierLineNo,
		SupplierSKU:       line.SupplierSKU,
		SubmittedQuantity: line.SubmittedQuantity,
		AcceptedQuantity:  acceptedQuantity,
		Status:            line.Status,
		ExtraData:         line.ExtraData,
		CreatedAt:         line.CreatedAt,
		UpdatedAt:         line.UpdatedAt,
	}
}

// domainToSupplierOrderDTO converts a domain SupplierOrder to a DTO.
func domainToSupplierOrderDTO(so *domain.SupplierOrder) dto.SupplierOrderDTO {
	if so == nil {
		return dto.SupplierOrderDTO{}
	}
	return dto.SupplierOrderDTO{
		ID:                   so.ID,
		WaveID:               so.WaveID,
		SupplierPlatform:     so.SupplierPlatform,
		TemplateID:           so.TemplateID,
		BatchNo:              so.BatchNo,
		ExternalOrderNo:      so.ExternalOrderNo,
		SubmissionMode:       so.SubmissionMode,
		SubmittedAt:          so.SubmittedAt,
		Status:               so.Status,
		RequestPayload:       so.RequestPayload,
		ResponsePayload:      so.ResponsePayload,
		BasisHistoryNodeID:   so.BasisHistoryNodeID,
		BasisProjectionHash:  so.BasisProjectionHash,
		BasisPayloadSnapshot: so.BasisPayloadSnapshot,
		ExtraData:            so.ExtraData,
		CreatedAt:            so.CreatedAt,
		UpdatedAt:            so.UpdatedAt,
	}
}
