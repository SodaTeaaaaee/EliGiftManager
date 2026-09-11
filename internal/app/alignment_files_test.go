package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/xuri/excelize/v2"
)

func writeTempFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func mustSerializeMapping(t *testing.T, cfg alignment.MappingConfig) string {
	t.Helper()
	raw, err := alignment.SerializeMappingConfig(cfg)
	if err != nil {
		t.Fatalf("serialize mapping: %v", err)
	}
	return raw
}

func platformByKind(t *testing.T, ws *Workspace, kind domain.PlatformKind) domain.Platform {
	t.Helper()
	platforms, err := ws.ListPlatforms(context.Background())
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	for _, p := range platforms {
		if p.Kind == string(kind) {
			return p
		}
	}
	t.Fatalf("no %s platform", kind)
	return domain.Platform{}
}

// seedBuiltins runs the startup seeding: platforms first, then the read-only
// template catalog.
func seedBuiltins(t *testing.T, ws *Workspace) {
	t.Helper()
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	if err := ws.EnsureBuiltinTemplates(ctx); err != nil {
		t.Fatalf("EnsureBuiltinTemplates: %v", err)
	}
}

// builtinTemplate finds the seeded read-only template for a platform and
// document type.
func builtinTemplate(t *testing.T, ws *Workspace, platformID uint, documentType string) domain.TemplateConfig {
	t.Helper()
	templates, err := ws.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	for _, tpl := range templates {
		if tpl.Builtin && tpl.PlatformID == platformID && tpl.DocumentType == documentType {
			return tpl
		}
	}
	t.Fatalf("no builtin %s template on platform %d", documentType, platformID)
	return domain.TemplateConfig{}
}

// cloneBuiltinTemplate copies a seeded built-in template into an active user
// template, the way the UI creates templates from the catalog.
func cloneBuiltinTemplate(t *testing.T, ws *Workspace, platformID uint, documentType string) *domain.TemplateConfig {
	t.Helper()
	src := builtinTemplate(t, ws, platformID, documentType)
	clone := &domain.TemplateConfig{
		PlatformID:   src.PlatformID,
		DocumentType: src.DocumentType,
		Direction:    src.Direction,
		Name:         strings.TrimSuffix(src.Name, "（内置）") + "（副本）",
		Notes:        src.Notes,
		MappingJSON:  src.MappingJSON,
		LayoutJSON:   src.LayoutJSON,
	}
	if err := ws.CreateTemplate(context.Background(), clone); err != nil {
		t.Fatalf("clone builtin %s: %v", documentType, err)
	}
	return clone
}

// readRenderedFile reads an exported csv/xlsx file back into records.
func readRenderedFile(t *testing.T, path string) [][]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export %q: %v", path, err)
	}
	records, err := alignment.ReadRows(raw, alignment.FormatFromExtension(filepath.Ext(path)), "")
	if err != nil {
		t.Fatalf("read rows of %q: %v", path, err)
	}
	return records
}

func TestImportFile_MembershipPositionalEndToEnd(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	tpl := cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeMembershipList)

	path := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_membership_positional.csv")
	result, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if result.FactsCreated != 3 || result.LinesCreated != 3 {
		t.Fatalf("created = %d facts / %d lines, want 3/3", result.FactsCreated, result.LinesCreated)
	}
	if len(result.Duplicates) != 0 || len(result.Issues) != 0 {
		t.Fatalf("duplicates = %v issues = %v", result.Duplicates, result.Issues)
	}
	if result.Document.TemplateID == nil || *result.Document.TemplateID != tpl.ID || result.Document.TemplateVersion != 1 {
		t.Fatalf("document template snapshot = %+v", result.Document.TemplateID)
	}
	if result.Document.OriginalName != "bilibili_membership_positional.csv" {
		t.Fatalf("original name = %q", result.Document.OriginalName)
	}

	// Every identity got a profile named after the display-name column and
	// is attached to it, so nothing is left unattached.
	customers, err := ws.ListCustomers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, c := range customers {
		names[c.DisplayName] = true
	}
	if len(customers) != 3 || !names["DisplayA"] || !names["DisplayB"] || !names["DisplayC"] {
		t.Fatalf("auto-created profiles = %+v, want DisplayA/B/C", customers)
	}
	unattached, err := ws.Store.ListUnattachedIdentities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unattached) != 0 {
		t.Fatalf("unattached identities = %+v, want none", unattached)
	}
	rows, err := ws.ListInboxRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Unattached || r.Fact.CustomerProfileID == nil {
			t.Fatalf("inbox row %+v must resolve its customer", r.Fact)
		}
	}

	// Re-import: same fingerprints must dedupe every fact, and no second
	// profile appears for a known identity.
	again, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if again.FactsCreated != 0 || len(again.Duplicates) != 3 {
		t.Fatalf("re-import created = %d dups = %d, want 0/3", again.FactsCreated, len(again.Duplicates))
	}
	for _, d := range again.Duplicates {
		if d.Reason != "stable_external_id" {
			t.Fatalf("dup reason = %q", d.Reason)
		}
	}
	customers, err = ws.ListCustomers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(customers) != 3 {
		t.Fatalf("customers after re-import = %d, want 3", len(customers))
	}
}

func TestImportFile_RetailOrderWithSkuSplit(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)

	tpl := &domain.TemplateConfig{
		PlatformID:   source.ID,
		DocumentType: DocumentTypeOrderExport,
		Direction:    string(domain.TemplateDirectionInput),
		Name:         "rouzao retail mirror",
		MappingJSON: mustSerializeMapping(t, alignment.MappingConfig{
			Mode: alignment.ModeHeader,
			Columns: map[string]string{
				"source.document_no": "订单号",
				"product.alias_spec": "规格&数量",
			},
			Transforms:       map[string][]string{"source.document_no": {"trim"}},
			SplitSkuQuantity: "product.alias_spec",
			Required:         []string{"source.document_no"},
			Fingerprint:      []string{"source.document_no"},
		}),
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}

	path := writeTempFixture(t, "orders.csv",
		"\uFEFF订单号,规格&数量\nORD-1,111_aaa * 1|222_bbb * 3\nORD-2,333_ccc * 2\n")
	result, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if result.FactsCreated != 2 || result.LinesCreated != 3 {
		t.Fatalf("created = %d facts / %d lines, want 2/3", result.FactsCreated, result.LinesCreated)
	}

	// Verify the split landed on the fact lines.
	rows, err := ws.ListInboxRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bySKU := map[string]int{}
	for _, r := range rows {
		bySKU[r.Line.ExternalSKU] = r.Line.Quantity
	}
	if bySKU["111"] != 1 || bySKU["222"] != 3 || bySKU["333"] != 2 {
		t.Fatalf("line skus/quantities = %v", bySKU)
	}

	// Re-import: document-number stable ids dedupe.
	again, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	if again.FactsCreated != 0 || len(again.Duplicates) != 2 {
		t.Fatalf("re-import created = %d dups = %d, want 0/2", again.FactsCreated, len(again.Duplicates))
	}
}

// readyRetailPath builds a ready-to-execute wave: customer with default
// address, product on the factory platform, alias, and one ingested aligned
// retail fact line assigned to a fresh wave. It returns the workspace-bound
// ids the export tests need.
type readyPath struct {
	ws      *Workspace
	ctx     context.Context
	wave    domain.Wave
	fact    domain.InputFact
	line    domain.InputFactLine
	product domain.ProductItem
	source  domain.Platform
	factory domain.Platform
}

func setupReadyRetailPath(t *testing.T, sku string) *readyPath {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	p := &readyPath{ws: ws, ctx: ctx}
	p.source = platformByKind(t, ws, domain.PlatformKindSource)
	p.factory = platformByKind(t, ws, domain.PlatformKindFactory)

	customer := &domain.CustomerProfile{DisplayName: "Alice"}
	if err := ws.CreateCustomer(ctx, customer); err != nil {
		t.Fatal(err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: customer.ID, RecipientName: "Alice Zhang", Phone: "13800000000", Province: "上海市", City: "上海市", District: "浦东新区", AddressLine1: "世纪大道100号", IsDefault: true}
	if err := ws.CreateAddress(ctx, addr); err != nil {
		t.Fatal(err)
	}
	p.product = domain.ProductItem{Name: "Medal", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MEDAL-001"}
	if err := ws.CreateProduct(ctx, &p.product); err != nil {
		t.Fatal(err)
	}
	alias := &domain.ProductAlias{ProductItemID: p.product.ID, PlatformID: p.source.ID, ExternalProductID: sku}
	if err := ws.CreateAlias(ctx, alias); err != nil {
		t.Fatal(err)
	}

	doc, _, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: p.source.ID, DocumentType: DocumentTypeOrderExport}, []IngestFactInput{{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "ORD-EXPORT-1",
		SourceDocumentNo: "ORD-EXPORT-1",
		IdentityValue:    "uid-777",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: sku, Quantity: 2}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	facts, err := ws.Store.ListFactsByDocument(ctx, doc.ID)
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts = %v err = %v", facts, err)
	}
	p.fact = facts[0]
	lines, err := ws.Store.ListFactLines(ctx, p.fact.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines = %v err = %v", lines, err)
	}
	p.line = lines[0]

	// Attach the identity so the retail result resolves the customer.
	ident, err := ws.Store.GetIdentity(ctx, *p.fact.PlatformIdentityID)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AttachIdentity(ctx, ident.ID, customer.ID); err != nil {
		t.Fatal(err)
	}

	wave, err := ws.CreateWave(ctx, "w1", "")
	if err != nil {
		t.Fatal(err)
	}
	p.wave = *wave
	if err := ws.AssignLines(ctx, wave.ID, []uint{p.line.ID}); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestExportFactoryOrderFile_NoActiveTemplateFails pins the removed fallback:
// generation still works without an output template (links snapshot version
// 0), but exporting the file needs an active template and fails clearly.
func TestExportFactoryOrderFile_NoActiveTemplateFails(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-9")
	ws, ctx := p.ws, p.ctx
	dataDir := t.TempDir()
	ws.ResolveDataDir = func() (string, error) { return dataDir, nil }

	order, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	links, err := ws.Store.ListLinksByOrderLine(ctx, lines[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].ConfigVersion != 0 {
		t.Fatalf("links = %+v, want ConfigVersion 0", links)
	}

	_, err = ws.ExportFactoryOrderFile(ctx, order.ID)
	if !errors.Is(err, ErrNoActiveTemplate) {
		t.Fatalf("export without template err = %v, want ErrNoActiveTemplate", err)
	}
	if !strings.Contains(err.Error(), p.factory.Name) || !strings.Contains(err.Error(), DocumentTypeFactoryOrder) {
		t.Fatalf("error must name the platform and document type: %v", err)
	}
	after, err := ws.Store.GetSupplierOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != string(domain.SupplierOrderGenerated) || after.ExportedAt != nil {
		t.Fatalf("order = %+v, want still generated", after)
	}

	// A built-in template alone does not count: it must be copied first.
	seedBuiltins(t, ws)
	if _, err := ws.ExportFactoryOrderFile(ctx, order.ID); !errors.Is(err, ErrNoActiveTemplate) {
		t.Fatalf("export with only a builtin template err = %v, want ErrNoActiveTemplate", err)
	}
}

// TestExportFactoryOrderFile_BuiltinCloneLayout renders through a copy of the
// seeded rouzao layout: an xlsx with the six bulk-order columns.
func TestExportFactoryOrderFile_BuiltinCloneLayout(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-9B")
	ws, ctx := p.ws, p.ctx
	seedBuiltins(t, ws)
	tpl := cloneBuiltinTemplate(t, ws, p.factory.ID, DocumentTypeFactoryOrder)
	dataDir := t.TempDir()
	ws.ResolveDataDir = func() (string, error) { return dataDir, nil }

	order, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	links, err := ws.Store.ListLinksByOrderLine(ctx, lines[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].ConfigVersion != 1 {
		t.Fatalf("links = %+v, want ConfigVersion 1", links)
	}

	result, err := ws.ExportFactoryOrderFile(ctx, order.ID)
	if err != nil {
		t.Fatalf("ExportFactoryOrderFile: %v", err)
	}
	if !strings.Contains(result.Path, filepath.Join("exports", "factory-orders")) || !strings.HasSuffix(result.Path, ".xlsx") {
		t.Fatalf("path = %q", result.Path)
	}
	records := readRenderedFile(t, result.Path)
	if len(records) != 2 {
		t.Fatalf("records = %d, want header + 1 row", len(records))
	}
	if got := strings.Join(records[0], ","); got != "第三方订单号,收件人,联系电话,收件地址,商家编码,下单数量" {
		t.Fatalf("header = %q", got)
	}
	row := records[1]
	if row[0] != lines[0].TrackingID || row[1] != "Alice Zhang" || row[4] != "ROZAO-MEDAL-001" || row[5] != "2" {
		t.Fatalf("row = %q", row)
	}
	updated, err := ws.Store.GetSupplierOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != string(domain.SupplierOrderExported) || updated.ExportedAt == nil {
		t.Fatalf("order state = %+v", updated)
	}
	if !strings.HasPrefix(updated.ExportPayload, "base64:") {
		t.Fatal("xlsx ExportPayload must be base64-prefixed")
	}
	if updated.TemplateID != tpl.ID || updated.TemplateVersion != 1 {
		t.Fatalf("template snapshot = %d/%d, want %d/1", updated.TemplateID, updated.TemplateVersion, tpl.ID)
	}
	// Export-after-export keeps the exported state machine semantics.
	if err := ws.VoidFactoryOrder(ctx, order.ID); err != ErrOrderExported {
		t.Fatalf("void after export err = %v, want ErrOrderExported", err)
	}
}

func TestExportFactoryOrderFile_TemplateLayoutSnapshotsVersion(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-10")
	ws, ctx := p.ws, p.ctx
	layoutRaw, err := alignment.SerializeLayoutConfig(alignment.LayoutConfig{
		Format:      alignment.FormatCSV,
		ColumnOrder: []string{"tracking.id", "product.factory_sku", "quantity"},
		HeaderNames: map[string]string{"tracking.id": "单号", "product.factory_sku": "编码", "quantity": "数量"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tpl := &domain.TemplateConfig{
		PlatformID:   p.factory.ID,
		DocumentType: DocumentTypeFactoryOrder,
		Direction:    string(domain.TemplateDirectionOutput),
		Name:         "rozao order import",
		Version:      4, // ignored: new templates always start at 1
		LayoutJSON:   layoutRaw,
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	// Two in-place edits: same row, version 3.
	for i := 0; i < 2; i++ {
		edited := *tpl
		edited.Notes = fmt.Sprintf("edit %d", i+1)
		updated, err := ws.UpdateTemplate(ctx, &edited)
		if err != nil {
			t.Fatalf("UpdateTemplate %d: %v", i+1, err)
		}
		tpl = updated
	}
	if tpl.Version != 3 {
		t.Fatalf("template version after two edits = %d, want 3", tpl.Version)
	}
	dataDir := t.TempDir()
	ws.ResolveDataDir = func() (string, error) { return dataDir, nil }

	order, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Links snapshot the output template's version.
	links, err := ws.Store.ListLinksByOrderLine(ctx, lines[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].ConfigVersion != 3 {
		t.Fatalf("links = %+v, want ConfigVersion 3", links)
	}

	result, err := ws.ExportFactoryOrderFile(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimPrefix(string(raw), "\uFEFF")
	if !strings.HasPrefix(text, "单号,编码,数量\n") {
		t.Fatalf("template header missing: %q", text)
	}
	updated, err := ws.Store.GetSupplierOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.TemplateID != tpl.ID || updated.TemplateVersion != 3 {
		t.Fatalf("template snapshot = %d/%d", updated.TemplateID, updated.TemplateVersion)
	}
}

func TestParseShipmentQuantity(t *testing.T) {
	cases := []struct {
		in        string
		want      int
		wantIssue bool
	}{
		{"111_x * 1", 1, false},
		{"a * 2|b * 3", 5, false},
		{"sku", 1, true},            // no multiplier defaults to 1, with an issue
		{"sku_x", 1, true},          // title blob without multiplier, with an issue
		{" a*2 | b * 3 ", 5, false}, // surrounding whitespace ignored
		{"a * 2|sku_x|b * 3", 6, true},
	}
	for _, c := range cases {
		got, issues, err := parseShipmentQuantity(c.in)
		if err != nil || got != c.want {
			t.Fatalf("parseShipmentQuantity(%q) = %d, %v; want %d, nil", c.in, got, err, c.want)
		}
		if (len(issues) > 0) != c.wantIssue {
			t.Fatalf("parseShipmentQuantity(%q) issues = %v, wantIssue = %v", c.in, issues, c.wantIssue)
		}
		for _, issue := range issues {
			if issue.Key != "shipment.quantity" {
				t.Fatalf("issue key = %q, want shipment.quantity", issue.Key)
			}
		}
	}
	for _, bad := range []string{"sku * bad", "sku * 0", "sku * -2", "sku * |x * 2", ""} {
		if got, _, err := parseShipmentQuantity(bad); err == nil {
			t.Fatalf("parseShipmentQuantity(%q) = %d, want error", bad, got)
		}
	}
}

func TestImportShipmentFile_EndToEnd(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-11")
	ws, ctx := p.ws, p.ctx
	seedBuiltins(t, ws)
	cloneBuiltinTemplate(t, ws, p.factory.ID, DocumentTypeFactoryOrder)
	returnTpl := cloneBuiltinTemplate(t, ws, p.factory.ID, DocumentTypeShipmentReturn)
	dataDir := t.TempDir()
	ws.ResolveDataDir = func() (string, error) { return dataDir, nil }
	order, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ExportFactoryOrderFile(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	trackingID := lines[0].TrackingID

	// A rouzao-shaped return CSV pointing at the real tracking id.
	path := writeTempFixture(t, "return.csv", "\uFEFF\"订单编号\",\"下单时间\",\"商品编码\",\"商品名称\",\"规格&数量\",\"应付金额\",\"收件人\",\"电话\",\"收件信息\",\"物流公司\",\"物流单号\",\"打印快递时间\",\"订单状态\"\n\""+trackingID+"\n\",\"2026-05-01 10:00:00\",\"SKU\",\"Name\",\"111_x * 1|222_y * 2\",\"0.00\",\"Alice\",\"13800000000\",\" addr \",\"申通快递\",\"'YT888999000\",\"2026-05-10 09:17:20\",\"运输中\"\n")
	result, err := ws.ImportShipmentFile(ctx, p.factory.ID, returnTpl.ID, path)
	if err != nil {
		t.Fatalf("ImportShipmentFile: %v", err)
	}
	if result.Imported != 1 || len(result.Skipped) != 0 {
		t.Fatalf("imported = %d skipped = %v", result.Imported, result.Skipped)
	}
	sh := result.Shipments[0]
	if sh.TrackingID != trackingID || sh.TrackingNo != "YT888999000" {
		t.Fatalf("shipment = %+v", sh)
	}
	// The 规格&数量 blob "111_x * 1|222_y * 2" must land as the summed
	// parcel quantity, not as a zero from a failed Atoi.
	if sh.Quantity != 3 {
		t.Fatalf("shipment quantity = %d, want 3", sh.Quantity)
	}
	// The carrier stays as the factory file spelled it; no code is resolved
	// on the factory side (translation belongs to the source platform's
	// writeback).
	if sh.CarrierName != "申通快递" || sh.CarrierCode != "" {
		t.Fatalf("carrier = %q/%q, want name kept and empty code", sh.CarrierName, sh.CarrierCode)
	}
	if sh.ShippedAt == nil || sh.ShippedAt.Format("2006-01-02") != "2026-05-10" {
		t.Fatalf("shippedAt = %v, want the printed courier time", sh.ShippedAt)
	}

	// Re-import the same parcel: skipped as duplicate, nothing new lands.
	again, err := ws.ImportShipmentFile(ctx, p.factory.ID, returnTpl.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Imported != 0 || len(again.Skipped) != 1 || again.Skipped[0].Reason != "shipment already imported" {
		t.Fatalf("re-import = %+v", again)
	}

	// The synthetic fixture's order numbers are not our tracking ids.
	fixturePath := filepath.Join("..", "..", "testdata", "integration_profile", "rouzao_shipment_return.csv")
	fixtureResult, err := ws.ImportShipmentFile(ctx, p.factory.ID, returnTpl.ID, fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if fixtureResult.Imported != 0 || len(fixtureResult.Skipped) != 3 {
		t.Fatalf("fixture import = %+v", fixtureResult)
	}
	for _, s := range fixtureResult.Skipped {
		if !strings.Contains(s.Reason, "tracking") {
			t.Fatalf("skip reason = %q", s.Reason)
		}
	}
}

// TestImportShipmentFile_TemplateChecks pins the template-explicit contract:
// the template must be an active shipment_return input template owned by the
// factory platform.
func TestImportShipmentFile_TemplateChecks(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-12")
	ws, ctx := p.ws, p.ctx
	seedBuiltins(t, ws)
	path := writeTempFixture(t, "return.csv", "订单编号,物流单号\nX,Y\n")

	builtin := builtinTemplate(t, ws, p.factory.ID, DocumentTypeShipmentReturn)
	if _, err := ws.ImportShipmentFile(ctx, p.factory.ID, builtin.ID, path); !errors.Is(err, ErrBuiltinTemplate) {
		t.Fatalf("builtin template err = %v, want ErrBuiltinTemplate", err)
	}
	factoryOrder := cloneBuiltinTemplate(t, ws, p.factory.ID, DocumentTypeFactoryOrder)
	if _, err := ws.ImportShipmentFile(ctx, p.factory.ID, factoryOrder.ID, path); !errors.Is(err, ErrTemplateMismatch) {
		t.Fatalf("output template err = %v, want ErrTemplateMismatch", err)
	}
	membership := cloneBuiltinTemplate(t, ws, p.source.ID, DocumentTypeMembershipList)
	if _, err := ws.ImportShipmentFile(ctx, p.factory.ID, membership.ID, path); !errors.Is(err, ErrTemplateMismatch) {
		t.Fatalf("foreign platform template err = %v, want ErrTemplateMismatch", err)
	}
	if _, err := ws.ImportShipmentFile(ctx, p.factory.ID, 99999, path); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing template err = %v, want ErrNotFound", err)
	}
}

func TestBundleExpansion_OnAssignment(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-BUNDLE")
	ws, ctx := p.ws, p.ctx

	// Turn the alias into a bundle: the parent product result is replaced by
	// two component results.
	alias, err := ws.Store.FindAlias(ctx, p.source.ID, "BILI-SKU-BUNDLE")
	if err != nil {
		t.Fatal(err)
	}
	comp2 := domain.ProductItem{Name: "Standee", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-STANDEE-001"}
	comp3 := domain.ProductItem{Name: "Magnet", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MAGNET-001"}
	if err := ws.CreateProduct(ctx, &comp2); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateProduct(ctx, &comp3); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateBundleComponent(ctx, &domain.ProductBundleComponent{AliasID: alias.ID, ProductItemID: comp2.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateBundleComponent(ctx, &domain.ProductBundleComponent{AliasID: alias.ID, ProductItemID: comp3.ID, Quantity: 2}); err != nil {
		t.Fatal(err)
	}

	// A second line on the same alias, assigned fresh.
	doc2, _, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: p.source.ID, DocumentType: DocumentTypeOrderExport}, []IngestFactInput{{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "ORD-BUNDLE-1",
		IdentityValue:    "uid-778",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "BILI-SKU-BUNDLE", Quantity: 3}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	facts2, _ := ws.Store.ListFactsByDocument(ctx, doc2.ID)
	lines2, _ := ws.Store.ListFactLines(ctx, facts2[0].ID)
	if err := ws.AssignLines(ctx, p.wave.ID, []uint{lines2[0].ID}); err != nil {
		t.Fatal(err)
	}

	results, err := ws.Store.ListResults(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	qtyByProduct := map[uint]int{}
	for _, r := range results {
		if r.InputFactLineID != nil && *r.InputFactLineID == lines2[0].ID {
			if r.ProductItemID == nil {
				t.Fatal("bundle result must be aligned")
			}
			qtyByProduct[*r.ProductItemID] += r.Quantity
			if !strings.Contains(r.ExtraData, bundleAliasLineMarker) {
				t.Fatalf("bundle marker missing: %q", r.ExtraData)
			}
		}
	}
	// 3 x 1 standee and 3 x 2 magnets, and no parent-product result.
	if qtyByProduct[comp2.ID] != 3 || qtyByProduct[comp3.ID] != 6 || qtyByProduct[p.product.ID] != 0 {
		t.Fatalf("bundle quantities = %v", qtyByProduct)
	}
}

func TestUpdateAlias_RealignsLinesAndResults(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-REALIGN")
	ws, ctx := p.ws, p.ctx

	alias, err := ws.Store.FindAlias(ctx, p.source.ID, "BILI-SKU-REALIGN")
	if err != nil {
		t.Fatal(err)
	}
	results, err := ws.Store.ListResults(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ProductItemID == nil || *results[0].ProductItemID != p.product.ID {
		t.Fatalf("pre-update results = %+v", results)
	}

	newProduct := domain.ProductItem{Name: "Medal v2", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MEDAL-002"}
	if err := ws.CreateProduct(ctx, &newProduct); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpdateAlias(ctx, alias.ID, newProduct.ID); err != nil {
		t.Fatalf("UpdateAlias: %v", err)
	}

	line, err := ws.Store.GetFactLine(ctx, p.line.ID)
	if err != nil {
		t.Fatal(err)
	}
	if line.ProductItemID == nil || *line.ProductItemID != newProduct.ID {
		t.Fatalf("line product = %v", line.ProductItemID)
	}
	results, err = ws.Store.ListResults(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].ProductItemID == nil || *results[0].ProductItemID != newProduct.ID {
		t.Fatalf("result product = %v", results[0].ProductItemID)
	}

	// Frozen results resist re-alignment: regenerate the path and re-point.
	order, _, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	third := domain.ProductItem{Name: "Medal v3", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MEDAL-003"}
	if err := ws.CreateProduct(ctx, &third); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpdateAlias(ctx, alias.ID, third.ID); err != nil {
		t.Fatal(err)
	}
	frozenResults, err := ws.Store.ListResults(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range frozenResults {
		if r.Frozen && (r.ProductItemID == nil || *r.ProductItemID != newProduct.ID) {
			t.Fatalf("frozen result must keep the aligned product, got %v", r.ProductItemID)
		}
	}
	_ = order
}

func TestUpdateAlias_RebuildsBundleResults(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-REBUNDLE")
	ws, ctx := p.ws, p.ctx
	alias, err := ws.Store.FindAlias(ctx, p.source.ID, "BILI-SKU-REBUNDLE")
	if err != nil {
		t.Fatal(err)
	}
	comp := domain.ProductItem{Name: "Standee", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-STANDEE-009"}
	if err := ws.CreateProduct(ctx, &comp); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateBundleComponent(ctx, &domain.ProductBundleComponent{AliasID: alias.ID, ProductItemID: comp.ID, Quantity: 1}); err != nil {
		t.Fatal(err)
	}

	// Re-run assignment to expand the now-bundled alias: recreate by
	// ingesting another line and assigning it.
	doc2, _, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: p.source.ID}, []IngestFactInput{{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "ORD-REBUNDLE",
		IdentityValue:    "uid-779",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "BILI-SKU-REBUNDLE", Quantity: 1}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	facts2, _ := ws.Store.ListFactsByDocument(ctx, doc2.ID)
	lines2, _ := ws.Store.ListFactLines(ctx, facts2[0].ID)
	if err := ws.AssignLines(ctx, p.wave.ID, []uint{lines2[0].ID}); err != nil {
		t.Fatal(err)
	}

	// Growing the bundle and re-pointing the alias rebuilds the expansion.
	comp2 := domain.ProductItem{Name: "Magnet", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MAGNET-009"}
	if err := ws.CreateProduct(ctx, &comp2); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateBundleComponent(ctx, &domain.ProductBundleComponent{AliasID: alias.ID, ProductItemID: comp2.ID, Quantity: 5}); err != nil {
		t.Fatal(err)
	}
	repoint := domain.ProductItem{Name: "Medal v2", FactoryPlatformID: p.factory.ID, FactorySKU: "ROZAO-MEDAL-010"}
	if err := ws.CreateProduct(ctx, &repoint); err != nil {
		t.Fatal(err)
	}
	if err := ws.UpdateAlias(ctx, alias.ID, repoint.ID); err != nil {
		t.Fatalf("UpdateAlias: %v", err)
	}
	results, err := ws.Store.ListResults(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[uint]int{}
	for _, r := range results {
		if r.InputFactLineID != nil && *r.InputFactLineID == lines2[0].ID && r.ProductItemID != nil {
			counts[*r.ProductItemID]++
		}
	}
	if counts[comp.ID] != 1 || counts[comp2.ID] != 1 {
		t.Fatalf("rebuilt bundle results = %v", counts)
	}
}

func TestPreviewTemplate_RunsAgainstSampleFile(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)
	tpl := &domain.TemplateConfig{
		PlatformID:   source.ID,
		DocumentType: DocumentTypeMembershipList,
		Direction:    string(domain.TemplateDirectionInput),
		Name:         "membership preview",
		MappingJSON: mustSerializeMapping(t, alignment.MappingConfig{
			Mode:      alignment.ModePositional,
			Positions: map[string]int{"membership.level": 0, "identity.value": 1},
			Required:  []string{"identity.value"},
		}),
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_membership_positional.csv")
	preview, err := ws.PreviewTemplate(ctx, tpl.ID, path, 2)
	if err != nil {
		t.Fatalf("PreviewTemplate: %v", err)
	}
	if len(preview.Rows) != 2 || preview.Rows[0].Values["membership.level"] != "总督" {
		t.Fatalf("preview = %+v", preview.Rows)
	}
	if preview.Issues == nil {
		t.Fatal("issues must be non-nil")
	}
	if preview.TotalRows != 3 || preview.DroppedRows != 0 {
		t.Fatalf("totals = %d/%d, want 3 produced, 0 dropped", preview.TotalRows, preview.DroppedRows)
	}

	// Built-in templates may be previewed before copying.
	seedBuiltins(t, ws)
	builtin := builtinTemplate(t, ws, source.ID, DocumentTypeMembershipList)
	bp, err := ws.PreviewTemplate(ctx, builtin.ID, path, 0)
	if err != nil {
		t.Fatalf("PreviewTemplate builtin: %v", err)
	}
	if len(bp.Rows) != 3 || bp.Rows[2].Values["customer.display_name"] != "DisplayC" || bp.Rows[2].Fingerprint == "" {
		t.Fatalf("builtin preview = %+v", bp.Rows)
	}
}

func TestPreviewMapping_UnsavedConfigAgainstSample(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	path := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_membership_positional.csv")
	mapping := mustSerializeMapping(t, alignment.MappingConfig{
		Mode:        alignment.ModePositional,
		Positions:   map[string]int{"membership.level": 0, "identity.value": 1, "customer.display_name": 2},
		Required:    []string{"identity.value"},
		Fingerprint: []string{"identity.value"},
	})
	preview, err := ws.PreviewMapping(ctx, mapping, DocumentTypeMembershipList, path, 2)
	if err != nil {
		t.Fatalf("PreviewMapping: %v", err)
	}
	if len(preview.Rows) != 2 || preview.TotalRows != 3 || preview.DroppedRows != 0 {
		t.Fatalf("preview = %+v", preview)
	}
	row := preview.Rows[1]
	if row.LineNo != 2 || row.SourceRow != 2 || row.Values["identity.value"] != "uid-10002" {
		t.Fatalf("row = %+v", row)
	}
	if row.Fingerprint != alignment.FingerprintValues(map[string]string{"identity.value": "uid-10002"}, []string{"identity.value"}) {
		t.Fatalf("fingerprint = %q, want the identity-only hash", row.Fingerprint)
	}

	// A required key on a column that is empty drops rows and counts them.
	strict := mustSerializeMapping(t, alignment.MappingConfig{
		Mode:      alignment.ModePositional,
		Positions: map[string]int{"identity.value": 1, "recipient.phone": 7},
		Required:  []string{"recipient.phone"},
	})
	dropped, err := ws.PreviewMapping(ctx, strict, DocumentTypeMembershipList, path, 10)
	if err != nil {
		t.Fatalf("PreviewMapping strict: %v", err)
	}
	if len(dropped.Rows) != 0 || dropped.TotalRows != 0 || dropped.DroppedRows != 3 || len(dropped.Issues) != 3 {
		t.Fatalf("dropped preview = %+v", dropped)
	}
	if dropped.Rows == nil {
		t.Fatal("rows must be non-nil for JSON transport")
	}

	if _, err := ws.PreviewMapping(ctx, "", DocumentTypeMembershipList, path, 2); err == nil {
		t.Fatal("empty mapping must fail")
	}
}

func TestInspectSampleFile_CSVAndXLSX(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()

	csvPath := filepath.Join("..", "..", "testdata", "integration_profile", "rouzao_shipment_return.csv")
	info, err := ws.InspectSampleFile(ctx, csvPath, "", 2)
	if err != nil {
		t.Fatalf("InspectSampleFile csv: %v", err)
	}
	if info.Format != alignment.FormatCSV || len(info.Sheets) != 0 || info.Sheets == nil {
		t.Fatalf("csv info = %+v", info)
	}
	if info.Total != 4 || len(info.Records) != 2 {
		t.Fatalf("csv total/records = %d/%d, want 4/2", info.Total, len(info.Records))
	}
	if len(info.Records[0]) != 13 || info.Records[0][0] != "订单编号" || info.Records[0][12] != "订单状态" {
		t.Fatalf("csv header = %q", info.Records[0])
	}

	// An xlsx with two sheets: the second sheet is selectable by name.
	f := excelize.NewFile()
	first := f.GetSheetName(0)
	if err := f.SetSheetRow(first, "A1", &[]string{"h1", "h2"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow(first, "A2", &[]string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet("Second"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Second", "A1", &[]string{"only"}); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	xlsxPath := writeTempFixture(t, "sample.xlsx", buf.String())

	xinfo, err := ws.InspectSampleFile(ctx, xlsxPath, "", 0)
	if err != nil {
		t.Fatalf("InspectSampleFile xlsx: %v", err)
	}
	if xinfo.Format != alignment.FormatXLSX || len(xinfo.Sheets) != 2 || xinfo.Sheets[0] != first || xinfo.Sheets[1] != "Second" {
		t.Fatalf("xlsx info = %+v", xinfo)
	}
	if xinfo.Total != 2 || len(xinfo.Records) != 2 || xinfo.Records[0][0] != "h1" {
		t.Fatalf("xlsx first sheet = %+v", xinfo.Records)
	}
	second, err := ws.InspectSampleFile(ctx, xlsxPath, "Second", 0)
	if err != nil {
		t.Fatalf("InspectSampleFile sheet: %v", err)
	}
	if second.Total != 1 || second.Records[0][0] != "only" {
		t.Fatalf("second sheet = %+v", second.Records)
	}
	if _, err := ws.InspectSampleFile(ctx, xlsxPath, "Missing", 0); err == nil {
		t.Fatal("unknown sheet must fail")
	}
	if _, err := ws.InspectSampleFile(ctx, csvPath+".txt", "", 0); err == nil {
		t.Fatal("unsupported extension must fail")
	}
}

func TestImportFile_RejectsForeignAndBuiltinTemplates(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	seedBuiltins(t, ws)
	source := platformByKind(t, ws, domain.PlatformKindSource)
	factory := platformByKind(t, ws, domain.PlatformKindFactory)
	tpl := &domain.TemplateConfig{
		PlatformID:   source.ID,
		DocumentType: DocumentTypeMembershipList,
		Direction:    string(domain.TemplateDirectionInput),
		Name:         "source template",
		MappingJSON: mustSerializeMapping(t, alignment.MappingConfig{
			Mode:      alignment.ModePositional,
			Positions: map[string]int{"identity.value": 0},
		}),
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	path := writeTempFixture(t, "x.csv", "a\n1\n")
	if _, err := ws.ImportFile(ctx, factory.ID, tpl.ID, path); !errors.Is(err, ErrTemplateMismatch) {
		t.Fatalf("platform mismatch err = %v, want ErrTemplateMismatch", err)
	}
	if _, err := ws.ImportFile(ctx, source.ID, tpl.ID, path+"-missing"); err == nil {
		t.Fatal("expected read error")
	}
	builtin := builtinTemplate(t, ws, source.ID, DocumentTypeMembershipList)
	if _, err := ws.ImportFile(ctx, source.ID, builtin.ID, path); !errors.Is(err, ErrBuiltinTemplate) {
		t.Fatalf("builtin import err = %v, want ErrBuiltinTemplate", err)
	}
	writeback := cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeWriteback)
	if _, err := ws.ImportFile(ctx, source.ID, writeback.ID, path); !errors.Is(err, ErrTemplateMismatch) {
		t.Fatalf("output template import err = %v, want ErrTemplateMismatch", err)
	}
	// Shipment returns are input templates too, but they feed
	// ImportShipmentFile, not fact ingestion.
	returns := cloneBuiltinTemplate(t, ws, factory.ID, DocumentTypeShipmentReturn)
	if _, err := ws.ImportFile(ctx, factory.ID, returns.ID, path); !errors.Is(err, ErrTemplateMismatch) {
		t.Fatalf("shipment_return via ImportFile err = %v, want ErrTemplateMismatch", err)
	}
}

// TestExportFactoryOrderFile_WriteFailureKeepsOrderGenerated pins the
// transaction boundary: the file write happens inside the transaction, so a
// failed write rolls the order back to generated instead of leaving an
// exported order without its file.
func TestExportFactoryOrderFile_WriteFailureKeepsOrderGenerated(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-EXPORTFAIL")
	ws, ctx := p.ws, p.ctx
	ws.ResolveDataDir = func() (string, error) { return "", fmt.Errorf("data dir unavailable") }
	order, _, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := ws.ExportFactoryOrderFile(ctx, order.ID); err == nil {
		t.Fatal("expected export to fail when the file cannot be written")
	}
	after, err := ws.Store.GetSupplierOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != string(domain.SupplierOrderGenerated) || after.ExportedAt != nil || after.ExportPayload != "" {
		t.Fatalf("order = %+v, want still generated with no export payload", after)
	}
}

// TestImportFile_XlsxRawPayloadBase64Prefix checks that imported xlsx
// originals are stored base64-prefixed (mirroring the export side) instead of
// raw binary bytes in a text column.
func TestImportFile_XlsxRawPayloadBase64Prefix(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)

	tpl := &domain.TemplateConfig{
		PlatformID:   source.ID,
		DocumentType: DocumentTypeMembershipList,
		Direction:    string(domain.TemplateDirectionInput),
		Name:         "bilibili membership xlsx",
		Version:      1,
		MappingJSON: mustSerializeMapping(t, alignment.MappingConfig{
			Mode: alignment.ModeHeader,
			Columns: map[string]string{
				"membership.level":      "等级",
				"identity.value":        "UID",
				"customer.display_name": "昵称",
			},
			Required:    []string{"identity.value"},
			Fingerprint: []string{"identity.value"},
		}),
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}

	xlsxBytes, err := alignment.Render([]map[string]string{
		{"membership.level": "captain", "identity.value": "uid-x1", "customer.display_name": "Xena"},
	}, alignment.LayoutConfig{
		Format:      alignment.FormatXLSX,
		ColumnOrder: []string{"membership.level", "identity.value", "customer.display_name"},
		HeaderNames: map[string]string{"membership.level": "等级", "identity.value": "UID", "customer.display_name": "昵称"},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := writeTempFixture(t, "members.xlsx", string(xlsxBytes))

	result, err := ws.ImportFile(ctx, source.ID, tpl.ID, path)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if result.FactsCreated != 1 {
		t.Fatalf("facts created = %d, want 1", result.FactsCreated)
	}
	stored, err := ws.Store.GetDocument(ctx, result.Document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored.RawPayload, "base64:") {
		t.Fatalf("raw payload prefix = %q, want base64:", stored.RawPayload[:16])
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored.RawPayload, "base64:"))
	if err != nil {
		t.Fatalf("stored payload is not valid base64: %v", err)
	}
	if !bytes.Equal(decoded, xlsxBytes) {
		t.Fatal("decoded audit copy must round-trip to the original file bytes")
	}
}
