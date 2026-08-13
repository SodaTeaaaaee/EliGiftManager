package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- Shipment ----

type shipmentUseCase struct {
	shipmentRepo domain.ShipmentRepository
	supplierRepo domain.SupplierOrderRepository
	fulfillRepo  domain.FulfillmentLineRepository
	waveRepo     domain.WaveRepository
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

// WithShipmentWaveRepo attaches a WaveRepository so CreateShipment can reject
// closed waves. Optional so existing unit callers keep compiling.
func WithShipmentWaveRepo(uc ShipmentUseCase, waveRepo domain.WaveRepository) ShipmentUseCase {
	s, ok := uc.(*shipmentUseCase)
	if ok {
		s.waveRepo = waveRepo
	}
	return uc
}

func (uc *shipmentUseCase) CreateShipment(ctx context.Context, input dto.CreateShipmentInput) (*domain.Shipment, []domain.ShipmentLine, error) {
	// 1. Empty line check
	if len(input.Lines) == 0 {
		return nil, nil, fmt.Errorf("shipment must have at least one line")
	}

	// 2. Validate supplier order existence
	supplierOrder, err := uc.supplierRepo.FindByID(ctx, input.SupplierOrderID)
	if err != nil {
		return nil, nil, fmt.Errorf("supplier order %d not found: %w", input.SupplierOrderID, err)
	}
	if uc.waveRepo != nil {
		wave, waveErr := uc.waveRepo.FindByID(ctx, supplierOrder.WaveID)
		if waveErr != nil {
			return nil, nil, fmt.Errorf("wave %d not found: %w", supplierOrder.WaveID, waveErr)
		}
		if wave != nil && wave.LifecycleStage == string(domain.LifecycleStageClosed) {
			return nil, nil, fmt.Errorf("wave %d is closed and cannot create shipments", supplierOrder.WaveID)
		}
	}
	// 3. Validate each line (all checks outside transaction)
	pendingBySOL := map[uint]int{}
	for _, li := range input.Lines {
		if li.Quantity <= 0 {
			return nil, nil, fmt.Errorf("quantity must be positive, got %d", li.Quantity)
		}
		// Validate supplier order line existence
		sol, err := uc.supplierRepo.FindLineByID(ctx, li.SupplierOrderLineID)
		if err != nil {
			return nil, nil, fmt.Errorf("supplier order line %d not found: %w", li.SupplierOrderLineID, err)
		}
		// Validate fulfillment line existence
		fl, err := uc.fulfillRepo.FindByID(ctx, li.FulfillmentLineID)
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
		// Validate cumulative shipped quantity does not exceed accepted (when
		// present) or submitted quantity. Voided shipments do not occupy quota.
		alreadyShipped, err := sumActiveShippedQuantityBySOL(ctx, uc.shipmentRepo, input.SupplierOrderID, sol.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query shipped quantity for SOL %d: %w", sol.ID, err)
		}
		cap := shippedQuantityCap(*sol)
		occupied := alreadyShipped + pendingBySOL[sol.ID]
		if occupied+li.Quantity > cap {
			return nil, nil, fmt.Errorf("over-shipment: SOL %d already shipped %d, requesting %d, max %d",
				sol.ID, occupied, li.Quantity, cap)
		}
		pendingBySOL[sol.ID] += li.Quantity
	}

	// 4. Resolve basis stamp from the supplier order's wave
	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		var err error
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(ctx, supplierOrder.WaveID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve basis for shipment: %w", err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	// 5. Build domain objects
	now := time.Now()
	shipment := &domain.Shipment{
		SupplierOrderID:      input.SupplierOrderID,
		SupplierPlatform:     supplierOrder.SupplierPlatform,
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
	if err := uc.shipmentRepo.AtomicCreateShipment(ctx, shipment, lines, pin); err != nil {
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
		if err := applyFulfillmentSupplierStates(ctx, uc.fulfillRepo, stateUpdates); err != nil {
			return nil, nil, fmt.Errorf("failed to mark fulfillment lines shipped after creating shipment: %w", err)
		}
	}

	// 8. Project SupplierOrder status based on cumulative shipped quantities across all SOLs.
	if err := uc.projectSupplierOrderStatus(ctx, supplierOrder.ID); err != nil {
		slog.Warn("supplier order status projection failed", "error", err)
	}

	// 9. Return domain objects
	domainLines := make([]domain.ShipmentLine, len(lines))
	for i, l := range lines {
		domainLines[i] = *l
	}
	return shipment, domainLines, nil
}

// projectSupplierOrderStatus recomputes and saves the SupplierOrder status after a shipment
// is created. It compares total shipped quantity against the per-line cap (accepted
// quantity when present, otherwise submitted) across all SOLs belonging to the order:
//   - all SOLs fully shipped → "shipped"
//   - at least one SOL has any shipped quantity → "partially_shipped"
//   - otherwise → accepted / submitted / unchanged draft
//
// Errors are intentionally swallowed by CreateShipment: status projection is
// best-effort and must not roll back an already-persisted shipment. VoidShipment
// returns projection errors so the surrounding transaction can roll back.
func (uc *shipmentUseCase) projectSupplierOrderStatus(ctx context.Context, supplierOrderID uint) error {
	return recomputeSupplierOrderShippingStatus(ctx, uc.shipmentRepo, uc.supplierRepo, supplierOrderID)
}

// shippedQuantityCap is the over-ship ceiling for a supplier order line.
// AcceptedQuantity, when recorded (>0), is the factory-confirmed cap; otherwise
// SubmittedQuantity is used.
func shippedQuantityCap(sol domain.SupplierOrderLine) int {
	if sol.AcceptedQuantity > 0 {
		return sol.AcceptedQuantity
	}
	return sol.SubmittedQuantity
}

// sumActiveShippedQuantityBySOL returns shipped quantity for a SOL excluding
// voided shipments. ShipmentRepository.SumShippedQuantityBySOL is the occupancy
// source of truth; supplierOrderID is kept so callers can stay order-scoped.
func sumActiveShippedQuantityBySOL(ctx context.Context, repo domain.ShipmentRepository, _, solID uint) (int, error) {
	return repo.SumShippedQuantityBySOL(ctx, solID)
}

// applyFulfillmentSupplierStates writes supplier_state (and optional channel
// sync state). If BulkUpdateStates fails, it compensates with per-line Update
// so a shipment is not left occupying quantity while fulfillment lines stay
// unshipped. Callers without an outer transaction rely on that compensation;
// callers inside a transaction still return error on total failure so the
// transaction can roll back the shipment.
func applyFulfillmentSupplierStates(ctx context.Context, fulfillRepo domain.FulfillmentLineRepository, updates []domain.FulfillmentLineStateUpdate) error {
	if fulfillRepo == nil || len(updates) == 0 {
		return nil
	}
	err := fulfillRepo.BulkUpdateStates(ctx, updates)
	if err == nil {
		return nil
	}
	for _, u := range updates {
		fl, findErr := fulfillRepo.FindByID(ctx, u.ID)
		if findErr != nil {
			return errors.Join(err, findErr)
		}
		if u.SupplierState != "" {
			fl.SupplierState = u.SupplierState
		}
		if u.ChannelSyncState != "" {
			fl.ChannelSyncState = u.ChannelSyncState
		}
		if updErr := fulfillRepo.Update(ctx, fl); updErr != nil {
			return errors.Join(err, updErr)
		}
	}
	return nil
}

func recomputeSupplierOrderShippingStatus(ctx context.Context, shipmentRepo domain.ShipmentRepository, supplierRepo domain.SupplierOrderRepository, supplierOrderID uint) error {
	if supplierRepo == nil || shipmentRepo == nil {
		return nil
	}
	sols, err := supplierRepo.ListLinesByOrder(ctx, supplierOrderID)
	if err != nil || len(sols) == 0 {
		return err
	}

	totalCap := 0
	totalShipped := 0
	anyShipped := false
	anyAccepted := false
	for _, sol := range sols {
		shipped, sumErr := sumActiveShippedQuantityBySOL(ctx, shipmentRepo, supplierOrderID, sol.ID)
		if sumErr != nil {
			return sumErr
		}
		totalCap += shippedQuantityCap(sol)
		totalShipped += shipped
		if shipped > 0 {
			anyShipped = true
		}
		if sol.AcceptedQuantity > 0 {
			anyAccepted = true
		}
	}

	order, err := supplierRepo.FindByID(ctx, supplierOrderID)
	if err != nil {
		return err
	}

	var newStatus string
	switch {
	case totalCap > 0 && totalShipped >= totalCap:
		newStatus = string(domain.SupplierOrderStatusShipped)
	case anyShipped:
		newStatus = string(domain.SupplierOrderStatusPartiallyShipped)
	case anyAccepted || order.Status == string(domain.SupplierOrderStatusAccepted):
		newStatus = string(domain.SupplierOrderStatusAccepted)
	case order.Status == string(domain.SupplierOrderStatusDraft):
		return nil
	default:
		newStatus = string(domain.SupplierOrderStatusSubmitted)
	}
	if order.Status == newStatus {
		return nil
	}
	order.Status = newStatus
	return supplierRepo.Update(ctx, order)
}

func restoredSupplierStateAfterVoid(sol domain.SupplierOrderLine, remaining, cap int, order *domain.SupplierOrder) string {
	if remaining > 0 {
		if cap > 0 && remaining >= cap {
			return string(domain.SupplierStateShipped)
		}
		return string(domain.SupplierStatePartiallyShipped)
	}
	if sol.AcceptedQuantity > 0 {
		return string(domain.SupplierStateAccepted)
	}
	if order != nil && order.Status == string(domain.SupplierOrderStatusDraft) {
		return string(domain.SupplierStateNotSubmitted)
	}
	return string(domain.SupplierStateSubmitted)
}
