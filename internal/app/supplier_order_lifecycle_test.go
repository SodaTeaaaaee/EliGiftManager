package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newSupplierOrderLifecycleTestDB is a unit-local in-memory sqlite helper,
// deliberately named distinctly from setupTestDB (customer_profile_usecase_test.go)
// to avoid symbol collisions with sibling test files added in the same
// package by other units.
func newSupplierOrderLifecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	if err := db.AutoMigrate(
		&persistence.SupplierOrder{},
		&persistence.SupplierOrderLine{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	return db
}

func mustCreateSupplierOrder(t *testing.T, ctx context.Context, repo domain.SupplierOrderRepository, status string) *domain.SupplierOrder {
	t.Helper()
	order := &domain.SupplierOrder{
		WaveID:           1,
		SupplierPlatform: "test.factory",
		BatchNo:          "WAVE-1-BATCH-1",
		SubmissionMode:   "csv",
		Status:           status,
	}
	if err := repo.Create(ctx, order); err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	return order
}

func mustCreateSupplierOrderLine(t *testing.T, ctx context.Context, repo domain.SupplierOrderRepository, orderID uint, lineNo int, submittedQty int) *domain.SupplierOrderLine {
	t.Helper()
	line := &domain.SupplierOrderLine{
		SupplierOrderID:   orderID,
		FulfillmentLineID: uint(1000 + lineNo),
		SupplierLineNo:    lineNo,
		SupplierSKU:       fmt.Sprintf("SKU-%d", lineNo),
		SubmittedQuantity: submittedQty,
		Status:            "draft",
	}
	if err := repo.CreateLine(ctx, line); err != nil {
		t.Fatalf("create supplier order line: %v", err)
	}
	return line
}

func TestMarkSupplierOrderSubmitted_TransitionsDraftToSubmitted(t *testing.T) {
	ctx := context.Background()
	db := newSupplierOrderLifecycleTestDB(t)
	supplierRepo := infra.NewSupplierOrderRepository(db)
	lineWriter := infra.NewSupplierOrderLineWriter(db)
	uc := NewSupplierOrderLifecycleUseCase(supplierRepo, lineWriter)

	order := mustCreateSupplierOrder(t, ctx, supplierRepo, string(domain.SupplierOrderStatusDraft))

	updated, err := uc.MarkSupplierOrderSubmitted(ctx, dto.MarkSupplierOrderSubmittedInput{
		OrderID:         order.ID,
		ExternalOrderNo: "EXT-0001",
	})
	if err != nil {
		t.Fatalf("MarkSupplierOrderSubmitted: %v", err)
	}
	if updated.Status != string(domain.SupplierOrderStatusSubmitted) {
		t.Errorf("expected status %q, got %q", domain.SupplierOrderStatusSubmitted, updated.Status)
	}
	if updated.ExternalOrderNo != "EXT-0001" {
		t.Errorf("expected externalOrderNo %q, got %q", "EXT-0001", updated.ExternalOrderNo)
	}
	if updated.SubmittedAt == nil {
		t.Errorf("expected submittedAt to be set")
	}

	// Verify persisted, not just returned.
	reloaded, err := supplierRepo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if reloaded.Status != string(domain.SupplierOrderStatusSubmitted) || reloaded.ExternalOrderNo != "EXT-0001" {
		t.Errorf("persisted order not updated: %+v", reloaded)
	}

	// Re-submitting a non-draft order must fail.
	if _, err := uc.MarkSupplierOrderSubmitted(ctx, dto.MarkSupplierOrderSubmittedInput{
		OrderID:         order.ID,
		ExternalOrderNo: "EXT-0002",
	}); err == nil {
		t.Errorf("expected error re-submitting a non-draft order")
	}
}

func TestRecordSupplierOrderAcceptance_SetsPerLineAcceptedQuantityAndStatus(t *testing.T) {
	ctx := context.Background()
	db := newSupplierOrderLifecycleTestDB(t)
	supplierRepo := infra.NewSupplierOrderRepository(db)
	lineWriter := infra.NewSupplierOrderLineWriter(db)
	uc := NewSupplierOrderLifecycleUseCase(supplierRepo, lineWriter)

	order := mustCreateSupplierOrder(t, ctx, supplierRepo, string(domain.SupplierOrderStatusSubmitted))
	line1 := mustCreateSupplierOrderLine(t, ctx, supplierRepo, order.ID, 1, 10)
	line2 := mustCreateSupplierOrderLine(t, ctx, supplierRepo, order.ID, 2, 5)

	updated, err := uc.RecordSupplierOrderAcceptance(ctx, dto.RecordSupplierOrderAcceptanceInput{
		OrderID: order.ID,
		Lines: []dto.SupplierOrderLineAcceptanceEntry{
			{LineID: line1.ID, AcceptedQuantity: 9},
			{LineID: line2.ID, AcceptedQuantity: 5},
		},
	})
	if err != nil {
		t.Fatalf("RecordSupplierOrderAcceptance: %v", err)
	}
	if updated.Status != string(domain.SupplierOrderStatusAccepted) {
		t.Errorf("expected order status %q, got %q", domain.SupplierOrderStatusAccepted, updated.Status)
	}

	reloadedLine1, err := supplierRepo.FindLineByID(ctx, line1.ID)
	if err != nil {
		t.Fatalf("FindLineByID line1: %v", err)
	}
	if reloadedLine1.AcceptedQuantity != 9 || reloadedLine1.Status != string(domain.SupplierOrderStatusAccepted) {
		t.Errorf("line1 not updated correctly: %+v", reloadedLine1)
	}

	reloadedLine2, err := supplierRepo.FindLineByID(ctx, line2.ID)
	if err != nil {
		t.Fatalf("FindLineByID line2: %v", err)
	}
	if reloadedLine2.AcceptedQuantity != 5 || reloadedLine2.Status != string(domain.SupplierOrderStatusAccepted) {
		t.Errorf("line2 not updated correctly: %+v", reloadedLine2)
	}

	// A line belonging to a different order must be rejected.
	otherOrder := mustCreateSupplierOrder(t, ctx, supplierRepo, string(domain.SupplierOrderStatusSubmitted))
	if _, err := uc.RecordSupplierOrderAcceptance(ctx, dto.RecordSupplierOrderAcceptanceInput{
		OrderID: otherOrder.ID,
		Lines: []dto.SupplierOrderLineAcceptanceEntry{
			{LineID: line1.ID, AcceptedQuantity: 1},
		},
	}); err == nil {
		t.Errorf("expected error accepting a line that belongs to a different order")
	}
}

func TestGenerateSupplierOrderFile_EmbedsLineIDsForReconciliation(t *testing.T) {
	ctx := context.Background()
	db := newSupplierOrderLifecycleTestDB(t)
	supplierRepo := infra.NewSupplierOrderRepository(db)

	order := mustCreateSupplierOrder(t, ctx, supplierRepo, string(domain.SupplierOrderStatusDraft))
	line1 := mustCreateSupplierOrderLine(t, ctx, supplierRepo, order.ID, 1, 10)
	line2 := mustCreateSupplierOrderLine(t, ctx, supplierRepo, order.ID, 2, 5)

	outputDir := t.TempDir()
	fileWriter := NewSupplierOrderFileWriter(supplierRepo, outputDir)

	result, err := fileWriter.GenerateSupplierOrderFile(ctx, order.ID)
	if err != nil {
		t.Fatalf("GenerateSupplierOrderFile: %v", err)
	}
	if result.LineCount != 2 {
		t.Errorf("expected lineCount 2, got %d", result.LineCount)
	}
	if filepath.Dir(result.FilePath) != outputDir {
		t.Errorf("expected file under %q, got %q", outputDir, result.FilePath)
	}

	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	var payload struct {
		SupplierOrderID uint   `json:"supplierOrderId"`
		BatchNo         string `json:"batchNo"`
		Lines           []struct {
			LineID  uint   `json:"lineId"`
			BatchNo string `json:"batchNo"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal generated file: %v", err)
	}

	if payload.SupplierOrderID != order.ID {
		t.Errorf("expected supplierOrderId %d, got %d", order.ID, payload.SupplierOrderID)
	}
	if len(payload.Lines) != 2 {
		t.Fatalf("expected 2 lines embedded, got %d", len(payload.Lines))
	}

	seen := map[uint]bool{}
	for _, l := range payload.Lines {
		seen[l.LineID] = true
		if l.BatchNo != order.BatchNo {
			t.Errorf("expected embedded batchNo %q, got %q", order.BatchNo, l.BatchNo)
		}
	}
	if !seen[line1.ID] || !seen[line2.ID] {
		t.Errorf("expected embedded line IDs to include %d and %d, got %v", line1.ID, line2.ID, payload.Lines)
	}
}
