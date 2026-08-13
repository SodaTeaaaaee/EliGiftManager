package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// mockWaveLifecycleRepo implements domain.WaveLifecycleRepository directly against
// a *mockWaveRepo's underlying storage (internal/app/use_cases_test.go), so
// UpdateWave/CloseWave observe the same wave state that FindByID reads back.
type mockWaveLifecycleRepo struct {
	waveRepo *mockWaveRepo
}

func newMockWaveLifecycleRepo(waveRepo *mockWaveRepo) *mockWaveLifecycleRepo {
	return &mockWaveLifecycleRepo{waveRepo: waveRepo}
}

func (m *mockWaveLifecycleRepo) UpdateWaveFields(ctx context.Context, waveID uint, name, notes, levelTags string) error {
	m.waveRepo.mu.Lock()
	defer m.waveRepo.mu.Unlock()
	w, ok := m.waveRepo.waves[waveID]
	if !ok {
		return fmt.Errorf("wave not found")
	}
	w.Name = name
	w.Notes = notes
	w.LevelTags = levelTags
	return nil
}

func (m *mockWaveLifecycleRepo) TransitionLifecycleStage(ctx context.Context, waveID uint, stage string) error {
	m.waveRepo.mu.Lock()
	defer m.waveRepo.mu.Unlock()
	w, ok := m.waveRepo.waves[waveID]
	if !ok {
		return fmt.Errorf("wave not found")
	}
	w.LifecycleStage = stage
	return nil
}

// newWaveLifecycleTestUC wires a WaveLifecycleUseCase from the shared in-memory
// mocks (see use_cases_test.go), returning the wave repo too so tests can
// seed/inspect wave state directly. profileRepo is nil: assignment tests do not
// bind an integration profile.
func newWaveLifecycleTestUC() (WaveLifecycleUseCase, *mockWaveRepo, *mockDemandRepo, *mockFulfillRepo) {
	waveRepo := newMockWaveRepo()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	fulfillRepo := newMockFulfillRepo()
	lifecycleRepo := newMockWaveLifecycleRepo(waveRepo)

	uc := NewWaveLifecycleUseCase(waveRepo, lifecycleRepo, demandRepo, assignmentRepo, fulfillRepo, nil)
	return uc, waveRepo, demandRepo, fulfillRepo
}

func TestUpdateWavePersistsNameAndNotes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "Wave 1700000000"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}

	result, err := uc.UpdateWave(ctx, dto.UpdateWaveInput{
		WaveID:    wave.ID,
		Name:      "2026-07 会员波",
		Notes:     "monthly membership batch",
		LevelTags: `["gold","silver"]`,
	})
	if err != nil {
		t.Fatalf("UpdateWave: %v", err)
	}
	if result.Name != "2026-07 会员波" || result.Notes != "monthly membership batch" {
		t.Fatalf("unexpected UpdateWave result: %+v", result)
	}

	// Re-fetch independently to confirm the write actually persisted, not just the
	// returned DTO.
	persisted, err := waveRepo.FindByID(ctx, wave.ID)
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if persisted.Name != "2026-07 会员波" || persisted.Notes != "monthly membership batch" || persisted.LevelTags != `["gold","silver"]` {
		t.Fatalf("wave not persisted correctly: %+v", persisted)
	}
}

func TestUpdateWaveRejectsBlankName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "kept"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}

	if _, err := uc.UpdateWave(ctx, dto.UpdateWaveInput{WaveID: wave.ID, Name: ""}); err == nil {
		t.Fatal("expected UpdateWave to reject a blank name, got nil error")
	}
}

func TestCloseWaveRequiresForceAndNoteWhenResidualItemsExist(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, fulfillRepo := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "residual-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	// A draft-allocation line is a residual (unresolved) item.
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
		WaveID:          wave.ID,
		Quantity:        1,
		AllocationState: string(domain.AllocationStateDraft),
		AddressState:    string(domain.AddressStateMissing),
		SupplierState:   string(domain.SupplierStateNotSubmitted),
	}); err != nil {
		t.Fatalf("seed fulfillment line: %v", err)
	}

	// Plain close (no force) must fail while residual items exist.
	if _, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID}); err == nil {
		t.Fatal("expected CloseWave without force to fail when residual items exist")
	}

	// Force without a note must also fail — the note is the audit trail.
	if _, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID, Force: true}); err == nil {
		t.Fatal("expected force-close without a note to fail")
	}

	// Force + note succeeds and records the note.
	result, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID, Force: true, Note: "shipped remainder manually offline"})
	if err != nil {
		t.Fatalf("force CloseWave: %v", err)
	}
	if !result.Forced {
		t.Fatal("expected result.Forced == true")
	}
	if result.ResidualItemCount != 1 {
		t.Fatalf("ResidualItemCount = %d, want 1", result.ResidualItemCount)
	}
	if result.Wave.LifecycleStage != string(domain.LifecycleStageClosed) {
		t.Fatalf("LifecycleStage = %q, want %q", result.Wave.LifecycleStage, domain.LifecycleStageClosed)
	}
	if !strings.Contains(result.Wave.Notes, "shipped remainder manually offline") {
		t.Fatalf("closure note not recorded in wave notes: %q", result.Wave.Notes)
	}
}

func TestCloseWaveWithoutResidualItemsDoesNotRequireForce(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "clean-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}

	result, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID})
	if err != nil {
		t.Fatalf("CloseWave: %v", err)
	}
	if result.Forced {
		t.Fatal("expected Forced == false when there are no residual items")
	}
	if result.Wave.LifecycleStage != string(domain.LifecycleStageClosed) {
		t.Fatalf("LifecycleStage = %q, want %q", result.Wave.LifecycleStage, domain.LifecycleStageClosed)
	}
}

func TestBatchAssignDemandToWaveReturnsPerItemResultsIncludingAFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "batch-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-1"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}

	const missingDocID = uint(999999)
	result, err := uc.BatchAssignDemandToWave(ctx, wave.ID, []uint{doc.ID, missingDocID})
	if err != nil {
		t.Fatalf("BatchAssignDemandToWave: %v", err)
	}
	if result.SuccessCount != 1 || result.FailureCount != 1 {
		t.Fatalf("SuccessCount=%d FailureCount=%d, want 1/1", result.SuccessCount, result.FailureCount)
	}
	if len(result.Results) != 2 {
		t.Fatalf("Results len = %d, want 2", len(result.Results))
	}
	var sawSuccess, sawFailure bool
	for _, r := range result.Results {
		switch r.DemandDocumentID {
		case doc.ID:
			if !r.Success {
				t.Fatalf("expected doc %d to succeed, got error %q", doc.ID, r.Error)
			}
			sawSuccess = true
		case missingDocID:
			if r.Success || r.Error == "" {
				t.Fatalf("expected doc %d to fail with a reason", missingDocID)
			}
			sawFailure = true
		}
	}
	if !sawSuccess || !sawFailure {
		t.Fatalf("expected both a success and a failure result, got: %+v", result.Results)
	}
}

func TestUnassignDemandFromWaveBlockedAfterAllocationStarted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, fulfillRepo := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "unassign-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-2"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}
	docID := doc.ID
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{WaveID: wave.ID, DemandDocumentID: &docID, Quantity: 1}); err != nil {
		t.Fatalf("seed fulfillment line: %v", err)
	}

	if err := uc.UnassignDemandFromWave(ctx, wave.ID, doc.ID); err == nil {
		t.Fatal("expected UnassignDemandFromWave to fail once allocation has started")
	}
}

func TestListWaveFulfillmentRowsFilteredReturnsExpectedSubset(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	waveRepo := newMockWaveRepo()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	overviewProjUC := NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{})
	overviewQueryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		staticProductRepo{}, profileRepo, scopeRepo, nodeRepo, overviewProjUC,
	)

	wave := &domain.Wave{Name: "filter-wave"}
	if err := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo).CreateWave(ctx, wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
		WaveID:           wave.ID,
		Quantity:         1,
		AllocationState:  string(domain.AllocationStateReady),
		AddressState:     string(domain.AddressStateReady),
		SupplierState:    string(domain.SupplierStateNotSubmitted),
		ChannelSyncState: string(domain.ChannelSyncStateNotRequired),
	}); err != nil {
		t.Fatalf("seed ready line: %v", err)
	}
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
		WaveID:           wave.ID,
		Quantity:         1,
		AllocationState:  string(domain.AllocationStateDraft),
		AddressState:     string(domain.AddressStateMissing),
		SupplierState:    string(domain.SupplierStateNotSubmitted),
		ChannelSyncState: string(domain.ChannelSyncStateNotRequired),
	}); err != nil {
		t.Fatalf("seed draft line: %v", err)
	}

	filterUC := NewWaveFulfillmentFilterUseCase(waveRepo, overviewQueryUC)
	page, err := filterUC.ListWaveFulfillmentRowsFiltered(ctx, dto.WaveFulfillmentFilterInput{
		WaveID:           wave.ID,
		AllocationStates: []string{string(domain.AllocationStateReady)},
		Pagination:       dto.PaginationInput{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("ListWaveFulfillmentRowsFiltered: %v", err)
	}
	if page.Pagination.TotalCount != 1 {
		t.Fatalf("TotalCount = %d, want 1", page.Pagination.TotalCount)
	}
	if len(page.Items) != 1 || page.Items[0].AllocationState != string(domain.AllocationStateReady) {
		t.Fatalf("unexpected filtered items: %+v", page.Items)
	}

	// Sanity: an unfiltered call should see both lines.
	allPage, err := filterUC.ListWaveFulfillmentRowsFiltered(ctx, dto.WaveFulfillmentFilterInput{
		WaveID:     wave.ID,
		Pagination: dto.PaginationInput{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("ListWaveFulfillmentRowsFiltered (unfiltered): %v", err)
	}
	if allPage.Pagination.TotalCount != 2 {
		t.Fatalf("unfiltered TotalCount = %d, want 2", allPage.Pagination.TotalCount)
	}
}

func TestListWavesPaginatedTypedFiltersByStageTypeAndKeyword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	waveRepo := newMockWaveRepo()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	shipmentRepo := newMockShipmentRepo()
	profileRepo := newMockProfileRepo()
	scopeRepo := newMockHistoryScopeRepo()
	nodeRepo := newMockHistoryNodeRepo()
	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	overviewProjUC := NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{})
	overviewQueryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo,
		staticProductRepo{}, profileRepo, scopeRepo, nodeRepo, overviewProjUC,
	)

	// Seed three waves: A (active/membership, "舰长回馈"), B (closed/retail,
	// "6月零售"), C (active/retail, "无关"). CreateWave keeps a pre-set
	// LifecycleStage and generates a dated WaveNo, matching production.
	waveUC := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)
	seeds := []*domain.Wave{
		{Name: "舰长回馈", LifecycleStage: "active", WaveType: "membership"},
		{Name: "6月零售", LifecycleStage: "closed", WaveType: "retail"},
		{Name: "无关", LifecycleStage: "active", WaveType: "retail"},
	}
	for _, w := range seeds {
		if err := waveUC.CreateWave(ctx, w); err != nil {
			t.Fatalf("seed wave %q: %v", w.Name, err)
		}
	}

	filterUC := NewWaveFulfillmentFilterUseCase(waveRepo, overviewQueryUC)

	// Sanity: an unfiltered call should see all three waves.
	allPage, err := filterUC.ListWavesPaginatedTyped(ctx, dto.WaveListFilterInput{
		PaginationInput: dto.PaginationInput{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("ListWavesPaginatedTyped (unfiltered): %v", err)
	}
	if allPage.Pagination.TotalCount != 3 {
		t.Fatalf("unfiltered TotalCount = %d, want 3", allPage.Pagination.TotalCount)
	}

	// (1) {LifecycleStage:"active", WaveType:"retail"} -> only C (B is closed).
	page, err := filterUC.ListWavesPaginatedTyped(ctx, dto.WaveListFilterInput{
		PaginationInput: dto.PaginationInput{Page: 1, PageSize: 20},
		LifecycleStage:  "active",
		WaveType:        "retail",
	})
	if err != nil {
		t.Fatalf("ListWavesPaginatedTyped (stage+type): %v", err)
	}
	if page.Pagination.TotalCount != 1 || len(page.Items) != 1 {
		t.Fatalf("stage+type filter: TotalCount=%d items=%d, want 1/1", page.Pagination.TotalCount, len(page.Items))
	}
	if page.Items[0].Name != "无关" || page.Items[0].LifecycleStage != "active" || page.Items[0].WaveType != "retail" {
		t.Fatalf("stage+type filter returned unexpected item: %+v", page.Items[0])
	}

	// (2) {NameKeyword:"舰长"} -> only A.
	kwPage, err := filterUC.ListWavesPaginatedTyped(ctx, dto.WaveListFilterInput{
		PaginationInput: dto.PaginationInput{Page: 1, PageSize: 20},
		NameKeyword:     "舰长",
	})
	if err != nil {
		t.Fatalf("ListWavesPaginatedTyped (keyword): %v", err)
	}
	if kwPage.Pagination.TotalCount != 1 || len(kwPage.Items) != 1 || kwPage.Items[0].Name != "舰长回馈" {
		t.Fatalf("keyword filter: TotalCount=%d items=%d, want only 舰长回馈; got %+v", kwPage.Pagination.TotalCount, len(kwPage.Items), kwPage.Items)
	}

	// (3) All three filters combined: stage=active already excludes B, and B is the
	// only wave whose name contains "6月" — so the result set is empty.
	emptyPage, err := filterUC.ListWavesPaginatedTyped(ctx, dto.WaveListFilterInput{
		PaginationInput: dto.PaginationInput{Page: 1, PageSize: 20},
		LifecycleStage:  "active",
		WaveType:        "retail",
		NameKeyword:     "6月",
	})
	if err != nil {
		t.Fatalf("ListWavesPaginatedTyped (combined): %v", err)
	}
	if emptyPage.Pagination.TotalCount != 0 || len(emptyPage.Items) != 0 {
		t.Fatalf("combined filter: TotalCount=%d items=%d, want empty", emptyPage.Pagination.TotalCount, len(emptyPage.Items))
	}
}

func TestBatchUnassignDemandFromWaveReturnsPerItemResults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, fulfillRepo := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "batch-unassign-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	if wave.ID != 1 {
		t.Fatalf("seed wave ID = %d, want 1", wave.ID)
	}

	// Pre-advance the mock's auto-increment counter so the seeded documents land
	// on the fixed IDs 10/11/12/13 the call below references.
	demandRepo.mu.Lock()
	demandRepo.lastID = 9
	demandRepo.mu.Unlock()
	for _, seedID := range []uint{10, 11, 12, 13} {
		doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: fmt.Sprintf("DOC-%d", seedID)}
		if err := demandRepo.Create(ctx, doc); err != nil {
			t.Fatalf("seed demand document %d: %v", seedID, err)
		}
		if doc.ID != seedID {
			t.Fatalf("seeded demand document ID = %d, want %d", doc.ID, seedID)
		}
	}

	// Docs 10/11/12 are mounted to the wave (doc 13 deliberately stays in the
	// unassigned pool — it must fail per-item with "nothing to unassign").
	assigned, err := uc.BatchAssignDemandToWave(ctx, 1, []uint{10, 11, 12})
	if err != nil {
		t.Fatalf("seed assignments: %v", err)
	}
	if assigned.SuccessCount != 3 || assigned.FailureCount != 0 {
		t.Fatalf("seed assignments result = %+v, want 3/0", assigned)
	}

	// doc 12 already has a fulfillment line — allocation has started for it, so
	// it must fail while docs 10 and 11 unassign cleanly.
	doc12 := uint(12)
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{WaveID: wave.ID, DemandDocumentID: &doc12, Quantity: 1}); err != nil {
		t.Fatalf("seed fulfillment line: %v", err)
	}

	result, err := uc.BatchUnassignDemandFromWave(ctx, 1, []uint{10, 11, 12, 13})
	if err != nil {
		t.Fatal(err)
	}
	if result.SuccessCount != 2 || result.FailureCount != 2 {
		t.Fatalf("want 2/2, got %+v", result)
	}
	// doc 12's failure reason must contain "allocation has started".
	if result.Results[2].Success || !strings.Contains(result.Results[2].Error, "allocation has started") {
		t.Fatalf("doc 12 should fail with allocation-started reason, got %+v", result.Results[2])
	}
	// doc 13 is not mounted anywhere — it must fail with "nothing to unassign"
	// instead of being counted as a phantom success.
	if result.Results[3].Success || !strings.Contains(result.Results[3].Error, "nothing to unassign") {
		t.Fatalf("doc 13 should fail with nothing-to-unassign reason, got %+v", result.Results[3])
	}
}

func TestAssignOneRejectsAlreadyAssignedDocument(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave1 := &domain.Wave{Name: "assign-gate-wave-1"}
	if err := waveRepo.Create(ctx, wave1); err != nil {
		t.Fatalf("seed wave 1: %v", err)
	}
	wave2 := &domain.Wave{Name: "assign-gate-wave-2"}
	if err := waveRepo.Create(ctx, wave2); err != nil {
		t.Fatalf("seed wave 2: %v", err)
	}

	// Pre-advance the mock's auto-increment counter so the seeded document lands
	// on the fixed ID 10 the call below references.
	demandRepo.mu.Lock()
	demandRepo.lastID = 9
	demandRepo.mu.Unlock()
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-10"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}
	if doc.ID != 10 {
		t.Fatalf("seeded demand document ID = %d, want 10", doc.ID)
	}

	// First assign doc 10 to wave 1 — this must succeed.
	first, err := uc.BatchAssignDemandToWave(ctx, wave1.ID, []uint{10})
	if err != nil {
		t.Fatalf("first BatchAssignDemandToWave: %v", err)
	}
	if first.SuccessCount != 1 || first.FailureCount != 0 {
		t.Fatalf("first assignment result = %+v, want 1 success / 0 failures", first)
	}

	// Assigning the same doc to wave 2 must be rejected item-wise (#34: no
	// cross-wave split).
	result, err := uc.BatchAssignDemandToWave(ctx, wave2.ID, []uint{10})
	if err != nil {
		t.Fatalf("second BatchAssignDemandToWave: %v", err)
	}
	if result.SuccessCount != 0 || result.FailureCount != 1 {
		t.Fatalf("second assignment result = %+v, want 0 successes / 1 failure", result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(result.Results))
	}
	item := result.Results[0]
	if item.DemandDocumentID != 10 {
		t.Fatalf("failed item DemandDocumentID = %d, want 10", item.DemandDocumentID)
	}
	if item.Success {
		t.Fatal("expected doc 10 to fail on the second assignment, got success")
	}
	if !strings.Contains(item.Error, "already assigned") {
		t.Fatalf("expected error to contain %q, got %q", "already assigned", item.Error)
	}
}

func TestAssignOneRejectsMembershipDocWithPendingIntakeLines(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "membership-gate-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	if wave.ID != 1 {
		t.Fatalf("seed wave ID = %d, want 1", wave.ID)
	}

	// Pre-advance the mock's auto-increment counter so the seeded document lands
	// on the fixed ID 20 the call below references.
	demandRepo.mu.Lock()
	demandRepo.lastID = 19
	demandRepo.mu.Unlock()
	doc := &domain.DemandDocument{
		Kind:             string(domain.DemandKindMembershipEntitlement),
		SourceChannel:    "test",
		SourceDocumentNo: "DOC-20",
	}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}
	if doc.ID != 20 {
		t.Fatalf("seeded demand document ID = %d, want 20", doc.ID)
	}
	if err := demandRepo.CreateLine(ctx, &domain.DemandLine{
		DemandDocumentID:   doc.ID,
		RoutingDisposition: string(domain.RoutingDispositionPendingIntake),
	}); err != nil {
		t.Fatalf("seed pending_intake demand line: %v", err)
	}

	// A membership_entitlement doc with a pending_intake line must not enter a
	// wave until triage completes.
	result, err := uc.BatchAssignDemandToWave(ctx, 1, []uint{20})
	if err != nil {
		t.Fatalf("BatchAssignDemandToWave: %v", err)
	}
	if result.SuccessCount != 0 || result.FailureCount != 1 {
		t.Fatalf("assignment result = %+v, want 0 successes / 1 failure", result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("Results len = %d, want 1", len(result.Results))
	}
	item := result.Results[0]
	if item.DemandDocumentID != 20 {
		t.Fatalf("failed item DemandDocumentID = %d, want 20", item.DemandDocumentID)
	}
	if item.Success {
		t.Fatal("expected doc 20 to fail with a pending_intake reason, got success")
	}
	if !strings.Contains(item.Error, "pending_intake") {
		t.Fatalf("expected error to contain %q, got %q", "pending_intake", item.Error)
	}
}

func TestCloseWaveRejectsAlreadyClosedWave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "already-closed"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	if _, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID}); err != nil {
		t.Fatalf("first CloseWave: %v", err)
	}

	_, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID, Force: true, Note: "retry"})
	if err == nil {
		t.Fatal("expected CloseWave to reject an already-closed wave")
	}
	if !strings.Contains(err.Error(), "already closed") {
		t.Fatalf("error = %q, want already closed", err.Error())
	}
}

func TestCloseWaveCountsInFlightSupplierStatesAsResidual(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	states := []domain.SupplierState{
		domain.SupplierStateSubmitted,
		domain.SupplierStateAccepted,
		domain.SupplierStateProducing,
		domain.SupplierStatePartiallyShipped,
	}
	for _, state := range states {
		state := state
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()
			uc, waveRepo, _, fulfillRepo := newWaveLifecycleTestUC()
			wave := &domain.Wave{Name: "inflight-" + string(state)}
			if err := waveRepo.Create(ctx, wave); err != nil {
				t.Fatalf("seed wave: %v", err)
			}
			if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
				WaveID:           wave.ID,
				Quantity:         1,
				AllocationState:  string(domain.AllocationStateReady),
				AddressState:     string(domain.AddressStateReady),
				SupplierState:    string(state),
				ChannelSyncState: string(domain.ChannelSyncStateNotRequired),
			}); err != nil {
				t.Fatalf("seed fulfillment line: %v", err)
			}
			if _, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID}); err == nil {
				t.Fatalf("expected CloseWave without force to fail for supplier state %s", state)
			}
			result, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID, Force: true, Note: "override in-flight remainder"})
			if err != nil {
				t.Fatalf("force CloseWave: %v", err)
			}
			if !result.Forced || result.ResidualItemCount != 1 {
				t.Fatalf("result = %+v, want Forced with ResidualItemCount 1", result)
			}
		})
	}
}

func TestCloseWaveDoesNotTreatShippedSyncedLineAsResidual(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, _, fulfillRepo := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "shipped-synced"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
		WaveID:           wave.ID,
		Quantity:         1,
		AllocationState:  string(domain.AllocationStateReady),
		AddressState:     string(domain.AddressStateReady),
		SupplierState:    string(domain.SupplierStateShipped),
		ChannelSyncState: string(domain.ChannelSyncStateSynced),
	}); err != nil {
		t.Fatalf("seed fulfillment line: %v", err)
	}

	result, err := uc.CloseWave(ctx, dto.CloseWaveInput{WaveID: wave.ID})
	if err != nil {
		t.Fatalf("CloseWave: %v", err)
	}
	if result.Forced || result.ResidualItemCount != 0 {
		t.Fatalf("result = %+v, want unforced close with no residual items", result)
	}
}

func TestAssignOneRejectsClosedWave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "closed-assign", LifecycleStage: string(domain.LifecycleStageClosed)}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-CLOSED"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}

	_, err := uc.BatchAssignDemandToWave(ctx, wave.ID, []uint{doc.ID})
	if err == nil {
		t.Fatal("expected BatchAssignDemandToWave to reject a closed wave")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Fatalf("error = %q, want closed-wave rejection", err.Error())
	}
}

func TestBatchUnassignDemandFromWaveRejectsDocumentAssignedToDifferentWave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave1 := &domain.Wave{Name: "unassign-src"}
	if err := waveRepo.Create(ctx, wave1); err != nil {
		t.Fatalf("seed wave 1: %v", err)
	}
	wave2 := &domain.Wave{Name: "unassign-other"}
	if err := waveRepo.Create(ctx, wave2); err != nil {
		t.Fatalf("seed wave 2: %v", err)
	}
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-OTHER-WAVE"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}
	assigned, err := uc.BatchAssignDemandToWave(ctx, wave1.ID, []uint{doc.ID})
	if err != nil || assigned.SuccessCount != 1 {
		t.Fatalf("seed assignment: %+v, %v", assigned, err)
	}

	result, err := uc.BatchUnassignDemandFromWave(ctx, wave2.ID, []uint{doc.ID})
	if err != nil {
		t.Fatalf("BatchUnassignDemandFromWave: %v", err)
	}
	if result.SuccessCount != 0 || result.FailureCount != 1 {
		t.Fatalf("result = %+v, want 0 success / 1 failure", result)
	}
	if result.Results[0].Success || !strings.Contains(result.Results[0].Error, "nothing to unassign") {
		t.Fatalf("expected wave-scoped unassign failure, got %+v", result.Results[0])
	}
	if !strings.Contains(result.Results[0].Error, fmt.Sprintf("assigned to wave %d", wave1.ID)) {
		t.Fatalf("expected error to name the actual wave, got %q", result.Results[0].Error)
	}

	stillThere, err := uc.BatchAssignDemandToWave(ctx, wave1.ID, []uint{doc.ID})
	if err != nil {
		t.Fatalf("re-check assignment: %v", err)
	}
	if stillThere.SuccessCount != 0 || stillThere.FailureCount != 1 || !strings.Contains(stillThere.Results[0].Error, "already assigned") {
		t.Fatalf("document should remain assigned to wave 1, got %+v", stillThere)
	}
}

func TestUnassignDemandFromWaveAllowsReassign(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "reassign-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	doc := &domain.DemandDocument{Kind: "retail_order", SourceChannel: "test", SourceDocumentNo: "DOC-REASSIGN"}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("seed demand document: %v", err)
	}
	if assigned, err := uc.BatchAssignDemandToWave(ctx, wave.ID, []uint{doc.ID}); err != nil || assigned.SuccessCount != 1 {
		t.Fatalf("first assign: %+v, %v", assigned, err)
	}
	if err := uc.UnassignDemandFromWave(ctx, wave.ID, doc.ID); err != nil {
		t.Fatalf("UnassignDemandFromWave: %v", err)
	}
	reassigned, err := uc.BatchAssignDemandToWave(ctx, wave.ID, []uint{doc.ID})
	if err != nil || reassigned.SuccessCount != 1 {
		t.Fatalf("reassign: %+v, %v", reassigned, err)
	}
}
