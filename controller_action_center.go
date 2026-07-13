package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// ActionCenterController exposes the Home-page / nav-badge aggregation Wails
// binding. Self-contained: constructs its own repos and use cases from the shared
// DB singleton, mirroring the other single-purpose controllers (e.g. MergeController).
type ActionCenterController struct {
	gdb       *gorm.DB
	summaryUC app.ActionCenterUseCase
}

func NewActionCenterController() *ActionCenterController {
	gormDB := database.GetDB()

	waveRepo := infra.NewWaveRepository(gormDB)
	demandRepo := infra.NewDemandRepository(gormDB)
	fulfillRepo := infra.NewFulfillmentRepository(gormDB)
	supplierRepo := infra.NewSupplierOrderRepository(gormDB)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gormDB)
	shipmentRepo := infra.NewShipmentRepository(gormDB)
	productRepo := infra.NewProductRepository(gormDB)
	profileRepo := infra.NewIntegrationProfileRepository(gormDB)
	channelSyncRepo := infra.NewChannelSyncRepository(gormDB)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gormDB)
	historyScopeRepo := infra.NewHistoryScopeRepository(gormDB)
	historyNodeRepo := infra.NewHistoryNodeRepository(gormDB)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(gormDB)

	basisDriftUC := app.NewBasisDriftDetectionUseCase(supplierRepo, shipmentRepo, channelSyncRepo, fulfillRepo)
	historyHeadUC := app.NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	overviewProjUC := app.NewWaveOverviewProjectionUseCase(channelSyncRepo, closureDecisionRepo, basisDriftUC, historyHeadUC)
	overviewQueryUC := app.NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo,
		shipmentRepo, productRepo, profileRepo, historyScopeRepo, historyNodeRepo,
		overviewProjUC, adjustmentRepo,
	)

	return &ActionCenterController{
		gdb:       gormDB,
		summaryUC: app.NewActionCenterUseCase(waveRepo, demandRepo, overviewQueryUC),
	}
}

// GetActionCenterSummary returns the aggregated Home-page payload: per-wave blocked
// buckets with deep-link filters, the inbox pending-intake count, and nav badges.
func (c *ActionCenterController) GetActionCenterSummary() (dto.ActionCenterSummaryDTO, error) {
	ctx := appContext
	return c.summaryUC.GetActionCenterSummary(ctx)
}
