package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	appcore "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/controller"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/middleware"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	application "github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	dbPath, err := resolveDatabasePath()
	if err != nil {
		logger.Error("resolve database path", "error", err)
		os.Exit(1)
	}
	gdb, err := database.InitDB(dbPath)
	if err != nil {
		logger.Error("initialize database", "error", err)
		os.Exit(1)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		logger.Error("get underlying sql.DB", "error", err)
		os.Exit(1)
	}
	database.SetDefaultDB(gdb)
	defer sqlDB.Close()

	ws := appcore.NewWorkspace(infra.NewGormStore(gdb))
	if err := ws.EnsureBuiltinPlatforms(context.Background()); err != nil {
		logger.Error("seed builtin platforms", "error", err)
		os.Exit(1)
	}
	if err := ws.EnsureBuiltinTemplates(context.Background()); err != nil {
		logger.Error("seed builtin templates", "error", err)
		os.Exit(1)
	}

	app := application.New(application.Options{
		Name:        cfg.Name,
		Description: cfg.Description,
		Services: []application.Service{
			application.NewService(controller.NewWorkspaceController(ws)),
			application.NewService(controller.NewFileSystemController()),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: middleware.LocalAssetsMiddleware("/local-images/"),
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "main",
		Title:              cfg.Name,
		Width:              cfg.WindowWidth,
		Height:             cfg.WindowHeight,
		MinWidth:           cfg.MinWindowWidth,
		MinHeight:          cfg.MinWindowHeight,
		BackgroundColour:   application.RGBA{Red: 20, Green: 18, Blue: 16, Alpha: 255},
		Zoom:               LoadZoomPercent() / 100.0,
		ZoomControlEnabled: true,
	})

	// Persist the zoom level when the window is about to close. The hook runs
	// before the webview is torn down, so GetZoom still reads a live value.
	// GetZoom returns -1 when the underlying zoom query fails (e.g. Windows
	// GetZoomFactor errors), so skip non-positive sentinels instead of letting
	// the clamp persist them as 25.
	win.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		if z := win.GetZoom(); z > 0 {
			if err := SaveZoomPercent(z * 100); err != nil {
				logger.Warn("save zoom", "error", err)
			}
		}
	})

	if err := app.Run(); err != nil {
		logger.Error("run wails application", "error", err)
		os.Exit(1)
	}
}

func resolveDatabasePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "eligiftmanager.db"), nil
}
