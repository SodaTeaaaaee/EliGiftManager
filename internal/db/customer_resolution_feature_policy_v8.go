package db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// v8 is append-only. V1-V7 are immutable. It introduces the fail-closed,
// versioned safety policy used to stop native customer-resolution writes
// without ever selecting a legacy implementation.
var customerResolutionFeaturePolicyV8Statements = []string{
	`CREATE TABLE IF NOT EXISTS customer_resolution_feature_policy (
id integer PRIMARY KEY CHECK (id = 1),
revision integer NOT NULL,
customer_resolution_writes_enabled integer NOT NULL,
candidate_scan_enabled integer NOT NULL,
merge_execution_enabled integer NOT NULL,
split_execution_enabled integer NOT NULL,
import_evidence_enabled integer NOT NULL,
carrier_registry_writes_enabled integer NOT NULL,
actor_ref text NOT NULL DEFAULT '',
reason text NOT NULL DEFAULT '',
updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
	`CREATE TABLE IF NOT EXISTS customer_resolution_feature_policy_revisions (
id integer PRIMARY KEY AUTOINCREMENT,
revision integer NOT NULL,
customer_resolution_writes_enabled integer NOT NULL,
candidate_scan_enabled integer NOT NULL,
merge_execution_enabled integer NOT NULL,
split_execution_enabled integer NOT NULL,
import_evidence_enabled integer NOT NULL,
carrier_registry_writes_enabled integer NOT NULL,
actor_ref text NOT NULL DEFAULT '',
reason text NOT NULL DEFAULT '',
created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_resolution_feature_policy_revision_v8
ON customer_resolution_feature_policy_revisions (revision)`,
	`INSERT OR IGNORE INTO customer_resolution_feature_policy
(id, revision, customer_resolution_writes_enabled, candidate_scan_enabled, merge_execution_enabled,
 split_execution_enabled, import_evidence_enabled, carrier_registry_writes_enabled, actor_ref, reason)
VALUES (1, 1, 1, 1, 1, 1, 1, 1, 'system:v8_default', 'native customer resolution enabled by default')`,
	`INSERT OR IGNORE INTO customer_resolution_feature_policy_revisions
(revision, customer_resolution_writes_enabled, candidate_scan_enabled, merge_execution_enabled,
 split_execution_enabled, import_evidence_enabled, carrier_registry_writes_enabled, actor_ref, reason)
VALUES (1, 1, 1, 1, 1, 1, 1, 'system:v8_default', 'native customer resolution enabled by default')`,
}

func customerResolutionFeaturePolicyV8Signature() string {
	return strings.Join(customerResolutionFeaturePolicyV8Statements, "\n")
}

func applyCustomerResolutionFeaturePolicyV8(tx *gorm.DB) error {
	for _, statement := range customerResolutionFeaturePolicyV8Statements {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply customer resolution feature policy v8: %w", err)
		}
	}
	return nil
}
