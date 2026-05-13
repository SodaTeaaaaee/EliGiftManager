package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---- App struct ----

type App struct {
	ctx context.Context
	cfg config.App
}

// ---- App: lifecycle ----

func NewApp(cfg config.App) *App { return &App{cfg: cfg} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(ctx context.Context) bool {
	// Trigger JS-side zoom persistence before WebView2 shuts down.
	wailsruntime.WindowExecJS(ctx, "if(window.__persistZoom)window.__persistZoom()")
	return false // false = allow close
}

func (a *App) resolveDatabasePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "eligiftmanager.db"), nil
}

// ---- Wails-bound file pickers ----

func (a *App) PickCSVFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择 CSV 文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
		},
	})
}

func (a *App) PickZIPFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择 ZIP 文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "ZIP Files", Pattern: "*.zip"},
		},
	})
}

// SaveZoom persists the current zoom level to zoom.cfg.
func (a *App) SaveZoom(zoomPercent float64) error {
	cfgPath, err := zoomFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, fmt.Appendf(nil, "%.2f", zoomPercent), 0o644)
}

// ---- Tool functions ----

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func copyFile(source, target string) error {
	sameFile, err := sameFilePath(source, target)
	if err != nil {
		return err
	}
	if sameFile {
		return fmt.Errorf("source and target must be different files")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func sameFilePath(left, right string) (bool, error) {
	leftPath, err := filepath.Abs(left)
	if err != nil {
		return false, err
	}
	rightPath, err := filepath.Abs(right)
	if err != nil {
		return false, err
	}
	cleanLeft := filepath.Clean(leftPath)
	cleanRight := filepath.Clean(rightPath)
	if goruntime.GOOS == "windows" {
		return strings.EqualFold(cleanLeft, cleanRight), nil
	}
	return cleanLeft == cleanRight, nil
}
