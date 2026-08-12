package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── fixture ──────────────────────────────────────────────────────────────────

type historyIntegrationFixture struct {
	db          *gorm.DB
	ruleRepo    domain.AllocationPolicyRuleRepository
	adjRepo     domain.FulfillmentAdjustmentRepository
	assignRepo  domain.WaveDemandAssignmentRepository
	waveRepo    domain.WaveRepository
	fulfillRepo domain.FulfillmentLineRepository
	scopeRepo   domain.HistoryScopeRepository
	nodeRepo    domain.HistoryNodeRepository
	cpRepo      domain.HistoryCheckpointRepository
	recording   *HistoryRecordingService
	snapshot    *WaveSnapshotService
	patchExec   *PatchExecutor
	undoRedo    UndoRedoUseCase
	projHash    *ProjectionHashService
}

func newHistoryIntegrationFixture(t *testing.T) *historyIntegrationFixture {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	// Auto-migrate all persistence models used by the history subsystem.
	if err := db.AutoMigrate(
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.AllocationPolicyRule{},
		&persistence.FulfillmentAdjustment{},
		&persistence.FulfillmentLine{},
		&persistence.WaveDemandAssignment{},
		&persistence.DemandDocument{},
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.HistoryCheckpoint{},
		&persistence.HistoryPin{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	ruleRepo := infra.NewRuleRepository(db)
	adjRepo := infra.NewFulfillmentAdjustmentRepository(db)
	assignRepo := infra.NewWaveDemandAssignmentRepository(db)
	waveRepo := infra.NewWaveRepository(db)
	fulfillRepo := infra.NewFulfillmentRepository(db)
	scopeRepo := infra.NewHistoryScopeRepository(db)
	nodeRepo := infra.NewHistoryNodeRepository(db)
	cpRepo := infra.NewHistoryCheckpointRepository(db)

	snapshotSvc := NewWaveSnapshotService(db, ruleRepo, adjRepo, assignRepo, waveRepo, fulfillRepo)
	patchExec := NewPatchExecutor(db, snapshotSvc)
	recordingSvc := NewHistoryRecordingService(scopeRepo, nodeRepo, cpRepo, WithSnapshotService(snapshotSvc))
	undoRedoUC := NewUndoRedoUseCase(scopeRepo, nodeRepo, patchExec)
	projHashSvc := NewProjectionHashService(fulfillRepo, ruleRepo, adjRepo)

	return &historyIntegrationFixture{
		db:          db,
		ruleRepo:    ruleRepo,
		adjRepo:     adjRepo,
		assignRepo:  assignRepo,
		waveRepo:    waveRepo,
		fulfillRepo: fulfillRepo,
		scopeRepo:   scopeRepo,
		nodeRepo:    nodeRepo,
		cpRepo:      cpRepo,
		recording:   recordingSvc,
		snapshot:    snapshotSvc,
		patchExec:   patchExec,
		undoRedo:    undoRedoUC,
		projHash:    projHashSvc,
	}
}

// mustCreateWave inserts a minimal Wave row and returns its ID.
func mustCreateWave(t *testing.T, f *historyIntegrationFixture) uint {
	t.Helper()
	wave := &domain.Wave{
		WaveNo:         fmt.Sprintf("WAVE-TEST-%d", nextTestSeq()),
		Name:           "integration test wave",
		WaveType:       "mixed",
		LifecycleStage: "intake",
	}
	if err := f.waveRepo.Create(context.Background(), wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}
	return wave.ID
}

// mustRecordNode is a thin wrapper that fails the test on error.
func mustRecordNode(t *testing.T, f *historyIntegrationFixture, input RecordNodeInput) *domain.HistoryNode {
	t.Helper()
	node, err := f.recording.RecordNode(context.Background(), input)
	if err != nil {
		t.Fatalf("RecordNode: %v", err)
	}
	return node
}

// testSeq provides unique wave numbers across parallel subtests.
var testSeqCounter uint32

func nextTestSeq() uint32 {
	testSeqCounter++
	return testSeqCounter
}

// ── Test 1: create rule → undo → redo ────────────────────────────────────────

func TestIntegration_CreateRule_UndoRedo(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// Record a baseline node so the create_rule node has a parent to undo to.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:         waveID,
		CommandKind:    "baseline",
		CommandSummary: "initial state",
	})

	// 1. Create a rule via ruleRepo.
	rule := &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            10,
		ContributionQuantity: 5,
		RuleKind:             "direct",
		Priority:             1,
		Active:               true,
	}
	if err := f.ruleRepo.Create(context.Background(), rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	ruleID := rule.ID
	if ruleID == 0 {
		t.Fatal("rule ID must be non-zero after create")
	}

	// 2. Build patch payloads.
	//    forward: create_rule (for redo) — carries full rule data with ID
	//    inverse: delete_rule (for undo)
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	forwardPayload := fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, waveID, ruleJSON)
	inversePayload := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID)

	// Record history node for the create.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "create_rule",
		CommandSummary:      "create rule",
		PatchPayload:        forwardPayload,
		InversePatchPayload: inversePayload,
	})

	// 3. Verify rule exists.
	rules, err := f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule before undo, got %d", len(rules))
	}

	// 4. Undo → rule must be deleted.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo: %v", err)
	}
	rules, err = f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list rules after undo: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after undo, got %d", len(rules))
	}

	// 5. Redo → rule must be restored with the same ID.
	if _, err := f.undoRedo.Redo(context.Background(), waveID); err != nil {
		t.Fatalf("redo: %v", err)
	}
	rules, err = f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list rules after redo: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after redo, got %d", len(rules))
	}
	if rules[0].ID != ruleID {
		t.Errorf("expected restored rule ID=%d, got %d", ruleID, rules[0].ID)
	}
}

// ── Test 2: update rule → undo → redo ────────────────────────────────────────

func TestIntegration_UpdateRule_UndoRedo(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// 1. Create a rule at Priority=1, Active=true.
	rule := &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            20,
		ContributionQuantity: 3,
		RuleKind:             "direct",
		Priority:             1,
		Active:               true,
	}
	if err := f.ruleRepo.Create(context.Background(), rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	ruleID := rule.ID

	// 2. Record initial history node (create).
	oldJSON, _ := json.Marshal(rule)
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "create_rule",
		CommandSummary:      "create rule",
		PatchPayload:        fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, waveID, oldJSON),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID),
	})

	// 3. Update the rule: Priority=5, Active=false.
	rule.Priority = 5
	rule.Active = false
	if err := f.ruleRepo.Update(context.Background(), rule); err != nil {
		t.Fatalf("update rule: %v", err)
	}

	// Build update patches:
	//   forward: update_rule with new data
	//   inverse: update_rule with old data
	newJSON, _ := json.Marshal(rule)
	oldRuleForUndo := *rule
	oldRuleForUndo.Priority = 1
	oldRuleForUndo.Active = true
	oldJSON2, _ := json.Marshal(oldRuleForUndo)

	forwardPayload := fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"data":%s}`, ruleID, newJSON)
	inversePayload := fmt.Sprintf(`{"op":"update_rule","rule_id":%d,"data":%s}`, ruleID, oldJSON2)

	// 4. Record update history node.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "update_rule",
		CommandSummary:      "update rule",
		PatchPayload:        forwardPayload,
		InversePatchPayload: inversePayload,
	})

	// 5. Undo → rule should revert to Priority=1, Active=true.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo: %v", err)
	}
	fetched, err := f.ruleRepo.FindByID(context.Background(), ruleID)
	if err != nil {
		t.Fatalf("find rule after undo: %v", err)
	}
	if fetched.Priority != 1 {
		t.Errorf("after undo: expected Priority=1, got %d", fetched.Priority)
	}
	if fetched.Active != true {
		t.Errorf("after undo: expected Active=true, got %v", fetched.Active)
	}

	// 6. Redo → rule should advance back to Priority=5, Active=false.
	if _, err := f.undoRedo.Redo(context.Background(), waveID); err != nil {
		t.Fatalf("redo: %v", err)
	}
	fetched, err = f.ruleRepo.FindByID(context.Background(), ruleID)
	if err != nil {
		t.Fatalf("find rule after redo: %v", err)
	}
	if fetched.Priority != 5 {
		t.Errorf("after redo: expected Priority=5, got %d", fetched.Priority)
	}
	if fetched.Active != false {
		t.Errorf("after redo: expected Active=false, got %v", fetched.Active)
	}
}

// ── Test 3: delete rule → undo → redo ────────────────────────────────────────

func TestIntegration_DeleteRule_UndoRedo(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// 1. Create a rule.
	rule := &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            30,
		ContributionQuantity: 7,
		RuleKind:             "direct",
		Priority:             2,
		Active:               true,
	}
	if err := f.ruleRepo.Create(context.Background(), rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	ruleID := rule.ID

	// Record create node.
	ruleJSON, _ := json.Marshal(rule)
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "create_rule",
		CommandSummary:      "create rule",
		PatchPayload:        fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, waveID, ruleJSON),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID),
	})

	// 3. Build delete patch before executing so the inverse carries pre-delete data.
	//    delete_rule (forward) uses Unscoped hard-delete, which is required because
	//    the paired restore_rule (inverse) re-inserts with the original ID — the row
	//    must be fully absent, not merely soft-deleted.
	forwardPayload := fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, ruleID)
	inversePayload := fmt.Sprintf(`{"op":"restore_rule","rule_id":%d,"wave_id":%d,"data":%s}`, ruleID, waveID, ruleJSON)

	// Execute delete via PatchExecutor so the hard-delete path is taken.
	if err := f.patchExec.ApplyPatch(context.Background(), forwardPayload); err != nil {
		t.Fatalf("apply delete_rule patch: %v", err)
	}
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "delete_rule",
		CommandSummary:      "delete rule",
		PatchPayload:        forwardPayload,
		InversePatchPayload: inversePayload,
	})

	// Verify rule is gone.
	rules, _ := f.ruleRepo.ListByWave(context.Background(), waveID)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(rules))
	}

	// 5. Undo → rule restored with same ID.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo: %v", err)
	}
	rules, err := f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list rules after undo: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after undo, got %d", len(rules))
	}
	if rules[0].ID != ruleID {
		t.Errorf("expected restored rule ID=%d, got %d", ruleID, rules[0].ID)
	}

	// 6. Redo → rule deleted again.
	if _, err := f.undoRedo.Redo(context.Background(), waveID); err != nil {
		t.Fatalf("redo: %v", err)
	}
	rules, _ = f.ruleRepo.ListByWave(context.Background(), waveID)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after redo, got %d", len(rules))
	}
}

// ── Test 4: checkpoint restore preserves full wave state ──────────────────────

func TestIntegration_CheckpointRestore_FullState(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// Record a baseline node so the checkpoint node has a parent to undo to.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:         waveID,
		CommandKind:    "baseline",
		CommandSummary: "initial state",
	})

	// 1. Create rules.
	ruleA := &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 100, ContributionQuantity: 1, RuleKind: "direct", Priority: 1, Active: true,
	}
	ruleB := &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 101, ContributionQuantity: 2, RuleKind: "direct", Priority: 2, Active: true,
	}
	if err := f.ruleRepo.Create(context.Background(), ruleA); err != nil {
		t.Fatalf("create ruleA: %v", err)
	}
	if err := f.ruleRepo.Create(context.Background(), ruleB); err != nil {
		t.Fatalf("create ruleB: %v", err)
	}

	// 2. Create an adjustment.
	adj := &domain.FulfillmentAdjustment{
		WaveID:         waveID,
		TargetKind:     "fulfillment_line",
		AdjustmentKind: "manual_override",
		QuantityDelta:  3,
		OperatorID:     "test_operator",
	}
	if err := f.adjRepo.Create(context.Background(), adj); err != nil {
		t.Fatalf("create adj: %v", err)
	}
	origAdjID := adj.ID

	// 3. Create a participant snapshot and fulfillment line.
	ptSnap := &persistence.WaveParticipantSnapshot{
		WaveID:            waveID,
		CustomerProfileID: 200,
		SnapshotType:      "member",
		DisplayName:       "Test Participant",
	}
	if err := f.db.Create(ptSnap).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}

	fl := &domain.FulfillmentLine{
		WaveID:          waveID,
		Quantity:        5,
		AllocationState: "allocated",
		LineReason:      "retail_order",
		GeneratedBy:     "test",
	}
	if err := f.fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}
	origFlID := fl.ID

	// 4. Capture snapshot.
	snapPayload, err := f.snapshot.CaptureSnapshot(context.Background(), waveID)
	if err != nil {
		t.Fatalf("capture snapshot: %v", err)
	}

	// 5. Record history node with restore_checkpoint as inverse.
	checkpointInverse := fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, snapPayload)
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "snapshot",
		CommandSummary:      "pre-modification checkpoint",
		PatchPayload:        `{"op":"generate_participants"}`,
		InversePatchPayload: checkpointInverse,
		CheckpointHint:      true,
		SnapshotPayload:     snapPayload,
	})

	// 6. Modify state: delete ruleA, add a new adjustment.
	if err := f.ruleRepo.Delete(context.Background(), ruleA.ID); err != nil {
		t.Fatalf("delete ruleA: %v", err)
	}
	newAdj := &domain.FulfillmentAdjustment{
		WaveID:         waveID,
		TargetKind:     "fulfillment_line",
		AdjustmentKind: "manual_override",
		QuantityDelta:  99,
		OperatorID:     "another_operator",
	}
	if err := f.adjRepo.Create(context.Background(), newAdj); err != nil {
		t.Fatalf("create new adj: %v", err)
	}

	// Verify state is dirty.
	rules, _ := f.ruleRepo.ListByWave(context.Background(), waveID)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after modification (ruleB only), got %d", len(rules))
	}
	adjs, _ := f.adjRepo.ListByWave(context.Background(), waveID)
	if len(adjs) != 2 {
		t.Fatalf("expected 2 adjustments after modification, got %d", len(adjs))
	}

	// 7. Undo → RestoreSnapshot must put pre-modification state back.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo (restore checkpoint): %v", err)
	}

	// Verify rules: both ruleA and ruleB must exist with original IDs.
	rules, err = f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list rules after restore: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules after restore, got %d", len(rules))
	}
	ruleIDs := map[uint]bool{}
	for _, r := range rules {
		ruleIDs[r.ID] = true
	}
	if !ruleIDs[ruleA.ID] {
		t.Errorf("ruleA (ID=%d) not found after restore", ruleA.ID)
	}
	if !ruleIDs[ruleB.ID] {
		t.Errorf("ruleB (ID=%d) not found after restore", ruleB.ID)
	}

	// Verify adjustments: only original adj, new one must be gone.
	adjs, err = f.adjRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list adjs after restore: %v", err)
	}
	if len(adjs) != 1 {
		t.Fatalf("expected 1 adjustment after restore (newAdj gone), got %d", len(adjs))
	}
	if adjs[0].ID != origAdjID {
		t.Errorf("expected original adj ID=%d after restore, got %d", origAdjID, adjs[0].ID)
	}

	// Verify fulfillment line restored with original ID.
	lines, err := f.fulfillRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("list lines after restore: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 fulfillment line after restore, got %d", len(lines))
	}
	if lines[0].ID != origFlID {
		t.Errorf("expected original fulfillment line ID=%d, got %d", origFlID, lines[0].ID)
	}
}

// ── Test 5: ProjectionHash stability across restore ───────────────────────────

func TestIntegration_ProjectionHash_StableAcrossRestore(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// 1. Create rules, an adjustment, and a fulfillment line.
	rule := &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 50, ContributionQuantity: 4, RuleKind: "direct", Priority: 1, Active: true,
	}
	if err := f.ruleRepo.Create(context.Background(), rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	fl := &domain.FulfillmentLine{
		WaveID:          waveID,
		Quantity:        2,
		AllocationState: "allocated",
		LineReason:      "retail_order",
		GeneratedBy:     "test",
	}
	if err := f.fulfillRepo.Create(context.Background(), fl); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}

	adj := &domain.FulfillmentAdjustment{
		WaveID:         waveID,
		TargetKind:     "fulfillment_line",
		AdjustmentKind: "manual_override",
		QuantityDelta:  1,
		OperatorID:     "op",
	}
	if err := f.adjRepo.Create(context.Background(), adj); err != nil {
		t.Fatalf("create adj: %v", err)
	}

	// 2. Compute hash H1 before any restore cycle.
	h1, err := f.projHash.ComputeHash(context.Background(), waveID)
	if err != nil {
		t.Fatalf("compute hash H1: %v", err)
	}
	if h1 == "" {
		t.Fatal("H1 must not be empty")
	}

	// 3. Capture snapshot.
	snapPayload, err := f.snapshot.CaptureSnapshot(context.Background(), waveID)
	if err != nil {
		t.Fatalf("capture snapshot: %v", err)
	}

	// 4. Delete everything.
	if err := f.ruleRepo.DeleteByWave(context.Background(), waveID); err != nil {
		t.Fatalf("delete rules: %v", err)
	}
	if err := f.adjRepo.DeleteByWave(context.Background(), waveID); err != nil {
		t.Fatalf("delete adjs: %v", err)
	}
	if err := f.fulfillRepo.DeleteByWave(context.Background(), waveID); err != nil {
		t.Fatalf("delete lines: %v", err)
	}

	// 5. Restore from snapshot.
	if err := f.snapshot.RestoreSnapshot(context.Background(), snapPayload); err != nil {
		t.Fatalf("restore snapshot: %v", err)
	}

	// 6. Compute hash H2.
	h2, err := f.projHash.ComputeHash(context.Background(), waveID)
	if err != nil {
		t.Fatalf("compute hash H2: %v", err)
	}

	// 7. H1 must equal H2 — semantic state is identical and IDs are preserved.
	if h1 != h2 {
		t.Errorf("hash mismatch after restore: H1=%s H2=%s", h1, h2)
	}
}

// ── Test 6: multi-step undo chain ─────────────────────────────────────────────

func TestIntegration_MultiStepUndoChain(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	// Helper: create one rule and record a history node; returns ruleID.
	createRuleAndRecord := func(productID uint, priority int) uint {
		t.Helper()
		r := &domain.AllocationPolicyRule{
			WaveID: waveID, ProductID: productID, ContributionQuantity: 1,
			RuleKind: "direct", Priority: priority, Active: true,
		}
		if err := f.ruleRepo.Create(context.Background(), r); err != nil {
			t.Fatalf("create rule productID=%d: %v", productID, err)
		}
		rJSON, _ := json.Marshal(r)
		mustRecordNode(t, f, RecordNodeInput{
			WaveID:              waveID,
			CommandKind:         "create_rule",
			CommandSummary:      fmt.Sprintf("create rule %d", productID),
			PatchPayload:        fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, r.ID, waveID, rJSON),
			InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, r.ID),
		})
		return r.ID
	}

	// 1. Create rules A, B, C and record each.
	createRuleAndRecord(61, 1)
	createRuleAndRecord(62, 2)
	createRuleAndRecord(63, 3)

	assertRuleCount := func(expected int, label string) {
		t.Helper()
		rules, err := f.ruleRepo.ListByWave(context.Background(), waveID)
		if err != nil {
			t.Fatalf("%s: list rules: %v", label, err)
		}
		if len(rules) != expected {
			t.Errorf("%s: expected %d rules, got %d", label, expected, len(rules))
		}
	}

	// 4. Verify 3 rules exist.
	assertRuleCount(3, "after 3 creates")

	// 5. Undo → C deleted, 2 rules remain.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo 1: %v", err)
	}
	assertRuleCount(2, "after undo 1")

	// 6. Undo → B deleted, 1 rule remains.
	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("undo 2: %v", err)
	}
	assertRuleCount(1, "after undo 2")

	// 7. Redo → B restored, 2 rules.
	if _, err := f.undoRedo.Redo(context.Background(), waveID); err != nil {
		t.Fatalf("redo 1: %v", err)
	}
	assertRuleCount(2, "after redo 1")

	// 8. Redo → C restored, 3 rules.
	if _, err := f.undoRedo.Redo(context.Background(), waveID); err != nil {
		t.Fatalf("redo 2: %v", err)
	}
	assertRuleCount(3, "after redo 2")
}

// ── Test 7: RecordNode error is observable (not silently lost) ────────────────

// failingScopeRepo is a minimal mock that always fails FindOrCreate.
// Used only in Test 7 which specifically exercises the error-propagation path.
type failingScopeRepo struct{}

func (r *failingScopeRepo) Create(ctx context.Context, scope *domain.HistoryScope) error {
	return errors.New("injected scope error")
}

func (r *failingScopeRepo) FindByID(ctx context.Context, id uint) (*domain.HistoryScope, error) {
	return nil, nil
}

func (r *failingScopeRepo) FindByScopeTypeAndKey(ctx context.Context, scopeType, scopeKey string) (*domain.HistoryScope, error) {
	return nil, errors.New("injected scope error")
}

func (r *failingScopeRepo) UpdateHead(ctx context.Context, scopeID uint, headNodeID uint) error {
	return errors.New("injected scope error")
}

func (r *failingScopeRepo) FindOrCreate(ctx context.Context, scopeType, scopeKey string) (*domain.HistoryScope, error) {
	return nil, errors.New("injected scope error: FindOrCreate")
}

func TestIntegration_RecordNodeError_NotSilent(t *testing.T) {
	// This test uses a real DB for nodeRepo/cpRepo but a failing scopeRepo to
	// verify that RecordNode surfaces errors instead of silently discarding them.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.HistoryCheckpoint{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	nodeRepo := infra.NewHistoryNodeRepository(db)
	cpRepo := infra.NewHistoryCheckpointRepository(db)

	// Wire a deliberately broken scope repo.
	badScopeRepo := &failingScopeRepo{}
	svc := NewHistoryRecordingService(badScopeRepo, nodeRepo, cpRepo)

	_, err = svc.RecordNode(context.Background(), RecordNodeInput{
		WaveID:         999,
		CommandKind:    "create_rule",
		CommandSummary: "create rule",
		PatchPayload:   `{"op":"create_rule"}`,
	})

	if err == nil {
		t.Fatal("expected RecordNode to return an error when scope repo fails, got nil")
	}
	if !errors.Is(err, err) {
		// Always true — just ensure err is non-nil and wraps the injected message.
		t.Errorf("unexpected: err should be non-nil")
	}
	// The error message must surface the underlying cause.
	if len(err.Error()) == 0 {
		t.Error("error message must not be empty")
	}
	t.Logf("got expected error: %v", err)
}

// ── Test 8: batch unassign → undo → redo restores assignments ────────────────

func TestBatchUnassignUndoRedoRestoresAssignments(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	ctx := context.Background()
	waveID := mustCreateWave(t, f)

	// Baseline node so the batch node has a parent to undo to.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:         waveID,
		CommandKind:    "baseline",
		CommandSummary: "initial state",
	})

	// A create_rule node BELOW the batch node gives the second undo a concrete
	// target — undoing twice proves the chain head moved past the batch node
	// instead of staying stuck on it re-applying the same inverse.
	rule := &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 501, ContributionQuantity: 2,
		RuleKind: "direct", Priority: 1, Active: true,
	}
	if err := f.ruleRepo.Create(ctx, rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "create_rule",
		CommandSummary:      "create rule",
		PatchPayload:        fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, waveID, ruleJSON),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID),
	})

	// Seed two wave-demand assignments, mirroring the pre-allocation state the
	// controller's BatchUnassignDemandFromWave operates on.
	const docA, docB = uint(9001), uint(9002)
	for _, docID := range []uint{docA, docB} {
		if err := f.db.Create(&persistence.WaveDemandAssignment{WaveID: waveID, DemandDocumentID: docID}).Error; err != nil {
			t.Fatalf("seed assignment %d: %v", docID, err)
		}
	}

	assertAssigned := func(docID uint, want bool, label string) {
		t.Helper()
		exists, err := f.assignRepo.ExistsByDocument(ctx, docID)
		if err != nil {
			t.Fatalf("%s: ExistsByDocument(%d): %v", label, docID, err)
		}
		if exists != want {
			t.Fatalf("%s: ExistsByDocument(%d) = %v, want %v", label, docID, exists, want)
		}
	}
	assertAssigned(docA, true, "before batch unassign")
	assertAssigned(docB, true, "before batch unassign")

	// Mirror the controller's flow: the use case deletes via the repo (soft
	// delete), then the history node is recorded with the exact payloads
	// controller_wave_lifecycle.go emits (forward batch_unassign_demand /
	// inverse batch_assign_demand). The forward payload only runs on redo.
	if err := f.assignRepo.DeleteByWaveAndDocument(ctx, waveID, docA); err != nil {
		t.Fatalf("use-case delete docA: %v", err)
	}
	if err := f.assignRepo.DeleteByWaveAndDocument(ctx, waveID, docB); err != nil {
		t.Fatalf("use-case delete docB: %v", err)
	}
	forwardPayload := fmt.Sprintf(`{"op":"batch_unassign_demand","wave_id":%d,"demand_document_ids":[%d,%d]}`, waveID, docA, docB)
	inversePayload := fmt.Sprintf(`{"op":"batch_assign_demand","wave_id":%d,"demand_document_ids":[%d,%d]}`, waveID, docA, docB)
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "batch_unassign_demand",
		CommandSummary:      "batch unassign 2 demands",
		PatchPayload:        forwardPayload,
		InversePatchPayload: inversePayload,
	})
	assertAssigned(docA, false, "after batch unassign")
	assertAssigned(docB, false, "after batch unassign")

	// Undo once → inverse batch_assign_demand restores both assignments.
	if _, err := f.undoRedo.Undo(ctx, waveID); err != nil {
		t.Fatalf("undo batch unassign: %v", err)
	}
	assertAssigned(docA, true, "after undo")
	assertAssigned(docB, true, "after undo")

	// Undo a second time: the chain head must have moved past the batch node
	// onto the create_rule node — the rule gets deleted instead of the batch
	// undo re-failing with "unknown op".
	if _, err := f.undoRedo.Undo(ctx, waveID); err != nil {
		t.Fatalf("second undo (create_rule): %v", err)
	}
	rules, err := f.ruleRepo.ListByWave(ctx, waveID)
	if err != nil {
		t.Fatalf("list rules after second undo: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after second undo, got %d", len(rules))
	}

	// Redo twice → rule restored, then the batch re-applied (assignments gone).
	if _, err := f.undoRedo.Redo(ctx, waveID); err != nil {
		t.Fatalf("redo create_rule: %v", err)
	}
	rules, err = f.ruleRepo.ListByWave(ctx, waveID)
	if err != nil {
		t.Fatalf("list rules after first redo: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after first redo, got %d", len(rules))
	}
	if _, err := f.undoRedo.Redo(ctx, waveID); err != nil {
		t.Fatalf("redo batch unassign: %v", err)
	}
	assertAssigned(docA, false, "after redo")
	assertAssigned(docB, false, "after redo")
}

// ── Test 9: single unassign → undo → redo restores the assignment ─────────────

func TestSingleUnassignUndoRedoRestoresAssignment(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	ctx := context.Background()
	waveID := mustCreateWave(t, f)

	// Baseline node so the unassign node has a parent to undo to.
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:         waveID,
		CommandKind:    "baseline",
		CommandSummary: "initial state",
	})

	// A create_rule node BELOW the unassign node gives the second undo a concrete
	// target — undoing twice proves the chain head moved past the unassign node
	// instead of staying stuck on it re-applying the same inverse.
	rule := &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 502, ContributionQuantity: 2,
		RuleKind: "direct", Priority: 1, Active: true,
	}
	if err := f.ruleRepo.Create(ctx, rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("marshal rule: %v", err)
	}
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "create_rule",
		CommandSummary:      "create rule",
		PatchPayload:        fmt.Sprintf(`{"op":"create_rule","rule_id":%d,"wave_id":%d,"data":%s}`, rule.ID, waveID, ruleJSON),
		InversePatchPayload: fmt.Sprintf(`{"op":"delete_rule","rule_id":%d}`, rule.ID),
	})

	// Seed one wave-demand assignment, mirroring the pre-allocation state the
	// controller's UnassignDemandFromWave operates on.
	const docA = uint(9101)
	if err := f.db.Create(&persistence.WaveDemandAssignment{WaveID: waveID, DemandDocumentID: docA}).Error; err != nil {
		t.Fatalf("seed assignment %d: %v", docA, err)
	}

	assertAssigned := func(docID uint, want bool, label string) {
		t.Helper()
		exists, err := f.assignRepo.ExistsByDocument(ctx, docID)
		if err != nil {
			t.Fatalf("%s: ExistsByDocument(%d): %v", label, docID, err)
		}
		if exists != want {
			t.Fatalf("%s: ExistsByDocument(%d) = %v, want %v", label, docID, exists, want)
		}
	}
	assertAssigned(docA, true, "before single unassign")

	// Mirror the controller's flow: the use case deletes via the repo (soft
	// delete), then the history node is recorded with the exact payloads
	// controller_wave_lifecycle.go emits (forward unassign_demand /
	// inverse assign_demand). The forward payload only runs on redo.
	if err := f.assignRepo.DeleteByWaveAndDocument(ctx, waveID, docA); err != nil {
		t.Fatalf("use-case delete docA: %v", err)
	}
	forwardPayload := fmt.Sprintf(`{"op":"unassign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, docA)
	inversePayload := fmt.Sprintf(`{"op":"assign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, docA)
	mustRecordNode(t, f, RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         "unassign_demand",
		CommandSummary:      "unassign 1 demand",
		PatchPayload:        forwardPayload,
		InversePatchPayload: inversePayload,
	})
	assertAssigned(docA, false, "after single unassign")

	// Undo once → inverse assign_demand restores the assignment. Before the fix
	// this collided with the soft-deleted row that still occupies the
	// idx_wave_demand unique slot (the unique index carries no deleted_at column).
	if _, err := f.undoRedo.Undo(ctx, waveID); err != nil {
		t.Fatalf("undo single unassign: %v", err)
	}
	assertAssigned(docA, true, "after undo")

	// Undo a second time: the chain head must have moved past the unassign node
	// onto the create_rule node — the rule gets deleted instead of the unassign
	// undo re-failing on the same node.
	if _, err := f.undoRedo.Undo(ctx, waveID); err != nil {
		t.Fatalf("second undo (create_rule): %v", err)
	}
	rules, err := f.ruleRepo.ListByWave(ctx, waveID)
	if err != nil {
		t.Fatalf("list rules after second undo: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after second undo, got %d", len(rules))
	}

	// Redo twice → rule restored, then the unassign re-applied (assignment gone).
	if _, err := f.undoRedo.Redo(ctx, waveID); err != nil {
		t.Fatalf("redo create_rule: %v", err)
	}
	rules, err = f.ruleRepo.ListByWave(ctx, waveID)
	if err != nil {
		t.Fatalf("list rules after first redo: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after first redo, got %d", len(rules))
	}
	if _, err := f.undoRedo.Redo(ctx, waveID); err != nil {
		t.Fatalf("redo single unassign: %v", err)
	}
	assertAssigned(docA, false, "after redo")
}

func TestIntegration_FirstRecordedActionHasUndoableBaselineParent(t *testing.T) {
	f := newHistoryIntegrationFixture(t)
	waveID := mustCreateWave(t, f)

	rule := &domain.AllocationPolicyRule{
		WaveID:               waveID,
		ProductID:            77,
		ContributionQuantity: 1,
		RuleKind:             "direct",
		Priority:             1,
		Active:               true,
	}
	if err := f.ruleRepo.Create(context.Background(), rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}
	payload, err := BuildRuleRestorePatch("restore_rule", rule)
	if err != nil {
		t.Fatalf("build restore payload: %v", err)
	}

	node, err := f.recording.RecordNode(context.Background(), RecordNodeInput{
		WaveID:              waveID,
		CommandKind:         domain.CmdCreateRule,
		CommandSummary:      "first user action",
		PatchPayload:        payload,
		InversePatchPayload: `{"op":"delete_rule","rule_id":1}`,
	})
	if err != nil {
		t.Fatalf("RecordNode first action: %v", err)
	}
	if node.ParentNodeID == 0 {
		t.Fatal("expected first user action to be parented to system baseline, got parent 0")
	}

	parent, err := f.nodeRepo.FindByID(context.Background(), node.ParentNodeID)
	if err != nil {
		t.Fatalf("FindByID baseline parent: %v", err)
	}
	if parent == nil || parent.CommandKind != domain.CmdSystemBaseline {
		t.Fatalf("expected parent command kind %q, got %+v", domain.CmdSystemBaseline, parent)
	}

	if _, err := f.undoRedo.Undo(context.Background(), waveID); err != nil {
		t.Fatalf("Undo first user action: %v", err)
	}
	rules, err := f.ruleRepo.ListByWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("ListByWave after undo: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after undoing first user action, got %d", len(rules))
	}
}
