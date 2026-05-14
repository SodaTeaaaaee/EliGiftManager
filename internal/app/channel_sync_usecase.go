package app

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type channelSyncUseCase struct {
	channelSyncRepo domain.ChannelSyncRepository
	shipmentRepo    domain.ShipmentRepository
	supplierRepo    domain.SupplierOrderRepository
	fulfillRepo     domain.FulfillmentLineRepository
	basisStamp      *BasisStampService
}

func NewChannelSyncUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	shipmentRepo domain.ShipmentRepository,
	supplierRepo domain.SupplierOrderRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	basisStamp *BasisStampService,
) ChannelSyncUseCase {
	return &channelSyncUseCase{
		channelSyncRepo: channelSyncRepo,
		shipmentRepo:    shipmentRepo,
		supplierRepo:    supplierRepo,
		fulfillRepo:     fulfillRepo,
		basisStamp:      basisStamp,
	}
}

func (uc *channelSyncUseCase) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (*domain.ChannelSyncJob, []domain.ChannelSyncItem, error) {
	if len(input.Items) == 0 {
		return nil, nil, fmt.Errorf("channel sync job must have at least one item")
	}

	if input.IntegrationProfileID == 0 {
		return nil, nil, fmt.Errorf("integration_profile_id is required")
	}

	if input.Direction != "push_tracking" {
		return nil, nil, fmt.Errorf("unsupported direction: %q (only push_tracking is allowed)", input.Direction)
	}

	// Validate reference chain for every item (all checks outside transaction).
	for i, it := range input.Items {
		shipment, err := uc.shipmentRepo.FindByID(it.ShipmentID)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d: shipment %d not found: %w", i, it.ShipmentID, err)
		}

		fulfillLine, err := uc.fulfillRepo.FindByID(it.FulfillmentLineID)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d: fulfillment line %d not found: %w", i, it.FulfillmentLineID, err)
		}

		supplierOrder, err := uc.supplierRepo.FindByID(shipment.SupplierOrderID)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d: supplier order %d not found (referenced by shipment %d): %w", i, shipment.SupplierOrderID, it.ShipmentID, err)
		}

		if supplierOrder.WaveID != input.WaveID {
			return nil, nil, fmt.Errorf("item %d: shipment %d belongs to supplier order %d in wave %d, not wave %d", i, it.ShipmentID, shipment.SupplierOrderID, supplierOrder.WaveID, input.WaveID)
		}

		if fulfillLine.WaveID != input.WaveID {
			return nil, nil, fmt.Errorf("item %d: fulfillment line %d belongs to wave %d, not wave %d", i, it.FulfillmentLineID, fulfillLine.WaveID, input.WaveID)
		}

		shipmentLines, err := uc.shipmentRepo.ListLinesByShipment(it.ShipmentID)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d: cannot list shipment lines for shipment %d: %w", i, it.ShipmentID, err)
		}

		linked := false
		for _, sl := range shipmentLines {
			if sl.FulfillmentLineID == it.FulfillmentLineID {
				linked = true
				break
			}
		}
		if !linked {
			return nil, nil, fmt.Errorf("item %d: shipment %d has no line covering fulfillment line %d", i, it.ShipmentID, it.FulfillmentLineID)
		}
	}

	// Resolve basis stamp before persisting
	var basisNodeID, basisHash string
	if uc.basisStamp != nil {
		var err error
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(input.WaveID)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve basis for channel sync job: %w", err)
		}
	}

	now := time.Now().Format(time.RFC3339)
	job := &domain.ChannelSyncJob{
		WaveID:               input.WaveID,
		IntegrationProfileID: input.IntegrationProfileID,
		Direction:            input.Direction,
		Status:               "pending",
		BasisHistoryNodeID:   basisNodeID,
		BasisProjectionHash:  basisHash,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	items := make([]*domain.ChannelSyncItem, len(input.Items))
	for i, it := range input.Items {
		items[i] = &domain.ChannelSyncItem{
			FulfillmentLineID:  it.FulfillmentLineID,
			ShipmentID:         it.ShipmentID,
			ExternalDocumentNo: it.ExternalDocumentNo,
			ExternalLineNo:     it.ExternalLineNo,
			TrackingNo:         it.TrackingNo,
			CarrierCode:        it.CarrierCode,
			Status:             "pending",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
	}

	if err := uc.channelSyncRepo.AtomicCreateChannelSync(job, items); err != nil {
		return nil, nil, err
	}

	// Create basis pin after persistence (job.ID is now set)
	if uc.basisStamp != nil && basisNodeID != "" {
		if err := uc.basisStamp.CreatePin(basisNodeID, "channel_sync_basis", "channel_sync_job", job.ID); err != nil {
			return nil, nil, fmt.Errorf("create basis pin for channel sync job: %w", err)
		}
	}

	domainItems := make([]domain.ChannelSyncItem, len(items))
	for i, it := range items {
		domainItems[i] = *it
	}
	return job, domainItems, nil
}
