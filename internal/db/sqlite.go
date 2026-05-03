package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("initialize SQLite database failed: database path is required")
	}

	cleanedPath := filepath.Clean(dbPath)
	if err := ensureDatabaseDir(cleanedPath); err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cleanedPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: open %q failed: %w", cleanedPath, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: get underlying connection failed: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: ping failed: %w", err)
	}
	// WAL mode + reduced sync for better concurrent read/write performance.
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	db.Exec("PRAGMA foreign_keys = ON;")
	if err := autoMigrateTables(db); err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: %w", err)
	}

	return db, nil
}

func ensureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory %q failed: %w", dir, err)
	}
	return nil
}

func autoMigrateTables(db *gorm.DB) error {
	// Drop old 3-column unique index so GORM can re-create it with the new 4-column (incl. tag_type) index.
	db.Exec("DROP INDEX IF EXISTS idx_prod_platform_tag")
	if err := db.AutoMigrate(
		&model.Member{},
		&model.MemberNickname{},
		&model.MemberAddress{},
		&model.Product{},
		&model.ProductTag{},
		&model.ProductImage{},
		&model.Wave{},
		&model.DispatchRecord{},
		&model.TemplateConfig{},
		&model.WaveMember{},
	); err != nil {
		return fmt.Errorf("auto migrate database tables failed: %w", err)
	}
	// Normalise legacy data: tag_type="" → "level" and quantity=0 → 1.
	db.Exec("UPDATE product_tags SET tag_type = 'level' WHERE tag_type = '' OR tag_type IS NULL")
	db.Exec("UPDATE product_tags SET quantity = 1 WHERE quantity = 0")

	// Remove duplicate tags that may have been created when tag_type was inconsistent.
	// Keep the one with the highest quantity, delete the rest.
	db.Exec(`DELETE FROM product_tags WHERE id IN (
		SELECT id FROM (
			SELECT t1.id FROM product_tags t1
			INNER JOIN product_tags t2 ON t1.product_id = t2.product_id
				AND t1.platform = t2.platform
				AND t1.tag_name = t2.tag_name
				AND t1.tag_type = t2.tag_type
				AND t1.id > t2.id
		) dup
	)`)

	// --- Migration: WaveMember snapshot + ProductTag wave_member_id ---
	//
	// (a) AutoMigrate already added WaveMember.{Platform,PlatformUID,GiftLevel,LatestNickname}
	//     and ProductTag.WaveMemberID via ALTER TABLE ADD COLUMN above.
	// (b) Create new unique index for user tags.  The existing idx_prod_platform_tag was
	//     re-created by AutoMigrate from struct tags; we add the second composite index
	//     and a single-column index for wave_member_id lookups here via raw SQL to avoid
	//     a table rebuild that GORM would trigger for index/constraint changes on SQLite.
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_prod_wm_tag ON product_tags(product_id, wave_member_id, tag_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_product_tags_wm_id ON product_tags(wave_member_id)")

	// (b2) Dedup dispatch_records then add unique constraint to prevent duplicate
	// (wave_id, member_id, product_id) rows that could cause stale quantity display.
	db.Exec(`DELETE FROM dispatch_records WHERE id IN (
		SELECT id FROM (
			SELECT d1.id FROM dispatch_records d1
			INNER JOIN dispatch_records d2 ON d1.wave_id = d2.wave_id
				AND d1.member_id = d2.member_id
				AND d1.product_id = d2.product_id
				AND d1.id > d2.id
		) dup
	)`)
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_dispatch_wave_member_product ON dispatch_records(wave_id, member_id, product_id)")

	// (c.1) Backfill wave_members snapshot fields from members + nickname history.
	db.Exec(`
		UPDATE wave_members SET
			platform        = COALESCE((SELECT platform FROM members WHERE members.id = wave_members.member_id), ''),
			platform_uid    = COALESCE((SELECT platform_uid FROM members WHERE members.id = wave_members.member_id), ''),
			gift_level      = COALESCE((SELECT json_extract(extra_data, '$.giftLevel') FROM members WHERE members.id = wave_members.member_id), ''),
			latest_nickname = COALESCE((SELECT nickname FROM member_nicknames WHERE member_nicknames.member_id = wave_members.member_id ORDER BY created_at DESC LIMIT 1), '')
		WHERE platform = ''
	`)

	// (c.2) Backfill product_tags.wave_member_id for existing user tags.
	db.Exec(`
		UPDATE product_tags SET wave_member_id = (
			SELECT wm.id FROM wave_members wm
			JOIN products p ON p.wave_id = wm.wave_id
			WHERE p.id = product_tags.product_id
				AND wm.platform = product_tags.platform
				AND wm.platform_uid = product_tags.tag_name
			LIMIT 1
		) WHERE tag_type = 'user' AND wave_member_id IS NULL
	`)

	// (c.3) Delete orphaned user tags that could not be matched to any wave_member.
	db.Exec(`DELETE FROM product_tags WHERE tag_type = 'user' AND wave_member_id IS NULL`)

	// (d) Rebuild waves.level_tags from wave_members(platform, gift_level).
	type levelTagEntry struct {
		Platform string `json:"platform"`
		TagName  string `json:"tagName"`
	}
	var waves []model.Wave
	if err := db.Find(&waves).Error; err == nil {
		for _, wave := range waves {
			var wms []model.WaveMember
			if db.Where("wave_id = ?", wave.ID).Find(&wms).Error != nil {
				continue
			}
			levelTagMap := map[string]struct{}{}
			for _, wm := range wms {
				if wm.Platform != "" && wm.GiftLevel != "" {
					levelTagMap[wm.Platform+"::"+wm.GiftLevel] = struct{}{}
				}
			}
			var levelTags []levelTagEntry
			for key := range levelTagMap {
				parts := strings.SplitN(key, "::", 2)
				if len(parts) == 2 {
					levelTags = append(levelTags, levelTagEntry{Platform: parts[0], TagName: parts[1]})
				}
			}
			levelTagsJSON, _ := json.Marshal(levelTags)
			db.Model(&model.Wave{}).Where("id = ?", wave.ID).Update("level_tags", string(levelTagsJSON))
		}
	}

	return nil
}
