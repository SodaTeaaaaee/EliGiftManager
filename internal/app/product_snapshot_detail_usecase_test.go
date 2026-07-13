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

func setupProductSnapshotDetailTestDB(t *testing.T) *gorm.DB {
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

func TestProductSnapshotDetailUseCase_ReportsCreatedAndSkipped(t *testing.T) {
	db := setupProductSnapshotDetailTestDB(t)
	ctx := context.Background()

	masterRepo := infra.NewProductMasterRepository(db)
	productRepo := infra.NewProductRepository(db)
	waveRepo := infra.NewWaveRepository(db)
	uc := NewProductSnapshotDetailUseCase(masterRepo, productRepo, waveRepo)

	wave := &domain.Wave{WaveNo: "W-0001", Name: "Test Wave", WaveType: "mixed"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}

	masterA := &domain.ProductMaster{SupplierPlatform: "taobao", FactorySKU: "SKU-A", Name: "Badge A", ProductKind: "badge"}
	if err := masterRepo.Create(ctx, masterA); err != nil {
		t.Fatalf("create master A: %v", err)
	}
	masterB := &domain.ProductMaster{SupplierPlatform: "taobao", FactorySKU: "SKU-B", Name: "Badge B", ProductKind: "badge"}
	if err := masterRepo.Create(ctx, masterB); err != nil {
		t.Fatalf("create master B: %v", err)
	}

	// First call: both masters are new to this wave -> both created.
	first, err := uc.SnapshotProductsForWaveDetailed(ctx, dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{masterA.ID, masterB.ID},
	})
	if err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if first.CreatedCount != 2 || first.SkippedCount != 0 {
		t.Fatalf("expected 2 created / 0 skipped, got created=%d skipped=%d", first.CreatedCount, first.SkippedCount)
	}
	for _, item := range first.Items {
		if item.AlreadyExisted {
			t.Fatalf("expected all items newly created on first call, got AlreadyExisted for master %d", item.MasterID)
		}
	}

	// Second call with the same master A (already snapshotted) plus master B
	// again: both should now be reported as already-existed (skip-detail).
	second, err := uc.SnapshotProductsForWaveDetailed(ctx, dto.SnapshotProductsInput{
		WaveID:    wave.ID,
		MasterIDs: []uint{masterA.ID, masterB.ID},
	})
	if err != nil {
		t.Fatalf("second snapshot: %v", err)
	}
	if second.CreatedCount != 0 || second.SkippedCount != 2 {
		t.Fatalf("expected 0 created / 2 skipped, got created=%d skipped=%d", second.CreatedCount, second.SkippedCount)
	}
	for _, item := range second.Items {
		if !item.AlreadyExisted {
			t.Fatalf("expected AlreadyExisted=true for master %d on repeat snapshot", item.MasterID)
		}
	}
}

func TestProductSnapshotDetailUseCase_RequiresWaveID(t *testing.T) {
	db := setupProductSnapshotDetailTestDB(t)
	ctx := context.Background()

	uc := NewProductSnapshotDetailUseCase(
		infra.NewProductMasterRepository(db),
		infra.NewProductRepository(db),
		infra.NewWaveRepository(db),
	)

	if _, err := uc.SnapshotProductsForWaveDetailed(ctx, dto.SnapshotProductsInput{}); err == nil {
		t.Fatal("expected error when wave_id is missing")
	}
}
