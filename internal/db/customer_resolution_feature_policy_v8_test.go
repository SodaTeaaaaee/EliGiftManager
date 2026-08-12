package db

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCustomerResolutionFeaturePolicyV8AppendOnlyDefaultAndIdempotent(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	for _, apply := range []func(*gorm.DB) error{applyCustomerResolutionV1, applyCustomerResolutionV2,
		applyCustomerResolutionV3, applyCustomerResolutionV4, applyCustomerResolutionV5,
		applyImportEvidenceCarrierV6, applySchemaCompatibilityV7,
		applyCustomerResolutionFeaturePolicyV8, applyCustomerResolutionFeaturePolicyV8} {
		if err := apply(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"customer_resolution_feature_policy", "customer_resolution_feature_policy_revisions"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing v8 table %s", table)
		}
	}
	var current persistence.CustomerResolutionFeaturePolicy
	if err := db.First(&current, 1).Error; err != nil {
		t.Fatal(err)
	}
	if current.Revision != 1 || !current.CustomerResolutionWritesEnabled || !current.CandidateScanEnabled ||
		!current.MergeExecutionEnabled || !current.SplitExecutionEnabled || !current.ImportEvidenceEnabled ||
		!current.CarrierRegistryWritesEnabled {
		t.Fatalf("unsafe v8 defaults: %+v", current)
	}
	var revisions int64
	if err := db.Model(&persistence.CustomerResolutionFeaturePolicyRevision{}).Count(&revisions).Error; err != nil || revisions != 1 {
		t.Fatalf("default revision history count=%d err=%v", revisions, err)
	}
	migrations := customerResolutionMigrations()
	if len(migrations) < 8 || migrations[7].Version != 8 || migrations[7].Name != "customer_resolution_feature_policy_v8" {
		t.Fatalf("unexpected v8 registration: %+v", migrations)
	}
}

func TestCustomerResolutionFeaturePolicyV8ChecksumFrozen(t *testing.T) {
	t.Parallel()
	migrations := customerResolutionMigrations()
	if len(migrations) < 8 {
		t.Fatalf("missing v8 migration: %+v", migrations)
	}
	const want = "7aae2dc14940c8247a5e1b4ad85d6dbf4572242ff5b06f19bd1f38e9c3122754"
	if got := migrations[7].checksum(); got != want {
		t.Fatalf("v8 migration checksum changed: got %s want %s", got, want)
	}
}
