package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// buildImportFixture sets up three groups across two supplier order lines.
//
// Wave 1 has supplier order 1.
// SOL 10 → FL 100 (submittedQty=5), SOL 11 → FL 101 (submittedQty=5).
// Group A ("EXT-A"): SOL 10, FL 100, qty 1  — valid
// Group B ("EXT-B"): SOL 11, FL 101, qty 1  — valid
// Group C ("EXT-C"): SOL 99 (nonexistent)   — invalid
func buildImportFixture() (
	*mockShipmentRepo,
	*mockSupplierRepoForShipment,
	*mockFulfillRepoForShipment,
) {
	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()

	now := "2026-01-01T00:00:00Z"
	supplierRepo.orders[1] = &domain.SupplierOrder{
		ID: 1, WaveID: 1, Status: "draft", SupplierPlatform: "test",
		CreatedAt: now, UpdatedAt: now,
	}
	supplierRepo.orderLines[10] = &domain.SupplierOrderLine{
		ID: 10, SupplierOrderID: 1, FulfillmentLineID: 100, SubmittedQuantity: 5,
	}
	supplierRepo.orderLines[11] = &domain.SupplierOrderLine{
		ID: 11, SupplierOrderID: 1, FulfillmentLineID: 101, SubmittedQuantity: 5,
	}
	fulfillRepo.lines[100] = &domain.FulfillmentLine{ID: 100, WaveID: 1}
	fulfillRepo.lines[101] = &domain.FulfillmentLine{ID: 101, WaveID: 1}

	return shipmentRepo, supplierRepo, fulfillRepo
}

func threeGroupEntries() []dto.ImportShipmentEntry {
	return []dto.ImportShipmentEntry{
		{ExternalShipmentNo: "EXT-A", SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-A"},
		{ExternalShipmentNo: "EXT-B", SupplierOrderLineID: 11, FulfillmentLineID: 101, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-B"},
		{ExternalShipmentNo: "EXT-C", SupplierOrderLineID: 99, FulfillmentLineID: 999, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-C"},
	}
}

// TestImportShipmentsSkipInvalidMode: 3 groups, 1 invalid → 2 succeed, 1 error reported.
func TestImportShipmentsSkipInvalidMode(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()
	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)

	result, err := uc.ImportShipments(dto.ImportShipmentInput{
		WaveID:     1,
		ImportMode: "skip_invalid",
		Entries:    threeGroupEntries(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalProcessed != 3 {
		t.Errorf("TotalProcessed = %d, want 3", result.TotalProcessed)
	}
	if result.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", result.SuccessCount)
	}
	if result.ErrorCount == 0 {
		t.Error("ErrorCount = 0, want > 0 for the invalid group")
	}
	if len(result.Errors) == 0 {
		t.Error("Errors is empty, want at least one error entry")
	}
	if len(shipmentRepo.shipments) != 2 {
		t.Errorf("persisted shipments = %d, want 2", len(shipmentRepo.shipments))
	}
}

// TestImportShipmentsRejectAllMode: 3 groups, 1 invalid → 0 succeed, errors reported, nothing persisted.
func TestImportShipmentsRejectAllMode(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()
	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)

	result, err := uc.ImportShipments(dto.ImportShipmentInput{
		WaveID:     1,
		ImportMode: "reject_all",
		Entries:    threeGroupEntries(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0 (reject_all must persist nothing)", result.SuccessCount)
	}
	if result.ErrorCount == 0 {
		t.Error("ErrorCount = 0, want > 0")
	}
	if len(result.Errors) == 0 {
		t.Error("Errors is empty, want at least one error entry")
	}
	if len(shipmentRepo.shipments) != 0 {
		t.Errorf("persisted shipments = %d, want 0 (reject_all must not persist anything)", len(shipmentRepo.shipments))
	}
}

// TestImportShipmentsRejectAllCleanOnAllValid: 3 groups, all valid → all succeed.
func TestImportShipmentsRejectAllCleanOnAllValid(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()

	// Add a third valid SOL/FL pair.
	supplierRepo.orderLines[12] = &domain.SupplierOrderLine{
		ID: 12, SupplierOrderID: 1, FulfillmentLineID: 102, SubmittedQuantity: 5,
	}
	fulfillRepo.lines[102] = &domain.FulfillmentLine{ID: 102, WaveID: 1}

	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)

	entries := []dto.ImportShipmentEntry{
		{ExternalShipmentNo: "EXT-A", SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-A"},
		{ExternalShipmentNo: "EXT-B", SupplierOrderLineID: 11, FulfillmentLineID: 101, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-B"},
		{ExternalShipmentNo: "EXT-C", SupplierOrderLineID: 12, FulfillmentLineID: 102, Quantity: 1, CarrierCode: "SF", TrackingNo: "T-C"},
	}

	result, err := uc.ImportShipments(dto.ImportShipmentInput{
		WaveID:     1,
		ImportMode: "reject_all",
		Entries:    entries,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", result.SuccessCount)
	}
	if result.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", result.ErrorCount)
	}
	if len(shipmentRepo.shipments) != 3 {
		t.Errorf("persisted shipments = %d, want 3", len(shipmentRepo.shipments))
	}
}

// TestImportShipmentsDefaultModeIsSkipInvalid: empty ImportMode → behaves like skip_invalid.
func TestImportShipmentsDefaultModeIsSkipInvalid(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()
	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)

	result, err := uc.ImportShipments(dto.ImportShipmentInput{
		WaveID:     1,
		ImportMode: "", // empty → default skip_invalid
		Entries:    threeGroupEntries(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same expectations as skip_invalid: 2 succeed, 1 error, 2 persisted.
	if result.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2 (default should be skip_invalid)", result.SuccessCount)
	}
	if result.ErrorCount == 0 {
		t.Error("ErrorCount = 0, want > 0")
	}
	if len(shipmentRepo.shipments) != 2 {
		t.Errorf("persisted shipments = %d, want 2", len(shipmentRepo.shipments))
	}
}
