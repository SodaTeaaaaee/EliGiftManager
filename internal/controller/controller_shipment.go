package controller

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// ShipmentController exposes shipment-management Wails bindings.
type ShipmentController struct {
	shipmentUC          app.ShipmentUseCase
	shipmentImportUC    app.ShipmentImportUseCase
	shipmentRepo        domain.ShipmentRepository
	gdb                 *gorm.DB
	historyRecordingSvc *app.HistoryRecordingService
	projHashSvc         *app.ProjectionHashService
	snapshotSvc         *app.WaveSnapshotService
}

func NewShipmentController() *ShipmentController {
	gdb := db.GetDB()
	shipmentRepo := infra.NewShipmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	ruleRepo := infra.NewRuleRepository(gdb)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	historyScopeRepo := infra.NewHistoryScopeRepository(gdb)
	historyNodeRepo := infra.NewHistoryNodeRepository(gdb)
	historyPinRepo := infra.NewHistoryPinRepository(gdb)
	historyCheckpointRepo := infra.NewHistoryCheckpointRepository(gdb)

	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
	snapshotSvc := app.NewWaveSnapshotService(gdb, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)

	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	addressRepo := infra.NewAddressRepository(gdb)
	carrierMappingRepo := infra.NewCarrierMappingRepository(gdb)
	mapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	carrierUC := app.NewCarrierMappingUseCase(carrierMappingRepo, profileRepo)
	masterRepo := infra.NewProductMasterRepository(gdb)
	importUC := app.NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
	importUC = app.WithShipmentImportWaveRepo(importUC, waveRepo)
	importUC = app.WithShipmentReconcileDeps(importUC, mapping, productRepo, masterRepo, addressRepo, carrierUC)
	importUC = app.WithShipmentImportEvidence(importUC, app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)))
	importUC = app.WithShipmentExternalCarrierRegistry(importUC, app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(gdb)))

	return &ShipmentController{
		shipmentUC:          app.WithShipmentWaveRepo(app.NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp), waveRepo),
		shipmentImportUC:    importUC,
		shipmentRepo:        shipmentRepo,
		gdb:                 gdb,
		historyRecordingSvc: app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc)),
		projHashSvc:         app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo),
		snapshotSvc:         snapshotSvc,
	}
}

func newShipmentLifecycleUC(repos *infra.TxRepos, writeRepo domain.ShipmentWriteRepository) app.ShipmentLifecycleUseCase {
	return app.NewShipmentLifecycleUseCase(repos.ShipmentRepo, writeRepo, repos.FulfillRepo, repos.SupplierRepo)
}

// CreateShipment creates a shipment with its lines.
func (c *ShipmentController) CreateShipment(input dto.CreateShipmentInput) (dto.ShipmentDTO, error) {
	ctx := appContext
	preSnapshot, err := c.snapshotSvc.CaptureSnapshotForSupplierOrder(ctx, input.SupplierOrderID)
	if err != nil {
		return dto.ShipmentDTO{}, err
	}

	var shipment *domain.Shipment
	var lines []domain.ShipmentLine
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		shipmentRepo := repos.ShipmentRepo
		supplierRepo := repos.SupplierRepo
		fulfillRepo := repos.FulfillRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyPinRepo := repos.HistoryPin
		historyCheckpointRepo := repos.HistoryCheckpoint

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		shipmentUC := app.WithShipmentWaveRepo(app.NewShipmentUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp), waveRepo)
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		createdShipment, createdLines, createErr := shipmentUC.CreateShipment(ctx, input)
		if createErr != nil {
			return createErr
		}
		shipment = createdShipment
		lines = createdLines

		supplierOrder, findErr := supplierRepo.FindByID(ctx, input.SupplierOrderID)
		if findErr != nil {
			return findErr
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, supplierOrder.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  supplierOrder.WaveID,
			CommandKind:             domain.CmdCreateShipment,
			CommandSummary:          fmt.Sprintf("create shipment %d for wave %d", shipment.ID, supplierOrder.WaveID),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return dto.ShipmentDTO{}, err
	}

	result := domainToShipmentDTO(shipment)
	result.Lines = make([]dto.ShipmentLineDTO, len(lines))
	for i, l := range lines {
		result.Lines[i] = domainToShipmentLineDTO(&l)
	}
	return result, nil
}

// ListShipmentsByWave lists all shipments for a given wave.
func (c *ShipmentController) ListShipmentsByWave(waveID uint) ([]dto.ShipmentDTO, error) {
	ctx := appContext
	shipments, err := c.shipmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ShipmentDTO, len(shipments))
	for i, s := range shipments {
		shipmentDTO := domainToShipmentDTO(&s)
		lines, err := c.shipmentRepo.ListLinesByShipment(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		shipmentDTO.Lines = make([]dto.ShipmentLineDTO, len(lines))
		for j, l := range lines {
			shipmentDTO.Lines[j] = domainToShipmentLineDTO(&l)
		}
		result[i] = shipmentDTO
	}
	return result, nil
}

// domainToShipmentDTO converts a domain Shipment to a DTO.
func domainToShipmentDTO(s *domain.Shipment) dto.ShipmentDTO {
	if s == nil {
		return dto.ShipmentDTO{}
	}
	return dto.ShipmentDTO{
		ID:                   s.ID,
		SupplierOrderID:      s.SupplierOrderID,
		SupplierPlatform:     s.SupplierPlatform,
		ShipmentNo:           s.ShipmentNo,
		ExternalShipmentNo:   s.ExternalShipmentNo,
		CarrierCode:          s.CarrierCode,
		CarrierName:          s.CarrierName,
		TrackingNo:           s.TrackingNo,
		Status:               s.Status,
		ShippedAt:            s.ShippedAt,
		BasisHistoryNodeID:   s.BasisHistoryNodeID,
		BasisProjectionHash:  s.BasisProjectionHash,
		BasisPayloadSnapshot: s.BasisPayloadSnapshot,
		ExtraData:            s.ExtraData,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// ImportShipments performs a bulk import of shipments from factory return data.
// Entries sharing the same ExternalShipmentNo are grouped into a single Shipment.
// Partial success is supported — failed groups are recorded in the result Errors slice.
func (c *ShipmentController) ImportShipments(input dto.ImportShipmentInput) (dto.ImportShipmentResult, error) {
	ctx := appContext
	evidence := app.NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(c.gdb))
	if err := app.PrepareShipmentEntryImportEvidence(ctx, evidence, input); err != nil {
		return dto.ImportShipmentResult{}, fmt.Errorf("prepare shipment import evidence: %w", err)
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.ImportShipmentResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}

	var importResult *dto.ImportShipmentResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		shipmentRepo := repos.ShipmentRepo
		supplierRepo := repos.SupplierRepo
		fulfillRepo := repos.FulfillRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyPinRepo := repos.HistoryPin
		historyCheckpointRepo := repos.HistoryCheckpoint

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		importUC := app.NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		importUC = app.WithShipmentImportWaveRepo(importUC, waveRepo)
		importUC = app.WithShipmentImportEvidence(importUC, evidence)
		importUC = app.WithShipmentExternalCarrierRegistry(importUC, app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		res, importErr := importUC.ImportShipments(ctx, input)
		if importErr != nil {
			return importErr
		}
		importResult = res
		if res.SuccessCount == 0 {
			return nil
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdCreateShipment,
			CommandSummary:          fmt.Sprintf("import %d shipments for wave %d (%d succeeded, %d failed)", res.TotalProcessed, input.WaveID, res.SuccessCount, res.ErrorCount),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return dto.ImportShipmentResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportShipmentResult{}, err
	}

	return *importResult, nil
}

// MapAndReconcileShipments maps an external factory-return sheet onto internal IDs
// then imports the reconciled entries. History recording mirrors ImportShipments.
func (c *ShipmentController) MapAndReconcileShipments(input dto.MapAndReconcileShipmentsInput) (dto.ImportShipmentResult, error) {
	ctx := appContext
	if err := validateShipmentImportFilePath(input.FilePath); err != nil {
		return dto.ImportShipmentResult{}, err
	}
	baseProfileRepo := infra.NewIntegrationProfileRepository(c.gdb)
	baseMapping := app.NewTemplateMappingService(infra.NewDocumentTemplateRepository(c.gdb), infra.NewProfileTemplateBindingRepository(c.gdb), baseProfileRepo)
	evidence := app.NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(c.gdb))
	if err := app.PrepareTemplateImportEvidence(ctx, evidence, baseMapping, app.PrepareTemplateImportEvidenceInput{
		ImportKind: "supplier_shipment", DocumentType: "import_supplier_shipment",
		IntegrationProfileID: input.IntegrationProfileID, ImportMode: input.ImportMode,
		FilePath: input.FilePath, Rows: input.Rows,
	}); err != nil {
		return dto.ImportShipmentResult{}, fmt.Errorf("prepare shipment import evidence: %w", err)
	}
	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.ImportShipmentResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}

	var importResult *dto.ImportShipmentResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		shipmentRepo := repos.ShipmentRepo
		supplierRepo := repos.SupplierRepo
		fulfillRepo := repos.FulfillRepo
		ruleRepo := repos.RuleRepo
		adjustmentRepo := repos.AdjustmentRepo
		assignmentRepo := repos.AssignmentRepo
		waveRepo := repos.WaveRepo
		productRepo := repos.ProductRepo
		closureDecisionRepo := repos.ClosureDecision
		historyScopeRepo := repos.HistoryScope
		historyNodeRepo := repos.HistoryNode
		historyPinRepo := repos.HistoryPin
		historyCheckpointRepo := repos.HistoryCheckpoint

		historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
		basisStamp := app.NewBasisStampService(historyHeadUC, historyPinRepo)
		importUC := app.NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, basisStamp)
		importUC = app.WithShipmentImportWaveRepo(importUC, waveRepo)

		templateRepo := infra.NewDocumentTemplateRepository(tx)
		bindingRepo := infra.NewProfileTemplateBindingRepository(tx)
		profileRepo := infra.NewIntegrationProfileRepository(tx)
		addressRepo := infra.NewAddressRepository(tx)
		carrierMappingRepo := infra.NewCarrierMappingRepository(tx)
		mapping := app.NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
		carrierUC := app.NewCarrierMappingUseCase(carrierMappingRepo, profileRepo)
		masterRepo := infra.NewProductMasterRepository(tx)
		importUC = app.WithShipmentReconcileDeps(importUC, mapping, productRepo, masterRepo, addressRepo, carrierUC)
		importUC = app.WithShipmentImportEvidence(importUC, evidence)
		importUC = app.WithShipmentExternalCarrierRegistry(importUC, app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))

		snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
		historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
		projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

		res, importErr := importUC.MapAndReconcileShipments(ctx, input)
		if importErr != nil {
			return importErr
		}
		importResult = res

		// Skip history when nothing was reconciled/imported successfully.
		if res.SuccessCount == 0 {
			return nil
		}

		projHash, hashErr := projHashSvc.ComputeHash(ctx, input.WaveID)
		if hashErr != nil {
			return hashErr
		}
		_, recordErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
			WaveID:                  input.WaveID,
			CommandKind:             domain.CmdCreateShipment,
			CommandSummary:          fmt.Sprintf("map+import %d shipment rows for wave %d (%d succeeded, %d failed)", res.TotalProcessed, input.WaveID, res.SuccessCount, res.ErrorCount),
			PatchPayload:            "",
			InversePatchPayload:     "",
			BaselineSnapshotPayload: preSnapshot,
			ProjectionHash:          projHash,
		})
		return recordErr
	})
	if err != nil {
		return dto.ImportShipmentResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportShipmentResult{}, err
	}

	return *importResult, nil
}

// domainToShipmentLineDTO converts a domain ShipmentLine to a DTO.
func domainToShipmentLineDTO(l *domain.ShipmentLine) dto.ShipmentLineDTO {
	if l == nil {
		return dto.ShipmentLineDTO{}
	}
	return dto.ShipmentLineDTO{
		ID:                  l.ID,
		ShipmentID:          l.ShipmentID,
		SupplierOrderLineID: l.SupplierOrderLineID,
		FulfillmentLineID:   l.FulfillmentLineID,
		Quantity:            l.Quantity,
		CreatedAt:           l.CreatedAt,
	}
}

const defaultMaxShipmentImportFileBytes int64 = 32 << 20

// maxShipmentImportFileBytes is the import size ceiling. Tests may lower it.
var maxShipmentImportFileBytes = defaultMaxShipmentImportFileBytes

func validateShipmentImportFilePath(path string) error {
	if path == "" {
		return nil
	}
	if strings.IndexByte(path, 0) >= 0 {
		return fmt.Errorf("import file path contains a NUL byte")
	}
	if importPathHasDotDot(path) {
		return fmt.Errorf("import file path must not contain ..")
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv", ".xlsx", ".xls":
	default:
		return fmt.Errorf("unsupported import file extension %q", ext)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat import file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("import path is a directory")
	}
	if info.Size() > maxShipmentImportFileBytes {
		return fmt.Errorf("import file exceeds maximum size of %d bytes", maxShipmentImportFileBytes)
	}
	return nil
}

func importPathHasDotDot(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}
