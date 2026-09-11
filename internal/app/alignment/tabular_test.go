package alignment

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "integration_profile", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestReadRows_CSVStripsBOMAndKeepsQuotedNewlines(t *testing.T) {
	data := fixture(t, "rouzao_shipment_return.csv")
	records, err := ReadRows(data, FormatCSV, "")
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("records = %d, want 4 (header + 3 rows)", len(records))
	}
	// The first data row has an embedded newline inside the quoted order id.
	if got := records[1][0]; got != "RZ-ORD-\n001" {
		t.Fatalf("embedded-newline cell = %q, want %q", got, "RZ-ORD-\n001")
	}
	if got := records[2][0]; got != "RZ-ORD-002" {
		t.Fatalf("cell = %q, want %q", got, "RZ-ORD-002")
	}
}

func TestReadRows_CSVKeepsRawLeadingApostrophe(t *testing.T) {
	data := fixture(t, "rouzao_shipment_return.csv")
	records, err := ReadRows(data, FormatCSV, "")
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if got := records[1][10]; got != "'YT1234567890" {
		t.Fatalf("tracking cell = %q, want quoted-apostrophe preserved", got)
	}
}

func TestReadRows_PositionalCSVHasNoHeaderHandling(t *testing.T) {
	data := fixture(t, "bilibili_membership_positional.csv")
	records, err := ReadRows(data, FormatCSV, "")
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("records = %d, want 3", len(records))
	}
	if records[0][0] != "总督" || records[0][1] != "uid-10001" || records[0][2] != "DisplayA" {
		t.Fatalf("first record = %v", records[0])
	}
}

func TestReadRows_XLSXRoundTrip(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if err := f.SetSheetRow(sheet, "A1", &[]string{"订单号", "快递公司编码"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow(sheet, "A2", &[]string{"260505170605000460\n", "shunfeng"}); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	records, err := ReadRows(buf.Bytes(), FormatXLSX, "")
	if err != nil {
		t.Fatalf("read xlsx: %v", err)
	}
	if len(records) != 2 || records[1][0] != "260505170605000460\n" || records[1][1] != "shunfeng" {
		t.Fatalf("records = %v", records)
	}
}

func TestReadRows_XLSXMissingSheetErrors(t *testing.T) {
	f := excelize.NewFile()
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRows(buf.Bytes(), FormatXLSX, "不存在"); err == nil {
		t.Fatal("expected error for missing sheet")
	}
}

func TestReadRows_XLSFixture(t *testing.T) {
	data := fixture(t, "bilibili_carrier_codes.xls")
	records, err := ReadRows(data, FormatXLS, "")
	if err != nil {
		t.Fatalf("read xls: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("records = %d, want 3", len(records))
	}
	if records[0][0] != "快递公司名称" || records[0][1] != "快递公司编码" {
		t.Fatalf("header = %v", records[0])
	}
	if records[1][0] != "顺丰" || records[1][1] != "shunfeng" {
		t.Fatalf("row 1 = %v", records[1])
	}
	if records[2][0] != "圆通" || records[2][1] != "yto" {
		t.Fatalf("row 2 = %v", records[2])
	}
}

func TestReadRows_GarbageXLSErrorsNotPanics(t *testing.T) {
	if _, err := ReadRows([]byte("not an ole file at all"), FormatXLS, ""); err == nil {
		t.Fatal("expected error for non-xls bytes")
	}
}

func TestReadRows_EmptyAndUnsupported(t *testing.T) {
	if _, err := ReadRows(nil, FormatCSV, ""); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := ReadRows([]byte("a,b\n1,2\n"), "pdf", ""); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestFormatFromExtension(t *testing.T) {
	cases := map[string]string{
		".csv":  FormatCSV,
		".CSV":  FormatCSV,
		"xlsx":  FormatXLSX,
		".XLS":  FormatXLS,
		".docx": "",
		"":      "",
	}
	for ext, want := range cases {
		if got := FormatFromExtension(ext); got != want {
			t.Fatalf("FormatFromExtension(%q) = %q, want %q", ext, got, want)
		}
	}
}

func TestReadRows_CSVWriterCompatibility(t *testing.T) {
	// Whatever Go's csv writer produces must read back byte-honestly,
	// including values that carry their own quotes.
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"a", `say "hi"`, "x,y"}); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	records, err := ReadRows(buf.Bytes(), FormatCSV, "")
	if err != nil {
		t.Fatal(err)
	}
	if records[0][1] != `say "hi"` || records[0][2] != "x,y" {
		t.Fatalf("records = %v", records)
	}
}
