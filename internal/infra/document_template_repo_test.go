package infra

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDocumentTemplateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.DocumentTemplate{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newTestDocumentTemplate(key string) *domain.DocumentTemplate {
	return &domain.DocumentTemplate{
		TemplateKey:  key,
		DocumentType: "invoice",
		Format:       "html",
	}
}

func TestDocumentTemplateRepositoryDeleteAllowsRecreateSameKey(t *testing.T) {
	t.Parallel()
	db := openDocumentTemplateTestDB(t)
	repo := NewDocumentTemplateRepository(db)
	ctx := context.Background()

	original := newTestDocumentTemplate("tmpl-a")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("create original: %v", err)
	}
	if err := repo.Delete(ctx, original.ID); err != nil {
		t.Fatalf("delete original: %v", err)
	}

	recreated := newTestDocumentTemplate("tmpl-a")
	if err := repo.Create(ctx, recreated); err != nil {
		t.Fatalf("recreate same TemplateKey: %v", err)
	}
	if recreated.ID == 0 {
		t.Fatal("recreated template should have an ID")
	}
	if recreated.ID == original.ID {
		t.Fatal("recreated template should be a new row")
	}
}

func TestDocumentTemplateRepositoryFindByKeyAfterDeleteReturnsNil(t *testing.T) {
	t.Parallel()
	db := openDocumentTemplateTestDB(t)
	repo := NewDocumentTemplateRepository(db)
	ctx := context.Background()

	tmpl := newTestDocumentTemplate("tmpl-a")
	if err := repo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete(ctx, tmpl.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := repo.FindByKey(ctx, "tmpl-a")
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if got != nil {
		t.Fatalf("FindByKey after delete = %+v, want nil", got)
	}
}

func TestDocumentTemplateRepositoryDeleteMissingIDReturnsError(t *testing.T) {
	t.Parallel()
	db := openDocumentTemplateTestDB(t)
	repo := NewDocumentTemplateRepository(db)

	if err := repo.Delete(context.Background(), 999); err == nil {
		t.Fatal("Delete of missing ID should return an error")
	}
}
