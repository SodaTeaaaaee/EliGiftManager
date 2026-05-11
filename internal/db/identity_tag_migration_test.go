package db

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestInitDBMigratesDirtyIdentityTagStateAndRebuildsLevelTags(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity-boundary-fix.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB failed: %v", err)
	}
	initialSQLDB := sqlDB
	t.Cleanup(func() { _ = initialSQLDB.Close() })

	waves := []model.Wave{
		{
			ID:        1,
			WaveNo:    "TASK-UPGRADE-001",
			Name:      "upgrade-wave",
			Status:    "draft",
			LevelTags: `[{"platform":"legacy","tagName":"x"}]`,
		},
		{
			ID:        99,
			WaveNo:    "TASK-UPGRADE-099",
			Name:      "cross-wave",
			Status:    "draft",
			LevelTags: `[]`,
		},
	}
	if err := db.Create(&waves).Error; err != nil {
		t.Fatalf("create waves failed: %v", err)
	}

	members := []model.Member{
		{ID: 1, Platform: "BILIBILI", PlatformUID: "uid-1", ExtraData: "{}"},
		{ID: 2, Platform: "BILIBILI", PlatformUID: "uid-2", ExtraData: "{}"},
		{ID: 3, Platform: "DOUYIN", PlatformUID: "uid-3", ExtraData: "{}"},
	}
	if err := db.Create(&members).Error; err != nil {
		t.Fatalf("create members failed: %v", err)
	}

	waveMembers := []model.WaveMember{
		{ID: 11, WaveID: 1, MemberID: 1, Platform: "BILIBILI", PlatformUID: "uid-1", GiftLevel: "提督", LatestNickname: "nick-1"},
		{ID: 12, WaveID: 1, MemberID: 2, Platform: "BILIBILI", PlatformUID: "uid-2", GiftLevel: "总督", LatestNickname: "nick-2"},
		{ID: 13, WaveID: 1, MemberID: 3, Platform: "DOUYIN", PlatformUID: "uid-3", GiftLevel: "骑士", LatestNickname: "nick-3"},
		{ID: 99, WaveID: 99, MemberID: 1, Platform: "BILIBILI", PlatformUID: "uid-1-cross", GiftLevel: "提督", LatestNickname: "cross"},
	}
	if err := db.Create(&waveMembers).Error; err != nil {
		t.Fatalf("create wave members failed: %v", err)
	}

	product := model.Product{
		ID:         21,
		Platform:   "BILIBILI",
		Factory:    "f",
		FactorySKU: "sku-1",
		Name:       "gift",
		WaveID:     &waves[0].ID,
		ExtraData:  "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	dirtyStmts := []string{
		`DROP INDEX IF EXISTS idx_prod_identity_tag`,
		`DROP INDEX IF EXISTS idx_prod_user_tag`,
		`INSERT INTO product_tags (id, product_id, platform, tag_name, match_mode, tag_type, quantity, wave_member_id, created_at, updated_at)
		 VALUES
		  (101, 21, 'BILIBILI', '提督', '', 'level', 2, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (102, 21, 'BILIBILI', '提督', '', 'level', 3, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (103, 21, 'BILIBILI', 'uid-1', 'user_member', 'identity', 5, 11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (104, 21, 'BILIBILI', 'uid-1', '', 'user', 2, 11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (105, 21, 'BILIBILI', 'uid-missing', 'user_member', 'identity', 9, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (106, 21, 'BILIBILI', 'uid-1-alt', 'user_member', 'identity', 1, 11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		  (107, 21, 'BILIBILI', 'uid-cross', 'user_member', 'identity', 7, 99, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, stmt := range dirtyStmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("inject dirty tag state failed: %v\nsql=%s", err, stmt)
		}
	}

	_ = sqlDB.Close()

	db, err = InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB re-open migration failed: %v", err)
	}
	sqlDB, err = db.DB()
	if err != nil {
		t.Fatalf("db.DB after reopen failed: %v", err)
	}
	reopenedSQLDB := sqlDB
	t.Cleanup(func() { _ = reopenedSQLDB.Close() })

	var identityTags []model.ProductTag
	if err := db.Where("product_id = ? AND tag_type = 'identity'", 21).Order("id ASC").Find(&identityTags).Error; err != nil {
		t.Fatalf("load identity tags failed: %v", err)
	}
	if len(identityTags) != 1 {
		t.Fatalf("expected 1 identity tag after migration, got %d", len(identityTags))
	}
	if identityTags[0].MatchMode != "gift_level" || identityTags[0].Quantity != 5 {
		t.Fatalf("unexpected identity tag after migration: %+v", identityTags[0])
	}

	var userTags []model.ProductTag
	if err := db.Where("product_id = ? AND tag_type = 'user'", 21).Order("id ASC").Find(&userTags).Error; err != nil {
		t.Fatalf("load user tags failed: %v", err)
	}
	if len(userTags) != 1 {
		t.Fatalf("expected 1 user tag after migration, got %d", len(userTags))
	}
	if userTags[0].WaveMemberID == nil || *userTags[0].WaveMemberID != 11 {
		t.Fatalf("unexpected user wave_member_id after migration: %+v", userTags[0])
	}
	if userTags[0].Quantity != 8 {
		t.Fatalf("expected merged user quantity 8, got %d", userTags[0].Quantity)
	}
	if userTags[0].MatchMode != "user_member" {
		t.Fatalf("expected migrated user match_mode=user_member, got %q", userTags[0].MatchMode)
	}

	var invalidConvertedCount int64
	if err := db.Model(&model.ProductTag{}).
		Where("product_id = ? AND tag_type = 'identity' AND match_mode = 'user_member'", 21).
		Count(&invalidConvertedCount).Error; err != nil {
		t.Fatalf("count identity/user_member rows failed: %v", err)
	}
	if invalidConvertedCount != 0 {
		t.Fatalf("expected all identity/user_member rows to be removed, found %d", invalidConvertedCount)
	}

	var crossWaveUserCount int64
	if err := db.Model(&model.ProductTag{}).
		Where("product_id = ? AND tag_type = 'user' AND wave_member_id = ?", 21, 99).
		Count(&crossWaveUserCount).Error; err != nil {
		t.Fatalf("count cross-wave user rows failed: %v", err)
	}
	if crossWaveUserCount != 0 {
		t.Fatalf("expected cross-wave identity/user_member row to be deleted, found converted user rows=%d", crossWaveUserCount)
	}

	var waveAfter model.Wave
	if err := db.First(&waveAfter, 1).Error; err != nil {
		t.Fatalf("reload wave failed: %v", err)
	}
	var levelTags []struct {
		Platform  string `json:"platform"`
		TagName   string `json:"tagName"`
		MatchMode string `json:"matchMode"`
	}
	if err := json.Unmarshal([]byte(waveAfter.LevelTags), &levelTags); err != nil {
		t.Fatalf("parse wave level_tags failed: %v", err)
	}

	got := map[string]bool{}
	for _, tag := range levelTags {
		got[tag.Platform+"::"+tag.TagName+"::"+tag.MatchMode] = true
	}
	for _, want := range []string{
		"BILIBILI::提督::gift_level",
		"BILIBILI::总督::gift_level",
		"DOUYIN::骑士::gift_level",
		"BILIBILI::::platform_all",
		"DOUYIN::::platform_all",
		"::::wave_all",
	} {
		if !got[want] {
			t.Fatalf("expected rebuilt level_tags to contain %s, got %s", want, waveAfter.LevelTags)
		}
	}

	var indexNames []string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'product_tags'").Scan(&indexNames).Error; err != nil {
		t.Fatalf("load product_tags indexes failed: %v", err)
	}
	indexSet := map[string]bool{}
	for _, name := range indexNames {
		indexSet[name] = true
	}
	if !indexSet["idx_prod_identity_tag"] || !indexSet["idx_prod_user_tag"] {
		t.Fatalf("expected new partial unique indexes to exist, got %#v", indexNames)
	}
}
