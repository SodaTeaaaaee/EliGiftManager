package tabular

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func readXLSX(path string, opts ReadOptions) (*Sheet, error) {
	return readXLSXWithLimits(path, opts, MaxFileBytes, MaxRows, MaxUnzipBytes, MaxUnzipXMLBytes)
}

func readXLSXWithLimits(path string, opts ReadOptions, maxBytes int64, maxRows int, unzipLimit, unzipXMLLimit int64) (*Sheet, error) {
	if err := checkFileByteLimit(path, maxBytes); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("open xlsx file %q: %w", path, err)
		}
		return nil, fmt.Errorf("xlsx file %q: %w", path, err)
	}

	if unzipLimit <= 0 {
		unzipLimit = MaxUnzipBytes
	}
	if unzipXMLLimit <= 0 || unzipXMLLimit > unzipLimit {
		unzipXMLLimit = unzipLimit
	}

	f, err := excelize.OpenFile(path, excelize.Options{
		UnzipSizeLimit:    unzipLimit,
		UnzipXMLSizeLimit: unzipXMLLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("open xlsx file %q: %w", path, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx file %q has no sheets", path)
	}
	name := strings.TrimSpace(opts.SheetName)
	if name != "" {
		found := false
		for _, candidate := range sheets {
			if candidate == name {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("xlsx sheet %q not found (available: %s)", name, strings.Join(sheets, ", "))
		}
	} else {
		if opts.SheetIndex < 0 || opts.SheetIndex >= len(sheets) {
			return nil, fmt.Errorf("xlsx sheet index %d out of range [0,%d)", opts.SheetIndex, len(sheets))
		}
		name = sheets[opts.SheetIndex]
	}

	rows, err := getXLSXRowsLimited(f, name, maxRows)
	if err != nil {
		return nil, fmt.Errorf("read xlsx sheet %q: %w", name, err)
	}
	// excelize may omit trailing empty cells; normalise nothing here — callers
	// already tolerate ragged rows.
	normalized := make([][]string, len(rows))
	for i, row := range rows {
		normalized[i] = append([]string(nil), row...)
		// Ensure no nil row slices.
		if normalized[i] == nil {
			normalized[i] = []string{}
		}
	}
	sheet, err := splitHeaderRows(normalized, opts.HasHeader)
	if err != nil {
		return nil, fmt.Errorf("xlsx file %q: %w", path, err)
	}
	// Strip BOM from first header if present (rare for xlsx but harmless).
	if len(sheet.Headers) > 0 {
		sheet.Headers[0] = strings.TrimPrefix(sheet.Headers[0], "\ufeff")
	}
	return sheet, nil
}

// getXLSXRowsLimited mirrors excelize.GetRows (including interstitial blank
// rows) but stops with a clear error when the row count would exceed maxRows.
func getXLSXRowsLimited(f *excelize.File, sheet string, maxRows int) ([][]string, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, cur, maxVal := make([][]string, 0, 64), 0, 0
	for rows.Next() {
		cur++
		if maxRows > 0 && cur > maxRows {
			return nil, errRowLimit(sheet, cur, maxRows)
		}
		row, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		if len(row) > 0 {
			if emptyRows := cur - maxVal - 1; emptyRows > 0 {
				results = append(results, make([][]string, emptyRows)...)
			}
			results = append(results, row)
			maxVal = cur
		}
	}
	if err := rows.Error(); err != nil {
		return nil, err
	}
	if maxRows > 0 && len(results) > maxRows {
		return nil, errRowLimit(sheet, len(results), maxRows)
	}
	return results, nil
}
