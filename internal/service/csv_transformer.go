package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

func upsertMember(db *gorm.DB, candidate model.Member) (model.Member, error) {
	var existing model.Member
	err := db.Where("platform = ? AND platform_uid = ?", candidate.Platform, candidate.PlatformUID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if createErr := db.Create(&candidate).Error; createErr != nil {
			return model.Member{}, fmt.Errorf("create member failed: %w", createErr)
		}

		return candidate, nil
	}
	if err != nil {
		return model.Member{}, fmt.Errorf("query member failed: %w", err)
	}

	existing.ExtraData = candidate.ExtraData
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return model.Member{}, fmt.Errorf("update member failed: %w", saveErr)
	}

	return existing, nil
}

func ensureLatestNickname(db *gorm.DB, memberID uint, nickname string) error {
	if strings.TrimSpace(nickname) == "" {
		return fmt.Errorf("nickname is required")
	}

	latestNickname, err := getLatestMemberNickname(db, memberID)
	if err != nil {
		return fmt.Errorf("query latest nickname failed: %w", err)
	}

	if latestNickname != nil && latestNickname.Nickname == nickname {
		return nil
	}

	record := model.MemberNickname{
		MemberID: memberID,
		Nickname: nickname,
	}

	if err := db.Create(&record).Error; err != nil {
		return fmt.Errorf("create nickname history record failed: %w", err)
	}

	return nil
}

func ensureTemplateType(template model.TemplateConfig, expectedType string, scene string) error {
	if template.Type != "" && template.Type != expectedType {
		return fmt.Errorf("template type %q is not applicable to %s", template.Type, scene)
	}

	return nil
}

func parseTemplateMappingRules(raw string, normalizeFieldName func(string) (string, error)) (map[string]string, error) {
	v2Result, err := parseTemplateMappingRulesV2(raw, normalizeFieldName)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(v2Result))
	for field, val := range v2Result {
		switch v := val.(type) {
		case string:
			result[field] = v
		case map[string]interface{}:
			if sc, ok := v["sourceColumn"]; ok {
				result[field] = strings.TrimSpace(fmt.Sprint(sc))
			} else {
				return nil, fmt.Errorf("template field %q in new format must specify sourceColumn for header-based CSV import", field)
			}
		default:
			return nil, fmt.Errorf("template field %q has unsupported mapping type %T", field, val)
		}
	}
	return result, nil
}

// parseTemplateMappingRulesV2 解析模板映射规则 JSON，向后兼容旧格式并支持 richer 新格式。
//
// 旧格式（向后兼容）：
//
//	{"field": "列名"}
//	→ 返回 map[string]any，每个 value 是 string（列名）
//
// 新格式：
//
//	{"hasHeader": true, "mapping": {"field": {"sourceColumn": "列名", "required": true}, ...}}
//	→ 返回 map[string]any，每个 value 是 map[string]interface{}（含 sourceColumn/columnIndex/required/transform）
func parseTemplateMappingRulesV2(raw string, normalizeFieldName func(string) (string, error)) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("template mapping rules is required")
	}

	// 先尝试旧格式：扁平 map[string]string
	var flatRules map[string]string
	if err := json.Unmarshal([]byte(raw), &flatRules); err == nil {
		if len(flatRules) == 0 {
			return nil, fmt.Errorf("template mapping rules cannot be empty")
		}
		result := make(map[string]any, len(flatRules))
		for internalField, externalHeader := range flatRules {
			normalizedField, err := normalizeFieldName(internalField)
			if err != nil {
				return nil, err
			}
			trimmedHeader := strings.TrimSpace(externalHeader)
			if trimmedHeader == "" {
				return nil, fmt.Errorf("external header for standard field %q cannot be empty", internalField)
			}
			result[normalizedField] = trimmedHeader
		}
		return result, nil
	}

	// 尝试新格式：{"hasHeader": ..., "mapping": {...}}
	var newRules struct {
		HasHeader *bool                  `json:"hasHeader"`
		Mapping   map[string]interface{} `json:"mapping"`
	}
	if err := json.Unmarshal([]byte(raw), &newRules); err != nil {
		return nil, fmt.Errorf("parse template mapping rules JSON failed: %w", err)
	}
	if newRules.Mapping == nil || len(newRules.Mapping) == 0 {
		return nil, fmt.Errorf("template mapping rules cannot be empty")
	}
	result := make(map[string]any, len(newRules.Mapping))
	for internalField, meta := range newRules.Mapping {
		normalizedField, err := normalizeFieldName(internalField)
		if err != nil {
			return nil, err
		}
		result[normalizedField] = meta
	}
	return result, nil
}

func buildHeaderIndex(headers []string) (map[string]int, []string, error) {
	headerIndex := make(map[string]int, len(headers))
	normalizedHeaders := make([]string, 0, len(headers))

	for index, header := range headers {
		trimmedHeader := strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))
		if trimmedHeader == "" {
			return nil, nil, fmt.Errorf("column %d header is empty, cannot build mapping", index+1)
		}

		normalizedHeader := normalizeHeaderName(trimmedHeader)
		if _, exists := headerIndex[normalizedHeader]; exists {
			return nil, nil, fmt.Errorf("duplicate header %q detected, cannot uniquely determine mapping", trimmedHeader)
		}

		headerIndex[normalizedHeader] = index
		normalizedHeaders = append(normalizedHeaders, trimmedHeader)
	}

	return headerIndex, normalizedHeaders, nil
}

func resolveFieldColumns(mappingRules map[string]any, headerIndex map[string]int) (map[string]int, map[int]struct{}, error) {
	fieldColumns := make(map[string]int, len(mappingRules))
	consumedColumns := make(map[int]struct{}, len(mappingRules))

	for fieldName, ruleVal := range mappingRules {
		var externalHeader string
		switch v := ruleVal.(type) {
		case string:
			externalHeader = v
		case map[string]interface{}:
			if sc, ok := v["sourceColumn"]; ok {
				externalHeader = fmt.Sprint(sc)
			} else {
				return nil, nil, fmt.Errorf("template field %q in new format must specify sourceColumn", fieldName)
			}
		default:
			return nil, nil, fmt.Errorf("template field %q has unsupported mapping type %T", fieldName, ruleVal)
		}

		trimmedHeader := strings.TrimSpace(externalHeader)
		if trimmedHeader == "" {
			return nil, nil, fmt.Errorf("external header for template field %q cannot be empty", fieldName)
		}

		columnIndex, exists := headerIndex[normalizeHeaderName(trimmedHeader)]
		if !exists {
			return nil, nil, fmt.Errorf("external header %q mapped to template field %q not found in CSV", trimmedHeader, fieldName)
		}

		fieldColumns[fieldName] = columnIndex
		consumedColumns[columnIndex] = struct{}{}
	}

	return fieldColumns, consumedColumns, nil
}

// anyMapFromStringMap 将 map[string]string 转换为 map[string]any，供 resolveFieldColumns 使用。
func anyMapFromStringMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func normalizeHeaderName(header string) string {
	normalized := strings.TrimSpace(header)
	normalized = strings.TrimPrefix(normalized, "\ufeff")
	normalized = strings.ToLower(normalized)
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)

	return normalized
}

func stripBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peek, _ := br.Peek(3)
	if len(peek) >= 3 && peek[0] == bom[0] && peek[1] == bom[1] && peek[2] == bom[2] {
		_, _ = br.Discard(3)
	}
	return br
}

func readCSVCell(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}

	return strings.TrimSpace(record[index])
}

func isEmptyRecord(record []string) bool {
	for _, item := range record {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}

	return true
}

func buildExtraDataJSON(normalizedHeaders []string, record []string, consumedColumns map[int]struct{}) (string, error) {
	extraData := make(map[string]string)

	for index, header := range normalizedHeaders {
		if _, consumed := consumedColumns[index]; consumed {
			continue
		}

		value := readCSVCell(record, index)
		if value == "" {
			continue
		}

		extraData[header] = value
	}

	if len(extraData) == 0 {
		return "{}", nil
	}

	encoded, err := json.Marshal(extraData)
	if err != nil {
		return "", fmt.Errorf("serialize ExtraData failed: %w", err)
	}

	return string(encoded), nil
}
