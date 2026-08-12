package db

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSchemaCompatibilityV7FreshDBSupportsMergeRecordCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh-v7.db")
	db, err := InitDB(path)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	assertV7CriticalColumns(t, db)
	now := time.Now().UTC()
	record := &persistence.CustomerMergeRecord{SourceProfileID: 1, TargetProfileID: 2, Payload: "{}", RowVersion: 1, Status: "completed", CreatedAt: now, CompletedAt: &now}
	if err := db.WithContext(context.Background()).Create(record).Error; err != nil {
		t.Fatalf("create CustomerMergeRecord after fresh v7: %v", err)
	}
}

func TestSchemaCompatibilityV7UpgradesEveryPriorLedgerStage(t *testing.T) {
	for _, stage := range []int{3, 4, 5, 6} {
		stage := stage
		t.Run(fmt.Sprintf("from_v%d", stage), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), fmt.Sprintf("from-v%d.db", stage))
			db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })
			migrations := customerResolutionMigrations()
			if err := runSchemaMigrations(db, nil, migrations[:stage]); err != nil {
				t.Fatalf("apply through v%d: %v", stage, err)
			}
			if err := runSchemaMigrations(db, nil, migrations[stage:]); err != nil {
				t.Fatalf("upgrade v%d through v7: %v", stage, err)
			}
			assertV7CriticalColumns(t, db)
			now := time.Now().UTC()
			record := &persistence.CustomerMergeRecord{SourceProfileID: 11, TargetProfileID: 12, Payload: "{}", RowVersion: 1, Status: "completed", CreatedAt: now, CompletedAt: &now}
			if err := db.Create(record).Error; err != nil {
				t.Fatalf("GORM create after v%d upgrade: %v", stage, err)
			}
		})
	}
}

func assertV7CriticalColumns(t *testing.T, db *gorm.DB) {
	t.Helper()
	expected := map[string][]string{"customer_merge_records": {"row_version", "operation_key", "undo_operation_key", "undone_target_row_version"}, "merge_moved_entities": {"mutation_kind", "restore_mode", "after_state_hash", "revert_operation_key"}, "merge_candidates": {"row_version", "executed_merge_record_id", "executed_at"}, "customer_split_records": {"row_version", "reverse_operation_kind"}, "import_runs": {"retention_policy_version", "parser_metadata"}, "external_carriers": {"canonical_key", "internal_carrier_code", "status"}}
	for table, columns := range expected {
		present, err := sqliteTableColumnsV7(db, table)
		if err != nil {
			t.Fatal(err)
		}
		for _, column := range columns {
			if !present[column] {
				t.Fatalf("%s missing %s", table, column)
			}
		}
	}
}

func TestV6V7MigrationChecksumsFrozen(t *testing.T) {
	migrations := customerResolutionMigrations()
	expected := map[int]string{6: "b7f561e19e191cd35c03c7098d8d545d127b07d5cb7a8b56d95d5a79297bcab1", 7: "69279a85d1d3acf2585927cf576462d23b02c31546509f63b968dfa1a2def350"}
	for version, want := range expected {
		got := migrations[version-1].checksum()
		if got != want {
			t.Fatalf("v%d checksum got %s want %s", version, got, want)
		}
	}
}
