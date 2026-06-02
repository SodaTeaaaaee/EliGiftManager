package main

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// WaveController exposes wave-management Wails bindings.
type WaveController struct {
	waveUC              app.WaveUseCase
	demandMappingUC     app.DemandMappingUseCase
	overviewQueryUC     app.WaveOverviewQueryUseCase
	assignmentRepo      domain.WaveDemandAssignmentRepository
	demandRepo          domain.DemandDocumentRepository
	gdb                 *gorm.DB
	nodeRepo            domain.HistoryNodeRepository
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
	historyGraphUC      app.HistoryGraphQueryUseCase
	historyGCSvc        *app.HistoryGCService
	lifecycleSvc        *app.LifecycleProjectionService
	workspaceGuard      *app.WorkspaceGuardService
}

func NewWaveController() *WaveController {
	gormDB := db.GetDB()
	waveRepo := infra.NewWaveRepository(gormDB)
	demandRepo := infra.NewDemandRepository(gormDB)
	ruleRepo := infra.NewRuleRepository(gormDB)
	fulfillRepo := infra.NewFulfillmentRepository(gormDB)
	supplierRepo := infra.NewSupplierOrderRepository(gormDB)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gormDB)
	shipmentRepo := infra.NewShipmentRepository(gormDB)
	channelSyncRepo := infra.NewChannelSyncRepository(gormDB)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gormDB)
	historyScopeRepo := infra.NewHistoryScopeRepository(gormDB)
	historyNodeRepo := infra.NewHistoryNodeRepository(gormDB)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gormDB)

	historyPinRepo := infra.NewHistoryPinRepository(gormDB)

	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gormDB)
	snapshotSvc := app.NewWaveSnapshotService(gormDB, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)

	basisDriftUC := app.NewBasisDriftDetectionUseCase(supplierRepo, shipmentRepo, channelSyncRepo, fulfillRepo)
	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	overviewProjUC := app.NewWaveOverviewProjectionUseCase(channelSyncRepo, closureDecisionRepo, basisDriftUC, historyHeadUC)
	productRepo := infra.NewProductRepository(gormDB)
	profileRepo := infra.NewIntegrationProfileRepository(gormDB)
	addressRepo := infra.NewAddressRepository(gormDB)

	return &WaveController{
		waveUC:              app.NewWaveUseCase(waveRepo, demandRepo, assignmentRepo),
		demandMappingUC:     app.NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil, addressRepo),
		overviewQueryUC:     app.NewWaveOverviewQueryUseCase(waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo, shipmentRepo, productRepo, profileRepo, historyScopeRepo, historyNodeRepo, overviewProjUC, adjustmentRepo),
		assignmentRepo:      assignmentRepo,
		demandRepo:          demandRepo,
		nodeRepo:            historyNodeRepo,
		gdb:                 gormDB,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo),
		snapshotSvc:         snapshotSvc,
		historyGraphUC:      app.NewHistoryGraphQueryUseCase(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, historyPinRepo),
		historyGCSvc:        app.NewHistoryGCService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, historyPinRepo),
		lifecycleSvc:        app.NewLifecycleProjectionService(waveRepo, fulfillRepo, supplierRepo, shipmentRepo, assignmentRepo, channelSyncRepo),
		workspaceGuard:      app.NewWorkspaceGuardService(assignmentRepo, fulfillRepo, supplierRepo, shipmentRepo),
	}
}

func (c *WaveController) persistLifecycle(waveID uint) {
	if c.lifecycleSvc != nil {
		_ = c.lifecycleSvc.ProjectAndPersist(waveID)
	}
}

// CreateWave creates a new wave.
func (c *WaveController) CreateWave(input dto.CreateWaveInput) (dto.WaveDTO, error) {
	var wave domain.Wave
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		waveRepo := infra.NewWaveRepository(tx)
		demandRepo := infra.NewDemandRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		waveUC := app.NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)
		w := domain.Wave{Name: input.Name}
		if err := waveUC.CreateWave(&w); err != nil {
			return err
		}
		wave = w
		return nil
	})
	if err != nil {
		return dto.WaveDTO{}, err
	}
	return domainToWaveDTO(&wave), nil
}

// ListWaves lists all waves.
func (c *WaveController) ListWaves() ([]dto.WaveDTO, error) {
	waves, err := c.waveUC.ListWaves()
	if err != nil {
		return nil, err
	}
	result := make([]dto.WaveDTO, len(waves))
	for i, w := range waves {
		result[i] = domainToWaveDTO(&w)
	}
	return result, nil
}

// GetWave returns a single wave by ID.
func (c *WaveController) GetWave(id uint) (dto.WaveDTO, error) {
	w, err := c.waveUC.GetWave(id)
	if err != nil {
		return dto.WaveDTO{}, err
	}
	return domainToWaveDTO(w), nil
}

// GetWaveOverview returns aggregated wave overview data.
func (c *WaveController) GetWaveOverview(waveID uint) (dto.WaveOverviewDTO, error) {
	return c.overviewQueryUC.GetWaveOverview(waveID)
}

func (c *WaveController) GetWaveWorkspaceSnapshot(waveID uint) (dto.WaveWorkspaceSnapshotDTO, error) {
	return c.overviewQueryUC.GetWaveWorkspaceSnapshot(waveID)
}

func (c *WaveController) ListWaveFulfillmentRows(waveID uint) ([]dto.WaveFulfillmentRowDTO, error) {
	return c.overviewQueryUC.ListWaveFulfillmentRows(waveID)
}

func (c *WaveController) ListWaveParticipantRows(waveID uint) ([]dto.WaveParticipantRowDTO, error) {
	return c.overviewQueryUC.ListWaveParticipantRows(waveID)
}

// ListWaveDashboardRows returns batch-projected dashboard rows with authoritative
// projected lifecycle stages. This is the only dashboard data source.
func (c *WaveController) ListWaveDashboardRows() ([]dto.WaveDashboardRowDTO, error) {
	return c.overviewQueryUC.ListDashboardRows()
}

// AssignDemandToWave assigns a demand document to a wave.
func (c *WaveController) AssignDemandToWave(waveID uint, demandDocumentID uint) error {
	defer c.persistLifecycle(waveID)

	gormDB := c.gdb

	// Validate wave existence
	if _, err := c.waveUC.GetWave(waveID); err != nil {
		return err
	}
	// Validate demand document existence
	if _, err := c.demandRepo.FindByID(demandDocumentID); err != nil {
		return err
	}

	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(waveID)
	if err != nil {
		return err
	}

	return gormDB.Transaction(func(tx *gorm.DB) error {
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		waveRepo := infra.NewWaveRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		closureDecisionRepo := infra.NewClosureDecisionRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		demandRepoTx := infra.NewDemandRepository(tx)
		integrationProfileRepoTx := infra.NewIntegrationProfileRepository(tx)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		now := time.Now().Format(time.RFC3339)
		assignment := &domain.WaveDemandAssignment{
			WaveID:           waveID,
			DemandDocumentID: demandDocumentID,
			AcceptedAt:       now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := assignmentRepo.Create(assignment); err != nil {
			return err
		}

		// Capture profile snapshot onto the demand document at assignment time.
		doc, docErr := demandRepoTx.FindByID(demandDocumentID)
		if docErr == nil && doc != nil && doc.IntegrationProfileID != nil {
			profile, profErr := integrationProfileRepoTx.FindByID(*doc.IntegrationProfileID)
			if profErr == nil && profile != nil {
				snapshot := app.CaptureProfileSnapshot(profile)
				_ = demandRepoTx.UpdateBoundProfileSnapshot(demandDocumentID, snapshot)
			}
		}

		projHash, hashErr := projHashSvc.ComputeHash(waveID)
		if hashErr != nil {
			return hashErr
		}
		_, err := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  waveID,
			CommandKind:             domain.CmdAssignDemand,
			CommandSummary:          fmt.Sprintf("assign demand %d to wave %d", demandDocumentID, waveID),
			PatchPayload:            fmt.Sprintf(`{"op":"assign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, demandDocumentID),
			InversePatchPayload:     fmt.Sprintf(`{"op":"unassign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, demandDocumentID),
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		if err != nil {
			return err
		}
		return nil
	})
}

// GenerateParticipants generates WaveParticipantSnapshots from accepted demand lines.
func (c *WaveController) GenerateParticipants(waveID uint) (int, error) {
	defer c.persistLifecycle(waveID)

	gormDB := c.gdb
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(waveID)
	if err != nil {
		return 0, err
	}

	var count int
	err = gormDB.Transaction(func(tx *gorm.DB) error {
		waveRepo := infra.NewWaveRepository(tx)
		demandRepo := infra.NewDemandRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		closureDecisionRepo := infra.NewClosureDecisionRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		waveUC := app.NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		productRepo := infra.NewProductRepository(tx)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		generatedCount, genErr := waveUC.GenerateParticipants(waveID)
		if genErr != nil {
			return genErr
		}
		count = generatedCount

		postSnapshot, snapErr := snapshotSvc.CaptureSnapshot(waveID)
		if snapErr != nil {
			return snapErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(waveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  waveID,
			CommandKind:             domain.CmdGenerateParticipants,
			CommandSummary:          fmt.Sprintf("generate participants for wave %d (%d created)", waveID, count),
			PatchPayload:            fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
			InversePatchPayload:     fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
			CheckpointHint:          true,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// MapDemandLines converts eligible demand-driven DemandLines into FulfillmentLines.
func (c *WaveController) MapDemandLines(waveID uint) (*dto.DemandMappingResult, error) {
	defer c.persistLifecycle(waveID)

	gormDB := c.gdb
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(waveID)
	if err != nil {
		return nil, err
	}

	var mappingResult *dto.DemandMappingResult
	err = gormDB.Transaction(func(tx *gorm.DB) error {
		waveRepo := infra.NewWaveRepository(tx)
		demandRepo := infra.NewDemandRepository(tx)
		fulfillRepo := infra.NewFulfillmentRepository(tx)
		assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
		ruleRepo := infra.NewRuleRepository(tx)
		adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
		productRepo := infra.NewProductRepository(tx)
		closureDecisionRepo := infra.NewClosureDecisionRepository(tx)
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)
		addressRepo := infra.NewAddressRepository(tx)

		dmUC := app.NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, productRepo, addressRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		result, applyErr := dmUC.MapDemandToFulfillment(waveID)
		if applyErr != nil {
			return applyErr
		}
		mappingResult = result

		postSnapshot, snapErr := snapshotSvc.CaptureSnapshot(waveID)
		if snapErr != nil {
			return snapErr
		}

		summary := fmt.Sprintf("map demand lines for wave %d (%d created, %d blocked)", waveID, len(result.CreatedLines), len(result.BlockedLines))
		projHash, hashErr := projHashSvc.ComputeHash(waveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                  waveID,
			CommandKind:             domain.CmdMapDemandLines,
			CommandSummary:          summary,
			PatchPayload:            fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
			InversePatchPayload:     fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
			CheckpointHint:          true,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return nil, err
	}

	return mappingResult, nil
}

// domainToWaveDTO converts a domain Wave to a DTO.
func domainToWaveDTO(w *domain.Wave) dto.WaveDTO {
	if w == nil {
		return dto.WaveDTO{}
	}
	return dto.WaveDTO{
		ID:               w.ID,
		WaveNo:           w.WaveNo,
		Name:             w.Name,
		WaveType:         w.WaveType,
		LifecycleStage:   w.LifecycleStage,
		ProgressSnapshot: w.ProgressSnapshot,
		Notes:            w.Notes,
		LevelTags:        w.LevelTags,
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
}

// domainToFulfillmentLineDTO converts a domain FulfillmentLine to a DTO.
func domainToFulfillmentLineDTO(fl *domain.FulfillmentLine) dto.FulfillmentLineDTO {
	if fl == nil {
		return dto.FulfillmentLineDTO{}
	}
	return dto.FulfillmentLineDTO{
		ID:                        fl.ID,
		WaveID:                    fl.WaveID,
		CustomerProfileID:         fl.CustomerProfileID,
		WaveParticipantSnapshotID: fl.WaveParticipantSnapshotID,
		ProductID:                 fl.ProductID,
		DemandDocumentID:          fl.DemandDocumentID,
		DemandLineID:              fl.DemandLineID,
		CustomerAddressID:         fl.CustomerAddressID,
		Quantity:                  fl.Quantity,
		AllocationState:           fl.AllocationState,
		AddressState:              fl.AddressState,
		SupplierState:             fl.SupplierState,
		ChannelSyncState:          fl.ChannelSyncState,
		LineReason:                fl.LineReason,
		GeneratedBy:               fl.GeneratedBy,
		ExtraData:                 fl.ExtraData,
		CreatedAt:                 fl.CreatedAt,
		UpdatedAt:                 fl.UpdatedAt,
	}
}

// ListAssignedDemandsByWave returns all demand documents assigned to the given wave.
func (c *WaveController) ListAssignedDemandsByWave(waveID uint) ([]dto.DemandDocumentDTO, error) {
	docs, err := c.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DemandDocumentDTO, len(docs))
	for i := range docs {
		result[i] = domainToDemandDTO(&docs[i])
	}
	return result, nil
}

// buildUndoRedoUC constructs an UndoRedoUseCase whose patch executor and history
// repositories are all bound to the given transaction, so the inverse-patch
// application and the head-pointer update commit (or roll back) atomically.
func (c *WaveController) buildUndoRedoUC(tx *gorm.DB) app.UndoRedoUseCase {
	ruleRepo := infra.NewRuleRepository(tx)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(tx)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(tx)
	waveRepo := infra.NewWaveRepository(tx)
	fulfillRepo := infra.NewFulfillmentRepository(tx)
	closureDecisionRepo := infra.NewClosureDecisionRepository(tx)
	scopeRepo := infra.NewHistoryScopeRepository(tx)
	nodeRepo := infra.NewHistoryNodeRepository(tx)
	snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
	return app.NewUndoRedoUseCase(scopeRepo, nodeRepo, app.NewPatchExecutor(tx, snapshotSvc))
}

// UndoWaveAction undoes the last action for the given wave.
func (c *WaveController) UndoWaveAction(waveID uint) (string, error) {
	defer c.persistLifecycle(waveID)
	var summary string
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		s, undoErr := c.buildUndoRedoUC(tx).Undo(waveID)
		summary = s
		return undoErr
	})
	if err != nil {
		return "", err
	}
	return summary, nil
}

// RedoWaveAction redoes the last undone action for the given wave.
func (c *WaveController) RedoWaveAction(waveID uint) (string, error) {
	defer c.persistLifecycle(waveID)
	var summary string
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		s, redoErr := c.buildUndoRedoUC(tx).Redo(waveID)
		summary = s
		return redoErr
	})
	if err != nil {
		return "", err
	}
	return summary, nil
}

// ListRecentHistory returns the most recent history nodes for a wave.
func (c *WaveController) ListRecentHistory(waveID uint, limit int) ([]dto.HistoryNodeDTO, error) {
	return c.overviewQueryUC.ListRecentHistory(waveID, limit)
}

// GetHistoryGraph returns the full history node graph for a wave.
func (c *WaveController) GetHistoryGraph(waveID uint) (dto.HistoryGraphDTO, error) {
	graph, err := c.historyGraphUC.GetHistoryGraph(waveID)
	if err != nil {
		return dto.HistoryGraphDTO{}, err
	}
	return *graph, nil
}

// RunHistoryGC runs garbage collection on the history for a wave, keeping the last 100 reachable nodes.
func (c *WaveController) RunHistoryGC(waveID uint) (int, error) {
	return c.historyGCSvc.CollectGarbageForWave(waveID, 100)
}

// ListWavesPaginated returns a paginated list of waves.
func (c *WaveController) ListWavesPaginated(input dto.PaginationInput) (map[string]any, error) {
	input = dto.NormalizePagination(input)
	offset := (input.Page - 1) * input.PageSize
	waves, total, err := c.waveUC.ListWavesPaginated(offset, input.PageSize)
	if err != nil {
		return nil, err
	}
	result := dto.PaginationResult{Page: input.Page, PageSize: input.PageSize, TotalCount: int(total)}
	result.ComputePages()

	dtos := make([]dto.WaveDTO, len(waves))
	for i, w := range waves {
		dtos[i] = domainToWaveDTO(&w)
	}
	return map[string]any{"items": dtos, "pagination": result}, nil
}

// ValidateStepAccess checks workspace guard invariants for a wave step.
func (c *WaveController) ValidateStepAccess(waveID uint, stepKey string) error {
	if c.workspaceGuard == nil {
		return nil
	}
	switch stepKey {
	case "allocation":
		return c.workspaceGuard.GuardAllocationRequiresDemandIntake(waveID)
	case "review":
		return c.workspaceGuard.GuardReviewRequiresFulfillment(waveID)
	case "execution":
		return c.workspaceGuard.GuardExecutionRequiresReview(waveID)
	case "shipment":
		return c.workspaceGuard.GuardShipmentRequiresSupplierOrder(waveID)
	case "sync":
		return c.workspaceGuard.GuardSyncRequiresShipment(waveID)
	default:
		return fmt.Errorf("unknown step key: %s", stepKey)
	}
}
