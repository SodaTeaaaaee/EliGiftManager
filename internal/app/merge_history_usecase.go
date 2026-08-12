package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type CustomerMergeHistoryUseCase interface {
	ListMergeHistory(ctx context.Context, query dto.CustomerMergeHistoryQuery) (*dto.CustomerMergeHistoryPage, error)
	GetMergeHistory(ctx context.Context, mergeID uint) (*dto.CustomerMergeHistoryDetail, error)
}

type customerMergeHistoryUseCase struct{ store domain.MergeExecutionStore }

func NewCustomerMergeHistoryUseCase(store domain.MergeExecutionStore) CustomerMergeHistoryUseCase {
	return &customerMergeHistoryUseCase{store: store}
}

func (u *customerMergeHistoryUseCase) ListMergeHistory(ctx context.Context, query dto.CustomerMergeHistoryQuery) (*dto.CustomerMergeHistoryPage, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 99 {
		limit = 99
	}
	records, err := u.store.ListMergeRecords(ctx, domain.MergeHistoryFilter{ProfileID: query.ProfileID,
		CandidateID: query.CandidateID, Status: query.Status, BeforeCreatedAt: query.BeforeCreatedAt,
		BeforeID: query.BeforeID, Limit: limit + 1})
	if err != nil {
		return nil, fmt.Errorf("list merge history: %w", err)
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	page := &dto.CustomerMergeHistoryPage{Items: make([]dto.CustomerMergeHistoryItem, len(records))}
	for i := range records {
		item, err := u.historyItem(ctx, &records[i])
		if err != nil {
			return nil, err
		}
		page.Items[i] = item
	}
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		page.NextCreatedAt = &last.CreatedAt
		page.NextID = last.ID
	}
	return page, nil
}

func (u *customerMergeHistoryUseCase) GetMergeHistory(ctx context.Context, mergeID uint) (*dto.CustomerMergeHistoryDetail, error) {
	if mergeID == 0 {
		return nil, errors.New("merge ID is required")
	}
	record, err := u.store.FindMergeRecord(ctx, mergeID)
	if err != nil {
		return nil, fmt.Errorf("merge record %d not found: %w", mergeID, err)
	}
	item, err := u.historyItem(ctx, record)
	if err != nil {
		return nil, err
	}
	detail := &dto.CustomerMergeHistoryDetail{CustomerMergeHistoryItem: item, EvidenceSnapshot: record.EvidenceSnapshot}
	moved, movedErr := u.store.ListMovedEntities(ctx, record.ID)
	if movedErr != nil && !errors.Is(movedErr, gorm.ErrRecordNotFound) {
		return nil, movedErr
	}
	for _, row := range moved {
		detail.PlannedEntities = append(detail.PlannedEntities, dto.MergePlannedEntity{
			EntityType: row.EntityType, EntityID: row.EntityID, MutationKind: row.MutationKind,
			FromProfileID: row.FromProfileID, ToProfileID: row.ToProfileID})
	}
	events, err := u.store.ListMergeOperationEvents(ctx, record.ID)
	if err != nil {
		return nil, fmt.Errorf("list merge operation events: %w", err)
	}
	for _, event := range events {
		detail.Events = append(detail.Events, dto.MergeOperationEventDTO{EventType: event.EventType,
			Status: event.Status, ActorRef: event.ActorRef, ReasonCode: event.ReasonCode, CreatedAt: event.CreatedAt})
	}
	return detail, nil
}

func (u *customerMergeHistoryUseCase) historyItem(ctx context.Context, record *domain.CustomerMergeRecord) (dto.CustomerMergeHistoryItem, error) {
	auditLevel := domain.MergeAuditLevelExact
	var counts dto.MergeEntityCounts
	if record.MovePlanHash == "" {
		auditLevel = domain.MergeAuditLevelLegacy
		counts = legacyMergeCounts(record.Payload)
	} else {
		moved, err := u.store.ListMovedEntities(ctx, record.ID)
		if err != nil {
			return dto.CustomerMergeHistoryItem{}, fmt.Errorf("load moved ledger for merge %d: %w", record.ID, err)
		}
		counts = countsFromMoved(moved)
	}
	sourceName := profileSnapshotDisplayName(record.SourceProfileSnapshot)
	targetName := profileSnapshotDisplayName(record.TargetProfileSnapshot)
	return dto.CustomerMergeHistoryItem{MergeID: record.ID, SourceProfileID: record.SourceProfileID,
		TargetProfileID: record.TargetProfileID, SourceDisplayName: sourceName, TargetDisplayName: targetName,
		Status: record.Status, MergeMode: record.MergeMode, ActorRef: record.ActorRef, DecisionReason: record.DecisionReason,
		CandidateID: record.MergeCandidateID, PolicyRevisionID: record.MergePolicyRevisionID, Counts: counts,
		AuditLevel: auditLevel, CreatedAt: record.CreatedAt, UndoneAt: record.UndoneAt,
		CanRequestUndoDryRun: record.Status == domain.MergeRecordStatusCompleted && record.UndoneAt == nil}, nil
}

func profileSnapshotDisplayName(raw string) string {
	var state domain.MergeEntityState
	if json.Unmarshal([]byte(raw), &state) == nil {
		return state.DisplayName
	}
	return ""
}
