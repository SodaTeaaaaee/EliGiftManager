package tabular

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func readXLSX(path string, opts ReadOptions) (*Sheet, error) {
	f, err := excelize.OpenFile(path)
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

	rows, err := f.GetRows(name)
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
