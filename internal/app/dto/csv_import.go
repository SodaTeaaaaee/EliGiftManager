package dto

// CSVFilePreviewDTO is the parsed content of a locally-picked CSV file.
// Headers holds the first record; Rows holds every subsequent record as a
// header-keyed map, so the frontend can slice the first few rows for preview
// while reusing the same parsed data for the real import (single file read).
type CSVFilePreviewDTO struct {
	Headers []string            `json:"headers"`
	Rows    []map[string]string `json:"rows"`
}
