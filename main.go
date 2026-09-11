package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	application "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/controller"
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

	dbPath, err := app.resolveDatabasePath()
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

	ws := application.NewWorkspace(infra.NewGormStore(gdb))
	if err := ws.EnsureBuiltinPlatforms(context.Background()); err != nil {
		logger.Error("seed builtin platforms", "error", err)
		os.Exit(1)
	}
	if err := ws.EnsureBuiltinTemplates(context.Background()); err != nil {
		logger.Error("seed builtin templates", "error", err)
		os.Exit(1)
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
			controller.NewWorkspaceController(ws),
			controller.NewFileSystemController(),
		},
	})
	if err != nil {
		logger.Error("run wails application", "error", err)
		os.Exit(1)
	}
}
