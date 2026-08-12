package app

import (
	"context"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func seedEnabledFeaturePolicyForFocusedDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&persistence.CustomerResolutionFeaturePolicy{},
		&persistence.CustomerResolutionFeaturePolicyRevision{}); err != nil {
		t.Fatalf("migrate focused feature policy: %v", err)
	}
	now := time.Now().UTC()
	current := persistence.CustomerResolutionFeaturePolicy{ID: 1, Revision: 1,
		CustomerResolutionWritesEnabled: true, CandidateScanEnabled: true, MergeExecutionEnabled: true,
		SplitExecutionEnabled: true, ImportEvidenceEnabled: true, CarrierRegistryWritesEnabled: true,
		ActorRef: "test:focused-db", Reason: "explicit test policy", UpdatedAt: now}
	if err := db.Where("id = ?", 1).FirstOrCreate(&current).Error; err != nil {
		t.Fatalf("seed focused feature policy: %v", err)
	}
	revision := persistence.CustomerResolutionFeaturePolicyRevision{Revision: 1,
		CustomerResolutionWritesEnabled: true, CandidateScanEnabled: true, MergeExecutionEnabled: true,
		SplitExecutionEnabled: true, ImportEvidenceEnabled: true, CarrierRegistryWritesEnabled: true,
		ActorRef: "test:focused-db", Reason: "explicit test policy", CreatedAt: now}
	if err := db.Where("revision = ?", 1).FirstOrCreate(&revision).Error; err != nil {
		t.Fatalf("seed focused feature policy revision: %v", err)
	}
}

func (*mockProfileRepoForIdentity) FeatureEnabled(context.Context, string) (bool, error) {
	return true, nil
}

func (*mockProfileRepoForIdentity) RequireFeature(context.Context, string) error { return nil }

func (*mockCarrierMappingRepoFull) FeatureEnabled(context.Context, string) (bool, error) {
	return true, nil
}

func (*mockCarrierMappingRepoFull) RequireFeature(context.Context, string) error { return nil }

var (
	_ domain.CustomerResolutionFeatureGate = (*mockProfileRepoForIdentity)(nil)
	_ domain.CustomerResolutionFeatureGate = (*mockCarrierMappingRepoFull)(nil)
)
