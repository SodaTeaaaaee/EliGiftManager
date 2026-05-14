package app

import (
	"fmt"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── mock closure decision repo ──

type mockClosureDecisionRepo struct {
	mu      sync.Mutex
	records map[uint]*domain.ChannelClosureDecisionRecord
	lastID  uint
}

func newMockClosureDecisionRepo() *mockClosureDecisionRepo {
	return &mockClosureDecisionRepo{records: make(map[uint]*domain.ChannelClosureDecisionRecord)}
}

func (m *mockClosureDecisionRepo) next() uint { m.lastID++; return m.lastID }

func (m *mockClosureDecisionRepo) Create(record *domain.ChannelClosureDecisionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	record.ID = m.next()
	cp := *record
	m.records[record.ID] = &cp
	return nil
}

func (m *mockClosureDecisionRepo) AtomicCreate(records []*domain.ChannelClosureDecisionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range records {
		r.ID = m.next()
		cp := *r
		m.records[r.ID] = &cp
	}
	return nil
}

func (m *mockClosureDecisionRepo) ListByFulfillmentLine(fulfillmentLineID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ChannelClosureDecisionRecord
	for _, r := range m.records {
		if r.FulfillmentLineID == fulfillmentLineID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func (m *mockClosureDecisionRepo) ListByWave(waveID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ChannelClosureDecisionRecord
	for _, r := range m.records {
		if r.WaveID == waveID {
			out = append(out, *r)
		}
	}
	return out, nil
}

func setupPendingJob(cs *mockChannelSyncRepo) (jobID uint, item1ID uint, item2ID uint) {
	job := &domain.ChannelSyncJob{
		WaveID:               1,
		IntegrationProfileID: 1,
		Direction:            "push_tracking",
		Status:               "pending",
	}
	if err := cs.CreateJob(job); err != nil {
		panic(fmt.Sprintf("setup: CreateJob: %v", err))
	}
	jobID = job.ID
	item1 := &domain.ChannelSyncItem{
		ChannelSyncJobID:  jobID,
		FulfillmentLineID: 1,
		ShipmentID:        1,
		Status:            "pending",
	}
	item2 := &domain.ChannelSyncItem{
		ChannelSyncJobID:  jobID,
		FulfillmentLineID: 2,
		ShipmentID:        1,
		Status:            "pending",
	}
	if err := cs.CreateItem(item1); err != nil {
		panic(fmt.Sprintf("setup: CreateItem 1: %v", err))
	}
	if err := cs.CreateItem(item2); err != nil {
		panic(fmt.Sprintf("setup: CreateItem 2: %v", err))
	}
	return jobID, item1.ID, item2.ID
}

func ep(executor ChannelSyncExecutor) ExecutorProvider {
	return &StaticExecutorProvider{Executor: executor}
}

// ── execute tests ──

func TestExecuteChannelSyncJobSuccess(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	result, err := uc.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JobStatus != "success" {
		t.Errorf("JobStatus = %q, want success", result.JobStatus)
	}
	if result.StartedAt == "" {
		t.Error("StartedAt should be set")
	}
	if result.FinishedAt == "" {
		t.Error("FinishedAt should be set")
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	for _, it := range result.Items {
		if it.Status != "success" {
			t.Errorf("item %d status = %q, want success", it.ID, it.Status)
		}
	}

	job, _ := cs.FindJobByID(jobID)
	if job.Status != "success" {
		t.Errorf("persisted status = %q, want success", job.Status)
	}
}

func TestExecuteChannelSyncJobPartialSuccess(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewFakePartialExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	result, err := uc.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JobStatus != "partial_success" {
		t.Errorf("JobStatus = %q, want partial_success", result.JobStatus)
	}
}

func TestExecuteChannelSyncJobFailed(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewFakeFailingExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	result, err := uc.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.JobStatus != "failed" {
		t.Errorf("JobStatus = %q, want failed", result.JobStatus)
	}
	for _, it := range result.Items {
		if it.Status != "failed" {
			t.Errorf("item %d status = %q, want failed", it.ID, it.Status)
		}
	}
}

func TestExecuteChannelSyncJobPersistsRequestAndResponsePayload(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	result, err := uc.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequestPayload == "" {
		t.Error("RequestPayload should not be empty")
	}
	if result.ResponsePayload == "" {
		t.Error("ResponsePayload should not be empty")
	}
}

func TestExecuteChannelSyncJobRejectsNonPendingJob(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)
	_, err := uc.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	_, err = uc.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error for re-executing non-pending job, got nil")
	}
}

// ── runtime truth: no executor ──

func TestExecuteChannelSyncJobFailsWhenNoExecutorIsAvailable(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{
		ID:                1,
		TrackingSyncMode:  "api_push",
		ConnectorKey:      "unknown.connector",
		ProfileKey:        "test.profile",
	}
	// runtimeExecutorProvider returns error for unknown connector_key
	uc := NewExecuteSyncUseCase(cs, pr, NewRuntimeExecutorProvider())

	jobID, _, _ := setupPendingJob(cs)

	_, err := uc.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error when no executor is available, got nil")
	}

	// Verify job is persisted as failed, NOT stuck in running
	job, _ := cs.FindJobByID(jobID)
	if job.Status != "failed" {
		t.Errorf("job status = %q, want failed (not running)", job.Status)
	}
	if job.FinishedAt == "" {
		t.Error("job.FinishedAt should be set even on executor resolution failure")
	}
	if job.ErrorMessage == "" {
		t.Error("job.ErrorMessage should be set")
	}
}

// ── runtime truth: executor hard error ──

func TestExecuteChannelSyncJobPersistsFailedStateWhenExecutorReturnsError(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}

	// Use an executor provider that always returns an error
	errorProvider := &failingExecutorProvider{}
	uc := NewExecuteSyncUseCase(cs, pr, errorProvider)

	jobID, _, _ := setupPendingJob(cs)

	_, err := uc.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify persisted state is failed, not running
	job, _ := cs.FindJobByID(jobID)
	if job.Status != "failed" {
		t.Errorf("job status = %q, want failed", job.Status)
	}
	if job.FinishedAt == "" {
		t.Error("job.FinishedAt should be set")
	}
	if job.ErrorMessage == "" {
		t.Error("job.ErrorMessage should be set")
	}
}

type failingExecutorProvider struct{}

func (p *failingExecutorProvider) Resolve(profile *domain.IntegrationProfile) (ChannelSyncExecutor, error) {
	return nil, fmt.Errorf("executor unavailable")
}

// ── retry tests ──

func TestRetryChannelSyncJobRetriesOnlyFailedItems(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	executeUC := NewExecuteSyncUseCase(cs, pr, ep(NewFakePartialExecutor()))
	retryUC := NewRetrySyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	res1, err := executeUC.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	if res1.JobStatus != "partial_success" {
		t.Fatalf("expected partial_success, got %q", res1.JobStatus)
	}

	res2, err := retryUC.RetryChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if res2.JobStatus != "success" {
		t.Errorf("after retry JobStatus = %q, want success", res2.JobStatus)
	}
}

func TestRetryChannelSyncJobRejectsSuccessJob(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	executeUC := NewExecuteSyncUseCase(cs, pr, ep(NewFakeExecutor()))
	retryUC := NewRetrySyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)
	_, err := executeUC.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	_, err = retryUC.RetryChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error for retrying success job, got nil")
	}
}

func TestRetryChannelSyncJobRejectsPendingJob(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	retryUC := NewRetrySyncUseCase(cs, pr, ep(NewFakeExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	_, err := retryUC.RetryChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error for retrying pending job, got nil")
	}
}

// ── runtime truth: hard error marks items failed ──

type hardErrorExecutor struct{}

func NewHardErrorExecutor() ChannelSyncExecutor { return &hardErrorExecutor{} }

func (e *hardErrorExecutor) Execute(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem, profile *domain.IntegrationProfile) (*ChannelSyncExecutionResult, error) {
	return nil, fmt.Errorf("hard executor error")
}

func TestExecuteChannelSyncJobMarksItemsFailedWhenExecutorProviderResolutionFails(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push", ConnectorKey: "unknown.connector"}
	uc := NewExecuteSyncUseCase(cs, pr, NewRuntimeExecutorProvider())

	jobID, item1ID, item2ID := setupPendingJob(cs)

	_, err := uc.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	job, _ := cs.FindJobByID(jobID)
	if job.Status != "failed" {
		t.Errorf("job status = %q, want failed", job.Status)
	}

	items, _ := cs.ListItemsByJob(jobID)
	if len(items) == 0 {
		t.Fatal("expected items to exist")
	}
	failedCount := 0
	for _, it := range items {
		if it.Status == "failed" && it.ErrorMessage != "" {
			failedCount++
		}
	}
	if failedCount != 2 {
		t.Errorf("expected 2 failed items, got %d", failedCount)
	}
	_ = item1ID
	_ = item2ID
}

func TestExecuteChannelSyncJobMarksItemsFailedWhenExecutorReturnsHardError(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}
	uc := NewExecuteSyncUseCase(cs, pr, ep(NewHardErrorExecutor()))

	jobID, _, _ := setupPendingJob(cs)

	_, err := uc.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	job, _ := cs.FindJobByID(jobID)
	if job.Status != "failed" {
		t.Errorf("job status = %q, want failed", job.Status)
	}

	items, _ := cs.ListItemsByJob(jobID)
	failedCount := 0
	for _, it := range items {
		if it.Status == "failed" && it.ErrorMessage != "" {
			failedCount++
		}
	}
	if failedCount != 2 {
		t.Errorf("expected 2 failed items, got %d", failedCount)
	}
}

func TestRetryChannelSyncJobCanRetryAfterProviderResolutionFailure(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push", ConnectorKey: "unknown.connector"}

	// First execute with failing provider → items marked failed
	failingUC := NewExecuteSyncUseCase(cs, pr, NewRuntimeExecutorProvider())
	jobID, _, _ := setupPendingJob(cs)
	_, err := failingUC.ExecuteChannelSyncJob(jobID)
	if err == nil {
		t.Fatal("expected error from failing provider, got nil")
	}

	// Verify items are failed
	items, _ := cs.ListItemsByJob(jobID)
	failedCount := 0
	for _, it := range items {
		if it.Status == "failed" {
			failedCount++
		}
	}
	if failedCount == 0 {
		t.Fatal("expected at least one failed item after provider resolution failure")
	}

	// Now retry with a succeeding executor
	retryUC := NewRetrySyncUseCase(cs, pr, ep(NewFakeExecutor()))
	result, err := retryUC.RetryChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("retry after provider resolution failure: %v", err)
	}
	if result.JobStatus != "success" {
		t.Errorf("after retry JobStatus = %q, want success", result.JobStatus)
	}
}

// ── runtime truth: retry after failed ──

func TestRetryChannelSyncJobCanRetryAfterExecuteFailure(t *testing.T) {
	t.Parallel()
	cs := newMockChannelSyncRepo()
	pr := newMockProfileRepo()
	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, TrackingSyncMode: "api_push"}

	// Execute with failing executor → job ends up as "failed"
	failingUC := NewExecuteSyncUseCase(cs, pr, ep(NewFakeFailingExecutor()))
	jobID, _, _ := setupPendingJob(cs)
	result1, err := failingUC.ExecuteChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1.JobStatus != "failed" {
		t.Fatalf("expected job status failed, got %q", result1.JobStatus)
	}

	// Verify persisted as failed
	job, _ := cs.FindJobByID(jobID)
	if job.Status != "failed" {
		t.Fatalf("persisted job status = %q, want failed", job.Status)
	}

	// Now retry with a succeeding executor
	retryUC := NewRetrySyncUseCase(cs, pr, ep(NewFakeExecutor()))
	result2, err := retryUC.RetryChannelSyncJob(jobID)
	if err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
	if result2.JobStatus != "success" {
		t.Errorf("after retry JobStatus = %q, want success", result2.JobStatus)
	}
}

// ── record closure decision tests ──

func TestRecordChannelClosureDecisionUnsupportedPersistsRecord(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, IntegrationProfileID: uintPtr(1)}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{
				FulfillmentLineID: 1,
				DecisionKind:      "mark_sync_unsupported",
				ReasonCode:        "no_api_access",
				Note:              "Platform does not support API",
				OperatorID:        "op-1",
			},
		},
	}

	records, err := uc.RecordChannelClosureDecision(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].DecisionKind != "mark_sync_unsupported" {
		t.Errorf("DecisionKind = %q, want mark_sync_unsupported", records[0].DecisionKind)
	}
}

func TestRecordChannelClosureDecisionManualCompletedPersistsRecord(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, IntegrationProfileID: uintPtr(1)}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{
				FulfillmentLineID: 1,
				DecisionKind:      "mark_sync_completed_manually",
				ReasonCode:        "offline_handover",
				OperatorID:        "op-2",
			},
		},
	}

	records, err := uc.RecordChannelClosureDecision(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records[0].DecisionKind != "mark_sync_completed_manually" {
		t.Errorf("DecisionKind = %q, want mark_sync_completed_manually", records[0].DecisionKind)
	}
}

func TestRecordChannelClosureDecisionRejectsCrossProfileLine(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, IntegrationProfileID: uintPtr(2)}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{FulfillmentLineID: 1, DecisionKind: "mark_sync_unsupported", OperatorID: "op-1"},
		},
	}

	_, err := uc.RecordChannelClosureDecision(input)
	if err == nil {
		t.Fatal("expected error for cross-profile fulfillment line, got nil")
	}
}

func TestRecordChannelClosureDecisionRejectsManualCompletedWithoutAllowsClosure(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: false}
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, IntegrationProfileID: uintPtr(1)}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{FulfillmentLineID: 1, DecisionKind: "mark_sync_completed_manually", OperatorID: "op-1"},
		},
	}

	_, err := uc.RecordChannelClosureDecision(input)
	if err == nil {
		t.Fatal("expected error for manual_completed with allows_manual_closure=false, got nil")
	}
}

// ── runtime truth: ownership chain gating ──

func TestRecordChannelClosureDecisionRejectsLineWithoutDemandDocument(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	// Fulfillment line has NO DemandDocumentID
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{FulfillmentLineID: 1, DecisionKind: "mark_sync_unsupported", OperatorID: "op-1"},
		},
	}

	_, err := uc.RecordChannelClosureDecision(input)
	if err == nil {
		t.Fatal("expected error for line without DemandDocumentID, got nil")
	}
}

func TestRecordChannelClosureDecisionDoesNotPersistPartialBatchOnValidationFailure(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	// Entry 1: valid fulfillment line
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	dm.docs[10] = &domain.DemandDocument{ID: 10, IntegrationProfileID: uintPtr(1)}
	// Entry 2: fulfillment line belongs to a different profile (cross-profile → invalid)
	fl.lines[2] = &domain.FulfillmentLine{ID: 2, WaveID: 1, DemandDocumentID: uintPtr(20)}
	dm.docs[20] = &domain.DemandDocument{ID: 20, IntegrationProfileID: uintPtr(2)}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{FulfillmentLineID: 1, DecisionKind: "mark_sync_unsupported", OperatorID: "op-1"},
			{FulfillmentLineID: 2, DecisionKind: "mark_sync_unsupported", OperatorID: "op-1"},
		},
	}

	_, err := uc.RecordChannelClosureDecision(input)
	if err == nil {
		t.Fatal("expected error for cross-profile entry in batch, got nil")
	}

	// Assert zero residual records
	if len(dr.records) != 0 {
		t.Errorf("expected 0 records after failed batch, got %d", len(dr.records))
	}
}

func TestRecordChannelClosureDecisionRejectsLineWithoutIntegrationProfileID(t *testing.T) {
	t.Parallel()
	dr := newMockClosureDecisionRepo()
	fl := newMockFulfillRepoForSync()
	pr := newMockProfileRepo()
	dm := newMockDemandRepoForClosure()

	pr.profiles[1] = &domain.IntegrationProfile{ID: 1, AllowsManualClosure: true}
	fl.lines[1] = &domain.FulfillmentLine{ID: 1, WaveID: 1, DemandDocumentID: uintPtr(10)}
	// Demand document exists but has nil IntegrationProfileID
	dm.docs[10] = &domain.DemandDocument{ID: 10}

	uc := NewRecordClosureDecisionUseCase(dr, fl, pr, dm)

	input := dto.RecordClosureDecisionInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		Entries: []dto.RecordClosureDecisionEntry{
			{FulfillmentLineID: 1, DecisionKind: "mark_sync_skipped", OperatorID: "op-1"},
		},
	}

	_, err := uc.RecordChannelClosureDecision(input)
	if err == nil {
		t.Fatal("expected error for line without IntegrationProfileID, got nil")
	}
}
