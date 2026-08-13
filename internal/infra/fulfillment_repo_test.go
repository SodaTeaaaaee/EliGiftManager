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

func openFulfillmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.FulfillmentLine{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func createTestFulfillmentLine(t *testing.T, repo domain.FulfillmentLineRepository, supplierState string) *domain.FulfillmentLine {
	t.Helper()
	line := &domain.FulfillmentLine{
		WaveID:        1,
		Quantity:      1,
		LineReason:    string(domain.LineReasonEntitlement),
		SupplierState: supplierState,
	}
	if err := repo.Create(context.Background(), line); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}
	if line.ID == 0 {
		t.Fatal("created fulfillment line should have an ID")
	}
	return line
}

func TestBulkUpdateStatesEmptySlice(t *testing.T) {
	db := openFulfillmentTestDB(t)
	repo := NewFulfillmentRepository(db)

	if err := repo.BulkUpdateStates(context.Background(), nil); err != nil {
		t.Fatalf("empty nil slice: %v", err)
	}
	if err := repo.BulkUpdateStates(context.Background(), []domain.FulfillmentLineStateUpdate{}); err != nil {
		t.Fatalf("empty slice: %v", err)
	}
}

func TestBulkUpdateStatesUpdatesExistingLine(t *testing.T) {
	db := openFulfillmentTestDB(t)
	repo := NewFulfillmentRepository(db)
	line := createTestFulfillmentLine(t, repo, "not_submitted")

	err := repo.BulkUpdateStates(context.Background(), []domain.FulfillmentLineStateUpdate{
		{ID: line.ID, SupplierState: "submitted"},
	})
	if err != nil {
		t.Fatalf("BulkUpdateStates: %v", err)
	}

	got, err := repo.FindByID(context.Background(), line.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.SupplierState != "submitted" {
		t.Errorf("SupplierState = %q, want submitted", got.SupplierState)
	}
}

func TestBulkUpdateStatesMissingID(t *testing.T) {
	db := openFulfillmentTestDB(t)
	repo := NewFulfillmentRepository(db)

	err := repo.BulkUpdateStates(context.Background(), []domain.FulfillmentLineStateUpdate{
		{ID: 999, SupplierState: "submitted"},
	})
	if err == nil {
		t.Fatal("expected error for missing ID, got nil")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("error %q should mention missing ID 999", err)
	}
}

func TestBulkUpdateStatesMixedIDsRollsBack(t *testing.T) {
	db := openFulfillmentTestDB(t)
	repo := NewFulfillmentRepository(db)
	line := createTestFulfillmentLine(t, repo, "not_submitted")

	err := repo.BulkUpdateStates(context.Background(), []domain.FulfillmentLineStateUpdate{
		{ID: line.ID, SupplierState: "submitted"},
		{ID: 999, SupplierState: "submitted"},
	})
	if err == nil {
		t.Fatal("expected error for mixed IDs, got nil")
	}

	got, findErr := repo.FindByID(context.Background(), line.ID)
	if findErr != nil {
		t.Fatalf("FindByID: %v", findErr)
	}
	if got.SupplierState != "not_submitted" {
		t.Errorf("SupplierState = %q, want not_submitted (transaction should roll back)", got.SupplierState)
	}
}
