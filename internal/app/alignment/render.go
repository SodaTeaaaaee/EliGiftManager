package alignment

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/csvformula"
	"github.com/xuri/excelize/v2"
)

// Render writes semantic rows into a platform file following LayoutConfig:
// ColumnOrder fixes the column sequence, HeaderNames renames columns
// (semantic key by default). CSV output starts with a UTF-8 BOM and every
// cell passes csvformula.Sanitize; the same sanitizing applies to xlsx cells
// so "=..." text cannot become a live formula there either. Empty rows still
// produce a header-only file.
func Render(rows []map[string]string, layout LayoutConfig) ([]byte, error) {
	if layout.Format == "" {
		return nil, fmt.Errorf("alignment: render: format is empty")
	}
	if len(layout.ColumnOrder) == 0 {
		return nil, fmt.Errorf("alignment: render: column order is empty")
	}
	header := make([]string, len(layout.ColumnOrder))
	for i, key := range layout.ColumnOrder {
		name, ok := layout.HeaderNames[key]
		if !ok || name == "" {
			name = key
		}
		header[i] = name
	}
	records := make([][]string, 0, len(rows)+1)
	records = append(records, header)
	for _, row := range rows {
		record := make([]string, len(layout.ColumnOrder))
		for i, key := range layout.ColumnOrder {
			record[i] = row[key]
		}
		records = append(records, record)
	}

	switch layout.Format {
	case FormatCSV:
		return renderCSV(records)
	case FormatXLSX:
		return renderXLSX(records)
	default:
		return nil, fmt.Errorf("alignment: render: unsupported format %q", layout.Format)
	}
}

func renderCSV(records [][]string) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	w := csv.NewWriter(&buf)
	for _, record := range records {
		sanitized := make([]string, len(record))
		for i, c := range record {
			sanitized[i] = csvformula.Sanitize(c)
		}
		if err := w.Write(sanitized); err != nil {
			return nil, fmt.Errorf("alignment: render csv: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("alignment: render csv: %w", err)
	}
	return buf.Bytes(), nil
}

func renderXLSX(records [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	for r, record := range records {
		sanitized := make([]string, len(record))
		for i, c := range record {
			sanitized[i] = csvformula.Sanitize(c)
		}
		cell, err := excelize.CoordinatesToCellName(1, r+1)
		if err != nil {
			return nil, fmt.Errorf("alignment: render xlsx: %w", err)
		}
		if err := f.SetSheetRow(sheet, cell, &sanitized); err != nil {
			return nil, fmt.Errorf("alignment: render xlsx: %w", err)
		}
	}
	out, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("alignment: render xlsx: %w", err)
	}
	return out.Bytes(), nil
}

// RenderPreviewString renders a small preview as display text (header row
// plus pipe-joined data rows), used by template testing surfaces.
func RenderPreviewString(rows []map[string]string, layout LayoutConfig, limit int) (string, error) {
	if limit >= 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	header := make([]string, len(layout.ColumnOrder))
	for i, key := range layout.ColumnOrder {
		name, ok := layout.HeaderNames[key]
		if !ok || name == "" {
			name = key
		}
		header[i] = name
	}
	lines := []string{strings.Join(header, " | ")}
	for _, row := range rows {
		record := make([]string, len(layout.ColumnOrder))
		for i, key := range layout.ColumnOrder {
			record[i] = row[key]
		}
		lines = append(lines, strings.Join(record, " | "))
	}
	return strings.Join(lines, "\n"), nil
}
