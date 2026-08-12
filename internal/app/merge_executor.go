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

type CustomerMergeExecutor interface {
	PreviewMerge(ctx context.Context, input dto.CustomerMergePreviewInput) (*dto.CustomerMergePreviewResult, error)
	ExecuteMerge(ctx context.Context, input dto.ExecuteCustomerMergeInput) (*dto.ExecuteCustomerMergeResult, error)
}

type customerMergeExecutor struct {
	store   domain.MergeExecutionStore
	builder *MergePlanBuilder
	now     func() time.Time
}

func NewCustomerMergeExecutor(store domain.MergeExecutionStore) CustomerMergeExecutor {
	return &customerMergeExecutor{store: store, builder: NewMergePlanBuilder(store), now: func() time.Time { return time.Now().UTC() }}
}

func (e *customerMergeExecutor) PreviewMerge(ctx context.Context, input dto.CustomerMergePreviewInput) (*dto.CustomerMergePreviewResult, error) {
	plan, err := e.builder.Build(ctx, input)
	if err != nil {
		return nil, err
	}
	return mergePlanPreviewDTO(plan), nil
}

func (e *customerMergeExecutor) ExecuteMerge(ctx context.Context, input dto.ExecuteCustomerMergeInput) (*dto.ExecuteCustomerMergeResult, error) {
	if err := requireCustomerResolutionFeature(ctx, e.store, domain.CustomerResolutionFeatureMergeExecution); err != nil {
		return nil, err
	}
	input.OperationKey = strings.TrimSpace(input.OperationKey)
	if input.OperationKey == "" {
		return nil, errors.New("merge operation key is required")
	}
	if strings.TrimSpace(input.ActorRef) == "" {
		return nil, errors.New("merge actor is required for explicit manual execution")
	}
	commandHash, err := hashMergeCommand(input)
	if err != nil {
		return nil, err
	}
	if existing, findErr := e.store.FindMergeRecordByOperationKey(ctx, input.OperationKey); findErr == nil {
		if existing.CommandHash != commandHash {
			return nil, fmt.Errorf("merge idempotency conflict: operation key %q was used for a different command", input.OperationKey)
		}
		return e.executionResult(ctx, existing, true)
	} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check merge operation key: %w", findErr)
	}
	previewInput := dto.CustomerMergePreviewInput{SourceProfileID: input.SourceProfileID, TargetProfileID: input.TargetProfileID,
		CandidateID: input.CandidateID, PrimaryIdentitySelections: input.PrimaryIdentitySelections,
		DefaultAddressID: input.DefaultAddressID, DisplayNameResolution: input.DisplayNameResolution}
	plan, err := e.builder.Build(ctx, previewInput)
	if err != nil {
		return nil, err
	}
	if input.ExpectedSourceRowVersion != plan.Source.RowVersion || input.ExpectedTargetRowVersion != plan.Target.RowVersion {
		return nil, fmt.Errorf("stale merge preview: profile row version changed")
	}
	if input.PreviewToken == "" || input.PreviewToken != plan.Token {
		return nil, fmt.Errorf("stale merge preview: token no longer matches the exact move plan")
	}
	if plan.Candidate != nil {
		if input.ExpectedCandidateRowVersion != plan.Candidate.RowVersion || input.ExpectedEvidenceHash != plan.Candidate.EvidenceHash ||
			input.ExpectedPolicyVersion != plan.Candidate.PolicyVersion || !equalOptionalUint(input.ExpectedPolicyRevisionID, plan.Candidate.MergePolicyRevisionID) {
			return nil, fmt.Errorf("stale merge preview: candidate evidence or policy revision changed")
		}
	} else if input.CandidateID != nil {
		return nil, errors.New("stale merge preview: candidate is unavailable")
	}
	if len(plan.Blockers) > 0 {
		return nil, fmt.Errorf("merge blocked: %s", mergeBlockerCodes(plan.Blockers))
	}
	now := e.now()
	payload, err := mergePayloadFromPlan(plan)
	if err != nil {
		return nil, err
	}
	sourceSnapshot, targetSnapshot := profileSnapshotsFromPlan(plan)
	record := &domain.CustomerMergeRecord{SourceProfileID: plan.Source.ID, TargetProfileID: plan.Target.ID,
		MergeMode: "manual", DecisionSource: "explicit_manual", DecisionReason: input.DecisionReason,
		ActorRef: input.ActorRef, CorrelationID: input.OperationKey, SourceRowVersion: plan.Source.RowVersion,
		TargetRowVersion: plan.Target.RowVersion, EvidenceSnapshot: maskedEvidenceSnapshot(plan.Evidence), Payload: payload,
		RowVersion: 1, OperationKey: input.OperationKey, CommandHash: commandHash, PreviewHash: plan.Hash,
		MovePlanHash: plan.Hash, Status: domain.MergeRecordStatusExecuting, DependsOnMergeRecordID: plan.Dependency,
		SourceProfileSnapshot: sourceSnapshot, TargetProfileSnapshot: targetSnapshot, CreatedAt: now}
	if plan.Candidate != nil {
		candidateID := plan.Candidate.ID
		record.MergeCandidateID = &candidateID
		record.MergePolicyRevisionID = plan.Candidate.MergePolicyRevisionID
		record.MergeMode = "candidate_review"
	}
	if err := e.store.CreateMergeRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("create merge audit record: %w", err)
	}
	for i := range plan.Moved {
		plan.Moved[i].MergeRecordID = record.ID
		plan.Moved[i].CreatedAt = now
	}
	if err := e.store.CreateMovedEntities(ctx, plan.Moved); err != nil {
		return nil, fmt.Errorf("create exact moved-entity ledger: %w", err)
	}
	for _, moved := range plan.Moved {
		current, err := e.store.CurrentEntityState(ctx, moved)
		if err != nil {
			return nil, fmt.Errorf("validate merge entity %s/%d: %w", moved.EntityType, moved.EntityID, err)
		}
		if !mergeStateMatches(moved.BeforeSnapshot, current) {
			return nil, fmt.Errorf("stale merge preview: %s/%d changed before execution", moved.EntityType, moved.EntityID)
		}
		var after domain.MergeEntityState
		if err := json.Unmarshal([]byte(moved.AfterSnapshot), &after); err != nil {
			return nil, fmt.Errorf("decode after snapshot: %w", err)
		}
		if err := e.store.ApplyEntityState(ctx, moved, after); err != nil {
			return nil, fmt.Errorf("apply merge entity %s/%d: %w", moved.EntityType, moved.EntityID, err)
		}
	}
	if plan.Candidate != nil {
		revisionID := uint(0)
		if plan.Candidate.MergePolicyRevisionID != nil {
			revisionID = *plan.Candidate.MergePolicyRevisionID
		}
		updated, err := e.store.MarkCandidateExecuted(ctx, plan.Candidate.ID, plan.Candidate.RowVersion,
			plan.Candidate.EvidenceHash, plan.Candidate.PolicyVersion, revisionID, record.ID, now)
		if err != nil {
			return nil, fmt.Errorf("mark merge candidate executed: %w", err)
		}
		if !updated {
			return nil, errors.New("stale merge preview: candidate changed during execution")
		}
	}
	executedCandidateID := uint(0)
	if plan.Candidate != nil {
		executedCandidateID = plan.Candidate.ID
	}
	if err := e.store.InvalidateCandidatesAfterMerge(ctx, plan.Source.ID, plan.Target.ID, executedCandidateID); err != nil {
		return nil, fmt.Errorf("invalidate affected merge candidates: %w", err)
	}
	record.SourceRowVersionAfter = plan.Source.RowVersion + 1
	record.TargetRowVersionAfter = plan.Target.RowVersion + 1
	record.CompletedAt = &now
	completed, err := e.store.CompleteMergeRecord(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("complete merge audit record: %w", err)
	}
	if !completed {
		return nil, errors.New("merge audit record changed concurrently")
	}
	eventPayload, _ := json.Marshal(plan.Counts)
	recordID := record.ID
	if err := e.store.CreateMergeOperationEvent(ctx, &domain.CustomerMergeOperationEvent{MergeRecordID: &recordID,
		EventKey: "merge:" + input.OperationKey + ":completed", OperationKey: input.OperationKey,
		EventType: "merge_completed", Status: domain.MergeRecordStatusCompleted, ActorRef: input.ActorRef,
		ReasonCode: input.DecisionReason, Payload: string(eventPayload), CreatedAt: now}); err != nil {
		return nil, fmt.Errorf("record merge completion event: %w", err)
	}
	return &dto.ExecuteCustomerMergeResult{MergeID: record.ID, OperationKey: record.OperationKey,
		Status: domain.MergeRecordStatusCompleted, Counts: plan.Counts, SourceRowVersion: record.SourceRowVersionAfter,
		TargetRowVersion: record.TargetRowVersionAfter, CandidateStatus: candidateStatus(plan.Candidate), UndoDryRunRequired: true}, nil
}

func (e *customerMergeExecutor) executionResult(ctx context.Context, record *domain.CustomerMergeRecord, replay bool) (*dto.ExecuteCustomerMergeResult, error) {
	moved, err := e.store.ListMovedEntities(ctx, record.ID)
	if err != nil {
		return nil, fmt.Errorf("load idempotent merge result: %w", err)
	}
	return &dto.ExecuteCustomerMergeResult{MergeID: record.ID, OperationKey: record.OperationKey, Status: record.Status,
		Counts: countsFromMoved(moved), SourceRowVersion: record.SourceRowVersionAfter, TargetRowVersion: record.TargetRowVersionAfter,
		CandidateStatus: func() string {
			if record.MergeCandidateID != nil {
				return domain.MergeCandidateStatusExecuted
			}
			return ""
		}(),
		UndoDryRunRequired: record.Status == domain.MergeRecordStatusCompleted, IdempotentReplay: replay}, nil
}

func mergePlanPreviewDTO(plan *customerMergePlan) *dto.CustomerMergePreviewResult {
	entities := make([]dto.MergePlannedEntity, len(plan.Moved))
	for i := range plan.Moved {
		entities[i] = dto.MergePlannedEntity{EntityType: plan.Moved[i].EntityType,
			EntityID: plan.Moved[i].EntityID, MutationKind: plan.Moved[i].MutationKind,
			FromProfileID: plan.Moved[i].FromProfileID, ToProfileID: plan.Moved[i].ToProfileID}
	}
	result := &dto.CustomerMergePreviewResult{PreviewToken: plan.Token, PreviewHash: plan.Hash, GeneratedAt: plan.GeneratedAt,
		SourceProfileID: plan.Source.ID, TargetProfileID: plan.Target.ID, SourceStatus: normalizedProfileStatus(plan.Source.Status),
		TargetStatus: normalizedProfileStatus(plan.Target.Status), SourceRowVersion: plan.Source.RowVersion,
		TargetRowVersion: plan.Target.RowVersion, DependsOnMergeRecordID: plan.Dependency, PlannedEntities: entities,
		FrozenDemandDocumentIDs: append([]uint(nil), plan.FrozenDemand...), Counts: plan.Counts,
		Blockers: append([]dto.MergeBlocker(nil), plan.Blockers...), CanExecute: len(plan.Blockers) == 0,
		PrimaryIdentityOptions:               append([]dto.PrimaryIdentityOption(nil), plan.PrimaryIdentityOptions...),
		RecommendedPrimaryIdentitySelections: append([]dto.PrimaryIdentitySelection(nil), plan.RecommendedPrimaryIdentitySelections...),
		DefaultAddressOptions:                append([]dto.DefaultAddressOption(nil), plan.DefaultAddressOptions...),
		RecommendedDefaultAddressID:          plan.RecommendedDefaultAddressID,
		DisplayNameOptions:                   append([]dto.DisplayNameOption(nil), plan.DisplayNameOptions...),
		RecommendedDisplayNameResolution:     plan.RecommendedDisplayNameResolution}
	if plan.Candidate != nil {
		id := plan.Candidate.ID
		result.CandidateID = &id
		result.CandidateRowVersion = plan.Candidate.RowVersion
		result.EvidenceHash = plan.Candidate.EvidenceHash
		result.PolicyVersion = plan.Candidate.PolicyVersion
		result.PolicyRevisionID = plan.Candidate.MergePolicyRevisionID
	}
	return result
}

func hashMergeCommand(input dto.ExecuteCustomerMergeInput) (string, error) {
	normalized := input
	sort.Slice(normalized.PrimaryIdentitySelections, func(i, j int) bool {
		left, right := normalized.PrimaryIdentitySelections[i], normalized.PrimaryIdentitySelections[j]
		if left.Namespace != right.Namespace {
			return left.Namespace < right.Namespace
		}
		if left.IdentityType != right.IdentityType {
			return left.IdentityType < right.IdentityType
		}
		return left.IdentityID < right.IdentityID
	})
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode merge command: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func mergePayloadFromPlan(plan *customerMergePlan) (string, error) {
	payload := domain.CustomerMergePayload{}
	for _, moved := range plan.Moved {
		if moved.MutationKind != domain.MergeMutationReassign {
			continue
		}
		switch moved.EntityType {
		case domain.MergeEntityIdentity:
			payload.IdentityIDs = append(payload.IdentityIDs, moved.EntityID)
		case domain.MergeEntityAddress:
			payload.AddressIDs = append(payload.AddressIDs, moved.EntityID)
		case domain.MergeEntityDemandDocument:
			payload.DemandDocumentIDs = append(payload.DemandDocumentIDs, moved.EntityID)
		case domain.MergeEntityNameObservation:
			payload.NameObservationIDs = append(payload.NameObservationIDs, moved.EntityID)
		case domain.MergeEntityNameEvent:
			payload.NameEventIDs = append(payload.NameEventIDs, moved.EntityID)
		case domain.MergeEntityOrigin:
			payload.OriginIDs = append(payload.OriginIDs, moved.EntityID)
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode merge payload: %w", err)
	}
	return string(data), nil
}

func profileSnapshotsFromPlan(plan *customerMergePlan) (string, string) {
	var source, target string
	for _, moved := range plan.Moved {
		if moved.EntityType != domain.MergeEntityProfile {
			continue
		}
		if moved.MutationKind == domain.MergeMutationProfileState {
			source = moved.BeforeSnapshot
		}
		if moved.MutationKind == domain.MergeMutationDisplayProjection {
			target = moved.BeforeSnapshot
		}
	}
	return source, target
}

func mergeStateMatches(expected string, actual domain.MergeEntityState) bool {
	data, err := json.Marshal(actual)
	return err == nil && string(data) == expected
}

func maskedEvidenceSnapshot(evidence []domain.MergeEvidence) string {
	type item struct {
		Kind, Polarity, Code, ValueHash, MaskedValue string
		EntityType                                   string
		EntityID                                     uint
	}
	items := make([]item, len(evidence))
	for i := range evidence {
		items[i] = item{Kind: evidence[i].EvidenceKind, Polarity: evidence[i].Polarity,
			Code: evidence[i].ExplanationCode, ValueHash: evidence[i].ValueHash, MaskedValue: evidence[i].MaskedValue,
			EntityType: evidence[i].SourceEntityType, EntityID: evidence[i].SourceEntityID}
	}
	data, _ := json.Marshal(items)
	return string(data)
}

func countsFromMoved(moved []domain.MergeMovedEntity) dto.MergeEntityCounts {
	counts := dto.MergeEntityCounts{}
	for _, item := range moved {
		if item.MutationKind != domain.MergeMutationReassign && item.EntityType != domain.MergeEntityProfile {
			continue
		}
		switch item.EntityType {
		case domain.MergeEntityIdentity:
			counts.Identities++
		case domain.MergeEntityAddress:
			counts.Addresses++
		case domain.MergeEntityDemandDocument:
			counts.DemandDocuments++
		case domain.MergeEntityNameObservation:
			counts.NameObservations++
		case domain.MergeEntityNameEvent:
			counts.NameEvents++
		case domain.MergeEntityOrigin:
			counts.Origins++
		case domain.MergeEntityProfile:
			counts.ProfileMutations++
		}
	}
	return counts
}

func equalOptionalUint(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func mergeBlockerCodes(blockers []dto.MergeBlocker) string {
	codes := make([]string, len(blockers))
	for i := range blockers {
		codes[i] = blockers[i].Code
	}
	return strings.Join(compactMergeCodes(codes), ",")
}

func candidateStatus(candidate *domain.MergeCandidate) string {
	if candidate == nil {
		return ""
	}
	return domain.MergeCandidateStatusExecuted
}
