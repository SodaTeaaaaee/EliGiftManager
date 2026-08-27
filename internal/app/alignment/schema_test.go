package alignment

import (
	"strings"
	"testing"
)

func TestMappingConfigRoundTrip(t *testing.T) {
	cfg := MappingConfig{
		Mode:     ModeHeader,
		Columns:  map[string]string{"identity.value": "UID"},
		Defaults: map[string]string{"quantity": "1"},
		Transforms: map[string][]string{
			"identity.value":   {"trim"},
			"membership.level": {"mapEnum"},
		},
		EnumMaps:         map[string]map[string]string{"membership.level": {"总督": "governor"}},
		JoinSources:      map[string][]string{"recipient.address_line1": {"省", "市"}},
		SplitSkuQuantity: "product.alias_spec",
		Required:         []string{"identity.value"},
		Fingerprint:      []string{"identity.value"},
	}
	raw, err := SerializeMappingConfig(cfg)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if !strings.Contains(raw, `"version":3`) {
		t.Fatalf("serialized config missing version 3: %s", raw)
	}
	back, err := ParseMappingConfig(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if back.Mode != ModeHeader || !back.HasHeader {
		t.Fatalf("mode round trip = %q %v", back.Mode, back.HasHeader)
	}
	if back.Columns["identity.value"] != "UID" || back.SplitSkuQuantity != "product.alias_spec" {
		t.Fatalf("round trip lost fields: %+v", back)
	}
	if back.EnumMaps["membership.level"]["总督"] != "governor" {
		t.Fatalf("enum maps lost: %+v", back.EnumMaps)
	}
}

func TestParseMappingConfigErrors(t *testing.T) {
	if _, err := ParseMappingConfig(""); err == nil {
		t.Fatal("empty mapping must error")
	}
	if _, err := ParseMappingConfig("{not json"); err == nil {
		t.Fatal("bad json must error")
	}
	if _, err := ParseMappingConfig(`{"version":2}`); err == nil {
		t.Fatal("foreign version must error")
	}
	if _, err := ParseMappingConfig(`{"version":3,"mode":"telepathy"}`); err == nil {
		t.Fatal("unknown mode must error")
	}
	if _, err := ParseMappingConfig(`{"version":3,"mode":"header"}`); err == nil {
		t.Fatal("header mode without any column mapping must error")
	}
}

func TestParseMappingConfig_ModeNormalizesHasHeader(t *testing.T) {
	cfg, err := ParseMappingConfig(`{"version":3,"mode":"positional","positions":{"quantity":2}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.HasHeader {
		t.Fatal("positional mode must clear HasHeader")
	}
	cfg, err = ParseMappingConfig(`{"version":3,"hasHeader":true,"columns":{"quantity":"数量"}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Mode != ModeHeader {
		t.Fatal("HasHeader true must derive header mode")
	}
}

func TestLayoutConfigRoundTrip(t *testing.T) {
	cfg := LayoutConfig{
		Format:      FormatCSV,
		ColumnOrder: []string{"tracking.id", "quantity"},
		HeaderNames: map[string]string{"tracking.id": "第三方订单号"},
	}
	raw, err := SerializeLayoutConfig(cfg)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	back, err := ParseLayoutConfig(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if back.Format != FormatCSV || back.HeaderNames["tracking.id"] != "第三方订单号" {
		t.Fatalf("round trip = %+v", back)
	}
	if _, err := ParseLayoutConfig(`{"version":9,"format":"csv"}`); err == nil {
		t.Fatal("foreign layout version must error")
	}
	if _, err := ParseLayoutConfig(`{"version":1,"format":"pdf"}`); err == nil {
		t.Fatal("unsupported layout format must error")
	}
	if _, err := ParseLayoutConfig(""); err != nil {
		t.Fatal("empty layout is allowed (callers fall back to built-ins)")
	}
}
