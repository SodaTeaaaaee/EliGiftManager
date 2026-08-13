package controller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FileSystemController exposes a single "reveal a file in its containing
// folder" Wails binding. Self-contained: pure OS call, no repo/DB dependency,
// mirroring the other single-purpose controllers (e.g. ActionCenterController).
type FileSystemController struct{}

func NewFileSystemController() *FileSystemController {
	return &FileSystemController{}
}

// RevealInFolder opens the OS file manager with the given file selected (or,
// on platforms without file-select support, opens its containing directory).
// The path always originates from our own generated output files (supplier
// order files, backfill files), never arbitrary user input, but is validated
// with os.Stat before any external process is spawned as a security guard.
func (c *FileSystemController) RevealInFolder(path string) error {
	if path == "" {
		return fmt.Errorf("reveal in folder: path is empty")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("reveal in folder: stat %q: %w", path, err)
	}

	switch runtime.GOOS {
	case "windows":
		// NOTE: Windows explorer.exe returns exit code 1 even on success, so
		// we must use Start() (fire-and-forget) rather than Run() and must
		// not treat a later non-zero exit as failure.
		if err := exec.Command("explorer", "/select,"+path).Start(); err != nil {
			return fmt.Errorf("reveal in folder: start explorer: %w", err)
		}
		return nil
	case "darwin":
		if err := exec.Command("open", "-R", path).Start(); err != nil {
			return fmt.Errorf("reveal in folder: start open: %w", err)
		}
		return nil
	default:
		// xdg-open cannot select a file, so open its containing directory.
		if err := exec.Command("xdg-open", filepath.Dir(path)).Start(); err != nil {
			return fmt.Errorf("reveal in folder: start xdg-open: %w", err)
		}
		return nil
	}
}
