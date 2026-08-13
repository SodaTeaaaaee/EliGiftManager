package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ShipmentLifecycleUseCase covers post-creation shipment mutations: field
// corrections (UpdateShipment) and the terminal void transition
// (VoidShipment). Neither operation participates in the undo/redo command
// history (HistoryRecordingService) — they are compensating writes, and per
// plan 5.2 that undo boundary is surfaced honestly rather than faked with an
// unimplemented inverse patch.
type ShipmentLifecycleUseCase interface {
	UpdateShipment(ctx context.Context, input dto.UpdateShipmentInput) (*domain.Shipment, error)
	VoidShipment(ctx context.Context, input dto.VoidShipmentInput) (*domain.Shipment, error)
}

type shipmentLifecycleUseCase struct {
	shipmentRepo domain.ShipmentRepository
	writeRepo    domain.ShipmentWriteRepository
	fulfillRepo  domain.FulfillmentLineRepository
	supplierRepo domain.SupplierOrderRepository
}

// NewShipmentLifecycleUseCase constructs a ShipmentLifecycleUseCase.
// Extra deps may be a FulfillmentLineRepository and/or SupplierOrderRepository
// so VoidShipment can restore supplier_state occupancy. Existing two-argument
// callers remain valid; occupancy rollback is skipped when those repos are omitted.
func NewShipmentLifecycleUseCase(shipmentRepo domain.ShipmentRepository, writeRepo domain.ShipmentWriteRepository, extra ...any) ShipmentLifecycleUseCase {
	uc := &shipmentLifecycleUseCase{shipmentRepo: shipmentRepo, writeRepo: writeRepo}
	for _, dep := range extra {
		switch v := dep.(type) {
		case domain.FulfillmentLineRepository:
			uc.fulfillRepo = v
		case domain.SupplierOrderRepository:
			uc.supplierRepo = v
		}
	}
	return uc
}

// WithShipmentLifecycleOccupancy attaches the repositories VoidShipment needs to
// restore fulfillment-line supplier_state and supplier-order status.
func WithShipmentLifecycleOccupancy(uc ShipmentLifecycleUseCase, fulfillRepo domain.FulfillmentLineRepository, supplierRepo domain.SupplierOrderRepository) ShipmentLifecycleUseCase {
	s, ok := uc.(*shipmentLifecycleUseCase)
	if !ok {
		return uc
	}
	s.fulfillRepo = fulfillRepo
	s.supplierRepo = supplierRepo
	return s
}

// UpdateShipment corrects mutable fields on an existing shipment. Voided
// shipments are terminal and reject further correction.
func (uc *shipmentLifecycleUseCase) UpdateShipment(ctx context.Context, input dto.UpdateShipmentInput) (*domain.Shipment, error) {
	shipment, err := uc.shipmentRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("shipment %d not found: %w", input.ID, err)
	}
	if shipment.Status == string(domain.ShipmentStatusVoided) {
		return nil, fmt.Errorf("shipment %d is voided and cannot be updated", input.ID)
	}

	shipment.SupplierPlatform = input.SupplierPlatform
	shipment.ShipmentNo = input.ShipmentNo
	shipment.ExternalShipmentNo = input.ExternalShipmentNo
	shipment.CarrierCode = input.CarrierCode
	shipment.CarrierName = input.CarrierName
	shipment.TrackingNo = input.TrackingNo
	shipment.ShippedAt = input.ShippedAt

	if err := uc.writeRepo.Update(ctx, shipment); err != nil {
		return nil, fmt.Errorf("failed to update shipment %d: %w", input.ID, err)
	}
	return shipment, nil
}

// VoidShipment marks a shipment as voided — a terminal, compensating state.
// It does not delete the shipment or its lines. Calling VoidShipment on an
// already-voided shipment is idempotent (returns the shipment unchanged).
// The void note/operator/timestamp are stashed in the shipment's ExtraData
// JSON blob (no dedicated audit column exists yet — see dto.VoidShipmentInput
// followup note).
func (uc *shipmentLifecycleUseCase) VoidShipment(ctx context.Context, input dto.VoidShipmentInput) (*domain.Shipment, error) {
	shipment, err := uc.shipmentRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("shipment %d not found: %w", input.ID, err)
	}
	if shipment.Status == string(domain.ShipmentStatusVoided) {
		return shipment, nil
	}

	extra := map[string]any{}
	if shipment.ExtraData != "" {
		// Best-effort merge: if the existing blob isn't a JSON object, the
		// void metadata below still gets written into a fresh object rather
		// than failing the void operation.
		_ = json.Unmarshal([]byte(shipment.ExtraData), &extra)
	}
	extra["voidNote"] = input.Note
	extra["voidedBy"] = input.OperatorID
	extra["voidedAt"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("failed to encode void metadata for shipment %d: %w", input.ID, err)
	}

	shipment.Status = string(domain.ShipmentStatusVoided)
	shipment.ExtraData = string(encoded)

	if err := uc.writeRepo.Update(ctx, shipment); err != nil {
		return nil, fmt.Errorf("failed to void shipment %d: %w", input.ID, err)
	}

	if uc.fulfillRepo != nil {
		if err := uc.restoreOccupancyAfterVoid(ctx, shipment); err != nil {
			return nil, err
		}
	}
	return shipment, nil
}

func (uc *shipmentLifecycleUseCase) restoreOccupancyAfterVoid(ctx context.Context, shipment *domain.Shipment) error {
	lines, err := uc.shipmentRepo.ListLinesByShipment(ctx, shipment.ID)
	if err != nil {
		return fmt.Errorf("failed to list lines for voided shipment %d: %w", shipment.ID, err)
	}

	var order *domain.SupplierOrder
	if uc.supplierRepo != nil {
		order, err = uc.supplierRepo.FindByID(ctx, shipment.SupplierOrderID)
		if err != nil {
			return fmt.Errorf("supplier order %d lookup failed while voiding shipment %d: %w", shipment.SupplierOrderID, shipment.ID, err)
		}
	}

	seenFL := map[uint]struct{}{}
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(lines))
	for _, line := range lines {
		if _, dup := seenFL[line.FulfillmentLineID]; dup {
			continue
		}
		seenFL[line.FulfillmentLineID] = struct{}{}

		remaining, sumErr := sumActiveShippedQuantityBySOL(ctx, uc.shipmentRepo, shipment.SupplierOrderID, line.SupplierOrderLineID)
		if sumErr != nil {
			return fmt.Errorf("failed to recompute shipped quantity for SOL %d: %w", line.SupplierOrderLineID, sumErr)
		}

		sol := domain.SupplierOrderLine{}
		cap := remaining
		if uc.supplierRepo != nil {
			found, findErr := uc.supplierRepo.FindLineByID(ctx, line.SupplierOrderLineID)
			if findErr != nil {
				return fmt.Errorf("supplier order line %d lookup failed while voiding shipment %d: %w", line.SupplierOrderLineID, shipment.ID, findErr)
			}
			if found != nil {
				sol = *found
				cap = shippedQuantityCap(sol)
			}
		}

		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:            line.FulfillmentLineID,
			SupplierState: restoredSupplierStateAfterVoid(sol, remaining, cap, order),
		})
	}

	if err := applyFulfillmentSupplierStates(ctx, uc.fulfillRepo, updates); err != nil {
		return fmt.Errorf("failed to restore fulfillment line supplier state after voiding shipment %d: %w", shipment.ID, err)
	}
	if err := recomputeSupplierOrderShippingStatus(ctx, uc.shipmentRepo, uc.supplierRepo, shipment.SupplierOrderID); err != nil {
		return fmt.Errorf("failed to recompute supplier order status after voiding shipment %d: %w", shipment.ID, err)
	}
	return nil
}
