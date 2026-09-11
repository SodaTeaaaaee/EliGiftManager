package app

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// csvWritebackTemplate creates an active CSV writeback template on the source
// platform (order no, carrier code, tracking no) so payload assertions can
// read plain text.
func csvWritebackTemplate(t *testing.T, p *readyPath) *domain.TemplateConfig {
	t.Helper()
	layoutRaw, err := alignment.SerializeLayoutConfig(alignment.LayoutConfig{
		Format:      alignment.FormatCSV,
		ColumnOrder: []string{"source.document_no", "shipment.carrier_code", "shipment.tracking_no"},
		HeaderNames: map[string]string{"source.document_no": "订单号", "shipment.carrier_code": "快递公司编码", "shipment.tracking_no": "运单号"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tpl := &domain.TemplateConfig{
		PlatformID:   p.source.ID,
		DocumentType: DocumentTypeWriteback,
		Direction:    string(domain.TemplateDirectionOutput),
		Name:         "bilibili writeback csv",
		LayoutJSON:   layoutRaw,
	}
	if err := p.ws.CreateTemplate(p.ctx, tpl); err != nil {
		t.Fatalf("CreateTemplate writeback: %v", err)
	}
	return tpl
}

// shippedRetailFact drives one retail fact through factory order, shipment,
// and writeback generation, returning the pending writeback items.
func shippedRetailFact(t *testing.T, sku string) (*readyPath, []domain.ChannelWritebackItem) {
	t.Helper()
	p := setupReadyRetailPath(t, sku)
	csvWritebackTemplate(t, p)
	_, lines, err := p.ws.GenerateFactoryOrder(p.ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := p.ws.ImportShipment(p.ctx, lines[0].TrackingID, "SF-WB-0001", "", "顺丰速运", 2); err != nil {
		t.Fatalf("ImportShipment: %v", err)
	}
	items, err := p.ws.GenerateWritebacks(p.ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("writeback items = %d, want 1", len(items))
	}
	return p, items
}

// TestGenerateWritebacks_NoActiveTemplateFails pins the removed fallback: a
// source platform without an active writeback template cannot generate
// writeback items, and a built-in template alone does not count.
func TestGenerateWritebacks_NoActiveTemplateFails(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB0")
	ws, ctx := p.ws, p.ctx
	seedBuiltins(t, ws)
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "SF-WB-0000", "", "顺丰速运", 2); err != nil {
		t.Fatalf("ImportShipment: %v", err)
	}
	_, err = ws.GenerateWritebacks(ctx, p.fact.ID)
	if !errors.Is(err, ErrNoActiveTemplate) {
		t.Fatalf("GenerateWritebacks err = %v, want ErrNoActiveTemplate", err)
	}
	if !strings.Contains(err.Error(), p.source.Name) || !strings.Contains(err.Error(), DocumentTypeWriteback) {
		t.Fatalf("error must name the platform and document type: %v", err)
	}
	stored, err := ws.Store.ListWritebacksByFact(ctx, p.fact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 0 {
		t.Fatalf("no writeback item may land without a template, got %d", len(stored))
	}

	// Copying the built-in bilibili layout makes generation work and renders
	// bilibili's three tracking-import columns as xlsx.
	tpl := cloneBuiltinTemplate(t, ws, p.source.ID, DocumentTypeWriteback)
	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks with clone: %v", err)
	}
	if len(items) != 1 || items[0].TemplateID != tpl.ID || items[0].TemplateVersion != 1 {
		t.Fatalf("items = %+v, want one item snapshotting template %d v1", items, tpl.ID)
	}
	payload, format, err := writebackPayloadBytes(items[0].Payload)
	if err != nil || format != alignment.FormatXLSX {
		t.Fatalf("payload format = %q err = %v, want xlsx", format, err)
	}
	records, err := alignment.ReadRows(payload, alignment.FormatXLSX, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0][0] != "订单号*" || records[0][2] != "物流单号*" || !strings.HasPrefix(records[0][1], "快递公司编码*") {
		t.Fatalf("builtin writeback header = %q", records[0])
	}
	if records[1][0] != "ORD-EXPORT-1" || records[1][2] != "SF-WB-0000" {
		t.Fatalf("builtin writeback row = %q", records[1])
	}
}

// TestMarkWritebackFailed_MakesWorkStateAndHomeReachable pins the whole
// writeback_failed chain: after a failed mark the home bucket counts the item
// and the linked result views report WorkStateWritebackFailed, the states that
// were previously unreachable because nothing ever wrote the failed status.
func TestMarkWritebackFailed_MakesWorkStateAndHomeReachable(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB1")
	ws, ctx := p.ws, p.ctx

	if err := ws.MarkWritebackFailed(ctx, items[0].ID, "bilibili rejected: order locked"); err != nil {
		t.Fatalf("MarkWritebackFailed: %v", err)
	}

	home, err := ws.Home(ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.WritebackFailed != 1 {
		t.Fatalf("home WritebackFailed = %d, want 1", home.WritebackFailed)
	}

	views, err := ws.ListResultViews(ctx, p.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	found := false
	for _, v := range views {
		if v.Result.InputFactID != nil && *v.Result.InputFactID == p.fact.ID {
			if v.WorkState != domain.WorkStateWritebackFailed {
				t.Fatalf("retail result work state = %s, want writeback_failed", v.WorkState)
			}
			if !v.WritebackFailed {
				t.Fatal("expected WritebackFailed flag on the result view")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected a view for the retail fact")
	}
}

// TestWritebackStateTransitions covers the retry-history contract: failures
// advance RetryCount and store (truncated) error text, a success clears the
// error but keeps the counter, and the result view falls back to shipped once
// every writeback item is no longer failed.
func TestWritebackStateTransitions(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB2")
	ws, ctx := p.ws, p.ctx

	if err := ws.MarkWritebackFailed(ctx, items[0].ID, "attempt 1 failed"); err != nil {
		t.Fatalf("MarkWritebackFailed 1: %v", err)
	}
	long := strings.Repeat("错", 600) + "tail"
	if err := ws.MarkWritebackFailed(ctx, items[0].ID, long); err != nil {
		t.Fatalf("MarkWritebackFailed 2: %v", err)
	}
	stored, err := ws.Store.GetWriteback(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("GetWriteback: %v", err)
	}
	if stored.Status != string(domain.WritebackFailed) {
		t.Fatalf("status = %s, want failed", stored.Status)
	}
	if stored.RetryCount != 2 {
		t.Fatalf("retry count = %d, want 2", stored.RetryCount)
	}
	got := []rune(stored.ErrorMessage)
	if len(got) != 500 {
		t.Fatalf("error message rune count = %d, want 500 (truncated)", len(got))
	}

	if err := ws.MarkWritebackSent(ctx, items[0].ID); err != nil {
		t.Fatalf("MarkWritebackSent: %v", err)
	}
	sent, err := ws.Store.GetWriteback(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if sent.Status != string(domain.WritebackSent) {
		t.Fatalf("status = %s, want sent", sent.Status)
	}
	if sent.ErrorMessage != "" {
		t.Fatalf("error message = %q, want cleared after sent", sent.ErrorMessage)
	}
	if sent.RetryCount != 2 {
		t.Fatalf("retry count = %d, want 2 kept after success", sent.RetryCount)
	}
	// The parcel shipped, so once no writeback is failed the result state is
	// shipped again rather than writeback_failed.
	views, err := ws.ListResultViews(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.Result.InputFactID != nil && *v.Result.InputFactID == p.fact.ID {
			if v.WorkState != domain.WorkStateShipped {
				t.Fatalf("retail result work state = %s, want shipped after sent", v.WorkState)
			}
		}
	}
	home, err := ws.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if home.WritebackFailed != 0 {
		t.Fatalf("home WritebackFailed = %d, want 0 after sent", home.WritebackFailed)
	}
}

// TestGenerateWritebacks_NewParcelsAppendAndRerunIsIdempotent pins the
// per-parcel idempotency: the first run records the first parcel, a second
// parcel arrives later and only it gets a new item, and re-running never
// duplicates anything. The old behavior bailed out on the first existing item
// and could never record the new parcel.
func TestGenerateWritebacks_NewParcelsAppendAndRerunIsIdempotent(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB3")
	ws, ctx := p.ws, p.ctx
	tpl := csvWritebackTemplate(t, p)
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	tracking := lines[0].TrackingID
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-0001", "", "顺丰速运", 2); err != nil {
		t.Fatalf("ImportShipment 1: %v", err)
	}

	first, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks 1: %v", err)
	}
	if len(first) != 1 || first[0].Quantity != 2 {
		t.Fatalf("first run = %+v, want 1 item with quantity 2", first)
	}
	if first[0].TemplateID != tpl.ID || first[0].TemplateVersion != 1 {
		t.Fatalf("template snapshot = %d/%d, want %d/1", first[0].TemplateID, first[0].TemplateVersion, tpl.ID)
	}

	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-0002", "", "顺丰速运", 3); err != nil {
		t.Fatalf("ImportShipment 2: %v", err)
	}
	second, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks 2: %v", err)
	}
	if len(second) != 2 {
		t.Fatalf("second run items = %d, want 2 (new parcel appended)", len(second))
	}
	if second[0].ID != first[0].ID {
		t.Fatalf("first item id changed: %d then %d", first[0].ID, second[0].ID)
	}
	if second[1].Quantity != 3 {
		t.Fatalf("new item quantity = %d, want 3", second[1].Quantity)
	}

	again, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks rerun: %v", err)
	}
	if len(again) != 2 || again[0].ID != second[0].ID || again[1].ID != second[1].ID {
		t.Fatalf("rerun = %+v, want the same 2 items unchanged", again)
	}
}

// TestGenerateWritebacks_PayloadCarriesExternalCarrierCode checks the rendered
// payload: the source fact's stable external id as the order number, the
// carrier translated by name through the source platform's mappings into the
// platform's carrier id, and a conservative empty cell when no mapping exists.
func TestGenerateWritebacks_PayloadCarriesExternalCarrierCode(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB4")
	ws, ctx := p.ws, p.ctx
	csvWritebackTemplate(t, p)
	// The platform lists the short name; the factory file spells the long one.
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: p.source.ID, ExternalCode: "shunfeng", InternalName: "顺丰"}); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	tracking := lines[0].TrackingID
	// First parcel carries a mappable carrier name, second one a carrier the
	// source platform has no mapping for. Neither carries an internal code.
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-MAPPED", "", "顺丰速运", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-UNMAPPED", "", "未知快递", 1); err != nil {
		t.Fatal(err)
	}

	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	// setupReadyRetailPath sets the fact's stable external id to ORD-EXPORT-N.
	mapped, unmapped := items[0].Payload, items[1].Payload
	if !strings.Contains(mapped, "订单号,快递公司编码,运单号\n") {
		t.Fatalf("header missing from payload: %q", mapped)
	}
	if !strings.Contains(mapped, "ORD-EXPORT-1,shunfeng,SF-WB-MAPPED\n") {
		t.Fatalf("mapped payload = %q, want order no + platform carrier id", mapped)
	}
	if items[0].CarrierCode != "shunfeng" {
		t.Fatalf("item carrier code = %q, want the translated platform id", items[0].CarrierCode)
	}
	if !strings.Contains(unmapped, "ORD-EXPORT-1,,SF-WB-UNMAPPED\n") {
		t.Fatalf("unmapped payload = %q, want conservative empty carrier cell", unmapped)
	}
	if items[1].CarrierCode != "" {
		t.Fatalf("unmapped item carrier code = %q, want empty", items[1].CarrierCode)
	}
	if strings.HasPrefix(mapped, "\uFEFF") != true {
		t.Fatal("payload should keep the UTF-8 BOM like every rendered CSV")
	}
}

// TestMatchCarrierCode pins the loose description-to-id matching: suffixes
// and case fold away, containment either way counts, and disagreeing
// candidates yield nothing.
func TestMatchCarrierCode(t *testing.T) {
	mappings := []domain.CarrierMapping{
		{InternalName: "申通", ExternalCode: "shentong"},
		{InternalName: "韵达快递", ExternalCode: "yunda"},
		{InternalName: "中通", ExternalCode: "zhongtong"},
		{InternalName: "中通快运", ExternalCode: "zhongtongkuaiyun"},
		{InternalName: "EMS", ExternalCode: "ems"},
		{InternalName: "无编码", ExternalCode: ""},
	}
	cases := []struct {
		name      string
		code      string
		ambiguous bool
	}{
		{"申通快递", "shentong", false},
		{" 申通 ", "shentong", false},
		{"韵达", "yunda", false},
		{"韵达速递", "yunda", false},
		{"ems", "ems", false},
		{"中通", "zhongtong", false},          // exact beats the containment candidate
		{"中通快运", "zhongtongkuaiyun", false}, // exact match on the longer name
		{"中通快递", "zhongtong", false},        // normalizes to 中通, exact
		{"无编码", "", false},                  // mapping without a code never matches
		{"顺丰速运", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		code, ambiguous := matchCarrierCode(mappings, c.name)
		if code != c.code || ambiguous != c.ambiguous {
			t.Fatalf("matchCarrierCode(%q) = %q/%v, want %q/%v", c.name, code, ambiguous, c.code, c.ambiguous)
		}
	}
	// Two containment candidates with different codes are ambiguous.
	amb := []domain.CarrierMapping{
		{InternalName: "顺丰", ExternalCode: "sf"},
		{InternalName: "丰网", ExternalCode: "fw"},
	}
	if code, ambiguous := matchCarrierCode(amb, "顺丰丰网速运"); code != "" || !ambiguous {
		t.Fatalf("ambiguous match = %q/%v, want empty and ambiguous", code, ambiguous)
	}
	// Same code twice is not ambiguous.
	same := []domain.CarrierMapping{
		{InternalName: "顺丰", ExternalCode: "sf"},
		{InternalName: "顺丰速运", ExternalCode: "sf"},
	}
	if code, ambiguous := matchCarrierCode(same, "顺丰快递"); code != "sf" || ambiguous {
		t.Fatalf("agreeing candidates = %q/%v, want sf", code, ambiguous)
	}
}

// TestExportWritebackFile_WritesStoredPayload pins the export step: the file
// lands under exports/writebacks and its bytes equal the payload snapshotted
// at generation time (CSV verbatim, xlsx base64-decoded).
func TestExportWritebackFile_WritesStoredPayload(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB6")
	ws, ctx := p.ws, p.ctx
	ws.ResolveDataDir = func() (string, error) { return t.TempDir(), nil }

	result, err := ws.ExportWritebackFile(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("ExportWritebackFile: %v", err)
	}
	if !strings.Contains(result.Path, filepath.Join("exports", "writebacks")) || !strings.HasSuffix(result.Path, ".csv") {
		t.Fatalf("path = %q", result.Path)
	}
	raw, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if string(raw) != items[0].Payload {
		t.Fatalf("file content = %q, want stored payload %q", raw, items[0].Payload)
	}
}

// TestExportWritebackFile_XlsxPayloadDecodes checks the binary branch: an
// xlsx writeback template stores the payload base64-prefixed, and the export
// decodes it back into .xlsx bytes.
func TestExportWritebackFile_XlsxPayloadDecodes(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB7")
	ws, ctx := p.ws, p.ctx
	ws.ResolveDataDir = func() (string, error) { return t.TempDir(), nil }
	layoutRaw, err := alignment.SerializeLayoutConfig(alignment.LayoutConfig{
		Format:      alignment.FormatXLSX,
		ColumnOrder: []string{"source.document_no"},
		HeaderNames: map[string]string{"source.document_no": "订单号"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tpl := &domain.TemplateConfig{
		PlatformID:   p.source.ID,
		DocumentType: DocumentTypeWriteback,
		Direction:    string(domain.TemplateDirectionOutput),
		Name:         "bilibili writeback xlsx",
		LayoutJSON:   layoutRaw,
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "SF-WB-XLSX", "", "顺丰速运", 1); err != nil {
		t.Fatalf("ImportShipment: %v", err)
	}
	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if !strings.HasPrefix(items[0].Payload, "base64:") {
		t.Fatalf("xlsx payload should be base64-prefixed, got %q", items[0].Payload[:32])
	}

	result, err := ws.ExportWritebackFile(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("ExportWritebackFile: %v", err)
	}
	if !strings.HasSuffix(result.Path, ".xlsx") {
		t.Fatalf("path = %q, want .xlsx", result.Path)
	}
	raw, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	want, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(items[0].Payload, "base64:"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(want) {
		t.Fatal("file content must equal the decoded stored payload")
	}
}

// TestListWritebacksByWave_FollowsFactLineMembership pins the wave listing:
// items surface through the wave's fact lines, not through results, and a
// different wave's facts stay out.
func TestListWritebacksByWave_FollowsFactLineMembership(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB8")
	ws, ctx := p.ws, p.ctx

	other, err := ws.CreateWave(ctx, "other", "")
	if err != nil {
		t.Fatal(err)
	}

	listed, err := ws.ListWritebacksByWave(ctx, p.wave.ID)
	if err != nil {
		t.Fatalf("ListWritebacksByWave: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != items[0].ID {
		t.Fatalf("listed = %+v, want the wave's item %d", listed, items[0].ID)
	}

	empty, err := ws.ListWritebacksByWave(ctx, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("other wave listed = %+v, want none", empty)
	}
}

// TestGenerateWritebacks_TemplateLayoutSnapshotsVersion checks that the
// active writeback output template drives both the rendered columns and the
// per-item template snapshot, and that in-place edits advance the snapshotted
// version while the most recently updated template wins the auto-pick.
func TestGenerateWritebacks_TemplateLayoutSnapshotsVersion(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB5")
	ws, ctx := p.ws, p.ctx
	older := csvWritebackTemplate(t, p)
	tpl := csvWritebackTemplate(t, p)
	edited := *tpl
	edited.Notes = "second revision"
	tpl, err := ws.UpdateTemplate(ctx, &edited)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if tpl.Version != 2 {
		t.Fatalf("version after edit = %d, want 2", tpl.Version)
	}
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: p.source.ID, ExternalCode: "ZTO", InternalName: "顺丰速运"}); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "SF-WB-TPL", "", "顺丰速运", 1); err != nil {
		t.Fatal(err)
	}

	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].TemplateID != tpl.ID || items[0].TemplateVersion != 2 {
		t.Fatalf("template snapshot = %d/%d, want %d/2 (not older template %d)", items[0].TemplateID, items[0].TemplateVersion, tpl.ID, older.ID)
	}
	if !strings.Contains(items[0].Payload, "订单号,快递公司编码,运单号\n") {
		t.Fatalf("template header missing: %q", items[0].Payload)
	}
	if !strings.Contains(items[0].Payload, "ORD-EXPORT-1,ZTO,SF-WB-TPL\n") {
		t.Fatalf("template row = %q, want order no, external code, tracking no", items[0].Payload)
	}
}
