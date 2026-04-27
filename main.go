package main

import (
	"embed"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
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
			Assets: assets,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/local-images/") {
					assetsDir, err := service.ResolveAssetsDir()
					if err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					relPath := strings.TrimPrefix(r.URL.Path, "/local-images/")
					cleanPath := filepath.Clean(relPath)
					filePath := filepath.Join(assetsDir, cleanPath)
					// Prevent directory traversal
					if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(assetsDir)) {
						http.Error(w, "Forbidden", http.StatusForbidden)
						return
					}
					http.ServeFile(w, r, filePath)
					return
				}
			}),
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
