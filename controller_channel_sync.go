package main

import (
	"errors"
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
	externalCarrierUC   *app.ExternalCarrierUseCase
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
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	mapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	carrierUC := app.NewCarrierMappingUseCase(carrierMappingRepo, profileRepo)
	carrierUC = app.WithCarrierImportDeps(carrierUC, mapping)
	carrierUC = app.WithCarrierImportEvidence(carrierUC, app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)))
	carrierUC = app.WithExternalCarrierRegistry(carrierUC, app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(gdb)))
	return &ChannelSyncController{
		channelSyncUC:       channelSyncUC,
		channelSyncRepo:     channelSyncRepo,
		closureUC:           app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC, carrierMappingRepo),
		executeSyncUC:       app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, executorProvider, nil),
		recordDecisionUC:    app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo),
		retrySyncUC:         app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, executorProvider),
		profileRepo:         profileRepo,
		fulfillRepo:         fulfillRepo,
		gdb:                 gdb,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc)),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo),
		snapshotSvc:         snapshotSvc,
		carrierMappingUC:    carrierUC,
		executorRegistry:    registry,
		externalCarrierUC:   app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(gdb)),
	}
}

func (c *ChannelSyncController) RegisterExternalCarrier(input dto.RegisterExternalCarrierInput) (dto.ExternalCarrierDTO, error) {
	result, err := c.externalCarrierUC.RegisterExternalCarrier(appContext, input)
	if err != nil {
		return dto.ExternalCarrierDTO{}, err
	}
	return *result, nil
}

func (c *ChannelSyncController) BindInternalCarrier(input dto.BindInternalCarrierInput) (dto.ExternalCarrierDTO, error) {
	result, err := c.externalCarrierUC.BindInternalCarrier(appContext, input)
	if err != nil {
		return dto.ExternalCarrierDTO{}, err
	}
	return *result, nil
}

func (c *ChannelSyncController) ListExternalCarriers(profileID uint) ([]dto.ExternalCarrierDTO, error) {
	return c.externalCarrierUC.ListByProfile(appContext, profileID)
}

// CreateChannelSyncJob creates a channel sync job with its items.
func (c *ChannelSyncController) CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (dto.ChannelSyncJobDTO, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.ChannelSyncJobDTO{}, err
	}

	var job *domain.ChannelSyncJob
	var items []domain.ChannelSyncItem
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		channelSyncRepo := repos.ChannelSync
		shipmentRepo := repos.ShipmentRepo
		supplierRepo := repos.SupplierRepo
		fulfillRepo := repos.FulfillRepo
		decisionRepo := repos.ClosureDecision
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyPinRepo := repos.HistoryPin
		historyCheckpointRepo := repos.HistoryCheckpoint

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		createdJob, createdItems, createErr := channelSyncUC.CreateChannelSyncJob(ctx, input)
		if createErr != nil {
			return createErr
		}
		job = createdJob
		items = createdItems
		projectChannelSyncPendingWithRepo(fulfillRepo, items)

		projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdCreateChannelSyncJob,
			CommandSummary:          fmt.Sprintf("create channel sync job %d for wave %d", job.ID, input.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
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
	ctx := appContext
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(items))
	for _, it := range items {
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:               it.FulfillmentLineID,
			ChannelSyncState: "pending",
		})
	}
	if len(updates) > 0 {
		_ = repo.BulkUpdateStates(ctx, updates)
	}
}

// PlanChannelClosure is the high-level orchestration entry point.
func (c *ChannelSyncController) PlanChannelClosure(input dto.PlanChannelClosureInput) (dto.PlanChannelClosureResult, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.PlanChannelClosureResult{}, err
	}

	var result *dto.PlanChannelClosureResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		channelSyncRepo := repos.ChannelSync
		shipmentRepo := repos.ShipmentRepo
		supplierRepo := repos.SupplierRepo
		fulfillRepo := repos.FulfillRepo
		demandRepo := repos.DemandRepo
		profileRepo := repos.Profile
		decisionRepo := repos.ClosureDecision
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyPinRepo := repos.HistoryPin
		historyCheckpointRepo := repos.HistoryCheckpoint

		carrierMappingRepo := repos.Mapping
		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		channelSyncUC := app.NewChannelSyncUseCase(channelSyncRepo, shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		closureUC := app.NewChannelClosureUseCase(profileRepo, shipmentRepo, fulfillRepo, demandRepo, channelSyncUC, carrierMappingRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		planned, planErr := closureUC.PlanChannelClosure(ctx, input)
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
			projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
			if hashErr != nil {
				return hashErr
			}
			_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
				WaveID:                  input.WaveID,
				CommandKind:             domain.CmdCreateChannelSyncJob,
				CommandSummary:          fmt.Sprintf("create channel sync job %d for wave %d", result.Job.ID, input.WaveID),
				PatchPayload:            "",
				InversePatchPayload:     "",
				BaselineSnapshotPayload: preSnapshot,
				ProjectionHash:          projHash,
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
	ctx := appContext
	job, err := c.channelSyncRepo.FindJobByID(ctx, jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, job.WaveID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}

	var result *dto.ExecuteSyncResult
	var executionErr error
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		channelSyncRepo := repos.ChannelSync
		profileRepo := repos.Profile
		fulfillRepo := repos.FulfillRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		decisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		executeSyncUC := app.NewExecuteSyncUseCase(channelSyncRepo, profileRepo, buildExecutorProvider(), nil)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		executed, execErr := executeSyncUC.ExecuteChannelSyncJob(ctx, jobID)
		if execErr != nil {
			// The use case has persisted a recoverable failed job/item state. Commit
			// that state, then return the execution error after the transaction.
			executionErr = execErr
			return nil
		}
		if projErr := projectChannelSyncStatesWithRepo(channelSyncRepo, fulfillRepo, jobID); projErr != nil {
			return projErr
		}
		result = executed
		projHash, hashErr := projHashSvc.ComputeHash(ctx, job.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  job.WaveID,
			CommandKind:             domain.CmdExecuteChannelSyncJob,
			CommandSummary:          fmt.Sprintf("execute channel sync job %d for wave %d (%s)", jobID, job.WaveID, result.JobStatus),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	if executionErr != nil {
		return dto.ExecuteSyncResult{}, executionErr
	}
	return *result, nil
}

// projectChannelSyncStates updates FulfillmentLine.ChannelSyncState based on
// the current state of all items in the given job.
func (c *ChannelSyncController) projectChannelSyncStates(jobID uint) {
	projectChannelSyncStatesWithRepo(c.channelSyncRepo, c.fulfillRepo, jobID)
}

func projectChannelSyncStatesWithRepo(channelSyncRepo domain.ChannelSyncRepository, fulfillRepo domain.FulfillmentLineRepository, jobID uint) error {
	ctx := appContext
	items, err := channelSyncRepo.ListItemsByJob(ctx, jobID)
	if err != nil {
		return err
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
		return fulfillRepo.BulkUpdateStates(ctx, updates)
	}
	return nil
}

// RecordChannelClosureDecision persists manual closure decisions and projects
// channel sync state onto the affected fulfillment lines.
func (c *ChannelSyncController) RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return nil, err
	}

	var records []dto.ClosureDecisionRecordDTO
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		decisionRepo := repos.ClosureDecision
		fulfillRepo := repos.FulfillRepo
		profileRepo := repos.Profile
		demandRepo := repos.DemandRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		recordDecisionUC := app.NewRecordClosureDecisionUseCase(decisionRepo, fulfillRepo, profileRepo, demandRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		recorded, recordErr := recordDecisionUC.RecordChannelClosureDecision(ctx, input)
		if recordErr != nil {
			return recordErr
		}
		records = recorded
		if projErr := projectManualClosureStatesWithRepo(fulfillRepo, input.Entries); projErr != nil {
			return projErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, historyErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdRecordClosureDecision,
			CommandSummary:          fmt.Sprintf("record %d closure decisions for wave %d", len(records), input.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
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
	"mark_sync_unsupported":        "unsupported",
	"mark_sync_skipped":            "skipped",
	"mark_sync_completed_manually": "manual_confirmed",
}

func (c *ChannelSyncController) projectManualClosureStates(entries []dto.RecordClosureDecisionEntry) {
	projectManualClosureStatesWithRepo(c.fulfillRepo, entries)
}

func projectManualClosureStatesWithRepo(repo domain.FulfillmentLineRepository, entries []dto.RecordClosureDecisionEntry) error {
	ctx := appContext
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
		return repo.BulkUpdateStates(ctx, updates)
	}
	return nil
}

// RetryChannelSyncJob retries failed items in a ChannelSyncJob.
func (c *ChannelSyncController) RetryChannelSyncJob(jobID uint) (dto.ExecuteSyncResult, error) {
	ctx := appContext
	job, err := c.channelSyncRepo.FindJobByID(ctx, jobID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, job.WaveID)
	if err != nil {
		return dto.ExecuteSyncResult{}, err
	}

	var result *dto.ExecuteSyncResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		channelSyncRepo := repos.ChannelSync
		profileRepo := repos.Profile
		fulfillRepo := repos.FulfillRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		decisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyCheckpointRepo := repos.HistoryCheckpoint

		retrySyncUC := app.NewRetrySyncUseCase(channelSyncRepo, profileRepo, buildExecutorProvider())
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, decisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, decisionRepo)

		retried, retryErr := retrySyncUC.RetryChannelSyncJob(ctx, jobID)
		if retryErr != nil {
			return retryErr
		}
		if projErr := projectChannelSyncStatesWithRepo(channelSyncRepo, fulfillRepo, jobID); projErr != nil {
			return projErr
		}
		result = retried
		projHash, hashErr := projHashSvc.ComputeHash(ctx, job.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  job.WaveID,
			CommandKind:             domain.CmdRetryChannelSyncJob,
			CommandSummary:          fmt.Sprintf("retry channel sync job %d for wave %d (%s)", jobID, job.WaveID, result.JobStatus),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
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
	ctx := appContext
	profiles, err := c.profileRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.IntegrationProfileSummaryDTO, len(profiles))
	for i, p := range profiles {
		result[i] = dto.IntegrationProfileSummaryDTO{
			ID:                             p.ID,
			ProfileKey:                     p.ProfileKey,
			SourceChannel:                  p.SourceChannel,
			SourceSurface:                  p.SourceSurface,
			TrackingSyncMode:               p.TrackingSyncMode,
			ClosurePolicy:                  p.ClosurePolicy,
			AllowsManualClosure:            p.AllowsManualClosure,
			SupportsExportSupplierOrder:    p.SupportsExportSupplierOrder,
			SupportsImportProductCatalog:   p.SupportsImportProductCatalog,
			SupportsImportSupplierShipment: p.SupportsImportSupplierShipment,
		}
	}
	return result, nil
}

// ListChannelSyncJobsByWave lists all channel sync jobs for a given wave.
func (c *ChannelSyncController) ListChannelSyncJobsByWave(waveID uint) ([]dto.ChannelSyncJobDTO, error) {
	ctx := appContext
	jobs, err := c.channelSyncRepo.ListJobsByWave(ctx, waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ChannelSyncJobDTO, len(jobs))
	for i, j := range jobs {
		jobDTO := domainToChannelSyncJobDTO(&j)
		items, err := c.channelSyncRepo.ListItemsByJob(ctx, j.ID)
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
	templates := buildTrackingTemplateSource()
	registry := app.NewExecutorRegistry()
	registry.Register(app.NewDocumentExportExecutor(exportsDir, templates))
	registry.Register(app.NewCSVExportExecutor(exportsDir, templates))
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
	templates := buildTrackingTemplateSource()
	docExportExec := app.NewDocumentExportExecutor(exportsDir, templates)
	csvExportExec := app.NewCSVExportExecutor(exportsDir, templates)
	registry := map[string]map[string]app.ChannelSyncExecutor{
		"document_export": {
			"eli.local_export": docExportExec,
			"eli.csv_export":   csvExportExec,
		},
	}
	return app.NewRuntimeExecutorProviderWith(registry)
}

// buildTrackingTemplateSource wires profile→template resolution so tracking
// executors can honour export_source_tracking_update MappingRules columnOrder,
// and attaches CarrierUC for internal→external carrier translation on export.
func buildTrackingTemplateSource() *app.TrackingTemplateSource {
	gdb := db.GetDB()
	if gdb == nil {
		return nil
	}
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	carrierRepo := infra.NewCarrierMappingRepository(gdb)
	return &app.TrackingTemplateSource{
		BindingRepo:  infra.NewProfileTemplateBindingRepository(gdb),
		TemplateRepo: infra.NewDocumentTemplateRepository(gdb),
		CarrierUC:    app.NewCarrierMappingUseCase(carrierRepo, profileRepo),
	}
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
	ctx := appContext
	result, err := c.carrierMappingUC.CreateMapping(ctx, input)
	if err != nil {
		return dto.CarrierMappingDTO{}, err
	}
	return *result, nil
}

// ListCarrierMappings returns all carrier mappings for the given integration profile.
func (c *ChannelSyncController) ListCarrierMappings(profileID uint) ([]dto.CarrierMappingDTO, error) {
	ctx := appContext
	return c.carrierMappingUC.ListMappingsByProfile(ctx, profileID)
}

// DeleteCarrierMapping removes a carrier mapping by ID.
func (c *ChannelSyncController) DeleteCarrierMapping(id uint) error {
	ctx := appContext
	return c.carrierMappingUC.DeleteMapping(ctx, id)
}

// ImportCarrierMappings upserts carrier mappings from a template-mapped sheet.
func (c *ChannelSyncController) ImportCarrierMappings(input dto.ImportCarrierMappingsInput) (dto.ImportCarrierMappingsResult, error) {
	ctx := appContext
	mappingRepo := infra.NewCarrierMappingRepository(c.gdb)
	profileRepo := infra.NewIntegrationProfileRepository(c.gdb)
	templateRepo := infra.NewDocumentTemplateRepository(c.gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(c.gdb)
	mappingSvc := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	evidence := app.NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(c.gdb))
	registry := app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(c.gdb))
	preflightUC := app.NewCarrierMappingUseCase(mappingRepo, profileRepo)
	preflightUC = app.WithCarrierImportDeps(preflightUC, mappingSvc)
	preflightUC = app.WithCarrierImportEvidence(preflightUC, evidence)
	preflightUC = app.WithExternalCarrierRegistry(preflightUC, registry)
	preflightUC = app.WithCarrierConflictAudit(preflightUC, registry)
	plan, err := preflightUC.PreflightCarrierMappings(ctx, input)
	if err != nil {
		return dto.ImportCarrierMappingsResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if plan.RejectsBusinessWrites() {
		result, executeErr := preflightUC.ExecuteCarrierImportPlan(ctx, plan)
		if executeErr != nil {
			return dto.ImportCarrierMappingsResult{}, errors.Join(executeErr, evidence.FinalizeFailure(ctx, "failed", executeErr))
		}
		if finalizeErr := evidence.FinalizePending(ctx); finalizeErr != nil {
			return dto.ImportCarrierMappingsResult{}, finalizeErr
		}
		return *result, nil
	}
	var result *dto.ImportCarrierMappingsResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		txUC := app.NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		txUC = app.WithCarrierImportEvidence(txUC, evidence)
		txUC = app.WithExternalCarrierRegistry(txUC, app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		imported, importErr := txUC.ExecuteCarrierImportPlan(ctx, plan)
		if importErr == nil {
			result = imported
		}
		return importErr
	})
	if err != nil {
		return dto.ImportCarrierMappingsResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportCarrierMappingsResult{}, err
	}
	return *result, nil
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
