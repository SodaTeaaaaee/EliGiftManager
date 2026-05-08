package service

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func ParseProductCSV(csvFile string, template model.TemplateConfig) ([]model.Product, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("parse product CSV failed: CSV path is required")
	}
	if err := ensureTemplateType(template, model.TemplateTypeImportProduct, "product import"); err != nil {
		return nil, fmt.Errorf("parse product CSV failed: %w", err)
	}

	var rules model.DynamicTemplateRules
	if err := json.Unmarshal([]byte(template.MappingRules), &rules); err != nil {
		return nil, fmt.Errorf("parse product CSV failed: parse mapping rules: %w", err)
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("parse product CSV failed: open %q failed: %w", csvFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(stripBOM(file))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("parse product CSV failed: CSV is empty")
		}
		return nil, fmt.Errorf("parse product CSV failed: read header failed: %w", err)
	}

	products := make([]model.Product, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("parse product CSV failed: read row %d failed: %w", rowNumber, readErr)
		}
		if isEmptyRecord(record) {
			continue
		}

		coreData, extraData, parseErr := ParseRowDynamically(record, headers, rules)
		if parseErr != nil {
			return nil, fmt.Errorf("parse product CSV failed: row %d: %w", rowNumber, parseErr)
		}

		product := model.Product{
			Platform:   strings.TrimSpace(coreData["platform"]),
			Factory:    strings.TrimSpace(coreData["factory"]),
			FactorySKU: strings.TrimSpace(coreData["factory_sku"]),
			Name:       strings.TrimSpace(coreData["name"]),
			CoverImage: strings.TrimSpace(coreData["cover_image"]),
			ExtraData:  "{}",
		}

		// Fallback: platform and factory default to template.Platform when
		// the CSV column is absent or empty.
		if product.Platform == "" {
			product.Platform = template.Platform
		}
		if product.Factory == "" {
			product.Factory = template.Platform
		}

		if err := validateProductRecord(product, rowNumber); err != nil {
			return nil, fmt.Errorf("parse product CSV failed: %w", err)
		}

		// Serialize extra data to JSON.
		if len(extraData) > 0 {
			encoded, jsonErr := json.Marshal(extraData)
			if jsonErr != nil {
				return nil, fmt.Errorf("parse product CSV failed: row %d serialize extra data: %w", rowNumber, jsonErr)
			}
			product.ExtraData = string(encoded)
		}

		products = append(products, product)
	}
	return products, nil
}

func validateProductRecord(product model.Product, rowNumber int) error {
	if product.Platform == "" {
		return fmt.Errorf("row %d missing required field Platform", rowNumber)
	}
	if product.FactorySKU == "" {
		return fmt.Errorf("row %d missing required field FactorySKU", rowNumber)
	}
	if product.Name == "" {
		return fmt.Errorf("row %d missing required field Name", rowNumber)
	}
	return nil
}

// productZIPTemplateConfig extracts ZIP-format specific fields from the
// template MappingRules.  It embeds DynamicTemplateRules so that the common
// format/hasHeader/mapping/extraData fields are parsed once.
type productZIPTemplateConfig struct {
	model.DynamicTemplateRules
	CSVPattern string `json:"csvPattern"`
	ImageDir   string `json:"imageDir"`
}

// Deprecated: use ParseProductFromPath.
//
// ParseProductZIP 解压 ZIP 文件，从中提取 CSV 并解析为 Product 列表。
//
// 流程：
//  1. 创建临时解压目录
//  2. archive/zip 解压
//  3. 从 MappingRules 读取 csvPattern（如 "*.csv"）
//  4. 找到匹配的 CSV 文件，调用 ParseProductCSV
//  5. 返回 products + 解压目录路径（后续传给 ProcessCoverImages）
func ParseProductZIP(zipPath string, template model.TemplateConfig) ([]model.Product, string, error) {
	if strings.TrimSpace(zipPath) == "" {
		return nil, "", fmt.Errorf("parse product ZIP failed: ZIP path is required")
	}
	if err := ensureTemplateType(template, model.TemplateTypeImportProduct, "product import"); err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: %w", err)
	}

	// Parse the template JSON as productZIPTemplateConfig, which embeds
	// DynamicTemplateRules and adds csvPattern/imageDir.
	var zipCfg productZIPTemplateConfig
	if err := json.Unmarshal([]byte(template.MappingRules), &zipCfg); err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: parse MappingRules: %w", err)
	}
	if zipCfg.CSVPattern == "" {
		zipCfg.CSVPattern = "*.csv"
	}

	// Create temp extract directory.
	extractDir, err := os.MkdirTemp("", "eligift-product-zip-*")
	if err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: create temp dir: %w", err)
	}

	// Unzip.
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product ZIP failed: open %q: %w", zipPath, err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		if f.FileInfo().IsDir() {
			os.MkdirAll(filepath.Join(extractDir, f.Name), 0o755)
			continue
		}
		// Sanitize filename for Windows compatibility (* → _)
		destPath := filepath.Join(extractDir, strings.ReplaceAll(f.Name, "*", "_"))
		// Prevent directory traversal.
		cleanExtract, _ := filepath.Abs(extractDir)
		cleanDest, _ := filepath.Abs(destPath)
		if !strings.HasPrefix(cleanDest, cleanExtract+string(os.PathSeparator)) && cleanDest != cleanExtract {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			continue
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			continue
		}
	}

	// Find matching CSV file.
	csvPattern := zipCfg.CSVPattern
	matches, err := filepath.Glob(filepath.Join(extractDir, csvPattern))
	if err != nil || len(matches) == 0 {
		// Fallback: recursively find any .csv file
		tryPattern := filepath.Join(extractDir, "**", "*.csv")
		globFiles, globErr := filepath.Glob(tryPattern)
		if globErr != nil {
			matches, _ = filepath.Glob(filepath.Join(extractDir, "*.csv"))
		} else {
			matches = globFiles
		}
	}
	if len(matches) == 0 {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product ZIP failed: no CSV file matching %q found in archive", csvPattern)
	}
	csvPath := matches[0]

	// Build a temporary TemplateConfig for ParseProductCSV using the embedded
	// DynamicTemplateRules serialized as MappingRules JSON.
	// Override format to "csv" so that ParseProductCSV treats it correctly.
	zipCfg.Format = model.TemplateFormatCSV
	rulesJSON, _ := json.Marshal(zipCfg.DynamicTemplateRules)
	csvTemplate := model.TemplateConfig{
		Platform:     template.Platform,
		Type:         template.Type,
		Name:         template.Name,
		MappingRules: string(rulesJSON),
	}

	products, err := ParseProductCSV(csvPath, csvTemplate)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product ZIP failed: %w", err)
	}

	return products, extractDir, nil
}

// Deprecated: use ParseProductFromPath.
//
// ParseProductArchive extracts an archive (zip/tar/tar.gz), finds a CSV inside,
// and parses it as products. Returns products and the extraction directory.
func ParseProductArchive(archivePath string, template model.TemplateConfig) ([]model.Product, string, error) {
	extractDir, err := ExtractArchive(archivePath)
	if err != nil {
		return nil, "", fmt.Errorf("parse product archive failed: %w", err)
	}

	var zipCfg productZIPTemplateConfig
	if err := json.Unmarshal([]byte(template.MappingRules), &zipCfg); err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product archive failed: parse MappingRules: %w", err)
	}

	csvPath, err := FindCSVInDir(extractDir, zipCfg.CSVPattern)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product archive failed: %w", err)
	}

	zipCfg.Format = model.TemplateFormatCSV
	rulesJSON, _ := json.Marshal(zipCfg.DynamicTemplateRules)
	csvTemplate := model.TemplateConfig{
		Platform:     template.Platform,
		Type:         template.Type,
		Name:         template.Name,
		MappingRules: string(rulesJSON),
	}

	products, err := ParseProductCSV(csvPath, csvTemplate)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product archive failed: %w", err)
	}

	return products, extractDir, nil
}

// Deprecated: use ParseProductFromPath.
//
// ParseProductDir scans a directory for a CSV file and parses it as products.
// Returns products and the directory path (for subsequent image processing).
func ParseProductDir(dirPath string, template model.TemplateConfig) ([]model.Product, error) {
	var zipCfg productZIPTemplateConfig
	if err := json.Unmarshal([]byte(template.MappingRules), &zipCfg); err != nil {
		return nil, fmt.Errorf("parse product dir failed: parse MappingRules: %w", err)
	}

	csvPath, err := FindCSVInDir(dirPath, zipCfg.CSVPattern)
	if err != nil {
		return nil, fmt.Errorf("parse product dir failed: %w", err)
	}

	zipCfg.Format = model.TemplateFormatCSV
	rulesJSON, _ := json.Marshal(zipCfg.DynamicTemplateRules)
	csvTemplate := model.TemplateConfig{
		Platform:     template.Platform,
		Type:         template.Type,
		Name:         template.Name,
		MappingRules: string(rulesJSON),
	}

	products, err := ParseProductCSV(csvPath, csvTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse product dir failed: %w", err)
	}

	return products, nil
}
