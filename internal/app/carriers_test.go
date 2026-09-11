package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestImportCarrierMappings_UpsertsByNormalizedName(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)
	fixture := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_carrier_codes.csv")

	// A pre-existing hand-made mapping with the long spelling is updated, not
	// duplicated, because 顺丰速运 and 顺丰 normalize alike.
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: source.ID, InternalName: "顺丰速运", ExternalCode: "old-sf"}); err != nil {
		t.Fatal(err)
	}

	result, err := ws.ImportCarrierMappings(ctx, source.ID, fixture, "快递公司名称", "快递公司编码")
	if err != nil {
		t.Fatalf("ImportCarrierMappings: %v", err)
	}
	if result.Created != 1 || result.Updated != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v, want created 1 (圆通), updated 1 (顺丰), skipped 1 (空编码)", result)
	}
	if len(result.Issues) != 1 || result.Issues[0].LineNo != 2 || result.Issues[0].Key != "shipment.carrier_code" {
		t.Fatalf("issues = %+v, want one empty-code issue on data row 2", result.Issues)
	}
	mappings, err := ws.ListCarrierMappings(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]domain.CarrierMapping{}
	for _, m := range mappings {
		byName[m.InternalName] = m
	}
	if len(mappings) != 2 {
		t.Fatalf("mappings = %+v, want 2", mappings)
	}
	if byName["顺丰"].ExternalCode != "shunfeng" || byName["圆通"].ExternalCode != "yto" {
		t.Fatalf("mappings = %+v", byName)
	}
	if _, stale := byName["顺丰速运"]; stale {
		t.Fatal("the pre-existing row must be renamed to the imported spelling, not duplicated")
	}

	// Re-import is idempotent: everything valid counts as updated, nothing new.
	again, err := ws.ImportCarrierMappings(ctx, source.ID, fixture, "快递公司名称", "快递公司编码")
	if err != nil {
		t.Fatal(err)
	}
	if again.Created != 0 || again.Updated != 2 || again.Skipped != 1 {
		t.Fatalf("re-import = %+v", again)
	}
	mappings, err = ws.ListCarrierMappings(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 2 {
		t.Fatalf("mappings after re-import = %d, want 2", len(mappings))
	}

	// Bad headers and unknown platforms fail clearly.
	if _, err := ws.ImportCarrierMappings(ctx, source.ID, fixture, "不存在", "快递公司编码"); err == nil {
		t.Fatal("missing name header must fail")
	}
	if _, err := ws.ImportCarrierMappings(ctx, source.ID, fixture, "快递公司名称", ""); err == nil {
		t.Fatal("empty code header must fail")
	}
	if _, err := ws.ImportCarrierMappings(ctx, 9999, fixture, "快递公司名称", "快递公司编码"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown platform err = %v, want ErrNotFound", err)
	}
}

func TestCarrierMapping_UpdateAndDelete(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)
	m := &domain.CarrierMapping{PlatformID: source.ID, InternalName: " 中通 ", ExternalCode: " zhongtong "}
	if err := ws.CreateCarrierMapping(ctx, m); err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 || m.InternalName != "中通" || m.ExternalCode != "zhongtong" {
		t.Fatalf("created = %+v, want trimmed fields", m)
	}
	if err := ws.CreateCarrierMapping(ctx, &domain.CarrierMapping{PlatformID: source.ID, InternalName: "无码"}); err == nil {
		t.Fatal("mapping without a code must be refused")
	}

	updated, err := ws.UpdateCarrierMapping(ctx, &domain.CarrierMapping{ID: m.ID, PlatformID: 42, InternalName: "中通快递", ExternalCode: "zto", InternalCode: "ZTO"})
	if err != nil {
		t.Fatalf("UpdateCarrierMapping: %v", err)
	}
	if updated.PlatformID != source.ID || updated.InternalName != "中通快递" || updated.ExternalCode != "zto" || updated.InternalCode != "ZTO" {
		t.Fatalf("updated = %+v, want new fields and the original platform", updated)
	}
	stored, err := ws.Store.GetCarrierMapping(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ExternalCode != "zto" || stored.PlatformID != source.ID {
		t.Fatalf("stored = %+v", stored)
	}
	if _, err := ws.UpdateCarrierMapping(ctx, &domain.CarrierMapping{ID: m.ID, InternalName: "", ExternalCode: "x"}); err == nil {
		t.Fatal("blank name must be refused")
	}
	if _, err := ws.UpdateCarrierMapping(ctx, &domain.CarrierMapping{ID: 9999, InternalName: "x", ExternalCode: "y"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing update err = %v, want ErrNotFound", err)
	}

	if err := ws.DeleteCarrierMapping(ctx, m.ID); err != nil {
		t.Fatalf("DeleteCarrierMapping: %v", err)
	}
	if err := ws.DeleteCarrierMapping(ctx, m.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second delete err = %v, want ErrNotFound", err)
	}
	left, err := ws.ListCarrierMappings(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("mappings after delete = %+v, want none", left)
	}
}

// TestImportCarrierMappings_FeedsWritebackTranslation runs the whole carrier
// chain: import bilibili's table, ship through the factory file's spelling,
// and see the platform's carrier id in the writeback payload.
func TestImportCarrierMappings_FeedsWritebackTranslation(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-CARRIER")
	ws, ctx := p.ws, p.ctx
	csvWritebackTemplate(t, p)
	fixture := filepath.Join("..", "..", "testdata", "integration_profile", "bilibili_carrier_codes.csv")
	if _, err := ws.ImportCarrierMappings(ctx, p.source.ID, fixture, "快递公司名称", "快递公司编码"); err != nil {
		t.Fatal(err)
	}
	_, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.ImportShipment(ctx, lines[0].TrackingID, "YT-1", "", "圆通速递", 2); err != nil {
		t.Fatal(err)
	}
	items, err := ws.GenerateWritebacks(ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 || items[0].CarrierCode != "yto" {
		t.Fatalf("items = %+v, want carrier code yto translated from 圆通速递", items)
	}
}
