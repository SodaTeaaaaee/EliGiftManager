package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type shipmentImportUseCase struct {
	shipmentRepo     domain.ShipmentRepository
	supplierRepo     domain.SupplierOrderRepository
	fulfillRepo      domain.FulfillmentLineRepository
	basisStamp       *BasisStampService
	reconcile        *shipmentReconcileDeps
	evidence         *ImportEvidenceUseCase
	externalRegistry *ExternalCarrierUseCase
}

func WithShipmentImportEvidence(uc ShipmentImportUseCase, evidence *ImportEvidenceUseCase) ShipmentImportUseCase {
	s, ok := uc.(*shipmentImportUseCase)
	if ok {
		s.evidence = evidence
	}
	return uc
}
func WithShipmentExternalCarrierRegistry(uc ShipmentImportUseCase, registry *ExternalCarrierUseCase) ShipmentImportUseCase {
	s, ok := uc.(*shipmentImportUseCase)
	if ok {
		s.externalRegistry = registry
	}
	return uc
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
// multiple lines.
//
// ImportMode controls failure handling:
//   - "skip_invalid" (default): skip failed groups, persist valid ones (partial success).
//   - "reject_all": validate all groups first; if any error exists, return errors and
//     persist nothing.
func (uc *shipmentImportUseCase) ImportShipments(ctx context.Context, input dto.ImportShipmentInput) (*dto.ImportShipmentResult, error) {
	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	var run *domain.ImportRun
	var records []domain.ImportRawRecord
	if uc.evidence != nil {
		rows := make([]any, len(input.Entries))
		unmapped := make([]map[string]string, len(input.Entries))
		for i, entry := range input.Entries {
			rows[i] = entry
			unmapped[i] = map[string]string{}
		}
		var err error
		run, records, err = uc.evidence.StartImportEvidence(ctx, "supplier_shipment", input.IntegrationProfileID, mode, "", `{"source":"mapped_entries"}`, rows, unmapped, nil)
		if err != nil {
			return nil, fmt.Errorf("start shipment import evidence: %w", err)
		}
	}
	result, err := uc.importShipmentsCore(ctx, input)
	if err != nil {
		return nil, err
	}
	if run != nil {
		result.ImportRunID = run.ID
		for i := range records {
			markImportEvidenceSuccess(records, i, "shipment", 0)
		}
		for _, itemErr := range result.Errors {
			markImportEvidenceFailure(records, itemErr.EntryIndex, "shipment_import_error", itemErr.Reason, nil)
		}
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		if mode == "reject_all" && result.ErrorCount > 0 {
			status = "rejected"
		}
		if err := uc.evidence.CompleteImportEvidence(ctx, run, records, status); err != nil {
			return nil, err
		}
	} else if uc.evidence != nil {
		result.EvidenceDisabled = true
	}
	return result, nil
}

type shipmentImportCoreOutcome struct {
	result                 *dto.ImportShipmentResult
	successfulEntryIndices map[int]struct{}
}

func (uc *shipmentImportUseCase) importShipmentsCore(ctx context.Context, input dto.ImportShipmentInput) (*dto.ImportShipmentResult, error) {
	outcome, err := uc.importShipmentsCoreWithOutcome(ctx, input)
	if err != nil {
		return nil, err
	}
	return outcome.result, nil
}

// importShipmentsCoreWithOutcome exposes which compacted input entries were
// actually persisted. Map-and-reconcile needs that internal bookkeeping to
// stage side effects (such as carrier observation) until after the whole-file
// gate and to apply them only to successful physical source rows.
func (uc *shipmentImportUseCase) importShipmentsCoreWithOutcome(ctx context.Context, input dto.ImportShipmentInput) (*shipmentImportCoreOutcome, error) {
	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	if mode != "reject_all" && mode != "skip_invalid" {
		return nil, fmt.Errorf("invalid importMode %q: must be \"reject_all\" or \"skip_invalid\"", mode)
	}

	// 1. Validate that the wave has at least one supplier order.
	supplierOrders, err := uc.supplierRepo.ListByWave(ctx, input.WaveID)
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
	outcome := &shipmentImportCoreOutcome{
		result:                 result,
		successfulEntryIndices: make(map[int]struct{}, len(input.Entries)),
	}

	// Resolve basis stamp once for the wave (same wave for all entries).
	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(ctx, input.WaveID)
		if err != nil {
			return nil, fmt.Errorf("resolve basis for wave %d: %w", input.WaveID, err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	now := time.Now()

	// validatedGroup holds the outcome of the validation pass for one group.
	type validatedGroup struct {
		grp                *group
		groupErr           []dto.ImportShipmentError
		groupLines         []*domain.ShipmentLine
		groupSupplierOrder uint
		groupPlatform      string
		firstEntry         dto.ImportShipmentEntry
		shipmentIndex      int
	}

	validated := make([]validatedGroup, 0, len(groupOrder))

	// 3. Validation pass — iterate all groups, collect errors and valid lines.
	//    No persistence happens here regardless of mode.
	for i, key := range groupOrder {
		grp := groupMap[key]

		var groupErr []dto.ImportShipmentError
		var groupLines []*domain.ShipmentLine
		var groupSupplierOrderID uint

		firstEntry := input.Entries[grp.indices[0]]

		for _, idx := range grp.indices {
			e := input.Entries[idx]

			// Validate supplier order line existence.
			sol, solErr := uc.supplierRepo.FindLineByID(ctx, e.SupplierOrderLineID)
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
			if input.IntegrationProfileID != 0 && so.FactoryIntegrationProfileID != nil && *so.FactoryIntegrationProfileID != input.IntegrationProfileID {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason: fmt.Sprintf(
						"supplier order %d belongs to factory profile %d, not import profile %d",
						so.ID, *so.FactoryIntegrationProfileID, input.IntegrationProfileID,
					),
				})
				continue
			}
			if groupSupplierOrderID != 0 && groupSupplierOrderID != sol.SupplierOrderID {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("external shipment %q mixes supplier orders %d and %d", grp.externalNo, groupSupplierOrderID, sol.SupplierOrderID),
				})
				continue
			}

			// Validate fulfillment line existence.
			fl, flErr := uc.fulfillRepo.FindByID(ctx, e.FulfillmentLineID)
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
			alreadyShipped, sumErr := uc.shipmentRepo.SumShippedQuantityBySOL(ctx, sol.ID)
			if sumErr != nil {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("entry %d: failed to query shipped quantity for SOL %d", idx, sol.ID),
				})
				continue
			}
			if alreadyShipped+e.Quantity > sol.SubmittedQuantity {
				groupErr = append(groupErr, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("entry %d: over-shipment: already shipped %d + %d > submitted %d for SOL %d", idx, alreadyShipped, e.Quantity, sol.SubmittedQuantity, sol.ID),
				})
				continue
			}

			// All checks passed for this entry.
			_ = so // used for wave membership check above
			groupSupplierOrderID = sol.SupplierOrderID

			shippedAt := e.ShippedAt
			if shippedAt == nil {
				shippedAt = &now
			}
			_ = shippedAt // stored per-line via shipment header

			groupLines = append(groupLines, &domain.ShipmentLine{
				SupplierOrderLineID: e.SupplierOrderLineID,
				FulfillmentLineID:   e.FulfillmentLineID,
				Quantity:            e.Quantity,
				CreatedAt:           now,
			})
		}

		validated = append(validated, validatedGroup{
			grp:                grp,
			groupErr:           groupErr,
			groupLines:         groupLines,
			groupSupplierOrder: groupSupplierOrderID,
			groupPlatform: func() string {
				if order := supplierOrderByID[groupSupplierOrderID]; order != nil {
					return order.SupplierPlatform
				}
				return ""
			}(),
			firstEntry:    firstEntry,
			shipmentIndex: i + 1,
		})
	}

	// 4. Collect all validation errors across groups.
	var allErrors []dto.ImportShipmentError
	for _, vg := range validated {
		allErrors = append(allErrors, vg.groupErr...)
		if len(vg.groupLines) == 0 && len(vg.groupErr) == 0 {
			// Guard: group produced no lines and no errors (shouldn't happen).
			allErrors = append(allErrors, dto.ImportShipmentError{
				EntryIndex: vg.grp.indices[0],
				Reason:     "group produced no valid lines",
			})
		}
	}

	// 5. reject_all early-return: if any error exists, return without persisting anything.
	if mode == "reject_all" && len(allErrors) > 0 {
		outcome.result = &dto.ImportShipmentResult{
			TotalProcessed: len(input.Entries),
			SuccessCount:   0,
			ErrorCount:     len(allErrors),
			Errors:         allErrors,
		}
		return outcome, nil
	}

	// 6. Persistence pass — skip_invalid: persist valid groups, record errors for invalid ones.
	for _, vg := range validated {
		// If any entry in the group failed, skip the whole group.
		if len(vg.groupErr) > 0 {
			result.Errors = append(result.Errors, vg.groupErr...)
			result.ErrorCount += len(vg.grp.indices)
			continue
		}

		// Guard: group produced no valid lines.
		if len(vg.groupLines) == 0 {
			result.Errors = append(result.Errors, dto.ImportShipmentError{
				EntryIndex: vg.grp.indices[0],
				Reason:     "group produced no valid lines",
			})
			result.ErrorCount += len(vg.grp.indices)
			continue
		}

		// Build shipment domain object.
		shippedAt := vg.firstEntry.ShippedAt
		if shippedAt == nil {
			shippedAt = &now
		}
		shipmentNo := fmt.Sprintf("IMP-%d-%d", input.WaveID, vg.shipmentIndex)
		shipment := &domain.Shipment{
			SupplierOrderID:     vg.groupSupplierOrder,
			SupplierPlatform:    vg.groupPlatform,
			ShipmentNo:          shipmentNo,
			ExternalShipmentNo:  vg.firstEntry.ExternalShipmentNo,
			CarrierCode:         vg.firstEntry.CarrierCode,
			CarrierName:         vg.firstEntry.CarrierName,
			TrackingNo:          vg.firstEntry.TrackingNo,
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

		// Persist atomically.
		if createErr := uc.shipmentRepo.AtomicCreateShipment(ctx, shipment, vg.groupLines, pin); createErr != nil {
			if mode == "reject_all" {
				return nil, fmt.Errorf("persist shipment group %q: %w", vg.grp.externalNo, createErr)
			}
			for _, idx := range vg.grp.indices {
				result.Errors = append(result.Errors, dto.ImportShipmentError{
					EntryIndex: idx,
					Reason:     fmt.Sprintf("persist failed: %v", createErr),
				})
			}
			result.ErrorCount += len(vg.grp.indices)
			continue
		}

		// Project supplier_state → FulfillmentLine (same logic as CreateShipment).
		stateUpdates := make([]domain.FulfillmentLineStateUpdate, 0, len(vg.groupLines))
		for _, l := range vg.groupLines {
			stateUpdates = append(stateUpdates, domain.FulfillmentLineStateUpdate{
				ID:            l.FulfillmentLineID,
				SupplierState: "shipped",
			})
		}
		if len(stateUpdates) > 0 {
			if updateErr := uc.fulfillRepo.BulkUpdateStates(ctx, stateUpdates); updateErr != nil {
				if mode == "reject_all" {
					return nil, fmt.Errorf("project shipment group %q supplier state: %w", vg.grp.externalNo, updateErr)
				}
				for _, idx := range vg.grp.indices {
					result.Errors = append(result.Errors, dto.ImportShipmentError{
						EntryIndex: idx,
						Reason:     fmt.Sprintf("project supplier state failed: %v", updateErr),
					})
				}
				result.ErrorCount += len(vg.grp.indices)
				continue
			}
		}

		// Collect result DTO.
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
		shipmentDTO.Lines = make([]dto.ShipmentLineDTO, len(vg.groupLines))
		for i, l := range vg.groupLines {
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
		result.SuccessCount += len(vg.grp.indices)
		for _, idx := range vg.grp.indices {
			outcome.successfulEntryIndices[idx] = struct{}{}
		}
	}

	return outcome, nil
}
