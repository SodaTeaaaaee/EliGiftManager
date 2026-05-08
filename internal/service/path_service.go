package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ResolveDataDir returns the single authoritative data directory.
//
// Priority (first match wins):
//  1. ELIGIFT_DATA_DIR env var — explicit override
//  2. .portable file next to executable → exe_dir/data (portable mode)
//  3. Executable lives under system temp → wd/data (dev mode)
//  4. Default → UserConfigDir/EliGiftManager/data (system install)
//
// In dev mode (3) the working directory is the project root; assets, database,
// and zoom.cfg are all stored under wd/data/ so that nothing leaks to C:.
func ResolveDataDir() (string, error) {
	// 1. Explicit override via environment variable.
	if env := strings.TrimSpace(os.Getenv("ELIGIFT_DATA_DIR")); env != "" {
		return ensureDir(env)
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)

		// 2. .portable marker — data lives next to the executable.
		if _, statErr := os.Stat(filepath.Join(execDir, ".portable")); statErr == nil {
			return ensureDir(filepath.Join(execDir, "data"))
		}

		tempDir := os.TempDir()

		// 3. Dev mode — executable was built into a temp directory by Wails.
		if isSubDir(execDir, tempDir) {
			wd, wdErr := os.Getwd()
			if wdErr != nil {
				return "", fmt.Errorf("resolve data dir (dev): %w", wdErr)
			}
			return ensureDir(filepath.Join(wd, "data"))
		}

		// 4. System install.
		uc, ucErr := os.UserConfigDir()
		if ucErr != nil {
			return "", fmt.Errorf("resolve data dir: %w", ucErr)
		}
		return ensureDir(filepath.Join(uc, "EliGiftManager", "data"))
	}

	// os.Executable failed — fall back to working directory.
	wd, wdErr := os.Getwd()
	if wdErr != nil {
		return "", fmt.Errorf("resolve data dir: %w", wdErr)
	}
	return ensureDir(filepath.Join(wd, "data"))
}

// ResolveAssetsDir returns the data/assets/ directory.
func ResolveAssetsDir() (string, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("resolve assets dir: %w", err)
	}
	return dir, nil
}

// isSubDir reports whether sub is equal to or a descendant of parent.
// Comparison is case-insensitive to handle Windows path casing inconsistencies.
func isSubDir(sub, parent string) bool {
	sub = filepath.Clean(sub)
	parent = filepath.Clean(parent)
	sl := strings.ToLower(sub)
	pl := strings.ToLower(parent)
	return sl == pl || strings.HasPrefix(sl, pl+strings.ToLower(string(filepath.Separator)))
}

func ensureDir(path string) (string, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("resolve data dir: mkdir %q: %w", path, err)
	}
	return path, nil
}

// CleanupTempDirs removes eligift-product-zip-* temporary directories that are
// older than 1 hour.  Called once at app startup to prevent stale unpacked ZIP
// directories from accumulating in the system temp folder.
func CleanupTempDirs() {
	dirs, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-1 * time.Hour)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		if !strings.HasPrefix(d.Name(), "eligift-product-zip-") {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.RemoveAll(filepath.Join(os.TempDir(), d.Name()))
		}
	}
}
