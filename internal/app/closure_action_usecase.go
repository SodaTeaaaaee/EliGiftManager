package app

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── RecordChannelClosureDecision ──

type recordClosureDecisionUseCase struct {
	decisionRepo domain.ChannelClosureDecisionRepository
	fulfillRepo  domain.FulfillmentLineRepository
	profileRepo  domain.IntegrationProfileRepository
	demandRepo   domain.DemandDocumentRepository
}

func NewRecordClosureDecisionUseCase(
	decisionRepo domain.ChannelClosureDecisionRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	profileRepo domain.IntegrationProfileRepository,
	demandRepo domain.DemandDocumentRepository,
) RecordClosureDecisionUseCase {
	return &recordClosureDecisionUseCase{
		decisionRepo: decisionRepo,
		fulfillRepo:  fulfillRepo,
		profileRepo:  profileRepo,
		demandRepo:   demandRepo,
	}
}

func (uc *recordClosureDecisionUseCase) RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error) {
	if len(input.Entries) == 0 {
		return nil, fmt.Errorf("at least one decision entry is required")
	}

	// Phase 1: validate every entry and build domain records.
	// No side effects — validation errors leave zero traces.
	var domainRecords []*domain.ChannelClosureDecisionRecord
	now := time.Now().Format(time.RFC3339)
	for i, entry := range input.Entries {
		fl, err := uc.fulfillRepo.FindByID(entry.FulfillmentLineID)
		if err != nil {
			return nil, fmt.Errorf("entry %d: fulfillment line %d not found: %w", i, entry.FulfillmentLineID, err)
		}
		if fl.WaveID != input.WaveID {
			return nil, fmt.Errorf("entry %d: fulfillment line %d belongs to wave %d, not wave %d", i, entry.FulfillmentLineID, fl.WaveID, input.WaveID)
		}

		profile, err := uc.profileRepo.FindByID(input.IntegrationProfileID)
		if err != nil {
			return nil, fmt.Errorf("entry %d: profile %d not found: %w", i, input.IntegrationProfileID, err)
		}

		// Verify profile ownership through the full demand chain.
		if fl.DemandDocumentID == nil {
			return nil, fmt.Errorf("entry %d: fulfillment line %d has no DemandDocumentID; cannot establish profile ownership", i, entry.FulfillmentLineID)
		}
		doc, err := uc.demandRepo.FindByID(*fl.DemandDocumentID)
		if err != nil {
			return nil, fmt.Errorf("entry %d: demand document %d not found: %w", i, *fl.DemandDocumentID, err)
		}
		if doc.IntegrationProfileID == nil {
			return nil, fmt.Errorf("entry %d: demand document %d has no IntegrationProfileID; cannot establish profile ownership for fulfillment line %d", i, doc.ID, entry.FulfillmentLineID)
		}
		if *doc.IntegrationProfileID != profile.ID {
			return nil, fmt.Errorf("entry %d: fulfillment line %d belongs to profile %d, not profile %d", i, entry.FulfillmentLineID, *doc.IntegrationProfileID, profile.ID)
		}

		switch entry.DecisionKind {
		case "mark_sync_unsupported", "mark_sync_skipped", "mark_sync_completed_manually":
		default:
			return nil, fmt.Errorf("entry %d: unknown decision_kind %q", i, entry.DecisionKind)
		}

		if entry.DecisionKind == "mark_sync_completed_manually" && !profile.AllowsManualClosure {
			return nil, fmt.Errorf("entry %d: profile %q does not allow manual closure", i, profile.ProfileKey)
		}

		domainRecords = append(domainRecords, &domain.ChannelClosureDecisionRecord{
			WaveID:               input.WaveID,
			IntegrationProfileID: input.IntegrationProfileID,
			FulfillmentLineID:    entry.FulfillmentLineID,
			DecisionKind:         entry.DecisionKind,
			ReasonCode:           entry.ReasonCode,
			Note:                 entry.Note,
			EvidenceRef:          entry.EvidenceRef,
			OperatorID:           entry.OperatorID,
			CreatedAt:            now,
			UpdatedAt:            now,
		})
	}

	// Phase 2: atomically persist all records — all-or-nothing.
	if err := uc.decisionRepo.AtomicCreate(domainRecords); err != nil {
		return nil, fmt.Errorf("persist decision records: %w", err)
	}

	records := make([]dto.ClosureDecisionRecordDTO, len(domainRecords))
	for i, r := range domainRecords {
		records[i] = dto.ClosureDecisionRecordDTO{
			ID:                   r.ID,
			WaveID:               r.WaveID,
			IntegrationProfileID: r.IntegrationProfileID,
			FulfillmentLineID:    r.FulfillmentLineID,
			DecisionKind:         r.DecisionKind,
			ReasonCode:           r.ReasonCode,
			Note:                 r.Note,
			EvidenceRef:          r.EvidenceRef,
			OperatorID:           r.OperatorID,
			CreatedAt:            r.CreatedAt,
		}
	}

	return records, nil
}

// ── RetryChannelSyncJob ──

type retrySyncUseCase struct {
	channelSyncRepo  domain.ChannelSyncRepository
	profileRepo      domain.IntegrationProfileRepository
	executorProvider ExecutorProvider
}

func NewRetrySyncUseCase(
	channelSyncRepo domain.ChannelSyncRepository,
	profileRepo domain.IntegrationProfileRepository,
	executorProvider ExecutorProvider,
) RetrySyncUseCase {
	return &retrySyncUseCase{
		channelSyncRepo:  channelSyncRepo,
		profileRepo:      profileRepo,
		executorProvider: executorProvider,
	}
}

func (uc *retrySyncUseCase) RetryChannelSyncJob(jobID uint) (*dto.ExecuteSyncResult, error) {
	job, err := uc.channelSyncRepo.FindJobByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("find job %d: %w", jobID, err)
	}

	if job.Status != "failed" && job.Status != "partial_success" {
		return nil, fmt.Errorf("job %d has status %q; only failed or partial_success jobs can be retried", jobID, job.Status)
	}

	profile, err := uc.profileRepo.FindByID(job.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("find profile %d: %w", job.IntegrationProfileID, err)
	}

	allItems, err := uc.channelSyncRepo.ListItemsByJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("list items for job %d: %w", jobID, err)
	}

	// Only retry failed items
	var failedItems []domain.ChannelSyncItem
	for _, it := range allItems {
		if it.Status == "failed" {
			failedItems = append(failedItems, it)
		}
	}
	if len(failedItems) == 0 {
		return nil, fmt.Errorf("job %d has no failed items to retry", jobID)
	}

	now := time.Now().Format(time.RFC3339)
	job.Status = "running"
	job.StartedAt = now
	job.UpdatedAt = now
	if err := uc.channelSyncRepo.SaveJob(job); err != nil {
		return nil, fmt.Errorf("save job running state: %w", err)
	}

	executor, err := uc.executorProvider.Resolve(profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(job)
		return nil, fmt.Errorf("resolve executor: %w", err)
	}
	result, err := executor.Execute(job, failedItems, profile)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
		job.FinishedAt = now
		job.UpdatedAt = now
		_ = uc.channelSyncRepo.SaveJob(job)
		return nil, fmt.Errorf("executor failed: %w", err)
	}

	// Update each retried item
	for i, item := range failedItems {
		for _, r := range result.Items {
			if r.ItemID == item.ID {
				failedItems[i].Status = r.Status
				failedItems[i].ErrorMessage = r.ErrorMessage
				failedItems[i].UpdatedAt = now
				if saveErr := uc.channelSyncRepo.SaveItem(&failedItems[i]); saveErr != nil {
					return nil, fmt.Errorf("save item %d: %w", item.ID, saveErr)
				}
				break
			}
		}
	}

	// Recalculate job aggregate status
	allItemsAfter, err := uc.channelSyncRepo.ListItemsByJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("list items after retry: %w", err)
	}

	hasSuccess := false
	hasFailure := false
	for _, it := range allItemsAfter {
		switch it.Status {
		case "success":
			hasSuccess = true
		case "failed":
			hasFailure = true
		}
	}
	if hasSuccess && !hasFailure {
		job.Status = "success"
	} else if hasFailure && !hasSuccess {
		job.Status = "failed"
	} else {
		job.Status = "partial_success"
	}
	job.RequestPayload = result.RequestPayload
	job.ResponsePayload = result.ResponsePayload
	job.ErrorMessage = result.ErrorMessage
	job.FinishedAt = now
	job.UpdatedAt = now
	if err := uc.channelSyncRepo.SaveJob(job); err != nil {
		return nil, fmt.Errorf("save job final state: %w", err)
	}

	var updatedItems []domain.ChannelSyncItem
	for _, it := range allItemsAfter {
		updatedItems = append(updatedItems, it)
	}
	return toExecuteSyncResult(job, updatedItems), nil
}
