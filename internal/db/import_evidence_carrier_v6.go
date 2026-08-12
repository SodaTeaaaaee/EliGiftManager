package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// V6 is append-only and owned by RAW evidence plus the profile-scoped external
// carrier registry. Do not edit V1-V5 signatures.
var importEvidenceCarrierV6TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS import_runs (
id integer PRIMARY KEY AUTOINCREMENT,
run_key text NOT NULL UNIQUE,
import_kind text NOT NULL,
integration_profile_id integer,
source_format text NOT NULL DEFAULT '',
source_file_name text NOT NULL DEFAULT '',
import_mode text NOT NULL DEFAULT 'skip_invalid',
status text NOT NULL DEFAULT 'running',
retention_days integer NOT NULL DEFAULT 90,
retention_policy_version integer NOT NULL DEFAULT 1,
expires_at datetime,
record_count integer NOT NULL DEFAULT 0,
success_count integer NOT NULL DEFAULT 0,
failure_count integer NOT NULL DEFAULT 0,
quarantined_count integer NOT NULL DEFAULT 0,
parser_metadata text NOT NULL DEFAULT '',
created_at datetime NOT NULL,
completed_at datetime
)`,
	`CREATE TABLE IF NOT EXISTS import_raw_records (
id integer PRIMARY KEY AUTOINCREMENT,
import_run_id integer NOT NULL,
row_index integer NOT NULL,
raw_logical_row text NOT NULL DEFAULT '',
unmapped_source text NOT NULL DEFAULT '',
parser_metadata text NOT NULL DEFAULT '',
warning_codes text NOT NULL DEFAULT '[]',
asset_members text NOT NULL DEFAULT '[]',
outcome text NOT NULL DEFAULT 'pending',
error_code text NOT NULL DEFAULT '',
error_message text NOT NULL DEFAULT '',
result_type text NOT NULL DEFAULT '',
result_id integer,
retention_days integer NOT NULL DEFAULT 90,
expires_at datetime,
created_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS import_evidence_settings (
id integer PRIMARY KEY,
retention_days integer NOT NULL DEFAULT 90,
revision integer NOT NULL DEFAULT 1,
updated_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS external_carriers (
id integer PRIMARY KEY AUTOINCREMENT,
integration_profile_id integer NOT NULL,
canonical_key text NOT NULL,
external_carrier_code text NOT NULL DEFAULT '',
external_carrier_name text NOT NULL DEFAULT '',
name_key_strategy text NOT NULL DEFAULT 'code_or_normalized_name_v1',
internal_carrier_code text,
status text NOT NULL DEFAULT 'provisional',
conflict_reason text NOT NULL DEFAULT '',
source_import_run_id integer,
source_raw_record_id integer,
created_at datetime NOT NULL,
updated_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS external_carrier_conflicts (
id integer PRIMARY KEY AUTOINCREMENT,
integration_profile_id integer NOT NULL,
canonical_key text NOT NULL,
conflict_kind text NOT NULL,
external_carrier_code text NOT NULL DEFAULT '',
external_carrier_name text NOT NULL DEFAULT '',
internal_carrier_code text NOT NULL DEFAULT '',
source_import_run_id integer,
source_raw_record_id integer,
legacy_carrier_mapping_id integer,
payload text NOT NULL DEFAULT '',
created_at datetime NOT NULL
)`,
}

var importEvidenceCarrierV6Indexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_import_runs_safe_list_v6 ON import_runs (created_at DESC, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_import_runs_expiry_v6 ON import_runs (expires_at)`,
	`CREATE INDEX IF NOT EXISTS idx_import_raw_run_row_v6 ON import_raw_records (import_run_id, row_index, id)`,
	`CREATE INDEX IF NOT EXISTS idx_import_raw_expiry_v6 ON import_raw_records (expires_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_external_carrier_profile_key_v6 ON external_carriers (integration_profile_id, canonical_key)`,
	`CREATE INDEX IF NOT EXISTS idx_external_carrier_profile_code_v6 ON external_carriers (integration_profile_id, external_carrier_code)`,
	`CREATE INDEX IF NOT EXISTS idx_external_carrier_profile_name_v6 ON external_carriers (integration_profile_id, external_carrier_name)`,
	`CREATE INDEX IF NOT EXISTS idx_external_carrier_conflict_review_v6 ON external_carrier_conflicts (integration_profile_id, canonical_key, created_at, id)`,
}

func importEvidenceCarrierV6Signature() string {
	parts := append([]string{}, importEvidenceCarrierV6TableDDL...)
	parts = append(parts, importEvidenceCarrierV6Indexes...)
	parts = append(parts, "legacy_carrier_mapping_conflict_isolation_v1")
	return strings.Join(parts, "\n")
}

func applyImportEvidenceCarrierV6(tx *gorm.DB) error {
	for _, statement := range importEvidenceCarrierV6TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create v6 table: %w", err)
		}
	}
	for _, statement := range importEvidenceCarrierV6Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create v6 index: %w", err)
		}
	}
	now := time.Now().UTC()
	if err := tx.Exec(`INSERT OR IGNORE INTO import_evidence_settings (id, retention_days, revision, updated_at) VALUES (1, 90, 1, ?)`, now).Error; err != nil {
		return fmt.Errorf("seed v6 import evidence setting: %w", err)
	}
	return backfillLegacyCarrierRegistryV6(tx, now)
}

type legacyCarrierMappingV6 struct {
	ID                   uint
	IntegrationProfileID uint
	InternalCarrierCode  string
	ExternalCarrierCode  string
	ExternalCarrierName  string
}

func backfillLegacyCarrierRegistryV6(tx *gorm.DB, now time.Time) error {
	if !tx.Migrator().HasTable("carrier_mappings") {
		return nil
	}
	var rows []legacyCarrierMappingV6
	if err := tx.Raw(`SELECT id, integration_profile_id, internal_carrier_code, external_carrier_code, external_carrier_name
FROM carrier_mappings WHERE deleted_at IS NULL ORDER BY integration_profile_id, id`).Scan(&rows).Error; err != nil {
		return fmt.Errorf("read legacy carrier mappings: %w", err)
	}
	groups := make(map[string][]legacyCarrierMappingV6)
	nameKeys := make(map[string]map[string]struct{})
	for _, row := range rows {
		canonical := externalCarrierCanonicalKeyV6(row.ExternalCarrierCode, row.ExternalCarrierName)
		groupKey := fmt.Sprintf("%d\x00%s", row.IntegrationProfileID, canonical)
		groups[groupKey] = append(groups[groupKey], row)
		nameKey := fmt.Sprintf("%d\x00%s", row.IntegrationProfileID, normalizeCarrierNameV6(row.ExternalCarrierName))
		if nameKeys[nameKey] == nil {
			nameKeys[nameKey] = map[string]struct{}{}
		}
		nameKeys[nameKey][canonical] = struct{}{}
	}
	for _, grouped := range groups {
		first := grouped[0]
		canonical := externalCarrierCanonicalKeyV6(first.ExternalCarrierCode, first.ExternalCarrierName)
		nameKey := fmt.Sprintf("%d\x00%s", first.IntegrationProfileID, normalizeCarrierNameV6(first.ExternalCarrierName))
		conflicted := len(grouped) > 1 || len(nameKeys[nameKey]) > 1
		status := "bound"
		conflictReason := ""
		var internal any = strings.TrimSpace(first.InternalCarrierCode)
		if conflicted {
			status = "review"
			conflictReason = "historical duplicate or external code/name conflict isolated during v6 migration"
			internal = nil
		}
		if err := tx.Exec(`INSERT INTO external_carriers
(integration_profile_id, canonical_key, external_carrier_code, external_carrier_name, name_key_strategy, internal_carrier_code, status, conflict_reason, created_at, updated_at)
VALUES (?, ?, ?, ?, 'code_or_normalized_name_v1', ?, ?, ?, ?, ?)`,
			first.IntegrationProfileID, canonical, strings.TrimSpace(first.ExternalCarrierCode), strings.TrimSpace(first.ExternalCarrierName),
			internal, status, conflictReason, now, now).Error; err != nil {
			return fmt.Errorf("backfill external carrier %q: %w", canonical, err)
		}
		if !conflicted {
			continue
		}
		for _, row := range grouped {
			legacyID := row.ID
			if err := tx.Exec(`INSERT INTO external_carrier_conflicts
(integration_profile_id, canonical_key, conflict_kind, external_carrier_code, external_carrier_name, internal_carrier_code, legacy_carrier_mapping_id, created_at)
VALUES (?, ?, 'historical_conflict', ?, ?, ?, ?, ?)`, row.IntegrationProfileID, canonical,
				row.ExternalCarrierCode, row.ExternalCarrierName, row.InternalCarrierCode, legacyID, now).Error; err != nil {
				return fmt.Errorf("isolate legacy carrier conflict %d: %w", row.ID, err)
			}
		}
	}
	return nil
}

func externalCarrierCanonicalKeyV6(code, name string) string {
	if normalized := strings.ToLower(strings.TrimSpace(code)); normalized != "" {
		return "code:" + normalized
	}
	return "name:" + normalizeCarrierNameV6(name)
}

func normalizeCarrierNameV6(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}
