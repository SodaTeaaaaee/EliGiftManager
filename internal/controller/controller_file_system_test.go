package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevealInFolder_RejectsPathsOutsideDataDir(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	c := &FileSystemController{
		resolveDataDir: func() (string, error) { return dataDir, nil },
		startReveal: func(goos, absPath string) error {
			t.Fatalf("must not start file manager for outside path %q", absPath)
			return nil
		},
	}

	err := c.RevealInFolder(outside)
	if err == nil {
		t.Fatal("expected RevealInFolder to reject a path outside the data dir")
	}
	if !strings.Contains(err.Error(), "outside the data directory") {
		t.Fatalf("error = %v, want outside-the-data-directory", err)
	}
}

func TestRevealInFolder_RejectsDotDotAndUNC(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	dataDir := filepath.Join(parent, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	outside := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	c := &FileSystemController{
		resolveDataDir: func() (string, error) { return dataDir, nil },
		startReveal: func(goos, absPath string) error {
			t.Fatalf("must not start file manager for rejected path %q", absPath)
			return nil
		},
	}

	dotDot := dataDir + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "secret.txt"
	if err := c.RevealInFolder(dotDot); err == nil {
		t.Fatal("expected .. escape to be rejected")
	}

	for _, p := range []string{
		`\\server\share\file.txt`,
		`//server/share/file.txt`,
	} {
		if err := c.RevealInFolder(p); err == nil {
			t.Errorf("expected UNC %q to be rejected", p)
		}
	}
}

func TestRevealInFolder_AllowsPathUnderDataDir(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	inside := filepath.Join(dataDir, "tmp", "catalog-import", "sheet.csv")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(inside, []byte("sku\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var started string
	c := &FileSystemController{
		resolveDataDir: func() (string, error) { return dataDir, nil },
		startReveal: func(goos, absPath string) error {
			started = absPath
			return nil
		},
	}
	if err := c.RevealInFolder(inside); err != nil {
		t.Fatalf("RevealInFolder inside data dir: %v", err)
	}
	if started == "" {
		t.Fatal("expected file manager to be started for a confined path")
	}
	absInside, err := filepath.Abs(inside)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if started != filepath.Clean(absInside) {
		t.Fatalf("started %q, want %q", started, absInside)
	}
}

func TestRevealCommand_WindowsSelectUsesSeparateArgs(t *testing.T) {
	t.Parallel()

	path := `C:\Users\test\EliGiftManager\data\exports\order,1.json`
	cmd := revealCommand("windows", path)
	if cmd == nil {
		t.Fatal("expected explorer command")
	}
	if len(cmd.Args) < 3 {
		t.Fatalf("args = %v, want /select, and path as separate argv", cmd.Args)
	}
	if cmd.Args[1] != "/select," {
		t.Fatalf("args[1] = %q, want %q", cmd.Args[1], "/select,")
	}
	if cmd.Args[2] != path {
		t.Fatalf("args[2] = %q, want %q", cmd.Args[2], path)
	}
	for _, a := range cmd.Args {
		if strings.HasPrefix(a, "/select,") && a != "/select," {
			t.Fatalf("path was concatenated into the select flag: %q", a)
		}
	}
}
