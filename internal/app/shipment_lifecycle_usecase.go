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
}

// NewShipmentLifecycleUseCase constructs a ShipmentLifecycleUseCase.
func NewShipmentLifecycleUseCase(shipmentRepo domain.ShipmentRepository, writeRepo domain.ShipmentWriteRepository) ShipmentLifecycleUseCase {
	return &shipmentLifecycleUseCase{shipmentRepo: shipmentRepo, writeRepo: writeRepo}
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
	return shipment, nil
}
