package tabular

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

func TestReadCSV_MaxBytesExceeded(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "oversize.csv")
	if err := os.WriteFile(path, []byte("Name,Qty\nA,1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := readCSVWithLimits(path, ReadOptions{HasHeader: true}, 8, MaxRows); err == nil {
		t.Fatal("expected max-bytes error")
	} else if !strings.Contains(err.Error(), "byte") || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention byte limit, got %v", err)
	}

	// Public API: sparse file larger than MaxFileBytes is rejected before parse.
	big := filepath.Join(t.TempDir(), "big.csv")
	if err := os.WriteFile(big, []byte("h\n1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Truncate(big, MaxFileBytes+1); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := ReadTabularFile(big, ReadOptions{Format: "csv", HasHeader: true}); err == nil {
		t.Fatal("expected max-bytes error from ReadTabularFile")
	} else if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error: %v", err)
	}
}

func TestReadCSV_MaxRowsExceeded(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "many.csv")
	content := "h\na\nb\nc\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := readCSVWithLimits(path, ReadOptions{HasHeader: true}, MaxFileBytes, 3); err == nil {
		t.Fatal("expected max-rows error")
	} else if !strings.Contains(err.Error(), "row") || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention row limit, got %v", err)
	}

	// Exactly at the limit still succeeds (header + 2 data rows = 3 records).
	sheet, err := readCSVWithLimits(path, ReadOptions{HasHeader: true}, MaxFileBytes, 4)
	if err != nil {
		t.Fatalf("at-limit should succeed: %v", err)
	}
	if len(sheet.Rows) != 3 {
		t.Fatalf("rows: %d", len(sheet.Rows))
	}
}

func TestReadXLSX_MaxBytesExceeded(t *testing.T) {
	t.Parallel()
	path := writeSampleXLSX(t, [][]any{{"Name", "Qty"}, {"Standee", "2"}})
	if _, err := readXLSXWithLimits(path, ReadOptions{HasHeader: true}, 32, MaxRows, MaxUnzipBytes, MaxUnzipXMLBytes); err == nil {
		t.Fatal("expected max-bytes error")
	} else if !strings.Contains(err.Error(), "byte") || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention byte limit, got %v", err)
	}

	big := filepath.Join(t.TempDir(), "big.xlsx")
	if err := os.WriteFile(big, []byte("PK"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Truncate(big, MaxFileBytes+1); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := ReadTabularFile(big, ReadOptions{Format: "xlsx", HasHeader: true}); err == nil {
		t.Fatal("expected max-bytes error from ReadTabularFile")
	}
}

func TestReadXLSX_MaxRowsExceeded(t *testing.T) {
	t.Parallel()
	path := writeSampleXLSX(t, [][]any{
		{"Name", "Qty"},
		{"A", "1"},
		{"B", "2"},
		{"C", "3"},
	})
	if _, err := readXLSXWithLimits(path, ReadOptions{HasHeader: true}, MaxFileBytes, 3, MaxUnzipBytes, MaxUnzipXMLBytes); err == nil {
		t.Fatal("expected max-rows error")
	} else if !strings.Contains(err.Error(), "row") || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention row limit, got %v", err)
	}
}

func TestReadXLSX_UnzipSizeLimit(t *testing.T) {
	t.Parallel()
	path := writeSampleXLSX(t, [][]any{{"Name", "Qty"}, {"Standee", "2"}})
	if _, err := readXLSXWithLimits(path, ReadOptions{HasHeader: true}, MaxFileBytes, MaxRows, 100, 100); err == nil {
		t.Fatal("expected unzip size limit error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "unzip") &&
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error should mention unzip/size limit, got %v", err)
	}
}

func TestDecodeXLSCell_GBKFallback(t *testing.T) {
	t.Parallel()
	const want = "姓名"

	gbkBytes, err := encodeGBK(want)
	if err != nil {
		t.Fatalf("encode gbk: %v", err)
	}
	if utf8.Valid(gbkBytes) {
		t.Fatal("fixture must not be valid utf-8")
	}

	// BIFF5-style: raw codepage bytes stuffed into a Go string.
	got := decodeXLSCell(string(gbkBytes), "auto")
	if got != want {
		t.Fatalf("raw GBK bytes: got %q want %q", got, want)
	}

	// BIFF8 compressed 8-bit: library expands each byte as Latin-1.
	latin1 := latin1FromBytes(gbkBytes)
	if !utf8.ValidString(latin1) {
		t.Fatal("latin-1 expansion should be valid utf-8")
	}
	got = decodeXLSCell(latin1, "auto")
	if got != want {
		t.Fatalf("latin-1 GBK: got %q want %q", got, want)
	}

	// Already-correct Unicode (BIFF8 UTF-16 path) must be left alone.
	if got := decodeXLSCell(want, "auto"); got != want {
		t.Fatalf("utf-8 chinese: got %q", got)
	}
	if got := decodeXLSCell("Standee", "auto"); got != "Standee" {
		t.Fatalf("ascii: got %q", got)
	}

	// Explicit utf-8 does not GBK-fallback (mirrors CSV).
	if got := decodeXLSCell(string(gbkBytes), "utf-8"); got == want {
		t.Fatal("utf-8 mode should not GBK-decode")
	}
}

func TestHeaderKeyedRows_KeepsExtraValuedCells(t *testing.T) {
	t.Parallel()
	sheet := &Sheet{
		Headers: []string{"Name", "Qty"},
		Rows: [][]string{
			{"Standee", "2"},
			{"Poster"},
			{"Sticker", "5", "overflow", ""},
			{"Badge", "1", "", "kept"},
		},
	}
	keyed := sheet.HeaderKeyedRows()
	if len(keyed) != 4 {
		t.Fatalf("len=%d", len(keyed))
	}
	if keyed[0]["Name"] != "Standee" || keyed[0]["Qty"] != "2" {
		t.Errorf("aligned row: %+v", keyed[0])
	}
	if _, ok := keyed[0]["_extra_2"]; ok {
		t.Errorf("aligned row should not grow extras: %+v", keyed[0])
	}
	if keyed[1]["Name"] != "Poster" {
		t.Errorf("short row: %+v", keyed[1])
	}
	if _, ok := keyed[1]["Qty"]; ok {
		t.Errorf("short row should omit Qty, got %+v", keyed[1])
	}
	if keyed[2]["Name"] != "Sticker" || keyed[2]["Qty"] != "5" {
		t.Errorf("row2 headers: %+v", keyed[2])
	}
	if keyed[2]["_extra_2"] != "overflow" {
		t.Errorf("row2 extra: %+v", keyed[2])
	}
	if _, ok := keyed[2]["_extra_3"]; ok {
		t.Errorf("empty extra cell must be omitted: %+v", keyed[2])
	}
	if keyed[3]["_extra_3"] != "kept" {
		t.Errorf("row3 extra col 3: %+v", keyed[3])
	}
	if _, ok := keyed[3]["_extra_2"]; ok {
		t.Errorf("empty extra col 2 must be omitted: %+v", keyed[3])
	}
	if len(sheet.Headers) != 2 {
		t.Fatalf("Headers must not be mutated, got %+v", sheet.Headers)
	}
}

func TestHeaderKeyedRows_NoHeadersKeepsValuedCells(t *testing.T) {
	t.Parallel()
	sheet := &Sheet{
		Headers: nil,
		Rows:    [][]string{{"a", "", "c"}},
	}
	keyed := sheet.HeaderKeyedRows()
	if keyed[0]["_extra_0"] != "a" || keyed[0]["_extra_2"] != "c" {
		t.Fatalf("got %+v", keyed[0])
	}
	if _, ok := keyed[0]["_extra_1"]; ok {
		t.Fatalf("empty extra omitted: %+v", keyed[0])
	}
}

func writeSampleXLSX(t *testing.T, rows [][]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheetName := f.GetSheetName(0)
	for i, row := range rows {
		row := row
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			t.Fatalf("cell name: %v", err)
		}
		if err := f.SetSheetRow(sheetName, cell, &row); err != nil {
			t.Fatalf("row %d: %v", i, err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	return path
}

func latin1FromBytes(raw []byte) string {
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return string(runes)
}
