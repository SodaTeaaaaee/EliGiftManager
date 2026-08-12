package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// v5 is append-only. It adds the immutable audit records and exact moved-row
// ledger for explicit customer split operations. Never edit v1-v4 or reuse
// this version for unrelated schema.
var customerResolutionV5TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS customer_split_records (
id integer PRIMARY KEY AUTOINCREMENT,
operation_key text NOT NULL DEFAULT '',
command_hash text NOT NULL DEFAULT '',
preview_hash text NOT NULL DEFAULT '',
move_plan_hash text NOT NULL DEFAULT '',
status text NOT NULL DEFAULT 'executing',
source_profile_id integer NOT NULL,
target_profile_id integer NOT NULL,
target_strategy text NOT NULL DEFAULT 'create_new',
actor_ref text NOT NULL DEFAULT '',
decision_reason text NOT NULL DEFAULT '',
source_row_version integer NOT NULL DEFAULT 0,
target_row_version integer NOT NULL DEFAULT 0,
source_row_version_after integer NOT NULL DEFAULT 0,
target_row_version_after integer NOT NULL DEFAULT 0,
source_profile_snapshot text NOT NULL DEFAULT '',
target_profile_snapshot text NOT NULL DEFAULT '',
payload text NOT NULL DEFAULT '',
row_version integer NOT NULL DEFAULT 1,
reverse_operation_kind text NOT NULL DEFAULT 'manual_merge_required',
created_at datetime NOT NULL,
completed_at datetime
)`,
	`CREATE TABLE IF NOT EXISTS split_moved_entities (
id integer PRIMARY KEY AUTOINCREMENT,
split_record_id integer NOT NULL,
entity_type text NOT NULL,
entity_id integer NOT NULL,
from_profile_id integer NOT NULL,
to_profile_id integer NOT NULL,
move_order integer NOT NULL DEFAULT 0,
before_snapshot text NOT NULL DEFAULT '',
after_snapshot text NOT NULL DEFAULT '',
after_state_hash text NOT NULL DEFAULT '',
mutation_kind text NOT NULL DEFAULT '',
snapshot_version integer NOT NULL DEFAULT 1,
created_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS customer_split_operation_events (
id integer PRIMARY KEY AUTOINCREMENT,
split_record_id integer NOT NULL,
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

var customerResolutionV5Indexes = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_split_records_operation_key_v5
ON customer_split_records (operation_key) WHERE operation_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_split_records_source_history_v5
ON customer_split_records (source_profile_id, created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_split_records_target_history_v5
ON customer_split_records (target_profile_id, created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_split_records_status_history_v5
ON customer_split_records (status, created_at DESC, id DESC)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_split_moved_entity_once_v5
ON split_moved_entities (split_record_id, entity_type, entity_id, mutation_kind)`,
	`CREATE INDEX IF NOT EXISTS idx_split_moved_order_v5
ON split_moved_entities (split_record_id, move_order, id)`,
	`CREATE INDEX IF NOT EXISTS idx_split_moved_entity_history_v5
ON split_moved_entities (entity_type, entity_id, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_split_moved_from_history_v5
ON split_moved_entities (from_profile_id, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_split_moved_to_history_v5
ON split_moved_entities (to_profile_id, created_at, id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_split_operation_event_key_v5
ON customer_split_operation_events (event_key) WHERE event_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_split_operation_events_operation_v5
ON customer_split_operation_events (operation_key, created_at, id)`,
	`CREATE INDEX IF NOT EXISTS idx_split_operation_events_record_v5
ON customer_split_operation_events (split_record_id, created_at, id)`,
}

func customerResolutionV5Signature() string {
	parts := append([]string{}, customerResolutionV5TableDDL...)
	parts = append(parts, customerResolutionV5Indexes...)
	return strings.Join(parts, "\n")
}

func applyCustomerResolutionV5(tx *gorm.DB) error {
	for _, statement := range customerResolutionV5TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v5 table: %w", err)
		}
	}
	for _, statement := range customerResolutionV5Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v5 index: %w", err)
		}
	}
	return nil
}
