package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupShipmentLifecycleTestDB opens an isolated in-memory sqlite DB and
// migrates only the tables this test needs.
func setupShipmentLifecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.Shipment{}, &persistence.ShipmentLine{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func seedShipment(t *testing.T, db *gorm.DB) *domain.Shipment {
	t.Helper()
	ctx := context.Background()
	repo := infra.NewShipmentRepository(db)
	shipment := &domain.Shipment{
		SupplierOrderID:    1,
		SupplierPlatform:   "acme-factory",
		ShipmentNo:         "SHP-0001",
		ExternalShipmentNo: "EXT-0001",
		CarrierCode:        "sf",
		CarrierName:        "SF Express",
		TrackingNo:         "SF1234567890",
		Status:             string(domain.ShipmentStatusShipped),
		ExtraData:          `{"batch":"A"}`,
	}
	if err := repo.Create(ctx, shipment); err != nil {
		t.Fatalf("seed shipment: %v", err)
	}
	return shipment
}

func TestUpdateShipment_PersistsChanges(t *testing.T) {
	db := setupShipmentLifecycleTestDB(t)
	seeded := seedShipment(t, db)

	shipmentRepo := infra.NewShipmentRepository(db)
	writeRepo := infra.NewShipmentWriterRepository(db)
	uc := NewShipmentLifecycleUseCase(shipmentRepo, writeRepo)

	ctx := context.Background()
	updated, err := uc.UpdateShipment(ctx, dto.UpdateShipmentInput{
		ID:                 seeded.ID,
		SupplierPlatform:   "acme-factory",
		ShipmentNo:         "SHP-0001",
		ExternalShipmentNo: "EXT-0001-CORRECTED",
		CarrierCode:        "zto",
		CarrierName:        "ZTO Express",
		TrackingNo:         "ZTO9876543210",
	})
	if err != nil {
		t.Fatalf("UpdateShipment: %v", err)
	}
	if updated.CarrierCode != "zto" || updated.TrackingNo != "ZTO9876543210" {
		t.Fatalf("returned shipment not updated in memory: %+v", updated)
	}

	// Re-fetch independently to confirm the write actually persisted.
	reloaded, err := shipmentRepo.FindByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if reloaded.CarrierCode != "zto" {
		t.Errorf("carrierCode = %q, want %q", reloaded.CarrierCode, "zto")
	}
	if reloaded.CarrierName != "ZTO Express" {
		t.Errorf("carrierName = %q, want %q", reloaded.CarrierName, "ZTO Express")
	}
	if reloaded.TrackingNo != "ZTO9876543210" {
		t.Errorf("trackingNo = %q, want %q", reloaded.TrackingNo, "ZTO9876543210")
	}
	if reloaded.ExternalShipmentNo != "EXT-0001-CORRECTED" {
		t.Errorf("externalShipmentNo = %q, want %q", reloaded.ExternalShipmentNo, "EXT-0001-CORRECTED")
	}
	// Status must be untouched by UpdateShipment.
	if reloaded.Status != string(domain.ShipmentStatusShipped) {
		t.Errorf("status = %q, want unchanged %q", reloaded.Status, domain.ShipmentStatusShipped)
	}
}

func TestUpdateShipment_RejectsVoided(t *testing.T) {
	db := setupShipmentLifecycleTestDB(t)
	seeded := seedShipment(t, db)

	shipmentRepo := infra.NewShipmentRepository(db)
	writeRepo := infra.NewShipmentWriterRepository(db)
	uc := NewShipmentLifecycleUseCase(shipmentRepo, writeRepo)
	ctx := context.Background()

	if _, err := uc.VoidShipment(ctx, dto.VoidShipmentInput{ID: seeded.ID, Note: "wrong address", OperatorID: "op-1"}); err != nil {
		t.Fatalf("VoidShipment: %v", err)
	}

	if _, err := uc.UpdateShipment(ctx, dto.UpdateShipmentInput{ID: seeded.ID, CarrierCode: "sf"}); err == nil {
		t.Fatal("expected UpdateShipment on a voided shipment to fail, got nil error")
	}
}

func TestVoidShipment_TransitionsStatusAndRecordsNote(t *testing.T) {
	db := setupShipmentLifecycleTestDB(t)
	seeded := seedShipment(t, db)

	shipmentRepo := infra.NewShipmentRepository(db)
	writeRepo := infra.NewShipmentWriterRepository(db)
	uc := NewShipmentLifecycleUseCase(shipmentRepo, writeRepo)
	ctx := context.Background()

	voided, err := uc.VoidShipment(ctx, dto.VoidShipmentInput{
		ID:         seeded.ID,
		Note:       "customer requested cancellation",
		OperatorID: "op-42",
	})
	if err != nil {
		t.Fatalf("VoidShipment: %v", err)
	}
	if voided.Status != string(domain.ShipmentStatusVoided) {
		t.Fatalf("status = %q, want %q", voided.Status, domain.ShipmentStatusVoided)
	}

	reloaded, err := shipmentRepo.FindByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("FindByID after void: %v", err)
	}
	if reloaded.Status != string(domain.ShipmentStatusVoided) {
		t.Errorf("persisted status = %q, want %q", reloaded.Status, domain.ShipmentStatusVoided)
	}

	var extra map[string]any
	if err := json.Unmarshal([]byte(reloaded.ExtraData), &extra); err != nil {
		t.Fatalf("ExtraData is not valid JSON: %v (raw=%q)", err, reloaded.ExtraData)
	}
	if extra["voidNote"] != "customer requested cancellation" {
		t.Errorf("voidNote = %v, want %q", extra["voidNote"], "customer requested cancellation")
	}
	if extra["voidedBy"] != "op-42" {
		t.Errorf("voidedBy = %v, want %q", extra["voidedBy"], "op-42")
	}
	if extra["batch"] != "A" {
		t.Errorf("pre-existing ExtraData key %q lost on void, extra=%v", "batch", extra)
	}

	// Idempotent: voiding again should not error and should keep the same status.
	again, err := uc.VoidShipment(ctx, dto.VoidShipmentInput{ID: seeded.ID, Note: "second attempt"})
	if err != nil {
		t.Fatalf("second VoidShipment call: %v", err)
	}
	if again.Status != string(domain.ShipmentStatusVoided) {
		t.Errorf("re-void status = %q, want %q", again.Status, domain.ShipmentStatusVoided)
	}
}
