package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// ParseProductFromPath is the unified entry point for product import. It detects
// the source type by examining path (CSV file, ZIP/tar archive, or directory),
// extracts archives when needed, locates a CSV file inside, and delegates to
// ParseProductCSV for the actual parsing.
//
// Returns the parsed products and the extraction directory (empty string for
// direct CSV imports where no extraction occurred).
func ParseProductFromPath(path string, template model.TemplateConfig) ([]model.Product, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, "", fmt.Errorf("parse product failed: path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("parse product failed: %w", err)
	}

	if info.IsDir() {
		products, err := parseProductFromDir(path, template)
		return products, path, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		products, err := ParseProductCSV(path, template)
		return products, "", err
	case ".zip", ".tar":
		return parseProductFromArchive(path, template)
	default:
		base := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(base, ".tar.gz") || strings.HasSuffix(base, ".tgz") ||
			strings.HasSuffix(base, ".tar.bz2") || strings.HasSuffix(base, ".tar.xz") {
			return parseProductFromArchive(path, template)
		}
		return nil, "", fmt.Errorf("parse product failed: unsupported file type %q", ext)
	}
}

func parseProductFromArchive(archivePath string, template model.TemplateConfig) ([]model.Product, string, error) {
	extractDir, err := ExtractArchive(archivePath)
	if err != nil {
		return nil, "", fmt.Errorf("parse product from archive failed: %w", err)
	}

	products, err := parseProductFromDir(extractDir, template)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", err
	}

	return products, extractDir, nil
}

func parseProductFromDir(dir string, template model.TemplateConfig) ([]model.Product, error) {
	csvPath, err := FindCSVInDir(dir, "")
	if err != nil {
		return nil, fmt.Errorf("parse product from dir failed: %w", err)
	}

	// Build a csv-format copy of the template for ParseProductCSV (which ignores
	// format, but we set it to "csv" for correctness).
	csvTemplate := template
	var raw map[string]any
	if err := json.Unmarshal([]byte(template.MappingRules), &raw); err == nil {
		raw["format"] = model.TemplateFormatCSV
		if rulesJSON, jsonErr := json.Marshal(raw); jsonErr == nil {
			csvTemplate.MappingRules = string(rulesJSON)
		}
	}

	return ParseProductCSV(csvPath, csvTemplate)
}
