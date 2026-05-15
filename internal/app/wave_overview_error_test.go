package app

import (
	"fmt"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── erroring mock repos for error-propagation tests ──

// errorChannelSyncRepo always returns an error from ListJobsByWave.
type errorChannelSyncRepo struct {
	*mockChannelSyncRepo
}

func (e *errorChannelSyncRepo) ListJobsByWave(_ uint) ([]domain.ChannelSyncJob, error) {
	return nil, fmt.Errorf("mock: channel sync repo unavailable")
}

// errorClosureDecisionRepo always returns an error from ListByWave.
type errorClosureDecisionRepo struct {
	*mockClosureDecisionRepo
}

func (e *errorClosureDecisionRepo) ListByWave(_ uint) ([]domain.ChannelClosureDecisionRecord, error) {
	return nil, fmt.Errorf("mock: closure decision repo unavailable")
}

// ── tests: WaveOverviewProjectionUseCase error propagation ──

// When the channel sync repo fails, ProjectWaveOverview must propagate the error.
func TestProjectWaveOverviewChannelSyncRepoErrorPropagates(t *testing.T) {
	t.Parallel()

	syncRepo := &errorChannelSyncRepo{newMockChannelSyncRepo()}
	closureRepo := newMockClosureDecisionRepo()
	uc := NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{})

	_, err := uc.ProjectWaveOverview(baseOverview(1, "fulfilling"))
	if err == nil {
		t.Fatal("expected error when channel sync repo fails, got nil")
	}
}

// When the closure decision repo fails, ProjectWaveOverview must propagate the error.
func TestProjectWaveOverviewClosureRepoErrorPropagates(t *testing.T) {
	t.Parallel()

	syncRepo := newMockChannelSyncRepo()
	closureRepo := &errorClosureDecisionRepo{newMockClosureDecisionRepo()}
	uc := NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, noopHistoryHeadUC{})

	_, err := uc.ProjectWaveOverview(baseOverview(1, "fulfilling"))
	if err == nil {
		t.Fatal("expected error when closure decision repo fails, got nil")
	}
}

// When no history scope exists (the current steady-state before Stage F),
// ProjectWaveOverview must succeed and return empty BasisDriftSignals.
// This validates the noopHistoryHeadUC path used in production until history recording lands.
func TestProjectWaveOverviewNoHistoryScopeReturnsEmptyDriftSignals(t *testing.T) {
	t.Parallel()

	p := newProjTestSetup() // uses noopHistoryHeadUC and noopDriftUC

	result, err := p.uc.ProjectWaveOverview(baseOverview(1, "fulfilling"))
	if err != nil {
		t.Fatalf("expected no error when no history scope exists, got: %v", err)
	}
	if len(result.BasisDriftSignals) != 0 {
		t.Errorf("expected 0 BasisDriftSignals when no history exists, got %d", len(result.BasisDriftSignals))
	}
	if result.HasDriftedBasis {
		t.Error("expected HasDriftedBasis=false when no history exists")
	}
	if result.HasRequiredReviewBasis {
		t.Error("expected HasRequiredReviewBasis=false when no history exists")
	}
}

// When history head returns an error, ProjectWaveOverview must propagate it.
func TestProjectWaveOverviewHistoryHeadErrorPropagates(t *testing.T) {
	t.Parallel()

	syncRepo := newMockChannelSyncRepo()
	closureRepo := newMockClosureDecisionRepo()

	// historyHeadUC that always errors
	errHeadUC := &errorHistoryHeadUC{}
	uc := NewWaveOverviewProjectionUseCase(syncRepo, closureRepo, noopDriftUC{}, errHeadUC)

	_, err := uc.ProjectWaveOverview(baseOverview(1, "fulfilling"))
	if err == nil {
		t.Fatal("expected error when history head query fails, got nil")
	}
}

// errorHistoryHeadUC always returns an error from GetCurrentProjectionHash.
type errorHistoryHeadUC struct{}

func (e *errorHistoryHeadUC) GetCurrentProjectionHash(_ uint) (string, error) {
	return "", fmt.Errorf("mock: history head unavailable")
}
func (e *errorHistoryHeadUC) GetCurrentHeadNodeIDAndHash(_ uint) (uint, string, error) {
	return 0, "", fmt.Errorf("mock: history head unavailable")
}

// ── tests: WaveUseCase.GetWave error propagation ──

// GetWave for a non-existent wave ID must return a clear error (not nil, not empty struct).
func TestGetWaveNotFoundReturnsError(t *testing.T) {
	t.Parallel()

	waveRepo := newMockWaveRepo()
	uc := NewWaveUseCase(waveRepo, nil, nil)

	_, err := uc.GetWave(9999)
	if err == nil {
		t.Fatal("expected error for non-existent wave ID, got nil")
	}
}

// GetWave for an existing wave must succeed and return the correct wave.
func TestGetWaveExistingReturnsWave(t *testing.T) {
	t.Parallel()

	waveRepo := newMockWaveRepo()
	uc := NewWaveUseCase(waveRepo, nil, nil)

	wave := &domain.Wave{Name: "test wave"}
	if err := uc.CreateWave(wave); err != nil {
		t.Fatalf("setup CreateWave: %v", err)
	}

	got, err := uc.GetWave(wave.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != wave.ID {
		t.Errorf("got wave ID %d, want %d", got.ID, wave.ID)
	}
	if got.Name != "test wave" {
		t.Errorf("got wave name %q, want %q", got.Name, "test wave")
	}
}

// ── tests: WaveOverviewProjectionUseCase base-field preservation ──

// ProjectWaveOverview must preserve all base fields passed in (DemandCount, FulfillmentCount, etc.)
// even when no jobs or decisions exist.
func TestProjectWaveOverviewPreservesBaseFields(t *testing.T) {
	t.Parallel()

	p := newProjTestSetup()

	base := dto.WaveOverviewDTO{
		Wave: dto.WaveDTO{
			ID:             3,
			LifecycleStage: "closed",
		},
		DemandCount:             12,
		FulfillmentCount:        8,
		SupplierOrderCount:      3,
		ShipmentCount:           2,
		TrackedFulfillmentCount: 5,
	}

	result, err := p.uc.ProjectWaveOverview(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DemandCount != 12 {
		t.Errorf("DemandCount = %d, want 12", result.DemandCount)
	}
	if result.FulfillmentCount != 8 {
		t.Errorf("FulfillmentCount = %d, want 8", result.FulfillmentCount)
	}
	if result.SupplierOrderCount != 3 {
		t.Errorf("SupplierOrderCount = %d, want 3", result.SupplierOrderCount)
	}
	if result.ShipmentCount != 2 {
		t.Errorf("ShipmentCount = %d, want 2", result.ShipmentCount)
	}
	if result.TrackedFulfillmentCount != 5 {
		t.Errorf("TrackedFulfillmentCount = %d, want 5", result.TrackedFulfillmentCount)
	}
	// No active jobs → stage preserved
	if result.ProjectedLifecycleStage != "closed" {
		t.Errorf("ProjectedLifecycleStage = %q, want closed", result.ProjectedLifecycleStage)
	}
}
