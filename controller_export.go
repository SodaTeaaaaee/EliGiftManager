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

	return &ExportController{
		exportUC:     app.NewExportUseCase(supplierRepo, fulfillRepo),
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
	// SupplierOrderRepository does not have a List() method, so list by wave 0
	// (which returns empty) as a placeholder. The use case layer will add a
	// proper List method when needed.
	return []dto.SupplierOrderDTO{}, nil
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
