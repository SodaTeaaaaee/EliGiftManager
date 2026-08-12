package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	application "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/middleware"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsWindows "github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()
	app := NewApp(cfg)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Initialize DB singleton.
	dbPath, err := app.resolveDatabasePath()
	if err != nil {
		logger.Error("resolve database path", "error", err)
		os.Exit(1)
	}
	db, err := database.InitDB(dbPath)
	if err != nil {
		logger.Error("initialize database", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("get underlying sql.DB", "error", err)
		os.Exit(1)
	}
	database.SetDefaultDB(db)
	defer sqlDB.Close()
	featurePolicyRepo := infra.NewCustomerResolutionFeaturePolicyRepository(db)
	featurePolicy, err := featurePolicyRepo.GetFeaturePolicy(context.Background())
	if err != nil {
		logger.Error("load customer resolution feature policy", "error", err)
		os.Exit(1)
	}
	if err := migrateLegacyMergePolicy(context.Background(), db); err != nil {
		logger.Error("migrate legacy merge policy", "error", err)
		os.Exit(1)
	}
	if featurePolicy.ImportEvidenceEnabled {
		evidenceUC := application.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(db))
		if runsDeleted, recordsDeleted, pruneErr := evidenceUC.PruneExpired(context.Background()); pruneErr != nil {
			logger.Error("prune expired import evidence", "error", pruneErr)
		} else if runsDeleted > 0 || recordsDeleted > 0 {
			logger.Info("pruned expired import evidence", "runs_deleted", runsDeleted, "records_deleted", recordsDeleted)
		}
	} else {
		logger.Warn("import evidence disabled by customer resolution feature policy; cold-start pruning skipped")
	}

	zoom := LoadZoom()

	err = wails.Run(&options.App{
		Title:     cfg.Name,
		Width:     cfg.WindowWidth,
		Height:    cfg.WindowHeight,
		MinWidth:  cfg.MinWindowWidth,
		MinHeight: cfg.MinWindowHeight,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: middleware.LocalAssetsMiddleware("/local-images/"),
		},
		BackgroundColour: &options.RGBA{R: 20, G: 18, B: 16, A: 1},
		Windows: &wailsWindows.Options{
			ZoomFactor:           zoom / 100.0,
			IsZoomControlEnabled: true,
		},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		Bind: []any{
			app,
			NewListPaginationController(),
			NewDemandController(),
			NewWaveController(),
			NewExportController(),
			NewShipmentController(),
			NewChannelSyncController(),
			NewAdjustmentController(),
			NewTemplateController(),
			NewAllocationPolicyController(),
			NewProductController(),
			NewProfileController(),
			NewAddressController(),
			NewMergeController(),
			NewMergeUndoController(),
			NewSplitController(),
			NewCustomerProfileController(),
			NewMergeGovernanceController(),
			NewActionCenterController(),
			NewFileSystemController(),
			NewImportEvidenceController(),
			NewCustomerResolutionFeaturePolicyController(),
		},
	})
	if err != nil {
		logger.Error("run wails application", "error", err)
		os.Exit(1)
	}
}
