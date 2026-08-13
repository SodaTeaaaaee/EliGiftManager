package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ShipmentReconciliationQueryUseCase reports per-supplier-order-line shipped/remaining
// quantity so the shipment-backfill UI can display 已发/剩余 before submission.
// Over-ship BLOCKING is already enforced server-side by ShipmentUseCase /
// ShipmentImportUseCase via ShipmentRepository.SumShippedQuantityBySOL (voided
// shipments excluded) — this use case is a read-only display query, not an
// enforcement path.
type ShipmentReconciliationQueryUseCase interface {
	GetSupplierOrderLineShippedSummary(ctx context.Context, orderID uint) ([]dto.SupplierOrderLineShippedDTO, error)
}

type shipmentReconciliationQueryUseCase struct {
	supplierRepo domain.SupplierOrderRepository
	shipmentRepo domain.ShipmentRepository
}

// NewShipmentReconciliationQueryUseCase constructs a ShipmentReconciliationQueryUseCase
// from the existing SupplierOrderRepository and ShipmentRepository ports only — no new
// domain port is introduced.
func NewShipmentReconciliationQueryUseCase(
	supplierRepo domain.SupplierOrderRepository,
	shipmentRepo domain.ShipmentRepository,
) ShipmentReconciliationQueryUseCase {
	return &shipmentReconciliationQueryUseCase{
		supplierRepo: supplierRepo,
		shipmentRepo: shipmentRepo,
	}
}

// GetSupplierOrderLineShippedSummary lists the given supplier order's lines and, for
// each line, computes ShippedQuantity via SumShippedQuantityBySOL (voided shipments
// excluded) and RemainingQuantity = cap - ShippedQuantity, where cap is
// AcceptedQuantity when recorded, otherwise SubmittedQuantity. RemainingQuantity is
// defensively clamped at 0 so the UI never renders a negative "remaining" value.
func (uc *shipmentReconciliationQueryUseCase) GetSupplierOrderLineShippedSummary(ctx context.Context, orderID uint) ([]dto.SupplierOrderLineShippedDTO, error) {
	order, err := uc.supplierRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find supplier order %d: %w", orderID, err)
	}

	lines, err := uc.supplierRepo.ListLinesByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list lines for supplier order %d: %w", orderID, err)
	}

	result := make([]dto.SupplierOrderLineShippedDTO, len(lines))
	for i, line := range lines {
		shipped, sumErr := uc.shipmentRepo.SumShippedQuantityBySOL(ctx, line.ID)
		if sumErr != nil {
			return nil, fmt.Errorf("sum shipped quantity for supplier order line %d: %w", line.ID, sumErr)
		}

		remaining := shippedQuantityCap(line) - shipped
		if remaining < 0 {
			remaining = 0
		}

		result[i] = dto.SupplierOrderLineShippedDTO{
			LineID:            line.ID,
			FulfillmentLineID: line.FulfillmentLineID,
			BatchNo:           order.BatchNo,
			SupplierLineNo:    line.SupplierLineNo,
			SupplierSKU:       line.SupplierSKU,
			SubmittedQuantity: line.SubmittedQuantity,
			AcceptedQuantity:  line.AcceptedQuantity,
			ShippedQuantity:   shipped,
			RemainingQuantity: remaining,
		}
	}

	return result, nil
}
