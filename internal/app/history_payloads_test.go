package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPayloadTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbConn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := dbConn.AutoMigrate(&persistence.AllocationPolicyRule{}, &persistence.FulfillmentAdjustment{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return dbConn
}

func TestBuildRuleRestorePatch_RoundTripsThroughPatchExecutor(t *testing.T) {
	dbConn := newPayloadTestDB(t)
	patchExec := NewPatchExecutor(dbConn)
	ruleRepo := infra.NewRuleRepository(dbConn)

	rule := &domain.AllocationPolicyRule{
		ID:                   101,
		WaveID:               11,
		ProductID:            21,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ProductTargetRef:     "product:21",
		ContributionQuantity: 3,
		RuleKind:             "standard",
		Priority:             4,
		Active:               true,
	}

	payload, err := BuildRuleRestorePatch("restore_rule", rule)
	if err != nil {
		t.Fatalf("BuildRuleRestorePatch: %v", err)
	}
	if err := patchExec.ApplyPatch(payload); err != nil {
		t.Fatalf("ApplyPatch restore_rule: %v", err)
	}

	fetched, err := ruleRepo.FindByID(rule.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected restored rule to exist")
	}
	if fetched.WaveID != rule.WaveID || fetched.ProductID != rule.ProductID || fetched.ProductTargetRef != rule.ProductTargetRef || fetched.ContributionQuantity != rule.ContributionQuantity {
		t.Fatalf("restored rule mismatch: got %+v want %+v", fetched, rule)
	}
	if fetched.SelectorPayload.Type != rule.SelectorPayload.Type {
		t.Fatalf("selector payload mismatch: got %+v want %+v", fetched.SelectorPayload, rule.SelectorPayload)
	}
}

func TestBuildRuleUpdatePatch_RoundTripsThroughPatchExecutor(t *testing.T) {
	dbConn := newPayloadTestDB(t)
	patchExec := NewPatchExecutor(dbConn)
	ruleRepo := infra.NewRuleRepository(dbConn)

	rule := &domain.AllocationPolicyRule{
		WaveID:               11,
		ProductID:            21,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ProductTargetRef:     "product:21",
		ContributionQuantity: 3,
		RuleKind:             "standard",
		Priority:             4,
		Active:               true,
	}
	if err := ruleRepo.Create(rule); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rule.Priority = 9
	rule.Active = false
	rule.ProductTargetRef = "product:21-updated"
	payload, err := BuildRuleUpdatePatch(rule)
	if err != nil {
		t.Fatalf("BuildRuleUpdatePatch: %v", err)
	}
	if err := patchExec.ApplyPatch(payload); err != nil {
		t.Fatalf("ApplyPatch update_rule: %v", err)
	}

	fetched, err := ruleRepo.FindByID(rule.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected updated rule to exist")
	}
	if fetched.Priority != 9 || fetched.Active != false || fetched.ProductTargetRef != "product:21-updated" {
		t.Fatalf("updated rule mismatch: got %+v", fetched)
	}
}

