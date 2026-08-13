package infra

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupShipmentRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.Shipment{}, &persistence.ShipmentLine{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestSumShippedQuantityBySOLExcludesVoidedShipments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := setupShipmentRepoTestDB(t)
	repo := NewShipmentRepository(db)

	live := &domain.Shipment{SupplierOrderID: 1, ShipmentNo: "LIVE", Status: string(domain.ShipmentStatusShipped)}
	if err := repo.Create(ctx, live); err != nil {
		t.Fatalf("create live shipment: %v", err)
	}
	if err := repo.CreateLine(ctx, &domain.ShipmentLine{
		ShipmentID: live.ID, SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 2,
	}); err != nil {
		t.Fatalf("create live line: %v", err)
	}

	voided := &domain.Shipment{SupplierOrderID: 1, ShipmentNo: "VOID", Status: string(domain.ShipmentStatusVoided)}
	if err := repo.Create(ctx, voided); err != nil {
		t.Fatalf("create voided shipment: %v", err)
	}
	if err := repo.CreateLine(ctx, &domain.ShipmentLine{
		ShipmentID: voided.ID, SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 5,
	}); err != nil {
		t.Fatalf("create voided line: %v", err)
	}

	sum, err := repo.SumShippedQuantityBySOL(ctx, 10)
	if err != nil {
		t.Fatalf("SumShippedQuantityBySOL: %v", err)
	}
	if sum != 2 {
		t.Fatalf("sum = %d, want 2 (voided quantity excluded)", sum)
	}

	other, err := repo.SumShippedQuantityBySOL(ctx, 99)
	if err != nil {
		t.Fatalf("SumShippedQuantityBySOL empty SOL: %v", err)
	}
	if other != 0 {
		t.Fatalf("empty SOL sum = %d, want 0", other)
	}
}
