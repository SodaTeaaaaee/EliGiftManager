package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- ProductUseCase ----

type productUseCase struct {
	masterRepo  domain.ProductMasterRepository
	productRepo domain.ProductRepository
	waveRepo    domain.WaveRepository
}

func NewProductUseCase(
	masterRepo domain.ProductMasterRepository,
	productRepo domain.ProductRepository,
	waveRepo domain.WaveRepository,
) ProductUseCase {
	return &productUseCase{
		masterRepo:  masterRepo,
		productRepo: productRepo,
		waveRepo:    waveRepo,
	}
}

var validProductKinds = map[string]bool{
	"badge":    true,
	"standee":  true,
	"charm":    true,
	"postcard": true,
	"print":    true,
	"bundle":   true,
	"other":    true,
}

func (uc *productUseCase) CreateProductMaster(input dto.CreateProductMasterInput) (*dto.ProductMasterDTO, error) {
	if input.SupplierPlatform == "" {
		return nil, fmt.Errorf("supplier_platform is required")
	}
	if input.FactorySKU == "" {
		return nil, fmt.Errorf("factory_sku is required")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Default to "other" if empty; validate otherwise
	productKind := input.ProductKind
	if productKind == "" {
		productKind = "other"
	}
	if !validProductKinds[productKind] {
		return nil, fmt.Errorf("invalid product_kind: %q", productKind)
	}

	master := &domain.ProductMaster{
		SupplierPlatform:   input.SupplierPlatform,
		FactorySKU:         input.FactorySKU,
		SupplierProductRef: input.SupplierProductRef,
		Name:               input.Name,
		ProductKind:        domain.ProductKind(productKind),
		Archived:           false,
		ExtraData:          "",
	}
	if err := uc.masterRepo.Create(master); err != nil {
		return nil, err
	}
	d := productMasterToDTO(master)
	return &d, nil
}

func (uc *productUseCase) ListProductMasters() ([]dto.ProductMasterDTO, error) {
	masters, err := uc.masterRepo.List()
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProductMasterDTO, len(masters))
	for i := range masters {
		result[i] = productMasterToDTO(&masters[i])
	}
	return result, nil
}

func (uc *productUseCase) UpdateProductMaster(input dto.UpdateProductMasterInput) (*dto.ProductMasterDTO, error) {
	master, err := uc.masterRepo.FindByID(input.ID)
	if err != nil {
		return nil, err
	}

	master.SupplierPlatform = input.SupplierPlatform
	master.FactorySKU = input.FactorySKU
	master.SupplierProductRef = input.SupplierProductRef
	master.Name = input.Name
	master.ProductKind = domain.ProductKind(input.ProductKind)
	master.Archived = input.Archived

	if err := uc.masterRepo.Update(master); err != nil {
		return nil, err
	}
	d := productMasterToDTO(master)
	return &d, nil
}

func (uc *productUseCase) SnapshotProductsForWave(input dto.SnapshotProductsInput) ([]dto.ProductDTO, error) {
	if input.WaveID == 0 {
		return nil, fmt.Errorf("wave_id is required")
	}

	// Validate wave exists
	if _, err := uc.waveRepo.FindByID(input.WaveID); err != nil {
		return nil, fmt.Errorf("wave %d does not exist: %w", input.WaveID, err)
	}

	var results []dto.ProductDTO

	for _, masterID := range input.MasterIDs {
		master, err := uc.masterRepo.FindByID(masterID)
		if err != nil {
			return nil, fmt.Errorf("product master %d not found: %w", masterID, err)
		}

		existing, err := uc.productRepo.FindByWaveAndSKU(input.WaveID, master.SupplierPlatform, master.FactorySKU)
		if err == nil && existing != nil {
			results = append(results, productToDTO(existing))
			continue
		}

		mid := masterID
		product := &domain.Product{
			WaveID:           input.WaveID,
			ProductMasterID:  &mid,
			SupplierPlatform: master.SupplierPlatform,
			FactorySKU:       master.FactorySKU,
			Name:             master.Name,
			ExtraData:        "",
		}
		if err := uc.productRepo.Create(product); err != nil {
			return nil, err
		}
		results = append(results, productToDTO(product))
	}

	return results, nil
}

func (uc *productUseCase) ListProductsByWave(waveID uint) ([]dto.ProductDTO, error) {
	products, err := uc.productRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ProductDTO, len(products))
	for i := range products {
		result[i] = productToDTO(&products[i])
	}
	return result, nil
}

// ---- helpers ----

func productMasterToDTO(m *domain.ProductMaster) dto.ProductMasterDTO {
	return dto.ProductMasterDTO{
		ID:                 m.ID,
		SupplierPlatform:   m.SupplierPlatform,
		FactorySKU:         m.FactorySKU,
		SupplierProductRef: m.SupplierProductRef,
		Name:               m.Name,
		ProductKind:        string(m.ProductKind),
		Archived:           m.Archived,
		ExtraData:          m.ExtraData,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func productToDTO(p *domain.Product) dto.ProductDTO {
	return dto.ProductDTO{
		ID:               p.ID,
		WaveID:           p.WaveID,
		ProductMasterID:  p.ProductMasterID,
		SupplierPlatform: p.SupplierPlatform,
		FactorySKU:       p.FactorySKU,
		Name:             p.Name,
		ExtraData:        p.ExtraData,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}
