package service

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type parsedMemberCSVRow struct {
	Member   model.Member
	Nickname string
}

// ParseMemberCSV 将外部会员 CSV 文件转换为内部统一标准的 Member 结构切片。
/*
ParseMemberCSV 的职责不是“把 CSV 读进来”这么简单，它本质上承担的是导入防腐层
（Anti-Corruption Layer）的第一道语义翻译工作。

在这个项目里，外部平台的数据结构天然是异构的：
1. 不同直播平台导出的列名不一致，例如“用户ID”“粉丝编号”“平台UID”可能表达的是同一概念。
2. 同一平台在不同时间点导出的模板也可能变化，例如“收货地址”变成“详细地址”。
3. 外部文件为了适配业务现场，常常会夹带很多系统内部暂时不关心、但又不能直接丢弃的附加字段。

而我们的内部数据库模型必须保持稳定、可推理、可演进：
1. Member 是标准模型，字段命名和含义由系统自己定义。
2. 上层业务、SQLite 持久化、后续匹配与导出流程，都只能依赖这套稳定语义。
3. 系统绝不能把“外部平台今天叫什么列名”这种不稳定知识泄漏到内部核心域模型中。

因此，这个函数的核心意图是：
1. 读取原始 CSV 表头，识别外部世界的字段名称。
2. 读取 TemplateConfig.MappingRules 中定义的 JSON 映射规则，将“外部表头”翻译为“内部标准字段”。
3. 仅把内部真正理解、真正依赖的字段写入 model.Member 的强类型字段。
4. 昵称不再直接落在 Member 表中，而是作为历史语义由后续导入流程写入 MemberNickname。
5. 把没有被标准模型吸收的额外列，原样以 JSON 形式沉淀到 ExtraData，既保留原始上下文，又不污染标准结构。

这样做的直接收益是：
1. 外部平台变化时，优先调整模板配置，而不是频繁改数据库表结构或核心业务代码。
2. 内部服务层始终只面对统一标准 Member，后续聚合、匹配、导出逻辑不会被平台差异拖垮。
3. ExtraData 为后续回溯、补字段、灰度升级模板提供了兜底数据，避免一次导入就把信息永久丢失。

简而言之，这里不是“CSV 解析器”，而是“外部异构表格 -> 内部标准领域模型”的语义隔离带。
只要这个边界保持清晰，后续接入新的直播平台、团购平台、工厂系统时，系统就能通过模板配置扩展，
而不是让核心模型不断被外部格式牵着走。
*/
func ParseMemberCSV(csvFile string, template model.TemplateConfig) ([]model.Member, error) {
	rows, err := parseMemberCSVRows(csvFile, template)
	if err != nil {
		return nil, err
	}

	members := make([]model.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, row.Member)
	}

	return members, nil
}

// ImportMembersFromCSV 解析并落库会员数据，同时维护昵称历史记录。
func ImportMembersFromCSV(db *gorm.DB, csvFile string, template model.TemplateConfig) ([]model.Member, error) {
	if db == nil {
		return nil, fmt.Errorf("import member CSV failed: database connection is required")
	}

	rows, err := parseMemberCSVRows(csvFile, template)
	if err != nil {
		return nil, err
	}

	importedMembers := make([]model.Member, 0, len(rows))
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			member, upsertErr := upsertMember(tx, row.Member)
			if upsertErr != nil {
				return fmt.Errorf("save member %s/%s failed: %w", row.Member.Platform, row.Member.PlatformUID, upsertErr)
			}

			if err := ensureLatestNickname(tx, member.ID, row.Nickname); err != nil {
				return fmt.Errorf("maintain nickname history for member %d failed: %w", member.ID, err)
			}

			importedMembers = append(importedMembers, member)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("import member CSV failed: %w", err)
	}

	return importedMembers, nil
}

func parseMemberCSVRows(csvFile string, template model.TemplateConfig) ([]parsedMemberCSVRow, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("parse member CSV failed: CSV path is required")
	}

	if err := ensureTemplateType(template, model.TemplateTypeImportMember, "member import"); err != nil {
		return nil, fmt.Errorf("parse member CSV failed: %w", err)
	}

	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeMemberFieldName)
	if err != nil {
		return nil, fmt.Errorf("parse member CSV failed: %w", err)
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("parse member CSV failed: open %q failed: %w", csvFile, err)
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(stripBOM(file))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("parse member CSV failed: CSV is empty")
		}

		return nil, fmt.Errorf("parse member CSV failed: read header failed: %w", err)
	}

	headerIndex, normalizedHeaders, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("parse member CSV failed: %w", err)
	}

	fieldColumns, consumedColumns, err := resolveFieldColumns(anyMapFromStringMap(mappingRules), headerIndex)
	if err != nil {
		return nil, fmt.Errorf("parse member CSV failed: %w", err)
	}

	rows := make([]parsedMemberCSVRow, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("parse member CSV failed: read row %d failed: %w", rowNumber, readErr)
		}

		if isEmptyRecord(record) {
			continue
		}

		row := parsedMemberCSVRow{
			Member: model.Member{
				ExtraData: "{}",
			},
		}

		for fieldName, columnIndex := range fieldColumns {
			value := readCSVCell(record, columnIndex)

			switch fieldName {
			case "platform":
				row.Member.Platform = value
			case "platform_uid":
				row.Member.PlatformUID = value
			case "nickname":
				row.Nickname = value
			default:
				return nil, fmt.Errorf("parse member CSV failed: row %d has unsupported field %q", rowNumber, fieldName)
			}
		}

		if err := validateMemberRecord(row, rowNumber); err != nil {
			return nil, fmt.Errorf("parse member CSV failed: %w", err)
		}

		extraDataJSON, err := buildExtraDataJSON(normalizedHeaders, record, consumedColumns)
		if err != nil {
			return nil, fmt.Errorf("parse member CSV failed: row %d build ExtraData failed: %w", rowNumber, err)
		}
		row.Member.ExtraData = extraDataJSON

		rows = append(rows, row)
	}

	return rows, nil
}

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

func normalizeMemberFieldName(field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)

	switch normalized {
	case "platform", "平台", "平台名", "平台名称":
		return "platform", nil
	case "platformuid", "uid", "用户id", "平台uid", "平台用户id", "平台用户编号":
		return "platform_uid", nil
	case "nickname", "nick", "昵称":
		return "nickname", nil
	default:
		return "", fmt.Errorf("unsupported member standard field %q", field)
	}
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

func validateMemberRecord(row parsedMemberCSVRow, rowNumber int) error {
	if row.Member.Platform == "" {
		return fmt.Errorf("row %d missing required field Platform", rowNumber)
	}

	if row.Member.PlatformUID == "" {
		return fmt.Errorf("row %d missing required field PlatformUID", rowNumber)
	}

	if row.Nickname == "" {
		return fmt.Errorf("row %d missing required field Nickname", rowNumber)
	}

	return nil
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
