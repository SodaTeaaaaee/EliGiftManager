package db

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitDBAddsNullableFactoryProfileToLegacySupplierOrders(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if err := legacy.Exec(`CREATE TABLE supplier_orders (
		id integer primary key autoincrement,
		wave_id integer NOT NULL,
		supplier_platform text
	)`).Error; err != nil {
		t.Fatalf("create legacy supplier_orders: %v", err)
	}
	if err := legacy.Exec(`INSERT INTO supplier_orders (wave_id, supplier_platform) VALUES (7, 'legacy')`).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	if sqlDB, err := legacy.DB(); err == nil {
		_ = sqlDB.Close()
	}

	migrated, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB migration: %v", err)
	}
	if sqlDB, dbErr := migrated.DB(); dbErr == nil {
		defer sqlDB.Close()
	}
	if !migrated.Migrator().HasColumn("supplier_orders", "factory_integration_profile_id") {
		t.Fatal("factory_integration_profile_id column was not added")
	}
	var row struct {
		FactoryIntegrationProfileID *uint
	}
	if err := migrated.Table("supplier_orders").Select("factory_integration_profile_id").Where("id = 1").Scan(&row).Error; err != nil {
		t.Fatalf("read migrated legacy row: %v", err)
	}
	if row.FactoryIntegrationProfileID != nil {
		t.Fatalf("legacy row profile ID = %v, want nil", row.FactoryIntegrationProfileID)
	}
}
