package infra

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDemandRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.DemandDocument{}, &persistence.WaveDemandAssignment{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedDemandDoc(t *testing.T, repo domain.DemandDocumentRepository, sourceNo string, profileID *uint) *domain.DemandDocument {
	t.Helper()
	doc := &domain.DemandDocument{
		Kind:              string(domain.DemandKindRetailOrder),
		CaptureMode:       string(domain.CaptureModeManualEntry),
		SourceChannel:     "test",
		SourceDocumentNo:  sourceNo,
		CustomerProfileID: profileID,
	}
	if err := repo.Create(context.Background(), doc); err != nil {
		t.Fatalf("create demand document %s: %v", sourceNo, err)
	}
	return doc
}

func TestListUnassignedIgnoresSoftDeletedAssignments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := setupDemandRepoTestDB(t)
	demandRepo := NewDemandRepository(db)
	assignRepo := NewWaveDemandAssignmentRepository(db)

	live := seedDemandDoc(t, demandRepo, "LIVE", nil)
	soft := seedDemandDoc(t, demandRepo, "SOFT", nil)
	free := seedDemandDoc(t, demandRepo, "FREE", nil)

	if err := assignRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: live.ID}); err != nil {
		t.Fatalf("assign live: %v", err)
	}
	softAssign := &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: soft.ID}
	if err := assignRepo.Create(ctx, softAssign); err != nil {
		t.Fatalf("assign soft: %v", err)
	}
	if err := db.Delete(&persistence.WaveDemandAssignment{}, softAssign.ID).Error; err != nil {
		t.Fatalf("soft-delete assignment: %v", err)
	}

	unassigned, err := demandRepo.ListUnassigned(ctx)
	if err != nil {
		t.Fatalf("ListUnassigned: %v", err)
	}
	got := map[uint]bool{}
	for _, doc := range unassigned {
		got[doc.ID] = true
	}
	if got[live.ID] {
		t.Fatalf("live assignment %d should not appear in ListUnassigned", live.ID)
	}
	if !got[soft.ID] {
		t.Fatalf("soft-deleted assignment %d should appear in ListUnassigned", soft.ID)
	}
	if !got[free.ID] {
		t.Fatalf("never-assigned %d should appear in ListUnassigned", free.ID)
	}
}

func TestListUnassignedByCustomerProfileIDIgnoresSoftDeletedAssignments(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := setupDemandRepoTestDB(t)
	demandRepo := NewDemandRepository(db)
	assignRepo := NewWaveDemandAssignmentRepository(db)

	profileA := uint(7)
	profileB := uint(8)
	assigned := seedDemandDoc(t, demandRepo, "A-LIVE", &profileA)
	soft := seedDemandDoc(t, demandRepo, "A-SOFT", &profileA)
	other := seedDemandDoc(t, demandRepo, "B-FREE", &profileB)

	if err := assignRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 3, DemandDocumentID: assigned.ID}); err != nil {
		t.Fatalf("assign live: %v", err)
	}
	softAssign := &domain.WaveDemandAssignment{WaveID: 3, DemandDocumentID: soft.ID}
	if err := assignRepo.Create(ctx, softAssign); err != nil {
		t.Fatalf("assign soft: %v", err)
	}
	if err := db.Delete(&persistence.WaveDemandAssignment{}, softAssign.ID).Error; err != nil {
		t.Fatalf("soft-delete assignment: %v", err)
	}

	mergeRepo := NewCustomerMergeDemandRepository(db)

	unassigned, err := mergeRepo.ListUnassignedByCustomerProfileID(ctx, profileA)
	if err != nil {
		t.Fatalf("ListUnassignedByCustomerProfileID: %v", err)
	}
	if len(unassigned) != 1 || unassigned[0].ID != soft.ID {
		t.Fatalf("unassigned for profile A = %+v, want only soft-deleted doc %d", unassigned, soft.ID)
	}
	for _, doc := range unassigned {
		if doc.ID == other.ID {
			t.Fatalf("profile B document %d leaked into profile A unassigned list", other.ID)
		}
	}
}
