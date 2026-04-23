package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// ParseProductCSV 将外部商品 CSV 文件转换为内部统一标准的 Product 结构切片。
func ParseProductCSV(csvFile string, template model.TemplateConfig) ([]model.Product, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("解析商品 CSV 失败: CSV 文件路径不能为空")
	}

	if err := ensureTemplateType(template, model.TemplateTypeImportProduct, "商品导入"); err != nil {
		return nil, fmt.Errorf("解析商品 CSV 失败: %w", err)
	}

	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeProductFieldName)
	if err != nil {
		return nil, fmt.Errorf("解析商品 CSV 失败: %w", err)
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("解析商品 CSV 失败: 打开文件 %q 失败: %w", csvFile, err)
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("解析商品 CSV 失败: CSV 文件为空")
		}

		return nil, fmt.Errorf("解析商品 CSV 失败: 读取表头失败: %w", err)
	}

	headerIndex, normalizedHeaders, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("解析商品 CSV 失败: %w", err)
	}

	fieldColumns, consumedColumns, err := resolveFieldColumns(mappingRules, headerIndex)
	if err != nil {
		return nil, fmt.Errorf("解析商品 CSV 失败: %w", err)
	}

	products := make([]model.Product, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("解析商品 CSV 失败: 读取第 %d 行失败: %w", rowNumber, readErr)
		}

		if isEmptyRecord(record) {
			continue
		}

		product := model.Product{
			ExtraData: "{}",
		}

		for fieldName, columnIndex := range fieldColumns {
			value := readCSVCell(record, columnIndex)

			switch fieldName {
			case "factory":
				product.Factory = value
			case "factory_sku":
				product.FactorySKU = value
			case "name":
				product.Name = value
			case "image_path":
				product.ImagePath = value
			default:
				return nil, fmt.Errorf("解析商品 CSV 失败: 第 %d 行存在未支持的标准字段 %q", rowNumber, fieldName)
			}
		}

		if err := validateProductRecord(product, rowNumber); err != nil {
			return nil, fmt.Errorf("解析商品 CSV 失败: %w", err)
		}

		extraDataJSON, err := buildExtraDataJSON(normalizedHeaders, record, consumedColumns)
		if err != nil {
			return nil, fmt.Errorf("解析商品 CSV 失败: 第 %d 行构建扩展数据失败: %w", rowNumber, err)
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
	case "factory", "工厂", "工厂名", "工厂名称", "供应商", "供应商名称":
		return "factory", nil
	case "factorysku", "sku", "工厂sku", "平台sku", "商品sku", "货号":
		return "factory_sku", nil
	case "name", "productname", "商品名", "商品名称", "产品名", "产品名称":
		return "name", nil
	case "imagepath", "image", "imageurl", "图片", "图片路径", "主图路径":
		return "image_path", nil
	default:
		return "", fmt.Errorf("不支持的商品标准字段 %q", field)
	}
}

func validateProductRecord(product model.Product, rowNumber int) error {
	if product.Factory == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 Factory", rowNumber)
	}

	if product.FactorySKU == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 FactorySKU", rowNumber)
	}

	if product.Name == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 Name", rowNumber)
	}

	return nil
}
