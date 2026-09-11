package alignment

import (
	"bytes"
	"strings"
	"testing"
)

func sampleLayout() LayoutConfig {
	return LayoutConfig{
		Version:     LayoutSchemaVersion,
		Format:      FormatCSV,
		ColumnOrder: []string{"tracking.id", "recipient.name", "quantity"},
		HeaderNames: map[string]string{
			"tracking.id":    "第三方订单号",
			"recipient.name": "收件人",
		},
	}
}

func TestRender_CSVHasBOMHeaderNamesAndSanitizer(t *testing.T) {
	rows := []map[string]string{
		{"tracking.id": "abc123", "recipient.name": "=SUM(A1:A2)", "quantity": "2"},
		{"tracking.id": "def456", "recipient.name": "小明", "quantity": "1"},
	}
	out, err := Render(rows, sampleLayout())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.HasPrefix(out, utf8BOM) {
		t.Fatal("csv output must start with a UTF-8 BOM")
	}
	text := string(bytes.TrimPrefix(out, utf8BOM))
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if lines[0] != "第三方订单号,收件人,quantity" {
		t.Fatalf("header = %q", lines[0])
	}
	if lines[1] != "abc123,'=SUM(A1:A2),2" {
		t.Fatalf("row 1 = %q (formula injection must be escaped)", lines[1])
	}
	if lines[2] != "def456,小明,1" {
		t.Fatalf("row 2 = %q", lines[2])
	}
}

func TestRender_EmptyRowsStillRenderHeader(t *testing.T) {
	out, err := Render(nil, sampleLayout())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	text := string(bytes.TrimPrefix(out, utf8BOM))
	if strings.TrimSuffix(text, "\n") != "第三方订单号,收件人,quantity" {
		t.Fatalf("header-only output = %q", text)
	}
}

func TestRender_XLSXRoundTripAndSanitize(t *testing.T) {
	layout := sampleLayout()
	layout.Format = FormatXLSX
	layout.HeaderNames = nil
	rows := []map[string]string{
		{"tracking.id": "abc123", "recipient.name": "+cmd", "quantity": "2"},
	}
	out, err := Render(rows, layout)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	records, err := ReadRows(out, FormatXLSX, "")
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if records[0][0] != "tracking.id" {
		t.Fatalf("default header name = %q", records[0][0])
	}
	if records[1][1] != "'+cmd" {
		t.Fatalf("xlsx formula injection must be escaped too, got %q", records[1][1])
	}
	if records[1][0] != "abc123" {
		t.Fatalf("cell = %q", records[1][0])
	}
}

func TestRender_RejectsBadLayouts(t *testing.T) {
	if _, err := Render(nil, LayoutConfig{Format: FormatCSV}); err == nil {
		t.Fatal("missing column order must error")
	}
	if _, err := Render(nil, LayoutConfig{Format: "pdf", ColumnOrder: []string{"a"}}); err == nil {
		t.Fatal("unsupported format must error")
	}
}

func TestRenderPreviewString(t *testing.T) {
	rows := []map[string]string{
		{"tracking.id": "a", "quantity": "1"},
		{"tracking.id": "b", "quantity": "2"},
		{"tracking.id": "c", "quantity": "3"},
	}
	got, err := RenderPreviewString(rows, sampleLayout(), 2)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], "第三方订单号") || strings.Contains(got, "| c") {
		t.Fatalf("preview = %q", got)
	}
}
