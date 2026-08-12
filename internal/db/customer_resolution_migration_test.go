package db

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomerResolutionMigrationUpgradesLegacyDatabaseWithBackup(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	legacyDDL := []string{
		`CREATE TABLE customer_profiles (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, deleted_at datetime, display_name text NOT NULL, profile_type text NOT NULL DEFAULT 'member', extra_data text)`,
		`CREATE TABLE customer_identities (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, deleted_at datetime, customer_profile_id integer NOT NULL, identity_platform text NOT NULL, identity_value text NOT NULL, identity_type text NOT NULL, is_primary numeric NOT NULL DEFAULT false, extra_data text)`,
		`CREATE TABLE customer_addresses (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, deleted_at datetime, customer_profile_id integer NOT NULL, label text, recipient_name text, phone text, country text, province text, city text, district text, address_line1 text, address_line2 text, postal_code text, is_default numeric NOT NULL DEFAULT false, is_test numeric NOT NULL DEFAULT false, validation_status text NOT NULL DEFAULT 'unvalidated', validation_detail text, extra_data text)`,
		`CREATE TABLE customer_merge_records (id integer PRIMARY KEY AUTOINCREMENT, source_profile_id integer NOT NULL, target_profile_id integer NOT NULL, payload text NOT NULL, created_at datetime NOT NULL, undone_at datetime)`,
		`INSERT INTO customer_profiles (display_name, profile_type) VALUES ('Legacy', 'member')`,
		`INSERT INTO customer_identities (customer_profile_id, identity_platform, identity_value, identity_type) VALUES (1, 'retail', 'duplicate', 'account'), (1, 'retail', 'duplicate', 'account')`,
	}
	for _, statement := range legacyDDL {
		if err := legacy.Exec(statement).Error; err != nil {
			t.Fatalf("prepare legacy database: %v", err)
		}
	}
	legacySQL, err := legacy.DB()
	if err != nil {
		t.Fatalf("get legacy SQL DB: %v", err)
	}
	if err := legacySQL.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	migrated, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := migrated.DB()
		_ = sqlDB.Close()
	})

	var ledger schemaMigrationLedger
	if err := migrated.First(&ledger, "version = ?", 1).Error; err != nil {
		t.Fatalf("read migration ledger: %v", err)
	}
	if ledger.Status != "applied" || ledger.Attempt != 1 {
		t.Fatalf("unexpected ledger state: status=%q attempt=%d", ledger.Status, ledger.Attempt)
	}
	if len(ledger.Checksum) != 64 || len(ledger.BackupChecksum) != 64 {
		t.Fatalf("expected SHA-256 checksums, migration=%q backup=%q", ledger.Checksum, ledger.BackupChecksum)
	}
	if _, err := os.Stat(ledger.BackupPath); err != nil {
		t.Fatalf("stat pre-migration backup %q: %v", ledger.BackupPath, err)
	}

	for _, table := range []string{
		"schema_migrations", "data_migrations", "customer_name_observations", "customer_name_events",
		"customer_profile_origins", "merge_candidates", "merge_evidence", "merge_policies",
		"merge_policy_revisions", "merge_moved_entities", "legacy_customer_maps",
		"legacy_customer_migration_cursors", "legacy_customer_migration_quarantines",
	} {
		if !migrated.Migrator().HasTable(table) {
			t.Errorf("expected migrated table %q", table)
		}
	}
	for table, columns := range map[string][]string{
		"customer_profiles":          {"status", "merged_into_profile_id", "row_version", "display_name_mode", "display_name_observation_id"},
		"customer_identities":        {"namespace", "normalized_value", "normalization_version", "authority", "verification_status", "source_integration_profile_id", "resolution_status", "first_seen_at", "last_seen_at"},
		"customer_addresses":         {"normalized_phone", "address_fingerprint", "normalization_version", "quality_status"},
		"customer_name_observations": {"source_event_key", "episode_key", "observation_count"},
		"customer_name_events":       {"event_key"},
	} {
		for _, column := range columns {
			if !migrated.Migrator().HasColumn(table, column) {
				t.Errorf("expected column %s.%s", table, column)
			}
		}
	}

	var profile struct {
		Status          string
		RowVersion      uint64
		DisplayNameMode string
	}
	if err := migrated.Table("customer_profiles").Select("status, row_version, display_name_mode").Where("id = ?", 1).Take(&profile).Error; err != nil {
		t.Fatalf("read migrated legacy profile: %v", err)
	}
	if profile.Status != "active" || profile.RowVersion != 1 || profile.DisplayNameMode != "auto" {
		t.Fatalf("unsafe profile defaults: %+v", profile)
	}

	var duplicateCount int64
	if err := migrated.Table("customer_identities").Where("identity_platform = ? AND identity_value = ?", "retail", "duplicate").Count(&duplicateCount).Error; err != nil {
		t.Fatalf("count retained duplicate identities: %v", err)
	}
	if duplicateCount != 2 {
		t.Fatalf("duplicate identities were not retained: got %d", duplicateCount)
	}
	assertOnlyExpectedUniqueCustomerResolutionIndexes(t, migrated)

	if err := runSchemaMigrations(migrated, nil, customerResolutionMigrations()); err != nil {
		t.Fatalf("rerun applied migrations: %v", err)
	}
	if err := migrated.First(&ledger, "version = ?", 1).Error; err != nil {
		t.Fatalf("read rerun ledger: %v", err)
	}
	if ledger.Attempt != 1 {
		t.Fatalf("applied migration reran unexpectedly: attempt=%d", ledger.Attempt)
	}
}

func TestSchemaMigrationFailureIsRecoverableAndChecksummed(t *testing.T) {
	t.Parallel()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "recovery.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
	})
	shouldFail := true
	migration := schemaMigration{Version: 99, Name: "recovery_test", Signature: "create recovery_marker v1", Up: func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TABLE IF NOT EXISTS recovery_marker (id integer PRIMARY KEY)`).Error; err != nil {
			return err
		}
		if shouldFail {
			return errors.New("injected failure")
		}
		return nil
	}}

	if err := runSchemaMigrations(database, nil, []schemaMigration{migration}); err == nil {
		t.Fatal("expected injected migration failure")
	}
	var ledger schemaMigrationLedger
	if err := database.First(&ledger, "version = ?", migration.Version).Error; err != nil {
		t.Fatalf("read failed ledger: %v", err)
	}
	if ledger.Status != "failed" || ledger.Attempt != 1 || !strings.Contains(ledger.ErrorMessage, "injected failure") {
		t.Fatalf("unexpected failed ledger: %+v", ledger)
	}

	shouldFail = false
	if err := runSchemaMigrations(database, nil, []schemaMigration{migration}); err != nil {
		t.Fatalf("retry failed migration: %v", err)
	}
	if err := database.First(&ledger, "version = ?", migration.Version).Error; err != nil {
		t.Fatalf("read recovered ledger: %v", err)
	}
	if ledger.Status != "applied" || ledger.Attempt != 2 || ledger.ErrorMessage != "" {
		t.Fatalf("unexpected recovered ledger: %+v", ledger)
	}

	tampered := migration
	tampered.Signature = "changed migration body"
	if err := runSchemaMigrations(database, nil, []schemaMigration{tampered}); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestDataMigrationLedgerIsIdempotent(t *testing.T) {
	t.Parallel()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "data-ledger.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := database.DB()
		_ = sqlDB.Close()
	})

	runs := 0
	migration := dataMigration{Version: 1, Name: "data_ledger_test", Signature: "v1", Up: func(tx *gorm.DB) (string, uint64, error) {
		runs++
		if err := tx.Exec(`CREATE TABLE IF NOT EXISTS data_marker (id integer PRIMARY KEY)`).Error; err != nil {
			return "", 0, err
		}
		return "complete", 7, nil
	}}
	if err := runDataMigrations(database, []dataMigration{migration}); err != nil {
		t.Fatalf("run data migration: %v", err)
	}
	if err := runDataMigrations(database, []dataMigration{migration}); err != nil {
		t.Fatalf("rerun data migration: %v", err)
	}
	if runs != 1 {
		t.Fatalf("data migration was not idempotent: runs=%d", runs)
	}
	var ledger dataMigrationLedger
	if err := database.First(&ledger, "version = ?", migration.Version).Error; err != nil {
		t.Fatalf("read data migration ledger: %v", err)
	}
	if ledger.Status != "applied" || ledger.Checkpoint != "complete" || ledger.RowsProcessed != 7 {
		t.Fatalf("unexpected data migration ledger: %+v", ledger)
	}
}

func assertOnlyExpectedUniqueCustomerResolutionIndexes(t *testing.T, database *gorm.DB) {
	t.Helper()
	tables := []string{"customer_profiles", "customer_identities", "customer_addresses", "customer_name_observations", "customer_profile_origins", "merge_candidates"}
	allowed := map[string]bool{
		"idx_customer_name_observations_source_event_key": true,
		"idx_customer_profile_origins_source_external":    true,
		"idx_merge_candidates_evaluation_v3":              true,
	}
	for _, table := range tables {
		type indexRow struct {
			Name   string
			Unique int
		}
		var indexes []indexRow
		if err := database.Raw("PRAGMA index_list(" + table + ")").Scan(&indexes).Error; err != nil {
			t.Fatalf("list indexes for %s: %v", table, err)
		}
		for _, index := range indexes {
			if index.Unique != 0 && !allowed[index.Name] {
				t.Errorf("unexpected unique index %q on %s", index.Name, table)
			}
		}
	}
}
