package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type failHeadScopeRepo struct {
	domain.HistoryScopeRepository
	failUpdate bool
}

func (f *failHeadScopeRepo) UpdateHead(ctx context.Context, scopeID uint, headNodeID uint) error {
	if f.failUpdate {
		return fmt.Errorf("update head failed")
	}
	return f.HistoryScopeRepository.UpdateHead(ctx, scopeID, headNodeID)
}

func setupWaveHistory(t *testing.T, inversePayload, patchPayload string) (*mockHistoryScopeRepo, *mockHistoryNodeRepo, *domain.HistoryScope, *domain.HistoryNode, *domain.HistoryNode) {
	t.Helper()
	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()

	parent := &domain.HistoryNode{CommandSummary: "baseline"}
	if err := nodeRepo.Create(context.Background(), parent); err != nil {
		t.Fatalf("create parent node: %v", err)
	}
	child := &domain.HistoryNode{
		ParentNodeID:        parent.ID,
		CommandSummary:      "create rule",
		InversePatchPayload: inversePayload,
		PatchPayload:        patchPayload,
	}
	if err := nodeRepo.Create(context.Background(), child); err != nil {
		t.Fatalf("create child node: %v", err)
	}
	if err := nodeRepo.UpdatePreferredRedoChild(context.Background(), parent.ID, child.ID); err != nil {
		t.Fatalf("set redo child: %v", err)
	}

	scope := &domain.HistoryScope{
		ScopeType:         "wave",
		ScopeKey:          "1",
		CurrentHeadNodeID: child.ID,
	}
	if err := scopeRepo.Create(context.Background(), scope); err != nil {
		t.Fatalf("create scope: %v", err)
	}
	return scopeRepo, nodeRepo, scope, parent, child
}

func openUndoRedoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.AllocationPolicyRule{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func mustCreateRule(t *testing.T, db *gorm.DB) *persistence.AllocationPolicyRule {
	t.Helper()
	rule := &persistence.AllocationPolicyRule{
		WaveID:               1,
		ProductID:            10,
		ContributionQuantity: 5,
		RuleKind:             "direct",
		Priority:             1,
		Active:               true,
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	return rule
}

func countRules(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&persistence.AllocationPolicyRule{}).Count(&count).Error; err != nil {
		t.Fatalf("count rules: %v", err)
	}
	return count
}

func TestUndoUpdateHeadFailureWithNilPatchExecutor(t *testing.T) {
	t.Parallel()

	inner, nodeRepo, _, _, _ := setupWaveHistory(t, `{"op":"delete_rule","rule_id":1}`, "")
	scopeRepo := &failHeadScopeRepo{HistoryScopeRepository: inner, failUpdate: true}
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo)

	_, err := uc.Undo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected UpdateHead error, got nil")
	}
	if !strings.Contains(err.Error(), "update head failed") {
		t.Fatalf("expected UpdateHead error, got %v", err)
	}
}

func TestRedoUpdateHeadFailureWithNilPatchExecutor(t *testing.T) {
	t.Parallel()

	inner, nodeRepo, scope, parent, _ := setupWaveHistory(t, "", `{"op":"delete_rule","rule_id":1}`)
	if err := inner.UpdateHead(context.Background(), scope.ID, parent.ID); err != nil {
		t.Fatalf("move head to parent: %v", err)
	}
	scopeRepo := &failHeadScopeRepo{HistoryScopeRepository: inner, failUpdate: true}
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo)

	_, err := uc.Redo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected UpdateHead error, got nil")
	}
	if !strings.Contains(err.Error(), "update head failed") {
		t.Fatalf("expected UpdateHead error, got %v", err)
	}
}

func TestUndoCannotUndoableErrorWrapping(t *testing.T) {
	t.Parallel()

	scopeRepo, nodeRepo, _, _, _ := setupWaveHistory(t, `{"op":"generate_participants"}`, "")
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, &PatchExecutor{})

	_, err := uc.Undo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrOperationNotUndoable) {
		t.Fatalf("expected ErrOperationNotUndoable, got %v", err)
	}
	if !strings.Contains(err.Error(), `cannot undo "create rule"`) {
		t.Fatalf("expected cannot-undo wrapping, got %v", err)
	}
}

func TestUndoApplyFailureErrorWrapping(t *testing.T) {
	t.Parallel()

	scopeRepo, nodeRepo, _, _, _ := setupWaveHistory(t, `{"op":"not_a_real_op"}`, "")
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, &PatchExecutor{})

	_, err := uc.Undo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `undo failed for "create rule"`) {
		t.Fatalf("expected undo-failed wrapping, got %v", err)
	}
}

func createRulePatchPayload(t *testing.T, rule *persistence.AllocationPolicyRule) string {
	t.Helper()
	data, err := json.Marshal(domain.AllocationPolicyRule{
		ID:                   rule.ID,
		WaveID:               rule.WaveID,
		ProductID:            rule.ProductID,
		ContributionQuantity: rule.ContributionQuantity,
		RuleKind:             rule.RuleKind,
		Priority:             rule.Priority,
		Active:               rule.Active,
	})
	if err != nil {
		t.Fatalf("marshal create_rule data: %v", err)
	}
	return fmt.Sprintf(`{"op":"create_rule","data":%s}`, data)
}

func TestUndoWrapsPatchAndHeadWhenDBPresent(t *testing.T) {
	t.Parallel()

	db := openUndoRedoTestDB(t)
	rule := mustCreateRule(t, db)
	inverse := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID)
	forward := createRulePatchPayload(t, rule)

	inner, nodeRepo, _, _, _ := setupWaveHistory(t, inverse, forward)
	scopeRepo := &failHeadScopeRepo{HistoryScopeRepository: inner, failUpdate: true}
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, NewPatchExecutor(db))

	_, err := uc.Undo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected UpdateHead error, got nil")
	}
	if !strings.Contains(err.Error(), "update head failed") {
		t.Fatalf("expected UpdateHead error, got %v", err)
	}
	if got := countRules(t, db); got != 1 {
		t.Fatalf("expected delete_rule to be compensated after UpdateHead failure, rule count=%d", got)
	}
}

func TestRedoWrapsPatchAndHeadWhenDBPresent(t *testing.T) {
	t.Parallel()

	db := openUndoRedoTestDB(t)
	rule := mustCreateRule(t, db)
	forward := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID)
	inverse := createRulePatchPayload(t, rule)

	inner, nodeRepo, scope, parent, _ := setupWaveHistory(t, inverse, forward)
	if err := inner.UpdateHead(context.Background(), scope.ID, parent.ID); err != nil {
		t.Fatalf("move head to parent: %v", err)
	}
	scopeRepo := &failHeadScopeRepo{HistoryScopeRepository: inner, failUpdate: true}
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, NewPatchExecutor(db))

	_, err := uc.Redo(context.Background(), 1)
	if err == nil {
		t.Fatal("expected UpdateHead error, got nil")
	}
	if !strings.Contains(err.Error(), "update head failed") {
		t.Fatalf("expected UpdateHead error, got %v", err)
	}
	if got := countRules(t, db); got != 1 {
		t.Fatalf("expected delete_rule to be compensated after UpdateHead failure, rule count=%d", got)
	}
}

func TestUndoSavepointRollsBackWhenAlreadyInTransaction(t *testing.T) {
	t.Parallel()

	db := openUndoRedoTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	rule := mustCreateRule(t, db)
	inverse := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID)

	inner, nodeRepo, _, _, _ := setupWaveHistory(t, inverse, "")
	scopeRepo := &failHeadScopeRepo{HistoryScopeRepository: inner, failUpdate: true}

	txErr := db.Transaction(func(tx *gorm.DB) error {
		uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, NewPatchExecutor(tx))
		_, undoErr := uc.Undo(context.Background(), 1)
		if undoErr == nil {
			t.Error("expected UpdateHead error inside transaction")
		}
		return undoErr
	})
	if txErr == nil {
		t.Fatal("expected transaction to fail")
	}
	if got := countRules(t, db); got != 1 {
		t.Fatalf("expected savepoint to roll back delete_rule, rule count=%d", got)
	}
}

func TestUndoCommitsPatchWhenUpdateHeadSucceeds(t *testing.T) {
	t.Parallel()

	db := openUndoRedoTestDB(t)
	rule := mustCreateRule(t, db)
	payload := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID)

	scopeRepo, nodeRepo, _, _, _ := setupWaveHistory(t, payload, "")
	uc := NewUndoRedoUseCase(scopeRepo, nodeRepo, NewPatchExecutor(db))

	summary, err := uc.Undo(context.Background(), 1)
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if summary != "create rule" {
		t.Fatalf("summary: got %q", summary)
	}
	if got := countRules(t, db); got != 0 {
		t.Fatalf("expected delete_rule to commit when UpdateHead succeeds, rule count=%d", got)
	}
}
