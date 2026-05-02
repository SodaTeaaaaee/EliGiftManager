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

// ImportDispatchWave imports a headless CSV into the wave.  It deduplicates rows by
// (platform, platformUid), upserts global members (CRM/address reuse, no giftLevel in
// extra_data), maintains nickname history, upserts wave_member snapshot rows, removes
// wave_members that disappeared from this import, and rebuilds waves.level_tags.
// DispatchRecord creation is deferred to ReconcileWave.
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

	// Parse CSV and deduplicate to unique (platform, platformUid) pairs.
	// Last occurrence wins when the same member appears multiple times.
	dedupMap := make(map[string]csvRow) // key: platform + "||" + platformUid
	levelTagMap := map[string]struct{}{} // key: platform + "::" + giftName

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

		// Deduplicate: last row for a given (platform, uid) wins.
		dedupKey := row.platform + "||" + row.platformUid
		dedupMap[dedupKey] = row

		// Collect unique level tags.
		ltKey := row.platform + "::" + row.giftName
		levelTagMap[ltKey] = struct{}{}
	}

	if len(dedupMap) == 0 {
		return 0, fmt.Errorf("import dispatch wave failed: CSV contains no data rows")
	}

	// Transaction: upsert members, maintain nicknames, upsert wave_members, prune stale.
	importedMemberIDs := make(map[uint]bool)

	err = db.Transaction(func(tx *gorm.DB) error {
		for _, row := range dedupMap {
			// Upsert global member (platform + platform_uid only; no giftLevel in extra_data).
			member := model.Member{
				Platform:    strings.TrimSpace(row.platform),
				PlatformUID: strings.TrimSpace(row.platformUid),
				ExtraData:   "{}",
			}
			if upsertErr := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "platform_uid"}, {Name: "platform"}},
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
			}).Create(&member).Error; upsertErr != nil {
				return fmt.Errorf("upsert member failed for platform_uid=%q: %w", row.platformUid, upsertErr)
			}

			// Maintain nickname history.
			if strings.TrimSpace(row.nickname) != "" {
				if nickErr := ensureLatestNickname(tx, member.ID, row.nickname); nickErr != nil {
					return fmt.Errorf("maintain nickname failed: %w", nickErr)
				}
			}

			// Determine latest nickname for the wave_member snapshot.
			latestNick := strings.TrimSpace(row.nickname)
			if latestNick == "" {
				latestNick = row.platformUid
			}

			// Upsert wave_member — FirstOrCreate to get the ID, then update snapshot fields.
			wm := model.WaveMember{WaveID: waveID, MemberID: member.ID}
			if wmErr := tx.Where("wave_id = ? AND member_id = ?", waveID, member.ID).
				FirstOrCreate(&wm).Error; wmErr != nil {
				return fmt.Errorf("upsert wave member failed: %w", wmErr)
			}
			if updateErr := tx.Model(&wm).Updates(map[string]any{
				"platform":        row.platform,
				"platform_uid":    row.platformUid,
				"gift_level":      row.giftName,
				"latest_nickname": latestNick,
			}).Error; updateErr != nil {
				return fmt.Errorf("update wave member snapshot failed: %w", updateErr)
			}

			importedMemberIDs[member.ID] = true
		}

		// Delete wave_members that are no longer in this import.
		// Manual cascade: delete user tags and dispatch records before the wave_member row.
		var existingWMs []model.WaveMember
		if err := tx.Where("wave_id = ?", waveID).Find(&existingWMs).Error; err != nil {
			return fmt.Errorf("query existing wave members for pruning failed: %w", err)
		}
		for _, wm := range existingWMs {
			if importedMemberIDs[wm.MemberID] {
				continue
			}
			// Delete user tags referencing this wave member.
			if err := tx.Where("wave_member_id = ? AND tag_type = 'user'", wm.ID).Delete(&model.ProductTag{}).Error; err != nil {
				return fmt.Errorf("delete user tags for removed wave member (id=%d) failed: %w", wm.ID, err)
			}
			// Delete dispatch records for this (wave, member) pair.
			if err := tx.Where("wave_id = ? AND member_id = ?", waveID, wm.MemberID).Delete(&model.DispatchRecord{}).Error; err != nil {
				return fmt.Errorf("delete dispatch records for removed member (id=%d) failed: %w", wm.MemberID, err)
			}
			// Delete the wave_member row itself.
			if err := tx.Delete(&wm).Error; err != nil {
				return fmt.Errorf("delete wave member (id=%d) failed: %w", wm.ID, err)
			}
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: %w", err)
	}

	// Rebuild waves.level_tags from current wave_members.
	type levelTagEntry struct {
		Platform string `json:"platform"`
		TagName  string `json:"tagName"`
	}
	var levelTags []levelTagEntry
	for key := range levelTagMap {
		parts := strings.SplitN(key, "::", 2)
		if len(parts) == 2 {
			levelTags = append(levelTags, levelTagEntry{Platform: parts[0], TagName: parts[1]})
		}
	}
	levelTagsJSON, _ := json.Marshal(levelTags)
	if updateErr := db.Model(&model.Wave{}).Where("id = ?", waveID).Update("level_tags", string(levelTagsJSON)).Error; updateErr != nil {
		return len(dedupMap), fmt.Errorf("import dispatch wave warning: records imported but level_tags update failed: %w", updateErr)
	}

	return len(dedupMap), nil
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
