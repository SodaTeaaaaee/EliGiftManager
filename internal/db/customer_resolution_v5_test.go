package db

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomerResolutionV5IsAppendOnlyAndIdempotent(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, apply := range []func(*gorm.DB) error{applyCustomerResolutionV1, applyCustomerResolutionV2,
		applyCustomerResolutionV3, applyCustomerResolutionV4, applyCustomerResolutionV5, applyCustomerResolutionV5} {
		if err := apply(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"customer_split_records", "split_moved_entities", "customer_split_operation_events"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing v5 table %s", table)
		}
	}
	migrations := customerResolutionMigrations()
	if len(migrations) < 5 || migrations[4].Version != 5 || migrations[4].Name != "customer_split_execution_audit_v5" {
		t.Fatalf("unexpected v5 migration registration: %+v", migrations)
	}

	now := time.Now().UTC()
	insertRecord := `INSERT INTO customer_split_records
(operation_key,source_profile_id,target_profile_id,actor_ref,payload,created_at) VALUES (?,?,?,?,?,?)`
	if err := db.Exec(insertRecord, "split-once", 1, 2, "test", `{}`, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(insertRecord, "split-once", 1, 3, "test", `{}`, now).Error; err == nil {
		t.Fatal("expected split operation-key uniqueness")
	}
	insertMoved := `INSERT INTO split_moved_entities
(split_record_id,entity_type,entity_id,from_profile_id,to_profile_id,mutation_kind,before_snapshot,after_snapshot,created_at)
VALUES (1,'identity',9,1,2,'split_reassign','{}','{}',?)`
	if err := db.Exec(insertMoved, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(insertMoved, now).Error; err == nil {
		t.Fatal("expected exact split moved-row uniqueness")
	}
	insertEvent := `INSERT INTO customer_split_operation_events
(split_record_id,event_key,operation_key,event_type,status,created_at) VALUES (1,'split:event','split-once','completed','completed',?)`
	if err := db.Exec(insertEvent, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(insertEvent, now).Error; err == nil {
		t.Fatal("expected split event-key uniqueness")
	}
}

func TestCustomerResolutionV5ChecksumFrozen(t *testing.T) {
	t.Parallel()
	migrations := customerResolutionMigrations()
	if len(migrations) < 5 {
		t.Fatalf("missing v5 migration: %+v", migrations)
	}
	const want = "de82a03c53ba985a4d3db1c5fd2a978a65cf122931228f8098bd8317e392cc07"
	if got := migrations[4].checksum(); got != want {
		t.Fatalf("v5 migration checksum changed: got %s want %s", got, want)
	}
}
