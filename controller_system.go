package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/config"
	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
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

// CreateConsistentDatabaseSnapshot produces a transactionally-consistent copy of the
// database at targetPath using SQLite's VACUUM INTO.  The target must not already exist.
// This is the single approved helper for all online backups — BackupDatabase and the
// RestoreDatabase pre-restore safety backup MUST both go through this function.
func CreateConsistentDatabaseSnapshot(db *gorm.DB, targetPath string) error {
	if db == nil {
		return fmt.Errorf("snapshot failed: database not available")
	}
	// VACUUM INTO requires the target file to not exist.
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("snapshot failed: remove existing target %q: %w", targetPath, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("snapshot failed: get underlying connection: %w", err)
	}
	// SQLite does not support parameterised VACUUM INTO, so we interpolate the
	// path with SQL-safe single-quote escaping.
	escaped := strings.ReplaceAll(targetPath, "'", "''")
	if _, err := sqlDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", escaped)); err != nil {
		return fmt.Errorf("snapshot failed: VACUUM INTO: %w", err)
	}
	// Verify the output file exists and is readable.
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("snapshot file not found after VACUUM INTO: %w", err)
	}
	return nil
}

func (c *SystemController) BackupDatabase() (string, error) {
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

	currentDB := c.db()
	if currentDB == nil {
		return "", fmt.Errorf("backup failed: database not available")
	}
	if err := CreateConsistentDatabaseSnapshot(currentDB, target); err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}
	return target, nil
}

func (c *SystemController) RestoreDatabase() (err error) {
	dbPath, dbErr := appDatabasePath()
	if dbErr != nil {
		return dbErr
	}
	if c.appCtx == nil {
		return fmt.Errorf("Wails runtime is required")
	}
	source, dlgErr := wailsruntime.OpenFileDialog(c.appCtx, wailsruntime.OpenDialogOptions{Title: "Select EliGiftManager backup database"})
	if dlgErr != nil {
		return dlgErr
	}
	if source == "" {
		return fmt.Errorf("restore canceled")
	}
	sameFile, sfErr := sameFilePath(source, dbPath)
	if sfErr != nil {
		return sfErr
	}
	if sameFile {
		return fmt.Errorf("restore source must be different from the active database")
	}

	return restoreDatabaseFromSource(dbPath, source)
}

func restoreDatabaseFromSource(dbPath, source string) (err error) {
	currentDB := dbpkg.GetDB()
	if currentDB == nil {
		return fmt.Errorf("restore failed: database not available")
	}

	// 1. Validate the restore source before touching anything.
	if err = validateDatabaseFile(source); err != nil {
		return err
	}

	// 2. Create a consistent safety snapshot of the current database (VACUUM INTO,
	//    NOT a raw file copy — the latter is not WAL-safe).
	backupPath := dbPath + ".before-restore-" + time.Now().Format("20060102150405")
	if _, statErr := os.Stat(dbPath); statErr == nil {
		if err = CreateConsistentDatabaseSnapshot(currentDB, backupPath); err != nil {
			return fmt.Errorf("restore failed: safety backup: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}

	// 3. Copy restore source to a temporary file next to the database.
	tmpPath := dbPath + ".restore-tmp"
	if err = copyFile(source, tmpPath); err != nil {
		return fmt.Errorf("restore failed: copy source to temp: %w", err)
	}

	// 4. Validate the temporary file before swapping it in.
	if err = validateDatabaseFile(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("restore failed: validate temp file: %w", err)
	}

	// 5. Close the current sql.DB connection so no file locks remain.
	sqlDB, sqlErr := currentDB.DB()
	if sqlErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("restore failed: get underlying sql.DB: %w", sqlErr)
	}
	if closeErr := sqlDB.Close(); closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("restore failed: close current database: %w", closeErr)
	}
	dbpkg.SetDefaultDB(nil)
	closed := true

	// State tracking for the deferred rollback / recovery.
	var oldRenamed bool
	rollbackPath := dbPath + ".rollback"

	// Deferred recovery: guarantees that when this function returns (success or
	// failure), dbpkg.GetDB() always points to a usable open connection (D9).
	defer func() {
		if err != nil {
			// --- Failure path: attempt rollback ---
			if oldRenamed {
				// The old database was renamed away — try to restore it.
				os.Remove(dbPath)
				os.Remove(dbPath + "-wal")
				os.Remove(dbPath + "-shm")
				if renameErr := os.Rename(rollbackPath, dbPath); renameErr == nil {
					if newDB, initErr := dbpkg.InitDB(dbPath); initErr == nil {
						dbpkg.SetDefaultDB(newDB)
						closed = false
						os.Remove(tmpPath)
						return
					}
				}
				// Rollback rename/init failed.  Last resort: try to re-open
				// whatever is at dbPath so defaultDB is not left nil (D9).
				if dbpkg.GetDB() == nil {
					if newDB, initErr := dbpkg.InitDB(dbPath); initErr == nil {
						dbpkg.SetDefaultDB(newDB)
						closed = false
					}
				}
				err = fmt.Errorf("restore failed and automatic rollback incomplete; original database saved at %s: %w", backupPath, err)
				os.Remove(tmpPath)
				return
			}
			// oldRenamed is false: the original DB file is still at dbPath.
			// Connection was closed — re-open it so defaultDB stays usable (D9).
			if closed && dbpkg.GetDB() == nil {
				if newDB, initErr := dbpkg.InitDB(dbPath); initErr == nil {
					dbpkg.SetDefaultDB(newDB)
					closed = false
				}
			}
			os.Remove(tmpPath)
			return
		}
		// --- Success path: clean up temp files ---
		// Keep backupPath as an extra safety copy; user can delete it manually.
		os.Remove(rollbackPath)
		os.Remove(tmpPath)
	}()

	// 6. Rename current database to rollback path (preserve for rollback).
	if err = os.Rename(dbPath, rollbackPath); err != nil {
		return fmt.Errorf("restore failed: rename current database: %w", err)
	}
	oldRenamed = true

	// 7. Remove WAL/SHM files that may be left from the old connection.
	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")

	// 8. Swap the validated temporary file into the formal database location.
	if err = os.Rename(tmpPath, dbPath); err != nil {
		return fmt.Errorf("restore failed: rename temp to database: %w", err)
	}

	// 9. Re-initialise the database (opens fresh connection, applies PRAGMAs, migrates schema).
	newDB, initErr := dbpkg.InitDB(dbPath)
	if initErr != nil {
		return fmt.Errorf("restore failed: re-initialise database: %w", initErr)
	}

	// 10. Switch the global singleton to the new connection.
	dbpkg.SetDefaultDB(newDB)
	closed = false

	// 11. Verify the new connection is healthy with a simple probe.
	if err = newDB.Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("restore failed: verify new connection: %w", err)
	}

	return nil
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

// CreateFakeAddresses generates a fake address for every member that does not
// currently have a valid address. Returns counts of how many were created.
func (c *SystemController) CreateFakeAddresses() (service.CreateFakeAddressesResult, error) {
	db := c.db()
	if db == nil {
		return service.CreateFakeAddressesResult{}, fmt.Errorf("database not available")
	}
	return service.CreateFakeAddressesForAllMembers(db)
}

// DeleteFakeAddresses removes all system-generated test addresses, clears their
// dispatch record bindings, and resets affected wave statuses.
func (c *SystemController) DeleteFakeAddresses() (service.DeleteFakeAddressesResult, error) {
	db := c.db()
	if db == nil {
		return service.DeleteFakeAddressesResult{}, fmt.Errorf("database not available")
	}
	return service.DeleteFakeAddressesForAllMembers(db)
}
