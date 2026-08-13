package controller

import (
	"context"
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
	uc            app.ProductUseCase
	gdb           *gorm.DB
	assetStore    *service.AssetStore
	assetStoreErr error
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
	// AssetStore is optional for non-image catalog imports; ImageLayout needs it.
	assetStore, assetErr := service.NewAssetStore()
	uc := app.NewProductUseCase(masterRepo, productRepo, waveRepo)
	uc = app.WithCatalogImportDeps(uc, mapping, profileRepo, assetStore)
	uc = app.WithCatalogImportEvidence(uc, app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)))
	return &ProductController{uc: uc, gdb: gdb, assetStore: assetStore, assetStoreErr: assetErr}
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
// All Creates run in one DB transaction so a later failure cannot leave a partial snapshot set.
func (c *ProductController) SnapshotProductsForWave(input dto.SnapshotProductsInput) ([]dto.ProductDTO, error) {
	ctx := appContext
	if c.gdb == nil {
		if c.uc == nil {
			return nil, fmt.Errorf("product database is not configured")
		}
		return c.uc.SnapshotProductsForWave(ctx, input)
	}
	var results []dto.ProductDTO
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		uc := app.NewProductUseCase(
			infra.NewProductMasterRepository(tx),
			infra.NewProductRepository(tx),
			infra.NewWaveRepository(tx),
		)
		products, snapErr := uc.SnapshotProductsForWave(ctx, input)
		if snapErr != nil {
			return snapErr
		}
		results = products
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ListProductsByWave returns all products snapshotted into a wave.
func (c *ProductController) ListProductsByWave(waveID uint) ([]dto.ProductDTO, error) {
	ctx := appContext
	return c.uc.ListProductsByWave(ctx, waveID)
}

// ImportProductCatalog upserts ProductMaster rows from a template-mapped catalog sheet.
func (c *ProductController) ImportProductCatalog(input dto.ImportProductCatalogInput) (dto.ImportProductCatalogResult, error) {
	ctx := appContext
	if c.gdb == nil {
		return dto.ImportProductCatalogResult{}, fmt.Errorf("product catalog import database is not configured")
	}
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
	if _, rules, resolveErr := mapping.ResolveTemplateAndRules(ctx, input.IntegrationProfileID, "import_product_catalog"); resolveErr == nil {
		if catalogImageLayoutEnabled(rules) {
			if err := c.requireAssetStore(); err != nil {
				return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
			}
		}
	}

	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	// reject_all with a working store stages images so a rolled-back transaction
	// cannot leave published assets. skip_invalid and non-image (non-ZIP) imports
	// still run inside a DB transaction, matching that ZIP/reject_all wrapping.
	if mode == "reject_all" && c.assetStore != nil {
		return c.importProductCatalogStaged(ctx, input, evidence)
	}
	return c.importProductCatalogInTransaction(ctx, input, evidence, c.assetStore)
}

func catalogImageLayoutEnabled(rules *app.TemplateMappingRules) bool {
	return rules != nil && rules.ImageLayout != nil && rules.ImageLayout.Enabled
}

func (c *ProductController) requireAssetStore() error {
	if c.assetStore != nil {
		return nil
	}
	if c.assetStoreErr != nil {
		return fmt.Errorf("asset store: %w", c.assetStoreErr)
	}
	return fmt.Errorf("asset store is not configured")
}

func (c *ProductController) importProductCatalogStaged(ctx context.Context, input dto.ImportProductCatalogInput, evidence *app.ImportEvidenceUseCase) (dto.ImportProductCatalogResult, error) {
	stage, err := c.assetStore.BeginStage()
	if err != nil {
		return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	var result *dto.ImportProductCatalogResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		imported, importErr := c.runProductCatalogImport(ctx, tx, input, evidence, stage.Store())
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

func (c *ProductController) importProductCatalogInTransaction(ctx context.Context, input dto.ImportProductCatalogInput, evidence *app.ImportEvidenceUseCase, store *service.AssetStore) (dto.ImportProductCatalogResult, error) {
	var result *dto.ImportProductCatalogResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		imported, importErr := c.runProductCatalogImport(ctx, tx, input, evidence, store)
		if importErr != nil {
			return importErr
		}
		result = imported
		return nil
	})
	if err != nil {
		return dto.ImportProductCatalogResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportProductCatalogResult{}, err
	}
	return *result, nil
}

func (c *ProductController) runProductCatalogImport(ctx context.Context, tx *gorm.DB, input dto.ImportProductCatalogInput, evidence *app.ImportEvidenceUseCase, store *service.AssetStore) (*dto.ImportProductCatalogResult, error) {
	masterRepo := infra.NewProductMasterRepository(tx)
	productRepo := infra.NewProductRepository(tx)
	waveRepo := infra.NewWaveRepository(tx)
	templateRepo := infra.NewDocumentTemplateRepository(tx)
	bindingRepo := infra.NewProfileTemplateBindingRepository(tx)
	profileRepo := infra.NewIntegrationProfileRepository(tx)
	txMapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	uc := app.NewProductUseCase(masterRepo, productRepo, waveRepo)
	uc = app.WithCatalogImportDeps(uc, txMapping, profileRepo, store)
	uc = app.WithCatalogImportEvidence(uc, evidence)
	return uc.ImportProductCatalog(ctx, input)
}
