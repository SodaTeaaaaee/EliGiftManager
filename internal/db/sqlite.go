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

	db, err := gorm.Open(sqlite.Open(cleanedPath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
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

	// Performance and integrity PRAGMAs.
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA foreign_keys = ON;")
	db.Exec("PRAGMA busy_timeout = 5000;")

	// AutoMigrate: V2 persistence models for the first vertical slice.
	if err := db.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerIdentity{},
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
	); err != nil {
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
