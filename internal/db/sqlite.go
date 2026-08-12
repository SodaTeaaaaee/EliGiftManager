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

// SetDefaultDB stores the app-wide DB instance.
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
	backup, err := backupDatabaseBeforeMigration(cleanedPath)
	if err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: pre-migration backup: %w", err)
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

	// SQLite single-writer model: limit to 1 connection.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Performance and integrity PRAGMAs.
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA foreign_keys = ON;")
	db.Exec("PRAGMA busy_timeout = 5000;")

	// Customer resolution and merge audit schema is critical data infrastructure.
	// It is applied through the checksummed ledger before the legacy best-effort
	// AutoMigrate set so partial failures remain visible and safely retryable.
	if err := runSchemaMigrations(db, backup, customerResolutionMigrations()); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize SQLite database failed: versioned migrations: %w", err)
	}
	if err := runBatchedDataMigrations(db, legacyCustomerDataMigrations()); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize SQLite database failed: data migrations: %w", err)
	}

	// AutoMigrate remains for non-critical legacy schema until each area receives
	// a versioned migration. Customer resolution models are intentionally absent.
	if err := db.AutoMigrate(
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.FulfillmentLine{},
		&persistence.AllocationPolicyRule{},
		&persistence.SupplierOrder{},
		&persistence.SupplierOrderLine{},
		&persistence.WaveDemandAssignment{},
		&persistence.Shipment{},
		&persistence.ShipmentLine{},
		&persistence.ChannelSyncJob{},
		&persistence.ChannelSyncItem{},
		&persistence.IntegrationProfile{},
		&persistence.ChannelClosureDecisionRecord{},
		&persistence.FulfillmentAdjustment{},
		&persistence.DocumentTemplate{},
		&persistence.IntegrationProfileTemplateBinding{},
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.HistoryCheckpoint{},
		&persistence.HistoryPin{},
		&persistence.ProductMaster{},
		&persistence.Product{},
		&persistence.CarrierMapping{},
		&persistence.MergeSuggestion{},
	); err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: auto migrate: %w", err)
	}

	// Partial unique index: at most one default binding per (profile, document_type).
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_binding_one_default
		ON integration_profile_template_bindings (integration_profile_id, document_type)
		WHERE is_default = true`).Error; err != nil {
		return nil, fmt.Errorf("initialize SQLite database failed: create idx_binding_one_default: %w", err)
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
