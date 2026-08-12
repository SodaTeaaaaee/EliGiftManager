package tabular

import (
	"fmt"
	"strings"

	"github.com/extrame/xls"
)

func readXLS(path string, opts ReadOptions) (*Sheet, error) {
	wb, err := xls.Open(path, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("open xls file %q: %w", path, err)
	}
	if wb == nil {
		return nil, fmt.Errorf("open xls file %q: nil workbook", path)
	}
	sheetIndex := opts.SheetIndex
	if name := strings.TrimSpace(opts.SheetName); name != "" {
		sheetIndex = -1
		available := make([]string, 0, wb.NumSheets())
		for i := 0; i < wb.NumSheets(); i++ {
			candidate := wb.GetSheet(i)
			if candidate == nil {
				continue
			}
			available = append(available, candidate.Name)
			if candidate.Name == name {
				sheetIndex = i
			}
		}
		if sheetIndex < 0 {
			return nil, fmt.Errorf("xls sheet %q not found (available: %s)", name, strings.Join(available, ", "))
		}
	}
	if sheetIndex < 0 || sheetIndex >= wb.NumSheets() {
		return nil, fmt.Errorf("xls sheet index %d out of range [0,%d)", sheetIndex, wb.NumSheets())
	}
	ws := wb.GetSheet(sheetIndex)
	if ws == nil {
		return nil, fmt.Errorf("xls file %q: sheet %d is nil", path, sheetIndex)
	}

	// Discover the max column width so short rows pad consistently enough for
	// positional mapping; trailing empties stay empty strings.
	maxCol := 0
	records := make([][]string, 0, int(ws.MaxRow)+1)
	for i := 0; i <= int(ws.MaxRow); i++ {
		row := ws.Row(i)
		if row == nil {
			records = append(records, []string{})
			continue
		}
		last := row.LastCol()
		if last > maxCol {
			maxCol = last
		}
		cells := make([]string, last)
		for c := 0; c < last; c++ {
			cells[c] = row.Col(c)
		}
		records = append(records, cells)
	}
	// Pad every row to maxCol so positional indexes are stable across rows.
	if maxCol > 0 {
		for i, row := range records {
			if len(row) < maxCol {
				padded := make([]string, maxCol)
				copy(padded, row)
				records[i] = padded
			}
		}
	}

	sheet, err := splitHeaderRows(records, opts.HasHeader)
	if err != nil {
		return nil, fmt.Errorf("xls file %q: %w", path, err)
	}
	if len(sheet.Headers) > 0 {
		sheet.Headers[0] = strings.TrimPrefix(sheet.Headers[0], "\ufeff")
	}
	return sheet, nil
}
