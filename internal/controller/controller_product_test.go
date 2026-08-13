package controller

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProductTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.ProductMaster{},
		&persistence.Product{},
		&persistence.Wave{},
		&persistence.IntegrationProfile{},
		&persistence.DocumentTemplate{},
		&persistence.IntegrationProfileTemplateBinding{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	seedEnabledFeaturePolicy(t, gdb)
	return gdb
}

func newProductTestController(gdb *gorm.DB) *ProductController {
	tUC := app.NewProductUseCase(
		infra.NewProductMasterRepository(gdb),
		infra.NewProductRepository(gdb),
		infra.NewWaveRepository(gdb),
	)
	return &ProductController{uc: tUC, gdb: gdb}
}

func TestSnapshotProductsForWave_RollsBackPartialCreates(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)

	wave := &domain.Wave{WaveNo: "W-SNAP-1", Name: "Snapshot Wave"}
	if err := infra.NewWaveRepository(gdb).Create(appContext, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}
	gold, err := c.CreateProductMaster(dto.CreateProductMasterInput{
		SupplierPlatform: "factory-a", FactorySKU: "GOLD-1", Name: "Gold", ProductKind: "badge",
	})
	if err != nil {
		t.Fatalf("create gold master: %v", err)
	}

	_, err = c.SnapshotProductsForWave(dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{gold.ID, 99999},
	})
	if err == nil {
		t.Fatal("expected snapshot to fail on missing master")
	}

	products, listErr := c.ListProductsByWave(wave.ID)
	if listErr != nil {
		t.Fatalf("ListProductsByWave: %v", listErr)
	}
	if len(products) != 0 {
		t.Fatalf("partial snapshot leaked after rollback: %+v", products)
	}
}

func TestSnapshotProductsForWave_CommitsAllCreates(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)

	wave := &domain.Wave{WaveNo: "W-SNAP-2", Name: "Snapshot Wave"}
	if err := infra.NewWaveRepository(gdb).Create(appContext, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}
	gold, err := c.CreateProductMaster(dto.CreateProductMasterInput{
		SupplierPlatform: "factory-a", FactorySKU: "GOLD-1", Name: "Gold", ProductKind: "badge",
	})
	if err != nil {
		t.Fatalf("create gold master: %v", err)
	}
	silver, err := c.CreateProductMaster(dto.CreateProductMasterInput{
		SupplierPlatform: "factory-a", FactorySKU: "SILVER-1", Name: "Silver", ProductKind: "charm",
	})
	if err != nil {
		t.Fatalf("create silver master: %v", err)
	}

	products, err := c.SnapshotProductsForWave(dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{gold.ID, silver.ID},
	})
	if err != nil {
		t.Fatalf("SnapshotProductsForWave: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("got %d products, want 2", len(products))
	}
	listed, err := c.ListProductsByWave(wave.ID)
	if err != nil {
		t.Fatalf("ListProductsByWave: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("listed %d products, want 2", len(listed))
	}
}

func TestImportProductCatalog_SkipInvalidNonZIPUsesTransaction(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)
	profileID := seedCatalogImportFixture(t, gdb, catalogFixtureOpts{})

	path := writeTempCatalogCSV(t, "SKU,Name,Kind\nGOLD-1,Gold Badge,badge\nBAD-1,Broken,not-a-kind\nSILVER-1,Silver Charm,charm\n")
	result, err := c.ImportProductCatalog(dto.ImportProductCatalogInput{
		IntegrationProfileID: profileID,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.SuccessCount != 2 || result.CreatedCount != 2 || result.ErrorCount < 1 {
		t.Fatalf("result=%+v errors=%+v", result, result.Errors)
	}
	masters, err := c.ListProductMasters()
	if err != nil {
		t.Fatalf("ListProductMasters: %v", err)
	}
	if len(masters) != 2 {
		t.Fatalf("persisted %d masters, want 2: %+v", len(masters), masters)
	}
}

func TestImportProductCatalog_RejectAllNonZIPRollsBack(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)
	profileID := seedCatalogImportFixture(t, gdb, catalogFixtureOpts{})

	path := writeTempCatalogCSV(t, "SKU,Name,Kind\nGOLD-1,Gold Badge,badge\nBAD-1,Broken,not-a-kind\n")
	result, err := c.ImportProductCatalog(dto.ImportProductCatalogInput{
		IntegrationProfileID: profileID,
		ImportMode:           "reject_all",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.SuccessCount != 0 || result.ErrorCount < 1 {
		t.Fatalf("result=%+v", result)
	}
	masters, err := c.ListProductMasters()
	if err != nil {
		t.Fatalf("ListProductMasters: %v", err)
	}
	if len(masters) != 0 {
		t.Fatalf("reject_all leaked masters: %+v", masters)
	}
}

func TestImportProductCatalog_ImageLayoutSurfacesAssetStoreError(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)
	c.assetStoreErr = errors.New("assets dir missing")
	profileID := seedCatalogImportFixture(t, gdb, catalogFixtureOpts{imageLayout: true})

	path := writeTempCatalogCSV(t, "SKU,Name,Kind\nGOLD-1,Gold Badge,badge\n")
	_, err := c.ImportProductCatalog(dto.ImportProductCatalogInput{
		IntegrationProfileID: profileID,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err == nil {
		t.Fatal("expected asset store error when ImageLayout is enabled")
	}
	if !strings.Contains(err.Error(), "assets dir missing") && !strings.Contains(err.Error(), "asset store") {
		t.Fatalf("unexpected error: %v", err)
	}
	masters, listErr := c.ListProductMasters()
	if listErr != nil {
		t.Fatalf("ListProductMasters: %v", listErr)
	}
	if len(masters) != 0 {
		t.Fatalf("import persisted masters despite asset store failure: %+v", masters)
	}
}

func TestImportProductCatalog_ImageLayoutWithStoreSucceeds(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)
	c.assetStore = service.NewAssetStoreAt(t.TempDir())
	profileID := seedCatalogImportFixture(t, gdb, catalogFixtureOpts{imageLayout: true})

	path := writeTempCatalogCSV(t, "SKU,Name,Kind\nGOLD-1,Gold Badge,badge\n")
	result, err := c.ImportProductCatalog(dto.ImportProductCatalogInput{
		IntegrationProfileID: profileID,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.SuccessCount != 1 || result.CreatedCount != 1 {
		t.Fatalf("result=%+v errors=%+v", result, result.Errors)
	}
}

func TestImportProductCatalog_SkipInvalidImageAttachDoesNotCountSuccess(t *testing.T) {
	gdb := setupProductTestDB(t)
	c := newProductTestController(gdb)
	c.assetStore = service.NewAssetStoreAt(t.TempDir())
	profileID := seedCatalogImportFixture(t, gdb, catalogFixtureOpts{imageLayout: true, coverPick: "highest"})

	path := writeTempCatalogCSV(t, "SKU,Name,Kind\nGOLD-1,Gold Badge,badge\nSILVER-1,Silver Charm,charm\n")
	result, err := c.ImportProductCatalog(dto.ImportProductCatalogInput{
		IntegrationProfileID: profileID,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("ImportProductCatalog: %v", err)
	}
	if result.SuccessCount != 0 || result.CreatedCount != 0 {
		t.Fatalf("image attach failure counted as success: %+v errors=%+v", result, result.Errors)
	}
	if result.ErrorCount < 2 {
		t.Fatalf("expected image errors for both rows, got %+v", result)
	}
	masters, err := c.ListProductMasters()
	if err != nil {
		t.Fatalf("ListProductMasters: %v", err)
	}
	if len(masters) != 0 {
		t.Fatalf("skip_invalid persisted rows after image attach failure: %+v", masters)
	}
}

type catalogFixtureOpts struct {
	imageLayout bool
	coverPick   string
}

func seedCatalogImportFixture(t *testing.T, gdb *gorm.DB, opts catalogFixtureOpts) uint {
	t.Helper()
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:                   "catalog-import-profile",
		SourceChannel:                "rouzao",
		SourceSurface:                string(domain.SourceSurfaceFactory),
		SupportsImportProductCatalog: true,
		FactorySupplierPlatform:      "rouzao",
		TrackingSyncMode:             "unsupported",
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	mappingRules := `{
		"version": 2, "mode": "header", "hasHeader": true,
		"columns": {
			"product.factory_sku": "SKU",
			"product.name": "Name",
			"product.product_kind": "Kind"
		}
	}`
	if opts.imageLayout {
		coverPick := opts.coverPick
		if coverPick == "" {
			coverPick = "lowest_nn"
		}
		mappingRules = `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {
				"product.factory_sku": "SKU",
				"product.name": "Name",
				"product.product_kind": "Kind"
			},
			"imageLayout": {
				"enabled": true,
				"matchField": "product.name",
				"coverDir": "covers",
				"detailDir": "details",
				"coverPick": "` + coverPick + `"
			}
		}`
	}

	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "catalog-import-template",
		DocumentType: "import_product_catalog",
		Format:       "csv",
		MappingRules: mappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}

	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_product_catalog",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	return profile.ID
}

func writeTempCatalogCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catalog.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}
