package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// GetSupplierOrderLineShippedSummary returns, for every line of the given supplier
// order, the shipped/remaining quantity so the shipment-backfill UI can display
// 已发/剩余 before submission. Over-ship BLOCKING is already enforced server-side by
// CreateShipment/ImportShipments — this is a display-only read.
func (c *ShipmentController) GetSupplierOrderLineShippedSummary(orderID uint) ([]dto.SupplierOrderLineShippedDTO, error) {
	ctx := appContext
	gdb := db.GetDB()
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	shipmentRepo := infra.NewShipmentRepository(gdb)

	uc := app.NewShipmentReconciliationQueryUseCase(supplierRepo, shipmentRepo)
	return uc.GetSupplierOrderLineShippedSummary(ctx, orderID)
}
