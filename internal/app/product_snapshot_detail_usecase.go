package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ProductSnapshotDetailUseCase supplements ProductUseCase.SnapshotProductsForWave
// with a per-master result that reports whether each requested product master
// was newly snapshotted into the wave or already existed there (skip-detail,
// plan 5.3). It is declared as a separate interface — rather than added to
// ProductUseCase in ports.go — so this unit does not need to touch the
// shared ports file.
type ProductSnapshotDetailUseCase interface {
	SnapshotProductsForWaveDetailed(ctx context.Context, input dto.SnapshotProductsInput) (*dto.SnapshotProductsDetailedResult, error)
}

type productSnapshotDetailUseCase struct {
	masterRepo  domain.ProductMasterRepository
	productRepo domain.ProductRepository
	waveRepo    domain.WaveRepository
}

// NewProductSnapshotDetailUseCase constructs a ProductSnapshotDetailUseCase
// from the same repository interfaces ProductUseCase already depends on.
func NewProductSnapshotDetailUseCase(
	masterRepo domain.ProductMasterRepository,
	productRepo domain.ProductRepository,
	waveRepo domain.WaveRepository,
) ProductSnapshotDetailUseCase {
	return &productSnapshotDetailUseCase{
		masterRepo:  masterRepo,
		productRepo: productRepo,
		waveRepo:    waveRepo,
	}
}

func (uc *productSnapshotDetailUseCase) SnapshotProductsForWaveDetailed(ctx context.Context, input dto.SnapshotProductsInput) (*dto.SnapshotProductsDetailedResult, error) {
	if input.WaveID == 0 {
		return nil, fmt.Errorf("wave_id is required")
	}

	// Validate wave exists
	if _, err := uc.waveRepo.FindByID(ctx, input.WaveID); err != nil {
		return nil, fmt.Errorf("wave %d does not exist: %w", input.WaveID, err)
	}

	result := &dto.SnapshotProductsDetailedResult{
		Items: make([]dto.SnapshotProductDetailItem, 0, len(input.MasterIDs)),
	}

	for _, masterID := range input.MasterIDs {
		master, err := uc.masterRepo.FindByID(ctx, masterID)
		if err != nil {
			return nil, fmt.Errorf("product master %d not found: %w", masterID, err)
		}

		existing, err := uc.productRepo.FindByWaveAndSKU(ctx, input.WaveID, master.SupplierPlatform, master.FactorySKU)
		if err == nil && existing != nil {
			result.Items = append(result.Items, dto.SnapshotProductDetailItem{
				MasterID:       masterID,
				Product:        productToDTO(existing),
				AlreadyExisted: true,
			})
			result.SkippedCount++
			continue
		}

		mid := masterID
		// Copy ExtraData from master; image fields stay Master-only (not snapshotted).
		product := &domain.Product{
			WaveID:           input.WaveID,
			ProductMasterID:  &mid,
			SupplierPlatform: master.SupplierPlatform,
			FactorySKU:       master.FactorySKU,
			Name:             master.Name,
			ExtraData:        master.ExtraData,
		}
		if err := uc.productRepo.Create(ctx, product); err != nil {
			return nil, err
		}
		result.Items = append(result.Items, dto.SnapshotProductDetailItem{
			MasterID:       masterID,
			Product:        productToDTO(product),
			AlreadyExisted: false,
		})
		result.CreatedCount++
	}

	return result, nil
}
