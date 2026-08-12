package app_test

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var importRunTestSequence atomic.Uint64

func TestListImportRunsPageTraversesFirstMiddleAndLastPages(t *testing.T) {
	t.Parallel()
	uc, repo := newImportEvidencePaginationUseCase(t)
	base := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		base.Add(5 * time.Minute),
		base.Add(4 * time.Minute),
		base.Add(4 * time.Minute),
		base.Add(3 * time.Minute),
		base.Add(2 * time.Minute),
		base.Add(time.Minute),
		base,
	}
	ids := make([]uint, len(times))
	for i := range times {
		ids[i] = createImportRun(t, repo, times[i], "completed", 1, "product_catalog")
	}
	want := []uint{ids[0], ids[2], ids[1], ids[3], ids[4], ids[5], ids[6]}

	var got []uint
	cursor := ""
	pageNumber := 0
	for {
		page, err := uc.ListRunsPage(context.Background(), dto.ListImportRunsPageInput{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("ListRunsPage page %d: %v", pageNumber+1, err)
		}
		pageNumber++
		for _, item := range page.Items {
			got = append(got, item.ID)
		}
		if pageNumber < 4 {
			if !page.HasMore || page.NextCursor == "" {
				t.Fatalf("page %d should have a next page: %+v", pageNumber, page)
			}
		} else if page.HasMore || page.NextCursor != "" {
			t.Fatalf("last page should not have a cursor: %+v", page)
		}
		if !page.HasMore {
			break
		}
		cursor = page.NextCursor
	}
	if pageNumber != 4 {
		t.Fatalf("page count = %d, want 4", pageNumber)
	}
	assertUintSliceEqual(t, got, want)
}

func TestListImportRunsPageRejectsInvalidCursor(t *testing.T) {
	t.Parallel()
	uc, repo := newImportEvidencePaginationUseCase(t)
	base := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	createImportRun(t, repo, base.Add(time.Minute), "completed", 1, "product_catalog")
	createImportRun(t, repo, base, "completed", 1, "product_catalog")

	t.Run("malformed base64url", func(t *testing.T) {
		_, err := uc.ListRunsPage(context.Background(), dto.ListImportRunsPageInput{Limit: 1, Cursor: "not+a+cursor"})
		if err == nil || !strings.Contains(err.Error(), "invalid import runs cursor") {
			t.Fatalf("error = %v, want invalid cursor", err)
		}
	})

	t.Run("unknown payload field", func(t *testing.T) {
		cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"createdAt":"2026-07-15T12:00:00Z","id":1,"filter":"x","extra":true}`))
		_, err := uc.ListRunsPage(context.Background(), dto.ListImportRunsPageInput{Limit: 1, Cursor: cursor})
		if err == nil || !strings.Contains(err.Error(), "invalid import runs cursor") {
			t.Fatalf("error = %v, want invalid cursor", err)
		}
	})

	t.Run("cursor bound to filters", func(t *testing.T) {
		page, err := uc.ListRunsPage(context.Background(), dto.ListImportRunsPageInput{Limit: 1, Status: "completed", DocumentType: "product_catalog"})
		if err != nil {
			t.Fatalf("first page: %v", err)
		}
		_, err = uc.ListRunsPage(context.Background(), dto.ListImportRunsPageInput{Limit: 1, Cursor: page.NextCursor, Status: "completed", DocumentType: "supplier_shipment"})
		if err == nil || !strings.Contains(err.Error(), "invalid import runs cursor") {
			t.Fatalf("error = %v, want cursor/filter mismatch", err)
		}
	})
}

func TestListImportRunsPageFiltersStatusProfileAndDocumentType(t *testing.T) {
	t.Parallel()
	uc, repo := newImportEvidencePaginationUseCase(t)
	base := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	wantFirst := createImportRun(t, repo, base.Add(5*time.Minute), "completed", 1, "product_catalog")
	wantSecond := createImportRun(t, repo, base.Add(4*time.Minute), "completed", 1, "product_catalog")
	createImportRun(t, repo, base.Add(3*time.Minute), "running", 1, "product_catalog")
	createImportRun(t, repo, base.Add(2*time.Minute), "completed", 2, "product_catalog")
	createImportRun(t, repo, base.Add(time.Minute), "completed", 1, "supplier_shipment")

	profileID := uint(1)
	input := dto.ListImportRunsPageInput{Limit: 1, Status: "completed", ProfileID: &profileID, DocumentType: "product_catalog"}
	first, err := uc.ListRunsPage(context.Background(), input)
	if err != nil {
		t.Fatalf("first filtered page: %v", err)
	}
	if !first.HasMore || len(first.Items) != 1 || first.Items[0].ID != wantFirst {
		t.Fatalf("unexpected first filtered page: %+v", first)
	}
	input.Cursor = first.NextCursor
	last, err := uc.ListRunsPage(context.Background(), input)
	if err != nil {
		t.Fatalf("last filtered page: %v", err)
	}
	if last.HasMore || last.NextCursor != "" || len(last.Items) != 1 || last.Items[0].ID != wantSecond {
		t.Fatalf("unexpected last filtered page: %+v", last)
	}
}

func newImportEvidencePaginationUseCase(t *testing.T) (*app.ImportEvidenceUseCase, domain.ImportEvidenceRepository) {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&persistence.ImportRun{}); err != nil {
		t.Fatalf("migrate import runs: %v", err)
	}
	repo := infra.NewImportEvidenceRepository(db)
	return app.NewImportEvidenceUseCase(repo), repo
}

func createImportRun(t *testing.T, repo domain.ImportEvidenceRepository, createdAt time.Time, status string, profileID uint, documentType string) uint {
	t.Helper()
	run := &domain.ImportRun{
		RunKey:               createdAt.Format(time.RFC3339Nano) + status + documentType + "-" + strconv.FormatUint(importRunTestSequence.Add(1), 10),
		ImportKind:           documentType,
		IntegrationProfileID: &profileID,
		Status:               status,
		RetentionDays:        domain.ImportRetention90Days,
		CreatedAt:            createdAt,
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("create import run: %v", err)
	}
	return run.ID
}

func assertUintSliceEqual(t *testing.T, got, want []uint) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}
