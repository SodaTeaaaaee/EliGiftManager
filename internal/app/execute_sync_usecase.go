package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type executeSyncUseCase struct {
	channelSyncRepo  domain.ChannelSyncRepository
	profileRepo      domain.IntegrationProfileRepository
	executorProvider ExecutorProvider
}

func NewExecuteSyncUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	profileRepo domain.IntegrationProfileRepository,
	executorProvider ExecutorProvider,
) ExecuteSyncUseCase {
	return &executeSyncUseCase{
		channelSyncRepo:  channelSyncRepo,
		profileRepo:      profileRepo,
		executorProvider: executorProvider,
	}
}

func (uc *executeSyncUseCase) ExecuteChannelSyncJob(jobID uint) (*dto.ExecuteSyncResult, error) {
	job, err := uc.channelSyncRepo.FindJobByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("find job %d: %w", jobID, err)
	}

	if job.Status != "pending" {
		return nil, fmt.Errorf("job %d has status %q; only pending jobs can be executed", jobID, job.Status)
	}

	profile, err := uc.profileRepo.FindByID(job.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("find profile %d for job %d: %w", job.IntegrationProfileID, jobID, err)
	}

	items, err := uc.channelSyncRepo.ListItemsByJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("list items for job %d: %w", jobID, err)
	}

	// Mark job as running
	now := time.Now().Format(time.RFC3339)
	job.Status = "running"
	job.StartedAt = now
	job.UpdatedAt = now

	// Build minimal request payload
	reqPayload := map[string]interface{}{
		"job_id":              job.ID,
		"direction":           job.Direction,
		"tracking_sync_mode":  profile.TrackingSyncMode,
		"connector_key":       profile.ConnectorKey,
		"item_count":          len(items),
	}
	payloadBytes, _ := json.Marshal(reqPayload)
	job.RequestPayload = string(payloadBytes)
	if err := uc.channelSyncRepo.SaveJob(job); err != nil {
		return nil, fmt.Errorf("save job running state: %w", err)
	}

	// Resolve executor at execution time (not construction time)
	// so that each call can use a different executor per profile/connector.
	executor, err := uc.executorProvider.Resolve(profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(job)
		for i := range items {
			items[i].Status = "failed"
			items[i].ErrorMessage = err.Error()
			items[i].UpdatedAt = now
			_ = uc.channelSyncRepo.SaveItem(&items[i])
		}
		return nil, fmt.Errorf("resolve executor: %w", err)
	}

	// Execute — if executor fails, persist job as failed before returning the error.
	result, err := executor.Execute(job, items, profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(job)
		for i := range items {
			items[i].Status = "failed"
			items[i].ErrorMessage = err.Error()
			items[i].UpdatedAt = now
			_ = uc.channelSyncRepo.SaveItem(&items[i])
		}
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
				if saveErr := uc.channelSyncRepo.SaveItem(&item); saveErr != nil {
					return nil, fmt.Errorf("save item %d: %w", item.ID, saveErr)
				}
				updatedItems = append(updatedItems, item)
				break
			}
		}
	}

	// Update job aggregate
	job.Status = result.AggregateStatus
	job.ResponsePayload = result.ResponsePayload
	job.ErrorMessage = result.ErrorMessage
	job.FinishedAt = now
	job.UpdatedAt = now
	if err := uc.channelSyncRepo.SaveJob(job); err != nil {
		return nil, fmt.Errorf("save job final state: %w", err)
	}

	return toExecuteSyncResult(job, updatedItems), nil
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
