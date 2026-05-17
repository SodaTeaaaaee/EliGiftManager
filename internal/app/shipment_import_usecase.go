package app

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type shipmentImportUseCase struct {
	shipmentRepo domain.ShipmentRepository
	supplierRepo domain.SupplierOrderRepository
	fulfillRepo  domain.FulfillmentLineRepository
	basisStamp   *BasisStampService
}

// NewShipmentImportUseCase constructs a ShipmentImportUseCase.
func NewShipmentImportUseCase(
	shipmentRepo domain.ShipmentRepository,
	supplierRepo domain.SupplierOrderRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	basisStamp *BasisStampService,
) ShipmentImportUseCase {
	return &shipmentImportUseCase{
		shipmentRepo: shipmentRepo,
		supplierRepo: supplierRepo,
		fulfillRepo:  fulfillRepo,
		basisStamp:   basisStamp,
	}
}

// ImportShipments performs a bulk import of shipments from factory return data.
// Entries are grouped by ExternalShipmentNo; each group becomes one Shipment with
// multiple lines. Groups that fail validation are skipped and recorded in Errors.
// Groups that pass are persisted atomically. Partial success is supported.
func (uc *shipmentImportUseCase) ImportShipments(input dto.ImportShipmentInput) (*dto.ImportShipmentResult, error) {
	// 1. Validate that the wave has at least one supplier order.
	supplierOrders, err := uc.supplierRepo.ListByWave(input.WaveID)
	if err != nil {
		return nil, fmt.Errorf("list supplier orders for wave %d: %w", input.WaveID, err)
	}
	if len(supplierOrders) == 0 {
		return nil, fmt.Errorf("wave %d has no supplier orders", input.WaveID)
	}

	// Build lookup sets for fast membership checks.
	supplierOrderByID := make(map[uint]*domain.SupplierOrder, len(supplierOrders))
	for i := range supplierOrders {
		supplierOrderByID[supplierOrders[i].ID] = &supplierOrders[i]
	}

	// 2. Group entries by ExternalShipmentNo (preserving insertion order via slice).
	type group struct {
		externalNo string
		indices    []int // original entry indices for error reporting
	}
	groupOrder := []string{}
	groupMap := map[string]*group{}
	for i, e := range input.Entries {
		key := e.ExternalShipmentNo
		if key == "" {
			// Treat blank external no as a unique per-entry group so it still gets
			// a generated shipment number rather than merging silently.
			key = fmt.Sprintf("__auto_%d", i)
		}
		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &group{externalNo: e.ExternalShipmentNo}
			groupOrder = append(groupOrder, key)
		}
		groupMap[key].indices = append(groupMap[key].indices, i)
	}

	result := &dto.ImportShipmentResult{
		TotalProcessed: len(input.Entries),
	}

	// Resolve basis stamp once for the wave (same wave for all entries).
	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(input.WaveID)
		if err != nil {
			return nil, fmt.Errorf("resolve basis for wave %d: %w", input.WaveID, err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	shipmentIndex := 0

	for _, key := range groupOrder {
		grp := groupMap[key]
		shipmentIndex++

		// 3. Validate all entries in this group.
		var groupErr []dto.ImportShipmentError
		var groupLines []*domain.ShipmentLine

		// Pick representative fields from first entry for the shipment header.
		firstEntry := input.Entries[grp.indices[0]]

		// Track the supplier order ID for this group — all lines in a group must
		// resolve to supplier orders belonging to the same wave. We derive the
		// supplier order from the supplier order line.
		var groupSupplierOrderID uint

		now := time.Now().Format(time.RFC3339)

		for _, idx := range grp.indices {
			e := input.Entries[idx]

			// Validate supplier order line existence.
			sol, solErr := uc.supplierRepo.FindLineByID(e.SupplierOrderLineID)
			if solErr != nil {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("supplier order line %d not found: %v", e.SupplierOrderLineID, solErr),
				})
				continue
			}

			// Validate the supplier order belongs to this wave.
			so, soExists := supplierOrderByID[sol.SupplierOrderID]
			if !soExists {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("supplier order line %d belongs to order %d which is not in wave %d", e.SupplierOrderLineID, sol.SupplierOrderID, input.WaveID),
				})
				continue
			}

			// Validate fulfillment line existence.
			fl, flErr := uc.fulfillRepo.FindByID(e.FulfillmentLineID)
			if flErr != nil {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("fulfillment line %d not found: %v", e.FulfillmentLineID, flErr),
				})
				continue
			}

			// Validate fulfillment line belongs to the wave.
			if fl.WaveID != input.WaveID {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("fulfillment line %d belongs to wave %d, not wave %d", e.FulfillmentLineID, fl.WaveID, input.WaveID),
				})
				continue
			}

			// Validate supplier order line references the correct fulfillment line.
			if sol.FulfillmentLineID != e.FulfillmentLineID {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("supplier order line %d references fulfillment line %d, not %d", e.SupplierOrderLineID, sol.FulfillmentLineID, e.FulfillmentLineID),
				})
				continue
			}

			// Validate quantity.
			if e.Quantity <= 0 {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("entry %d: quantity must be positive, got %d", idx, e.Quantity),
				})
				continue
			}
			if e.Quantity > sol.SubmittedQuantity {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("entry %d: quantity %d exceeds supplier order line %d submitted quantity %d", idx, e.Quantity, e.SupplierOrderLineID, sol.SubmittedQuantity),
				})
				continue
			}

			// All checks passed for this entry.
			_ = so // used for wave membership check above
			groupSupplierOrderID = sol.SupplierOrderID

			shippedAt := e.ShippedAt
			if shippedAt == "" {
				shippedAt = now
			}
			_ = shippedAt // stored per-line via shipment header

			groupLines = append(groupLines, &domain.ShipmentLine{
				SupplierOrderLineID: e.SupplierOrderLineID,
				FulfillmentLineID:   e.FulfillmentLineID,
				Quantity:            e.Quantity,
				CreatedAt:           now,
			})
		}

		// If any entry in the group failed, skip the whole group.
		if len(groupErr) > 0 {
			result.Errors = append(result.Errors, groupErr...)
			result.ErrorCount += len(grp.indices)
			continue
		}

		// If no lines were built (shouldn't happen given the above, but guard anyway).
		if len(groupLines) == 0 {
			result.Errors = append(result.Errors, dto.ImportShipmentError{
				EntryIndex: grp.indices[0],
				Reason:     "group produced no valid lines",
			})
			result.ErrorCount += len(grp.indices)
			continue
		}

		// 4. Build shipment domain object.
		shippedAt := firstEntry.ShippedAt
		if shippedAt == "" {
			shippedAt = now
		}
		shipmentNo := fmt.Sprintf("IMP-%d-%d", input.WaveID, shipmentIndex)
		shipment := &domain.Shipment{
			SupplierOrderID:     groupSupplierOrderID,
			ShipmentNo:          shipmentNo,
			ExternalShipmentNo:  firstEntry.ExternalShipmentNo,
			CarrierCode:         firstEntry.CarrierCode,
			CarrierName:         firstEntry.CarrierName,
			TrackingNo:          firstEntry.TrackingNo,
			Status:              "shipped",
			ShippedAt:           shippedAt,
			BasisHistoryNodeID:  basisNodeID,
			BasisProjectionHash: basisHash,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		var pin *domain.BasisPinParam
		if pinNodeID != 0 {
			pin = &domain.BasisPinParam{
				HistoryNodeID: pinNodeID,
				PinKind:       "shipment_basis",
				RefType:       "shipment",
			}
		}

		// 5. Persist atomically.
		if createErr := uc.shipmentRepo.AtomicCreateShipment(shipment, groupLines, pin); createErr != nil {
			for _, idx := range grp.indices {
				result.Errors = append(result.Errors, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("persist failed: %v", createErr),
				})
			}
			result.ErrorCount += len(grp.indices)
			continue
		}

		// 6. Project supplier_state → FulfillmentLine (same logic as CreateShipment).
		stateUpdates := make([]domain.FulfillmentLineStateUpdate, 0, len(groupLines))
		for _, l := range groupLines {
			stateUpdates = append(stateUpdates, domain.FulfillmentLineStateUpdate{
				ID:            l.FulfillmentLineID,
				SupplierState: "shipped",
			})
		}
		if len(stateUpdates) > 0 {
			_ = uc.fulfillRepo.BulkUpdateStates(stateUpdates)
		}

		// 7. Collect result DTO.
		shipmentDTO := dto.ShipmentDTO{
			ID:                  shipment.ID,
			SupplierOrderID:     shipment.SupplierOrderID,
			SupplierPlatform:    shipment.SupplierPlatform,
			ShipmentNo:          shipment.ShipmentNo,
			ExternalShipmentNo:  shipment.ExternalShipmentNo,
			CarrierCode:         shipment.CarrierCode,
			CarrierName:         shipment.CarrierName,
			TrackingNo:          shipment.TrackingNo,
			Status:              shipment.Status,
			ShippedAt:           shipment.ShippedAt,
			BasisHistoryNodeID:  shipment.BasisHistoryNodeID,
			BasisProjectionHash: shipment.BasisProjectionHash,
			CreatedAt:           shipment.CreatedAt,
			UpdatedAt:           shipment.UpdatedAt,
		}
		shipmentDTO.Lines = make([]dto.ShipmentLineDTO, len(groupLines))
		for i, l := range groupLines {
			shipmentDTO.Lines[i] = dto.ShipmentLineDTO{
				ID:                  l.ID,
				ShipmentID:          l.ShipmentID,
				SupplierOrderLineID: l.SupplierOrderLineID,
				FulfillmentLineID:   l.FulfillmentLineID,
				Quantity:            l.Quantity,
				CreatedAt:           l.CreatedAt,
			}
		}
		result.CreatedShipments = append(result.CreatedShipments, shipmentDTO)
		result.SuccessCount += len(grp.indices)
	}

	return result, nil
}
