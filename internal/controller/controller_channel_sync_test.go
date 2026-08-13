package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestProjectChannelSyncPendingWithRepoSurfacesBulkUpdateError(t *testing.T) {
	t.Parallel()
	repo := &stubFulfillRepo{bulkErr: errors.New("rows locked")}
	err := projectChannelSyncPendingWithRepo(repo, []domain.ChannelSyncItem{
		{FulfillmentLineID: 11},
		{FulfillmentLineID: 12},
	})
	if err == nil {
		t.Fatal("expected BulkUpdateStates error")
	}
	if !strings.Contains(err.Error(), "project channel sync pending state") {
		t.Errorf("error = %q, want project channel sync pending state", err)
	}
	if !strings.Contains(err.Error(), "rows locked") {
		t.Errorf("error = %q, want wrapped rows locked", err)
	}
	if len(repo.got) != 2 {
		t.Fatalf("expected 2 state updates, got %d", len(repo.got))
	}
	if repo.got[0].ChannelSyncState != "pending" || repo.got[1].ID != 12 {
		t.Errorf("updates = %+v", repo.got)
	}
}

func TestProjectChannelSyncPendingWithRepoNoItemsIsNoop(t *testing.T) {
	t.Parallel()
	repo := &stubFulfillRepo{bulkErr: errors.New("should not be called")}
	if err := projectChannelSyncPendingWithRepo(repo, nil); err != nil {
		t.Fatalf("empty items should skip BulkUpdateStates: %v", err)
	}
}

func TestChannelSyncJobAlreadyExists(t *testing.T) {
	t.Parallel()
	jobs := []domain.ChannelSyncJob{{ID: 4}, {ID: 9}}
	if !channelSyncJobAlreadyExists(jobs, 9) {
		t.Fatal("expected job 9 to be found")
	}
	if channelSyncJobAlreadyExists(jobs, 1) {
		t.Fatal("job 1 must not be treated as existing")
	}
	if channelSyncJobAlreadyExists(nil, 4) {
		t.Fatal("empty list must not match")
	}
}

func TestResolvingExportsExecutorFailsWithoutTempDirFallback(t *testing.T) {
	resolveChannelSyncExportsDirMu.Lock()
	orig := resolveChannelSyncExportsDirFn
	resolveChannelSyncExportsDirFn = func() (string, error) {
		return "", errors.New("forced resolve failure")
	}
	resolveChannelSyncExportsDirMu.Unlock()
	t.Cleanup(func() {
		resolveChannelSyncExportsDirMu.Lock()
		resolveChannelSyncExportsDirFn = orig
		resolveChannelSyncExportsDirMu.Unlock()
	})

	provider := buildExecutorProvider()
	exec, err := provider.Resolve(&domain.IntegrationProfile{
		ProfileKey:       "test.profile",
		TrackingSyncMode: "document_export",
		ConnectorKey:     "eli.local_export",
	})
	if err != nil {
		t.Fatalf("Resolve must still succeed for profile readiness: %v", err)
	}

	_, execErr := exec.Execute(context.Background(), &domain.ChannelSyncJob{ID: 1}, nil, &domain.IntegrationProfile{})
	if execErr == nil {
		t.Fatal("expected resolve exports dir error")
	}
	if !strings.Contains(execErr.Error(), "resolve exports dir") {
		t.Errorf("error = %q, want resolve exports dir", execErr)
	}
	if !strings.Contains(execErr.Error(), "forced resolve failure") {
		t.Errorf("error = %q, want wrapped forced resolve failure", execErr)
	}
	legacyTemp := filepath.Join(os.TempDir(), "EliGiftManager", "exports")
	if strings.Contains(execErr.Error(), legacyTemp) {
		t.Errorf("must not fall back to %q: %v", legacyTemp, execErr)
	}
}

func TestResolvingExportsExecutorWritesUnderResolvedDataExportsDir(t *testing.T) {
	exportsDir := filepath.Join(t.TempDir(), "data", "exports")
	resolveChannelSyncExportsDirMu.Lock()
	orig := resolveChannelSyncExportsDirFn
	resolveChannelSyncExportsDirFn = func() (string, error) {
		return exportsDir, nil
	}
	resolveChannelSyncExportsDirMu.Unlock()
	t.Cleanup(func() {
		resolveChannelSyncExportsDirMu.Lock()
		resolveChannelSyncExportsDirFn = orig
		resolveChannelSyncExportsDirMu.Unlock()
	})

	exec := newResolvingDocumentExportExecutor(nil)
	result, err := exec.Execute(
		context.Background(),
		&domain.ChannelSyncJob{ID: 42, WaveID: 1, IntegrationProfileID: 1, Direction: "push_tracking"},
		[]domain.ChannelSyncItem{{ID: 7, FulfillmentLineID: 1, ShipmentID: 1}},
		&domain.IntegrationProfile{
			SourceSurface:    string(domain.SourceSurfaceRetail),
			TrackingSyncMode: "document_export",
			ConnectorKey:     "eli.local_export",
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Fatal("expected execution result")
	}
	var payload struct {
		OutputFile string `json:"output_file"`
	}
	if unmarshalErr := json.Unmarshal([]byte(result.ResponsePayload), &payload); unmarshalErr != nil {
		t.Fatalf("parse response payload %q: %v", result.ResponsePayload, unmarshalErr)
	}
	if payload.OutputFile == "" || !strings.HasPrefix(payload.OutputFile, exportsDir+string(os.PathSeparator)) {
		t.Fatalf("output must be under resolved data/exports %q, got %q", exportsDir, payload.OutputFile)
	}
	legacyTemp := filepath.Join(os.TempDir(), "EliGiftManager", "exports")
	if strings.HasPrefix(payload.OutputFile, legacyTemp) {
		t.Errorf("must not write under TempDir fallback %q", legacyTemp)
	}
}

func TestBuildExecutorRegistryKeepsConnectorKeysWhenExportsDirUnresolved(t *testing.T) {
	resolveChannelSyncExportsDirMu.Lock()
	orig := resolveChannelSyncExportsDirFn
	resolveChannelSyncExportsDirFn = func() (string, error) {
		return "", errors.New("forced resolve failure")
	}
	resolveChannelSyncExportsDirMu.Unlock()
	t.Cleanup(func() {
		resolveChannelSyncExportsDirMu.Lock()
		resolveChannelSyncExportsDirFn = orig
		resolveChannelSyncExportsDirMu.Unlock()
	})

	caps := buildExecutorRegistry().ListCapabilities()
	if _, ok := caps["eli.local_export"]; !ok {
		t.Fatal("expected eli.local_export capabilities even when exports dir cannot be resolved yet")
	}
	if _, ok := caps["eli.csv_export"]; !ok {
		t.Fatal("expected eli.csv_export capabilities even when exports dir cannot be resolved yet")
	}
}

type stubFulfillRepo struct {
	bulkErr error
	got     []domain.FulfillmentLineStateUpdate
}

func (s *stubFulfillRepo) Create(context.Context, *domain.FulfillmentLine) error {
	return fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) FindByID(context.Context, uint) (*domain.FulfillmentLine, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) ListByWave(context.Context, uint) ([]domain.FulfillmentLine, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) Update(context.Context, *domain.FulfillmentLine) error {
	return fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) DeleteByWave(context.Context, uint) error {
	return fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) DeleteByWaveAndGeneratedBy(context.Context, uint, string) error {
	return fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) ReplaceByWaveAndGeneratedBy(context.Context, uint, string, []domain.FulfillmentLine) error {
	return fmt.Errorf("not implemented")
}
func (s *stubFulfillRepo) BulkUpdateStates(_ context.Context, updates []domain.FulfillmentLineStateUpdate) error {
	s.got = append([]domain.FulfillmentLineStateUpdate(nil), updates...)
	return s.bulkErr
}
func (s *stubFulfillRepo) BulkUpdateCustomerProfileID(context.Context, uint, uint) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}
