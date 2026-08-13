package tabular

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/extrame/xls"
)

func readXLS(path string, opts ReadOptions) (*Sheet, error) {
	return readXLSWithLimits(path, opts, MaxFileBytes, MaxRows)
}

func readXLSWithLimits(path string, opts ReadOptions, maxBytes int64, maxRows int) (*Sheet, error) {
	if err := checkFileByteLimit(path, maxBytes); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("open xls file %q: %w", path, err)
		}
		return nil, fmt.Errorf("xls file %q: %w", path, err)
	}

	enc := normalizeXLSEncoding(opts.Encoding)
	// extrame/xls passes charset into ole2, which currently ignores it; cell
	// strings are decoded below like CSV auto (UTF-8, else GBK). Still pass GBK
	// when the caller asked for it so we do not hard-code utf-8 only.
	wb, err := xls.Open(path, xlsOpenCharset(enc))
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
			decodedName := decodeXLSCell(candidate.Name, enc)
			available = append(available, decodedName)
			if decodedName == name || candidate.Name == name {
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

	nRecords := int(ws.MaxRow) + 1
	if maxRows > 0 && nRecords > maxRows {
		return nil, errRowLimit(path, nRecords, maxRows)
	}

	// Discover the max column width so short rows pad consistently enough for
	// positional mapping; trailing empties stay empty strings.
	maxCol := 0
	records := make([][]string, 0, nRecords)
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
			cells[c] = decodeXLSCell(row.Col(c), enc)
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

func normalizeXLSEncoding(encoding string) string {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	if enc == "" {
		return "auto"
	}
	return enc
}

func xlsOpenCharset(encoding string) string {
	switch encoding {
	case "gbk", "gb18030":
		return "gbk"
	default:
		return "utf-8"
	}
}

// decodeXLSCell mirrors CSV auto-detect: valid UTF-8 is kept; otherwise GBK
// (common for Chinese Excel 97-2003 / BIFF5). BIFF8 compressed 8-bit strings
// are often misread as Latin-1; those are recovered when GBK yields Han.
func decodeXLSCell(s, encoding string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	switch encoding {
	case "utf-8", "utf8":
		return s
	}

	raw := []byte(s)
	if !utf8.Valid(raw) {
		decoded, err := decodeGBK(raw)
		if err == nil {
			return decoded
		}
		return s
	}
	latin1, ok := highLatin1Bytes(s)
	if !ok {
		return s
	}
	decoded, err := decodeGBK(latin1)
	if err == nil && containsHan(decoded) && !containsHan(s) {
		return decoded
	}
	return s
}

func highLatin1Bytes(s string) ([]byte, bool) {
	out := make([]byte, 0, len(s))
	high := false
	for _, r := range s {
		if r > 0xFF {
			return nil, false
		}
		if r >= 0x80 {
			high = true
		}
		out = append(out, byte(r))
	}
	if !high {
		return nil, false
	}
	return out, true
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
