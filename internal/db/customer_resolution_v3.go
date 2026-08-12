package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// v3 is append-only. Never edit v1/v2 or reuse this version for later changes.
var customerResolutionV3TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS merge_scan_runs (
id integer PRIMARY KEY AUTOINCREMENT,
merge_policy_id integer NOT NULL,
policy_revision_id integer NOT NULL,
policy_version integer NOT NULL,
status text NOT NULL,
started_at datetime NOT NULL,
completed_at datetime,
profiles_scanned integer NOT NULL DEFAULT 0,
pairs_evaluated integer NOT NULL DEFAULT 0,
candidates_created integer NOT NULL DEFAULT 0,
candidates_updated integer NOT NULL DEFAULT 0,
candidates_blocked integer NOT NULL DEFAULT 0,
error_message text NOT NULL DEFAULT '',
created_at datetime NOT NULL,
updated_at datetime NOT NULL
)`,
}

var customerResolutionV3Columns = []migrationColumn{
	{table: "merge_policies", name: "row_version", definition: "integer NOT NULL DEFAULT 1"},
	{table: "merge_policies", name: "needs_scan", definition: "numeric NOT NULL DEFAULT true"},
	{table: "merge_policies", name: "last_scan_at", definition: "datetime"},
	{table: "merge_policy_revisions", name: "schema_version", definition: "integer NOT NULL DEFAULT 1"},
	{table: "merge_candidates", name: "canonical_pair_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_candidates", name: "evidence_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_candidates", name: "policy_version", definition: "integer NOT NULL DEFAULT 0"},
	{table: "merge_candidates", name: "explanation_code", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_candidates", name: "confidence", definition: "real NOT NULL DEFAULT 0"},
	{table: "merge_candidates", name: "blockers", definition: "text NOT NULL DEFAULT '[]'"},
	{table: "merge_candidates", name: "last_evaluated_at", definition: "datetime"},
	{table: "merge_candidates", name: "expires_at", definition: "datetime"},
	{table: "merge_candidates", name: "scan_run_id", definition: "integer"},
	{table: "merge_evidence", name: "evidence_key", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_evidence", name: "polarity", definition: "text NOT NULL DEFAULT 'positive'"},
	{table: "merge_evidence", name: "explanation_code", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_evidence", name: "value_hash", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_evidence", name: "masked_value", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_evidence", name: "source_entity_type", definition: "text NOT NULL DEFAULT ''"},
	{table: "merge_evidence", name: "source_entity_id", definition: "integer NOT NULL DEFAULT 0"},
	{table: "merge_evidence", name: "observed_at", definition: "datetime"},
}

var customerResolutionV3Indexes = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_policies_active_key_v3
ON merge_policies (policy_key)
WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_policy_revisions_version_v3
ON merge_policy_revisions (merge_policy_id, revision)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_candidates_evaluation_v3
ON merge_candidates (canonical_pair_key, evidence_hash, policy_version)
WHERE canonical_pair_key <> '' AND evidence_hash <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_policy_scan_v3
ON merge_candidates (policy_version, scan_run_id, status)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_merge_evidence_candidate_key_v3
ON merge_evidence (merge_candidate_id, evidence_key)
WHERE evidence_key <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_merge_scan_runs_policy_v3
ON merge_scan_runs (merge_policy_id, policy_version, started_at)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_scan_runs_status_v3
ON merge_scan_runs (status)`,
}

func customerResolutionV3Signature() string {
	parts := append([]string{}, customerResolutionV3TableDDL...)
	for _, column := range customerResolutionV3Columns {
		parts = append(parts, column.table+"."+column.name+" "+column.definition)
	}
	parts = append(parts, customerResolutionV3Indexes...)
	return strings.Join(parts, "\n")
}

func applyCustomerResolutionV3(tx *gorm.DB) error {
	for _, statement := range customerResolutionV3TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v3 table: %w", err)
		}
	}
	for _, column := range customerResolutionV3Columns {
		if tx.Migrator().HasColumn(column.table, column.name) {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", column.table, column.name, column.definition)
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("add customer resolution v3 column %s.%s: %w", column.table, column.name, err)
		}
	}
	for _, statement := range customerResolutionV3Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution v3 index: %w", err)
		}
	}
	return nil
}
