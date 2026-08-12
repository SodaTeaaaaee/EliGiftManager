package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// v4 is append-only. It adds the immutable execution ledger used by audited
// merge, history, and undo dry-run. Never edit v1/v2/v3 or reuse this version.
var customerResolutionV4TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS customer_merge_operation_events (
id integer PRIMARY KEY AUTOINCREMENT,
merge_record_id integer,
event_key text NOT NULL DEFAULT '',
operation_key text NOT NULL DEFAULT '',
event_type text NOT NULL,
status text NOT NULL,
actor_ref text NOT NULL DEFAULT '',
reason_code text NOT NULL DEFAULT '',
payload text NOT NULL DEFAULT '',
created_at datetime NOT NULL
)`,
}

var customerResolutionV4Columns = []migrationColumn{
	{table: "customer_merge_records", name: "row_version", definition: "integer NOT NULL DEFAULT 1"},
	{table: "customer_merge_records", name: "operation_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "command_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "preview_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "move_plan_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "status", definition: "text NOT NULL DEFAULT 'completed'"},
	{table: "customer_merge_records", name: "depends_on_merge_record_id", definition: "integer"},
	{table: "customer_merge_records", name: "source_row_version_after", definition: "integer NOT NULL DEFAULT 0"},
	{table: "customer_merge_records", name: "target_row_version_after", definition: "integer NOT NULL DEFAULT 0"},
	{table: "customer_merge_records", name: "source_profile_snapshot", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "target_profile_snapshot", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "completed_at", definition: "datetime"},
	{table: "customer_merge_records", name: "undo_operation_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "last_undo_plan_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "last_undo_checked_at", definition: "datetime"},
	{table: "customer_merge_records", name: "undone_by", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "undo_reason", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "undone_source_row_version", definition: "integer NOT NULL DEFAULT 0"},
	{table: "customer_merge_records", name: "undone_target_row_version", definition: "integer NOT NULL DEFAULT 0"},

	{table: "merge_moved_entities", name: "mutation_kind", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "restore_mode", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "snapshot_version", definition: "integer NOT NULL DEFAULT 0"},
	{table: "merge_moved_entities", name: "after_snapshot", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "after_state_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "entity_updated_at_after", definition: "datetime"},
	{table: "merge_moved_entities", name: "undo_state", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "undo_blocker_code", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_moved_entities", name: "revert_operation_key", definition: "text NOT NULL DEFAULT ''"},

	{table: "merge_candidates", name: "row_version", definition: "integer NOT NULL DEFAULT 1"},
	{table: "merge_candidates", name: "executed_merge_record_id", definition: "integer"},
	{table: "merge_candidates", name: "executed_at", definition: "datetime"},
}

var customerResolutionV4Backfill = []string{
	`UPDATE customer_merge_records
SET status = CASE WHEN undone_at IS NULL THEN 'completed' ELSE 'undone' END,
    completed_at = COALESCE(completed_at, created_at)
WHERE operation_key = '' AND move_plan_hash = ''`,
}

var customerResolutionV4Indexes = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_records_operation_key_v4
ON customer_merge_records (operation_key) WHERE operation_key <> ''`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_records_undo_operation_key_v4
ON customer_merge_records (undo_operation_key) WHERE undo_operation_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_records_source_history_v4
ON customer_merge_records (source_profile_id, created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_records_target_history_v4
ON customer_merge_records (target_profile_id, created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_records_status_history_v4
ON customer_merge_records (status, created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_records_dependency_v4
ON customer_merge_records (depends_on_merge_record_id, status)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_moved_entity_once_v4
ON merge_moved_entities (merge_record_id, entity_type, entity_id, mutation_kind)
WHERE mutation_kind <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_order_v4
ON merge_moved_entities (merge_record_id, move_order, id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_entity_state_v4
ON merge_moved_entities (entity_type, entity_id, reverted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_from_history_v4
ON merge_moved_entities (from_profile_id, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_to_history_v4
ON merge_moved_entities (to_profile_id, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_revert_key_v4
ON merge_moved_entities (revert_operation_key) WHERE revert_operation_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_executed_record_v4
ON merge_candidates (executed_merge_record_id) WHERE executed_merge_record_id IS NOT NULL`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_status_version_v4
ON merge_candidates (status, row_version, last_evaluated_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_operation_event_key_v4
ON customer_merge_operation_events (event_key) WHERE event_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_operation_events_operation_v4
ON customer_merge_operation_events (operation_key, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_operation_events_record_v4
ON customer_merge_operation_events (merge_record_id, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_operation_events_status_v4
ON customer_merge_operation_events (event_type, status, created_at)`,
}

func customerResolutionV4Signature() string {
	parts := append([]string{}, customerResolutionV4TableDDL...)
	for _, column := range customerResolutionV4Columns {
		parts = append(parts, column.table+"."+column.name+" "+column.definition)
	}
	parts = append(parts, customerResolutionV4Backfill...)
	parts = append(parts, customerResolutionV4Indexes...)
	return strings.Join(parts, "\n")
}

func applyCustomerResolutionV4(tx *gorm.DB) error {
	for _, statement := range customerResolutionV4TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v4 table: %w", err)
		}
	}
	for _, column := range customerResolutionV4Columns {
		if tx.Migrator().HasColumn(column.table, column.name) {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", column.table, column.name, column.definition)
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("add customer resolution v4 column %s.%s: %w", column.table, column.name, err)
		}
	}
	for _, statement := range customerResolutionV4Backfill {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("backfill customer resolution v4: %w", err)
		}
	}
	for _, statement := range customerResolutionV4Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v4 index: %w", err)
		}
	}
	return nil
}
