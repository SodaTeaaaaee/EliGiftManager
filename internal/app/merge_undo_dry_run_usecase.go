package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type CustomerMergeUndoService interface {
	DryRunUndo(ctx context.Context, input dto.CustomerMergeUndoDryRunInput) (*dto.CustomerMergeUndoDryRunResult, error)
	ExecuteUndo(ctx context.Context, input dto.ExecuteCustomerMergeUndoInput) (*dto.ExecuteCustomerMergeUndoResult, error)
}

type customerMergeUndoService struct {
	store domain.MergeExecutionStore
	now   func() time.Time
}

func NewCustomerMergeUndoService(store domain.MergeExecutionStore) CustomerMergeUndoService {
	return &customerMergeUndoService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *customerMergeUndoService) DryRunUndo(ctx context.Context, input dto.CustomerMergeUndoDryRunInput) (*dto.CustomerMergeUndoDryRunResult, error) {
	if input.MergeID == 0 {
		return nil, errors.New("merge ID is required")
	}
	record, err := s.store.FindMergeRecord(ctx, input.MergeID)
	if err != nil {
		return nil, fmt.Errorf("merge record %d not found: %w", input.MergeID, err)
	}
	result := &dto.CustomerMergeUndoDryRunResult{MergeID: record.ID, GeneratedAt: s.now(), AuditLevel: domain.MergeAuditLevelExact}
	if record.MovePlanHash == "" {
		result.AuditLevel = domain.MergeAuditLevelLegacy
		result.Blockers = append(result.Blockers, mergeBlocker("legacy_audit_incomplete", "merge_record", record.ID,
			"exact primary/default/display/name-event state was not recorded"))
		result.Warnings = append(result.Warnings, "legacy merge records remain available through the legacy compatibility adapter only")
		result.RestoreCounts = legacyMergeCounts(record.Payload)
		return result, nil
	}
	if record.Status != domain.MergeRecordStatusCompleted || record.UndoneAt != nil {
		result.Blockers = append(result.Blockers, mergeBlocker("merge_not_completed", "merge_record", record.ID, record.Status))
		return result, nil
	}
	source, sourceErr := s.store.FindProfileForMerge(ctx, record.SourceProfileID, true)
	target, targetErr := s.store.FindProfileForMerge(ctx, record.TargetProfileID, true)
	if sourceErr != nil {
		result.Blockers = append(result.Blockers, mergeBlocker("source_profile_missing", domain.MergeEntityProfile, record.SourceProfileID, sourceErr.Error()))
	}
	if targetErr != nil {
		result.Blockers = append(result.Blockers, mergeBlocker("target_profile_missing", domain.MergeEntityProfile, record.TargetProfileID, targetErr.Error()))
	}
	if source != nil {
		result.SourceRowVersion = source.RowVersion
		if source.Status != domain.CustomerProfileStatusMerged || source.MergedIntoProfileID == nil || *source.MergedIntoProfileID != record.TargetProfileID {
			result.Blockers = append(result.Blockers, mergeBlocker("source_merge_edge_changed", domain.MergeEntityProfile, source.ID, ""))
		}
	}
	if target != nil {
		result.TargetRowVersion = target.RowVersion
		if !activeMergeProfile(target) || target.MergedIntoProfileID != nil {
			result.Blockers = append(result.Blockers, mergeBlocker("target_not_active_root", domain.MergeEntityProfile, target.ID, ""))
		}
	}
	active, err := s.store.ListActiveMergeRecords(ctx, []uint{record.SourceProfileID, record.TargetProfileID})
	if err != nil {
		return nil, fmt.Errorf("inspect merge graph dependencies: %w", err)
	}
	for _, edge := range active {
		if edge.ID == record.ID {
			continue
		}
		dependent := edge.DependsOnMergeRecordID != nil && *edge.DependsOnMergeRecordID == record.ID
		laterSameTarget := edge.TargetProfileID == record.TargetProfileID && mergeRecordAfter(edge, *record)
		targetConsumed := edge.SourceProfileID == record.TargetProfileID
		if dependent || laterSameTarget || targetConsumed {
			result.DependentMergeIDs = append(result.DependentMergeIDs, edge.ID)
		}
	}
	if len(result.DependentMergeIDs) > 0 {
		sort.Slice(result.DependentMergeIDs, func(i, j int) bool { return result.DependentMergeIDs[i] < result.DependentMergeIDs[j] })
		result.Blockers = append(result.Blockers, mergeBlocker("merge_graph_has_dependents", "merge_record", record.ID, fmt.Sprint(result.DependentMergeIDs)))
	}
	moved, err := s.store.ListMovedEntities(ctx, record.ID)
	if err != nil {
		return nil, fmt.Errorf("load exact moved-entity ledger: %w", err)
	}
	if len(moved) == 0 {
		result.Blockers = append(result.Blockers, mergeBlocker("exact_ledger_missing", "merge_record", record.ID, ""))
	}
	states := make([]string, 0, len(moved))
	for _, item := range moved {
		if item.SnapshotVersion != 1 || item.MutationKind == "" || item.AfterSnapshot == "" {
			result.Blockers = append(result.Blockers, mergeBlocker("ledger_entry_incomplete", item.EntityType, item.EntityID, item.MutationKind))
			continue
		}
		current, currentErr := s.store.CurrentEntityState(ctx, item)
		if currentErr != nil {
			result.Blockers = append(result.Blockers, mergeBlocker("moved_entity_missing", item.EntityType, item.EntityID, currentErr.Error()))
			continue
		}
		stateJSON, _ := json.Marshal(current)
		states = append(states, item.EntityType+":"+fmt.Sprint(item.EntityID)+":"+item.MutationKind+":"+string(stateJSON))
		if !mergeStateMatchesForUndo(item, current) {
			result.Blockers = append(result.Blockers, mergeBlocker("moved_entity_changed", item.EntityType, item.EntityID, item.MutationKind))
		}
		if item.EntityType == domain.MergeEntityDemandDocument {
			assigned, assignErr := s.store.IsDemandDocumentAssigned(ctx, item.EntityID)
			if assignErr != nil {
				return nil, fmt.Errorf("check demand document %d assignment: %w", item.EntityID, assignErr)
			}
			if assigned {
				result.Blockers = append(result.Blockers, mergeBlocker("demand_document_frozen_after_merge", item.EntityType, item.EntityID, ""))
			}
		}
	}
	result.RestoreCounts = countsFromMoved(moved)
	result.Blockers = compactMergeBlockers(result.Blockers)
	if len(result.Blockers) == 0 {
		result.Eligible = true
		result.EligibilityToken = undoEligibilityToken(record, result.SourceRowVersion, result.TargetRowVersion, states, result.DependentMergeIDs)
	}
	return result, nil
}

func (s *customerMergeUndoService) ExecuteUndo(ctx context.Context, input dto.ExecuteCustomerMergeUndoInput) (*dto.ExecuteCustomerMergeUndoResult, error) {
	if err := requireCustomerResolutionFeature(ctx, s.store, domain.CustomerResolutionFeatureMergeExecution); err != nil {
		return nil, err
	}
	input.UndoOperationKey = strings.TrimSpace(input.UndoOperationKey)
	if input.MergeID == 0 {
		return nil, errors.New("merge ID is required")
	}
	if input.UndoOperationKey == "" {
		return nil, errors.New("undo operation key is required")
	}
	if strings.TrimSpace(input.ActorRef) == "" {
		return nil, errors.New("undo actor is required")
	}
	if existing, err := s.store.FindMergeRecordByUndoOperationKey(ctx, input.UndoOperationKey); err == nil {
		if existing.ID != input.MergeID {
			return nil, fmt.Errorf("undo idempotency conflict: operation key %q belongs to merge %d", input.UndoOperationKey, existing.ID)
		}
		return s.undoResult(ctx, existing, true)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check undo operation key: %w", err)
	}
	record, err := s.store.FindMergeRecord(ctx, input.MergeID)
	if err != nil {
		return nil, fmt.Errorf("merge record %d not found: %w", input.MergeID, err)
	}
	if record.Status == domain.MergeRecordStatusUndone || record.UndoneAt != nil {
		return s.undoResult(ctx, record, true)
	}
	dryRun, err := s.DryRunUndo(ctx, dto.CustomerMergeUndoDryRunInput{MergeID: input.MergeID})
	if err != nil {
		return nil, err
	}
	if !dryRun.Eligible {
		return nil, fmt.Errorf("merge undo blocked: %s", mergeBlockerCodes(dryRun.Blockers))
	}
	if input.EligibilityToken == "" || input.EligibilityToken != dryRun.EligibilityToken {
		return nil, errors.New("stale undo dry-run: eligibility token no longer matches current state")
	}
	if input.ExpectedSourceRowVersion != dryRun.SourceRowVersion || input.ExpectedTargetRowVersion != dryRun.TargetRowVersion {
		return nil, errors.New("stale undo dry-run: profile row version changed")
	}
	moved, err := s.store.ListMovedEntities(ctx, record.ID)
	if err != nil {
		return nil, fmt.Errorf("load exact moved-entity ledger: %w", err)
	}
	for i := len(moved) - 1; i >= 0; i-- {
		item := moved[i]
		current, currentErr := s.store.CurrentEntityState(ctx, item)
		if currentErr != nil {
			return nil, fmt.Errorf("revalidate moved entity %s/%d: %w", item.EntityType, item.EntityID, currentErr)
		}
		if !mergeStateMatchesForUndo(item, current) {
			return nil, fmt.Errorf("stale undo dry-run: %s/%d changed", item.EntityType, item.EntityID)
		}
		var before domain.MergeEntityState
		if err := json.Unmarshal([]byte(item.BeforeSnapshot), &before); err != nil {
			return nil, fmt.Errorf("decode before snapshot: %w", err)
		}
		if err := s.store.ApplyEntityState(ctx, item, before); err != nil {
			return nil, fmt.Errorf("restore merge entity %s/%d: %w", item.EntityType, item.EntityID, err)
		}
	}
	now := s.now()
	if err := s.store.MarkMovedEntitiesReverted(ctx, record.ID, input.UndoOperationKey, now); err != nil {
		return nil, fmt.Errorf("mark moved ledger reverted: %w", err)
	}
	if record.MergeCandidateID != nil {
		if err := s.store.MarkCandidateStaleAfterUndo(ctx, *record.MergeCandidateID, record.ID); err != nil {
			return nil, fmt.Errorf("mark merge candidate stale: %w", err)
		}
	}
	if err := s.store.MarkPolicyNeedsScan(ctx, record.MergePolicyRevisionID); err != nil {
		return nil, fmt.Errorf("schedule merge policy rescan: %w", err)
	}
	source, err := s.store.FindProfileForMerge(ctx, record.SourceProfileID, true)
	if err != nil {
		return nil, fmt.Errorf("reload restored source profile: %w", err)
	}
	target, err := s.store.FindProfileForMerge(ctx, record.TargetProfileID, true)
	if err != nil {
		return nil, fmt.Errorf("reload target profile: %w", err)
	}
	updated, err := s.store.MarkMergeUndone(ctx, record.ID, record.RowVersion, input.UndoOperationKey,
		dryRun.EligibilityToken, input.ActorRef, input.Reason, source.RowVersion, target.RowVersion, now)
	if err != nil {
		return nil, fmt.Errorf("mark merge undone: %w", err)
	}
	if !updated {
		return nil, errors.New("merge record changed concurrently during undo")
	}
	recordID := record.ID
	payload, _ := json.Marshal(dryRun.RestoreCounts)
	if err := s.store.CreateMergeOperationEvent(ctx, &domain.CustomerMergeOperationEvent{MergeRecordID: &recordID,
		EventKey: "merge:" + input.UndoOperationKey + ":undone", OperationKey: input.UndoOperationKey,
		EventType: "merge_undone", Status: domain.MergeRecordStatusUndone, ActorRef: input.ActorRef,
		ReasonCode: input.Reason, Payload: string(payload), CreatedAt: now}); err != nil {
		return nil, fmt.Errorf("record merge undo event: %w", err)
	}
	return &dto.ExecuteCustomerMergeUndoResult{MergeID: record.ID, Status: domain.MergeRecordStatusUndone,
		RestoredSourceProfileID: record.SourceProfileID, TargetProfileID: record.TargetProfileID,
		RestoreCounts: dryRun.RestoreCounts}, nil
}

func (s *customerMergeUndoService) undoResult(ctx context.Context, record *domain.CustomerMergeRecord, replay bool) (*dto.ExecuteCustomerMergeUndoResult, error) {
	moved, err := s.store.ListMovedEntities(ctx, record.ID)
	if err != nil && record.MovePlanHash != "" {
		return nil, err
	}
	counts := countsFromMoved(moved)
	if record.MovePlanHash == "" {
		counts = legacyMergeCounts(record.Payload)
	}
	return &dto.ExecuteCustomerMergeUndoResult{MergeID: record.ID, Status: domain.MergeRecordStatusUndone,
		RestoredSourceProfileID: record.SourceProfileID, TargetProfileID: record.TargetProfileID,
		RestoreCounts: counts, IdempotentReplay: replay}, nil
}

func undoEligibilityToken(record *domain.CustomerMergeRecord, sourceVersion, targetVersion uint64, states []string, dependents []uint) string {
	sort.Strings(states)
	value := struct {
		RecordID      uint
		RecordVersion uint64
		SourceVersion uint64
		TargetVersion uint64
		PlanHash      string
		States        []string
		Dependents    []uint
	}{RecordID: record.ID, RecordVersion: record.RowVersion, SourceVersion: sourceVersion,
		TargetVersion: targetVersion, PlanHash: record.MovePlanHash, States: states, Dependents: dependents}
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "undo-v1:" + hex.EncodeToString(sum[:])
}

func legacyMergeCounts(payload string) dto.MergeEntityCounts {
	var value domain.CustomerMergePayload
	if json.Unmarshal([]byte(payload), &value) != nil {
		return dto.MergeEntityCounts{}
	}
	return dto.MergeEntityCounts{Identities: len(value.IdentityIDs), Addresses: len(value.AddressIDs),
		DemandDocuments: len(value.DemandDocumentIDs), NameObservations: len(value.NameObservationIDs),
		NameEvents: len(value.NameEventIDs), Origins: len(value.OriginIDs), ProfileMutations: 1}
}

func mergeRecordAfter(left, right domain.CustomerMergeRecord) bool {
	if left.CreatedAt.Equal(right.CreatedAt) {
		return left.ID > right.ID
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func mergeStateMatchesForUndo(moved domain.MergeMovedEntity, actual domain.MergeEntityState) bool {
	if moved.EntityType != domain.MergeEntityProfile {
		return mergeStateMatches(moved.AfterSnapshot, actual)
	}
	var expected domain.MergeEntityState
	if json.Unmarshal([]byte(moved.AfterSnapshot), &expected) != nil {
		return false
	}
	// Row versions stay monotonic across a later merge followed by its LIFO
	// undo. Semantic profile state must match; the eligibility token and
	// expected-version command still bind the undo to the current versions.
	expected.RowVersion = actual.RowVersion
	data, err := json.Marshal(actual)
	expectedData, expectedErr := json.Marshal(expected)
	return err == nil && expectedErr == nil && string(data) == string(expectedData)
}
