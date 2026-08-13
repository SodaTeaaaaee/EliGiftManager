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

func setupDemandAssignmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.WaveDemandAssignment{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestDeleteByWaveAndDocumentHardDeletesSoReassignSucceeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := setupDemandAssignmentTestDB(t)
	repo := NewWaveDemandAssignmentRepository(db)

	assignment := &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: 10}
	if err := repo.Create(ctx, assignment); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.DeleteByWaveAndDocument(ctx, 1, 10); err != nil {
		t.Fatalf("unassign: %v", err)
	}

	var leftover int64
	if err := db.Unscoped().Model(&persistence.WaveDemandAssignment{}).
		Where("wave_id = ? AND demand_document_id = ?", 1, 10).Count(&leftover).Error; err != nil {
		t.Fatalf("count leftover: %v", err)
	}
	if leftover != 0 {
		t.Fatalf("hard-delete leftover rows = %d, want 0", leftover)
	}

	if err := repo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: 10}); err != nil {
		t.Fatalf("reassign after hard-delete: %v", err)
	}
}

func TestCreatePurgesSoftDeletedResidueAndMapsDuplicateError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := setupDemandAssignmentTestDB(t)
	repo := NewWaveDemandAssignmentRepository(db)

	first := &domain.WaveDemandAssignment{WaveID: 2, DemandDocumentID: 20}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Delete(&persistence.WaveDemandAssignment{}, first.ID).Error; err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	if err := repo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 2, DemandDocumentID: 20}); err != nil {
		t.Fatalf("reassign after soft-delete residue: %v", err)
	}

	dupErr := repo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 2, DemandDocumentID: 20})
	if dupErr == nil {
		t.Fatal("expected duplicate assignment error")
	}
	if !strings.Contains(dupErr.Error(), "already assigned to wave 2") {
		t.Fatalf("duplicate error = %q, want readable already-assigned message", dupErr.Error())
	}
}

func TestDeleteByWaveAndDocumentReportsMissingAssignment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := NewWaveDemandAssignmentRepository(setupDemandAssignmentTestDB(t))

	err := repo.DeleteByWaveAndDocument(ctx, 9, 99)
	if err == nil {
		t.Fatal("expected error when Delete matches 0 rows")
	}
	if !strings.Contains(err.Error(), "nothing to unassign") {
		t.Fatalf("error = %q, want nothing to unassign", err.Error())
	}
}
