package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// SystemController handles system-level endpoints: dashboard, backup/restore, bootstrap.
type SystemController struct {
	appCfg config.App
	appCtx context.Context
	db     *gorm.DB
}

// SetContext is called from App.startup once the Wails runtime context is available.
func (c *SystemController) SetContext(ctx context.Context) { c.appCtx = ctx }

func (c *SystemController) Bootstrap() BootstrapPayload {
	return BootstrapPayload{Name: c.appCfg.Name, Version: c.appCfg.Version, Module: c.appCfg.Module, Description: c.appCfg.Description, Runtime: goruntime.Version(), Frontend: c.appCfg.FrontendRuntime, Highlights: []string{"Wave workflow", "Platform isolation", "SQLite backup and restore"}}
}

func (c *SystemController) PingDB() string {
	db := c.db
	if err := db.Exec("SELECT 1").Error; err != nil {
		return fmt.Sprintf("database probe failed: %v", err)
	}
	return "SQLite database connection is healthy"
}

func (c *SystemController) GetDashboard() (DashboardPayload, error) {
	db := c.db
	dbPath, _ := appDatabasePath()
	payload := DashboardPayload{DatabasePath: dbPath}
	if err := db.Model(&model.Member{}).Count(&payload.MemberCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.Product{}).Count(&payload.ProductCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.DispatchRecord{}).Count(&payload.DispatchCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.TemplateConfig{}).Count(&payload.TemplateCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.Wave{}).Count(&payload.WaveCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.MemberAddress{}).Where("is_deleted = ?", false).Count(&payload.AddressCount).Error; err != nil {
		return payload, err
	}
	if err := db.Model(&model.DispatchRecord{}).Where("status = ?", model.DispatchStatusPendingAddress).Count(&payload.PendingAddresses).Error; err != nil {
		return payload, err
	}
	active := db.Model(&model.MemberAddress{}).Select("member_id").Where("is_deleted = ?", false)
	if err := db.Model(&model.Member{}).Where("id NOT IN (?)", active).Count(&payload.MissingAddresses).Error; err != nil {
		return payload, err
	}
	var err error
	payload.RecentWaves, err = queryWaves(db, 8, "")
	if err != nil {
		return payload, err
	}
	payload.RecentDispatches, err = queryDispatchRecords(db, 0, 8)
	if err != nil {
		return payload, err
	}
	payload.Warnings = buildDashboardWarnings(payload)
	return payload, nil
}

func (c *SystemController) BackupDatabase() (string, error) {
	dbPath, err := appDatabasePath()
	if err != nil {
		return "", err
	}
	target := filepath.Join(os.TempDir(), "eligiftmanager-backup.db")
	if c.appCtx != nil {
		selected, dialogErr := wailsruntime.SaveFileDialog(c.appCtx, wailsruntime.SaveDialogOptions{DefaultFilename: fmt.Sprintf("eligiftmanager-%s.db", time.Now().Format("20060102150405"))})
		if dialogErr != nil {
			return "", dialogErr
		}
		if selected == "" {
			return "", fmt.Errorf("backup canceled")
		}
		target = selected
	}
	if err := copyFile(dbPath, target); err != nil {
		return "", err
	}
	return target, nil
}

func (c *SystemController) RestoreDatabase() error {
	dbPath, err := appDatabasePath()
	if err != nil {
		return err
	}
	if c.appCtx == nil {
		return fmt.Errorf("Wails runtime is required")
	}
	source, err := wailsruntime.OpenFileDialog(c.appCtx, wailsruntime.OpenDialogOptions{Title: "Select EliGiftManager backup database"})
	if err != nil {
		return err
	}
	if source == "" {
		return fmt.Errorf("restore canceled")
	}
	sameFile, err := sameFilePath(source, dbPath)
	if err != nil {
		return err
	}
	if sameFile {
		return fmt.Errorf("restore source must be different from the active database")
	}
	if err := validateDatabaseFile(source); err != nil {
		return err
	}
	backupPath := dbPath + ".before-restore-" + time.Now().Format("20060102150405")
	if _, statErr := os.Stat(dbPath); statErr == nil {
		if err := copyFile(dbPath, backupPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	if err := copyFile(source, dbPath); err != nil {
		return err
	}
	return validateDatabaseFile(dbPath)
}

// appDatabasePath resolves the database file path for controller use.
func appDatabasePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "eligiftmanager.db"), nil
}

func zoomFilePath() (string, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "zoom.cfg"), nil
}

// SaveZoom persists the UI zoom percentage to a config file.
func (c *SystemController) SaveZoom(percent float64) error {
	path, err := zoomFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.FormatFloat(percent, 'f', 1, 64)), 0o644)
}

// LoadZoom reads the saved zoom percentage. Returns 100 if no saved value.
func LoadZoom() float64 {
	path, err := zoomFilePath()
	if err != nil {
		return 100
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 100
	}
	v, err := strconv.ParseFloat(string(data), 64)
	if err != nil || v < 25 || v > 500 {
		return 100
	}
	return v
}
