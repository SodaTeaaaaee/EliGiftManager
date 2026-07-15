package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	setAppContext(ctx)
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

// PickTabularFile opens a native dialog for CSV / XLSX / XLS spreadsheet files.
func (a *App) PickTabularFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择表格文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Tabular Files", Pattern: "*.csv;*.xlsx;*.xls"},
			{DisplayName: "CSV Files", Pattern: "*.csv"},
			{DisplayName: "Excel Files", Pattern: "*.xlsx;*.xls"},
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

// PickCatalogImportFile opens a native dialog for product-catalog imports:
// ZIP (with images) or tabular CSV / XLSX / XLS.
func (a *App) PickCatalogImportFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择商品目录文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Catalog Files", Pattern: "*.zip;*.csv;*.xlsx;*.xls"},
			{DisplayName: "ZIP Files", Pattern: "*.zip"},
			{DisplayName: "Tabular Files", Pattern: "*.csv;*.xlsx;*.xls"},
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
