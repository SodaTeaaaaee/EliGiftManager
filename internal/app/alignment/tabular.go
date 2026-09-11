// Package alignment is the external semantic alignment engine: it reads
// tabular files, applies versioned mapping templates (schema v3), and renders
// output files. Everything here is pure over bytes and configs; persistence
// and workspace concerns stay in the app layer.
package alignment

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

// Supported tabular formats.
const (
	FormatCSV  = "csv"
	FormatXLSX = "xlsx"
	FormatXLS  = "xls"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// FormatFromExtension maps a file extension (with or without the leading dot,
// any case) to a format constant. Unknown extensions return "".
func FormatFromExtension(ext string) string {
	e := strings.ToLower(strings.TrimSpace(ext))
	e = strings.TrimPrefix(e, ".")
	switch e {
	case "csv":
		return FormatCSV
	case "xlsx":
		return FormatXLSX
	case "xls":
		return FormatXLS
	default:
		return ""
	}
}

// ReadRows reads every record of a tabular file into memory. Header handling
// is a mapping decision, not a reader decision: the returned records include
// the header row when the file has one, and Parse decides whether record 0 is
// a header. sheetName selects the sheet for xlsx (empty means the first
// sheet); it is ignored for csv and xls.
func ReadRows(data []byte, format, sheetName string) ([][]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("alignment: read %s: file is empty", format)
	}
	switch format {
	case FormatCSV:
		return readCSV(data)
	case FormatXLSX:
		return readXLSX(data, sheetName)
	case FormatXLS:
		return readXLS(data)
	case "":
		return nil, fmt.Errorf("alignment: read: format is empty")
	default:
		return nil, fmt.Errorf("alignment: read: unsupported format %q", format)
	}
}

// ListSheets returns the sheet names of an xlsx workbook in workbook order.
// csv and xls files have no selectable sheets and yield an empty list (xls
// always reads its first sheet).
func ListSheets(data []byte, format string) ([]string, error) {
	if format != FormatXLSX {
		return []string{}, nil
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("alignment: read xlsx: file is empty")
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("alignment: read xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()
	return append([]string{}, f.GetSheetList()...), nil
}

// readCSV decodes UTF-8 CSV only; a leading UTF-8 BOM is tolerated and
// stripped. Known limitation: files in other encodings (GBK, BIG5, ...) are
// not transcoded and must be re-saved as UTF-8 before import, otherwise the
// bytes garble into invalid header and value text.
func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, utf8BOM)
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("alignment: read csv: %w", err)
	}
	return records, nil
}

func readXLSX(data []byte, sheetName string) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("alignment: read xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()
	sheet := strings.TrimSpace(sheetName)
	if sheet == "" {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("alignment: read xlsx: workbook has no sheets")
		}
		sheet = sheets[0]
	} else if !sheetExists(f, sheet) {
		return nil, fmt.Errorf("alignment: read xlsx: sheet %q not found", sheetName)
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("alignment: read xlsx sheet %q: %w", sheet, err)
	}
	return rows, nil
}

func sheetExists(f *excelize.File, sheet string) bool {
	for _, s := range f.GetSheetList() {
		if s == sheet {
			return true
		}
	}
	return false
}

// readXLS goes through extrame/xls, which can panic on malformed BIFF
// streams, so the parse runs under a recover guard.
func readXLS(data []byte) (records [][]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			records = nil
			err = fmt.Errorf("alignment: read xls: parser panic: %v", r)
		}
	}()
	wb, err := xls.OpenReader(bytes.NewReader(data), "utf-8")
	if err != nil {
		return nil, fmt.Errorf("alignment: read xls: %w", err)
	}
	if wb.NumSheets() == 0 {
		return nil, fmt.Errorf("alignment: read xls: workbook has no sheets")
	}
	sheet := wb.GetSheet(0)
	if sheet == nil {
		return nil, fmt.Errorf("alignment: read xls: first sheet is unreadable")
	}
	for i := 0; i <= int(sheet.MaxRow); i++ {
		row := sheet.Row(i)
		if row == nil {
			records = append(records, nil)
			continue
		}
		n := row.LastCol()
		if n < 0 {
			n = 0
		}
		cells := make([]string, n)
		for c := 0; c < n; c++ {
			cells[c] = row.Col(c)
		}
		records = append(records, cells)
	}
	return records, nil
}

// cell returns record[i] or "" when the record is shorter, so positional
// mappings over ragged rows degrade to empty values instead of panicking.
func cell(record []string, i int) string {
	if i < 0 || i >= len(record) {
		return ""
	}
	return record[i]
}
