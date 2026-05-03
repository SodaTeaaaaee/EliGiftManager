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

	// Initialize DB singleton early so controllers can use it.
	dbPath, _ := app.resolveDatabasePath()
	if db, err := database.InitDB(dbPath); err == nil {
		sqlDB, _ := db.DB()
		database.SetDefaultDB(db)
		defer sqlDB.Close()
	}

	// Controllers
	memberCtrl := &MemberController{}
	productCtrl := &ProductController{}
	waveCtrl := &WaveController{}
	systemCtrl := &SystemController{appCfg: cfg}
	templateCtrl := &TemplateController{}
	sysCtrl = systemCtrl // wire into startup() via app.go global

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
