package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// ChannelSyncController exposes channel-sync Wails bindings.
type ChannelSyncController struct {
	channelSyncUC    app.ChannelSyncUseCase
	channelSyncRepo  domain.ChannelSyncRepository
	closureUC        app.ChannelClosureUseCase
	executeSyncUC    app.ExecuteSyncUseCase
	recordDecisionUC app.RecordClosureDecisionUseCase
	retrySyncUC      app.RetrySyncUseCase
	profileRepo      domain.IntegrationProfileRepository
	fulfillRepo      domain.FulfillmentLineRepository
}

func NewChannelSyncController() *ChannelSyncController {
	gdb := db.GetDB()
	channelSyncRepo := infra.NewChannelSyncRepository(gdb)
	shipmentRepo := infra.NewShipmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	demandRepo := infra.NewDemandRepository(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	decisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyPinRepo := infra.NewHistoryPinRepository(gdb)

	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)

	channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
	executorProvider := buildExecutorProvider()
	return &ChannelSyncController{
		channelSyncUC:    channelSyncUC,
		channelSyncRepo:  channelSyncRepo,
		closureUC:        app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC),
		executeSyncUC:    app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		recordDecisionUC: app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo),
		retrySyncUC:      app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		profileRepo:      profileRepo,
		fulfillRepo:      fulfillRepo,
	}
}

// CreateChannelSyncJob creates a channel sync job with its items.
func (c *ChannelSyncController) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (dto.ChannelSyncJobDTO, error) {
	job, items, err := c.channelSyncUC.CreateChannelSyncJob(input)
	if err != nil {
		return dto.ChannelSyncJobDTO{}, err
	}
	// Project channel_sync_state → pending for all candidate fulfillment lines
	c.projectChannelSyncPending(items)
	result := domainToChannelSyncJobDTO(job)
	result.Items = make([]dto.ChannelSyncItemDTO, len(items))
	for i, it := range items {
		result.Items[i] = domainToChannelSyncItemDTO(&it)
	}
	return result, nil
}

func (c *ChannelSyncController) projectChannelSyncPending(items []domain.ChannelSyncItem) {
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(items))
	for _, it := range items {
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               it.FulfillmentLineID,
			ChannelSyncState: "pending",
		})
	}
	if len(updates) > 0 {
		_ = c.fulfillRepo.BulkUpdateStates(updates)
	}
}

// PlanChannelClosure is the high-level orchestration entry point.
func (c *ChannelSyncController) PlanChannelClosure(input dto.PlanChannelClosureInput) (dto.PlanChannelClosureResult, error) {
	result, err := c.closureUC.PlanChannelClosure(input)
	if err != nil {
		return dto.PlanChannelClosureResult{}, err
	}
	return *result, nil
}

// ExecuteChannelSyncJob executes a pending ChannelSyncJob.
func (c *ChannelSyncController) ExecuteChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	result, err := c.executeSyncUC.ExecuteChannelSyncJob(jobID)
	// Always project — success or failure, the items have been persisted with their
	// final statuses by the use case before returning.
	c.projectChannelSyncStates(jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	return *result, nil
}

// projectChannelSyncStates updates FulfillmentLine.ChannelSyncState based on
// the current state of all items in the given job.
func (c *ChannelSyncController) projectChannelSyncStates(jobID uint) {
	items, err := c.channelSyncRepo.ListItemsByJob(jobID)
	if err != nil {
		return
	}
	var updates []domain.FulfillmentLineStateUpdate
	for _, it := range items {
		var csState string
		switch it.Status {
		case "success":
			csState = "synced"
		case "failed":
			csState = "failed"
		default:
			continue
		}
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               it.FulfillmentLineID,
			ChannelSyncState: csState,
		})
	}
	if len(updates) > 0 {
		_ = c.fulfillRepo.BulkUpdateStates(updates)
	}
}

// RecordChannelClosureDecision persists manual closure decisions and projects
// channel sync state onto the affected fulfillment lines.
func (c *ChannelSyncController) RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error) {
	records, err := c.recordDecisionUC.RecordChannelClosureDecision(input)
	if err != nil {
		return nil, err
	}
	c.projectManualClosureStates(input.Entries)
	return records, nil
}

// decisionKindToChannelSyncState maps manual closure decision kinds to FulfillmentLine.ChannelSyncState.
var decisionKindToChannelSyncState = map[string]string{
	"mark_sync_unsupported":       "unsupported",
	"mark_sync_skipped":           "skipped",
	"mark_sync_completed_manually": "manual_confirmed",
}

func (c *ChannelSyncController) projectManualClosureStates(entries []dto.RecordClosureDecisionEntry) {
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(entries))
	for _, e := range entries {
		csState, ok := decisionKindToChannelSyncState[e.DecisionKind]
		if !ok {
			continue
		}
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               e.FulfillmentLineID,
			ChannelSyncState: csState,
		})
	}
	if len(updates) > 0 {
		_ = c.fulfillRepo.BulkUpdateStates(updates)
	}
}

// RetryChannelSyncJob retries failed items in a ChannelSyncJob.
func (c *ChannelSyncController) RetryChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	result, err := c.retrySyncUC.RetryChannelSyncJob(jobID)
	// Always project — success or failure, items have been persisted by the use case.
	c.projectChannelSyncStates(jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	return *result, nil
}

// ListIntegrationProfiles returns all integration profiles.
func (c *ChannelSyncController) ListIntegrationProfiles() ([]dto.IntegrationProfileSummaryDTO, error) {
	profiles, err := c.profileRepo.List()
	if err != nil {
		return nil, err
	}
	result := make([]dto.IntegrationProfileSummaryDTO, len(profiles))
	for i, p := range profiles {
		result[i] = dto.IntegrationProfileSummaryDTO{
			ID:                  p.ID,
			ProfileKey:          p.ProfileKey,
			SourceChannel:       p.SourceChannel,
			TrackingSyncMode:    p.TrackingSyncMode,
			ClosurePolicy:       p.ClosurePolicy,
			AllowsManualClosure: p.AllowsManualClosure,
		}
	}
	return result, nil
}

// ListChannelSyncJobsByWave lists all channel sync jobs for a given wave.
func (c *ChannelSyncController) ListChannelSyncJobsByWave(waveID uint) ([]dto.ChannelSyncJobDTO, error) {
	jobs, err := c.channelSyncRepo.ListJobsByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ChannelSyncJobDTO, len(jobs))
	for i, j := range jobs {
		jobDTO := domainToChannelSyncJobDTO(&j)
		items, err := c.channelSyncRepo.ListItemsByJob(j.ID)
		if err != nil {
			return nil, err
		}
		jobDTO.Items = make([]dto.ChannelSyncItemDTO, len(items))
		for k, it := range items {
			jobDTO.Items[k] = domainToChannelSyncItemDTO(&it)
		}
		result[i] = jobDTO
	}
	return result, nil
}

func domainToChannelSyncJobDTO(j *domain.ChannelSyncJob) dto.ChannelSyncJobDTO {
	if j == nil {
		return dto.ChannelSyncJobDTO{}
	}
	return dto.ChannelSyncJobDTO{
		ID:                   j.ID,
		WaveID:               j.WaveID,
		IntegrationProfileID: j.IntegrationProfileID,
		Direction:            j.Direction,
		Status:               j.Status,
		BasisHistoryNodeID:   j.BasisHistoryNodeID,
		BasisProjectionHash:  j.BasisProjectionHash,
		BasisPayloadSnapshot: j.BasisPayloadSnapshot,
		RequestPayload:       j.RequestPayload,
		ResponsePayload:      j.ResponsePayload,
		ErrorMessage:         j.ErrorMessage,
		StartedAt:            j.StartedAt,
		FinishedAt:           j.FinishedAt,
		CreatedAt:            j.CreatedAt,
		UpdatedAt:            j.UpdatedAt,
	}
}

// buildExecutorProvider resolves the exports directory and wires the
// document_export executor for the "eli.local_export" connector key.
func buildExecutorProvider() app.ExecutorProvider {
	exportsDir, err := service.ResolveExportsDir()
	if err != nil {
		log.Printf("[channel_sync] resolve exports dir: %v — falling back to os.TempDir", err)
		exportsDir = filepath.Join(os.TempDir(), "EliGiftManager", "exports")
	}
	docExportExec := app.NewDocumentExportExecutor(exportsDir)
	registry := map[string]map[string]app.ChannelSyncExecutor{
		"document_export": {
			"eli.local_export": docExportExec,
		},
	}
	return app.NewRuntimeExecutorProviderWith(registry)
}

func domainToChannelSyncItemDTO(it *domain.ChannelSyncItem) dto.ChannelSyncItemDTO {
	if it == nil {
		return dto.ChannelSyncItemDTO{}
	}
	return dto.ChannelSyncItemDTO{
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
