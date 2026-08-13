package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// findLineResult returns the result entry for the given SOL LineID, or nil.
func findLineResult(result []dto.SupplierOrderLineShippedDTO, lineID uint) *dto.SupplierOrderLineShippedDTO {
	for i := range result {
		if result[i].LineID == lineID {
			return &result[i]
		}
	}
	return nil
}

// TestGetSupplierOrderLineShippedSummary_PartiallyShipped: a line with
// submitted=5 and 2 already shipped reports shipped=2, remaining=3.
func TestGetSupplierOrderLineShippedSummary_PartiallyShipped(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, _ := buildImportFixture()
	// buildImportFixture() sets up SOL 10 -> FL 100, SubmittedQuantity=5, under
	// SupplierOrder 1. Record 2 units already shipped against SOL 10.
	shipmentRepo.shipmentLines[1] = []*domain.ShipmentLine{
		{ID: 1, ShipmentID: 1, SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 2},
	}

	uc := NewShipmentReconciliationQueryUseCase(supplierRepo, shipmentRepo)

	result, err := uc.GetSupplierOrderLineShippedSummary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line10 := findLineResult(result, 10)
	if line10 == nil {
		t.Fatalf("expected a result entry for SOL 10, got %+v", result)
	}
	if line10.SubmittedQuantity != 5 {
		t.Errorf("SubmittedQuantity = %d, want 5", line10.SubmittedQuantity)
	}
	if line10.ShippedQuantity != 2 {
		t.Errorf("ShippedQuantity = %d, want 2", line10.ShippedQuantity)
	}
	if line10.RemainingQuantity != 3 {
		t.Errorf("RemainingQuantity = %d, want 3", line10.RemainingQuantity)
	}
}

// TestGetSupplierOrderLineShippedSummary_FullyShipped: a line that has been shipped
// in full reports remaining=0.
func TestGetSupplierOrderLineShippedSummary_FullyShipped(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, _ := buildImportFixture()
	// SOL 11 -> FL 101, SubmittedQuantity=5. Ship the full 5 units.
	shipmentRepo.shipmentLines[2] = []*domain.ShipmentLine{
		{ID: 2, ShipmentID: 2, SupplierOrderLineID: 11, FulfillmentLineID: 101, Quantity: 5},
	}

	uc := NewShipmentReconciliationQueryUseCase(supplierRepo, shipmentRepo)

	result, err := uc.GetSupplierOrderLineShippedSummary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line11 := findLineResult(result, 11)
	if line11 == nil {
		t.Fatalf("expected a result entry for SOL 11, got %+v", result)
	}
	if line11.ShippedQuantity != 5 {
		t.Errorf("ShippedQuantity = %d, want 5", line11.ShippedQuantity)
	}
	if line11.RemainingQuantity != 0 {
		t.Errorf("RemainingQuantity = %d, want 0", line11.RemainingQuantity)
	}
}

func TestGetSupplierOrderLineShippedSummary_ExcludesVoidedShipments(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, _ := buildImportFixture()
	shipmentRepo.shipments[1] = &domain.Shipment{
		ID:              1,
		SupplierOrderID: 1,
		Status:          string(domain.ShipmentStatusShipped),
	}
	shipmentRepo.shipmentLines[1] = []*domain.ShipmentLine{
		{ID: 1, ShipmentID: 1, SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 2},
	}
	shipmentRepo.shipments[3] = &domain.Shipment{
		ID:              3,
		SupplierOrderID: 1,
		Status:          string(domain.ShipmentStatusVoided),
	}
	shipmentRepo.shipmentLines[3] = []*domain.ShipmentLine{
		{ID: 3, ShipmentID: 3, SupplierOrderLineID: 10, FulfillmentLineID: 100, Quantity: 5},
	}

	uc := NewShipmentReconciliationQueryUseCase(supplierRepo, shipmentRepo)
	result, err := uc.GetSupplierOrderLineShippedSummary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	line10 := findLineResult(result, 10)
	if line10 == nil {
		t.Fatalf("expected a result entry for SOL 10, got %+v", result)
	}
	if line10.ShippedQuantity != 2 {
		t.Errorf("ShippedQuantity = %d, want 2 (voided qty excluded)", line10.ShippedQuantity)
	}
	if line10.RemainingQuantity != 3 {
		t.Errorf("RemainingQuantity = %d, want 3", line10.RemainingQuantity)
	}
}
