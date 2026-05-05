package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

// TemplateValidationResult holds the outcome of template rules validation.
// Errors are blocking (save should be rejected); Warnings are advisory.
type TemplateValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// ValidateTemplateRules checks that mappingRules JSON structurally conforms
// to the expected schema for the given templateType.
func ValidateTemplateRules(templateType, mappingRules string) (*TemplateValidationResult, error) {
	result := &TemplateValidationResult{Valid: true, Errors: []string{}, Warnings: []string{}}
	mappingRules = strings.TrimSpace(mappingRules)

	if mappingRules == "" || mappingRules == "{}" {
		result.Valid = false
		result.Errors = append(result.Errors, "mapping rules must not be empty")
		return result, nil
	}

	switch templateType {
	case model.TemplateTypeImportDispatchRecord:
		validateImportDispatchRecord(mappingRules, result)
	case model.TemplateTypeImportProduct:
		validateImportProduct(mappingRules, result)
	case model.TemplateTypeExportOrder:
		validateExportOrder(mappingRules, result)
	default:
		return nil, fmt.Errorf("unknown template type: %s", templateType)
	}

	result.Valid = len(result.Errors) == 0
	return result, nil
}

func validateImportDispatchRecord(mappingRules string, result *TemplateValidationResult) {
	var rules model.DynamicTemplateRules
	if err := json.Unmarshal([]byte(mappingRules), &rules); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	if rules.Format != "csv" && rules.Format != "zip" {
		result.Errors = append(result.Errors, fmt.Sprintf("format must be 'csv' or 'zip', got '%s'", rules.Format))
	}

	if len(rules.Mapping) == 0 {
		result.Errors = append(result.Errors, "mapping must not be empty")
		return
	}

	mustKeys := []string{"platform_uid", "gift_level"}
	for _, key := range mustKeys {
		if _, ok := rules.Mapping[key]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("mapping MUST contain key '%s'", key))
		}
	}

	shouldKeys := []string{"nickname", "recipient_name", "phone", "address"}
	for _, key := range shouldKeys {
		if _, ok := rules.Mapping[key]; !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("mapping SHOULD contain key '%s'", key))
		}
	}

	validateExtraData(&rules, result)

	if rules.HasHeader {
		for key, fm := range rules.Mapping {
			if fm.SourceColumn == "" && fm.ColumnIndex == 0 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("mapping '%s': sourceColumn empty with columnIndex=0 (may be JSON default — verify intent)", key))
			}
		}
	}
}

func validateImportProduct(mappingRules string, result *TemplateValidationResult) {
	var rules model.DynamicTemplateRules
	if err := json.Unmarshal([]byte(mappingRules), &rules); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	if rules.Format != "csv" && rules.Format != "zip" {
		result.Errors = append(result.Errors, fmt.Sprintf("format must be 'csv' or 'zip', got '%s'", rules.Format))
	}

	if len(rules.Mapping) == 0 {
		result.Errors = append(result.Errors, "mapping must not be empty")
		return
	}

	mustKeys := []string{"factory_sku", "name"}
	for _, key := range mustKeys {
		if _, ok := rules.Mapping[key]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("mapping MUST contain key '%s'", key))
		}
	}

	shouldKeys := []string{"platform", "factory", "cover_image"}
	for _, key := range shouldKeys {
		if _, ok := rules.Mapping[key]; !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("mapping SHOULD contain key '%s'", key))
		}
	}

	validateExtraData(&rules, result)

	if rules.HasHeader {
		for key, fm := range rules.Mapping {
			if fm.SourceColumn == "" && fm.ColumnIndex == 0 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("mapping '%s': sourceColumn empty with columnIndex=0 (may be JSON default — verify intent)", key))
			}
		}
	}

	if rules.Format == "zip" {
		var raw map[string]any
		if err := json.Unmarshal([]byte(mappingRules), &raw); err == nil {
			if csvPattern, ok := raw["csvPattern"]; !ok || csvPattern == "" || csvPattern == nil {
				result.Warnings = append(result.Warnings, "ZIP format: csvPattern is empty or missing (will default to '*.csv')")
			}
		}
	}
}

func validateExportOrder(mappingRules string, result *TemplateValidationResult) {
	var rules model.DynamicExportRules
	if err := json.Unmarshal([]byte(mappingRules), &rules); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	if len(rules.Columns) == 0 {
		result.Errors = append(result.Errors, "columns must not be empty")
		return
	}

	validValueTypes := map[string]bool{
		"order_no": true, "recipient": true, "phone": true, "address": true,
		"sku": true, "quantity": true, "static": true,
		"member_uid": true, "member_nickname": true,
	}

	for i, col := range rules.Columns {
		if strings.TrimSpace(col.HeaderName) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("column[%d]: headerName must not be empty", i))
		}

		if !validValueTypes[col.ValueType] {
			result.Errors = append(result.Errors, fmt.Sprintf("column[%d]: unknown valueType '%s'", i, col.ValueType))
		}

		if col.ValueType == "order_no" && strings.TrimSpace(col.Prefix) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("column[%d]: 'order_no' type must have a non-empty prefix", i))
		}

		if col.ValueType == "static" && strings.TrimSpace(col.DefaultValue) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("column[%d]: 'static' type must have a non-empty defaultValue", i))
		}
	}
}

func validateExtraData(rules *model.DynamicTemplateRules, result *TemplateValidationResult) {
	strategy := rules.ExtraData.Strategy
	if strategy != "catch_all" && strategy != "explicit" && strategy != "" {
		result.Errors = append(result.Errors,
			fmt.Sprintf("extraData.strategy must be 'catch_all', 'explicit', or empty, got '%s'", strategy))
	}
	if strategy == "explicit" && len(rules.ExtraData.ExplicitMapping) == 0 {
		result.Warnings = append(result.Warnings,
			"extraData.strategy is 'explicit' but explicitMapping is empty")
	}
}
