package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// ProductController exposes product Wails bindings.
type ProductController struct {
	uc app.ProductUseCase
}

func NewProductController() *ProductController {
	gdb := database.GetDB()
	masterRepo := infra.NewProductMasterRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	mapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	// AssetStore is optional for non-image catalog imports; zip imageLayout needs it.
	assetStore, _ := service.NewAssetStore()
	uc := app.NewProductUseCase(masterRepo, productRepo, waveRepo)
	uc = app.WithCatalogImportDeps(uc, mapping, profileRepo, assetStore)
	return &ProductController{uc: uc}
}

// CreateProductMaster creates a new product master record.
func (c *ProductController) CreateProductMaster(input dto.CreateProductMasterInput) (*dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.CreateProductMaster(ctx, input)
}

// ListProductMasters returns all product masters.
func (c *ProductController) ListProductMasters() ([]dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.ListProductMasters(ctx)
}

// UpdateProductMaster updates an existing product master.
func (c *ProductController) UpdateProductMaster(input dto.UpdateProductMasterInput) (*dto.ProductMasterDTO, error) {
	ctx := appContext
	return c.uc.UpdateProductMaster(ctx, input)
}

// SnapshotProductsForWave creates wave-scoped product snapshots from master IDs.
func (c *ProductController) SnapshotProductsForWave(input dto.SnapshotProductsInput) ([]dto.ProductDTO, error) {
	ctx := appContext
	return c.uc.SnapshotProductsForWave(ctx, input)
}

// ListProductsByWave returns all products snapshotted into a wave.
func (c *ProductController) ListProductsByWave(waveID uint) ([]dto.ProductDTO, error) {
	ctx := appContext
	return c.uc.ListProductsByWave(ctx, waveID)
}

// ImportProductCatalog upserts ProductMaster rows from a template-mapped catalog sheet.
func (c *ProductController) ImportProductCatalog(input dto.ImportProductCatalogInput) (dto.ImportProductCatalogResult, error) {
	ctx := appContext
	result, err := c.uc.ImportProductCatalog(ctx, input)
	if err != nil {
		return dto.ImportProductCatalogResult{}, err
	}
	return *result, nil
}
