package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// SystemController handles system-level endpoints: dashboard, backup/restore, bootstrap.
type SystemController struct {
	appCfg config.App
	appCtx context.Context
}

func (c *SystemController) db() *gorm.DB { return dbpkg.GetDB() }

// SetContext is called from App.startup once the Wails runtime context is available.
func (c *SystemController) SetContext(ctx context.Context) { c.appCtx = ctx }

func (c *SystemController) Bootstrap() BootstrapPayload {
	return BootstrapPayload{Name: c.appCfg.Name, Version: c.appCfg.Version, Module: c.appCfg.Module, Description: c.appCfg.Description, Runtime: goruntime.Version(), Frontend: c.appCfg.FrontendRuntime, Highlights: []string{"Wave workflow", "Platform isolation", "SQLite backup and restore"}}
}

func (c *SystemController) PingDB() string {
	db := c.db()
	if db == nil {
		return "database not available"
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		return fmt.Sprintf("database probe failed: %v", err)
	}
	return "SQLite database connection is healthy"
}

func (c *SystemController) GetDashboard() (DashboardPayload, error) {
	db := c.db()
	if db == nil {
		return DashboardPayload{}, fmt.Errorf("database not available")
	}
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
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if !strings.HasPrefix(execDir, os.TempDir()) {
			return filepath.Join(execDir, "data", "eligiftmanager.db"), nil
		}
	}
	workDir, workDirErr := os.Getwd()
	if workDirErr != nil {
		return "", fmt.Errorf("resolve database path failed: %w", workDirErr)
	}
	return filepath.Join(workDir, "data", "eligiftmanager.db"), nil
}
