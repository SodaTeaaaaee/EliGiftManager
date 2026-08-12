package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// The v1 declaration is intentionally independent from live persistence
// structs. Once shipped, a migration's behavior and checksum must remain
// immutable; later schema changes belong in a new migration version.
var customerResolutionV1TableDDL = []string{
	`CREATE TABLE IF NOT EXISTS customer_profiles (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
display_name text NOT NULL,
profile_type text NOT NULL DEFAULT 'member',
status text NOT NULL DEFAULT 'active',
merged_into_profile_id integer,
row_version integer NOT NULL DEFAULT 1,
display_name_mode text NOT NULL DEFAULT 'auto',
display_name_observation_id integer,
extra_data text
)`,
	`CREATE TABLE IF NOT EXISTS customer_identities (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
customer_profile_id integer NOT NULL,
identity_platform text NOT NULL,
identity_value text NOT NULL,
identity_type text NOT NULL,
namespace text NOT NULL DEFAULT '',
normalized_value text NOT NULL DEFAULT '',
normalization_version text NOT NULL DEFAULT '',
authority text NOT NULL DEFAULT '',
verification_status text NOT NULL DEFAULT 'unverified',
source_integration_profile_id integer,
resolution_status text NOT NULL DEFAULT 'unresolved',
first_seen_at datetime,
last_seen_at datetime,
is_primary numeric NOT NULL DEFAULT false,
extra_data text
)`,
	`CREATE TABLE IF NOT EXISTS customer_addresses (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
customer_profile_id integer NOT NULL,
label text,
recipient_name text,
phone text,
normalized_phone text NOT NULL DEFAULT '',
address_fingerprint text NOT NULL DEFAULT '',
normalization_version text NOT NULL DEFAULT '',
quality_status text NOT NULL DEFAULT 'unknown',
country text,
province text,
city text,
district text,
address_line1 text,
address_line2 text,
postal_code text,
is_default numeric NOT NULL DEFAULT false,
is_test numeric NOT NULL DEFAULT false,
validation_status text NOT NULL DEFAULT 'unvalidated',
validation_detail text,
extra_data text
)`,
	`CREATE TABLE IF NOT EXISTS customer_merge_records (
id integer PRIMARY KEY AUTOINCREMENT,
source_profile_id integer NOT NULL,
target_profile_id integer NOT NULL,
merge_candidate_id integer,
merge_policy_revision_id integer,
merge_mode text NOT NULL DEFAULT 'manual',
decision_source text NOT NULL DEFAULT '',
decision_reason text NOT NULL DEFAULT '',
actor_ref text NOT NULL DEFAULT '',
correlation_id text NOT NULL DEFAULT '',
source_row_version integer NOT NULL DEFAULT 0,
target_row_version integer NOT NULL DEFAULT 0,
evidence_snapshot text NOT NULL DEFAULT '',
payload text NOT NULL,
created_at datetime NOT NULL,
undone_at datetime
)`,
	`CREATE TABLE IF NOT EXISTS customer_name_observations (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
customer_profile_id integer NOT NULL,
name text NOT NULL,
normalized_name text NOT NULL DEFAULT '',
name_kind text NOT NULL DEFAULT '',
authority text NOT NULL DEFAULT '',
trust_score real NOT NULL DEFAULT 0,
source_integration_profile_id integer,
source_document_id integer,
source_identity_id integer,
observed_at datetime,
first_seen_at datetime,
last_seen_at datetime,
is_pinned numeric NOT NULL DEFAULT false,
is_active numeric NOT NULL DEFAULT true,
extra_data text NOT NULL DEFAULT ''
)`,
	`CREATE TABLE IF NOT EXISTS customer_name_events (
id integer PRIMARY KEY AUTOINCREMENT,
customer_profile_id integer NOT NULL,
observation_id integer,
event_kind text NOT NULL,
previous_name text NOT NULL DEFAULT '',
new_name text NOT NULL DEFAULT '',
reason_code text NOT NULL DEFAULT '',
actor_ref text NOT NULL DEFAULT '',
payload text NOT NULL DEFAULT '',
created_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS customer_profile_origins (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
customer_profile_id integer NOT NULL,
origin_kind text NOT NULL DEFAULT '',
source_integration_profile_id integer,
source_document_id integer,
external_ref text NOT NULL DEFAULT '',
is_provisional numeric NOT NULL DEFAULT false,
first_seen_at datetime,
last_seen_at datetime,
extra_data text NOT NULL DEFAULT ''
)`,
	`CREATE TABLE IF NOT EXISTS merge_candidates (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
source_profile_id integer NOT NULL,
target_profile_id integer NOT NULL,
status text NOT NULL DEFAULT 'pending',
score real NOT NULL DEFAULT 0,
merge_policy_revision_id integer,
reason text NOT NULL DEFAULT ''
)`,
	`CREATE TABLE IF NOT EXISTS merge_evidence (
id integer PRIMARY KEY AUTOINCREMENT,
merge_candidate_id integer NOT NULL,
evidence_kind text NOT NULL,
source_ref text NOT NULL DEFAULT '',
weight real NOT NULL DEFAULT 0,
confidence real NOT NULL DEFAULT 0,
payload text NOT NULL DEFAULT '',
created_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS merge_policies (
id integer PRIMARY KEY AUTOINCREMENT,
created_at datetime,
updated_at datetime,
deleted_at datetime,
policy_key text NOT NULL,
name text NOT NULL,
is_active numeric NOT NULL DEFAULT true,
default_action text NOT NULL DEFAULT 'suggest_only',
current_revision_id integer,
extra_data text NOT NULL DEFAULT ''
)`,
	`CREATE TABLE IF NOT EXISTS merge_policy_revisions (
id integer PRIMARY KEY AUTOINCREMENT,
merge_policy_id integer NOT NULL,
revision integer NOT NULL DEFAULT 1,
action text NOT NULL DEFAULT 'suggest_only',
rules text NOT NULL DEFAULT '',
checksum text NOT NULL DEFAULT '',
created_by text NOT NULL DEFAULT '',
created_at datetime NOT NULL
)`,
	`CREATE TABLE IF NOT EXISTS merge_moved_entities (
id integer PRIMARY KEY AUTOINCREMENT,
merge_record_id integer NOT NULL,
entity_type text NOT NULL,
entity_id integer NOT NULL,
from_profile_id integer NOT NULL,
to_profile_id integer NOT NULL,
move_order integer NOT NULL DEFAULT 0,
before_snapshot text NOT NULL DEFAULT '',
reverted_at datetime,
created_at datetime NOT NULL
)`,
}

type migrationColumn struct {
	table      string
	name       string
	definition string
}

var customerResolutionV1Columns = []migrationColumn{
	{table: "customer_profiles", name: "status", definition: "text NOT NULL DEFAULT 'active'"},
	{table: "customer_profiles", name: "merged_into_profile_id", definition: "integer"},
	{table: "customer_profiles", name: "row_version", definition: "integer NOT NULL DEFAULT 1"},
	{table: "customer_profiles", name: "display_name_mode", definition: "text NOT NULL DEFAULT 'auto'"},
	{table: "customer_profiles", name: "display_name_observation_id", definition: "integer"},
	{table: "customer_identities", name: "namespace", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_identities", name: "normalized_value", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_identities", name: "normalization_version", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_identities", name: "authority", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_identities", name: "verification_status", definition: "text NOT NULL DEFAULT 'unverified'"},
	{table: "customer_identities", name: "source_integration_profile_id", definition: "integer"},
	{table: "customer_identities", name: "resolution_status", definition: "text NOT NULL DEFAULT 'unresolved'"},
	{table: "customer_identities", name: "first_seen_at", definition: "datetime"},
	{table: "customer_identities", name: "last_seen_at", definition: "datetime"},
	{table: "customer_addresses", name: "normalized_phone", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_addresses", name: "address_fingerprint", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_addresses", name: "normalization_version", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_addresses", name: "quality_status", definition: "text NOT NULL DEFAULT 'unknown'"},
	{table: "customer_merge_records", name: "merge_candidate_id", definition: "integer"},
	{table: "customer_merge_records", name: "merge_policy_revision_id", definition: "integer"},
	{table: "customer_merge_records", name: "merge_mode", definition: "text NOT NULL DEFAULT 'manual'"},
	{table: "customer_merge_records", name: "decision_source", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "decision_reason", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "actor_ref", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "correlation_id", definition: "text NOT NULL DEFAULT ''"},
	{table: "customer_merge_records", name: "source_row_version", definition: "integer NOT NULL DEFAULT 0"},
	{table: "customer_merge_records", name: "target_row_version", definition: "integer NOT NULL DEFAULT 0"},
	{table: "customer_merge_records", name: "evidence_snapshot", definition: "text NOT NULL DEFAULT ''"},
}

var customerResolutionV1Indexes = []string{
	`CREATE INDEX IF NOT EXISTS idx_customer_profiles_deleted_at ON customer_profiles (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profiles_status ON customer_profiles (status)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profiles_merged_into_profile_id ON customer_profiles (merged_into_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profiles_display_name_mode ON customer_profiles (display_name_mode)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profiles_display_name_observation_id ON customer_profiles (display_name_observation_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_deleted_at ON customer_identities (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_customer_profile_id ON customer_identities (customer_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_identity_platform_value ON customer_identities (identity_platform, identity_value)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_namespace ON customer_identities (namespace)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_normalized_value ON customer_identities (normalized_value)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_authority ON customer_identities (authority)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_verification_status ON customer_identities (verification_status)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_source_integration_profile_id ON customer_identities (source_integration_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_identities_resolution_status ON customer_identities (resolution_status)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_addresses_deleted_at ON customer_addresses (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_addresses_customer_profile_id ON customer_addresses (customer_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_addresses_normalized_phone ON customer_addresses (normalized_phone)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_addresses_address_fingerprint ON customer_addresses (address_fingerprint)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_addresses_quality_status ON customer_addresses (quality_status)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_source_profile_id ON customer_merge_records (source_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_target_profile_id ON customer_merge_records (target_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_merge_candidate_id ON customer_merge_records (merge_candidate_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_merge_policy_revision_id ON customer_merge_records (merge_policy_revision_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_correlation_id ON customer_merge_records (correlation_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_merge_records_undone_at ON customer_merge_records (undone_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_deleted_at ON customer_name_observations (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_customer_profile_id ON customer_name_observations (customer_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_normalized_name ON customer_name_observations (normalized_name)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_source_document_id ON customer_name_observations (source_document_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_source_identity_id ON customer_name_observations (source_identity_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_observations_is_pinned ON customer_name_observations (is_pinned)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_events_customer_profile_id ON customer_name_events (customer_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_name_events_observation_id ON customer_name_events (observation_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profile_origins_deleted_at ON customer_profile_origins (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profile_origins_customer_profile_id ON customer_profile_origins (customer_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profile_origins_source_integration_profile_id ON customer_profile_origins (source_integration_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profile_origins_source_document_id ON customer_profile_origins (source_document_id)`,
	`CREATE INDEX IF NOT EXISTS idx_customer_profile_origins_external_ref ON customer_profile_origins (external_ref)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_deleted_at ON merge_candidates (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_source_profile_id ON merge_candidates (source_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_target_profile_id ON merge_candidates (target_profile_id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_candidates_status ON merge_candidates (status)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_evidence_merge_candidate_id ON merge_evidence (merge_candidate_id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_policies_deleted_at ON merge_policies (deleted_at)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_policies_policy_key ON merge_policies (policy_key)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_policies_is_active ON merge_policies (is_active)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_policy_revisions_merge_policy_id ON merge_policy_revisions (merge_policy_id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_policy_revisions_checksum ON merge_policy_revisions (checksum)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_entities_merge_record_id ON merge_moved_entities (merge_record_id)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_entities_entity_type ON merge_moved_entities (entity_type)`,
	`CREATE INDEX IF NOT EXISTS idx_merge_moved_entities_entity_id ON merge_moved_entities (entity_id)`,
}

func customerResolutionV1Signature() string {
	parts := append([]string{}, customerResolutionV1TableDDL...)
	for _, column := range customerResolutionV1Columns {
		parts = append(parts, column.table+"."+column.name+" "+column.definition)
	}
	parts = append(parts, customerResolutionV1Indexes...)
	return strings.Join(parts, "\n")
}

func applyCustomerResolutionV1(tx *gorm.DB) error {
	for _, statement := range customerResolutionV1TableDDL {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution table: %w", err)
		}
	}
	for _, column := range customerResolutionV1Columns {
		if tx.Migrator().HasColumn(column.table, column.name) {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", column.table, column.name, column.definition)
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("add customer resolution column %s.%s: %w", column.table, column.name, err)
		}
	}
	for _, statement := range customerResolutionV1Indexes {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create customer resolution index: %w", err)
		}
	}
	return nil
}
