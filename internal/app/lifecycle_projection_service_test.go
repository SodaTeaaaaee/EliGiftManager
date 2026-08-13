package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestDeriveStageFromCountsHonorsPersistedClosed(t *testing.T) {
	t.Parallel()

	// Counts that would otherwise derive to syncing_back (or intake if empty).
	snap := ProgressSnapshot{
		DemandCount:        2,
		FulfillmentCount:   2,
		SupplierOrderCount: 1,
		ShipmentCount:      1,
		SyncJobCount:       1,
	}
	jobs := []domain.ChannelSyncJob{{Status: "pending"}}
	got := deriveStageFromCounts(string(domain.LifecycleStageClosed), snap, jobs)
	if got != string(domain.LifecycleStageClosed) {
		t.Errorf("deriveStageFromCounts = %q, want %q", got, domain.LifecycleStageClosed)
	}

	empty := deriveStageFromCounts(string(domain.LifecycleStageClosed), ProgressSnapshot{}, nil)
	if empty != string(domain.LifecycleStageClosed) {
		t.Errorf("deriveStageFromCounts(empty counts) = %q, want %q", empty, domain.LifecycleStageClosed)
	}
}

func TestProjectAndPersistDoesNotReopenClosedWave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	waveRepo := newMockWaveRepo()
	wave := &domain.Wave{Name: "closed", LifecycleStage: string(domain.LifecycleStageClosed)}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("Create wave: %v", err)
	}

	assignmentRepo := newMockAssignmentRepo(nil)
	if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{
		WaveID:           wave.ID,
		DemandDocumentID: 9,
	}); err != nil {
		t.Fatalf("Create assignment: %v", err)
	}

	svc := NewLifecycleProjectionService(
		waveRepo,
		newMockFulfillRepo(),
		newMockSupplierRepo(),
		newMockShipmentRepo(),
		assignmentRepo,
		newMockChannelSyncRepo(),
	)
	if err := svc.ProjectAndPersist(ctx, wave.ID); err != nil {
		t.Fatalf("ProjectAndPersist: %v", err)
	}

	got, err := waveRepo.FindByID(ctx, wave.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.LifecycleStage != string(domain.LifecycleStageClosed) {
		t.Errorf("LifecycleStage = %q, want %q (counts would otherwise be allocation)", got.LifecycleStage, domain.LifecycleStageClosed)
	}
}
