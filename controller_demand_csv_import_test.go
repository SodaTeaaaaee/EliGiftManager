package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDemandCSVImportTestDB spins up an in-memory sqlite DB migrated with the tables the
// dual-mode demand CSV import path touches.
func setupDemandCSVImportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.IntegrationProfile{},
		&persistence.DocumentTemplate{},
		&persistence.IntegrationProfileTemplateBinding{},
		&persistence.CustomerProfile{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return gdb
}

// newDemandCSVImportTestController builds a DemandController wired to the given in-memory DB,
// bypassing NewDemandController's db.GetDB() singleton lookup.
func newDemandCSVImportTestController(gdb *gorm.DB) *DemandController {
	demandRepo := infra.NewDemandRepository(gdb)
	profileRepo := infra.NewProfileRepository(gdb)
	integrationProfileRepo := infra.NewIntegrationProfileRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	return &DemandController{
		intakeUC:           app.NewDemandIntakeUseCase(demandRepo),
		demandRepo:         demandRepo,
		profileRepo:        profileRepo,
		integrationProfile: integrationProfileRepo,
		identityResolution: app.NewIdentityResolutionService(profileRepo),
		templateMapping:    app.NewTemplateMappingService(templateRepo, bindingRepo, integrationProfileRepo),
	}
}

// seedDemandCSVImportFixture creates an integration profile bound (as default) to a document
// template that maps "Name" -> external_title and "Qty" -> requested_quantity, defaulting
// line_type to "sku_order".
func seedDemandCSVImportFixture(t *testing.T, gdb *gorm.DB) (profileID uint) {
	t.Helper()
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{ProfileKey: "csv-import-profile", SourceChannel: "csv", DemandKind: "membership_entitlement"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "csv-import-template",
		DocumentType: "import_entitlement",
		Format:       "csv",
		MappingRules: `{"columns":{"external_title":"Name","requested_quantity":"Qty"},"defaults":{"line_type":"sku_order"}}`,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}

	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	binding := &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	}
	if err := bindingRepo.Create(ctx, binding); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	return profile.ID
}

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "import.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

func TestParseCSVFile_HeadersRowsAndBOM(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	// Leading BOM on the header line; second data row is ragged (missing the Qty column).
	content := "\ufeffName,Qty\nStandee,2\nPoster\nSticker,5,extra\n"
	path := writeTempCSV(t, content)

	preview, err := c.ParseCSVFile(path)
	if err != nil {
		t.Fatalf("ParseCSVFile: %v", err)
	}
	if len(preview.Headers) != 2 || preview.Headers[0] != "Name" || preview.Headers[1] != "Qty" {
		t.Fatalf("expected BOM-stripped headers [Name Qty], got %+v", preview.Headers)
	}
	if len(preview.Rows) != 3 {
		t.Fatalf("expected 3 data rows, got %d", len(preview.Rows))
	}
	if preview.Rows[0]["Name"] != "Standee" || preview.Rows[0]["Qty"] != "2" {
		t.Errorf("row 0 mismatch: %+v", preview.Rows[0])
	}
	// Ragged short row: only the overlapping "Name" column should be populated.
	if preview.Rows[1]["Name"] != "Poster" {
		t.Errorf("row 1 mismatch: %+v", preview.Rows[1])
	}
	if _, ok := preview.Rows[1]["Qty"]; ok {
		t.Errorf("row 1 should have no Qty value for a short ragged row, got %+v", preview.Rows[1])
	}
	// Ragged long row: extra trailing field beyond the header count is dropped, not panicking.
	if preview.Rows[2]["Name"] != "Sticker" || preview.Rows[2]["Qty"] != "5" {
		t.Errorf("row 2 mismatch: %+v", preview.Rows[2])
	}
}

func TestParseCSVFile_EmptyFileErrors(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	path := writeTempCSV(t, "")
	if _, err := c.ParseCSVFile(path); err == nil {
		t.Fatal("expected an error for an empty csv file")
	}
}

func TestImportDemandCSV_SkipInvalid_PersistsGoodLinesAndSurfacesErrors(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	input := dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "skip_invalid",
		Rows: []map[string]string{
			{"Name": "Standee", "Qty": "2"},
			{"Name": "Poster", "Qty": "not-a-number"},
			{"Name": "Sticker", "Qty": "5"},
		},
	}

	result, err := c.ImportDemandCSV(input)
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.Document == nil {
		t.Fatal("expected a persisted document")
	}
	if result.TotalProcessed != 3 || result.SuccessCount != 2 || result.ErrorCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].RowIndex != 1 {
		t.Fatalf("expected exactly 1 error at row index 1, got %+v", result.Errors)
	}

	lines, err := c.demandRepo.ListLinesByDocument(appContext, result.Document.ID)
	if err != nil {
		t.Fatalf("ListLinesByDocument: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 persisted lines, got %d", len(lines))
	}
}

func TestImportDemandCSV_RejectAll_PersistsNothingOnBadRow(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	input := dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Name": "Standee", "Qty": "2"},
			{"Name": "Poster", "Qty": "not-a-number"},
			{"Name": "Sticker", "Qty": "5"},
		},
	}

	result, err := c.ImportDemandCSV(input)
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.Document != nil {
		t.Fatalf("expected no persisted document on reject_all failure, got %+v", result.Document)
	}
	if result.SuccessCount != 0 || result.ErrorCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}

	docs, err := c.demandRepo.List(appContext)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected zero persisted documents, got %d", len(docs))
	}
}

func TestImportDemandCSV_InvalidImportMode(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	_, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "bogus_mode",
		Rows:                 []map[string]string{{"Name": "Standee", "Qty": "2"}},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid importMode")
	}
}
