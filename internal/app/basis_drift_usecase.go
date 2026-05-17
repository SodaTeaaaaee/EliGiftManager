package app

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type basisDriftDetectionUseCase struct {
	supplierRepo    domain.SupplierOrderRepository
	shipmentRepo    domain.ShipmentRepository
	channelSyncRepo domain.ChannelSyncRepository
	fulfillRepo     domain.FulfillmentLineRepository
}

func NewBasisDriftDetectionUseCase(
	supplierRepo domain.SupplierOrderRepository,
	shipmentRepo domain.ShipmentRepository,
	channelSyncRepo domain.ChannelSyncRepository,
	fulfillRepo domain.FulfillmentLineRepository,
) BasisDriftDetectionUseCase {
	return &basisDriftDetectionUseCase{
		supplierRepo:    supplierRepo,
		shipmentRepo:    shipmentRepo,
		channelSyncRepo: channelSyncRepo,
		fulfillRepo:     fulfillRepo,
	}
}

func (uc *basisDriftDetectionUseCase) DetectWaveBasisDrift(waveID uint, currentProjectionHash string) ([]dto.BasisDriftSignalDTO, error) {
	var signals []dto.BasisDriftSignalDTO

	// Check supplier orders
	orders, err := uc.supplierRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		if signal := uc.checkBasis(o.BasisHistoryNodeID, o.BasisProjectionHash, currentProjectionHash, "supplier_order_basis"); signal != nil {
			signals = append(signals, *signal)
		}
	}

	// Check shipments
	shipments, err := uc.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	for _, s := range shipments {
		if signal := uc.checkBasis(s.BasisHistoryNodeID, s.BasisProjectionHash, currentProjectionHash, "shipment_basis"); signal != nil {
			signals = append(signals, *signal)
		}
	}

	// Check channel sync jobs
	jobs, err := uc.channelSyncRepo.ListJobsByWave(waveID)
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		if signal := uc.checkBasis(j.BasisHistoryNodeID, j.BasisProjectionHash, currentProjectionHash, "channel_sync_basis"); signal != nil {
			signals = append(signals, *signal)
		}
	}

	// Structural integrity check — detects target_deleted regardless of hash state.
	// These signals carry ReviewRequirement "required" because safe replay is no longer possible.
	structuralSignals := uc.detectStructuralUnsafety(waveID)
	signals = append(signals, structuralSignals...)

	return signals, nil
}

// detectStructuralUnsafety checks whether any non-draft external object references a
// fulfillment line that no longer exists in the wave. Such a state makes safe basis
// replay impossible, so it always emits ReviewRequirement "required".
//
// Graceful degradation: if any repo call fails, that check is skipped rather than
// surfacing an error — hash-based signals are still returned to the caller.
func (uc *basisDriftDetectionUseCase) detectStructuralUnsafety(waveID uint) []dto.BasisDriftSignalDTO {
	var signals []dto.BasisDriftSignalDTO

	// Build the set of fulfillment line IDs that currently exist in this wave.
	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil // graceful degradation
	}
	flIDSet := make(map[uint]bool, len(fulfillLines))
	for _, fl := range fulfillLines {
		flIDSet[fl.ID] = true
	}

	// Check supplier order lines — skip draft orders because they are rebuilt on re-export.
	orders, err := uc.supplierRepo.ListByWave(waveID)
	if err == nil {
		for _, order := range orders {
			if order.Status == "draft" {
				continue
			}
			lines, err := uc.supplierRepo.ListLinesByOrder(order.ID)
			if err != nil {
				continue // graceful degradation per order
			}
			for _, sol := range lines {
				if !flIDSet[sol.FulfillmentLineID] {
					signals = append(signals, dto.BasisDriftSignalDTO{
						BasisKind:         "supplier_order",
						BasisDriftStatus:  "drifted",
						ReviewRequirement: "required",
						DriftReasonCodes:  []string{"target_deleted"},
					})
					break // one signal per order is sufficient
				}
			}
		}
	}

	// Check shipment lines — all shipments are structural once created.
	shipments, err := uc.shipmentRepo.ListByWave(waveID)
	if err == nil {
		for _, shipment := range shipments {
			shipLines, err := uc.shipmentRepo.ListLinesByShipment(shipment.ID)
			if err != nil {
				continue // graceful degradation per shipment
			}
			for _, sl := range shipLines {
				if !flIDSet[sl.FulfillmentLineID] {
					signals = append(signals, dto.BasisDriftSignalDTO{
						BasisKind:         "shipment",
						BasisDriftStatus:  "drifted",
						ReviewRequirement: "required",
						DriftReasonCodes:  []string{"target_deleted"},
					})
					break // one signal per shipment is sufficient
				}
			}
		}
	}

	return signals
}

func (uc *basisDriftDetectionUseCase) checkBasis(nodeID, storedHash, currentHash, kind string) *dto.BasisDriftSignalDTO {
	// No basis reference yet — skip (object created before history infra)
	if nodeID == "" {
		return nil
	}

	// Has node ID but no stored hash — basis is stale (history infra not yet populating hashes)
	if storedHash == "" {
		return &dto.BasisDriftSignalDTO{
			BasisKind:         kind,
			BasisDriftStatus:  "drifted",
			ReviewRequirement: "recommended",
			DriftReasonCodes:  []string{"external_basis_stale"},
		}
	}

	// Current projection hash not available (Phase 9 not yet active)
	if currentHash == "" {
		return &dto.BasisDriftSignalDTO{
			BasisKind:         kind,
			BasisDriftStatus:  "drifted",
			ReviewRequirement: "recommended",
			DriftReasonCodes:  []string{"projection_hash_unavailable"},
		}
	}

	// Both hashes available — compare
	if storedHash != currentHash {
		return &dto.BasisDriftSignalDTO{
			BasisKind:         kind,
			BasisDriftStatus:  "drifted",
			ReviewRequirement: "recommended",
			DriftReasonCodes:  []string{"projection_changed"},
		}
	}

	// In sync
	return &dto.BasisDriftSignalDTO{
		BasisKind:         kind,
		BasisDriftStatus:  "in_sync",
		ReviewRequirement: "none",
		DriftReasonCodes:  nil,
	}
}
