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
	channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo)
	executorProvider := buildExecutorProvider()
	return &ChannelSyncController{
		channelSyncUC:    channelSyncUC,
		channelSyncRepo:  channelSyncRepo,
		closureUC:        app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC),
		executeSyncUC:    app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		recordDecisionUC: app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo),
		retrySyncUC:      app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, executorProvider),
	}
}

// CreateChannelSyncJob creates a channel sync job with its items.
func (c *ChannelSyncController) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (dto.ChannelSyncJobDTO, error) {
	job, items, err := c.channelSyncUC.CreateChannelSyncJob(input)
	if err != nil {
		return dto.ChannelSyncJobDTO{}, err
	}
	result := domainToChannelSyncJobDTO(job)
	result.Items = make([]dto.ChannelSyncItemDTO, len(items))
	for i, it := range items {
		result.Items[i] = domainToChannelSyncItemDTO(&it)
	}
	return result, nil
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
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	return *result, nil
}

// RecordChannelClosureDecision persists manual closure decisions.
func (c *ChannelSyncController) RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error) {
	return c.recordDecisionUC.RecordChannelClosureDecision(input)
}

// RetryChannelSyncJob retries failed items in a ChannelSyncJob.
func (c *ChannelSyncController) RetryChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	result, err := c.retrySyncUC.RetryChannelSyncJob(jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	return *result, nil
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
