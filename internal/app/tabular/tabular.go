// Package tabular provides a unified reader for CSV / XLSX / XLS sheet data.
package tabular

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Read limits applied by ReadTabularFile. Sized for desktop catalog/demand
// imports: large enough for real workbooks, small enough to bound memory.
const (
	// MaxFileBytes is the maximum on-disk size accepted (32 MiB).
	MaxFileBytes int64 = 32 << 20
	// MaxRows is the maximum number of records accepted, including the header
	// row when HasHeader is set.
	MaxRows = 100_000
	// MaxUnzipBytes is excelize UnzipSizeLimit: decompressed xlsx cap (64 MiB).
	MaxUnzipBytes int64 = 64 << 20
	// MaxUnzipXMLBytes is excelize UnzipXMLSizeLimit per worksheet / shared
	// strings part (16 MiB). Must be <= MaxUnzipBytes.
	MaxUnzipXMLBytes int64 = 16 << 20
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
	// Encoding controls CSV and XLS text decoding. "auto" (or empty) tries UTF-8
	// (with BOM) then falls back to GBK. Explicit values: "utf-8", "gbk".
	// XLSX is Unicode and ignores Encoding.
	Encoding string
}

// Sheet is a uniform tabular representation shared by all adapters.
type Sheet struct {
	Headers []string
	Rows    [][]string
}

// HeaderKeyedRows converts each data row into a header→value map.
//
// Short rows omit keys for missing trailing cells (unchanged).
//
// If a row has more cells than Headers and those extra cells are non-empty,
// they are kept under synthetic keys "_extra_N" where N is the 0-based column
// index (e.g. a third cell with two headers is "_extra_2"). Empty extra cells
// are omitted. Sheet.Headers is not mutated, so positional callers that use
// Headers + Rows stay stable. Normal header-aligned rows are unchanged.
func (s *Sheet) HeaderKeyedRows() []map[string]string {
	if s == nil {
		return nil
	}
	headerN := len(s.Headers)
	out := make([]map[string]string, 0, len(s.Rows))
	for _, record := range s.Rows {
		row := make(map[string]string, headerN)
		n := headerN
		if len(record) < n {
			n = len(record)
		}
		for i := 0; i < n; i++ {
			row[s.Headers[i]] = record[i]
		}
		for i := headerN; i < len(record); i++ {
			if record[i] == "" {
				continue
			}
			row[fmt.Sprintf("_extra_%d", i)] = record[i]
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

func checkFileByteLimit(path string, limit int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > limit {
		return errByteLimit(path, info.Size(), limit)
	}
	return nil
}

func errByteLimit(path string, size, limit int64) error {
	return fmt.Errorf("file %q is %d bytes, exceeds the %d-byte limit", path, size, limit)
}

func errRowLimit(path string, got, limit int) error {
	return fmt.Errorf("file %q has %d rows, exceeds the %d-row limit", path, got, limit)
}
