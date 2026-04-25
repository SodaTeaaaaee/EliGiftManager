package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
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

	reader := csv.NewReader(file)
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
	fieldColumns, consumedColumns, err := resolveFieldColumns(mappingRules, headerIndex)
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
		product := model.Product{Platform: template.Platform, ExtraData: "{}"}
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
	if product.Factory == "" {
		return fmt.Errorf("row %d missing required field Factory", rowNumber)
	}
	if product.FactorySKU == "" {
		return fmt.Errorf("row %d missing required field FactorySKU", rowNumber)
	}
	if product.Name == "" {
		return fmt.Errorf("row %d missing required field Name", rowNumber)
	}
	return nil
}
