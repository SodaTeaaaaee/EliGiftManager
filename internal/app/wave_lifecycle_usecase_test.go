package app

import (
	"context"
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
		return &testErr{msg: "wave not found"}
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
		return &testErr{msg: "wave not found"}
	}
	w.LifecycleStage = stage
	return nil
}

// newWaveLifecycleTestUC wires a WaveLifecycleUseCase from the shared in-memory
// mocks (see use_cases_test.go / channel_closure_test.go), returning the wave repo
// too so tests can seed/inspect wave state directly.
func newWaveLifecycleTestUC() (WaveLifecycleUseCase, *mockWaveRepo, *mockDemandRepo, *mockFulfillRepo) {
	waveRepo := newMockWaveRepo()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	fulfillRepo := newMockFulfillRepo()
	profileRepo := newMockProfileRepo()
	lifecycleRepo := newMockWaveLifecycleRepo(waveRepo)

	uc := NewWaveLifecycleUseCase(waveRepo, lifecycleRepo, demandRepo, assignmentRepo, fulfillRepo, profileRepo)
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
