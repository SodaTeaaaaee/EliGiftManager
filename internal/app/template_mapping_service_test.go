package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestParseMappingRules_V1Compat(t *testing.T) {
	t.Parallel()
	raw := `{"columns":{"external_title":"Name","requested_quantity":"Qty"},"defaults":{"line_type":"sku_order"}}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.Version != 1 {
		t.Errorf("expected version=1, got %d", rules.Version)
	}
	if rules.Mode != "header" {
		t.Errorf("expected mode=header, got %q", rules.Mode)
	}
	if !rules.HasHeader {
		t.Error("expected hasHeader=true for v1")
	}
	if rules.Columns["external_title"] != "Name" {
		t.Errorf("expected column external_title=Name, got %q", rules.Columns["external_title"])
	}
	if rules.Defaults["line_type"] != "sku_order" {
		t.Errorf("expected default line_type=sku_order, got %q", rules.Defaults["line_type"])
	}
}

func TestParseMappingRules_V2Header(t *testing.T) {
	t.Parallel()
	raw := `{
		"version": 2,
		"mode": "header",
		"hasHeader": true,
		"columns": {"line.gift_level_snapshot": "大航海等级", "line.external_title": "Name"},
		"defaults": {"line.line_type": "entitlement_rule"},
		"transforms": {"line.gift_level_snapshot": ["trim"]},
		"columnOrder": ["line.external_title", "line.gift_level_snapshot"],
		"required": ["line.external_title"]
	}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.Version != 2 || rules.Mode != "header" {
		t.Fatalf("unexpected rules: version=%d mode=%q", rules.Version, rules.Mode)
	}
	if rules.Columns["line.gift_level_snapshot"] != "大航海等级" {
		t.Errorf("columns: %+v", rules.Columns)
	}
	if len(rules.ColumnOrder) != 2 || rules.ColumnOrder[0] != "line.external_title" {
		t.Errorf("columnOrder: %+v", rules.ColumnOrder)
	}
	if len(rules.Required) != 1 || rules.Required[0] != "line.external_title" {
		t.Errorf("required: %+v", rules.Required)
	}
}

func TestParseMappingRules_V2Positional(t *testing.T) {
	t.Parallel()
	raw := `{
		"version": 2,
		"mode": "positional",
		"hasHeader": false,
		"positions": {"line.external_title": 0, "line.requested_quantity": 1},
		"defaults": {"line.line_type": "sku_order"}
	}`
	rules, err := ParseMappingRules(raw)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.Mode != "positional" {
		t.Fatalf("mode: %q", rules.Mode)
	}
	if rules.Positions["line.external_title"] != 0 {
		t.Errorf("positions: %+v", rules.Positions)
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

func TestParseMappingRules_UnsupportedVersion(t *testing.T) {
	t.Parallel()
	_, err := ParseMappingRules(`{"version": 9, "columns":{"a":"b"}}`)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestApplyRow_HeaderTransformsAndRequired(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"hasHeader": true,
		"columns": {
			"line.external_title": "Name",
			"shipment.tracking_no": "Tracking"
		},
		"transforms": {
			"line.external_title": ["trim", "strip_quotes"],
			"shipment.tracking_no": ["trim", "strip_leading_quote"]
		},
		"defaults": {"line.line_type": "sku_order"},
		"required": ["line.external_title", "shipment.tracking_no"]
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	headers := []string{"Name", "Tracking"}
	row := []string{`  "Standee Set"  `, "'SF123456"}
	out, warnings, err := ApplyRow(row, headers, rules)
	if err != nil {
		t.Fatalf("ApplyRow: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if out["line.external_title"] != "Standee Set" {
		t.Errorf("external_title: %q", out["line.external_title"])
	}
	if out["shipment.tracking_no"] != "SF123456" {
		t.Errorf("tracking_no: %q", out["shipment.tracking_no"])
	}
	if out["line.line_type"] != "sku_order" {
		t.Errorf("line_type default: %q", out["line.line_type"])
	}
}

func TestApplyRow_DefaultDoesNotOverwriteMapped(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"defaults": {"line.external_title": "DEFAULT_TITLE"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _, err := ApplyRow([]string{"Mapped Title"}, []string{"Name"}, rules)
	if err != nil {
		t.Fatalf("ApplyRow: %v", err)
	}
	if out["line.external_title"] != "Mapped Title" {
		t.Errorf("mapped value must win over default: got %q", out["line.external_title"])
	}
}

func TestApplyRow_DefaultFillsEmptyDest(t *testing.T) {
	t.Parallel()

	t.Run("dest not in columns", func(t *testing.T) {
		t.Parallel()
		rules, err := ParseMappingRules(`{
			"version": 2,
			"mode": "header",
			"columns": {"line.external_title": "Name"},
			"defaults": {"line.line_type": "sku_order"}
		}`)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		out, _, err := ApplyRow([]string{"Poster"}, []string{"Name"}, rules)
		if err != nil {
			t.Fatalf("ApplyRow: %v", err)
		}
		if out["line.line_type"] != "sku_order" {
			t.Errorf("missing dest should get default: got %q", out["line.line_type"])
		}
		if out["line.external_title"] != "Poster" {
			t.Errorf("mapped dest: %q", out["line.external_title"])
		}
	})

	t.Run("mapped to blank", func(t *testing.T) {
		t.Parallel()
		rules, err := ParseMappingRules(`{
			"version": 2,
			"mode": "header",
			"columns": {"line.external_title": "Name", "line.line_type": "Type"},
			"defaults": {"line.line_type": "sku_order"}
		}`)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		out, _, err := ApplyRow([]string{"Poster", "  "}, []string{"Name", "Type"}, rules)
		if err != nil {
			t.Fatalf("ApplyRow: %v", err)
		}
		if out["line.line_type"] != "sku_order" {
			t.Errorf("blank mapped dest should get default: got %q", out["line.line_type"])
		}
	})
}

func TestApplyRow_HeaderTrimSpace(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, _, err := ApplyRow([]string{"Standee", "1"}, []string{" Name ", "Qty"}, rules)
	if err != nil {
		t.Fatalf("ApplyRow: %v", err)
	}
	if out["line.external_title"] != "Standee" {
		t.Errorf("padded header should still map: got %q", out["line.external_title"])
	}
}

func TestApplyRow_RequiredMissing(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"required": ["line.external_title"]
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, _, err = ApplyRow([]string{""}, []string{"Name"}, rules)
	if err == nil {
		t.Fatal("expected required-field error")
	}
}

func TestApplyRow_Positional(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "positional",
		"positions": {"line.external_title": 0, "line.requested_quantity": 2},
		"transforms": {"line.external_title": ["trim"]}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, warnings, err := ApplyRow([]string{"  Poster  ", "ignored", "3"}, nil, rules)
	if err != nil {
		t.Fatalf("ApplyRow: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if out["line.external_title"] != "Poster" {
		t.Errorf("title: %q", out["line.external_title"])
	}
	if out["line.requested_quantity"] != "3" {
		t.Errorf("qty: %q", out["line.requested_quantity"])
	}
}

func TestApplyRow_UnknownTransform(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"transforms": {"line.external_title": ["nope"]}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, _, err = ApplyRow([]string{"x"}, []string{"Name"}, rules)
	if err == nil {
		t.Fatal("expected unknown transform error")
	}
}

func TestApplyRow_UnknownDestWarning(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {
			"line.external_title": "Name",
			"totally.unknown_field": "X"
		}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, warnings, err := ApplyRow([]string{"Badge", "keep-me"}, []string{"Name", "X"}, rules)
	if err != nil {
		t.Fatalf("ApplyRow: %v", err)
	}
	if out["totally.unknown_field"] != "keep-me" {
		t.Fatalf("unknown dest value dropped: got %q", out["totally.unknown_field"])
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "totally.unknown_field") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestMapCSVRowToDemandLine(t *testing.T) {
	t.Parallel()
	rules := &TemplateMappingRules{
		Version: 1,
		Mode:    "header",
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

func TestMapCSVRowToDemandLine_V2Prefixed(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {
			"line.external_title": "Name",
			"line.requested_quantity": "Qty"
		},
		"defaults": {"line.line_type": "sku_order"},
		"transforms": {"line.external_title": ["trim"]}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	line, err := MapCSVRowToDemandLine(map[string]string{
		"Name": "  Badge  ",
		"Qty":  "1",
	}, rules)
	if err != nil {
		t.Fatalf("MapCSVRowToDemandLine: %v", err)
	}
	if line.ExternalTitle != "Badge" {
		t.Errorf("title: %q", line.ExternalTitle)
	}
	if line.RequestedQuantity != 1 {
		t.Errorf("qty: %d", line.RequestedQuantity)
	}
	if line.LineType != "sku_order" {
		t.Errorf("line_type: %q", line.LineType)
	}
}

func TestMapCSVRowUnknownField(t *testing.T) {
	rules := &TemplateMappingRules{
		Mode: "header",
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
	t.Parallel()
	rules := &TemplateMappingRules{
		Mode: "header",
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

func TestRequestedQuantity_RejectsNonPositiveAndNonInteger(t *testing.T) {
	t.Parallel()

	t.Run("2 succeeds", func(t *testing.T) {
		t.Parallel()
		line := &domain.DemandLine{}
		if err := setDemandLineField(line, "requested_quantity", "2"); err != nil {
			t.Fatalf("expected success for %q: %v", "2", err)
		}
		if line.RequestedQuantity != 2 {
			t.Errorf("got qty %d", line.RequestedQuantity)
		}

		rowRules := &TemplateMappingRules{
			Mode:    "header",
			Columns: map[string]string{"requested_quantity": "Qty"},
		}
		mapped, err := MapCSVRowToDemandLine(map[string]string{"Qty": "2"}, rowRules)
		if err != nil {
			t.Fatalf("MapCSVRowToDemandLine: %v", err)
		}
		if mapped.RequestedQuantity != 2 {
			t.Errorf("MapCSVRowToDemandLine qty: %d", mapped.RequestedQuantity)
		}
	})

	for _, raw := range []string{"0", "-1", "1.5", "abc"} {
		raw := raw
		t.Run("invalid_"+raw, func(t *testing.T) {
			t.Parallel()
			line := &domain.DemandLine{RequestedQuantity: 7}
			err := setDemandLineField(line, "requested_quantity", raw)
			if err == nil {
				t.Fatalf("expected error for quantity %q", raw)
			}
			if line.RequestedQuantity != 7 {
				t.Errorf("must not write invalid quantity %q: got %d", raw, line.RequestedQuantity)
			}

			rowRules := &TemplateMappingRules{
				Mode:    "header",
				Columns: map[string]string{"requested_quantity": "Qty"},
			}
			if _, err := MapCSVRowToDemandLine(map[string]string{"Qty": raw}, rowRules); err == nil {
				t.Fatalf("MapCSVRowToDemandLine expected error for quantity %q", raw)
			}
		})
	}
}

func TestMapCSVRow_PositionalRejected(t *testing.T) {
	t.Parallel()
	rules := &TemplateMappingRules{
		Version:   2,
		Mode:      "positional",
		Positions: map[string]int{"line.external_title": 0},
	}
	_, err := MapCSVRowToDemandLine(map[string]string{"0": "x"}, rules)
	if err == nil {
		t.Fatal("expected error for positional via MapCSVRowToDemandLine")
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
	if parsed.Version != 1 {
		t.Errorf("expected v1 after roundtrip, got %d", parsed.Version)
	}
}
