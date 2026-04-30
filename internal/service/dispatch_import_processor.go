package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ImportDispatchWave imports a headless CSV into the wave: parses rows, upserts
// members with giftLevel stored in ExtraData, collects levelTags on the wave.
// DispatchRecord creation is deferred to AllocateByTags.
func ImportDispatchWave(db *gorm.DB, waveID uint, csvPath string, importTemplate model.TemplateConfig) (int, error) {
	if strings.TrimSpace(csvPath) == "" {
		return 0, fmt.Errorf("import dispatch wave failed: CSV path is required")
	}
	if err := ensureTemplateType(importTemplate, model.TemplateTypeImportDispatchRecord, "dispatch import"); err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: %w", err)
	}

	mappingRules, err := parseTemplateMappingRulesV2(importTemplate.MappingRules, normalizeDispatchImportFieldName)
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: %w", err)
	}

	giftNameIdx, err := dispatchImportColumnIndex(mappingRules, "gift_name")
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: gift_name column index: %w", err)
	}
	platformUidIdx, err := dispatchImportColumnIndex(mappingRules, "platform_uid")
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: platform_uid column index: %w", err)
	}
	nicknameIdx, err := dispatchImportColumnIndex(mappingRules, "nickname")
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: nickname column index: %w", err)
	}

	platformIdx := -1
	if idx, platformIdxErr := dispatchImportColumnIndex(mappingRules, "platform"); platformIdxErr == nil {
		platformIdx = idx
	}

	type csvRow struct {
		giftName    string
		platformUid string
		nickname    string
		platform    string
	}
	var rows []csvRow

	file, oErr := os.Open(csvPath)
	if oErr != nil {
		return 0, fmt.Errorf("import dispatch wave failed: open %q failed: %w", csvPath, oErr)
	}
	defer file.Close()

	reader := csv.NewReader(stripBOM(file))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	for lineNo := 1; ; lineNo++ {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, fmt.Errorf("import dispatch wave failed: read line %d failed: %w", lineNo, readErr)
		}
		if isEmptyRecord(record) {
			continue
		}

		row := csvRow{
			giftName:    readCSVCell(record, giftNameIdx),
			platformUid: readCSVCell(record, platformUidIdx),
			nickname:    readCSVCell(record, nicknameIdx),
		}
		if platformIdx >= 0 {
			row.platform = readCSVCell(record, platformIdx)
		}
		if row.platform == "" {
			row.platform = importTemplate.Platform
		}

		if row.giftName == "" || row.platformUid == "" {
			return 0, fmt.Errorf("import dispatch wave failed: line %d missing required fields (giftName=%q, platformUid=%q)", lineNo, row.giftName, row.platformUid)
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return 0, fmt.Errorf("import dispatch wave failed: CSV contains no data rows")
	}

	// Collect unique levelTags.
	type levelTagEntry struct {
		Platform string `json:"platform"`
		TagName  string `json:"tagName"`
	}
	levelTagMap := map[string]struct{}{}
	for _, row := range rows {
		key := row.platform + "::" + row.giftName
		levelTagMap[key] = struct{}{}
	}

	// Transaction: member upsert + wave-member association + nickname maintenance.
	// DispatchRecord creation is deferred to AllocateByTags.
	importedCount := 0
	err = db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			extraDataJSON := fmt.Sprintf(`{"giftLevel":%q}`, row.giftName)

			member := model.Member{
				Platform:    strings.TrimSpace(row.platform),
				PlatformUID: strings.TrimSpace(row.platformUid),
				ExtraData:   extraDataJSON,
			}
			if upsertErr := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "platform_uid"}, {Name: "platform"}},
				DoUpdates: clause.AssignmentColumns([]string{"extra_data", "updated_at"}),
			}).Create(&member).Error; upsertErr != nil {
				return fmt.Errorf("upsert member failed for platform_uid=%q: %w", row.platformUid, upsertErr)
			}

			// Record wave-member association (幂等)
			if wmErr := tx.Where("wave_id = ? AND member_id = ?", waveID, member.ID).
				FirstOrCreate(&model.WaveMember{WaveID: waveID, MemberID: member.ID}).Error; wmErr != nil {
				return fmt.Errorf("record wave-member association failed: %w", wmErr)
			}

			if strings.TrimSpace(row.nickname) != "" {
				if nickErr := ensureLatestNickname(tx, member.ID, row.nickname); nickErr != nil {
					return fmt.Errorf("maintain nickname failed: %w", nickErr)
				}
			}

			importedCount++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: %w", err)
	}

	// Update wave levelTags.
	var levelTags []levelTagEntry
	for key := range levelTagMap {
		parts := strings.SplitN(key, "::", 2)
		levelTags = append(levelTags, levelTagEntry{Platform: parts[0], TagName: parts[1]})
	}
	levelTagsJSON, _ := json.Marshal(levelTags)
	if updateErr := db.Model(&model.Wave{}).Where("id = ?", waveID).Update("level_tags", string(levelTagsJSON)).Error; updateErr != nil {
		return importedCount, fmt.Errorf("import dispatch wave warning: records imported but level_tags update failed: %w", updateErr)
	}

	return importedCount, nil
}

func normalizeDispatchImportFieldName(field string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(field))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
	switch normalized {
	case "giftname", "gift", "礼物名", "礼物名称", "gift_name":
		return "gift_name", nil
	case "platformuid", "uid", "用户id", "平台uid", "platform_uid":
		return "platform_uid", nil
	case "nickname", "nick", "昵称":
		return "nickname", nil
	case "platform", "平台":
		return "platform", nil
	default:
		return "", fmt.Errorf("unsupported dispatch import field %q", field)
	}
}

func dispatchImportColumnIndex(mappingRules map[string]any, fieldName string) (int, error) {
	val, ok := mappingRules[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %q not found in mapping rules", fieldName)
	}
	meta, ok := val.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("expected mapping rule object for field %q, got %T (use new format with columnIndex)", fieldName, val)
	}
	ci, ok := meta["columnIndex"]
	if !ok {
		return 0, fmt.Errorf("field %q mapping rule missing columnIndex", fieldName)
	}
	idx, ok := ci.(float64)
	if !ok {
		return 0, fmt.Errorf("field %q columnIndex must be a number, got %T", fieldName, ci)
	}
	return int(idx), nil
}
