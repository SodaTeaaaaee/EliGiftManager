package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type CustomerSplitHistoryUseCase interface {
	ListSplitHistory(ctx context.Context, query dto.CustomerSplitHistoryQuery) (*dto.CustomerSplitHistoryPage, error)
	GetSplitHistory(ctx context.Context, splitID uint) (*dto.CustomerSplitHistoryDetail, error)
}

type customerSplitHistoryUseCase struct{ store domain.SplitExecutionStore }

func NewCustomerSplitHistoryUseCase(store domain.SplitExecutionStore) CustomerSplitHistoryUseCase {
	return &customerSplitHistoryUseCase{store: store}
}

func (u *customerSplitHistoryUseCase) ListSplitHistory(ctx context.Context, query dto.CustomerSplitHistoryQuery) (*dto.CustomerSplitHistoryPage, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 99 {
		limit = 99
	}
	records, err := u.store.ListSplitRecords(ctx, domain.SplitHistoryFilter{ProfileID: query.ProfileID, Status: query.Status,
		BeforeCreatedAt: query.BeforeCreatedAt, BeforeID: query.BeforeID, Limit: limit + 1})
	if err != nil {
		return nil, fmt.Errorf("list split history: %w", err)
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	page := &dto.CustomerSplitHistoryPage{Items: make([]dto.CustomerSplitHistoryItem, len(records)), HasMore: hasMore}
	for i := range records {
		page.Items[i] = splitHistoryItem(&records[i])
	}
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		page.NextBefore = &last.CreatedAt
		page.NextID = last.ID
	}
	return page, nil
}

func (u *customerSplitHistoryUseCase) GetSplitHistory(ctx context.Context, splitID uint) (*dto.CustomerSplitHistoryDetail, error) {
	record, err := u.store.FindSplitRecord(ctx, splitID)
	if err != nil {
		return nil, fmt.Errorf("find split history %d: %w", splitID, err)
	}
	moved, err := u.store.ListSplitMovedEntities(ctx, splitID)
	if err != nil {
		return nil, fmt.Errorf("list split moved entities: %w", err)
	}
	events, err := u.store.ListSplitOperationEvents(ctx, splitID)
	if err != nil {
		return nil, fmt.Errorf("list split operation events: %w", err)
	}
	detail := &dto.CustomerSplitHistoryDetail{CustomerSplitHistoryItem: splitHistoryItem(record), PlanHash: record.MovePlanHash,
		ReverseGuidance: "direct split undo is not supported; preview and execute an explicit reviewed merge to reverse this split"}
	detail.MovedEntities = make([]dto.CustomerSplitMovedEntityDTO, len(moved))
	for i := range moved {
		detail.MovedEntities[i] = dto.CustomerSplitMovedEntityDTO{EntityType: moved[i].EntityType, EntityID: moved[i].EntityID,
			FromProfileID: moved[i].FromProfileID, ToProfileID: moved[i].ToProfileID, MutationKind: moved[i].MutationKind,
			BeforeSnapshot: moved[i].BeforeSnapshot, AfterSnapshot: moved[i].AfterSnapshot}
	}
	detail.Events = make([]dto.CustomerSplitOperationEventDTO, len(events))
	for i := range events {
		detail.Events[i] = dto.CustomerSplitOperationEventDTO{EventType: events[i].EventType, Status: events[i].Status,
			ActorRef: events[i].ActorRef, ReasonCode: events[i].ReasonCode, CreatedAt: events[i].CreatedAt}
	}
	return detail, nil
}

func splitHistoryItem(record *domain.CustomerSplitRecord) dto.CustomerSplitHistoryItem {
	var payload customerSplitAuditPayload
	_ = json.Unmarshal([]byte(record.Payload), &payload)
	return dto.CustomerSplitHistoryItem{SplitID: record.ID, OperationType: "split", SourceProfileID: record.SourceProfileID,
		TargetProfileID: record.TargetProfileID, TargetStrategy: record.TargetStrategy, Status: record.Status,
		ActorRef: record.ActorRef, DecisionReason: record.DecisionReason, Counts: payload.Counts,
		CreatedAt: record.CreatedAt, CompletedAt: record.CompletedAt, DirectUndoSupported: false,
		ReverseOperationKind: record.ReverseOperationKind}
}
