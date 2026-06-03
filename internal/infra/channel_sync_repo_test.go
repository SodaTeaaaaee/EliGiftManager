package infra

import (
	"context"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	db.Exec("PRAGMA foreign_keys = ON;")
	return db
}

func TestChannelSyncRepositorySaveJobPreservesCreatedAt(t *testing.T) {
	db := openTestDB(t)
	if err := db.AutoMigrate(&persistence.ChannelSyncJob{}, &persistence.ChannelSyncItem{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := NewChannelSyncRepository(db)

	// Create a job
	job := &domain.ChannelSyncJob{
		WaveID:               1,
		IntegrationProfileID: 1,
		Direction:            "push_tracking",
		Status:               "pending",
	}
	if err := repo.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	originalCreatedAt := job.CreatedAt
	if originalCreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set by CreateJob")
	}

	// Update runtime fields via SaveJob
	job.Status = "running"
	now := time.Now()
	job.StartedAt = &now
	job.RequestPayload = `{"test":true}`
	if err := repo.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	// Read back
	got, err := repo.FindJobByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}

	if got.Status != "running" {
		t.Errorf("Status = %q, want running", got.Status)
	}
	if got.RequestPayload != `{"test":true}` {
		t.Errorf("RequestPayload = %q, want {\"test\":true}", got.RequestPayload)
	}
	if got.StartedAt == nil || got.StartedAt.IsZero() {
		t.Error("StartedAt should be updated")
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v (should be preserved)", got.CreatedAt, originalCreatedAt)
	}
}

func TestChannelSyncRepositorySaveItemPreservesCreatedAt(t *testing.T) {
	db := openTestDB(t)
	if err := db.AutoMigrate(&persistence.ChannelSyncJob{}, &persistence.ChannelSyncItem{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := NewChannelSyncRepository(db)

	// Create job + item
	job := &domain.ChannelSyncJob{
		WaveID:               1,
		IntegrationProfileID: 1,
		Direction:            "push_tracking",
		Status:               "pending",
	}
	if err := repo.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	item := &domain.ChannelSyncItem{
		ChannelSyncJobID:  job.ID,
		FulfillmentLineID: 1,
		ShipmentID:        1,
		Status:            "pending",
	}
	if err := repo.CreateItem(context.Background(), item); err != nil {
		t.Fatalf("create item: %v", err)
	}

	originalCreatedAt := item.CreatedAt
	if originalCreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set by CreateItem")
	}

	// Update runtime fields
	item.Status = "success"
	item.ErrorMessage = ""
	item.TrackingNo = "TRACK-UPDATED"
	if err := repo.SaveItem(context.Background(), item); err != nil {
		t.Fatalf("save item: %v", err)
	}

	// Read back via ListItemsByJob
	items, err := repo.ListItemsByJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got := items[0]

	if got.Status != "success" {
		t.Errorf("Status = %q, want success", got.Status)
	}
	if got.TrackingNo != "TRACK-UPDATED" {
		t.Errorf("TrackingNo = %q, want TRACK-UPDATED", got.TrackingNo)
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v (should be preserved)", got.CreatedAt, originalCreatedAt)
	}
}

func TestChannelSyncRepositorySaveJobDoesNotOverwriteUnrelatedColumns(t *testing.T) {
	db := openTestDB(t)
	if err := db.AutoMigrate(&persistence.ChannelSyncJob{}, &persistence.ChannelSyncItem{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := NewChannelSyncRepository(db)

	job := &domain.ChannelSyncJob{
		WaveID:               1,
		IntegrationProfileID: 1,
		Direction:            "push_tracking",
		Status:               "pending",
		BasisPayloadSnapshot: `{"original":"basis"}`,
	}
	if err := repo.CreateJob(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	originalBasis := job.BasisPayloadSnapshot

	// SaveJob via a partial domain object — only Status and ErrorMessage are set
	loaded, _ := repo.FindJobByID(context.Background(), job.ID)
	loaded.Status = "failed"
	loaded.ErrorMessage = "test error"
	// Intentionally NOT setting BasisPayloadSnapshot
	if err := repo.SaveJob(context.Background(), loaded); err != nil {
		t.Fatalf("save job: %v", err)
	}

	got, err := repo.FindJobByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if got.Status != "failed" {
		t.Errorf("Status = %q, want failed", got.Status)
	}
	// Columns not carried by the partial domain object should survive
	if got.BasisPayloadSnapshot != originalBasis {
		t.Errorf("BasisPayloadSnapshot = %q, want %q (unrelated column should be preserved)", got.BasisPayloadSnapshot, originalBasis)
	}
}
