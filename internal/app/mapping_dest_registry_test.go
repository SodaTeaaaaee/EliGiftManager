package app

import (
	"strings"
	"testing"
)

func TestDestCatalog_DemandImport(t *testing.T) {
	t.Parallel()
	for _, docType := range []string{"import_entitlement", "import_sales_order"} {
		cat := DestCatalog(docType)
		if len(cat) == 0 {
			t.Fatalf("%s: empty catalog", docType)
		}
		if !IsLegalDest(docType, "external_title") {
			t.Errorf("%s: unprefixed external_title should be legal", docType)
		}
		if !IsLegalDest(docType, "line.external_title") {
			t.Errorf("%s: line.external_title should be legal", docType)
		}
		if !IsLegalDest(docType, "document.source_customer_ref") {
			t.Errorf("%s: document.source_customer_ref should be legal", docType)
		}
		if !IsLegalDest(docType, "recipient.phone") {
			t.Errorf("%s: recipient.phone should be legal", docType)
		}
		if IsLegalDest(docType, "product.factory_sku") {
			t.Errorf("%s: product.factory_sku should NOT be legal", docType)
		}
	}
}

func TestDestCatalog_OtherDocTypes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		docType string
		legal   string
		illegal string
	}{
		{"import_product_catalog", "product.factory_sku", "shipment.tracking_no"},
		{"import_supplier_shipment", "shipment.tracking_no", "product.name"},
		{"import_carrier_mapping", "carrier.internal_carrier_code", "line.external_title"},
		{"export_supplier_order", "export.factory_sku", "product.factory_sku"},
		{"export_source_tracking_update", "tracking.tracking_no", "product.name"},
	}
	for _, tc := range cases {
		if !IsLegalDest(tc.docType, tc.legal) {
			t.Errorf("%s: %q should be legal", tc.docType, tc.legal)
		}
		if IsLegalDest(tc.docType, tc.illegal) {
			t.Errorf("%s: %q should be illegal", tc.docType, tc.illegal)
		}
		if len(DestCatalog(tc.docType)) == 0 {
			t.Errorf("%s: empty catalog", tc.docType)
		}
	}
	if DestCatalog("not_a_real_type") != nil {
		t.Error("unknown docType should yield nil catalog")
	}
}

func TestNormalizeDestKey(t *testing.T) {
	t.Parallel()
	if got := NormalizeDestKey("  Line.External_Title "); got != "line.external_title" {
		t.Errorf("got %q", got)
	}
}

func TestValidateMappingRulesConfig_OK(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {
			"line.external_title": "Name",
			"document.source_customer_ref": "UID"
		},
		"defaults": {"line.line_type": "sku_order"},
		"transforms": {"line.external_title": ["trim", "strip_quotes"]},
		"required": ["line.external_title"]
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ValidateMappingRulesConfig("import_entitlement", rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}
}

func TestValidateMappingRulesConfig_IllegalDest(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"product.factory_sku": "SKU"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = ValidateMappingRulesConfig("import_entitlement", rules)
	if err == nil || !strings.Contains(err.Error(), "illegal dest") {
		t.Fatalf("expected illegal dest error, got %v", err)
	}
}

func TestValidateMappingRulesConfig_UnknownTransform(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"transforms": {"line.external_title": ["explode"]}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = ValidateMappingRulesConfig("import_entitlement", rules)
	if err == nil || !strings.Contains(err.Error(), "unknown transform") {
		t.Fatalf("expected unknown transform error, got %v", err)
	}
}

func TestValidateMappingRulesConfig_RequiredMissingFromColumns(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"required": ["line.requested_quantity"]
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = ValidateMappingRulesConfig("import_entitlement", rules)
	if err == nil || !strings.Contains(err.Error(), "not present in columns or positions") {
		t.Fatalf("expected required-missing error, got %v", err)
	}
}

func TestValidateMappingRulesConfig_DuplicateRequired(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"line.external_title": "Name"},
		"required": ["line.external_title", "line.external_title"]
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = ValidateMappingRulesConfig("import_entitlement", rules)
	if err == nil || !strings.Contains(err.Error(), "duplicate dest") {
		t.Fatalf("expected duplicate dest error, got %v", err)
	}
}

func TestValidateMappingRulesConfig_DuplicateColumnsAndPositions(t *testing.T) {
	t.Parallel()
	rules := &TemplateMappingRules{
		Version:   2,
		Mode:      "header",
		Columns:   map[string]string{"line.external_title": "Name"},
		Positions: map[string]int{"line.external_title": 0},
	}
	err := ValidateMappingRulesConfig("import_entitlement", rules)
	if err == nil || !strings.Contains(err.Error(), "duplicate dest") {
		t.Fatalf("expected duplicate dest error, got %v", err)
	}
}

func TestValidateMappingRulesConfig_CatalogImageLayout(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"columns": {"product.factory_sku": "SKU", "product.name": "Name"},
		"imageLayout": {"enabled": true, "matchField": "product.name", "coverDir": "主图"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ValidateMappingRulesConfig("import_product_catalog", rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}
}
