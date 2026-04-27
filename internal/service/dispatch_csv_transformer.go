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

func ParseDispatchRecordCSV(csvFile string, template model.TemplateConfig) ([]model.DispatchRecord, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("parse dispatch record CSV failed: CSV path is required")
	}
	if err := ensureTemplateType(template, model.TemplateTypeImportDispatchRecord, "dispatch record import"); err != nil {
		return nil, fmt.Errorf("parse dispatch record CSV failed: %w", err)
	}
	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeDispatchFieldName)
	if err != nil {
		return nil, fmt.Errorf("parse dispatch record CSV failed: %w", err)
	}
	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("parse dispatch record CSV failed: open %q failed: %w", csvFile, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("parse dispatch record CSV failed: CSV is empty")
		}
		return nil, fmt.Errorf("parse dispatch record CSV failed: read header failed: %w", err)
	}
	headerIndex, _, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("parse dispatch record CSV failed: %w", err)
	}
	fieldColumns, _, err := resolveFieldColumns(anyMapFromStringMap(mappingRules), headerIndex)
	if err != nil {
		return nil, fmt.Errorf("parse dispatch record CSV failed: %w", err)
	}
	records := make([]model.DispatchRecord, 0)
	for rowNumber := 2; ; rowNumber++ {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("parse dispatch record CSV failed: read row %d failed: %w", rowNumber, readErr)
		}
		if isEmptyRecord(row) {
			continue
		}
		dispatchRecord := model.DispatchRecord{Quantity: 1, Status: model.DispatchStatusPending}
		for fieldName, columnIndex := range fieldColumns {
			value := readCSVCell(row, columnIndex)
			switch fieldName {
			case "wave_id":
				id, err := parseUintFieldValue(value, "WaveID", rowNumber)
				if err != nil {
					return nil, err
				}
				dispatchRecord.WaveID = id
			case "member_id":
				if value != "" {
					id, err := parseUintFieldValue(value, "MemberID", rowNumber)
					if err != nil {
						return nil, err
					}
					dispatchRecord.MemberID = id
				}
			case "product_id":
				if value != "" {
					id, err := parseUintFieldValue(value, "ProductID", rowNumber)
					if err != nil {
						return nil, err
					}
					dispatchRecord.ProductID = id
				}
			case "quantity":
				if value != "" {
					quantity, err := parseIntFieldValue(value, "Quantity", rowNumber)
					if err != nil {
						return nil, err
					}
					dispatchRecord.Quantity = quantity
				}
			case "status":
				if value != "" {
					dispatchRecord.Status = value
				}
			default:
				return nil, fmt.Errorf("parse dispatch record CSV failed: row %d has unsupported field %q", rowNumber, fieldName)
			}
		}
		if err := validateDispatchRecord(dispatchRecord, rowNumber); err != nil {
			return nil, fmt.Errorf("parse dispatch record CSV failed: %w", err)
		}
		records = append(records, dispatchRecord)
	}
	return records, nil
}

func normalizeDispatchFieldName(field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
	switch normalized {
	case "waveid", "taskid", "dispatchtaskid", "波次id", "发货任务id":
		return "wave_id", nil
	case "memberid", "会员id", "用户id", "收件人id":
		return "member_id", nil
	case "productid", "商品id", "产品id", "skuid":
		return "product_id", nil
	case "quantity", "qty", "数量", "发货数量":
		return "quantity", nil
	case "status", "状态", "发货状态":
		return "status", nil
	default:
		return "", fmt.Errorf("unsupported dispatch record field %q", field)
	}
}

func parseUintFieldValue(raw string, fieldName string, rowNumber int) (uint, error) {
	parsedValue, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("row %d field %s is not a valid unsigned integer: %w", rowNumber, fieldName, err)
	}
	return uint(parsedValue), nil
}

func parseIntFieldValue(raw string, fieldName string, rowNumber int) (int, error) {
	parsedValue, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("row %d field %s is not a valid integer: %w", rowNumber, fieldName, err)
	}
	return parsedValue, nil
}

func validateDispatchRecord(record model.DispatchRecord, rowNumber int) error {
	if record.MemberID == 0 {
		return fmt.Errorf("row %d missing required field MemberID", rowNumber)
	}
	if record.ProductID == 0 {
		return fmt.Errorf("row %d missing required field ProductID", rowNumber)
	}
	if record.Quantity <= 0 {
		return fmt.Errorf("row %d field Quantity must be greater than 0", rowNumber)
	}
	if record.Status == "" {
		return fmt.Errorf("row %d missing required field Status", rowNumber)
	}
	return nil
}
