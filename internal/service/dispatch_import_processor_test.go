package service

import (
	"encoding/json"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestImportDispatchWaveRebuildsLevelTagsFromFinalWaveMembers(t *testing.T) {
	t.Parallel()

	db := newServiceTestDB(t)

	wave := model.Wave{WaveNo: "TASK-TEST-IMPORT-001", Name: "import test", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("failed to seed wave: %v", err)
	}

	template := model.TemplateConfig{
		Platform: "BILIBILI",
		Type:     model.TemplateTypeImportDispatchRecord,
		Name:     "dispatch import",
		MappingRules: `{
			"mapping": {
				"giftName": {"columnIndex": 0},
				"platformUid": {"columnIndex": 1},
				"nickname": {"columnIndex": 2}
			}
		}`,
	}

	csvPath := writeTestCSVFile(t, []string{
		"舰长,uid-1,alpha",
		"提督,uid-1,beta",
		"总督,uid-2,gamma",
	})

	imported, err := ImportDispatchWave(db, wave.ID, csvPath, template)
	if err != nil {
		t.Fatalf("ImportDispatchWave returned unexpected error: %v", err)
	}
	if imported != 2 {
		t.Fatalf("expected 2 deduplicated members, got %d", imported)
	}

	var waveMembers []model.WaveMember
	if err := db.Where("wave_id = ?", wave.ID).Order("platform_uid ASC").Find(&waveMembers).Error; err != nil {
		t.Fatalf("failed to query wave members: %v", err)
	}
	if len(waveMembers) != 2 {
		t.Fatalf("expected 2 wave members, got %d", len(waveMembers))
	}
	if waveMembers[0].PlatformUID != "uid-1" || waveMembers[0].GiftLevel != "提督" || waveMembers[0].LatestNickname != "beta" {
		t.Fatalf("expected uid-1 snapshot to keep last row, got %+v", waveMembers[0])
	}
	if waveMembers[1].PlatformUID != "uid-2" || waveMembers[1].GiftLevel != "总督" {
		t.Fatalf("unexpected second wave member snapshot: %+v", waveMembers[1])
	}

	var updatedWave model.Wave
	if err := db.First(&updatedWave, wave.ID).Error; err != nil {
		t.Fatalf("failed to reload wave: %v", err)
	}

	var levelTags []struct {
		Platform  string `json:"platform"`
		TagName   string `json:"tagName"`
		MatchMode string `json:"matchMode"`
	}
	if err := json.Unmarshal([]byte(updatedWave.LevelTags), &levelTags); err != nil {
		t.Fatalf("failed to parse wave level tags: %v", err)
	}
	// New BuildWaveIdentityTagCandidates emits: 2 gift_level + 1 platform_all + 1 wave_all = 4 entries.
	if len(levelTags) != 4 {
		t.Fatalf("expected 4 level tags (2 gift_level + 1 platform_all + 1 wave_all), got %d: %s", len(levelTags), updatedWave.LevelTags)
	}

	got := map[string]bool{}
	for _, item := range levelTags {
		got[item.Platform+"::"+item.TagName+"::"+item.MatchMode] = true
	}
	if !got["BILIBILI::提督::gift_level"] || !got["BILIBILI::总督::gift_level"] {
		t.Fatalf("expected final level tags to include 提督 and 总督 gift_level entries, got %s", updatedWave.LevelTags)
	}
	if got["BILIBILI::舰长::gift_level"] {
		t.Fatalf("expected overwritten level tag 舰长 to be absent, got %s", updatedWave.LevelTags)
	}
	if !got["BILIBILI::::platform_all"] {
		t.Fatalf("expected a platform_all entry for BILIBILI, got %s", updatedWave.LevelTags)
	}
	if !got["::::wave_all"] {
		t.Fatalf("expected a wave_all entry, got %s", updatedWave.LevelTags)
	}
}
