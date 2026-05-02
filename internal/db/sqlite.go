package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var defaultDB *gorm.DB

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

	return nil
}
