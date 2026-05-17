package app

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- Shipment ----

type shipmentUseCase struct {
	shipmentRepo domain.ShipmentRepository
	supplierRepo domain.SupplierOrderRepository
	fulfillRepo  domain.FulfillmentLineRepository
	basisStamp   *BasisStampService
}

func NewShipmentUseCase(shipmentRepo domain.ShipmentRepository, supplierRepo domain.SupplierOrderRepository, fulfillRepo domain.FulfillmentLineRepository, basisStamp *BasisStampService) ShipmentUseCase {
	return &shipmentUseCase{
		shipmentRepo: shipmentRepo,
		supplierRepo: supplierRepo,
		fulfillRepo:  fulfillRepo,
		basisStamp:   basisStamp,
	}
}

func (uc *shipmentUseCase) CreateShipment(input dto.CreateShipmentInput) (*domain.Shipment, []domain.ShipmentLine, error) {
	// 1. Empty line check
	if len(input.Lines) == 0 {
		return nil, nil, fmt.Errorf("shipment must have at least one line")
	}

	// 2. Validate supplier order existence
	supplierOrder, err := uc.supplierRepo.FindByID(input.SupplierOrderID)
	if err != nil {
		return nil, nil, fmt.Errorf("supplier order %d not found: %w", input.SupplierOrderID, err)
	}

	// 3. Validate each line (all checks outside transaction)
	for _, li := range input.Lines {
		// Validate supplier order line existence
		sol, err := uc.supplierRepo.FindLineByID(li.SupplierOrderLineID)
		if err != nil {
			return nil, nil, fmt.Errorf("supplier order line %d not found: %w", li.SupplierOrderLineID, err)
		}
		// Validate fulfillment line existence
		fl, err := uc.fulfillRepo.FindByID(li.FulfillmentLineID)
		if err != nil {
			return nil, nil, fmt.Errorf("fulfillment line %d not found: %w", li.FulfillmentLineID, err)
		}
		// Validate supplier order line belongs to this supplier order
		if sol.SupplierOrderID != input.SupplierOrderID {
			return nil, nil, fmt.Errorf("supplier order line %d belongs to order %d, not %d", li.SupplierOrderLineID, sol.SupplierOrderID, input.SupplierOrderID)
		}
		// Validate supplier order line references the correct fulfillment line
		if sol.FulfillmentLineID != li.FulfillmentLineID {
			return nil, nil, fmt.Errorf("supplier order line %d references fulfillment line %d, not %d", li.SupplierOrderLineID, sol.FulfillmentLineID, li.FulfillmentLineID)
		}
		// Validate cross-wave consistency
		if fl.WaveID != supplierOrder.WaveID {
			return nil, nil, fmt.Errorf("fulfillment line %d belongs to wave %d, not wave %d", li.FulfillmentLineID, fl.WaveID, supplierOrder.WaveID)
		}
		// Validate cumulative shipped quantity does not exceed submitted quantity.
		alreadyShipped, err := uc.shipmentRepo.SumShippedQuantityBySOL(sol.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query shipped quantity for SOL %d: %w", sol.ID, err)
		}
		if alreadyShipped+li.Quantity > sol.SubmittedQuantity {
			return nil, nil, fmt.Errorf("over-shipment: SOL %d already shipped %d, requesting %d, max %d",
				sol.ID, alreadyShipped, li.Quantity, sol.SubmittedQuantity)
		}
	}

	// 4. Resolve basis stamp from the supplier order's wave
	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		var err error
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(supplierOrder.WaveID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve basis for shipment: %w", err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	// 5. Build domain objects
	now := time.Now().Format(time.RFC3339)
	shipment := &domain.Shipment{
		SupplierOrderID:      input.SupplierOrderID,
		SupplierPlatform:     input.SupplierPlatform,
		ShipmentNo:           input.ShipmentNo,
		ExternalShipmentNo:   input.ExternalShipmentNo,
		CarrierCode:          input.CarrierCode,
		CarrierName:          input.CarrierName,
		TrackingNo:           input.TrackingNo,
		Status:               input.Status,
		ShippedAt:            input.ShippedAt,
		BasisHistoryNodeID:   basisNodeID,
		BasisProjectionHash:  basisHash,
		BasisPayloadSnapshot: input.BasisPayloadSnapshot,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	lines := make([]*domain.ShipmentLine, len(input.Lines))
	for i, li := range input.Lines {
		lines[i] = &domain.ShipmentLine{
			SupplierOrderLineID: li.SupplierOrderLineID,
			FulfillmentLineID:   li.FulfillmentLineID,
			Quantity:            li.Quantity,
			CreatedAt:           now,
		}
	}

	// 6. Atomic persistence (shipment + lines + basis pin)
	var pin *domain.BasisPinParam
	if pinNodeID != 0 {
		pin = &domain.BasisPinParam{
			HistoryNodeID: pinNodeID,
			PinKind:       "shipment_basis",
			RefType:       "shipment",
		}
	}
	if err := uc.shipmentRepo.AtomicCreateShipment(shipment, lines, pin); err != nil {
		return nil, nil, err
	}

	// 7. Project supplier state → FulfillmentLine (driven by actual shipment status).
	// Only "shipped" / "in_transit" / "delivered" count as shipped;
	// pending does NOT upgrade.
	if shipment.Status == "shipped" || shipment.Status == "in_transit" || shipment.Status == "delivered" {
		stateUpdates := make([]domain.FulfillmentLineStateUpdate, 0, len(lines))
		for _, l := range lines {
			stateUpdates = append(stateUpdates, domain.FulfillmentLineStateUpdate{
				ID:            l.FulfillmentLineID,
				SupplierState: "shipped",
			})
		}
		if len(stateUpdates) > 0 {
			_ = uc.fulfillRepo.BulkUpdateStates(stateUpdates)
		}
	}

	// 8. Project SupplierOrder status based on cumulative shipped quantities across all SOLs.
	_ = uc.projectSupplierOrderStatus(supplierOrder.ID)

	// 9. Return domain objects
	domainLines := make([]domain.ShipmentLine, len(lines))
	for i, l := range lines {
		domainLines[i] = *l
	}
	return shipment, domainLines, nil
}

// projectSupplierOrderStatus recomputes and saves the SupplierOrder status after a shipment
// is created. It compares total shipped quantity against total submitted quantity across all
// SOLs belonging to the order:
//   - all SOLs fully shipped → "shipped"
//   - at least one SOL has any shipped quantity → "partially_shipped"
//   - otherwise → no change
//
// Errors are intentionally swallowed: status projection is best-effort and must not
// roll back an already-persisted shipment.
func (uc *shipmentUseCase) projectSupplierOrderStatus(supplierOrderID uint) error {
	sols, err := uc.supplierRepo.ListLinesByOrder(supplierOrderID)
	if err != nil || len(sols) == 0 {
		return err
	}

	totalSubmitted := 0
	totalShipped := 0
	anyShipped := false

	for _, sol := range sols {
		shipped, err := uc.shipmentRepo.SumShippedQuantityBySOL(sol.ID)
		if err != nil {
			return err
		}
		totalSubmitted += sol.SubmittedQuantity
		totalShipped += shipped
		if shipped > 0 {
			anyShipped = true
		}
	}

	var newStatus string
	if totalSubmitted > 0 && totalShipped >= totalSubmitted {
		newStatus = "shipped"
	} else if anyShipped {
		newStatus = "partially_shipped"
	} else {
		return nil // no change needed
	}

	order, err := uc.supplierRepo.FindByID(supplierOrderID)
	if err != nil {
		return err
	}
	order.Status = newStatus
	return uc.supplierRepo.Update(order)
}
