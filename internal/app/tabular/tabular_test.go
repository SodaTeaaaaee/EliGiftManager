package tabular

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestReadTabularFile_CSV_BOMAndRagged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "import.csv")
	// Leading UTF-8 BOM; second data row is short; third is long.
	content := "\ufeffName,Qty\nStandee,2\nPoster\nSticker,5,extra\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sheet, err := ReadTabularFile(path, ReadOptions{Format: "csv", HasHeader: true, Encoding: "auto"})
	if err != nil {
		t.Fatalf("ReadTabularFile: %v", err)
	}
	if len(sheet.Headers) != 2 || sheet.Headers[0] != "Name" || sheet.Headers[1] != "Qty" {
		t.Fatalf("headers: %+v", sheet.Headers)
	}
	if len(sheet.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(sheet.Rows))
	}

	keyed := sheet.HeaderKeyedRows()
	if keyed[0]["Name"] != "Standee" || keyed[0]["Qty"] != "2" {
		t.Errorf("row0: %+v", keyed[0])
	}
	if keyed[1]["Name"] != "Poster" {
		t.Errorf("row1: %+v", keyed[1])
	}
	if _, ok := keyed[1]["Qty"]; ok {
		t.Errorf("short row should omit Qty, got %+v", keyed[1])
	}
	if keyed[2]["Name"] != "Sticker" || keyed[2]["Qty"] != "5" {
		t.Errorf("row2: %+v", keyed[2])
	}
}

func TestReadTabularFile_CSV_EmptyErrors(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadTabularFile(path, ReadOptions{Format: "csv", HasHeader: true}); err == nil {
		t.Fatal("expected error for empty csv")
	}
}

func TestReadTabularFile_CSV_GBKFallback(t *testing.T) {
	t.Parallel()
	// "姓名,数量\n礼物,1\n" encoded as GBK (not valid UTF-8).
	gbkBytes, err := encodeGBK("姓名,数量\n礼物,1\n")
	if err != nil {
		t.Fatalf("encode gbk: %v", err)
	}
	path := filepath.Join(t.TempDir(), "gbk.csv")
	if err := os.WriteFile(path, gbkBytes, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	sheet, err := ReadTabularFile(path, ReadOptions{Format: "csv", HasHeader: true, Encoding: "auto"})
	if err != nil {
		t.Fatalf("ReadTabularFile: %v", err)
	}
	if len(sheet.Headers) != 2 || sheet.Headers[0] != "姓名" || sheet.Headers[1] != "数量" {
		t.Fatalf("headers: %+v", sheet.Headers)
	}
	if len(sheet.Rows) != 1 || sheet.Rows[0][0] != "礼物" {
		t.Fatalf("rows: %+v", sheet.Rows)
	}
}

func TestReadTabularFile_XLSX(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sample.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheetName := f.GetSheetName(0)
	if err := f.SetSheetRow(sheetName, "A1", &[]any{"Name", "Qty"}); err != nil {
		t.Fatalf("header: %v", err)
	}
	if err := f.SetSheetRow(sheetName, "A2", &[]any{"Standee", "2"}); err != nil {
		t.Fatalf("row: %v", err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	sheet, err := ReadTabularFile(path, ReadOptions{Format: "xlsx", HasHeader: true})
	if err != nil {
		t.Fatalf("ReadTabularFile: %v", err)
	}
	if len(sheet.Headers) != 2 || sheet.Headers[0] != "Name" || sheet.Headers[1] != "Qty" {
		t.Fatalf("headers: %+v", sheet.Headers)
	}
	if len(sheet.Rows) != 1 || sheet.Rows[0][0] != "Standee" || sheet.Rows[0][1] != "2" {
		t.Fatalf("rows: %+v", sheet.Rows)
	}

	// Extension-based detection with empty Format.
	sheet2, err := ReadTabularFile(path, ReadOptions{HasHeader: true})
	if err != nil {
		t.Fatalf("detect format: %v", err)
	}
	if sheet2.Headers[0] != "Name" {
		t.Fatalf("detect headers: %+v", sheet2.Headers)
	}
}

func encodeGBK(s string) ([]byte, error) {
	out, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(s))
	return out, err
}
