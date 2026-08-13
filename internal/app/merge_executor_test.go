package app

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type stubMergeExecutionStore struct {
	record     *domain.CustomerMergeRecord
	moved      []domain.MergeMovedEntity
	current    map[string]domain.MergeEntityState
	mergeCand  *domain.MergeCandidate
	events     []domain.CustomerMergeOperationEvent
	applied    []domain.MergeMovedEntity
	eventActor string

	completeOK     bool
	completeStatus string
	markUpdated    bool
	markErr        error

	applyCalls      int
	completeCalls   int
	eventCalls      int
	invalidateCalls int
	markCalls       int
}

func mergeEntityKey(moved domain.MergeMovedEntity) string {
	return moved.EntityType + "/" + strconv.FormatUint(uint64(moved.EntityID), 10)
}

func (s *stubMergeExecutionStore) RequireFeature(context.Context, string) error { return nil }
func (s *stubMergeExecutionStore) FeatureEnabled(context.Context, string) (bool, error) {
	return true, nil
}

func (s *stubMergeExecutionStore) FindMergeRecordByOperationKey(_ context.Context, operationKey string) (*domain.CustomerMergeRecord, error) {
	if s.record != nil && s.record.OperationKey == operationKey {
		cp := *s.record
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubMergeExecutionStore) FindMergeRecord(_ context.Context, id uint) (*domain.CustomerMergeRecord, error) {
	if s.record != nil && s.record.ID == id {
		cp := *s.record
		if s.completeStatus != "" {
			cp.Status = s.completeStatus
		}
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubMergeExecutionStore) ListMovedEntities(_ context.Context, mergeRecordID uint) ([]domain.MergeMovedEntity, error) {
	if s.record == nil || s.record.ID != mergeRecordID {
		return nil, nil
	}
	out := make([]domain.MergeMovedEntity, len(s.moved))
	copy(out, s.moved)
	return out, nil
}

func (s *stubMergeExecutionStore) CurrentEntityState(_ context.Context, moved domain.MergeMovedEntity) (domain.MergeEntityState, error) {
	if s.current == nil {
		return domain.MergeEntityState{}, nil
	}
	return s.current[mergeEntityKey(moved)], nil
}

func (s *stubMergeExecutionStore) ApplyEntityState(_ context.Context, moved domain.MergeMovedEntity, state domain.MergeEntityState) error {
	s.applyCalls++
	s.applied = append(s.applied, moved)
	if s.current == nil {
		s.current = map[string]domain.MergeEntityState{}
	}
	s.current[mergeEntityKey(moved)] = state
	return nil
}

func (s *stubMergeExecutionStore) MarkCandidateExecuted(_ context.Context, _ uint, _ uint64, _ string, _, _, _ uint, _ time.Time) (bool, error) {
	s.markCalls++
	if s.markErr != nil {
		return false, s.markErr
	}
	if s.mergeCand != nil && s.mergeCand.Status == domain.MergeCandidateStatusExecuted {
		return false, nil
	}
	if s.mergeCand != nil {
		s.mergeCand.Status = domain.MergeCandidateStatusExecuted
	}
	return s.markUpdated, nil
}

func (s *stubMergeExecutionStore) FindCandidateExecutionContext(_ context.Context, id uint) (*domain.MergeCandidate, []domain.MergeEvidence, *domain.MergePolicyRevision, error) {
	if s.mergeCand == nil || s.mergeCand.ID != id {
		return nil, nil, nil, errors.New("candidate not found")
	}
	cp := *s.mergeCand
	return &cp, nil, nil, nil
}

func (s *stubMergeExecutionStore) InvalidateCandidatesAfterMerge(context.Context, uint, uint, uint) error {
	s.invalidateCalls++
	return nil
}

func (s *stubMergeExecutionStore) CompleteMergeRecord(_ context.Context, record *domain.CustomerMergeRecord) (bool, error) {
	s.completeCalls++
	if !s.completeOK {
		return false, nil
	}
	if s.record != nil && s.record.ID == record.ID {
		s.record.Status = domain.MergeRecordStatusCompleted
		s.record.CompletedAt = record.CompletedAt
		s.record.SourceRowVersionAfter = record.SourceRowVersionAfter
		s.record.TargetRowVersionAfter = record.TargetRowVersionAfter
		s.record.RowVersion++
	}
	record.Status = domain.MergeRecordStatusCompleted
	return true, nil
}

func (s *stubMergeExecutionStore) CreateMergeOperationEvent(_ context.Context, event *domain.CustomerMergeOperationEvent) error {
	s.eventCalls++
	s.eventActor = event.ActorRef
	s.events = append(s.events, *event)
	return nil
}

func (s *stubMergeExecutionStore) ListMergeOperationEvents(_ context.Context, _ uint) ([]domain.CustomerMergeOperationEvent, error) {
	out := make([]domain.CustomerMergeOperationEvent, len(s.events))
	copy(out, s.events)
	return out, nil
}

func (s *stubMergeExecutionStore) FindProfileForMerge(context.Context, uint, bool) (*domain.CustomerProfile, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListIdentitiesForMerge(context.Context, uint) ([]domain.CustomerIdentity, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListAddressesForMerge(context.Context, uint) ([]domain.CustomerAddress, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListDemandForMerge(context.Context, uint) ([]domain.DemandDocument, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListNameObservationsForMerge(context.Context, uint) ([]domain.CustomerNameObservation, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListNameEventsForMerge(context.Context, uint) ([]domain.CustomerNameEvent, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListOriginsForMerge(context.Context, uint) ([]domain.CustomerProfileOrigin, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) FindMergeRecordByUndoOperationKey(context.Context, string) (*domain.CustomerMergeRecord, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListActiveMergeRecords(context.Context, []uint) ([]domain.CustomerMergeRecord, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) ListMergeRecords(context.Context, domain.MergeHistoryFilter) ([]domain.CustomerMergeRecord, error) {
	return nil, errStubMergeUnused
}
func (s *stubMergeExecutionStore) CreateMergeRecord(context.Context, *domain.CustomerMergeRecord) error {
	return errStubMergeUnused
}
func (s *stubMergeExecutionStore) CreateMovedEntities(context.Context, []domain.MergeMovedEntity) error {
	return errStubMergeUnused
}
func (s *stubMergeExecutionStore) MarkMovedEntitiesReverted(context.Context, uint, string, time.Time) error {
	return errStubMergeUnused
}
func (s *stubMergeExecutionStore) MarkCandidateStaleAfterUndo(context.Context, uint, uint) error {
	return errStubMergeUnused
}
func (s *stubMergeExecutionStore) MarkPolicyNeedsScan(context.Context, *uint) error {
	return errStubMergeUnused
}
func (s *stubMergeExecutionStore) IsDemandDocumentAssigned(context.Context, uint) (bool, error) {
	return false, errStubMergeUnused
}
func (s *stubMergeExecutionStore) MarkMergeUndone(context.Context, uint, uint64, string, string, string, string, uint64, uint64, time.Time) (bool, error) {
	return false, errStubMergeUnused
}

var errStubMergeUnused = errors.New("stub merge store: unexpected call")

var _ domain.MergeExecutionStore = (*stubMergeExecutionStore)(nil)

func mustMergeSnapshot(t *testing.T, state domain.MergeEntityState) string {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func newResumeExecutor(store *stubMergeExecutionStore, now time.Time) *customerMergeExecutor {
	return &customerMergeExecutor{store: store, builder: NewMergePlanBuilder(store), now: func() time.Time { return now }}
}

func TestExecuteMergeReplaysCompletedAndUndone(t *testing.T) {
	t.Parallel()

	for _, status := range []string{domain.MergeRecordStatusCompleted, domain.MergeRecordStatusUndone} {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-replay", ActorRef: "operator:test",
				SourceProfileID: 1, TargetProfileID: 2, DecisionReason: "replay"}
			hash, err := hashMergeCommand(input)
			if err != nil {
				t.Fatal(err)
			}
			store := &stubMergeExecutionStore{
				record: &domain.CustomerMergeRecord{ID: 7, OperationKey: input.OperationKey, CommandHash: hash,
					Status: status, SourceRowVersionAfter: 2, TargetRowVersionAfter: 3},
				completeOK: true,
			}
			result, err := newResumeExecutor(store, time.Now().UTC()).ExecuteMerge(context.Background(), input)
			if err != nil {
				t.Fatalf("ExecuteMerge: %v", err)
			}
			if !result.IdempotentReplay {
				t.Fatal("expected idempotent replay")
			}
			if result.Status != status {
				t.Fatalf("status = %q, want %q", result.Status, status)
			}
			if store.applyCalls != 0 || store.completeCalls != 0 {
				t.Fatalf("replay must not resume apply/complete: apply=%d complete=%d", store.applyCalls, store.completeCalls)
			}
		})
	}
}

func TestExecuteMergeResumesStuckExecuting(t *testing.T) {
	t.Parallel()

	before := domain.MergeEntityState{ProfileID: mergeUintPtr(1), IsPrimary: mergeBoolPtr(true)}
	after := domain.MergeEntityState{ProfileID: mergeUintPtr(2), IsPrimary: mergeBoolPtr(true)}
	input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-resume", ActorRef: "operator:resume",
		SourceProfileID: 1, TargetProfileID: 2, DecisionReason: "stuck"}
	hash, err := hashMergeCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 8, 13, 20, 0, 0, 0, time.UTC)
	moved := domain.MergeMovedEntity{EntityType: domain.MergeEntityIdentity, EntityID: 11,
		FromProfileID: 1, ToProfileID: 2, MutationKind: domain.MergeMutationReassign,
		BeforeSnapshot: mustMergeSnapshot(t, before), AfterSnapshot: mustMergeSnapshot(t, after)}
	store := &stubMergeExecutionStore{
		record: &domain.CustomerMergeRecord{ID: 9, OperationKey: input.OperationKey, CommandHash: hash,
			Status: domain.MergeRecordStatusExecuting, SourceProfileID: 1, TargetProfileID: 2,
			SourceRowVersion: 4, TargetRowVersion: 5, RowVersion: 1},
		moved:      []domain.MergeMovedEntity{moved},
		current:    map[string]domain.MergeEntityState{mergeEntityKey(moved): before},
		completeOK: true,
	}

	result, err := newResumeExecutor(store, fixed).ExecuteMerge(context.Background(), input)
	if err != nil {
		t.Fatalf("ExecuteMerge: %v", err)
	}
	if result.Status != domain.MergeRecordStatusCompleted {
		t.Fatalf("status = %q, want completed (must not succeed while executing)", result.Status)
	}
	if result.IdempotentReplay {
		t.Fatal("successful resume is completion, not an idempotent replay")
	}
	if store.applyCalls != 1 {
		t.Fatalf("applyCalls = %d, want 1", store.applyCalls)
	}
	if store.completeCalls != 1 || store.eventCalls != 1 || store.invalidateCalls != 1 {
		t.Fatalf("complete=%d event=%d invalidate=%d, want 1 each", store.completeCalls, store.eventCalls, store.invalidateCalls)
	}
	if store.eventActor != input.ActorRef {
		t.Fatalf("event actor = %q, want %q", store.eventActor, input.ActorRef)
	}
	if result.SourceRowVersion != 5 || result.TargetRowVersion != 6 {
		t.Fatalf("row versions after = %d/%d, want 5/6", result.SourceRowVersion, result.TargetRowVersion)
	}
	if store.record.Status != domain.MergeRecordStatusCompleted || store.record.CompletedAt == nil || !store.record.CompletedAt.Equal(fixed) {
		t.Fatalf("record not completed at %v: %+v", fixed, store.record)
	}
	if !result.UndoDryRunRequired {
		t.Fatal("expected UndoDryRunRequired")
	}
}

func TestExecuteMergeResumeSkipsAlreadyAppliedEntities(t *testing.T) {
	t.Parallel()

	after := domain.MergeEntityState{ProfileID: mergeUintPtr(2)}
	input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-skip-applied", ActorRef: "operator:resume",
		SourceProfileID: 1, TargetProfileID: 2}
	hash, err := hashMergeCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	moved := domain.MergeMovedEntity{EntityType: domain.MergeEntityAddress, EntityID: 22,
		MutationKind: domain.MergeMutationReassign, AfterSnapshot: mustMergeSnapshot(t, after)}
	store := &stubMergeExecutionStore{
		record: &domain.CustomerMergeRecord{ID: 3, OperationKey: input.OperationKey, CommandHash: hash,
			Status: domain.MergeRecordStatusExecuting, SourceProfileID: 1, TargetProfileID: 2,
			SourceRowVersion: 1, TargetRowVersion: 1, RowVersion: 1},
		moved:      []domain.MergeMovedEntity{moved},
		current:    map[string]domain.MergeEntityState{mergeEntityKey(moved): after},
		completeOK: true,
	}

	result, err := newResumeExecutor(store, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).ExecuteMerge(context.Background(), input)
	if err != nil {
		t.Fatalf("ExecuteMerge: %v", err)
	}
	if result.Status != domain.MergeRecordStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if store.applyCalls != 0 {
		t.Fatalf("already-matching after snapshot must skip ApplyEntityState, got %d calls", store.applyCalls)
	}
	if store.completeCalls != 1 {
		t.Fatalf("completeCalls = %d, want 1", store.completeCalls)
	}
}

func TestExecuteMergeResumeDoesNotSucceedWhileExecuting(t *testing.T) {
	t.Parallel()

	input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-still-executing", ActorRef: "operator:resume",
		SourceProfileID: 1, TargetProfileID: 2}
	hash, err := hashMergeCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	store := &stubMergeExecutionStore{
		record: &domain.CustomerMergeRecord{ID: 4, OperationKey: input.OperationKey, CommandHash: hash,
			Status: domain.MergeRecordStatusExecuting, SourceProfileID: 1, TargetProfileID: 2,
			SourceRowVersion: 1, TargetRowVersion: 1, RowVersion: 1},
		completeOK:     false,
		completeStatus: domain.MergeRecordStatusExecuting,
	}

	result, err := newResumeExecutor(store, time.Now().UTC()).ExecuteMerge(context.Background(), input)
	if err == nil {
		t.Fatalf("expected error while status remains executing, got result %+v", result)
	}
	if result != nil && result.Status == domain.MergeRecordStatusCompleted {
		t.Fatal("must not return completed success while record is still executing")
	}
}

func TestExecuteMergeResumeReloadsWhenAlreadyCompleted(t *testing.T) {
	t.Parallel()

	input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-already-done", ActorRef: "operator:resume",
		SourceProfileID: 1, TargetProfileID: 2}
	hash, err := hashMergeCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	store := &stubMergeExecutionStore{
		record: &domain.CustomerMergeRecord{ID: 5, OperationKey: input.OperationKey, CommandHash: hash,
			Status: domain.MergeRecordStatusExecuting, SourceProfileID: 1, TargetProfileID: 2,
			SourceRowVersion: 1, TargetRowVersion: 1, SourceRowVersionAfter: 2, TargetRowVersionAfter: 2, RowVersion: 2},
		completeOK:     false,
		completeStatus: domain.MergeRecordStatusCompleted,
	}

	result, err := newResumeExecutor(store, time.Now().UTC()).ExecuteMerge(context.Background(), input)
	if err != nil {
		t.Fatalf("ExecuteMerge: %v", err)
	}
	if result.Status != domain.MergeRecordStatusCompleted || !result.IdempotentReplay {
		t.Fatalf("expected completed idempotent replay, got %+v", result)
	}
	if store.eventCalls != 0 {
		t.Fatalf("already-completed complete must not create another event, got %d", store.eventCalls)
	}
}

func TestExecuteMergeResumeSkipsAlreadyExecutedCandidate(t *testing.T) {
	t.Parallel()

	candidateID := uint(42)
	input := dto.ExecuteCustomerMergeInput{OperationKey: "merge-candidate-resume", ActorRef: "operator:resume",
		SourceProfileID: 1, TargetProfileID: 2, CandidateID: &candidateID,
		ExpectedCandidateRowVersion: 3, ExpectedEvidenceHash: "ev", ExpectedPolicyVersion: 1}
	hash, err := hashMergeCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	store := &stubMergeExecutionStore{
		record: &domain.CustomerMergeRecord{ID: 6, OperationKey: input.OperationKey, CommandHash: hash,
			Status: domain.MergeRecordStatusExecuting, SourceProfileID: 1, TargetProfileID: 2,
			MergeCandidateID: &candidateID, SourceRowVersion: 1, TargetRowVersion: 1, RowVersion: 1},
		mergeCand:   &domain.MergeCandidate{ID: candidateID, Status: domain.MergeCandidateStatusExecuted, RowVersion: 4},
		completeOK:  true,
		markUpdated: false,
	}

	result, err := newResumeExecutor(store, time.Now().UTC()).ExecuteMerge(context.Background(), input)
	if err != nil {
		t.Fatalf("already-executed candidate must not fail resume: %v", err)
	}
	if result.Status != domain.MergeRecordStatusCompleted {
		t.Fatalf("status = %q, want completed", result.Status)
	}
	if result.CandidateStatus != domain.MergeCandidateStatusExecuted {
		t.Fatalf("CandidateStatus = %q, want executed", result.CandidateStatus)
	}
	if store.markCalls != 1 || store.completeCalls != 1 {
		t.Fatalf("markCalls=%d completeCalls=%d, want 1 each", store.markCalls, store.completeCalls)
	}
}
