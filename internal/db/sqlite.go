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
	); err != nil {
		return fmt.Errorf("auto migrate database tables failed: %w", err)
	}
	return nil
}
