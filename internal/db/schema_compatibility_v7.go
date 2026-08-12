package db

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"
)

var sqliteIdentifierV7 = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// V7 repairs columns skipped by older GORM Migrator.HasColumn(table string, ...)
// calls. It uses table-scoped SQLite PRAGMA inspection and never mutates the
// checksummed declarations for V1-V6.
var schemaCompatibilityV7Columns = func() []migrationColumn {
	columns := append([]migrationColumn{}, customerResolutionV1Columns...)
	columns = append(columns, customerResolutionV2Columns...)
	columns = append(columns, customerResolutionV3Columns...)
	columns = append(columns, customerResolutionV4Columns...)
	columns = append(columns,
		// Complete V4 audit for the three execution-critical tables.
		migrationColumn{"customer_merge_records", "id", "integer"},
		migrationColumn{"customer_merge_records", "source_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"customer_merge_records", "target_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"customer_merge_records", "payload", "text NOT NULL DEFAULT ''"},
		migrationColumn{"customer_merge_records", "created_at", "datetime"},
		migrationColumn{"customer_merge_records", "undone_at", "datetime"},
		migrationColumn{"merge_moved_entities", "id", "integer"},
		migrationColumn{"merge_moved_entities", "merge_record_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_moved_entities", "entity_type", "text NOT NULL DEFAULT ''"},
		migrationColumn{"merge_moved_entities", "entity_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_moved_entities", "from_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_moved_entities", "to_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_moved_entities", "move_order", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_moved_entities", "before_snapshot", "text NOT NULL DEFAULT ''"},
		migrationColumn{"merge_moved_entities", "reverted_at", "datetime"},
		migrationColumn{"merge_moved_entities", "created_at", "datetime"},
		migrationColumn{"merge_candidates", "id", "integer"},
		migrationColumn{"merge_candidates", "created_at", "datetime"},
		migrationColumn{"merge_candidates", "updated_at", "datetime"},
		migrationColumn{"merge_candidates", "deleted_at", "datetime"},
		migrationColumn{"merge_candidates", "source_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_candidates", "target_profile_id", "integer NOT NULL DEFAULT 0"},
		migrationColumn{"merge_candidates", "status", "text NOT NULL DEFAULT 'pending'"},
		migrationColumn{"merge_candidates", "score", "real NOT NULL DEFAULT 0"},
		migrationColumn{"merge_candidates", "merge_policy_revision_id", "integer"},
		migrationColumn{"merge_candidates", "reason", "text NOT NULL DEFAULT ''"},
	)
	columns = append(columns, v5CompatibilityColumns()...)
	columns = append(columns, v6CompatibilityColumns()...)
	return columns
}()

func v5CompatibilityColumns() []migrationColumn {
	return []migrationColumn{
		{"customer_split_records", "id", "integer"}, {"customer_split_records", "operation_key", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "command_hash", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "preview_hash", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "move_plan_hash", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "status", "text NOT NULL DEFAULT 'executing'"}, {"customer_split_records", "source_profile_id", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "target_profile_id", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "target_strategy", "text NOT NULL DEFAULT 'create_new'"}, {"customer_split_records", "actor_ref", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "decision_reason", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "source_row_version", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "target_row_version", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "source_row_version_after", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "target_row_version_after", "integer NOT NULL DEFAULT 0"}, {"customer_split_records", "source_profile_snapshot", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "target_profile_snapshot", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "payload", "text NOT NULL DEFAULT ''"}, {"customer_split_records", "row_version", "integer NOT NULL DEFAULT 1"}, {"customer_split_records", "reverse_operation_kind", "text NOT NULL DEFAULT 'manual_merge_required'"}, {"customer_split_records", "created_at", "datetime"}, {"customer_split_records", "completed_at", "datetime"},
		{"split_moved_entities", "id", "integer"}, {"split_moved_entities", "split_record_id", "integer NOT NULL DEFAULT 0"}, {"split_moved_entities", "entity_type", "text NOT NULL DEFAULT ''"}, {"split_moved_entities", "entity_id", "integer NOT NULL DEFAULT 0"}, {"split_moved_entities", "from_profile_id", "integer NOT NULL DEFAULT 0"}, {"split_moved_entities", "to_profile_id", "integer NOT NULL DEFAULT 0"}, {"split_moved_entities", "move_order", "integer NOT NULL DEFAULT 0"}, {"split_moved_entities", "before_snapshot", "text NOT NULL DEFAULT ''"}, {"split_moved_entities", "after_snapshot", "text NOT NULL DEFAULT ''"}, {"split_moved_entities", "after_state_hash", "text NOT NULL DEFAULT ''"}, {"split_moved_entities", "mutation_kind", "text NOT NULL DEFAULT ''"}, {"split_moved_entities", "snapshot_version", "integer NOT NULL DEFAULT 1"}, {"split_moved_entities", "created_at", "datetime"},
		{"customer_split_operation_events", "id", "integer"}, {"customer_split_operation_events", "split_record_id", "integer NOT NULL DEFAULT 0"}, {"customer_split_operation_events", "event_key", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "operation_key", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "event_type", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "status", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "actor_ref", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "reason_code", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "payload", "text NOT NULL DEFAULT ''"}, {"customer_split_operation_events", "created_at", "datetime"},
	}
}

func v6CompatibilityColumns() []migrationColumn {
	return []migrationColumn{
		{"import_runs", "id", "integer"}, {"import_runs", "run_key", "text NOT NULL DEFAULT ''"}, {"import_runs", "import_kind", "text NOT NULL DEFAULT ''"}, {"import_runs", "integration_profile_id", "integer"}, {"import_runs", "source_format", "text NOT NULL DEFAULT ''"}, {"import_runs", "source_file_name", "text NOT NULL DEFAULT ''"}, {"import_runs", "import_mode", "text NOT NULL DEFAULT 'skip_invalid'"}, {"import_runs", "status", "text NOT NULL DEFAULT 'running'"}, {"import_runs", "retention_days", "integer NOT NULL DEFAULT 90"}, {"import_runs", "retention_policy_version", "integer NOT NULL DEFAULT 1"}, {"import_runs", "expires_at", "datetime"}, {"import_runs", "record_count", "integer NOT NULL DEFAULT 0"}, {"import_runs", "success_count", "integer NOT NULL DEFAULT 0"}, {"import_runs", "failure_count", "integer NOT NULL DEFAULT 0"}, {"import_runs", "quarantined_count", "integer NOT NULL DEFAULT 0"}, {"import_runs", "parser_metadata", "text NOT NULL DEFAULT ''"}, {"import_runs", "created_at", "datetime"}, {"import_runs", "completed_at", "datetime"},
		{"import_raw_records", "id", "integer"}, {"import_raw_records", "import_run_id", "integer NOT NULL DEFAULT 0"}, {"import_raw_records", "row_index", "integer NOT NULL DEFAULT 0"}, {"import_raw_records", "raw_logical_row", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "unmapped_source", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "parser_metadata", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "warning_codes", "text NOT NULL DEFAULT '[]'"}, {"import_raw_records", "asset_members", "text NOT NULL DEFAULT '[]'"}, {"import_raw_records", "outcome", "text NOT NULL DEFAULT 'pending'"}, {"import_raw_records", "error_code", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "error_message", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "result_type", "text NOT NULL DEFAULT ''"}, {"import_raw_records", "result_id", "integer"}, {"import_raw_records", "retention_days", "integer NOT NULL DEFAULT 90"}, {"import_raw_records", "expires_at", "datetime"}, {"import_raw_records", "created_at", "datetime"},
		{"import_evidence_settings", "id", "integer"}, {"import_evidence_settings", "retention_days", "integer NOT NULL DEFAULT 90"}, {"import_evidence_settings", "revision", "integer NOT NULL DEFAULT 1"}, {"import_evidence_settings", "updated_at", "datetime"},
		{"external_carriers", "id", "integer"}, {"external_carriers", "integration_profile_id", "integer NOT NULL DEFAULT 0"}, {"external_carriers", "canonical_key", "text NOT NULL DEFAULT ''"}, {"external_carriers", "external_carrier_code", "text NOT NULL DEFAULT ''"}, {"external_carriers", "external_carrier_name", "text NOT NULL DEFAULT ''"}, {"external_carriers", "name_key_strategy", "text NOT NULL DEFAULT 'code_or_normalized_name_v1'"}, {"external_carriers", "internal_carrier_code", "text"}, {"external_carriers", "status", "text NOT NULL DEFAULT 'provisional'"}, {"external_carriers", "conflict_reason", "text NOT NULL DEFAULT ''"}, {"external_carriers", "source_import_run_id", "integer"}, {"external_carriers", "source_raw_record_id", "integer"}, {"external_carriers", "created_at", "datetime"}, {"external_carriers", "updated_at", "datetime"},
		{"external_carrier_conflicts", "id", "integer"}, {"external_carrier_conflicts", "integration_profile_id", "integer NOT NULL DEFAULT 0"}, {"external_carrier_conflicts", "canonical_key", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "conflict_kind", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "external_carrier_code", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "external_carrier_name", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "internal_carrier_code", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "source_import_run_id", "integer"}, {"external_carrier_conflicts", "source_raw_record_id", "integer"}, {"external_carrier_conflicts", "legacy_carrier_mapping_id", "integer"}, {"external_carrier_conflicts", "payload", "text NOT NULL DEFAULT ''"}, {"external_carrier_conflicts", "created_at", "datetime"},
	}
}

func schemaCompatibilityV7Indexes() []string {
	indexes := append([]string{}, customerResolutionV1Indexes...)
	indexes = append(indexes, customerResolutionV2Indexes...)
	indexes = append(indexes, customerResolutionV3Indexes...)
	indexes = append(indexes, customerResolutionV4Indexes...)
	indexes = append(indexes, customerResolutionV5Indexes...)
	indexes = append(indexes, importEvidenceCarrierV6Indexes...)
	return indexes
}

func schemaCompatibilityV7Signature() string {
	parts := []string{"sqlite_table_info_compatibility_audit_v1"}
	for _, column := range schemaCompatibilityV7Columns {
		parts = append(parts, column.table+"."+column.name+" "+column.definition)
	}
	parts = append(parts, schemaCompatibilityV7Indexes()...)
	return strings.Join(parts, "\n")
}

func applySchemaCompatibilityV7(tx *gorm.DB) error {
	// Recreate missing post-V3 tables before auditing their columns.
	for _, statements := range [][]string{customerResolutionV3TableDDL, customerResolutionV4TableDDL, customerResolutionV5TableDDL, importEvidenceCarrierV6TableDDL} {
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("v7 ensure table: %w", err)
			}
		}
	}
	byTable := map[string][]migrationColumn{}
	for _, column := range schemaCompatibilityV7Columns {
		byTable[column.table] = append(byTable[column.table], column)
	}
	tables := make([]string, 0, len(byTable))
	for table := range byTable {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		present, err := sqliteTableColumnsV7(tx, table)
		if err != nil {
			return err
		}
		for _, column := range byTable[table] {
			if present[column.name] {
				continue
			}
			if column.name == "id" {
				return fmt.Errorf("v7 cannot safely add missing primary key %s.id", table)
			}
			statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column.name, column.definition)
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("v7 add %s.%s: %w", table, column.name, err)
			}
			present[column.name] = true
		}
	}
	for _, statement := range customerResolutionV4Backfill {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("v7 v4 backfill: %w", err)
		}
	}
	for _, statement := range schemaCompatibilityV7Indexes() {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("v7 ensure index: %w", err)
		}
	}
	return nil
}

func sqliteTableColumnsV7(tx *gorm.DB, table string) (map[string]bool, error) {
	if !sqliteIdentifierV7.MatchString(table) {
		return nil, fmt.Errorf("invalid v7 table identifier %q", table)
	}
	type row struct {
		Name string `gorm:"column:name"`
	}
	var rows []row
	if err := tx.Raw(`PRAGMA table_info("` + table + `")`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("inspect v7 table %s: %w", table, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("v7 required table %s does not exist", table)
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Name] = true
	}
	return out, nil
}
