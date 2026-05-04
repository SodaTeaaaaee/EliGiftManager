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

// ImportDispatchWave imports a CSV into the wave.  It deduplicates rows by
// (platform, platformUid), upserts global members (CRM/address reuse), maintains
// nickname history, upserts wave_member snapshot rows, removes wave_members that
// disappeared from this import, and rebuilds waves.level_tags.
// setDefault controls whether imported addresses are set as the member's default.
// DispatchRecord creation is deferred to ReconcileWave.
func ImportDispatchWave(db *gorm.DB, waveID uint, csvPath string, importTemplate model.TemplateConfig, setDefault bool) (int, error) {
	if strings.TrimSpace(csvPath) == "" {
		return 0, fmt.Errorf("import dispatch wave failed: CSV path is required")
	}
	if err := ensureTemplateType(importTemplate, model.TemplateTypeImportDispatchRecord, "dispatch import"); err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: %w", err)
	}

	// Parse template mapping rules to DynamicTemplateRules.
	var rules model.DynamicTemplateRules
	if err := json.Unmarshal([]byte(importTemplate.MappingRules), &rules); err != nil {
		return 0, fmt.Errorf("import dispatch wave failed: parse mapping rules: %w", err)
	}

	type csvRow struct {
		giftLevel   string
		platformUid string
		nickname    string
		platform    string
		extraData   string // JSON string
		recipient   string
		phone       string
		address     string
	}

	// Parse CSV and deduplicate to unique (platform, platformUid) pairs.
	// Last occurrence wins when the same member appears multiple times.
	dedupMap := make(map[string]csvRow)  // key: platform + "||" + platformUid
	levelTagMap := map[string]struct{}{} // key: platform + "::" + giftLevel

	file, oErr := os.Open(csvPath)
	if oErr != nil {
		return 0, fmt.Errorf("import dispatch wave failed: open %q failed: %w", csvPath, oErr)
	}
	defer file.Close()

	reader := csv.NewReader(stripBOM(file))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	// Read header line if the template expects one.
	var headers []string
	lineStart := 1
	if rules.HasHeader {
		hdr, hdrErr := reader.Read()
		if hdrErr != nil {
			return 0, fmt.Errorf("import dispatch wave failed: read header: %w", hdrErr)
		}
		headers = hdr
		lineStart = 2
	}

	for lineNo := lineStart; ; lineNo++ {
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

		coreData, extraDataMap, parseErr := ParseRowDynamically(record, headers, rules)
		if parseErr != nil {
			return 0, fmt.Errorf("import dispatch wave failed: line %d: %w", lineNo, parseErr)
		}

		extraDataJSON, _ := json.Marshal(extraDataMap)
		extraDataStr := string(extraDataJSON)
		if extraDataStr == "null" || extraDataStr == "{}" {
			extraDataStr = "{}"
		}

		row := csvRow{
			giftLevel:   coreData["gift_level"],
			platformUid: coreData["platform_uid"],
			nickname:    coreData["nickname"],
			platform:    coreData["platform"],
			extraData:   extraDataStr,
			recipient:   strings.TrimSpace(coreData["recipient_name"]),
			phone:       strings.TrimSpace(coreData["phone"]),
			address:     strings.TrimSpace(coreData["address"]),
		}

		// platform fallback when columnIndex is -1 (column absent by design).
		if row.platform == "" {
			row.platform = importTemplate.Platform
		}

		if row.giftLevel == "" || row.platformUid == "" {
			return 0, fmt.Errorf("import dispatch wave failed: line %d missing required fields (giftLevel=%q, platformUid=%q)", lineNo, row.giftLevel, row.platformUid)
		}

		// Deduplicate: last row for a given (platform, uid) wins.
		dedupKey := row.platform + "||" + row.platformUid
		dedupMap[dedupKey] = row

		// Collect unique level tags.
		ltKey := row.platform + "::" + row.giftLevel
		levelTagMap[ltKey] = struct{}{}
	}

	if len(dedupMap) == 0 {
		return 0, fmt.Errorf("import dispatch wave failed: CSV contains no data rows")
	}

	// Transaction: upsert members, maintain nicknames, upsert wave_members, prune stale.
	importedMemberIDs := make(map[uint]bool)

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, row := range dedupMap {
			// Upsert global member (platform + platform_uid only; no giftLevel in extra_data).
			member := model.Member{
				Platform:    strings.TrimSpace(row.platform),
				PlatformUID: strings.TrimSpace(row.platformUid),
				ExtraData:   row.extraData,
			}
			if upsertErr := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "platform_uid"}, {Name: "platform"}},
				DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
			}).Create(&member).Error; upsertErr != nil {
				return fmt.Errorf("upsert member failed for platform_uid=%q: %w", row.platformUid, upsertErr)
			}

			// Address insertion (3-tuple present → dedup + insert, or promote existing to default).
			if row.recipient != "" && row.phone != "" && row.address != "" {
				var existingAddr model.MemberAddress
				findErr := tx.Where("member_id = ? AND recipient_name = ? AND phone = ? AND address = ? AND is_deleted = ?",
					member.ID, row.recipient, row.phone, row.address, false).First(&existingAddr).Error
				if findErr == gorm.ErrRecordNotFound {
					newAddr := model.MemberAddress{
						MemberID:      member.ID,
						RecipientName: row.recipient,
						Phone:         row.phone,
						Address:       row.address,
						IsDefault:     setDefault,
						IsDeleted:     false,
					}
					if createErr := tx.Create(&newAddr).Error; createErr != nil {
						return fmt.Errorf("insert member address failed: %w", createErr)
					}
					// Unset IsDefault for other addresses of this member.
					if setDefault {
						tx.Model(&model.MemberAddress{}).Where("member_id = ? AND id != ?", member.ID, newAddr.ID).Update("is_default", false)
					}
				} else if setDefault && !existingAddr.IsDefault {
					existingAddr.IsDefault = true
					if saveErr := tx.Save(&existingAddr).Error; saveErr != nil {
						return fmt.Errorf("update existing address default flag failed: %w", saveErr)
					}
					tx.Model(&model.MemberAddress{}).Where("member_id = ? AND id != ?", member.ID, existingAddr.ID).Update("is_default", false)
				}
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
				"gift_level":      row.giftLevel,
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
			if err := tx.Where("wave_member_id = ? AND tag_type = ?", wm.ID, model.TagTypeUser).Delete(&model.ProductTag{}).Error; err != nil {
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
