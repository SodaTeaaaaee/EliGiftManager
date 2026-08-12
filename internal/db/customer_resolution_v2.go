package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// v2 is append-only. Do not edit v1 or reuse this version for later changes.
var customerResolutionV2TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS legacy_customer_maps (
id integer PRIMARY KEY AUTOINCREMENT,
legacy_member_id integer NOT NULL,
legacy_platform text NOT NULL,
legacy_platform_uid text NOT NULL,
customer_profile_id integer NOT NULL,
created_at datetime NOT NULL,
updated_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS legacy_customer_migration_cursors (
stream text PRIMARY KEY NOT NULL,
last_legacy_id integer NOT NULL DEFAULT 0,
status text NOT NULL DEFAULT 'pending',
updated_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS legacy_customer_migration_quarantines (
id integer PRIMARY KEY AUTOINCREMENT,
stream text NOT NULL,
legacy_row_id integer NOT NULL,
event_key text NOT NULL DEFAULT '',
reason text NOT NULL,
raw_payload text NOT NULL DEFAULT '',
created_at datetime NOT NULL,
updated_at datetime NOT NULL
)`,
}

var customerResolutionV2Columns = []migrationColumn{
	{table: "customer_name_observations", name: "source_event_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_name_observations", name: "episode_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_name_observations", name: "observation_count", definition: "integer NOT NULL DEFAULT 1"},
	{table: "customer_name_events", name: "event_key", definition: "text NOT NULL DEFAULT ''"},
}

var customerResolutionV2Indexes = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_name_observations_source_event_key
ON customer_name_observations (source_event_key)
WHERE source_event_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_profile_observed_id
ON customer_name_observations (customer_profile_id, observed_at, id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_name_events_event_key
ON customer_name_events (event_key)
WHERE event_key <> ''`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_profile_origins_source_external
ON customer_profile_origins (origin_kind, source_integration_profile_id, external_ref)
WHERE origin_kind <> '' AND source_integration_profile_id IS NOT NULL AND external_ref <> ''`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_legacy_customer_maps_member
ON legacy_customer_maps (legacy_member_id)`,
	`CREATE INDEX IF NOT EXISTS idx_legacy_customer_maps_identity
ON legacy_customer_maps (legacy_platform, legacy_platform_uid)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_legacy_customer_quarantine_stream_row
ON legacy_customer_migration_quarantines (stream, legacy_row_id)`,
}

func customerResolutionV2Signature() string {
	parts := append([]string{}, customerResolutionV2TableDDL...)
	for _, column := range customerResolutionV2Columns {
		parts = append(parts, column.table+"."+column.name+" "+column.definition)
	}
	parts = append(parts, customerResolutionV2Indexes...)
	return strings.Join(parts, "\n")
}

func applyCustomerResolutionV2(tx *gorm.DB) error {
	for _, statement := range customerResolutionV2TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v2 table: %w", err)
		}
	}
	for _, column := range customerResolutionV2Columns {
		if tx.Migrator().HasColumn(column.table, column.name) {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", column.table, column.name, column.definition)
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("add customer resolution v2 column %s.%s: %w", column.table, column.name, err)
		}
	}
	for _, statement := range customerResolutionV2Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v2 index: %w", err)
		}
	}
	return nil
}
