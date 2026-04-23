package service

import (
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
		return nil, fmt.Errorf("导入会员 CSV 失败: 数据库连接不能为空")
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
				return fmt.Errorf("保存会员 %s/%s 失败: %w", row.Member.Platform, row.Member.PlatformUID, upsertErr)
			}

			if err := ensureLatestNickname(tx, member.ID, row.Nickname); err != nil {
				return fmt.Errorf("维护会员 %d 的昵称历史失败: %w", member.ID, err)
			}

			importedMembers = append(importedMembers, member)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("导入会员 CSV 失败: %w", err)
	}

	return importedMembers, nil
}

func parseMemberCSVRows(csvFile string, template model.TemplateConfig) ([]parsedMemberCSVRow, error) {
	if strings.TrimSpace(csvFile) == "" {
		return nil, fmt.Errorf("解析会员 CSV 失败: CSV 文件路径不能为空")
	}

	if err := ensureTemplateType(template, model.TemplateTypeImportMember, "会员导入"); err != nil {
		return nil, fmt.Errorf("解析会员 CSV 失败: %w", err)
	}

	mappingRules, err := parseTemplateMappingRules(template.MappingRules, normalizeMemberFieldName)
	if err != nil {
		return nil, fmt.Errorf("解析会员 CSV 失败: %w", err)
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("解析会员 CSV 失败: 打开文件 %q 失败: %w", csvFile, err)
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
			return nil, fmt.Errorf("解析会员 CSV 失败: CSV 文件为空")
		}

		return nil, fmt.Errorf("解析会员 CSV 失败: 读取表头失败: %w", err)
	}

	headerIndex, normalizedHeaders, err := buildHeaderIndex(headers)
	if err != nil {
		return nil, fmt.Errorf("解析会员 CSV 失败: %w", err)
	}

	fieldColumns, consumedColumns, err := resolveFieldColumns(mappingRules, headerIndex)
	if err != nil {
		return nil, fmt.Errorf("解析会员 CSV 失败: %w", err)
	}

	rows := make([]parsedMemberCSVRow, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("解析会员 CSV 失败: 读取第 %d 行失败: %w", rowNumber, readErr)
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
				return nil, fmt.Errorf("解析会员 CSV 失败: 第 %d 行存在未支持的标准字段 %q", rowNumber, fieldName)
			}
		}

		if err := validateMemberRecord(row, rowNumber); err != nil {
			return nil, fmt.Errorf("解析会员 CSV 失败: %w", err)
		}

		extraDataJSON, err := buildExtraDataJSON(normalizedHeaders, record, consumedColumns)
		if err != nil {
			return nil, fmt.Errorf("解析会员 CSV 失败: 第 %d 行构建扩展数据失败: %w", rowNumber, err)
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
			return model.Member{}, fmt.Errorf("创建会员失败: %w", createErr)
		}

		return candidate, nil
	}
	if err != nil {
		return model.Member{}, fmt.Errorf("查询会员失败: %w", err)
	}

	existing.ExtraData = candidate.ExtraData
	if saveErr := db.Save(&existing).Error; saveErr != nil {
		return model.Member{}, fmt.Errorf("更新会员失败: %w", saveErr)
	}

	return existing, nil
}

func ensureLatestNickname(db *gorm.DB, memberID uint, nickname string) error {
	if strings.TrimSpace(nickname) == "" {
		return fmt.Errorf("昵称不能为空")
	}

	latestNickname, err := getLatestMemberNickname(db, memberID)
	if err != nil {
		return fmt.Errorf("查询最新昵称失败: %w", err)
	}

	if latestNickname != nil && latestNickname.Nickname == nickname {
		return nil
	}

	record := model.MemberNickname{
		MemberID: memberID,
		Nickname: nickname,
	}

	if err := db.Create(&record).Error; err != nil {
		return fmt.Errorf("创建昵称历史记录失败: %w", err)
	}

	return nil
}

func ensureTemplateType(template model.TemplateConfig, expectedType string, scene string) error {
	if template.Type != "" && template.Type != expectedType {
		return fmt.Errorf("模板类型 %q 不适用于%s", template.Type, scene)
	}

	return nil
}

func parseTemplateMappingRules(raw string, normalizeFieldName func(string) (string, error)) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("模板映射规则不能为空")
	}

	var mappingRules map[string]string
	if err := json.Unmarshal([]byte(raw), &mappingRules); err != nil {
		return nil, fmt.Errorf("解析模板映射规则 JSON 失败: %w", err)
	}

	if len(mappingRules) == 0 {
		return nil, fmt.Errorf("模板映射规则不能为空对象")
	}

	normalizedRules := make(map[string]string, len(mappingRules))
	for internalField, externalHeader := range mappingRules {
		normalizedField, err := normalizeFieldName(internalField)
		if err != nil {
			return nil, err
		}

		trimmedHeader := strings.TrimSpace(externalHeader)
		if trimmedHeader == "" {
			return nil, fmt.Errorf("标准字段 %q 对应的外部表头不能为空", internalField)
		}

		normalizedRules[normalizedField] = trimmedHeader
	}

	return normalizedRules, nil
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
		return "", fmt.Errorf("不支持的会员标准字段 %q", field)
	}
}

func buildHeaderIndex(headers []string) (map[string]int, []string, error) {
	headerIndex := make(map[string]int, len(headers))
	normalizedHeaders := make([]string, 0, len(headers))

	for index, header := range headers {
		trimmedHeader := strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))
		if trimmedHeader == "" {
			return nil, nil, fmt.Errorf("第 %d 列表头为空，无法建立映射", index+1)
		}

		normalizedHeader := normalizeHeaderName(trimmedHeader)
		if _, exists := headerIndex[normalizedHeader]; exists {
			return nil, nil, fmt.Errorf("检测到重复表头 %q，无法唯一确定映射关系", trimmedHeader)
		}

		headerIndex[normalizedHeader] = index
		normalizedHeaders = append(normalizedHeaders, trimmedHeader)
	}

	return headerIndex, normalizedHeaders, nil
}

func resolveFieldColumns(mappingRules map[string]string, headerIndex map[string]int) (map[string]int, map[int]struct{}, error) {
	fieldColumns := make(map[string]int, len(mappingRules))
	consumedColumns := make(map[int]struct{}, len(mappingRules))

	for fieldName, externalHeader := range mappingRules {
		columnIndex, exists := headerIndex[normalizeHeaderName(externalHeader)]
		if !exists {
			return nil, nil, fmt.Errorf("模板字段 %q 对应的外部表头 %q 在 CSV 中不存在", fieldName, externalHeader)
		}

		fieldColumns[fieldName] = columnIndex
		consumedColumns[columnIndex] = struct{}{}
	}

	return fieldColumns, consumedColumns, nil
}

func normalizeHeaderName(header string) string {
	normalized := strings.TrimSpace(header)
	normalized = strings.TrimPrefix(normalized, "\ufeff")
	normalized = strings.ToLower(normalized)
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)

	return normalized
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
		return fmt.Errorf("第 %d 行缺少必填字段 Platform", rowNumber)
	}

	if row.Member.PlatformUID == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 PlatformUID", rowNumber)
	}

	if row.Nickname == "" {
		return fmt.Errorf("第 %d 行缺少必填字段 Nickname", rowNumber)
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
		return "", fmt.Errorf("序列化 ExtraData 失败: %w", err)
	}

	return string(encoded), nil
}
