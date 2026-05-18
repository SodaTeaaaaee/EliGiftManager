package app

import (
	"encoding/json"
	"testing"
)

func TestParseMappingRules(t *testing.T) {
	t.Parallel()
	raw := `{"columns":{"external_title":"Name","requested_quantity":"Qty"},"defaults":{"line_type":"sku_order"}}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.Columns["external_title"] != "Name" {
		t.Errorf("expected column external_title=Name, got %q", rules.Columns["external_title"])
	}
	if rules.Defaults["line_type"] != "sku_order" {
		t.Errorf("expected default line_type=sku_order, got %q", rules.Defaults["line_type"])
	}
}

func TestParseMappingRulesEmpty(t *testing.T) {
	t.Parallel()
	_, err := ParseMappingRules("")
	if err == nil {
		t.Error("expected error for empty rules")
	}
}

func TestParseMappingRulesNoColumns(t *testing.T) {
	t.Parallel()
	raw := `{"columns":{},"defaults":{"line_type":"sku_order"}}`
	_, err := ParseMappingRules(raw)
	if err == nil {
		t.Error("expected error for rules with no columns")
	}
}

func TestParseMappingRulesInvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := ParseMappingRules("{not json}")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMapCSVRowToDemandLine(t *testing.T) {
	t.Parallel()
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"external_title":     "Product",
			"requested_quantity": "Qty",
			"line_type":          "Type",
		},
		Defaults: map[string]string{
			"entitlement_authority": "upstream_platform",
		},
	}

	row := map[string]string{
		"Product": "Standee Set",
		"Qty":     "3",
		"Type":    "sku_order",
	}

	line, err := MapCSVRowToDemandLine(row, rules)
	if err != nil {
		t.Fatalf("MapCSVRowToDemandLine: %v", err)
	}
	if line.ExternalTitle != "Standee Set" {
		t.Errorf("expected ExternalTitle=Standee Set, got %q", line.ExternalTitle)
	}
	if line.RequestedQuantity != 3 {
		t.Errorf("expected RequestedQuantity=3, got %d", line.RequestedQuantity)
	}
	if line.LineType != "sku_order" {
		t.Errorf("expected LineType=sku_order, got %q", line.LineType)
	}
	if line.EntitlementAuthority != "upstream_platform" {
		t.Errorf("expected EntitlementAuthority=upstream_platform, got %q", line.EntitlementAuthority)
	}
}

func TestMapCSVRowUnknownField(t *testing.T) {
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"unknown_field": "Column1",
		},
	}
	row := map[string]string{"Column1": "value"}
	_, err := MapCSVRowToDemandLine(row, rules)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestMapCSVRowInvalidQuantity(t *testing.T) {
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"requested_quantity": "Qty",
		},
	}
	row := map[string]string{"Qty": "not-a-number"}
	_, err := MapCSVRowToDemandLine(row, rules)
	if err == nil {
		t.Error("expected error for invalid quantity")
	}
}

func TestTemplateMappingRulesJSONRoundtrip(t *testing.T) {
	rules := TemplateMappingRules{
		Columns:  map[string]string{"external_title": "Name"},
		Defaults: map[string]string{"line_type": "entitlement_rule"},
	}
	data, _ := json.Marshal(rules)
	parsed, err := ParseMappingRules(string(data))
	if err != nil {
		t.Fatalf("roundtrip parse: %v", err)
	}
	if parsed.Columns["external_title"] != "Name" {
		t.Error("roundtrip mismatch")
	}
}
