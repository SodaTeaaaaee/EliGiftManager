package app

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type basisDriftDetectionUseCase struct {
	supplierRepo    domain.SupplierOrderRepository
	shipmentRepo    domain.ShipmentRepository
	channelSyncRepo domain.ChannelSyncRepository
}

func NewBasisDriftDetectionUseCase(
	supplierRepo domain.SupplierOrderRepository,
	shipmentRepo domain.ShipmentRepository,
	channelSyncRepo domain.ChannelSyncRepository,
) BasisDriftDetectionUseCase {
	return &basisDriftDetectionUseCase{
		supplierRepo:    supplierRepo,
		shipmentRepo:    shipmentRepo,
		channelSyncRepo: channelSyncRepo,
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

	return signals, nil
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
