package app

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestRenderTrackingExportCSV_SanitizesFormulaCells(t *testing.T) {
	t.Parallel()

	rules := &TemplateMappingRules{
		Version: 2,
		Mode:    "header",
		Columns: map[string]string{
			"export.tracking_no":          "物流单号",
			"export.carrier_code":         "快递公司",
			"export.external_document_no": "单号",
		},
		ColumnOrder: []string{
			"export.tracking_no",
			"export.carrier_code",
			"export.external_document_no",
		},
	}
	items := []domain.ChannelSyncItem{{
		ID:                 1,
		FulfillmentLineID:  42,
		TrackingNo:         "=1+1",
		CarrierCode:        "@SUM(1)",
		ExternalDocumentNo: "\tcmd",
	}}

	csvText, err := NewTemplatePayloadRenderer().RenderTrackingExportCSV(items, rules)
	if err != nil {
		t.Fatalf("RenderTrackingExportCSV: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(csvText)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%v", rows)
	}
	if rows[1][0] != "'=1+1" || rows[1][1] != "'@SUM(1)" || rows[1][2] != "'\tcmd" {
		t.Fatalf("data row = %#v", rows[1])
	}
}

func TestRenderSupplierExportCSV_SanitizesFormulaCells(t *testing.T) {
	t.Parallel()

	rules := &TemplateMappingRules{
		Version: 2,
		Mode:    "header",
		Columns: map[string]string{
			"export.factory_sku": "SKU",
			"export.quantity":    "Qty",
		},
		ColumnOrder: []string{"export.factory_sku", "export.quantity"},
	}
	lines := []*domain.SupplierOrderLine{{
		SupplierLineNo:    1,
		FulfillmentLineID: 8,
		SupplierSKU:       "+cmd|' /C calc",
		SubmittedQuantity: 2,
	}}
	fulfillLines := map[uint]*domain.FulfillmentLine{
		8: {ID: 8},
	}

	csvText, err := NewTemplatePayloadRenderer().RenderSupplierExportCSV(
		&domain.SupplierOrder{ID: 1},
		lines,
		fulfillLines,
		rules,
	)
	if err != nil {
		t.Fatalf("RenderSupplierExportCSV: %v", err)
	}

	rows, err := csv.NewReader(strings.NewReader(csvText)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%v", rows)
	}
	if rows[1][0] != `'+cmd|' /C calc` {
		t.Fatalf("sku cell = %q", rows[1][0])
	}
	if rows[1][1] != "2" {
		t.Fatalf("qty cell = %q", rows[1][1])
	}
}
