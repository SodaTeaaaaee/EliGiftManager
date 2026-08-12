package db

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomerResolutionV4IsAppendOnlyIdempotentAndBackfillsLegacy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, apply := range []func(*gorm.DB) error{applyCustomerResolutionV1, applyCustomerResolutionV2, applyCustomerResolutionV3} {
		if err := apply(db); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO customer_merge_records
(source_profile_id,target_profile_id,payload,created_at,undone_at) VALUES (1,2,'{}',?,NULL),(3,4,'{}',?,?)`, now, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO merge_moved_entities
(merge_record_id,entity_type,entity_id,from_profile_id,to_profile_id,move_order,before_snapshot,created_at)
VALUES (1,'identity',9,1,2,1,'{}',?),(1,'identity',9,1,2,2,'{}',?)`, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV4(db); err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV4(db); err != nil {
		t.Fatalf("v4 rerun was not idempotent: %v", err)
	}
	for table, columns := range map[string][]string{
		"customer_merge_records": {"row_version", "operation_key", "move_plan_hash", "status", "undo_operation_key"},
		"merge_moved_entities":   {"mutation_kind", "restore_mode", "snapshot_version", "after_snapshot", "after_state_hash"},
		"merge_candidates":       {"row_version", "executed_merge_record_id", "executed_at"},
	} {
		for _, column := range columns {
			if !db.Migrator().HasColumn(table, column) {
				t.Errorf("missing v4 column %s.%s", table, column)
			}
		}
	}
	if !db.Migrator().HasTable("customer_merge_operation_events") {
		t.Fatal("missing v4 operation event table")
	}
	var statuses []string
	if err := db.Raw("SELECT status FROM customer_merge_records ORDER BY id").Scan(&statuses).Error; err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0] != "completed" || statuses[1] != "undone" {
		t.Fatalf("unexpected legacy status backfill: %v", statuses)
	}
	if err := db.Exec(`UPDATE customer_merge_records SET operation_key='same' WHERE id=1`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE customer_merge_records SET operation_key='same' WHERE id=2`).Error; err == nil {
		t.Fatal("expected non-empty operation key uniqueness")
	}
	if err := db.Exec(`UPDATE merge_moved_entities SET mutation_kind='reassign' WHERE id=1`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE merge_moved_entities SET mutation_kind='reassign' WHERE id=2`).Error; err == nil {
		t.Fatal("expected exact moved ledger uniqueness for v4 rows")
	}
	migrations := customerResolutionMigrations()
	if len(migrations) < 4 || migrations[3].Version != 4 || migrations[3].Signature == "" {
		t.Fatalf("unexpected migration sequence: %+v", migrations)
	}
}

func TestCustomerResolutionV1ToV4ChecksumsRemainFrozen(t *testing.T) {
	want := []string{
		"aec3364f1241d913333d46b8e4abc2a4308c891254363043dc485ca6164dee8d",
		"b2711b0c63b07528ec7082919641bab9de8a97e9965d2cb509d0eb1facaf94b8",
		"3536e7d86199f65ff7ff605cfcbc0446d366242830fa7e343c580fc469e009ab",
		"15918966df483f6aea9405300dece3d83654500c85f796559ce16695aa78c274",
	}
	migrations := customerResolutionMigrations()
	for i := range want {
		if got := migrations[i].checksum(); got != want[i] {
			t.Fatalf("migration v%d checksum changed: got %s want %s", i+1, got, want[i])
		}
	}
}
