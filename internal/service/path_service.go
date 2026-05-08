package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	appDataDirName            = "EliGiftManager"
	dataDirName               = "data"
	assetsDirName             = "assets"
	tempDirName               = "tmp"
	currentImportTempPrefix   = "eligift-product-zip-"
	legacyImportTempPrefix    = "eligift-product-archive-"
	staleTempCleanupThreshold = 30 * time.Minute
)

// ResolveDataDir 三级单选：
// 1. Wails dev 环境变量存在 → 工作目录/data
// 2. exe 同级有 .portable 占位文件 → exe/data（便携模式）
// 3. 兜底 → os.UserConfigDir()/EliGiftManager/data（系统安装）
//
// 目录不存在时自动 MkdirAll。
func ResolveDataDir() (string, error) {
	dir, err := resolveDataDirCandidate()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve data dir: mkdir %q: %w", dir, err)
	}
	return dir, nil
}

func resolveDataDirCandidate() (string, error) {
	if isWailsDevMode() {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve data dir: %w", err)
		}
		return filepath.Join(wd, dataDirName), nil
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if _, statErr := os.Stat(filepath.Join(execDir, ".portable")); statErr == nil {
			return filepath.Join(execDir, dataDirName), nil
		}
	}

	uc, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return filepath.Join(uc, appDataDirName, dataDirName), nil
}

func isWailsDevMode() bool {
	return os.Getenv("devserver") != "" || os.Getenv("frontenddevserverurl") != ""
}

// ResolveAssetsDir 返回 data/assets/ 目录。
func ResolveAssetsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, assetsDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve assets dir: %w", err)
	}
	return dir, nil
}

// ResolveTempDir returns the app-managed temporary directory under data/tmp.
func ResolveTempDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, tempDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("resolve temp dir: %w", err)
	}
	return dir, nil
}

// CreateImportTempDir allocates a managed temporary directory for ZIP extraction.
func CreateImportTempDir() (string, error) {
	tempDir, err := ResolveTempDir()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(tempDir, currentImportTempPrefix)
	if err != nil {
		return "", fmt.Errorf("create import temp dir: %w", err)
	}
	return dir, nil
}

// CleanupStaleTempArtifacts removes stale temporary import directories from the
// managed app temp dir and the legacy OS temp root used by older builds.
func CleanupStaleTempArtifacts() error {
	var errs []error
	managedTempDir, err := ResolveTempDir()
	if err == nil {
		if cleanupErr := cleanupTempEntries(managedTempDir, []string{currentImportTempPrefix}, staleTempCleanupThreshold, time.Now()); cleanupErr != nil {
			errs = append(errs, cleanupErr)
		}
	} else {
		errs = append(errs, err)
	}

	if cleanupErr := cleanupTempEntries(os.TempDir(), []string{currentImportTempPrefix, legacyImportTempPrefix}, staleTempCleanupThreshold, time.Now()); cleanupErr != nil {
		errs = append(errs, cleanupErr)
	}

	if len(errs) == 0 {
		return nil
	}

	msg := "cleanup stale temp artifacts failed"
	for _, err := range errs {
		msg += ": " + err.Error()
	}
	return fmt.Errorf("%s", msg)
}

func cleanupTempEntries(root string, prefixes []string, olderThan time.Duration, now time.Time) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir %q: %w", root, err)
	}

	var errs []error
	for _, entry := range entries {
		name := entry.Name()
		if !hasAnyPrefix(name, prefixes) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			errs = append(errs, fmt.Errorf("stat %q: %w", filepath.Join(root, name), err))
			continue
		}
		if now.Sub(info.ModTime()) < olderThan {
			continue
		}

		target := filepath.Join(root, name)
		if info.IsDir() {
			if err := os.RemoveAll(target); err != nil {
				errs = append(errs, fmt.Errorf("remove dir %q: %w", target, err))
			}
			continue
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove file %q: %w", target, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	msg := "cleanup temp entries failed"
	for _, err := range errs {
		msg += ": " + err.Error()
	}
	return fmt.Errorf("%s", msg)
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if len(prefix) > 0 && len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
