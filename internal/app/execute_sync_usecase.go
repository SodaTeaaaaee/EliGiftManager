package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type executeSyncUseCase struct {
	channelSyncRepo  domain.ChannelSyncRepository
	profileRepo      domain.IntegrationProfileRepository
	executorProvider ExecutorProvider
	fulfillmentRepo  domain.FulfillmentLineRepository
}

func NewExecuteSyncUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	profileRepo domain.IntegrationProfileRepository,
	executorProvider ExecutorProvider,
	fulfillmentRepo domain.FulfillmentLineRepository,
) ExecuteSyncUseCase {
	return &executeSyncUseCase{
		channelSyncRepo:  channelSyncRepo,
		profileRepo:      profileRepo,
		executorProvider: executorProvider,
		fulfillmentRepo:  fulfillmentRepo,
	}
}

func (uc *executeSyncUseCase) ExecuteChannelSyncJob(ctx context.Context, jobID uint) (*dto.ExecuteSyncResult, error) {
	job, err := uc.channelSyncRepo.FindJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("find job %d: %w", jobID, err)
	}

	if job.Status != "pending" {
		return nil, fmt.Errorf("job %d has status %q; only pending jobs can be executed", jobID, job.Status)
	}

	profile, err := uc.profileRepo.FindByID(ctx, job.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("find profile %d for job %d: %w", job.IntegrationProfileID, jobID, err)
	}

	items, err := uc.channelSyncRepo.ListItemsByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("list items for job %d: %w", jobID, err)
	}

	// Mark job as running
	now := time.Now()
	job.Status = "running"
	job.StartedAt = &now
	job.UpdatedAt = now
	if err := uc.channelSyncRepo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save job running state: %w", err)
	}

	// Resolve executor at execution time (not construction time)
	// so that each call can use a different executor per profile/connector.
	executor, err := uc.executorProvider.Resolve(profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = &now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(ctx, job)
		for i := range items {
			items[i].Status = "failed"
			items[i].ErrorMessage = err.Error()
			items[i].UpdatedAt = now
			_ = uc.channelSyncRepo.SaveItem(ctx, &items[i])
		}
		uc.projectSyncStateToFulfillment(ctx, items)
		return nil, fmt.Errorf("resolve executor: %w", err)
	}

	// Execute — if executor fails, persist job as failed before returning the error.
	result, err := executor.Execute(ctx, job, items, profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = &now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(ctx, job)
		for i := range items {
			items[i].Status = "failed"
			items[i].ErrorMessage = err.Error()
			items[i].UpdatedAt = now
			_ = uc.channelSyncRepo.SaveItem(ctx, &items[i])
		}
		uc.projectSyncStateToFulfillment(ctx, items)
		return nil, fmt.Errorf("executor failed: %w", err)
	}

	// Update each item
	var updatedItems []domain.ChannelSyncItem
	for _, item := range items {
		for _, r := range result.Items {
			if r.ItemID == item.ID {
				item.Status = r.Status
				item.ErrorMessage = r.ErrorMessage
				item.UpdatedAt = now
				if saveErr := uc.channelSyncRepo.SaveItem(ctx, &item); saveErr != nil {
					return nil, fmt.Errorf("save item %d: %w", item.ID, saveErr)
				}
				updatedItems = append(updatedItems, item)
				break
			}
		}
	}

	// Project sync state to fulfillment lines
	uc.projectSyncStateToFulfillment(ctx, updatedItems)

	// Update job aggregate
	job.Status = result.AggregateStatus
	job.RequestPayload = result.RequestPayload
	job.ResponsePayload = result.ResponsePayload
	job.ErrorMessage = result.ErrorMessage
	job.FinishedAt = &now
	job.UpdatedAt = now
	if err := uc.channelSyncRepo.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("save job final state: %w", err)
	}

	return toExecuteSyncResult(job, updatedItems), nil
}

// projectSyncStateToFulfillment projects each ChannelSyncItem.Status back into
// the corresponding FulfillmentLine.ChannelSyncState.
func (uc *executeSyncUseCase) projectSyncStateToFulfillment(ctx context.Context, items []domain.ChannelSyncItem) {
	if uc.fulfillmentRepo == nil {
		return
	}
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(items))
	for _, item := range items {
		state := syncItemStatusToFulfillmentState(item.Status)
		if state == "" {
			continue
		}
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               item.FulfillmentLineID,
			ChannelSyncState: state,
		})
	}
	if len(updates) > 0 {
		_ = uc.fulfillmentRepo.BulkUpdateStates(ctx, updates)
	}
}

func syncItemStatusToFulfillmentState(status string) string {
	switch status {
	case "success":
		return "synced"
	case "failed":
		return "failed"
	default:
		return ""
	}
}

func toExecuteSyncResult(job *domain.ChannelSyncJob, items []domain.ChannelSyncItem) *dto.ExecuteSyncResult {
	dtoItems := make([]dto.ChannelSyncItemDTO, len(items))
	for i, it := range items {
		dtoItems[i] = dto.ChannelSyncItemDTO{
			ID:                 it.ID,
			ChannelSyncJobID:   it.ChannelSyncJobID,
			FulfillmentLineID:  it.FulfillmentLineID,
			ShipmentID:         it.ShipmentID,
			ExternalDocumentNo: it.ExternalDocumentNo,
			ExternalLineNo:     it.ExternalLineNo,
			TrackingNo:         it.TrackingNo,
			CarrierCode:        it.CarrierCode,
			Status:             it.Status,
			ErrorMessage:       it.ErrorMessage,
			CreatedAt:          it.CreatedAt,
			UpdatedAt:          it.UpdatedAt,
		}
	}
	return &dto.ExecuteSyncResult{
		JobID:           job.ID,
		JobStatus:       job.Status,
		RequestPayload:  job.RequestPayload,
		ResponsePayload: job.ResponsePayload,
		ErrorMessage:    job.ErrorMessage,
		StartedAt:       job.StartedAt,
		FinishedAt:      job.FinishedAt,
		Items:           dtoItems,
	}
}
