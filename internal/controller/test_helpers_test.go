package controller

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// testdataPath resolves testdata/<parts...> from the module root, so fixtures
// stay at the repo root regardless of which package the test lives in.
func testdataPath(t *testing.T, parts ...string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/controller → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	path := filepath.Join(append([]string{root, "testdata"}, parts...)...)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", path, err)
	}
	return path
}

func seedEnabledFeaturePolicy(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&persistence.CustomerResolutionFeaturePolicy{},
		&persistence.CustomerResolutionFeaturePolicyRevision{}, &persistence.ImportEvidenceSetting{},
		&persistence.ImportRun{}, &persistence.ImportRawRecord{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	current := persistence.CustomerResolutionFeaturePolicy{ID: 1, Revision: 1,
		CustomerResolutionWritesEnabled: true, CandidateScanEnabled: true, MergeExecutionEnabled: true,
		SplitExecutionEnabled: true, ImportEvidenceEnabled: true, CarrierRegistryWritesEnabled: true,
		ActorRef: "test:root-db", Reason: "explicit test policy", UpdatedAt: now}
	if err := db.Where("id = ?", 1).FirstOrCreate(&current).Error; err != nil {
		t.Fatal(err)
	}
	revision := persistence.CustomerResolutionFeaturePolicyRevision{Revision: 1,
		CustomerResolutionWritesEnabled: true, CandidateScanEnabled: true, MergeExecutionEnabled: true,
		SplitExecutionEnabled: true, ImportEvidenceEnabled: true, CarrierRegistryWritesEnabled: true,
		ActorRef: "test:root-db", Reason: "explicit test policy", CreatedAt: now}
	if err := db.Where("revision = ?", 1).FirstOrCreate(&revision).Error; err != nil {
		t.Fatal(err)
	}
	evidenceSetting := persistence.ImportEvidenceSetting{ID: 1, RetentionDays: 90, Revision: 1, UpdatedAt: now}
	if err := db.Where("id = ?", 1).FirstOrCreate(&evidenceSetting).Error; err != nil {
		t.Fatal(err)
	}
}
