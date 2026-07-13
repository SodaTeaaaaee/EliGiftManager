package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDualModeTestDB spins up an in-memory sqlite DB migrated with the tables the
// template mapping dual-mode pipeline touches.
func setupDualModeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.IntegrationProfile{},
		&persistence.DocumentTemplate{},
		&persistence.IntegrationProfileTemplateBinding{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return gdb
}

// seedDualModeFixture creates an integration profile, a document template with a known
// mapping (external_title <- "Name", requested_quantity <- "Qty", line_type default
// "sku_order"), and a default binding between them for docType "import_entitlement".
func seedDualModeFixture(t *testing.T, gdb *gorm.DB) (profileID uint, svc *TemplateMappingService) {
	t.Helper()
	ctx := context.Background()

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{ProfileKey: "test-profile", SourceChannel: "test"}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "test-template",
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

	svc = NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	return profile.ID, svc
}

func TestBuildImportPipelineWithMode_SkipInvalid_MixedRows(t *testing.T) {
	gdb := setupDualModeTestDB(t)
	profileID, svc := seedDualModeFixture(t, gdb)
	ctx := context.Background()

	rows := []map[string]string{
		{"Name": "Standee", "Qty": "2"},  // good
		{"Name": "Poster", "Qty": "bad"}, // bad — invalid quantity
		{"Name": "Sticker", "Qty": "5"},  // good
	}

	_, lines, rowErrs, err := svc.BuildImportPipelineWithMode(ctx, profileID, "import_entitlement", rows, "skip_invalid")
	if err != nil {
		t.Fatalf("BuildImportPipelineWithMode: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 successfully-mapped lines, got %d", len(lines))
	}
	if len(rowErrs) != 1 {
		t.Fatalf("expected 1 row error, got %d: %+v", len(rowErrs), rowErrs)
	}
	if rowErrs[0].RowIndex != 1 {
		t.Errorf("expected failing row index 1, got %d", rowErrs[0].RowIndex)
	}
	if lines[0].ExternalTitle != "Standee" || lines[1].ExternalTitle != "Sticker" {
		t.Errorf("unexpected surviving lines: %+v", lines)
	}
}

func TestBuildImportPipelineWithMode_RejectAll_AbortsOnFirstBadRow(t *testing.T) {
	gdb := setupDualModeTestDB(t)
	profileID, svc := seedDualModeFixture(t, gdb)
	ctx := context.Background()

	rows := []map[string]string{
		{"Name": "Standee", "Qty": "2"},  // good
		{"Name": "Poster", "Qty": "bad"}, // bad — should abort here
		{"Name": "Sticker", "Qty": "5"},  // never reached
	}

	_, lines, rowErrs, err := svc.BuildImportPipelineWithMode(ctx, profileID, "import_entitlement", rows, "reject_all")
	if err != nil {
		t.Fatalf("BuildImportPipelineWithMode: %v", err)
	}
	if len(rowErrs) != 1 {
		t.Fatalf("expected exactly 1 row error (abort on first failure), got %d: %+v", len(rowErrs), rowErrs)
	}
	if rowErrs[0].RowIndex != 1 {
		t.Errorf("expected failing row index 1, got %d", rowErrs[0].RowIndex)
	}
	// The loop must not have processed row 2 ("Sticker") after aborting.
	for _, l := range lines {
		if l.ExternalTitle == "Sticker" {
			t.Errorf("reject_all should not have mapped rows after the first failure, found: %+v", lines)
		}
	}
}

func TestBuildImportPipelineWithMode_AllGoodRows_BothModesSucceedIdentically(t *testing.T) {
	ctx := context.Background()
	rows := []map[string]string{
		{"Name": "Standee", "Qty": "2"},
		{"Name": "Sticker", "Qty": "5"},
	}

	for _, mode := range []string{"reject_all", "skip_invalid"} {
		gdb := setupDualModeTestDB(t)
		profileID, svc := seedDualModeFixture(t, gdb)

		_, lines, rowErrs, err := svc.BuildImportPipelineWithMode(ctx, profileID, "import_entitlement", rows, mode)
		if err != nil {
			t.Fatalf("[mode=%s] BuildImportPipelineWithMode: %v", mode, err)
		}
		if len(rowErrs) != 0 {
			t.Fatalf("[mode=%s] expected no row errors, got %+v", mode, rowErrs)
		}
		if len(lines) != 2 {
			t.Fatalf("[mode=%s] expected 2 mapped lines, got %d", mode, len(lines))
		}
	}
}

func TestBuildImportPipelineWithMode_TemplateResolutionFailureIsHardError(t *testing.T) {
	gdb := setupDualModeTestDB(t)
	profileID, svc := seedDualModeFixture(t, gdb)
	ctx := context.Background()

	_, _, _, err := svc.BuildImportPipelineWithMode(ctx, profileID, "unbound_document_type", nil, "skip_invalid")
	if err == nil {
		t.Fatal("expected a hard error when no template binding resolves")
	}
}
