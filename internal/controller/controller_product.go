package controller

import (
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

// ProductController exposes product Wails bindings.
type ProductController struct {
	uc         app.ProductUseCase
	gdb        *gorm.DB
	assetStore *service.AssetStore
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
	uc = app.WithCatalogImportEvidence(uc, app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)))
	return &ProductController{uc: uc, gdb: gdb, assetStore: assetStore}
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
	templateRepo := infra.NewDocumentTemplateRepository(c.gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(c.gdb)
	profileRepo := infra.NewIntegrationProfileRepository(c.gdb)
	mapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	evidence := app.NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(c.gdb))
	if err := app.PrepareTemplateImportEvidence(ctx, evidence, mapping, app.PrepareTemplateImportEvidenceInput{
		ImportKind: "product_catalog", DocumentType: "import_product_catalog",
		IntegrationProfileID: input.IntegrationProfileID, ImportMode: input.ImportMode,
		FilePath: input.FilePath, Rows: input.Rows, IncludeZIPAssets: true,
	}); err != nil {
		return dto.ImportProductCatalogResult{}, fmt.Errorf("prepare catalog import evidence: %w", err)
	}
	if input.ImportMode == "reject_all" {
		stage, err := c.assetStore.BeginStage()
		if err != nil {
			return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
		}
		var result *dto.ImportProductCatalogResult
		err = c.gdb.Transaction(func(tx *gorm.DB) error {
			masterRepo := infra.NewProductMasterRepository(tx)
			productRepo := infra.NewProductRepository(tx)
			waveRepo := infra.NewWaveRepository(tx)
			templateRepo := infra.NewDocumentTemplateRepository(tx)
			bindingRepo := infra.NewProfileTemplateBindingRepository(tx)
			profileRepo := infra.NewIntegrationProfileRepository(tx)
			txMapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
			uc := app.NewProductUseCase(masterRepo, productRepo, waveRepo)
			uc = app.WithCatalogImportDeps(uc, txMapping, profileRepo, stage.Store())
			uc = app.WithCatalogImportEvidence(uc, evidence)
			imported, importErr := uc.ImportProductCatalog(ctx, input)
			if importErr != nil {
				return importErr
			}
			result = imported
			return stage.Commit()
		})
		if err != nil {
			_ = stage.Rollback()
			return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
		}
		if err := stage.Finalize(); err != nil {
			return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizePending(ctx))
		}
		if err := evidence.FinalizePending(ctx); err != nil {
			return dto.ImportProductCatalogResult{}, err
		}
		return *result, nil
	}
	masterRepo := infra.NewProductMasterRepository(c.gdb)
	productRepo := infra.NewProductRepository(c.gdb)
	waveRepo := infra.NewWaveRepository(c.gdb)
	uc := app.NewProductUseCase(masterRepo, productRepo, waveRepo)
	uc = app.WithCatalogImportDeps(uc, mapping, profileRepo, c.assetStore)
	uc = app.WithCatalogImportEvidence(uc, evidence)
	result, err := uc.ImportProductCatalog(ctx, input)
	if err != nil {
		return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportProductCatalogResult{}, err
	}
	return *result, nil
}
