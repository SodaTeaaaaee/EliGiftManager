package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type CustomerSplitExecutor interface {
	PreviewSplit(ctx context.Context, input dto.CustomerSplitPreviewInput) (*dto.CustomerSplitPreviewResult, error)
	ExecuteSplit(ctx context.Context, input dto.ExecuteCustomerSplitInput) (*dto.ExecuteCustomerSplitResult, error)
}

type customerSplitExecutor struct {
	store   domain.SplitExecutionStore
	builder *customerSplitPlanBuilder
	now     func() time.Time
}

func NewCustomerSplitExecutor(store domain.SplitExecutionStore) CustomerSplitExecutor {
	return &customerSplitExecutor{store: store, builder: newCustomerSplitPlanBuilder(store), now: func() time.Time { return time.Now().UTC() }}
}

func (e *customerSplitExecutor) PreviewSplit(ctx context.Context, input dto.CustomerSplitPreviewInput) (*dto.CustomerSplitPreviewResult, error) {
	plan, err := e.builder.Build(ctx, input)
	if err != nil {
		return nil, err
	}
	return splitPlanPreviewDTO(plan), nil
}

func (e *customerSplitExecutor) ExecuteSplit(ctx context.Context, input dto.ExecuteCustomerSplitInput) (*dto.ExecuteCustomerSplitResult, error) {
	if err := requireCustomerResolutionFeature(ctx, e.store, domain.CustomerResolutionFeatureSplitExecution); err != nil {
		return nil, err
	}
	input.OperationKey = strings.TrimSpace(input.OperationKey)
	input.ActorRef = strings.TrimSpace(input.ActorRef)
	input.Plan = normalizeSplitPreviewInput(input.Plan)
	if input.OperationKey == "" {
		return nil, errors.New("split operation key is required")
	}
	if input.ActorRef == "" {
		return nil, errors.New("split actor is required for explicit manual execution")
	}
	commandHash, err := hashSplitCommand(input)
	if err != nil {
		return nil, err
	}
	if existing, findErr := e.store.FindSplitRecordByOperationKey(ctx, input.OperationKey); findErr == nil {
		if existing.CommandHash != commandHash {
			return nil, fmt.Errorf("split idempotency conflict: operation key %q was used for a different command", input.OperationKey)
		}
		return e.executionResult(existing, true)
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check split operation key: %w", findErr)
	}

	plan, err := e.builder.Build(ctx, input.Plan)
	if err != nil {
		return nil, err
	}
	if input.ExpectedSourceRowVersion != plan.SourceBefore.RowVersion {
		return nil, errors.New("stale split preview: source profile row version changed")
	}
	if input.ExpectedTargetRowVersion != 0 {
		return nil, errors.New("stale split preview: create_new target must have expected row version 0")
	}
	if input.PlanToken == "" || input.PlanToken != plan.Token {
		return nil, errors.New("stale split preview: token no longer matches the exact selection plan")
	}
	if len(plan.Blockers) > 0 {
		return nil, fmt.Errorf("split blocked: %s", mergeBlockerCodes(plan.Blockers))
	}
	if plan.TargetStrategy != domain.SplitTargetStrategyCreateNew {
		return nil, fmt.Errorf("split target strategy %q is not executable; %s", plan.TargetStrategy, plan.RestoreHint)
	}

	now := e.now()
	target := *plan.TargetDraft
	target.ID = 0
	target.RowVersion = 1
	target.Status = domain.CustomerProfileStatusActive
	target.MergedIntoProfileID = nil
	target.CreatedAt = now
	target.UpdatedAt = now
	target.ExtraData = splitTargetExtraData(plan.Source.ID, input.OperationKey)
	if err := e.store.CreateSplitTargetProfile(ctx, &target); err != nil {
		return nil, fmt.Errorf("create split target profile: %w", err)
	}
	plan.TargetDraft = &target
	plan.bindTarget(target.ID)

	payload, err := splitPayload(plan, input.Plan.Selection)
	if err != nil {
		return nil, err
	}
	sourceSnapshot, _ := json.Marshal(plan.SourceBefore)
	targetSnapshot := splitTargetState(target)
	targetSnapshotJSON, _ := json.Marshal(targetSnapshot)
	record := &domain.CustomerSplitRecord{
		OperationKey: input.OperationKey, CommandHash: commandHash, PreviewHash: plan.Hash, MovePlanHash: plan.Hash,
		Status: domain.SplitRecordStatusExecuting, SourceProfileID: plan.Source.ID, TargetProfileID: target.ID,
		TargetStrategy: domain.SplitTargetStrategyCreateNew, ActorRef: input.ActorRef, DecisionReason: input.DecisionReason,
		SourceRowVersion: plan.SourceBefore.RowVersion, TargetRowVersion: 0, SourceProfileSnapshot: string(sourceSnapshot),
		TargetProfileSnapshot: string(targetSnapshotJSON), Payload: payload, RowVersion: 1,
		ReverseOperationKind: domain.SplitReverseOperationManualMerge, CreatedAt: now,
	}
	if err := e.store.CreateSplitRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("create split audit record: %w", err)
	}
	moved := make([]domain.SplitMovedEntity, len(plan.Moves))
	for i := range plan.Moves {
		moved[i] = plan.Moves[i].Moved
		moved[i].SplitRecordID = record.ID
		moved[i].CreatedAt = now
	}
	if err := e.store.CreateSplitMovedEntities(ctx, moved); err != nil {
		return nil, fmt.Errorf("create exact split moved-entity ledger: %w", err)
	}

	// The target row is created first only to obtain its database ID. The exact
	// ledger is persisted before any existing ownership row or source profile is
	// mutated. All work remains inside the caller's transaction.
	for i := range plan.Moves {
		move := plan.Moves[i]
		current, currentErr := e.store.CurrentSplitEntityState(ctx, move.Moved)
		if currentErr != nil {
			return nil, fmt.Errorf("validate split entity %s/%d: %w", move.Moved.EntityType, move.Moved.EntityID, currentErr)
		}
		expected := move.Before
		if move.Moved.MutationKind == domain.SplitMutationTargetCreated {
			expected = move.After
		}
		if !splitStateEqual(current, expected) {
			return nil, fmt.Errorf("stale split preview: %s/%d changed before execution", move.Moved.EntityType, move.Moved.EntityID)
		}
	}
	for i := range plan.Moves {
		move := plan.Moves[i]
		if move.Moved.MutationKind == domain.SplitMutationTargetCreated {
			continue
		}
		if err := e.store.ApplySplitEntityState(ctx, move.Moved, move.Before, move.After); err != nil {
			return nil, fmt.Errorf("apply split entity %s/%d: %w", move.Moved.EntityType, move.Moved.EntityID, err)
		}
	}
	if err := e.store.InvalidateCandidatesAfterSplit(ctx, plan.Source.ID, target.ID); err != nil {
		return nil, fmt.Errorf("invalidate merge candidates after split: %w", err)
	}
	record.SourceRowVersionAfter = plan.SourceBefore.RowVersion + 1
	record.TargetRowVersionAfter = target.RowVersion
	record.CompletedAt = splitTimePointer(now)
	completed, err := e.store.CompleteSplitRecord(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("complete split audit record: %w", err)
	}
	if !completed {
		return nil, errors.New("split audit record changed concurrently")
	}
	eventPayload, _ := json.Marshal(plan.Counts)
	if err := e.store.CreateSplitOperationEvent(ctx, &domain.CustomerSplitOperationEvent{
		SplitRecordID: record.ID, EventKey: "split:" + input.OperationKey + ":completed", OperationKey: input.OperationKey,
		EventType: "customer_split_completed", Status: domain.SplitRecordStatusCompleted,
		ActorRef: input.ActorRef, ReasonCode: input.DecisionReason, Payload: string(eventPayload), CreatedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("record split completion event: %w", err)
	}
	return &dto.ExecuteCustomerSplitResult{SplitID: record.ID, OperationKey: record.OperationKey,
		Status: domain.SplitRecordStatusCompleted, SourceProfileID: plan.Source.ID, TargetProfileID: target.ID,
		Counts: plan.Counts, SourceRowVersion: record.SourceRowVersionAfter, TargetRowVersion: target.RowVersion,
		DirectUndoSupported: false, ReverseOperationKind: domain.SplitReverseOperationManualMerge}, nil
}

func (e *customerSplitExecutor) executionResult(record *domain.CustomerSplitRecord, replay bool) (*dto.ExecuteCustomerSplitResult, error) {
	var payload customerSplitAuditPayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		return nil, fmt.Errorf("decode split idempotent result: %w", err)
	}
	return &dto.ExecuteCustomerSplitResult{SplitID: record.ID, OperationKey: record.OperationKey,
		Status: record.Status, SourceProfileID: record.SourceProfileID, TargetProfileID: record.TargetProfileID,
		Counts: payload.Counts, SourceRowVersion: record.SourceRowVersionAfter, TargetRowVersion: record.TargetRowVersionAfter,
		IdempotentReplay: replay, DirectUndoSupported: false, ReverseOperationKind: record.ReverseOperationKind}, nil
}

type customerSplitAuditPayload struct {
	Counts           dto.MergeEntityCounts            `json:"counts"`
	Selection        dto.CustomerSplitSelection       `json:"selection"`
	ImmutableHistory domain.SplitImmutableHistoryRefs `json:"immutableHistory"`
	ReverseGuidance  string                           `json:"reverseGuidance"`
}

func splitPayload(plan *customerSplitPlan, selection dto.CustomerSplitSelection) (string, error) {
	encoded, err := json.Marshal(customerSplitAuditPayload{Counts: plan.Counts, Selection: selection,
		ImmutableHistory: plan.ImmutableHistory,
		ReverseGuidance:  "direct split undo is not supported; preview and execute an explicit reviewed merge to reverse this operation"})
	if err != nil {
		return "", fmt.Errorf("encode split audit payload: %w", err)
	}
	return string(encoded), nil
}

func splitPlanPreviewDTO(plan *customerSplitPlan) *dto.CustomerSplitPreviewResult {
	result := &dto.CustomerSplitPreviewResult{PlanToken: plan.Token, PlanHash: plan.Hash, GeneratedAt: plan.GeneratedAt,
		TargetStrategy: plan.TargetStrategy, Counts: plan.Counts, Blockers: plan.Blockers,
		ImmutableHistory: dto.SplitImmutableHistoryDTO{WaveParticipantSnapshotIDs: plan.ImmutableHistory.WaveParticipantSnapshotIDs,
			FulfillmentLineIDs: plan.ImmutableHistory.FulfillmentLineIDs, WillRewrite: false},
		CanExecute: len(plan.Blockers) == 0, DirectUndoSupported: false,
		ReverseOperationKind: domain.SplitReverseOperationManualMerge, UnsupportedTargetStrategyHint: plan.RestoreHint}
	if plan.Source != nil {
		result.SourceProfileID = plan.Source.ID
		result.SourceRowVersion = plan.SourceBefore.RowVersion
		result.SourceDisplayNameAfter = plan.Source.DisplayName
	}
	if plan.TargetDraft != nil {
		result.TargetDisplayNameAfter = plan.TargetDraft.DisplayName
	}
	result.PlannedEntities = make([]dto.MergePlannedEntity, len(plan.Moves))
	for i := range plan.Moves {
		move := plan.Moves[i].Moved
		result.PlannedEntities[i] = dto.MergePlannedEntity{EntityType: move.EntityType, EntityID: move.EntityID,
			MutationKind: move.MutationKind, FromProfileID: move.FromProfileID, ToProfileID: move.ToProfileID}
	}
	return result
}

func hashSplitCommand(input dto.ExecuteCustomerSplitInput) (string, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode split command: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func splitStateEqual(left, right domain.SplitEntityState) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func splitTargetState(target domain.CustomerProfile) domain.SplitEntityState {
	return domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(target.ID), Status: target.Status,
		MergedIntoProfileID: target.MergedIntoProfileID, RowVersion: target.RowVersion, DisplayName: target.DisplayName,
		DisplayNameMode: target.DisplayNameMode, DisplayNameObservationID: target.DisplayNameObservationID}
}

func splitTargetExtraData(sourceProfileID uint, operationKey string) string {
	encoded, _ := json.Marshal(map[string]any{"createdBy": "customer_split", "sourceProfileId": sourceProfileID,
		"operationKey": operationKey})
	return string(encoded)
}

func splitTimePointer(value time.Time) *time.Time { return &value }
