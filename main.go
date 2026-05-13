package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
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
			NewDemandController(),
			NewWaveController(),
			NewExportController(),
		},
	})
	if err != nil {
		logger.Error("run wails application", "error", err)
		os.Exit(1)
	}
}
