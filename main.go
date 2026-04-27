package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/middleware"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()
	app := NewApp(cfg)

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
		OnStartup:        app.startup,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("run wails application", "error", err)
		os.Exit(1)
	}
}
