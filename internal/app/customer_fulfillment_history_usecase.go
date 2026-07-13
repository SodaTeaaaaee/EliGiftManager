package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// CustomerFulfillmentHistoryUseCase answers "did this person actually get
// shipped" across every wave for a given customer profile (plan 5.4 / 3.6).
// It is intentionally self-contained (holds *gorm.DB directly and reads
// persistence models) rather than routed through CustomerProfileUseCase,
// which is a concrete struct with no fulfillment/wave repository dependency
// to extend.
type CustomerFulfillmentHistoryUseCase struct {
	gdb *gorm.DB
}

// NewCustomerFulfillmentHistoryUseCase constructs the use case from a DB
// handle (typically database.GetDB()).
func NewCustomerFulfillmentHistoryUseCase(gdb *gorm.DB) *CustomerFulfillmentHistoryUseCase {
	return &CustomerFulfillmentHistoryUseCase{gdb: gdb}
}

// GetCustomerFulfillmentHistory returns every fulfillment line belonging to
// customerProfileID, across all waves, newest first, enriched with wave name/
// number, product name/SKU, and — where a shipment has been recorded for the
// line — shipment status/tracking/carrier.
func (uc *CustomerFulfillmentHistoryUseCase) GetCustomerFulfillmentHistory(ctx context.Context, customerProfileID uint) ([]dto.CustomerFulfillmentHistoryRowDTO, error) {
	profileIDs, err := uc.mergedProfileHistoryIDs(ctx, customerProfileID)
	if err != nil {
		return nil, err
	}
	var lines []persistence.FulfillmentLine
	if err := uc.gdb.WithContext(ctx).
		Where("customer_profile_id IN ?", profileIDs).
		Order("id DESC").
		Find(&lines).Error; err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return []dto.CustomerFulfillmentHistoryRowDTO{}, nil
	}

	waveIDSet := make(map[uint]struct{})
	productIDSet := make(map[uint]struct{})
	lineIDs := make([]uint, 0, len(lines))
	for _, l := range lines {
		waveIDSet[l.WaveID] = struct{}{}
		if l.ProductID != nil {
			productIDSet[*l.ProductID] = struct{}{}
		}
		lineIDs = append(lineIDs, l.ID)
	}

	waveIDs := make([]uint, 0, len(waveIDSet))
	for id := range waveIDSet {
		waveIDs = append(waveIDs, id)
	}
	var waves []persistence.Wave
	if len(waveIDs) > 0 {
		if err := uc.gdb.WithContext(ctx).Where("id IN ?", waveIDs).Find(&waves).Error; err != nil {
			return nil, err
		}
	}
	waveMap := make(map[uint]persistence.Wave, len(waves))
	for _, w := range waves {
		waveMap[w.ID] = w
	}

	productIDs := make([]uint, 0, len(productIDSet))
	for id := range productIDSet {
		productIDs = append(productIDs, id)
	}
	var products []persistence.Product
	if len(productIDs) > 0 {
		if err := uc.gdb.WithContext(ctx).Where("id IN ?", productIDs).Find(&products).Error; err != nil {
			return nil, err
		}
	}
	productMap := make(map[uint]persistence.Product, len(products))
	for _, p := range products {
		productMap[p.ID] = p
	}

	// Tracking info hangs off shipment_lines (fulfillment_line_id -> shipment_id)
	// -> shipments. A fulfillment line should map to at most one shipment in
	// practice; if duplicates exist (e.g. re-shipment), keep the most recently
	// created shipment line.
	var shipLines []persistence.ShipmentLine
	if len(lineIDs) > 0 {
		if err := uc.gdb.WithContext(ctx).Where("fulfillment_line_id IN ?", lineIDs).Find(&shipLines).Error; err != nil {
			return nil, err
		}
	}
	lineToShipLine := make(map[uint]persistence.ShipmentLine, len(shipLines))
	shipmentIDSet := make(map[uint]struct{})
	for _, sl := range shipLines {
		if existing, ok := lineToShipLine[sl.FulfillmentLineID]; !ok || sl.CreatedAt.After(existing.CreatedAt) {
			lineToShipLine[sl.FulfillmentLineID] = sl
		}
		shipmentIDSet[sl.ShipmentID] = struct{}{}
	}
	shipmentIDs := make([]uint, 0, len(shipmentIDSet))
	for id := range shipmentIDSet {
		shipmentIDs = append(shipmentIDs, id)
	}
	var shipments []persistence.Shipment
	if len(shipmentIDs) > 0 {
		if err := uc.gdb.WithContext(ctx).Where("id IN ?", shipmentIDs).Find(&shipments).Error; err != nil {
			return nil, err
		}
	}
	shipmentMap := make(map[uint]persistence.Shipment, len(shipments))
	for _, s := range shipments {
		shipmentMap[s.ID] = s
	}

	rows := make([]dto.CustomerFulfillmentHistoryRowDTO, 0, len(lines))
	for _, l := range lines {
		row := dto.CustomerFulfillmentHistoryRowDTO{
			FulfillmentLineID: l.ID,
			WaveID:            l.WaveID,
			ProductID:         l.ProductID,
			Quantity:          l.Quantity,
			AllocationState:   l.AllocationState,
			AddressState:      l.AddressState,
			SupplierState:     l.SupplierState,
			ChannelSyncState:  l.ChannelSyncState,
			CreatedAt:         l.CreatedAt,
		}
		if w, ok := waveMap[l.WaveID]; ok {
			row.WaveNo = w.WaveNo
			row.WaveName = w.Name
		}
		if l.ProductID != nil {
			if p, ok := productMap[*l.ProductID]; ok {
				row.ProductName = p.Name
				row.ProductSKU = p.FactorySKU
			}
		}
		if sl, ok := lineToShipLine[l.ID]; ok {
			if s, ok := shipmentMap[sl.ShipmentID]; ok {
				sid := s.ID
				row.ShipmentID = &sid
				row.ShipmentStatus = string(s.Status)
				row.TrackingNo = s.TrackingNo
				row.CarrierName = s.CarrierName
			}
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (uc *CustomerFulfillmentHistoryUseCase) mergedProfileHistoryIDs(ctx context.Context, targetProfileID uint) ([]uint, error) {
	visited := map[uint]struct{}{targetProfileID: {}}
	frontier := []uint{targetProfileID}

	for len(frontier) > 0 {
		var records []persistence.CustomerMergeRecord
		if err := uc.gdb.WithContext(ctx).
			Where("target_profile_id IN ? AND undone_at IS NULL", frontier).
			Find(&records).Error; err != nil {
			return nil, fmt.Errorf("expand merged fulfillment history: %w", err)
		}
		frontier = frontier[:0]
		for _, record := range records {
			if _, seen := visited[record.SourceProfileID]; seen {
				continue
			}
			visited[record.SourceProfileID] = struct{}{}
			frontier = append(frontier, record.SourceProfileID)
		}
	}

	profileIDs := make([]uint, 0, len(visited))
	for id := range visited {
		profileIDs = append(profileIDs, id)
	}
	return profileIDs, nil
}
