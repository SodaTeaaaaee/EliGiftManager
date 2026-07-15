package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProductUseCaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.Wave{},
		&persistence.ProductMaster{},
		&persistence.Product{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func TestCreateProductMaster_PersistsExtraDataAndImageFields(t *testing.T) {
	db := setupProductUseCaseTestDB(t)
	ctx := context.Background()
	uc := NewProductUseCase(
		infra.NewProductMasterRepository(db),
		infra.NewProductRepository(db),
		infra.NewWaveRepository(db),
	)

	created, err := uc.CreateProductMaster(ctx, dto.CreateProductMasterInput{
		SupplierPlatform: "rouzao",
		FactorySKU:       "SKU-IMG-1",
		Name:             "Cover Badge",
		ProductKind:      "badge",
		CoverImagePath:   "products/1/cover.webp",
		DetailImagePaths: `["products/1/d1.webp","products/1/d2.webp"]`,
		ExtraData:        `{"color":"gold"}`,
	})
	if err != nil {
		t.Fatalf("CreateProductMaster: %v", err)
	}
	if created.CoverImagePath != "products/1/cover.webp" {
		t.Errorf("CoverImagePath = %q, want products/1/cover.webp", created.CoverImagePath)
	}
	if created.DetailImagePaths != `["products/1/d1.webp","products/1/d2.webp"]` {
		t.Errorf("DetailImagePaths = %q", created.DetailImagePaths)
	}
	if created.ExtraData != `{"color":"gold"}` {
		t.Errorf("ExtraData = %q, want {\"color\":\"gold\"}", created.ExtraData)
	}

	// Round-trip through List to confirm persistence/mapper carry the fields.
	listed, err := uc.ListProductMasters(ctx)
	if err != nil {
		t.Fatalf("ListProductMasters: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 master, got %d", len(listed))
	}
	if listed[0].CoverImagePath != "products/1/cover.webp" {
		t.Errorf("listed CoverImagePath = %q", listed[0].CoverImagePath)
	}
	if listed[0].ExtraData != `{"color":"gold"}` {
		t.Errorf("listed ExtraData = %q", listed[0].ExtraData)
	}
}

func TestUpdateProductMaster_WritesExtraDataAndImageFields(t *testing.T) {
	db := setupProductUseCaseTestDB(t)
	ctx := context.Background()
	uc := NewProductUseCase(
		infra.NewProductMasterRepository(db),
		infra.NewProductRepository(db),
		infra.NewWaveRepository(db),
	)

	created, err := uc.CreateProductMaster(ctx, dto.CreateProductMasterInput{
		SupplierPlatform: "rouzao",
		FactorySKU:       "SKU-UPD-1",
		Name:             "Badge",
		ProductKind:      "badge",
		CoverImagePath:   "old/cover.webp",
		ExtraData:        `{"v":1}`,
	})
	if err != nil {
		t.Fatalf("CreateProductMaster: %v", err)
	}

	updated, err := uc.UpdateProductMaster(ctx, dto.UpdateProductMasterInput{
		ID:                 created.ID,
		SupplierPlatform:   "rouzao",
		FactorySKU:         "SKU-UPD-1",
		Name:               "Badge Updated",
		ProductKind:        "badge",
		Archived:           false,
		CoverImagePath:     "new/cover.webp",
		DetailImagePaths:   `["new/d1.webp"]`,
		ExtraData:          `{"v":2,"note":"updated"}`,
	})
	if err != nil {
		t.Fatalf("UpdateProductMaster: %v", err)
	}
	if updated.CoverImagePath != "new/cover.webp" {
		t.Errorf("CoverImagePath = %q, want new/cover.webp", updated.CoverImagePath)
	}
	if updated.DetailImagePaths != `["new/d1.webp"]` {
		t.Errorf("DetailImagePaths = %q", updated.DetailImagePaths)
	}
	if updated.ExtraData != `{"v":2,"note":"updated"}` {
		t.Errorf("ExtraData = %q", updated.ExtraData)
	}
}

func TestSnapshotProductsForWave_CopiesMasterExtraData(t *testing.T) {
	db := setupProductUseCaseTestDB(t)
	ctx := context.Background()
	masterRepo := infra.NewProductMasterRepository(db)
	productRepo := infra.NewProductRepository(db)
	waveRepo := infra.NewWaveRepository(db)
	uc := NewProductUseCase(masterRepo, productRepo, waveRepo)

	wave := &domain.Wave{WaveNo: "W-EXTRA-1", Name: "ExtraData Wave", WaveType: "mixed"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}

	master := &domain.ProductMaster{
		SupplierPlatform: "rouzao",
		FactorySKU:       "SKU-EXTRA",
		Name:             "Extra Badge",
		ProductKind:      "badge",
		CoverImagePath:   "products/9/cover.webp",
		DetailImagePaths: `["products/9/d1.webp"]`,
		ExtraData:        `{"material":"acrylic","batch":"A"}`,
	}
	if err := masterRepo.Create(ctx, master); err != nil {
		t.Fatalf("create master: %v", err)
	}

	products, err := uc.SnapshotProductsForWave(ctx, dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{master.ID},
	})
	if err != nil {
		t.Fatalf("SnapshotProductsForWave: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].ExtraData != `{"material":"acrylic","batch":"A"}` {
		t.Errorf("product ExtraData = %q, want master ExtraData copied", products[0].ExtraData)
	}
	// Image fields must stay Master-only — Product DTO has no image fields, so
	// also assert the persisted Product row has ExtraData only (no image bleed).
	stored, err := productRepo.FindByWaveAndSKU(ctx, wave.ID, "rouzao", "SKU-EXTRA")
	if err != nil {
		t.Fatalf("FindByWaveAndSKU: %v", err)
	}
	if stored.ExtraData != master.ExtraData {
		t.Errorf("stored ExtraData = %q, want %q", stored.ExtraData, master.ExtraData)
	}
}

func TestSnapshotProductsForWaveDetailed_CopiesMasterExtraData(t *testing.T) {
	db := setupProductUseCaseTestDB(t)
	ctx := context.Background()
	masterRepo := infra.NewProductMasterRepository(db)
	productRepo := infra.NewProductRepository(db)
	waveRepo := infra.NewWaveRepository(db)
	uc := NewProductSnapshotDetailUseCase(masterRepo, productRepo, waveRepo)

	wave := &domain.Wave{WaveNo: "W-EXTRA-2", Name: "Detail Extra Wave", WaveType: "mixed"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}

	master := &domain.ProductMaster{
		SupplierPlatform: "rouzao",
		FactorySKU:       "SKU-DETAIL-EXTRA",
		Name:             "Detail Extra Badge",
		ProductKind:      "badge",
		ExtraData:        `{"src":"detail-usecase"}`,
	}
	if err := masterRepo.Create(ctx, master); err != nil {
		t.Fatalf("create master: %v", err)
	}

	result, err := uc.SnapshotProductsForWaveDetailed(ctx, dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{master.ID},
	})
	if err != nil {
		t.Fatalf("SnapshotProductsForWaveDetailed: %v", err)
	}
	if result.CreatedCount != 1 || len(result.Items) != 1 {
		t.Fatalf("expected 1 created item, got created=%d items=%d", result.CreatedCount, len(result.Items))
	}
	if result.Items[0].Product.ExtraData != `{"src":"detail-usecase"}` {
		t.Errorf("detailed product ExtraData = %q, want master ExtraData copied", result.Items[0].Product.ExtraData)
	}
}
