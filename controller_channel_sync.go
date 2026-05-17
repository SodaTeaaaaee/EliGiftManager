package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

// ChannelSyncController exposes channel-sync Wails bindings.
type ChannelSyncController struct {
	channelSyncUC       app.ChannelSyncUseCase
	channelSyncRepo     domain.ChannelSyncRepository
	closureUC           app.ChannelClosureUseCase
	executeSyncUC       app.ExecuteSyncUseCase
	recordDecisionUC    app.RecordClosureDecisionUseCase
	retrySyncUC         app.RetrySyncUseCase
	profileRepo         domain.IntegrationProfileRepository
	fulfillRepo         domain.FulfillmentLineRepository
	gdb                 *gorm.DB
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
	carrierMappingUC    app.CarrierMappingUseCase
	executorRegistry    *app.ExecutorRegistry
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
	ruleRepo := infra.NewRuleRepository(gdb)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyPinRepo := infra.NewHistoryPinRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)

	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)

	channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
	executorProvider := buildExecutorProvider()
	registry := buildExecutorRegistry()
	carrierMappingRepo := infra.NewCarrierMappingRepository(gdb)
	return &ChannelSyncController{
		channelSyncUC:       channelSyncUC,
		channelSyncRepo:     channelSyncRepo,
		closureUC:           app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC, carrierMappingRepo),
		executeSyncUC:       app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		recordDecisionUC:    app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo),
		retrySyncUC:         app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		profileRepo:         profileRepo,
		fulfillRepo:         fulfillRepo,
		gdb:                 gdb,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo),
		snapshotSvc:         snapshotSvc,
		carrierMappingUC:    app.NewCarrierMappingUseCase(carrierMappingRepo, profileRepo),
		executorRegistry:    registry,
	}
}

// CreateChannelSyncJob creates a channel sync job with its items.
func (c *ChannelSyncController) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (dto.ChannelSyncJobDTO, error) {
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(input.WaveID)
	if err != nil {
		return dto.ChannelSyncJobDTO{}, err
	}

	var job *domain.ChannelSyncJob
	var items []domain.ChannelSyncItem
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		channelSyncRepo := infra.NewChannelSyncRepository(tx)
		shipmentRepo := infra.NewShipmentRepository(tx)
		supplierRepo := infra.NewSupplierOrderRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		decisionRepo := infra.NewClosureDecisionRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyPinRepo := infra.NewHistoryPinRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		createdJob, createdItems, createErr := channelSyncUC.CreateChannelSyncJob(input)
		if createErr != nil {
			return createErr
		}
		job = createdJob
		items = createdItems
		projectChannelSyncPendingWithRepo(fulfillRepo, items)

		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdCreateChannelSyncJob,
			CommandSummary:          fmt.Sprintf("create channel sync job %d for wave %d", job.ID, input.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHashSvc.ComputeHash(input.WaveID),
		})
		return recordErr
	})
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

func (c *ChannelSyncController) projectChannelSyncPending(items []domain.ChannelSyncItem) {
	projectChannelSyncPendingWithRepo(c.fulfillRepo, items)
}

func projectChannelSyncPendingWithRepo(repo domain.FulfillmentLineRepository, items []domain.ChannelSyncItem) {
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(items))
	for _, it := range items {
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               it.FulfillmentLineID,
			ChannelSyncState: "pending",
		})
	}
	if len(updates) > 0 {
		_ = repo.BulkUpdateStates(updates)
	}
}

// PlanChannelClosure is the high-level orchestration entry point.
func (c *ChannelSyncController) PlanChannelClosure(input dto.PlanChannelClosureInput) (dto.PlanChannelClosureResult, error) {
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(input.WaveID)
	if err != nil {
		return dto.PlanChannelClosureResult{}, err
	}

	var result *dto.PlanChannelClosureResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		channelSyncRepo := infra.NewChannelSyncRepository(tx)
		shipmentRepo := infra.NewShipmentRepository(tx)
		supplierRepo := infra.NewSupplierOrderRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		demandRepo := infra.NewDemandRepository(tx)
		profileRepo := infra.NewIntegrationProfileRepository(tx)
		decisionRepo := infra.NewClosureDecisionRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyPinRepo := infra.NewHistoryPinRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		carrierMappingRepo := infra.NewCarrierMappingRepository(tx)
		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		closureUC := app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC, carrierMappingRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		planned, planErr := closureUC.PlanChannelClosure(input)
		if planErr != nil {
			return planErr
		}
		result = planned
		if result.Decision == dto.ClosureDecisionCreateJob && result.Job != nil {
			items := make([]domain.ChannelSyncItem, len(result.Items))
			for i := range result.Items {
				items[i] = domain.ChannelSyncItem{
					FulfillmentLineID: result.Items[i].FulfillmentLineID,
				}
			}
			projectChannelSyncPendingWithRepo(fulfillRepo, items)
			_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
				WaveID:                  input.WaveID,
				CommandKind:             domain.CmdCreateChannelSyncJob,
				CommandSummary:          fmt.Sprintf("create channel sync job %d for wave %d", result.Job.ID, input.WaveID),
				PatchPayload:            "",
				InversePatchPayload:     "",
				BaselineSnapshotPayload: preSnapshot,
				ProjectionHash:          projHashSvc.ComputeHash(input.WaveID),
			})
			return recordErr
		}
		return nil
	})
	if err != nil {
		return dto.PlanChannelClosureResult{}, err
	}
	return *result, nil
}

// ExecuteChannelSyncJob executes a pending ChannelSyncJob.
func (c *ChannelSyncController) ExecuteChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	job, err := c.channelSyncRepo.FindJobByID(jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(job.WaveID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}

	var result *dto.ExecuteSyncResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		channelSyncRepo := infra.NewChannelSyncRepository(tx)
		profileRepo := infra.NewIntegrationProfileRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		decisionRepo := infra.NewClosureDecisionRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		executeSyncUC := app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, buildExecutorProvider())
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		executed, execErr := executeSyncUC.ExecuteChannelSyncJob(jobID)
		projectChannelSyncStatesWithRepo(channelSyncRepo, fulfillRepo, jobID)
		if execErr != nil {
			return execErr
		}
		result = executed
		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  job.WaveID,
			CommandKind:             domain.CmdExecuteChannelSyncJob,
			CommandSummary:          fmt.Sprintf("execute channel sync job %d for wave %d (%s)", jobID, job.WaveID, result.JobStatus),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHashSvc.ComputeHash(job.WaveID),
		})
		return recordErr
	})
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	return *result, nil
}

// projectChannelSyncStates updates FulfillmentLine.ChannelSyncState based on
// the current state of all items in the given job.
func (c *ChannelSyncController) projectChannelSyncStates(jobID uint) {
	projectChannelSyncStatesWithRepo(c.channelSyncRepo, c.fulfillRepo, jobID)
}

func projectChannelSyncStatesWithRepo(channelSyncRepo domain.ChannelSyncRepository, fulfillRepo domain.FulfillmentLineRepository, jobID uint) {
	items, err := channelSyncRepo.ListItemsByJob(jobID)
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
		_ = fulfillRepo.BulkUpdateStates(updates)
	}
}

// RecordChannelClosureDecision persists manual closure decisions and projects
// channel sync state onto the affected fulfillment lines.
func (c *ChannelSyncController) RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error) {
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(input.WaveID)
	if err != nil {
		return nil, err
	}

	var records []dto.ClosureDecisionRecordDTO
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		decisionRepo := infra.NewClosureDecisionRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		profileRepo := infra.NewIntegrationProfileRepository(tx)
		demandRepo := infra.NewDemandRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		recordDecisionUC := app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		recorded, recordErr := recordDecisionUC.RecordChannelClosureDecision(input)
		if recordErr != nil {
			return recordErr
		}
		records = recorded
		projectManualClosureStatesWithRepo(fulfillRepo, input.Entries)

		_, historyErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdRecordClosureDecision,
			CommandSummary:          fmt.Sprintf("record %d closure decisions for wave %d", len(records), input.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHashSvc.ComputeHash(input.WaveID),
		})
		return historyErr
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// decisionKindToChannelSyncState maps manual closure decision kinds to FulfillmentLine.ChannelSyncState.
var decisionKindToChannelSyncState = map[string]string{
	"mark_sync_unsupported":       "unsupported",
	"mark_sync_skipped":           "skipped",
	"mark_sync_completed_manually": "manual_confirmed",
}

func (c *ChannelSyncController) projectManualClosureStates(entries []dto.RecordClosureDecisionEntry) {
	projectManualClosureStatesWithRepo(c.fulfillRepo, entries)
}

func projectManualClosureStatesWithRepo(repo domain.FulfillmentLineRepository, entries []dto.RecordClosureDecisionEntry) {
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
		_ = repo.BulkUpdateStates(updates)
	}
}

// RetryChannelSyncJob retries failed items in a ChannelSyncJob.
func (c *ChannelSyncController) RetryChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	job, err := c.channelSyncRepo.FindJobByID(jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(job.WaveID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}

	var result *dto.ExecuteSyncResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		channelSyncRepo := infra.NewChannelSyncRepository(tx)
		profileRepo := infra.NewIntegrationProfileRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		decisionRepo := infra.NewClosureDecisionRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		retrySyncUC := app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, buildExecutorProvider())
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		retried, retryErr := retrySyncUC.RetryChannelSyncJob(jobID)
		projectChannelSyncStatesWithRepo(channelSyncRepo, fulfillRepo, jobID)
		if retryErr != nil {
			return retryErr
		}
		result = retried
		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  job.WaveID,
			CommandKind:             domain.CmdRetryChannelSyncJob,
			CommandSummary:          fmt.Sprintf("retry channel sync job %d for wave %d (%s)", jobID, job.WaveID, result.JobStatus),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHashSvc.ComputeHash(job.WaveID),
		})
		return recordErr
	})
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

// buildExecutorRegistry constructs an ExecutorRegistry pre-populated with all
// known CapableExecutor implementations.  It is kept in sync with
// buildExecutorProvider so the two registrations never diverge.
func buildExecutorRegistry() *app.ExecutorRegistry {
	exportsDir, err := service.ResolveExportsDir()
	if err != nil {
		log.Printf("[channel_sync] resolve exports dir for registry: %v — falling back to os.TempDir", err)
		exportsDir = filepath.Join(os.TempDir(), "EliGiftManager", "exports")
	}
	registry := app.NewExecutorRegistry()
	registry.Register(app.NewDocumentExportExecutor(exportsDir))
	return registry
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

// ── CarrierMapping methods ──

// CreateCarrierMapping creates a new carrier code mapping for an integration profile.
func (c *ChannelSyncController) CreateCarrierMapping(input dto.CreateCarrierMappingInput) (dto.CarrierMappingDTO, error) {
	result, err := c.carrierMappingUC.CreateMapping(input)
	if err != nil {
		return dto.CarrierMappingDTO{}, err
	}
	return *result, nil
}

// ListCarrierMappings returns all carrier mappings for the given integration profile.
func (c *ChannelSyncController) ListCarrierMappings(profileID uint) ([]dto.CarrierMappingDTO, error) {
	return c.carrierMappingUC.ListMappingsByProfile(profileID)
}

// DeleteCarrierMapping removes a carrier mapping by ID.
func (c *ChannelSyncController) DeleteCarrierMapping(id uint) error {
	return c.carrierMappingUC.DeleteMapping(id)
}

// ListConnectorCapabilities returns capability metadata for all registered connectors.
func (c *ChannelSyncController) ListConnectorCapabilities() (map[string]any, error) {
	caps := c.executorRegistry.ListCapabilities()
	result := make(map[string]any, len(caps))
	for k, v := range caps {
		result[k] = map[string]any{
			"supportsTrackingPush":    v.SupportsTrackingPush,
			"supportsOrderExport":     v.SupportsOrderExport,
			"supportsStatusQuery":     v.SupportsStatusQuery,
			"requiresCarrierMapping":  v.RequiresCarrierMapping,
			"requiresExternalOrderNo": v.RequiresExternalOrderNo,
			"supportedDirections":     v.SupportedDirections,
		}
	}
	return result, nil
}
