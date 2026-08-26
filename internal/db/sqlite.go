package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var defaultDB *gorm.DB

func SetDefaultDB(db *gorm.DB) { defaultDB = db }

func GetDB() *gorm.DB { return defaultDB }

func AutoMigrateAll(db *gorm.DB) error {
	if err := db.AutoMigrate(persistence.AllModels()...); err != nil {
		return err
	}
	// Recreate: IF NOT EXISTS would keep a previous WHERE that treated exported rows as occupying the slot.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_supplier_orders_wave_factory_open`).Error; err != nil {
		return fmt.Errorf("drop open-order unique index: %w", err)
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_input_facts_stable ON input_facts (platform_id, stable_external_id) WHERE stable_external_id != ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_supplier_orders_wave_factory_open ON supplier_orders (wave_id, factory_platform_id) WHERE status IN ('draft','generated')`,
	}
	for _, sql := range statements {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("unique index: %w", err)
		}
	}
	return nil
}

func InitDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("initialize SQLite database failed: database path is required")
	}
	cleanedPath := filepath.Clean(dbPath)
	if err := ensureDatabaseDir(cleanedPath); err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: %w", err)
	}
	db, err := gorm.Open(sqlite.Open(cleanedPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
		Logger:                                   logger.Default.LogMode(logger.Error),
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
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.Exec("PRAGMA journal_mode = WAL;").Error; err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: set journal_mode WAL: %w", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: enable foreign_keys: %w", err)
	}
	var foreignKeys int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: verify foreign_keys: %w", err)
	}
	if foreignKeys != 1 {
		return nil, fmt.Errorf("initialize SQLite database failed: foreign_keys pragma did not take effect (got %d)", foreignKeys)
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000;").Error; err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: set busy_timeout: %w", err)
	}
	if err := AutoMigrateAll(db); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize SQLite database failed: auto migrate: %w", err)
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
