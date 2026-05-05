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

//go:embed presets/templates/*.json
var presetFS embed.FS

func main() {
	cfg := config.Load()
	app := NewApp(cfg, nil, nil)

	// Initialize database.
	dbPath, _ := app.resolveDatabasePath()
	db, dbErr := database.InitDB(dbPath)
	if dbErr != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("initialize database", "error", dbErr)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Controllers — wire *gorm.DB via constructor injection.
	waveCtrl := &WaveController{db: db}
	systemCtrl := &SystemController{appCfg: cfg, db: db}
	memberCtrl := &MemberController{db: db}
	productCtrl := &ProductController{db: db}
	templateCtrl := &TemplateController{db: db, presetFS: presetFS}

	// Wire controllers into App for startup() SetContext injection.
	app.waveCtrl = waveCtrl
	app.systemCtrl = systemCtrl

	zoom := LoadZoom()

	err := wails.Run(&options.App{
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
			memberCtrl,
			productCtrl,
			waveCtrl,
			systemCtrl,
			templateCtrl,
		},
	})
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("run wails application", "error", err)
		os.Exit(1)
	}
}
