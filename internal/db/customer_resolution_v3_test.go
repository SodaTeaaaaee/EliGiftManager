package db

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomerResolutionV3IsAppendOnlyAndIdempotent(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV1(gdb); err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV2(gdb); err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV3(gdb); err != nil {
		t.Fatal(err)
	}
	if err := applyCustomerResolutionV3(gdb); err != nil {
		t.Fatalf("v3 rerun was not idempotent: %v", err)
	}
	for table, columns := range map[string][]string{
		"merge_policies":         {"row_version", "needs_scan", "last_scan_at"},
		"merge_policy_revisions": {"schema_version"},
		"merge_candidates":       {"canonical_pair_key", "evidence_hash", "policy_version", "explanation_code", "scan_run_id"},
		"merge_evidence":         {"evidence_key", "polarity", "value_hash", "masked_value"},
	} {
		for _, column := range columns {
			if !gdb.Migrator().HasColumn(table, column) {
				t.Errorf("missing v3 column %s.%s", table, column)
			}
		}
	}
	if !gdb.Migrator().HasTable("merge_scan_runs") {
		t.Fatal("missing merge_scan_runs")
	}
	migrations := customerResolutionMigrations()
	if len(migrations) < 3 || migrations[2].Version != 3 || migrations[2].Signature == "" {
		t.Fatalf("unexpected immutable migration sequence: %+v", migrations)
	}
}
