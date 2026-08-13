package controller

import (
	"os"
	"path/filepath"
	"strings"
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
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
		&persistence.CustomerNameObservation{},
		&persistence.CustomerNameEvent{},
		&persistence.CustomerProfileOrigin{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	seedEnabledFeaturePolicy(t, gdb)
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
		gdb:                gdb,
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
	profile := &domain.IntegrationProfile{
		ProfileKey:    "csv-import-profile",
		SourceChannel: "csv",
		SourceSurface: string(domain.SourceSurfaceMembership),
		DemandKind:    string(domain.DemandKindMembershipEntitlement),
	}
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

func TestParseTabularFile_DelegatesFromParseCSVFile(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	path := writeTempCSV(t, "Name,Qty\nStandee,2\n")
	viaTabular, err := c.ParseTabularFile(path, true)
	if err != nil {
		t.Fatalf("ParseTabularFile: %v", err)
	}
	viaCSV, err := c.ParseCSVFile(path)
	if err != nil {
		t.Fatalf("ParseCSVFile: %v", err)
	}
	if len(viaTabular.Headers) != len(viaCSV.Headers) || len(viaTabular.Rows) != len(viaCSV.Rows) {
		t.Fatalf("ParseCSVFile should match ParseTabularFile: tabular=%+v csv=%+v", viaTabular, viaCSV)
	}
	if viaTabular.Headers[0] != "Name" || viaTabular.Rows[0]["Name"] != "Standee" {
		t.Errorf("unexpected preview: %+v", viaTabular)
	}
}

func TestParseTabularFile_HasHeaderFalse_KeepsRow0AsData(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	// UTF-8 BOM + 3 headerless membership-like rows.
	content := "\ufeff总督,uid-10001,DisplayA\n提督,uid-10002,DisplayB\n舰长,uid-10003,DisplayC\n"
	path := writeTempCSV(t, content)

	preview, err := c.ParseTabularFile(path, false)
	if err != nil {
		t.Fatalf("ParseTabularFile(hasHeader=false): %v", err)
	}
	if len(preview.Rows) != 3 {
		t.Fatalf("expected 3 data rows when hasHeader=false, got %d headers=%+v rows=%+v",
			len(preview.Rows), preview.Headers, preview.Rows)
	}
	// Synthetic col_N headers so maps still carry values.
	if len(preview.Headers) != 3 || preview.Headers[0] != "col_0" {
		t.Fatalf("expected synthetic col_N headers, got %+v", preview.Headers)
	}
	if preview.Rows[0]["col_0"] != "总督" || preview.Rows[0]["col_1"] != "uid-10001" {
		t.Fatalf("row0 should remain data (BOM stripped), got %+v", preview.Rows[0])
	}

	// hasHeader=true peels first membership row into headers (contrast).
	asHeader, err := c.ParseTabularFile(path, true)
	if err != nil {
		t.Fatalf("ParseTabularFile(hasHeader=true): %v", err)
	}
	if len(asHeader.Headers) != 3 || asHeader.Headers[0] != "总督" {
		t.Fatalf("hasHeader=true should peel first row into headers, got %+v", asHeader.Headers)
	}
	if len(asHeader.Rows) != 2 {
		t.Fatalf("hasHeader=true should leave 2 data rows, got %d", len(asHeader.Rows))
	}
}

func TestParseTabularFile_UnsupportedExtension(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	dir := t.TempDir()
	path := filepath.Join(dir, "not-a-sheet.txt")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if _, err := c.ParseTabularFile(path, true); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

// TestImportDemandCSV_MultiUID_PerRowIdentity ensures a headerless multi-UID membership
// sheet creates one CustomerProfile per distinct source_customer_ref (not first-wins fold).
func TestImportDemandCSV_MultiUID_PerRowIdentity(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:       "bili-multi-uid",
		SourceChannel:    "bilibili",
		SourceSurface:    "membership",
		DemandKind:       "membership_entitlement",
		IdentityStrategy: "platform_uid",
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// Positional MappingRules matching bilibili membership preset (hasHeader=false).
	mappingRules := `{
		"version": 2,
		"mode": "positional",
		"hasHeader": false,
		"positions": {
			"line.gift_level_snapshot": 0,
			"document.source_customer_ref": 1,
			"document.display_name": 2
		},
		"defaults": {
			"line.line_type": "entitlement_rule",
			"line.requested_quantity": "1"
		}
	}`
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "bili-multi-uid-tmpl",
		DocumentType: "import_entitlement",
		Format:       "csv",
		MappingRules: mappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_entitlement",
		TemplateID:           tmpl.ID,
		IsDefault:            true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	// UTF-8 BOM, no header, 3 distinct UIDs — drives the real FilePath import path.
	content := "\ufeff总督,uid-10001,DisplayA\n提督,uid-10002,DisplayB\n舰长,uid-10003,DisplayC\n"
	path := writeTempCSV(t, content)

	importInput := dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		FilePath:             path,
	}
	result, err := c.ImportDemandCSV(importInput)
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.SuccessCount != 3 || result.ErrorCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.Document == nil {
		t.Fatal("expected at least one persisted document")
	}
	if len(result.Documents) != 3 {
		t.Fatalf("expected Documents to list all 3 created documents, got %d", len(result.Documents))
	}
	if result.Documents[0].ID != result.Document.ID {
		t.Fatalf("Document should be the first of Documents: document=%d documents[0]=%d", result.Document.ID, result.Documents[0].ID)
	}

	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		t.Fatalf("List docs: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 documents (one per UID), got %d", len(docs))
	}

	refs := map[string]uint{}
	for _, doc := range docs {
		if doc.Kind != string(domain.DemandKindMembershipEntitlement) {
			t.Errorf("document %d Kind=%q, want membership_entitlement", doc.ID, doc.Kind)
		}
		if doc.SourceSurface != string(domain.SourceSurfaceMembership) {
			t.Errorf("document %d SourceSurface=%q, want membership", doc.ID, doc.SourceSurface)
		}
		if doc.SourceCustomerRef == "" {
			t.Fatalf("document %d missing SourceCustomerRef", doc.ID)
		}
		if doc.CustomerProfileID == nil {
			t.Fatalf("document %d (%s) missing CustomerProfileID", doc.ID, doc.SourceCustomerRef)
		}
		if prev, dup := refs[doc.SourceCustomerRef]; dup {
			t.Fatalf("duplicate SourceCustomerRef %q on docs %d and %d", doc.SourceCustomerRef, prev, doc.ID)
		}
		refs[doc.SourceCustomerRef] = *doc.CustomerProfileID
	}
	for _, want := range []string{"uid-10001", "uid-10002", "uid-10003"} {
		if _, ok := refs[want]; !ok {
			t.Errorf("missing document for %s; got refs=%v", want, refs)
		}
	}
	// Each UID must bind a distinct CustomerProfile (no first-wins collapse).
	seenPID := map[uint]string{}
	for ref, pid := range refs {
		if other, ok := seenPID[pid]; ok {
			t.Errorf("profile %d shared by %q and %q — identity folded", pid, other, ref)
		}
		seenPID[pid] = ref
	}
	if len(seenPID) != 3 {
		t.Fatalf("expected 3 distinct customer profiles, got %d", len(seenPID))
	}

	// Display names applied per-row identity, not first-wins batch fold.
	wantDisplay := map[string]string{
		"uid-10001": "DisplayA",
		"uid-10002": "DisplayB",
		"uid-10003": "DisplayC",
	}
	for ref, pid := range refs {
		cp, err := c.profileRepo.FindByID(ctx, pid)
		if err != nil || cp == nil {
			t.Fatalf("FindByID profile %d for %s: %v", pid, ref, err)
		}
		if cp.DisplayName != wantDisplay[ref] {
			t.Errorf("profile for %s DisplayName=%q, want %q", ref, cp.DisplayName, wantDisplay[ref])
		}
	}
}

func TestImportDemandCSV_IgnoresProfileIdentityStrategyForEntitlement(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:    "leftover-order-scoped-membership",
		SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceMembership),
		DemandKind:    string(domain.DemandKindMembershipEntitlement),
		// Leftover retail strategy must not override import_entitlement semantics.
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "leftover-entitlement-template",
		DocumentType: "import_entitlement",
		Format:       "csv",
		MappingRules: `{
			"version":2,
			"mode":"header",
			"hasHeader":true,
			"columns":{
				"document.source_customer_ref":"UID",
				"document.display_name":"Name",
				"line.external_title":"Item"
			},
			"defaults":{"line.line_type":"entitlement_rule","line.requested_quantity":"1"},
			"required":["document.source_customer_ref"]
		}`,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := infra.NewProfileTemplateBindingRepository(gdb).Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_entitlement", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"UID": "uid-alpha", "Name": "Alpha", "Item": "Gift A"},
			{"UID": "uid-beta", "Name": "Beta", "Item": "Gift B"},
		},
	})
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.SuccessCount != 2 || result.ErrorCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}

	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		t.Fatalf("List docs: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents grouped by UID, got %d (must not collapse empty order numbers)", len(docs))
	}

	refs := map[string]uint{}
	for _, doc := range docs {
		if doc.Kind != string(domain.DemandKindMembershipEntitlement) {
			t.Errorf("document %d Kind=%q, want membership_entitlement", doc.ID, doc.Kind)
		}
		if doc.SourceSurface != string(domain.SourceSurfaceMembership) {
			t.Errorf("document %d SourceSurface=%q, want membership", doc.ID, doc.SourceSurface)
		}
		if doc.SourceCustomerRef == "" {
			t.Fatalf("document %d missing SourceCustomerRef", doc.ID)
		}
		if doc.CustomerProfileID == nil {
			t.Fatalf("document %d (%s) missing CustomerProfileID", doc.ID, doc.SourceCustomerRef)
		}
		if prev, dup := refs[doc.SourceCustomerRef]; dup {
			t.Fatalf("duplicate SourceCustomerRef %q on docs %d and %d", doc.SourceCustomerRef, prev, doc.ID)
		}
		refs[doc.SourceCustomerRef] = *doc.CustomerProfileID
	}
	if _, ok := refs["uid-alpha"]; !ok {
		t.Errorf("missing document for uid-alpha; got refs=%v", refs)
	}
	if _, ok := refs["uid-beta"]; !ok {
		t.Errorf("missing document for uid-beta; got refs=%v", refs)
	}
	if refs["uid-alpha"] != 0 && refs["uid-beta"] != 0 && refs["uid-alpha"] == refs["uid-beta"] {
		t.Fatalf("distinct UIDs folded onto one CustomerProfile: %v", refs)
	}

	var identityCount, originCount int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error; err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if err := gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&originCount).Error; err != nil {
		t.Fatalf("count origins: %v", err)
	}
	if identityCount != 2 || originCount != 0 {
		t.Fatalf("entitlement leftover strategy must still create CustomerIdentity, not order-scoped origins: identities=%d origins=%d", identityCount, originCount)
	}
}

func TestImportDemandCSV_RetailSalesOrderCreatesOrderScopedProvisionalProfiles(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:              "bili-retail-orders",
		SourceChannel:           "bilibili",
		SourceSurface:           string(domain.SourceSurfaceRetail),
		DemandKind:              string(domain.DemandKindRetailOrder),
		IdentityStrategy:        app.IdentityStrategyOrderScopedProvisional,
		RequiresExternalOrderNo: true,
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	const rules = `{
		"version":2,
		"mode":"header",
		"hasHeader":true,
		"columns":{
			"document.source_document_no":"Order",
			"document.source_customer_ref":"Buyer",
			"line.external_title":"Item",
			"line.requested_quantity":"Qty"
		},
		"defaults":{"line.line_type":"sku_order"},
		"required":["document.source_document_no"]
	}`
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "bili-retail-orders-template", DocumentType: "import_sales_order", Format: "xls", MappingRules: rules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_sales_order", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	importInput := dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_sales_order",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Order": "ORDER-1", "Buyer": "same-buyer", "Item": "Standee", "Qty": "1"},
			{"Order": "ORDER-2", "Buyer": "same-buyer", "Item": "Badge", "Qty": "2"},
			{"Order": "ORDER-1", "Buyer": "same-buyer", "Item": "Postcard", "Qty": "1"},
		},
	}
	result, err := c.ImportDemandCSV(importInput)
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.SuccessCount != 3 || result.ErrorCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Documents) != 2 {
		t.Fatalf("expected Documents to list both order documents, got %d", len(result.Documents))
	}

	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		t.Fatalf("List docs: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("same buyer's two orders were merged: docs=%+v", docs)
	}
	byOrder := make(map[string]domain.DemandDocument, len(docs))
	for _, doc := range docs {
		byOrder[doc.SourceDocumentNo] = doc
		if doc.Kind != string(domain.DemandKindRetailOrder) {
			t.Errorf("order %q Kind=%q, want retail_order", doc.SourceDocumentNo, doc.Kind)
		}
		if doc.SourceSurface != string(domain.SourceSurfaceRetail) {
			t.Errorf("order %q SourceSurface=%q, want retail", doc.SourceDocumentNo, doc.SourceSurface)
		}
		if doc.SourceCustomerRef != "same-buyer" {
			t.Errorf("order %q lost customer ref: %+v", doc.SourceDocumentNo, doc)
		}
		if doc.CustomerProfileID == nil {
			t.Errorf("order %q missing resolved customer profile", doc.SourceDocumentNo)
		}
	}
	if byOrder["ORDER-1"].CustomerProfileID == nil || byOrder["ORDER-2"].CustomerProfileID == nil ||
		*byOrder["ORDER-1"].CustomerProfileID == *byOrder["ORDER-2"].CustomerProfileID {
		t.Fatalf("different orders must use different provisional profiles even when nicknames match: %+v", byOrder)
	}
	var identityCount int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Where("identity_value = ?", "same-buyer").Count(&identityCount).Error; err != nil {
		t.Fatalf("count buyer nickname identities: %v", err)
	}
	if identityCount != 0 {
		t.Fatalf("buyer nickname must not become CustomerIdentity, got %d", identityCount)
	}
	var origins []persistence.CustomerProfileOrigin
	if err := gdb.Order("external_ref").Find(&origins).Error; err != nil {
		t.Fatalf("list retail origins: %v", err)
	}
	if len(origins) != 2 || origins[0].ExternalRef != "ORDER-1" || origins[1].ExternalRef != "ORDER-2" {
		t.Fatalf("unexpected retail origins: %+v", origins)
	}
	for order, wantLineNos := range map[string][]int{"ORDER-1": {1, 3}, "ORDER-2": {2}} {
		lines, listErr := c.demandRepo.ListLinesByDocument(ctx, byOrder[order].ID)
		if listErr != nil || len(lines) != len(wantLineNos) {
			t.Fatalf("order %s lines=%+v err=%v", order, lines, listErr)
		}
		for i, wantLineNo := range wantLineNos {
			if lines[i].SourceLineNo != wantLineNo {
				t.Errorf("order %s line %d SourceLineNo=%d, want %d", order, i, lines[i].SourceLineNo, wantLineNo)
			}
		}
	}
	_, err = c.ImportDemandCSV(importInput)
	if err == nil {
		t.Fatal("reimporting the same source_document_no must not create a second demand document")
	}
	if !strings.Contains(err.Error(), "duplicate demand document") {
		t.Fatalf("reimport error=%v", err)
	}
	var originCount int64
	if err := gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&originCount).Error; err != nil || originCount != 2 {
		t.Fatalf("reimport mutated origins: count=%d err=%v", originCount, err)
	}
}

func TestImportDemandCSV_IgnoresProfileIdentityStrategyForSalesOrder(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:    "leftover-platform-uid-retail",
		SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail),
		DemandKind:    string(domain.DemandKindRetailOrder),
		// Leftover membership strategy must not override import_sales_order semantics.
		IdentityStrategy:        "platform_uid",
		RequiresExternalOrderNo: true,
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	const rules = `{
		"version":2,
		"mode":"header",
		"hasHeader":true,
		"columns":{
			"document.source_document_no":"Order",
			"document.source_customer_ref":"Buyer",
			"line.external_title":"Item",
			"line.requested_quantity":"Qty"
		},
		"defaults":{"line.line_type":"sku_order"},
		"required":["document.source_document_no"]
	}`
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "leftover-sales-order-template", DocumentType: "import_sales_order", Format: "csv", MappingRules: rules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := infra.NewProfileTemplateBindingRepository(gdb).Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_sales_order", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_sales_order",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Order": "ORDER-1", "Buyer": "same-buyer", "Item": "Standee", "Qty": "1"},
			{"Order": "ORDER-2", "Buyer": "same-buyer", "Item": "Badge", "Qty": "2"},
		},
	})
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.SuccessCount != 2 || result.ErrorCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}

	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		t.Fatalf("List docs: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("same buyer's two orders were merged under leftover platform_uid: docs=%+v", docs)
	}
	byOrder := make(map[string]domain.DemandDocument, len(docs))
	for _, doc := range docs {
		byOrder[doc.SourceDocumentNo] = doc
		if doc.Kind != string(domain.DemandKindRetailOrder) {
			t.Errorf("order %q Kind=%q, want retail_order", doc.SourceDocumentNo, doc.Kind)
		}
		if doc.SourceSurface != string(domain.SourceSurfaceRetail) {
			t.Errorf("order %q SourceSurface=%q, want retail", doc.SourceDocumentNo, doc.SourceSurface)
		}
		if doc.CustomerProfileID == nil {
			t.Fatalf("order %q missing CustomerProfileID", doc.SourceDocumentNo)
		}
	}
	if byOrder["ORDER-1"].CustomerProfileID == nil || byOrder["ORDER-2"].CustomerProfileID == nil ||
		*byOrder["ORDER-1"].CustomerProfileID == *byOrder["ORDER-2"].CustomerProfileID {
		t.Fatalf("different orders must use different provisional profiles even when nicknames match: %+v", byOrder)
	}
	var identityCount int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Where("identity_value = ?", "same-buyer").Count(&identityCount).Error; err != nil {
		t.Fatalf("count buyer nickname identities: %v", err)
	}
	if identityCount != 0 {
		t.Fatalf("buyer nickname must not become CustomerIdentity, got %d", identityCount)
	}
}

func TestImportDemandDocument_RetailStrategyIsOrderScopedAndIdempotent(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-retail", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	input := dto.CreateDemandInput{
		Kind: "import_sales_order", IntegrationProfileID: &profileID, CaptureMode: "manual", SourceDocumentNo: "MANUAL-ORDER-1",
		SourceCustomerRef: "same-nickname", Lines: []dto.CreateDemandLineInput{{LineType: "sku_order", RequestedQuantity: 1}},
	}
	first, err := c.ImportDemandDocument(input)
	if err != nil || first.CustomerProfileID == nil {
		t.Fatalf("first manual retail import: doc=%+v err=%v", first, err)
	}
	if first.Kind != string(domain.DemandKindRetailOrder) {
		t.Errorf("Kind=%q, want retail_order", first.Kind)
	}
	if first.SourceSurface != string(domain.SourceSurfaceRetail) {
		t.Errorf("SourceSurface=%q, want retail", first.SourceSurface)
	}
	_, err = c.ImportDemandDocument(input)
	if err == nil {
		t.Fatal("same-order manual retry must not create a duplicate demand document")
	}
	if !strings.Contains(err.Error(), "duplicate demand document") {
		t.Fatalf("same-order retry error=%v", err)
	}
	input.SourceDocumentNo = "MANUAL-ORDER-2"
	third, err := c.ImportDemandDocument(input)
	if err != nil || third.CustomerProfileID == nil {
		t.Fatalf("different-order manual import: doc=%+v err=%v", third, err)
	}
	if *third.CustomerProfileID == *first.CustomerProfileID {
		t.Fatalf("different orders with the same nickname shared profile %d", *third.CustomerProfileID)
	}
	var identities, origins int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Count(&identities).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&origins).Error; err != nil {
		t.Fatal(err)
	}
	if identities != 0 || origins != 2 {
		t.Fatalf("retail nickname leaked into stable identity or origins duplicated: identities=%d origins=%d", identities, origins)
	}
}

func TestImportDemandDocument_KindFromDocumentTypeShapedInput(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-sales-order-doctype", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	doc, err := c.ImportDemandDocument(dto.CreateDemandInput{
		Kind: "import_sales_order", IntegrationProfileID: &profileID, CaptureMode: "manual",
		SourceDocumentNo: "DOCTYPE-ORDER-1", SourceCustomerRef: "nickname-only",
		Lines: []dto.CreateDemandLineInput{{LineType: "sku_order", RequestedQuantity: 1}},
	})
	if err != nil || doc.CustomerProfileID == nil {
		t.Fatalf("manual import with documentType-shaped Kind: doc=%+v err=%v", doc, err)
	}
	if doc.Kind != string(domain.DemandKindRetailOrder) {
		t.Fatalf("persisted Kind=%q, want retail_order (mapped from import_sales_order)", doc.Kind)
	}
	if doc.SourceSurface != string(domain.SourceSurfaceRetail) {
		t.Fatalf("persisted SourceSurface=%q, want retail", doc.SourceSurface)
	}
	var identities, origins int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Count(&identities).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&origins).Error; err != nil {
		t.Fatal(err)
	}
	if identities != 0 || origins != 1 {
		t.Fatalf("nickname leaked into stable identity or origin missing: identities=%d origins=%d", identities, origins)
	}
}

func TestImportDemandDocument_RetailStrategyRequiresOrderNumber(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-retail-missing-order", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	_, err := c.ImportDemandDocument(dto.CreateDemandInput{
		Kind: "import_sales_order", IntegrationProfileID: &profileID, CaptureMode: "manual", SourceCustomerRef: "nickname",
	})
	if err == nil || !strings.Contains(err.Error(), "requires sourceDocumentNo") {
		t.Fatalf("missing order number error=%v", err)
	}
	var profiles, origins, docs int64
	_ = gdb.Model(&persistence.CustomerProfile{}).Count(&profiles).Error
	_ = gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&origins).Error
	_ = gdb.Model(&persistence.DemandDocument{}).Count(&docs).Error
	if profiles != 0 || origins != 0 || docs != 0 {
		t.Fatalf("missing order validation leaked writes: profiles=%d origins=%d docs=%d", profiles, origins, docs)
	}
}

func TestImportDemandFromCSV_RetailUsesOrderScopedResolution(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "legacy-csv-retail", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "legacy-csv-retail-template", DocumentType: "import_sales_order", Format: "csv",
		MappingRules: `{"columns":{"external_title":"Name","requested_quantity":"Qty"},"defaults":{"line_type":"sku_order"}}`,
	}
	if err := infra.NewDocumentTemplateRepository(gdb).Create(appContext, tmpl); err != nil {
		t.Fatal(err)
	}
	if err := infra.NewProfileTemplateBindingRepository(gdb).Create(appContext, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_sales_order", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatal(err)
	}
	input := dto.ImportDemandTemplateInput{
		IntegrationProfileID: profile.ID, DocumentType: "import_sales_order", SourceDocumentNo: "LEGACY-ORDER-1",
		SourceCustomerRef: "nickname-only", Rows: []map[string]string{{"Name": "Gift", "Qty": "1"}},
	}
	first, err := c.ImportDemandFromCSV(input)
	if err != nil || first.CustomerProfileID == nil {
		t.Fatalf("legacy CSV retail import: doc=%+v err=%v", first, err)
	}
	if first.Kind != string(domain.DemandKindRetailOrder) {
		t.Errorf("Kind=%q, want retail_order", first.Kind)
	}
	if first.SourceSurface != string(domain.SourceSurfaceRetail) {
		t.Errorf("SourceSurface=%q, want retail", first.SourceSurface)
	}
	_, err = c.ImportDemandFromCSV(input)
	if err == nil {
		t.Fatal("legacy CSV same-order retry must not create a duplicate demand document")
	}
	if !strings.Contains(err.Error(), "duplicate demand document") {
		t.Fatalf("legacy CSV same-order retry: first=%+v err=%v", first, err)
	}
	var identityCount int64
	if err := gdb.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error; err != nil || identityCount != 0 {
		t.Fatalf("legacy retail nickname became stable identity: count=%d err=%v", identityCount, err)
	}
}

func TestImportDemandDocument_MembershipStrategyRemainsStableAcrossDocuments(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-membership", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceMembership), DemandKind: string(domain.DemandKindMembershipEntitlement),
		IdentityStrategy: "platform_uid",
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	input := dto.CreateDemandInput{
		Kind: "import_entitlement", IntegrationProfileID: &profileID, CaptureMode: "manual",
		SourceDocumentNo: "MEMBER-DOC-1", SourceCustomerRef: "uid-stable",
	}
	first, err := c.ImportDemandDocument(input)
	if err != nil || first.CustomerProfileID == nil {
		t.Fatalf("first membership import: doc=%+v err=%v", first, err)
	}
	if first.Kind != string(domain.DemandKindMembershipEntitlement) {
		t.Errorf("Kind=%q, want membership_entitlement", first.Kind)
	}
	if first.SourceSurface != string(domain.SourceSurfaceMembership) {
		t.Errorf("SourceSurface=%q, want membership", first.SourceSurface)
	}
	input.SourceDocumentNo = "MEMBER-DOC-2"
	second, err := c.ImportDemandDocument(input)
	if err != nil || second.CustomerProfileID == nil || *second.CustomerProfileID != *first.CustomerProfileID {
		t.Fatalf("membership identity was not stable: first=%+v second=%+v err=%v", first, second, err)
	}
	var identityCount, originCount int64
	_ = gdb.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error
	_ = gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&originCount).Error
	if identityCount != 1 || originCount != 0 {
		t.Fatalf("membership resolution rows: identities=%d origins=%d", identityCount, originCount)
	}
}

func TestImportDemandCSV_PersistenceFailureRollsBackProfilesIdentitiesNamesAndDocuments(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "rollback-membership", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceMembership), DemandKind: string(domain.DemandKindMembershipEntitlement),
		IdentityStrategy: "platform_uid",
	}
	if err := profileRepo.Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{TemplateKey: "rollback-membership-template", DocumentType: "import_entitlement", Format: "csv", MappingRules: `{
		"version":2,"mode":"header","hasHeader":true,
		"columns":{"document.source_customer_ref":"UID","document.display_name":"Name","line.external_title":"Item"},
		"defaults":{"line.line_type":"entitlement_rule","line.requested_quantity":"1"},
		"required":["document.source_customer_ref"]
	}`}
	if err := templateRepo.Create(appContext, tmpl); err != nil {
		t.Fatal(err)
	}
	if err := infra.NewProfileTemplateBindingRepository(gdb).Create(appContext, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_entitlement", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`CREATE TRIGGER fail_uid_2 BEFORE INSERT ON demand_documents
		WHEN NEW.source_customer_ref = 'uid-2'
		BEGIN SELECT RAISE(ABORT, 'injected persistence failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID, DocumentType: "import_entitlement", ImportMode: "reject_all",
		Rows: []map[string]string{
			{"UID": "uid-1", "Name": "A", "Item": "Gift A"},
			{"UID": "uid-2", "Name": "B", "Item": "Gift B"},
		},
	})
	if err == nil {
		t.Fatal("expected injected persistence error")
	}
	assertDemandImportEvidenceFailed(t, gdb, result.ImportRunID)
	checks := []struct {
		name  string
		model any
	}{
		{name: "profiles", model: &persistence.CustomerProfile{}},
		{name: "identities", model: &persistence.CustomerIdentity{}},
		{name: "documents", model: &persistence.DemandDocument{}},
		{name: "name observations", model: &persistence.CustomerNameObservation{}},
		{name: "name events", model: &persistence.CustomerNameEvent{}},
	}
	for _, check := range checks {
		var count int64
		if countErr := gdb.Model(check.model).Count(&count).Error; countErr != nil || count != 0 {
			t.Fatalf("%s survived rollback: count=%d err=%v", check.name, count, countErr)
		}
	}
}

func TestImportDemandCSV_RequestMappingOverrideIsValidatedAndNotPersisted(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)

	before, err := templateRepo.FindByKey(appContext, "csv-import-template")
	if err != nil || before == nil {
		t.Fatalf("default template before import: template=%v err=%v", before, err)
	}
	const override = `{
		"version":2,
		"mode":"header",
		"hasHeader":true,
		"columns":{"document.source_customer_ref":"UID","line.external_title":"Item","line.requested_quantity":"Count"},
		"defaults":{"line.line_type":"sku_order"},
		"transforms":{"line.requested_quantity":["trim"]},
		"required":["line.requested_quantity"]
	}`
	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		MappingRules:         override,
		Rows:                 []map[string]string{{"UID": "uid-override", "Item": "Override Item", "Count": " 3 "}},
	})
	if err != nil {
		t.Fatalf("ImportDemandCSV override: %v", err)
	}
	if result.Document == nil || result.SuccessCount != 1 {
		t.Fatalf("override result: %+v", result)
	}
	lines, err := c.demandRepo.ListLinesByDocument(appContext, result.Document.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("override lines=%+v err=%v", lines, err)
	}
	if lines[0].ExternalTitle != "Override Item" || lines[0].RequestedQuantity != 3 {
		t.Fatalf("override was not applied: %+v", lines[0])
	}
	after, _ := templateRepo.FindByKey(appContext, "csv-import-template")
	if after == nil || after.ID != before.ID || after.MappingRules != before.MappingRules {
		t.Fatalf("request override mutated default template: before=%+v after=%+v", before, after)
	}
	requiredResult, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		MappingRules:         override,
		Rows:                 []map[string]string{{"UID": "uid-override", "Item": "Missing Count"}},
	})
	if err != nil {
		t.Fatalf("required override import: %v", err)
	}
	if requiredResult.SuccessCount != 0 || requiredResult.ErrorCount != 1 {
		t.Fatalf("override required rule was bypassed: %+v", requiredResult)
	}

	_, err = c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		MappingRules: `{
			"version":2,
			"mode":"header",
			"hasHeader":true,
			"columns":{"shipment.tracking_no":"Tracking"}
		}`,
		Rows: []map[string]string{{"Tracking": "T-1"}},
	})
	if err == nil {
		t.Fatal("expected illegal override dest to be rejected")
	}
}

func TestImportDemandCSV_EmptyDocumentTypeIsRejected(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	_, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		ImportMode:           "reject_all",
		Rows:                 []map[string]string{{"Name": "Standee", "Qty": "2"}},
	})
	if err == nil || !strings.Contains(err.Error(), "explicit documentType") {
		t.Fatalf("empty documentType error=%v", err)
	}
}

func TestImportDemandCSV_SourcePlatformAcceptsEntitlementOnRetailLeftover(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "retail-leftover-accepts-entitlement", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := profileRepo.Create(appContext, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		MappingRules: `{
			"version":2,"mode":"header","hasHeader":true,
			"columns":{"document.source_customer_ref":"UID","line.external_title":"Name"},
			"defaults":{"line.line_type":"entitlement_rule","line.requested_quantity":"1"},
			"required":["document.source_customer_ref"]
		}`,
		Rows: []map[string]string{{"UID": "uid-cross", "Name": "Gift"}},
	})
	if err != nil {
		t.Fatalf("source platform must accept import_entitlement: %v", err)
	}
	if result.Document == nil || result.Document.Kind != string(domain.DemandKindMembershipEntitlement) {
		t.Fatalf("Kind from documentType, not leftover DemandKind: %+v", result.Document)
	}
	if result.Document.SourceSurface != string(domain.SourceSurfaceMembership) {
		t.Fatalf("SourceSurface from documentType, want membership, got %+v", result.Document)
	}
	var identities, origins int64
	_ = gdb.Model(&persistence.CustomerIdentity{}).Count(&identities).Error
	_ = gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&origins).Error
	if identities != 1 || origins != 0 {
		t.Fatalf("entitlement on retail leftover must use platform_uid: identities=%d origins=%d", identities, origins)
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
		MappingRules: `{
			"version":2,"mode":"header","hasHeader":true,
			"columns":{
				"document.source_customer_ref":"UID",
				"line.external_title":"Name",
				"line.requested_quantity":"Qty"
			},
			"defaults":{"line.line_type":"sku_order"}
		}`,
		Rows: []map[string]string{
			{"UID": "uid-1", "Name": "Standee", "Qty": "2"},
			{"UID": "uid-2", "Name": "Poster", "Qty": "not-a-number"},
			{"UID": "uid-3", "Name": "Sticker", "Qty": "5"},
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
	if len(result.Documents) != 2 {
		t.Fatalf("distinct UIDs must persist as separate documents, got %d", len(result.Documents))
	}

	lineNos := map[int]bool{}
	for _, doc := range result.Documents {
		if doc.CustomerProfileID == nil {
			t.Fatalf("membership document %d persisted without customer", doc.ID)
		}
		lines, err := c.demandRepo.ListLinesByDocument(appContext, doc.ID)
		if err != nil {
			t.Fatalf("ListLinesByDocument: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("expected 1 line per UID document, got %d on doc %d", len(lines), doc.ID)
		}
		lineNos[lines[0].SourceLineNo] = true
	}
	if !lineNos[1] || !lineNos[3] {
		t.Fatalf("SourceLineNo must preserve original row positions across skipped rows: %v", lineNos)
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

func assertDemandImportEvidenceFailed(t *testing.T, gdb *gorm.DB, importRunID uint) {
	t.Helper()
	if importRunID == 0 {
		t.Fatal("expected ImportRunID on failure")
	}
	var run persistence.ImportRun
	if err := gdb.First(&run, importRunID).Error; err != nil {
		t.Fatalf("load import run %d: %v", importRunID, err)
	}
	if run.Status != "failed" && run.Status != "rejected" {
		t.Fatalf("evidence run status=%q, want failed or rejected", run.Status)
	}
	var records []persistence.ImportRawRecord
	if err := gdb.Where("import_run_id = ?", importRunID).Find(&records).Error; err != nil {
		t.Fatalf("list evidence records: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected evidence rows on failure")
	}
	for _, rec := range records {
		if rec.Outcome == "success" {
			t.Fatalf("evidence row marked success after failure: %+v", rec)
		}
		if rec.ResultID != nil {
			t.Fatalf("evidence result_id=%d leaked after failure", *rec.ResultID)
		}
	}
}

func seedRecipientDemandCSVFixture(t *testing.T, gdb *gorm.DB) uint {
	t.Helper()
	ctx := appContext
	profile := &domain.IntegrationProfile{
		ProfileKey:       "recipient-csv",
		SourceChannel:    "csv",
		SourceSurface:    string(domain.SourceSurfaceMembership),
		DemandKind:       string(domain.DemandKindMembershipEntitlement),
		IdentityStrategy: "platform_uid",
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	tmpl := &domain.DocumentTemplate{
		TemplateKey:  "recipient-csv-template",
		DocumentType: "import_entitlement",
		Format:       "csv",
		MappingRules: `{
			"version":2,
			"mode":"header",
			"hasHeader":true,
			"columns":{
				"document.source_customer_ref":"UID",
				"document.display_name":"Name",
				"line.external_title":"Item",
				"recipient.name":"RecvName",
				"recipient.phone":"Phone",
				"recipient.address_line1":"Addr"
			},
			"defaults":{"line.line_type":"entitlement_rule","line.requested_quantity":"1"},
			"required":["document.source_customer_ref"]
		}`,
	}
	if err := infra.NewDocumentTemplateRepository(gdb).Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := infra.NewProfileTemplateBindingRepository(gdb).Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_entitlement", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	return profile.ID
}

func TestParseTabularFile_RejectsNULDotDotAndNonTabularExt(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)

	if _, err := c.ParseTabularFile("foo\x00.csv", true); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("NUL path error=%v", err)
	}
	if _, err := c.ParseCSVFile("foo\x00.csv"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("ParseCSVFile NUL path error=%v", err)
	}
	dotDot := `downloads\..\secret.csv`
	if _, err := c.ParseTabularFile(dotDot, true); err == nil || !strings.Contains(err.Error(), "..") {
		t.Fatalf("dot-dot path error=%v", err)
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "sheet.zip")
	if err := os.WriteFile(zipPath, []byte("not-a-sheet"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ParseTabularFile(zipPath, true); err == nil {
		t.Fatal("expected error for .zip (not a demand tabular format)")
	}

	tiny := filepath.Join(dir, "tiny.csv")
	if err := os.WriteFile(tiny, []byte("Name,Qty\nStandee,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateDemandImportFilePath(tiny, 1, demandTabularAllowedExts()); err == nil {
		t.Fatal("expected oversize error when maxBytes is 1")
	}
	if err := validateDemandImportFilePath(tiny, demandTabularMaxBytes, demandTabularAllowedExts()); err != nil {
		t.Fatalf("valid csv should pass: %v", err)
	}
}

func TestImportDemandCSV_FilePathRejectsDotDotAndNUL(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedDemandCSVImportFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	_, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		FilePath:             "foo\x00.csv",
	})
	if err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("FilePath NUL error=%v", err)
	}
	_, err = c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		FilePath:             `tmp\..\import.csv`,
	})
	if err == nil || !strings.Contains(err.Error(), "..") {
		t.Fatalf("FilePath dot-dot error=%v", err)
	}
}

func TestImportDemandCSV_RecipientColumnsPersistCustomerAddress(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedRecipientDemandCSVFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"UID": "uid-recv", "Name": "Member", "Item": "Gift", "RecvName": "Alice", "Phone": "123456", "Addr": "1 Flower St"},
		},
	})
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.Document == nil || result.SuccessCount != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	var addrs []persistence.CustomerAddress
	if err := gdb.Find(&addrs).Error; err != nil {
		t.Fatalf("list addresses: %v", err)
	}
	if len(addrs) != 1 {
		t.Fatalf("expected 1 CustomerAddress from recipient.* columns, got %d", len(addrs))
	}
	if addrs[0].RecipientName != "Alice" || addrs[0].Phone != "123456" || addrs[0].AddressLine1 != "1 Flower St" {
		t.Fatalf("address fields: %+v", addrs[0])
	}
	if result.Document.CustomerProfileID == nil || addrs[0].CustomerProfileID != *result.Document.CustomerProfileID {
		t.Fatalf("address profile=%d document profile=%v", addrs[0].CustomerProfileID, result.Document.CustomerProfileID)
	}
}

func TestImportDemandCSV_SkipInvalid_AddressFailureDoesNotPersistThatDocument(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedRecipientDemandCSVFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)
	if err := gdb.Exec(`CREATE TRIGGER fail_bad_recipient BEFORE INSERT ON customer_addresses
		WHEN NEW.recipient_name = 'Bad Recipient'
		BEGIN SELECT RAISE(ABORT, 'injected address failure'); END`).Error; err != nil {
		t.Fatal(err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "skip_invalid",
		Rows: []map[string]string{
			{"UID": "uid-good", "Name": "Good", "Item": "Gift A", "RecvName": "Alice", "Phone": "111", "Addr": "Good St"},
			{"UID": "uid-bad", "Name": "Bad", "Item": "Gift B", "RecvName": "Bad Recipient", "Phone": "222", "Addr": "Bad St"},
		},
	})
	if err != nil {
		t.Fatalf("skip_invalid should not hard-fail: %v", err)
	}
	if result.SuccessCount != 1 || result.ErrorCount != 1 || result.Document == nil {
		t.Fatalf("unexpected skip_invalid address result: %+v", result)
	}
	if len(result.Documents) != 1 {
		t.Fatalf("expected only the successful document, got %d", len(result.Documents))
	}
	docs, err := c.demandRepo.List(appContext)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].SourceCustomerRef != "uid-good" {
		t.Fatalf("failed address row was persisted: %+v", docs)
	}
	var addrs []persistence.CustomerAddress
	if err := gdb.Find(&addrs).Error; err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0].RecipientName != "Alice" {
		t.Fatalf("unexpected addresses: %+v", addrs)
	}
	foundAddrErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Reason, "address upsert") {
			foundAddrErr = true
		}
	}
	if !foundAddrErr {
		t.Fatalf("expected address upsert error, got %+v", result.Errors)
	}
}

func TestImportDemandCSV_RejectAll_AddressFailureRollsBackAndFailsImport(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profileID := seedRecipientDemandCSVFixture(t, gdb)
	c := newDemandCSVImportTestController(gdb)
	if err := gdb.Exec(`CREATE TRIGGER fail_bad_recipient BEFORE INSERT ON customer_addresses
		WHEN NEW.recipient_name = 'Bad Recipient'
		BEGIN SELECT RAISE(ABORT, 'injected address failure'); END`).Error; err != nil {
		t.Fatal(err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profileID,
		DocumentType:         "import_entitlement",
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"UID": "uid-good", "Name": "Good", "Item": "Gift A", "RecvName": "Alice", "Phone": "111", "Addr": "Good St"},
			{"UID": "uid-bad", "Name": "Bad", "Item": "Gift B", "RecvName": "Bad Recipient", "Phone": "222", "Addr": "Bad St"},
		},
	})
	if err == nil {
		t.Fatal("reject_all must fail the whole import on address errors")
	}
	if !strings.Contains(err.Error(), "address upsert") {
		t.Fatalf("error=%v", err)
	}
	assertDemandImportEvidenceFailed(t, gdb, result.ImportRunID)

	docs, listErr := c.demandRepo.List(appContext)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(docs) != 0 {
		t.Fatalf("reject_all committed documents: %+v", docs)
	}
	var addrCount, profileCount int64
	_ = gdb.Model(&persistence.CustomerAddress{}).Count(&addrCount).Error
	_ = gdb.Model(&persistence.CustomerProfile{}).Count(&profileCount).Error
	if addrCount != 0 || profileCount != 0 {
		t.Fatalf("reject_all leaked rows: addresses=%d profiles=%d", addrCount, profileCount)
	}
}

func seedEmptyMembershipRefProfile(t *testing.T, gdb *gorm.DB, profileKey string) uint {
	t.Helper()
	profile := &domain.IntegrationProfile{
		ProfileKey: profileKey, SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceMembership), DemandKind: string(domain.DemandKindMembershipEntitlement),
		IdentityStrategy: "platform_uid",
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	return profile.ID
}

const emptyMembershipRefMapping = `{
			"version":2,"mode":"header","hasHeader":true,
			"columns":{"document.source_customer_ref":"UID","line.external_title":"Item"},
			"defaults":{"line.line_type":"entitlement_rule","line.requested_quantity":"1"}
		}`

func TestImportDemandCSV_EmptyMembershipRefDoesNotPersistNilCustomer(t *testing.T) {
	emptyRows := []map[string]string{
		{"UID": "", "Item": "Gift A"},
		{"UID": "", "Item": "Gift B"},
	}

	t.Run("reject_all rolls back", func(t *testing.T) {
		gdb := setupDemandCSVImportTestDB(t)
		c := newDemandCSVImportTestController(gdb)
		profileID := seedEmptyMembershipRefProfile(t, gdb, "empty-ref-membership-reject")

		result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
			IntegrationProfileID: profileID,
			DocumentType:         "import_entitlement",
			ImportMode:           "reject_all",
			MappingRules:         emptyMembershipRefMapping,
			Rows:                 emptyRows,
		})
		if err == nil {
			t.Fatal("reject_all must fail on empty membership source_customer_ref")
		}
		if !strings.Contains(err.Error(), "sourceCustomerRef") {
			t.Fatalf("error=%v", err)
		}
		assertDemandImportEvidenceFailed(t, gdb, result.ImportRunID)
		docs, listErr := c.demandRepo.List(appContext)
		if listErr != nil {
			t.Fatal(listErr)
		}
		if len(docs) != 0 {
			t.Fatalf("reject_all persisted nil-customer documents: %+v", docs)
		}
		var profiles int64
		_ = gdb.Model(&persistence.CustomerProfile{}).Count(&profiles).Error
		if profiles != 0 {
			t.Fatalf("reject_all leaked customer profiles: %d", profiles)
		}
	})

	t.Run("skip_invalid skips without saving", func(t *testing.T) {
		gdb := setupDemandCSVImportTestDB(t)
		c := newDemandCSVImportTestController(gdb)
		profileID := seedEmptyMembershipRefProfile(t, gdb, "empty-ref-membership-skip")

		result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
			IntegrationProfileID: profileID,
			DocumentType:         "import_entitlement",
			ImportMode:           "skip_invalid",
			MappingRules:         emptyMembershipRefMapping,
			Rows:                 emptyRows,
		})
		if err != nil {
			t.Fatalf("skip_invalid should not hard-fail: %v", err)
		}
		if result.SuccessCount != 0 || result.Document != nil || len(result.Documents) != 0 {
			t.Fatalf("empty membership refs must not persist: %+v", result)
		}
		if result.ErrorCount != 2 {
			t.Fatalf("empty refs stay split into per-row groups: ErrorCount=%d errors=%+v", result.ErrorCount, result.Errors)
		}
		seen := map[int]bool{}
		for _, e := range result.Errors {
			seen[e.RowIndex] = true
			if !strings.Contains(e.Reason, "sourceCustomerRef") {
				t.Errorf("row %d reason=%q", e.RowIndex, e.Reason)
			}
		}
		if !seen[0] || !seen[1] {
			t.Fatalf("expected per-row errors at indexes 0 and 1, got %+v", result.Errors)
		}
		docs, listErr := c.demandRepo.List(appContext)
		if listErr != nil {
			t.Fatal(listErr)
		}
		if len(docs) != 0 {
			t.Fatalf("skip_invalid persisted nil-customer documents: %+v", docs)
		}
	})

	t.Run("skip_invalid mixed persists only UID rows", func(t *testing.T) {
		gdb := setupDemandCSVImportTestDB(t)
		c := newDemandCSVImportTestController(gdb)
		profileID := seedEmptyMembershipRefProfile(t, gdb, "empty-ref-membership-mixed")

		result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
			IntegrationProfileID: profileID,
			DocumentType:         "import_entitlement",
			ImportMode:           "skip_invalid",
			MappingRules:         emptyMembershipRefMapping,
			Rows: []map[string]string{
				{"UID": "", "Item": "Gift A"},
				{"UID": "uid-ok", "Item": "Gift B"},
				{"UID": "", "Item": "Gift C"},
			},
		})
		if err != nil {
			t.Fatalf("skip_invalid mixed: %v", err)
		}
		if result.SuccessCount != 1 || result.ErrorCount != 2 || len(result.Documents) != 1 {
			t.Fatalf("unexpected mixed result: %+v", result)
		}
		if result.Documents[0].CustomerProfileID == nil || result.Documents[0].SourceCustomerRef != "uid-ok" {
			t.Fatalf("persisted document must be the UID row with a customer: %+v", result.Documents[0])
		}
		docs, listErr := c.demandRepo.List(appContext)
		if listErr != nil {
			t.Fatal(listErr)
		}
		if len(docs) != 1 || docs[0].SourceCustomerRef != "uid-ok" || docs[0].CustomerProfileID == nil {
			t.Fatalf("nil-customer empty-ref document leaked: %+v", docs)
		}
	})
}

func TestImportDemandDocument_DocumentTypeTakesPrecedenceOverKind(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-doctype-wins", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	doc, err := c.ImportDemandDocument(dto.CreateDemandInput{
		Kind: "import_entitlement", DocumentType: "import_sales_order",
		IntegrationProfileID: &profileID, CaptureMode: "manual",
		SourceDocumentNo: "DOCTYPE-WINS-1", SourceCustomerRef: "nickname-only",
		Lines: []dto.CreateDemandLineInput{{LineType: "sku_order", RequestedQuantity: 1}},
	})
	if err != nil {
		t.Fatalf("ImportDemandDocument: %v", err)
	}
	if doc.Kind != string(domain.DemandKindRetailOrder) {
		t.Fatalf("DocumentType should win: Kind=%q", doc.Kind)
	}
	if doc.SourceSurface != string(domain.SourceSurfaceRetail) {
		t.Fatalf("SourceSurface=%q, want retail", doc.SourceSurface)
	}
}

func TestImportDemandDocument_HonorsExplicitCustomerProfileID(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey: "manual-explicit-customer", SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceRetail), DemandKind: string(domain.DemandKindRetailOrder),
		IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	existing := &domain.CustomerProfile{DisplayName: "explicit-customer", ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1}
	if err := infra.NewProfileRepository(gdb).Create(appContext, existing); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	customerID := existing.ID
	doc, err := c.ImportDemandDocument(dto.CreateDemandInput{
		Kind: "import_sales_order", IntegrationProfileID: &profileID, CaptureMode: "manual",
		SourceDocumentNo: "EXPLICIT-ORDER-1", SourceCustomerRef: "should-not-replace",
		CustomerProfileID: &customerID,
		Lines:             []dto.CreateDemandLineInput{{LineType: "sku_order", RequestedQuantity: 1}},
	})
	if err != nil {
		t.Fatalf("ImportDemandDocument: %v", err)
	}
	if doc.CustomerProfileID == nil || *doc.CustomerProfileID != customerID {
		t.Fatalf("explicit customerProfileId was replaced: want %d got %+v", customerID, doc.CustomerProfileID)
	}
	var profiles, origins int64
	_ = gdb.Model(&persistence.CustomerProfile{}).Count(&profiles).Error
	_ = gdb.Model(&persistence.CustomerProfileOrigin{}).Count(&origins).Error
	if profiles != 1 {
		t.Fatalf("order-scoped resolution created an extra profile: count=%d", profiles)
	}
	if origins != 0 {
		t.Fatalf("explicit customerProfileId should skip provisional origin writes: origins=%d", origins)
	}
}
