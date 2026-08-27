package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// shippedRetailFact drives one retail fact through factory order, shipment,
// and writeback generation, returning the pending writeback items.
func shippedRetailFact(t *testing.T, sku string) (*readyPath, []domain.ChannelWritebackItem) {
	t.Helper()
	p := setupReadyRetailPath(t, sku)
	_, lines, err := p.ws.GenerateFactoryOrder(p.ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := p.ws.ImportShipment(p.ctx, lines[0].TrackingID, "SF-WB-0001", "SF", "顺丰速运", 2); err != nil {
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
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	tracking := lines[0].TrackingID
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-0001", "SF", "顺丰速运", 2); err != nil {
		t.Fatalf("ImportShipment 1: %v", err)
	}

	first, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks 1: %v", err)
	}
	if len(first) != 1 || first[0].Quantity != 2 {
		t.Fatalf("first run = %+v, want 1 item with quantity 2", first)
	}

	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-0002", "SF", "顺丰速运", 3); err != nil {
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
// carrier mapped from the internal code to the source platform's external
// vocabulary, and a conservative empty cell when no mapping exists.
func TestGenerateWritebacks_PayloadCarriesExternalCarrierCode(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB4")
	ws, ctx := p.ws, p.ctx
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: p.source.ID, ExternalCode: "SF-EXPRESS", InternalCode: "SF", InternalName: "顺丰速运"}); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	tracking := lines[0].TrackingID
	// First parcel carries a mappable internal code, second one a code without
	// a mapping on the source platform.
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-MAPPED", "SF", "顺丰速运", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ImportShipment(ctx, tracking, "SF-WB-UNMAPPED", "XX", "未知快递", 1); err != nil {
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
	if !strings.Contains(mapped, "订单号,快递公司编码\n") {
		t.Fatalf("builtin header missing from payload: %q", mapped)
	}
	if !strings.Contains(mapped, "ORD-EXPORT-1,SF-EXPRESS\n") {
		t.Fatalf("mapped payload = %q, want order no + external carrier code", mapped)
	}
	if !strings.Contains(unmapped, "ORD-EXPORT-1,\n") {
		t.Fatalf("unmapped payload = %q, want conservative empty carrier cell", unmapped)
	}
	if strings.HasPrefix(mapped, "\uFEFF") != true {
		t.Fatal("payload should keep the UTF-8 BOM like every rendered CSV")
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
		Version:      1,
		LayoutJSON:   layoutRaw,
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "SF-WB-XLSX", "SF", "顺丰速运", 1); err != nil {
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

// TestGenerateWritebacks_TemplateLayoutSnapshotsVersion checks that a
// configured writeback output template drives both the rendered columns and
// the per-item template snapshot.
func TestGenerateWritebacks_TemplateLayoutSnapshotsVersion(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-WB5")
	ws, ctx := p.ws, p.ctx
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
		Name:         "bilibili writeback",
		Version:      7,
		LayoutJSON:   layoutRaw,
	}
	if err := ws.CreateTemplate(ctx, tpl); err != nil {
		t.Fatal(err)
	}
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: p.source.ID, ExternalCode: "ZTO", InternalCode: "SF", InternalName: "顺丰速运"}); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "SF-WB-TPL", "SF", "顺丰速运", 1); err != nil {
		t.Fatal(err)
	}

	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].TemplateID != tpl.ID || items[0].TemplateVersion != 7 {
		t.Fatalf("template snapshot = %d/%d, want %d/7", items[0].TemplateID, items[0].TemplateVersion, tpl.ID)
	}
	if !strings.Contains(items[0].Payload, "订单号,快递公司编码,运单号\n") {
		t.Fatalf("template header missing: %q", items[0].Payload)
	}
	if !strings.Contains(items[0].Payload, "ORD-EXPORT-1,ZTO,SF-WB-TPL\n") {
		t.Fatalf("template row = %q, want order no, external code, tracking no", items[0].Payload)
	}
}
