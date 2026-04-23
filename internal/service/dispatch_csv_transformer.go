package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// ParseDispatchRecordCSV 将外部分发记录 CSV 文件转换为内部统一标准的 DispatchRecord 结构切片。
func ParseDispatchRecordCSV(csvFile string, template model.TemplateConfig) ([]model.DispatchRecord, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: CSV 文件路径不能为空")
	}

	if err := ensureTemplateType(template, model.TemplateTypeImportDispatchRecord, "发货记录导入"); err != nil {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", err)
	}

	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeDispatchFieldName)
	if err != nil {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", err)
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: 打开文件 %q 失败: %w", csvFile, err)
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
			return nil, fmt.Errorf("解析发货记录 CSV 失败: CSV 文件为空")
		}

		return nil, fmt.Errorf("解析发货记录 CSV 失败: 读取表头失败: %w", err)
	}

	headerIndex, _, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", err)
	}

	fieldColumns, _, err := resolveFieldColumns(mappingRules, headerIndex)
	if err != nil {
		return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", err)
	}

	records := make([]model.DispatchRecord, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("解析发货记录 CSV 失败: 读取第 %d 行失败: %w", rowNumber, readErr)
		}

		if isEmptyRecord(record) {
			continue
		}

		dispatchRecord := model.DispatchRecord{
			Quantity: 1,
			Status:   model.DispatchStatusPending,
		}

		for fieldName, columnIndex := range fieldColumns {
			value := readCSVCell(record, columnIndex)

			switch fieldName {
			case "batch_name":
				dispatchRecord.BatchName = value
			case "member_id":
				if value == "" {
					continue
				}

				memberID, parseErr := parseUintFieldValue(value, "MemberID", rowNumber)
				if parseErr != nil {
					return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", parseErr)
				}
				dispatchRecord.MemberID = memberID
			case "product_id":
				if value == "" {
					continue
				}

				productID, parseErr := parseUintFieldValue(value, "ProductID", rowNumber)
				if parseErr != nil {
					return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", parseErr)
				}
				dispatchRecord.ProductID = productID
			case "quantity":
				if value == "" {
					continue
				}

				quantity, parseErr := parseIntFieldValue(value, "Quantity", rowNumber)
				if parseErr != nil {
					return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", parseErr)
				}
				dispatchRecord.Quantity = quantity
			case "status":
				if value == "" {
					continue
				}

				dispatchRecord.Status = value
			default:
				return nil, fmt.Errorf("解析发货记录 CSV 失败: 第 %d 行存在未支持的标准字段 %q", rowNumber, fieldName)
			}
		}

		if err := validateDispatchRecord(dispatchRecord, rowNumber); err != nil {
			return nil, fmt.Errorf("解析发货记录 CSV 失败: %w", err)
		}

		records = append(records, dispatchRecord)
	}

	return records, nil
}

func normalizeDispatchFieldName(field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)

	switch normalized {
	case "batchname", "batch", "批次", "批次号", "批次名称":
		return "batch_name", nil
	case "memberid", "会员id", "用户id", "收件人id":
		return "member_id", nil
	case "productid", "商品id", "产品id", "skuid":
		return "product_id", nil
	case "quantity", "qty", "数量", "发货数量":
		return "quantity", nil
	case "status", "状态", "发货状态":
		return "status", nil
	default:
		return "", fmt.Errorf("不支持的发货记录标准字段 %q", field)
	}
}

func parseUintFieldValue(raw string, fieldName string, rowNumber int) (uint, error) {
	parsedValue, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("第 %d 行字段 %s 不是合法的无符号整数: %w", rowNumber, fieldName, err)
	}

	return uint(parsedValue), nil
}

func parseIntFieldValue(raw string, fieldName string, rowNumber int) (int, error) {
	parsedValue, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("第 %d 行字段 %s 不是合法的整数: %w", rowNumber, fieldName, err)
	}

	return parsedValue, nil
}

func validateDispatchRecord(record model.DispatchRecord, rowNumber int) error {
	if record.BatchName == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 BatchName", rowNumber)
	}

	if record.MemberID == 0 {
		return fmt.Errorf("第 %d 行缺少必填字段 MemberID", rowNumber)
	}

	if record.ProductID == 0 {
		return fmt.Errorf("第 %d 行缺少必填字段 ProductID", rowNumber)
	}

	if record.Quantity <= 0 {
		return fmt.Errorf("第 %d 行字段 Quantity 必须大于 0", rowNumber)
	}

	if record.Status == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 Status", rowNumber)
	}

	return nil
}
