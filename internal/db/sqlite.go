package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var defaultDB *gorm.DB

// productMasterMigration is a migration-safe shape of product_masters without
// the final unique index.  We use it during the first AutoMigrate pass so dirty
// upgraded databases can be cleaned before uniqueness is enforced.
type productMasterMigration struct {
	ID         uint   `gorm:"primaryKey"`
	Platform   string `gorm:"size:100;not null;index"`
	Factory    string `gorm:"size:100;not null"`
	FactorySKU string `gorm:"size:255;not null;index"`
	Name       string `gorm:"size:255;not null"`
	CoverImage string `gorm:"type:text"`
	ExtraData  string `gorm:"type:text;not null;default:'{}'"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (productMasterMigration) TableName() string { return "product_masters" }

// productMigration is a migration-safe shape of products without the final
// (wave_id, platform, factory_sku) unique index.
type productMigration struct {
	ID              uint   `gorm:"primaryKey"`
	Platform        string `gorm:"size:100;not null;index"`
	Factory         string `gorm:"size:100;not null"`
	FactorySKU      string `gorm:"size:255;not null;index"`
	Name            string `gorm:"size:255;not null"`
	CoverImage      string `gorm:"type:text"`
	WaveID          *uint  `gorm:"index"`
	ExtraData       string `gorm:"type:text;not null;default:'{}'"`
	ProductMasterID *uint  `gorm:"index"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (productMigration) TableName() string { return "products" }

// productMasterImageMigration is a migration-safe shape of
// product_master_images without the final unique index on (product_master_id, path).
type productMasterImageMigration struct {
	ID              uint   `gorm:"primaryKey"`
	ProductMasterID uint   `gorm:"not null;index"`
	Path            string `gorm:"type:text;not null"`
	SortOrder       int    `gorm:"not null;default:0"`
	SourceDir       string `gorm:"size:100;not null;default:''"`
	CreatedAt       time.Time
}

func (productMasterImageMigration) TableName() string { return "product_master_images" }

// SetDefaultDB stores the app-wide DB instance for controllers.
func SetDefaultDB(db *gorm.DB) { defaultDB = db }

// GetDB returns the app-wide DB instance; nil before SetDefaultDB is called.
func GetDB() *gorm.DB { return defaultDB }

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
	if err := autoMigrateTablesPhaseOne(db); err != nil {
		return err
	}

	if err := migrateWaveMemberAndLegacyTags(db); err != nil {
		return err
	}

	if err := migrateProductMasterAndSnapshotSplit(db); err != nil {
		return err
	}

	if err := autoMigrateTablesPhaseTwo(db); err != nil {
		return err
	}

	return nil
}

func autoMigrateTablesPhaseOne(db *gorm.DB) error {
	// Drop old 3-column unique index so GORM can re-create it with the new 4-column (incl. tag_type) index.
	if err := db.Exec("DROP INDEX IF EXISTS idx_prod_platform_tag").Error; err != nil {
		return fmt.Errorf("drop legacy idx_prod_platform_tag failed: %w", err)
	}
	if err := db.AutoMigrate(
		&model.Member{},
		&model.MemberNickname{},
		&model.MemberAddress{},
		&productMasterMigration{},
		&productMigration{},
		&model.ProductTag{},
		&model.ProductImage{},
		&productMasterImageMigration{},
		&model.Wave{},
		&model.DispatchRecord{},
		&model.TemplateConfig{},
		&model.WaveMember{},
	); err != nil {
		return fmt.Errorf("auto migrate database tables phase one failed: %w", err)
	}
	return nil
}

func migrateWaveMemberAndLegacyTags(db *gorm.DB) error {
	// Normalise legacy data: tag_type="" → "level" and quantity=0 → 1.
	if err := db.Exec("UPDATE product_tags SET tag_type = 'level' WHERE tag_type = '' OR tag_type IS NULL").Error; err != nil {
		return fmt.Errorf("normalize legacy product_tags.tag_type failed: %w", err)
	}
	if err := db.Exec("UPDATE product_tags SET quantity = 1 WHERE quantity = 0").Error; err != nil {
		return fmt.Errorf("normalize legacy product_tags.quantity failed: %w", err)
	}

	// Remove duplicate tags that may have been created when tag_type was inconsistent.
	// Keep the one with the highest quantity, delete the rest.
	if err := db.Exec(`DELETE FROM product_tags WHERE id IN (
		SELECT id FROM (
			SELECT t1.id FROM product_tags t1
			INNER JOIN product_tags t2 ON t1.product_id = t2.product_id
				AND t1.platform = t2.platform
				AND t1.tag_name = t2.tag_name
				AND t1.tag_type = t2.tag_type
				AND t1.id > t2.id
		) dup
	)`).Error; err != nil {
		return fmt.Errorf("delete duplicate legacy product_tags failed: %w", err)
	}

	// --- Migration: WaveMember snapshot + ProductTag wave_member_id ---
	//
	// (a) AutoMigrate already added WaveMember.{Platform,PlatformUID,GiftLevel,LatestNickname}
	//     and ProductTag.WaveMemberID via ALTER TABLE ADD COLUMN above.
	// (b) Create new unique index for user tags.  The existing idx_prod_platform_tag was
	//     re-created by AutoMigrate from struct tags; we add the second composite index
	//     and a single-column index for wave_member_id lookups here via raw SQL to avoid
	//     a table rebuild that GORM would trigger for index/constraint changes on SQLite.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_prod_wm_tag ON product_tags(product_id, wave_member_id, tag_type)").Error; err != nil {
		return fmt.Errorf("create idx_prod_wm_tag failed: %w", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_product_tags_wm_id ON product_tags(wave_member_id)").Error; err != nil {
		return fmt.Errorf("create idx_product_tags_wm_id failed: %w", err)
	}

	// (b2) Dedup dispatch_records then add unique constraint to prevent duplicate
	// (wave_id, member_id, product_id) rows that could cause stale quantity display.
	if err := db.Exec(`DELETE FROM dispatch_records WHERE id IN (
		SELECT id FROM (
			SELECT d1.id FROM dispatch_records d1
			INNER JOIN dispatch_records d2 ON d1.wave_id = d2.wave_id
				AND d1.member_id = d2.member_id
				AND d1.product_id = d2.product_id
				AND d1.id > d2.id
		) dup
	)`).Error; err != nil {
		return fmt.Errorf("delete duplicate dispatch_records failed: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_dispatch_wave_member_product ON dispatch_records(wave_id, member_id, product_id)").Error; err != nil {
		return fmt.Errorf("create idx_dispatch_wave_member_product failed: %w", err)
	}

	// (c.1) Backfill wave_members snapshot fields from members + nickname history.
	if err := db.Exec(`
		UPDATE wave_members SET
			platform        = COALESCE((SELECT platform FROM members WHERE members.id = wave_members.member_id), ''),
			platform_uid    = COALESCE((SELECT platform_uid FROM members WHERE members.id = wave_members.member_id), ''),
			gift_level      = COALESCE((SELECT json_extract(extra_data, '$.giftLevel') FROM members WHERE members.id = wave_members.member_id), ''),
			latest_nickname = COALESCE((SELECT nickname FROM member_nicknames WHERE member_nicknames.member_id = wave_members.member_id ORDER BY created_at DESC LIMIT 1), '')
		WHERE platform = ''
	`).Error; err != nil {
		return fmt.Errorf("backfill wave_members snapshot fields failed: %w", err)
	}

	// (c.2) Backfill product_tags.wave_member_id for existing user tags.
	if err := db.Exec(`
		UPDATE product_tags SET wave_member_id = (
			SELECT wm.id FROM wave_members wm
			JOIN products p ON p.wave_id = wm.wave_id
			WHERE p.id = product_tags.product_id
				AND wm.platform = product_tags.platform
				AND wm.platform_uid = product_tags.tag_name
			LIMIT 1
		) WHERE tag_type = 'user' AND wave_member_id IS NULL
	`).Error; err != nil {
		return fmt.Errorf("backfill product_tags.wave_member_id failed: %w", err)
	}

	// (c.3) Delete orphaned user tags that could not be matched to any wave_member.
	if err := db.Exec(`DELETE FROM product_tags WHERE tag_type = 'user' AND wave_member_id IS NULL`).Error; err != nil {
		return fmt.Errorf("delete orphaned user tags failed: %w", err)
	}

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
			if err := db.Model(&model.Wave{}).Where("id = ?", wave.ID).Update("level_tags", string(levelTagsJSON)).Error; err != nil {
				return fmt.Errorf("rebuild wave %d level_tags failed: %w", wave.ID, err)
			}
		}
	}

	// Do not backfill is_test_address from mutable address text on upgrade.
	// Legacy generated addresses cannot be identified safely once edited,
	// and guessing would risk marking real user data as deletable test data.

	return nil
}

func migrateProductMasterAndSnapshotSplit(db *gorm.DB) error {
	// --- Migration: ProductMaster / Product snapshot split ---
	// Every Exec is checked; failure aborts InitDB immediately.

	// (a) Backfill product_masters from existing products.
	//     INSERT OR IGNORE ensures idempotent migration — if a (platform, factory_sku)
	//     pair already exists, it is skipped.
	if err := db.Exec(`
		INSERT OR IGNORE INTO product_masters (platform, factory, factory_sku, name, cover_image, extra_data, created_at, updated_at)
		SELECT platform, factory, factory_sku, name, cover_image, extra_data, created_at, updated_at
		FROM products
	`).Error; err != nil {
		return fmt.Errorf("migrate: backfill product_masters failed: %w", err)
	}

	// (b) Backfill products.product_master_id from product_masters.
	if err := db.Exec(`
		UPDATE products SET product_master_id = (
			SELECT pm.id FROM product_masters pm
			WHERE pm.platform = products.platform AND pm.factory_sku = products.factory_sku
			LIMIT 1
		) WHERE product_master_id IS NULL
	`).Error; err != nil {
		return fmt.Errorf("migrate: backfill product_master_id failed: %w", err)
	}

	// (c) ProductMaster dedup — merge references before deleting duplicates.
	//     Canonical: smallest ID per (platform, factory_sku) group.
	//     Step 1: re-point products.product_master_id to canonical.
	if err := db.Exec(`
		UPDATE products SET product_master_id = (
			SELECT MIN(pm2.id) FROM product_masters pm2
			WHERE pm2.platform  = (SELECT pm3.platform   FROM product_masters pm3 WHERE pm3.id = products.product_master_id)
			  AND pm2.factory_sku = (SELECT pm3.factory_sku FROM product_masters pm3 WHERE pm3.id = products.product_master_id)
		)
		WHERE product_master_id IS NOT NULL
	`).Error; err != nil {
		return fmt.Errorf("migrate: re-point products.product_master_id to canonical failed: %w", err)
	}

	//     Step 2: re-point product_master_images.product_master_id to canonical.
	if err := db.Exec(`
		UPDATE product_master_images SET product_master_id = (
			SELECT MIN(pm2.id) FROM product_masters pm2
			WHERE pm2.platform  = (SELECT pm3.platform   FROM product_masters pm3 WHERE pm3.id = product_master_images.product_master_id)
			  AND pm2.factory_sku = (SELECT pm3.factory_sku FROM product_masters pm3 WHERE pm3.id = product_master_images.product_master_id)
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: re-point product_master_images.product_master_id to canonical failed: %w", err)
	}

	//     Step 3: delete non-canonical product_masters.
	if err := db.Exec(`
		DELETE FROM product_masters WHERE id NOT IN (
			SELECT MIN(id) FROM product_masters GROUP BY platform, factory_sku
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: delete duplicate product_masters failed: %w", err)
	}

	// (d) Product dedup — merge references before deleting duplicate snapshots.
	//     NULL wave_id rows are naturally excluded (NULL != NULL in comparisons).
	//     Canonical: smallest ID per (wave_id, platform, factory_sku) group.
	//     Pre-clean: delete dispatch_records that would violate the unique constraint
	//     after re-point (same wave+member already has a record for the canonical product).
	if err := db.Exec(`
		DELETE FROM dispatch_records WHERE id IN (
			SELECT dr1.id FROM dispatch_records dr1
			INNER JOIN products p_dup ON p_dup.id = dr1.product_id
			WHERE p_dup.wave_id IS NOT NULL
			  AND EXISTS (
			    SELECT 1 FROM products p_other
			    WHERE p_other.wave_id = p_dup.wave_id
			      AND p_other.platform = p_dup.platform
			      AND p_other.factory_sku = p_dup.factory_sku
			      AND p_other.id < p_dup.id
			  )
			  AND EXISTS (
			    SELECT 1 FROM dispatch_records dr2
			    WHERE dr2.wave_id = dr1.wave_id
			      AND dr2.member_id = dr1.member_id
			      AND dr2.product_id = (
			        SELECT MIN(p_min.id) FROM products p_min
			        WHERE p_min.wave_id = p_dup.wave_id
			          AND p_min.platform = p_dup.platform
			          AND p_min.factory_sku = p_dup.factory_sku
			      )
			  )
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: pre-clean conflicting dispatch_records before product dedup failed: %w", err)
	}

	//     Step 1: re-point dispatch_records.product_id to canonical.
	if err := db.Exec(`
		UPDATE dispatch_records SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = dispatch_records.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = dispatch_records.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = dispatch_records.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: re-point dispatch_records.product_id to canonical failed: %w", err)
	}

	//     Step 2: re-point product_tags.product_id to canonical.
	if err := db.Exec(`
		UPDATE product_tags SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = product_tags.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = product_tags.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = product_tags.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: re-point product_tags.product_id to canonical failed: %w", err)
	}

	//     Step 3: re-point product_images.product_id to canonical.
	if err := db.Exec(`
		UPDATE product_images SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = product_images.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = product_images.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = product_images.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		return fmt.Errorf("migrate: re-point product_images.product_id to canonical failed: %w", err)
	}

	//     Step 4: delete non-canonical products (only those with non-NULL wave_id).
	if err := db.Exec(`
		DELETE FROM products WHERE id NOT IN (
			SELECT MIN(id) FROM products WHERE wave_id IS NOT NULL
			GROUP BY wave_id, platform, factory_sku
		) AND wave_id IS NOT NULL
	`).Error; err != nil {
		return fmt.Errorf("migrate: delete duplicate products failed: %w", err)
	}

	// (e) Create unique indexes.  All use IF NOT EXISTS for safe re-runs.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_product_master_platform_sku ON product_masters(platform, factory_sku)").Error; err != nil {
		return fmt.Errorf("migrate: create idx_product_master_platform_sku failed: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_product_wave_platform_sku ON products(wave_id, platform, factory_sku)").Error; err != nil {
		return fmt.Errorf("migrate: create idx_product_wave_platform_sku failed: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pmi_master_path ON product_master_images(product_master_id, path)").Error; err != nil {
		return fmt.Errorf("migrate: create idx_pmi_master_path failed: %w", err)
	}

	// (f) Orphan cleanup — delete products with wave_id IS NULL that have no
	//     references in dispatch_records, product_tags, or product_images.
	if err := db.Exec(`
		DELETE FROM products WHERE wave_id IS NULL
		  AND id NOT IN (SELECT DISTINCT product_id FROM dispatch_records)
		  AND id NOT IN (SELECT DISTINCT product_id FROM product_tags)
		  AND id NOT IN (SELECT DISTINCT product_id FROM product_images)
	`).Error; err != nil {
		return fmt.Errorf("migrate: orphan product cleanup failed: %w", err)
	}

	return nil
}

func autoMigrateTablesPhaseTwo(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Member{},
		&model.MemberNickname{},
		&model.MemberAddress{},
		&model.ProductMaster{},
		&model.Product{},
		&model.ProductTag{},
		&model.ProductImage{},
		&model.ProductMasterImage{},
		&model.Wave{},
		&model.DispatchRecord{},
		&model.TemplateConfig{},
		&model.WaveMember{},
	); err != nil {
		return fmt.Errorf("auto migrate database tables phase two failed: %w", err)
	}
	return nil
}
