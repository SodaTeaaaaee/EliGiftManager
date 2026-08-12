// Package tabular provides a unified reader for CSV / XLSX / XLS sheet data.
package tabular

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReadOptions controls how ReadTabularFile interprets an on-disk spreadsheet.
type ReadOptions struct {
	// Format is "" | "csv" | "xlsx" | "xls". Empty means detect from file extension.
	Format string
	// SheetIndex selects the zero-based sheet for multi-sheet formats (xlsx/xls).
	SheetIndex int
	// SheetName selects an exact sheet name for multi-sheet formats. When set it
	// takes precedence over SheetIndex.
	SheetName string
	// HasHeader treats the first row as column headers when true.
	HasHeader bool
	// Encoding controls CSV text decoding. "auto" (or empty) tries UTF-8 (with BOM)
	// then falls back to GBK. Explicit values: "utf-8", "gbk".
	Encoding string
}

// Sheet is a uniform tabular representation shared by all adapters.
type Sheet struct {
	Headers []string
	Rows    [][]string
}

// HeaderKeyedRows converts each data row into a header→value map.
// Cells beyond the header count are dropped; short rows omit trailing keys.
func (s *Sheet) HeaderKeyedRows() []map[string]string {
	if s == nil {
		return nil
	}
	out := make([]map[string]string, 0, len(s.Rows))
	for _, record := range s.Rows {
		row := make(map[string]string, len(s.Headers))
		n := len(s.Headers)
		if len(record) < n {
			n = len(record)
		}
		for i := 0; i < n; i++ {
			row[s.Headers[i]] = record[i]
		}
		out = append(out, row)
	}
	return out
}

// ReadTabularFile opens path and returns a Sheet according to opts.
func ReadTabularFile(path string, opts ReadOptions) (*Sheet, error) {
	format, err := resolveFormat(path, opts.Format)
	if err != nil {
		return nil, err
	}
	switch format {
	case "csv":
		return readCSV(path, opts)
	case "xlsx":
		return readXLSX(path, opts)
	case "xls":
		return readXLS(path, opts)
	default:
		return nil, fmt.Errorf("unsupported tabular format %q", format)
	}
}

func resolveFormat(path, format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "" {
		switch format {
		case "csv", "xlsx", "xls":
			return format, nil
		default:
			return "", fmt.Errorf("unsupported tabular format %q", format)
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return "csv", nil
	case ".xlsx":
		return "xlsx", nil
	case ".xls":
		return "xls", nil
	default:
		return "", fmt.Errorf("cannot detect tabular format from extension %q", ext)
	}
}

// splitHeaderRows peels the first record into Headers when HasHeader is set.
func splitHeaderRows(records [][]string, hasHeader bool) (*Sheet, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("tabular file is empty")
	}
	if !hasHeader {
		return &Sheet{Headers: nil, Rows: records}, nil
	}
	headers := append([]string(nil), records[0]...)
	if len(headers) == 0 {
		return nil, fmt.Errorf("tabular file has no headers")
	}
	// Strip a UTF-8 BOM that may cling to the first header cell (Excel CSV).
	if len(headers) > 0 {
		headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	}
	rows := records[1:]
	if rows == nil {
		rows = [][]string{}
	}
	return &Sheet{Headers: headers, Rows: rows}, nil
}
