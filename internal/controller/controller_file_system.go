package controller

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/pathconfine"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// FileSystemController exposes a single "reveal a file in its containing
// folder" Wails binding. Self-contained: pure OS call, no repo/DB dependency,
// mirroring the other single-purpose controllers (e.g. ActionCenterController).
type FileSystemController struct {
	resolveDataDir func() (string, error)
	startReveal    func(goos, absPath string) error
}

func NewFileSystemController() *FileSystemController {
	return &FileSystemController{}
}

func (c *FileSystemController) resolvedDataDir() (string, error) {
	if c != nil && c.resolveDataDir != nil {
		return c.resolveDataDir()
	}
	return service.ResolveDataDir()
}

// RevealInFolder opens the OS file manager with the given file selected (or,
// on platforms without file-select support, opens its containing directory).
// Only paths under the application data directory are allowed (catalog-import
// extracts live at data/tmp/catalog-import and are covered by that root).
func (c *FileSystemController) RevealInFolder(path string) error {
	if path == "" {
		return fmt.Errorf("reveal in folder: path is empty")
	}

	dataDir, err := c.resolvedDataDir()
	if err != nil {
		return fmt.Errorf("reveal in folder: resolve data dir: %w", err)
	}

	if pathconfine.HasDotDot(path) {
		return fmt.Errorf("reveal in folder: path is outside the data directory")
	}
	absPath, err := pathconfine.Confine(dataDir, path)
	if err != nil {
		return fmt.Errorf("reveal in folder: path is outside the data directory: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("reveal in folder: stat %q: %w", absPath, err)
	}

	start := c.startReveal
	if start == nil {
		start = startRevealProcess
	}
	return start(runtime.GOOS, absPath)
}

func startRevealProcess(goos, absPath string) error {
	cmd := revealCommand(goos, absPath)
	if cmd == nil {
		return fmt.Errorf("reveal in folder: unsupported OS %q", goos)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reveal in folder: start %s: %w", cmd.Args[0], err)
	}
	return nil
}

// revealCommand builds a file-manager invocation. On Windows, /select, and the
// path are separate argv entries so a comma in the filename cannot be parsed
// as an extra explorer switch, and the path is never concatenated into a
// shell command string.
func revealCommand(goos, absPath string) *exec.Cmd {
	switch goos {
	case "windows":
		// NOTE: Windows explorer.exe returns exit code 1 even on success, so
		// callers must use Start() (fire-and-forget) rather than Run().
		return exec.Command("explorer", "/select,", absPath)
	case "darwin":
		return exec.Command("open", "-R", absPath)
	default:
		target := absPath
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			target = filepath.Dir(absPath)
		}
		return exec.Command("xdg-open", target)
	}
}
