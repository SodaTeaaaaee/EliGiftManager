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
	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeProductFieldName)
	if err != nil {
		return nil, fmt.Errorf("parse product CSV failed: %w", err)
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
	headerIndex, normalizedHeaders, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("parse product CSV failed: %w", err)
	}
	fieldColumns, consumedColumns, err := resolveFieldColumns(anyMapFromStringMap(mappingRules), headerIndex)
	if err != nil {
		return nil, fmt.Errorf("parse product CSV failed: %w", err)
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
		product := model.Product{Platform: template.Platform, Factory: template.Platform, ExtraData: "{}"}
		for fieldName, columnIndex := range fieldColumns {
			value := readCSVCell(record, columnIndex)
			switch fieldName {
			case "platform":
				product.Platform = value
			case "factory":
				product.Factory = value
			case "factory_sku":
				product.FactorySKU = value
			case "name":
				product.Name = value
			case "cover_image":
				product.CoverImage = value
			default:
				return nil, fmt.Errorf("parse product CSV failed: row %d has unsupported field %q", rowNumber, fieldName)
			}
		}
		if err := validateProductRecord(product, rowNumber); err != nil {
			return nil, fmt.Errorf("parse product CSV failed: %w", err)
		}
		extraDataJSON, err := buildExtraDataJSON(normalizedHeaders, record, consumedColumns)
		if err != nil {
			return nil, fmt.Errorf("parse product CSV failed: row %d build ExtraData failed: %w", rowNumber, err)
		}
		product.ExtraData = extraDataJSON
		products = append(products, product)
	}
	return products, nil
}

func normalizeProductFieldName(field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
	switch normalized {
	case "platform", "平台", "平台名", "平台名称":
		return "platform", nil
	case "factory", "工厂", "工厂名", "工厂名称", "供应商", "供应商名称":
		return "factory", nil
	case "factorysku", "sku", "工厂sku", "平台sku", "商品sku", "货号":
		return "factory_sku", nil
	case "name", "productname", "商品名", "商品名称", "产品名", "产品名称":
		return "name", nil
	case "coverimage", "imagepath", "image", "imageurl", "图片", "图片路径", "主图", "主图路径":
		return "cover_image", nil
	default:
		return "", fmt.Errorf("unsupported product field %q", field)
	}
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

// productZIPTemplateConfig 从 ZIP 模板 MappingRules 中提取 ZIP 格式特有字段。
type productZIPTemplateConfig struct {
	Format     string            `json:"format"`
	CSVPattern string            `json:"csvPattern"`
	ImageDir   string            `json:"imageDir"`
	Mapping    map[string]string `json:"mapping"`
}

// ParseProductZIP 解压 ZIP 文件，从中提取 CSV 并解析为 Product 列表。
//
// 流程：
//  1. 创建临时解压目录
//  2. archive/zip 解压
//  3. 从 MappingRules 读取 csvPattern（如 "*.csv"）
//  4. 找到匹配的 CSV 文件，调用现有 ParseProductCSV 逻辑
//  5. 返回 products + 解压目录路径（后续传给 ProcessCoverImages）
func ParseProductZIP(zipPath string, template model.TemplateConfig) ([]model.Product, string, error) {
	if strings.TrimSpace(zipPath) == "" {
		return nil, "", fmt.Errorf("parse product ZIP failed: ZIP path is required")
	}
	if err := ensureTemplateType(template, model.TemplateTypeImportProduct, "product import"); err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: %w", err)
	}

	// 解析模板配置
	var zipCfg productZIPTemplateConfig
	if err := json.Unmarshal([]byte(template.MappingRules), &zipCfg); err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: parse MappingRules: %w", err)
	}
	if zipCfg.CSVPattern == "" {
		zipCfg.CSVPattern = "*.csv"
	}
	if zipCfg.Mapping == nil {
		zipCfg.Mapping = map[string]string{}
	}

	// 创建受控的临时解压目录，避免把大体积导入残留到系统 Temp。
	extractDir, err := CreateImportTempDir()
	if err != nil {
		return nil, "", fmt.Errorf("parse product ZIP failed: %w", err)
	}

	// 解压
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
		// 防止目录遍历
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

	// 找到匹配的 CSV 文件
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

	// 构建一个临时 template 给 ParseProductCSV 使用（使用 ZIP 模板内的 mapping 段）
	csvTemplate := model.TemplateConfig{
		Platform: template.Platform,
		Type:     template.Type,
		Name:     template.Name,
	}
	if len(zipCfg.Mapping) > 0 {
		mappingJSON, _ := json.Marshal(zipCfg.Mapping)
		csvTemplate.MappingRules = string(mappingJSON)
	} else {
		csvTemplate.MappingRules = template.MappingRules
	}

	products, err := ParseProductCSV(csvPath, csvTemplate)
	if err != nil {
		os.RemoveAll(extractDir)
		return nil, "", fmt.Errorf("parse product ZIP failed: %w", err)
	}

	return products, extractDir, nil
}
