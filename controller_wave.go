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
	fulfillRepo         domain.FulfillmentLineRepository
	supplierRepo        domain.SupplierOrderRepository
	assignmentRepo      domain.WaveDemandAssignmentRepository
	demandRepo          domain.DemandDocumentRepository
	shipmentRepo        domain.ShipmentRepository
	gdb                 *gorm.DB
	nodeRepo            domain.HistoryNodeRepository
	overviewProjUC      app.WaveOverviewProjectionUseCase
	undoRedoUC          app.UndoRedoUseCase
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
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

	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gormDB)
	snapshotSvc := app.NewWaveSnapshotService(gormDB, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)

	basisDriftUC := app.NewBasisDriftDetectionUseCase(supplierRepo, shipmentRepo, channelSyncRepo)
	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)

	return &WaveController{
		waveUC:              app.NewWaveUseCase(waveRepo, demandRepo, assignmentRepo),
		demandMappingUC:     app.NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil),
		fulfillRepo:         fulfillRepo,
		supplierRepo:        supplierRepo,
		assignmentRepo:      assignmentRepo,
		demandRepo:          demandRepo,
		shipmentRepo:        shipmentRepo,
		nodeRepo:            historyNodeRepo,
		overviewProjUC:      app.NewWaveOverviewProjectionUseCase(channelSyncRepo, closureDecisionRepo, basisDriftUC, historyHeadUC),
		gdb:                gormDB,
		undoRedoUC:          app.NewUndoRedoUseCase(historyScopeRepo, historyNodeRepo, app.NewPatchExecutor(gormDB, snapshotSvc)),
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo),
		snapshotSvc:         snapshotSvc,
	}
}

// CreateWave creates a new wave.
func (c *WaveController) CreateWave(input dto.CreateWaveInput) (dto.WaveDTO, error) {
	wave := domain.Wave{
		Name: input.Name,
	}
	if err := c.waveUC.CreateWave(&wave); err != nil {
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
	w, err := c.waveUC.GetWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	fulfillLines, err := c.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	supplierOrders, err := c.supplierRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	// Build wave-scoped product lookup for mapping-blocked detection
	productRepo := infra.NewProductRepository(c.gdb)
	waveProducts, _ := productRepo.ListByWave(waveID)
	productMasterToWaveProduct := make(map[uint]uint, len(waveProducts))
	for _, wp := range waveProducts {
		if wp.ProductMasterID != nil {
			productMasterToWaveProduct[*wp.ProductMasterID] = wp.ID
		}
	}

	// Count demand lines for this wave via assignments, bucketed by routing/input state
	docs, err := c.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}

	var (
		demandCount              int
		acceptedReadyOrNotReq    int
		acceptedWaitingForInput  int
		deferredCount            int
		excludedManualCount      int
		excludedDuplicateCount   int
		excludedRevokedCount     int
		mappingBlockedCount      int
	)
	for _, doc := range docs {
		lines, err := c.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, line := range lines {
			switch line.RoutingDisposition {
			case "accepted":
				demandCount++
				if line.RecipientInputState == "ready" || line.RecipientInputState == "not_required" {
					acceptedReadyOrNotReq++
					if doc.Kind == "retail_order" && line.ProductMasterID != nil {
						if _, ok := productMasterToWaveProduct[*line.ProductMasterID]; !ok {
							mappingBlockedCount++
						}
					}
				} else if line.RecipientInputState == "waiting_for_input" || line.RecipientInputState == "partially_collected" {
					acceptedWaitingForInput++
				}
			case "deferred":
				deferredCount++
			case "excluded_manual":
				excludedManualCount++
			case "excluded_duplicate":
				excludedDuplicateCount++
			case "excluded_revoked":
				excludedRevokedCount++
			}
		}
	}

	// Collect shipment stats
	shipments, err := c.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return dto.WaveOverviewDTO{}, err
	}
	shipmentCount := len(shipments)

	trackedFulfillmentCount := 0
	trackedSet := make(map[uint]bool)
	for _, s := range shipments {
		if s.TrackingNo == "" {
			continue
		}
		lines, err := c.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return dto.WaveOverviewDTO{}, err
		}
		for _, l := range lines {
			if !trackedSet[l.FulfillmentLineID] {
				trackedSet[l.FulfillmentLineID] = true
				trackedFulfillmentCount++
			}
		}
	}

	base := dto.WaveOverviewDTO{
		Wave:                     domainToWaveDTO(w),
		DemandCount:              demandCount,
		FulfillmentCount:         len(fulfillLines),
		SupplierOrderCount:       len(supplierOrders),
		ShipmentCount:            shipmentCount,
		TrackedFulfillmentCount:  trackedFulfillmentCount,
		AcceptedReadyOrNotRequired: acceptedReadyOrNotReq,
		AcceptedWaitingForInput:    acceptedWaitingForInput,
		DeferredCount:              deferredCount,
		ExcludedManualCount:       excludedManualCount,
		ExcludedDuplicateCount:    excludedDuplicateCount,
		ExcludedRevokedCount:      excludedRevokedCount,
		MappingBlockedCount:       mappingBlockedCount,
	}
	return c.overviewProjUC.ProjectWaveOverview(base)
}


// AssignDemandToWave assigns a demand document to a wave.
func (c *WaveController) AssignDemandToWave(waveID uint, demandDocumentID uint) error {
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
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo)

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

		_, err := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                 waveID,
			CommandKind:            domain.CmdAssignDemand,
			CommandSummary:         fmt.Sprintf("assign demand %d to wave %d", demandDocumentID, waveID),
			PatchPayload:           fmt.Sprintf(`{"op":"assign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, demandDocumentID),
			InversePatchPayload:    fmt.Sprintf(`{"op":"unassign_demand","wave_id":%d,"demand_document_id":%d}`, waveID, demandDocumentID),
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:         projHashSvc.ComputeHash(waveID),
		})
		if err != nil {
			return err
		}
		return nil
	})
}

// GenerateParticipants generates WaveParticipantSnapshots from accepted demand lines.
func (c *WaveController) GenerateParticipants(waveID uint) (int, error) {
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
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		waveUC := app.NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo)

		generatedCount, genErr := waveUC.GenerateParticipants(waveID)
		if genErr != nil {
			return genErr
		}
		count = generatedCount

		postSnapshot, snapErr := snapshotSvc.CaptureSnapshot(waveID)
		if snapErr != nil {
			return snapErr
		}

		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                 waveID,
			CommandKind:            domain.CmdGenerateParticipants,
			CommandSummary:         fmt.Sprintf("generate participants for wave %d (%d created)", waveID, count),
			PatchPayload:           fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
			InversePatchPayload:    fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
			CheckpointHint:         true,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:         projHashSvc.ComputeHash(waveID),
		})
		return recordErr
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// MapDemandLines converts eligible demand-driven DemandLines into FulfillmentLines
// for retail_order demand documents assigned to the given wave.
// Returns a DemandMappingResult that includes both successfully mapped lines
// and any lines blocked by missing product references.
func (c *WaveController) MapDemandLines(waveID uint) (*dto.DemandMappingResult, error) {
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
		historyScopeRepo := infra.NewHistoryScopeRepository(tx)
		historyNodeRepo := infra.NewHistoryNodeRepository(tx)
		historyCheckpointRepo := infra.NewHistoryCheckpointRepository(tx)

		dmUC := app.NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, productRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, snapshotSvc)
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo)

		result, applyErr := dmUC.MapDemandToFulfillment(waveID)
		if applyErr != nil {
			return applyErr
		}
		mappingResult = result

		postSnapshot, snapErr := snapshotSvc.CaptureSnapshot(waveID)
		if snapErr != nil {
			return snapErr
		}

		totalLines := len(result.CreatedLines) + len(result.BlockedLines)
		summary := fmt.Sprintf("map demand lines for wave %d (%d created, %d blocked)", waveID, len(result.CreatedLines), len(result.BlockedLines))
		_ = totalLines
		_, recordErr := historySvc.RecordNode(app.RecordNodeInput{
			WaveID:                 waveID,
			CommandKind:            domain.CmdMapDemandLines,
			CommandSummary:         summary,
			PatchPayload:           fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, postSnapshot),
			InversePatchPayload:    fmt.Sprintf(`{"op":"restore_checkpoint","data":%q}`, preSnapshot),
			CheckpointHint:         true,
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:         projHashSvc.ComputeHash(waveID),
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

// UndoWaveAction undoes the last action for the given wave.
func (c *WaveController) UndoWaveAction(waveID uint) (string, error) {
	return c.undoRedoUC.Undo(waveID)
}

// RedoWaveAction redoes the last undone action for the given wave.
func (c *WaveController) RedoWaveAction(waveID uint) (string, error) {
	return c.undoRedoUC.Redo(waveID)
}

// ListRecentHistory returns the most recent history nodes for a wave.
func (c *WaveController) ListRecentHistory(waveID uint, limit int) ([]dto.HistoryNodeDTO, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	scope, err := c.historyRecordingSvc.FindScope(waveID)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return []dto.HistoryNodeDTO{}, nil
	}
	nodes, err := c.nodeRepo.ListByScopeRecent(scope.ID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]dto.HistoryNodeDTO, len(nodes))
	for i, n := range nodes {
		if n.CommandKind == domain.CmdSystemBaseline {
			continue
		}
		result[i] = dto.HistoryNodeDTO{
			ID:             n.ID,
			CommandKind:    n.CommandKind,
			CommandSummary: n.CommandSummary,
			CreatedAt:      n.CreatedAt,
			CreatedBy:      n.CreatedBy,
		}
	}
	filtered := make([]dto.HistoryNodeDTO, 0, len(result))
	for _, item := range result {
		if item.CommandKind == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}
