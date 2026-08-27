package alignment

import (
	"strings"
	"testing"
)

func TestParse_MembershipPositionalFixture(t *testing.T) {
	data := fixture(t, "bilibili_membership_positional.csv")
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:     3,
		Mode:        ModePositional,
		Positions:   map[string]int{"membership.level": 0, "identity.value": 1, "customer.display_name": 2},
		Required:    []string{"identity.value"},
		Fingerprint: []string{"identity.value"},
	}}
	rows, issues, err := Parse(data, FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	first := rows[0]
	if first.Values["membership.level"] != "总督" || first.Values["identity.value"] != "uid-10001" || first.Values["customer.display_name"] != "DisplayA" {
		t.Fatalf("first row = %v", first.Values)
	}
	if rows[0].Fingerprint != rows[0].Fingerprint || rows[0].Fingerprint == rows[1].Fingerprint {
		t.Fatal("fingerprints must be stable per identity and differ across rows")
	}
	if rows[0].SourceRow != 1 || rows[2].SourceRow != 3 {
		t.Fatalf("source rows = %d,%d", rows[0].SourceRow, rows[2].SourceRow)
	}
}

func TestParse_RouzaoShipmentReturnFixture(t *testing.T) {
	data := fixture(t, "rouzao_shipment_return.csv")
	spec := TemplateSpec{Mapping: MappingConfig{
		Version: 3,
		Mode:    ModeHeader,
		Columns: map[string]string{
			"tracking.id":             "订单编号",
			"source.created_at":       "下单时间",
			"product.alias_id":        "商品编码",
			"product.alias_title":     "商品名称",
			"product.alias_spec":      "规格&数量",
			"recipient.name":          "收件人",
			"recipient.phone":         "电话",
			"recipient.address_line1": "收件信息",
			"shipment.carrier_name":   "物流公司",
			"shipment.tracking_no":    "物流单号",
			"shipment.shipped_at":     "打印快递时间",
		},
		Transforms: map[string][]string{
			"tracking.id":          {"trim", "strip_quotes"},
			"shipment.tracking_no": {"trim", "strip_quotes"},
			"recipient.phone":      {"normalizePhone"},
		},
		Required:    []string{"tracking.id", "shipment.tracking_no"},
		Fingerprint: []string{"tracking.id", "shipment.tracking_no"},
	}}
	rows, issues, err := Parse(data, FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	// Trailing newline inside the quoted order id must be trimmed away.
	if rows[0].Values["tracking.id"] != "RZ-ORD-001" {
		t.Fatalf("tracking id = %q", rows[0].Values["tracking.id"])
	}
	// Leading apostrophe on the tracking number must be stripped.
	if rows[0].Values["shipment.tracking_no"] != "YT1234567890" {
		t.Fatalf("tracking no = %q", rows[0].Values["shipment.tracking_no"])
	}
	if rows[0].Values["shipment.carrier_name"] != "申通快递" {
		t.Fatalf("carrier name = %q", rows[0].Values["shipment.carrier_name"])
	}
	if rows[0].Fingerprint == rows[1].Fingerprint || rows[0].Fingerprint == rows[2].Fingerprint {
		t.Fatal("fingerprint must key on (tracking id, tracking no)")
	}
}

func TestParse_SplitSkuQuantityExpandsRows(t *testing.T) {
	data := fixture(t, "rouzao_shipment_return.csv")
	spec := TemplateSpec{Mapping: MappingConfig{
		Version: 3,
		Mode:    ModeHeader,
		Columns: map[string]string{
			"tracking.id":        "订单编号",
			"product.alias_id":   "商品编码",
			"product.alias_spec": "规格&数量",
		},
		Transforms: map[string][]string{
			"tracking.id": {"trim", "strip_quotes"},
		},
		SplitSkuQuantity: "product.alias_spec",
		Required:         []string{"tracking.id", "product.alias_id"},
	}}
	rows, issues, err := Parse(data, FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %v", issues)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4 (row 1 expands to 2)", len(rows))
	}
	first, second := rows[0], rows[1]
	if first.SourceRow != 1 || second.SourceRow != 1 {
		t.Fatalf("expanded rows must share the source row, got %d and %d", first.SourceRow, second.SourceRow)
	}
	if first.LineNo == second.LineNo {
		t.Fatal("expanded rows must have distinct line numbers")
	}
	if first.Values["product.alias_id"] != "206068021" || first.Values["quantity"] != "1" {
		t.Fatalf("first segment = %v", first.Values)
	}
	if second.Values["product.alias_id"] != "63098307" || second.Values["quantity"] != "2" {
		t.Fatalf("second segment = %v", second.Values)
	}
	if second.Values["product.alias_title"] != "设计师款透明单插立牌-底座可印刷15cm" && second.Values["product.alias_title"] != "magnet-label" {
		// fixture row 1: 206068021_standee-label * 1 | 63098307_magnet-label * 2
		t.Fatalf("second segment title = %q", second.Values["product.alias_title"])
	}
	if second.Values["tracking.id"] != first.Values["tracking.id"] {
		t.Fatalf("expanded rows must carry the source row's other values")
	}
}

func TestParse_JoinSourcesWithAddressTransform(t *testing.T) {
	csvData := "省,市,区,详细地址,收件人\n江苏省,南京市,栖霞区,仙林大道1号,小明\n"
	spec := TemplateSpec{Mapping: MappingConfig{
		Version: 3,
		Mode:    ModeHeader,
		Columns: map[string]string{
			"recipient.name":          "收件人",
			"recipient.address_line1": "详细地址",
		},
		JoinSources: map[string][]string{
			"recipient.address_line1": {"省", "市", "区", "详细地址"},
		},
	}}
	rows, issues, err := Parse([]byte(csvData), FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(issues) != 0 || len(rows) != 1 {
		t.Fatalf("issues = %v rows = %d", issues, len(rows))
	}
	if got := rows[0].Values["recipient.address_line1"]; got != "江苏省南京市栖霞区仙林大道1号" {
		t.Fatalf("joined address = %q", got)
	}
}

func TestParse_DefaultsOverrideAndRequired(t *testing.T) {
	csvData := "订单号,数量\nA1,\n,5\n"
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:  3,
		Mode:     ModeHeader,
		Columns:  map[string]string{"source.document_no": "订单号", "quantity": "数量"},
		Defaults: map[string]string{"quantity": "1"},
		Required: []string{"source.document_no"},
	}}
	rows, issues, err := Parse([]byte(csvData), FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 (second row dropped)", len(rows))
	}
	if rows[0].Values["quantity"] != "1" {
		t.Fatalf("default quantity = %q", rows[0].Values["quantity"])
	}
	if len(issues) != 1 || issues[0].Key != "source.document_no" || issues[0].LineNo != 2 {
		t.Fatalf("issues = %v", issues)
	}
}

func TestParse_MissingHeaderColumnReportedOnce(t *testing.T) {
	csvData := "订单号,备注\nA1,x\n"
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:  3,
		Mode:     ModeHeader,
		Columns:  map[string]string{"source.document_no": "订单号", "shipment.tracking_no": "不存在的列"},
		Required: []string{"source.document_no"},
	}}
	rows, issues, err := Parse([]byte(csvData), FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	if len(issues) != 1 || issues[0].LineNo != 0 || !strings.Contains(issues[0].Message, "不存在的列") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestParse_TransformFailureKeepsRowAndRecordsIssue(t *testing.T) {
	csvData := "时间\nwhenever\n"
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:    3,
		Mode:       ModeHeader,
		Columns:    map[string]string{"source.created_at": "时间"},
		Transforms: map[string][]string{"source.created_at": {"parseDate"}},
	}}
	rows, issues, err := Parse([]byte(csvData), FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 (transform failure is not fatal)", len(rows))
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "whenever") {
		t.Fatalf("issues = %v", issues)
	}
	if rows[0].Values["source.created_at"] != "whenever" {
		t.Fatalf("value on failure should stay at the pre-transform text, got %q", rows[0].Values["source.created_at"])
	}
}

// finishRow must surface an unregistered transformer as an issue instead of
// silently skipping the transform step (defense in depth behind
// buildTransformers' up-front validation).
func TestFinishRow_UnknownTransformerRecordsIssue(t *testing.T) {
	values := map[string]string{"quantity": " 5 "}
	cfg := MappingConfig{Transforms: map[string][]string{"quantity": {"trim", "explode"}}}
	row, issues := finishRow(values, cfg, map[string]Transformer{"trim": transformTrim}, 7)
	if row == nil {
		t.Fatal("row must survive an unknown transformer")
	}
	if len(issues) != 1 || issues[0].LineNo != 7 || issues[0].Key != "quantity" || !strings.Contains(issues[0].Message, "explode") {
		t.Fatalf("issues = %+v, want one unknown-transformer issue for quantity", issues)
	}
}

func TestParse_SplitFailureDropsRow(t *testing.T) {
	csvData := "规格&数量\nsku_a * many\n"
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:          3,
		Mode:             ModeHeader,
		Columns:          map[string]string{"product.alias_spec": "规格&数量"},
		SplitSkuQuantity: "product.alias_spec",
	}}
	rows, issues, err := Parse([]byte(csvData), FormatCSV, spec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(rows))
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "many") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestParse_DefaultFingerprintUsesAllValues(t *testing.T) {
	spec := TemplateSpec{Mapping: MappingConfig{Version: 3, Mode: ModeHeader, Columns: map[string]string{"a": "A"}}}
	rowsA, _, err := Parse([]byte("A\nx\n"), FormatCSV, spec)
	if err != nil {
		t.Fatal(err)
	}
	rowsB, _, err := Parse([]byte("A\nx\n"), FormatCSV, spec)
	if err != nil {
		t.Fatal(err)
	}
	rowsC, _, err := Parse([]byte("A\ny\n"), FormatCSV, spec)
	if err != nil {
		t.Fatal(err)
	}
	if rowsA[0].Fingerprint != rowsB[0].Fingerprint {
		t.Fatal("identical rows must fingerprint identically")
	}
	if rowsA[0].Fingerprint == rowsC[0].Fingerprint {
		t.Fatal("different rows must fingerprint differently")
	}
}

func TestTestTemplate_ReturnsPreviewAndIssues(t *testing.T) {
	data := fixture(t, "bilibili_membership_positional.csv")
	spec := TemplateSpec{Mapping: MappingConfig{
		Version:   3,
		Positions: map[string]int{"membership.level": 0, "identity.value": 1},
		Required:  []string{"identity.value"},
	}}
	preview, err := TestTemplate(data, FormatCSV, spec, 2)
	if err != nil {
		t.Fatalf("test template: %v", err)
	}
	if len(preview.Rows) != 2 {
		t.Fatalf("preview rows = %d, want 2", len(preview.Rows))
	}
	if preview.Issues == nil {
		t.Fatal("issues must be non-nil for JSON transport")
	}
}
