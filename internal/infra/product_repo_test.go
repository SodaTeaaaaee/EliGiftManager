package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openProductRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	db.Exec("PRAGMA foreign_keys = ON;")
	if err := db.AutoMigrate(&persistence.ProductMaster{}, &persistence.Product{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestProductMasterCreateTrimsFactorySKU(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductMasterRepository(db)
	master := &domain.ProductMaster{
		SupplierPlatform: "factory-a",
		FactorySKU:       "  ABC  ",
		Name:             "Trimmed SKU",
		ProductKind:      domain.ProductKindOther,
	}
	if err := repo.Create(context.Background(), master); err != nil {
		t.Fatalf("create master: %v", err)
	}
	if master.FactorySKU != "ABC" {
		t.Errorf("Create returned FactorySKU = %q, want %q", master.FactorySKU, "ABC")
	}
	got, err := repo.FindByID(context.Background(), master.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got.FactorySKU != "ABC" {
		t.Errorf("stored FactorySKU = %q, want %q", got.FactorySKU, "ABC")
	}
}

func TestProductMasterCreateRejectsEmptyFactorySKU(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductMasterRepository(db)
	for _, sku := range []string{"", "   "} {
		master := &domain.ProductMaster{
			SupplierPlatform: "factory-a",
			FactorySKU:       sku,
			Name:             "Missing SKU",
			ProductKind:      domain.ProductKindOther,
		}
		err := repo.Create(context.Background(), master)
		if err == nil || !strings.Contains(err.Error(), "factory_sku is required") {
			t.Errorf("Create(%q) error = %v, want factory_sku is required", sku, err)
		}
	}
}

func TestProductMasterFindByPlatformAndSKUTrimsLookup(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductMasterRepository(db)
	master := &domain.ProductMaster{
		SupplierPlatform: "factory-a",
		FactorySKU:       "ABC",
		Name:             "Lookup SKU",
		ProductKind:      domain.ProductKindOther,
	}
	if err := repo.Create(context.Background(), master); err != nil {
		t.Fatalf("create master: %v", err)
	}
	got, err := repo.FindByPlatformAndSKU(context.Background(), "factory-a", " ABC ")
	if err != nil {
		t.Fatalf("FindByPlatformAndSKU: %v", err)
	}
	if got.ID != master.ID || got.FactorySKU != "ABC" {
		t.Errorf("got id=%d sku=%q, want id=%d sku=%q", got.ID, got.FactorySKU, master.ID, "ABC")
	}
}

func TestProductCreateAndFindByWaveAndSKUTrims(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductRepository(db)
	product := &domain.Product{
		WaveID:           1,
		SupplierPlatform: "factory-a",
		FactorySKU:       "  SKU-1  ",
		Name:             "Wave product",
	}
	if err := repo.Create(context.Background(), product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	if product.FactorySKU != "SKU-1" {
		t.Errorf("Create returned FactorySKU = %q, want %q", product.FactorySKU, "SKU-1")
	}
	got, err := repo.FindByWaveAndSKU(context.Background(), 1, "factory-a", " SKU-1 ")
	if err != nil {
		t.Fatalf("FindByWaveAndSKU: %v", err)
	}
	if got.ID != product.ID || got.FactorySKU != "SKU-1" {
		t.Errorf("got id=%d sku=%q, want id=%d sku=%q", got.ID, got.FactorySKU, product.ID, "SKU-1")
	}
}

func TestProductMasterSKUUniquenessRemainsCaseSensitive(t *testing.T) {
	t.Parallel()
	// idx_pm_platform_sku is a binary/case-sensitive SQLite unique index;
	// "abc" and "ABC" on the same platform are distinct as far as this repo is concerned.
	db := openProductRepoTestDB(t)
	repo := NewProductMasterRepository(db)
	lower := &domain.ProductMaster{
		SupplierPlatform: "factory-a",
		FactorySKU:       "abc",
		Name:             "lower",
		ProductKind:      domain.ProductKindOther,
	}
	upper := &domain.ProductMaster{
		SupplierPlatform: "factory-a",
		FactorySKU:       "ABC",
		Name:             "upper",
		ProductKind:      domain.ProductKindOther,
	}
	if err := repo.Create(context.Background(), lower); err != nil {
		t.Fatalf("create abc: %v", err)
	}
	if err := repo.Create(context.Background(), upper); err != nil {
		t.Fatalf("create ABC: %v", err)
	}
	if lower.ID == 0 || upper.ID == 0 || lower.ID == upper.ID {
		t.Fatalf("expected two distinct rows, got ids %d and %d", lower.ID, upper.ID)
	}
}

func TestProductMasterUpdateTrimsAndRejectsEmptyFactorySKU(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductMasterRepository(db)
	master := &domain.ProductMaster{
		SupplierPlatform: "factory-a",
		FactorySKU:       "KEEP",
		Name:             "Update SKU",
		ProductKind:      domain.ProductKindOther,
	}
	if err := repo.Create(context.Background(), master); err != nil {
		t.Fatalf("create master: %v", err)
	}
	master.FactorySKU = "  NEW  "
	if err := repo.Update(context.Background(), master); err != nil {
		t.Fatalf("update master: %v", err)
	}
	if master.FactorySKU != "NEW" {
		t.Errorf("Update returned FactorySKU = %q, want %q", master.FactorySKU, "NEW")
	}
	master.FactorySKU = "   "
	if err := repo.Update(context.Background(), master); err == nil || !strings.Contains(err.Error(), "factory_sku is required") {
		t.Errorf("Update empty SKU error = %v, want factory_sku is required", err)
	}
}

func TestProductCreateRejectsEmptyFactorySKU(t *testing.T) {
	t.Parallel()
	db := openProductRepoTestDB(t)
	repo := NewProductRepository(db)
	product := &domain.Product{
		WaveID:           1,
		SupplierPlatform: "factory-a",
		FactorySKU:       "   ",
		Name:             "Missing SKU",
	}
	err := repo.Create(context.Background(), product)
	if err == nil || !strings.Contains(err.Error(), "factory_sku is required") {
		t.Errorf("Create empty SKU error = %v, want factory_sku is required", err)
	}
}
