package app

import (
	"testing"
)

func TestOrderedColumns_StableAlphabeticalFallback(t *testing.T) {
	t.Parallel()
	r := NewTemplatePayloadRenderer()
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"zebra":  "Z",
			"alpha":  "A",
			"middle": "M",
		},
	}

	// Run multiple times — map range order is random; result must be stable.
	for i := 0; i < 20; i++ {
		cols := r.orderedColumns(rules)
		if len(cols) != 3 {
			t.Fatalf("run %d: len=%d", i, len(cols))
		}
		if cols[0].src != "alpha" || cols[1].src != "middle" || cols[2].src != "zebra" {
			t.Fatalf("run %d: unstable order %+v", i, cols)
		}
		if cols[0].output != "A" || cols[1].output != "M" || cols[2].output != "Z" {
			t.Fatalf("run %d: outputs %+v", i, cols)
		}
	}
}

func TestOrderedColumns_UsesColumnOrder(t *testing.T) {
	t.Parallel()
	r := NewTemplatePayloadRenderer()
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"export.sku":   "SKU",
			"export.qty":   "Qty",
			"export.name":  "Name",
			"export.extra": "Extra",
		},
		ColumnOrder: []string{"export.name", "export.sku", "export.qty"},
	}
	cols := r.orderedColumns(rules)
	if len(cols) != 4 {
		t.Fatalf("expected 4 cols (3 ordered + 1 remainder), got %d", len(cols))
	}
	if cols[0].src != "export.name" || cols[1].src != "export.sku" || cols[2].src != "export.qty" {
		t.Fatalf("columnOrder prefix: %+v", cols)
	}
	// Remainder appended alphabetically.
	if cols[3].src != "export.extra" {
		t.Fatalf("remainder: %+v", cols)
	}
}

func TestRenderMappedRow_AppliesTransformsDefaultsAndRequired(t *testing.T) {
	t.Parallel()
	r := NewTemplatePayloadRenderer()
	rules := &TemplateMappingRules{
		Columns: map[string]string{
			"export.factory_sku": "SKU",
			"export.quantity":    "Qty",
		},
		ColumnOrder: []string{"export.factory_sku", "export.quantity"},
		Transforms: map[string][]string{
			"export.factory_sku": {"trim"},
			"export.quantity":    {"trim"},
		},
		Defaults: map[string]string{"export.quantity": " 3 "},
		Required: []string{"export.factory_sku", "export.quantity"},
	}
	columns := r.orderedColumns(rules)
	row, err := r.renderMappedRow(columns, rules, func(dest string) string {
		if dest == "export.factory_sku" {
			return " SKU-1 "
		}
		return ""
	})
	if err != nil {
		t.Fatalf("renderMappedRow: %v", err)
	}
	if row[0] != "SKU-1" || row[1] != "3" {
		t.Fatalf("row = %#v, want [SKU-1 3]", row)
	}

	_, err = r.renderMappedRow(columns, rules, func(string) string { return "" })
	if err == nil || err.Error() != `required field "export.factory_sku" is missing or empty` {
		t.Fatalf("required error = %v", err)
	}
}
