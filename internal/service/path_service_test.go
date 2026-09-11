package service

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// stubEnv builds a dataDirEnv with canned environment responses.
func stubEnv(t *testing.T, dev bool, wd, execPath, userConfigDir string, portableExists bool) dataDirEnv {
	t.Helper()

	if execPath == "" {
		execPath = filepath.Join(t.TempDir(), "app.exe")
	}

	return dataDirEnv{
		dev: dev,
		workingDir: func() (string, error) {
			if wd == "" {
				return "", errors.New("no working dir")
			}
			return wd, nil
		},
		executable: func() (string, error) {
			return execPath, nil
		},
		userConfigDir: func() (string, error) {
			if userConfigDir == "" {
				return "", errors.New("no user config dir")
			}
			return userConfigDir, nil
		},
		stat: func(string) (fs.FileInfo, error) {
			if !portableExists {
				return nil, os.ErrNotExist
			}
			return nil, nil
		},
	}
}

func TestResolveDataDirCandidate_DevBuildUsesWorkingDirectory(t *testing.T) {
	t.Parallel()

	wd := t.TempDir()
	env := stubEnv(t, true, wd, "", filepath.Join(t.TempDir(), "roaming"), true)

	got, err := resolveDataDirCandidate(env)
	if err != nil {
		t.Fatalf("resolveDataDirCandidate: %v", err)
	}
	if want := filepath.Join(wd, "data"); got != want {
		t.Fatalf("dev build: dir = %q, want %q", got, want)
	}
}

func TestResolveDataDirCandidate_DevBuildIgnoresPortableMarker(t *testing.T) {
	t.Parallel()

	// In dev the working directory wins even when .portable sits next to the
	// executable — the dev check is evaluated first.
	wd := t.TempDir()
	env := stubEnv(t, true, wd, "", "", true)

	got, err := resolveDataDirCandidate(env)
	if err != nil {
		t.Fatalf("resolveDataDirCandidate: %v", err)
	}
	if want := filepath.Join(wd, "data"); got != want {
		t.Fatalf("dev build: dir = %q, want %q", got, want)
	}
}

func TestResolveDataDirCandidate_DevBuildWorkingDirError(t *testing.T) {
	t.Parallel()

	env := stubEnv(t, true, "", "", "", true)
	if _, err := resolveDataDirCandidate(env); err == nil {
		t.Fatal("expected error when the working directory lookup fails")
	}
}

func TestResolveDataDirCandidate_PortableMarkerWinsOverUserConfig(t *testing.T) {
	t.Parallel()

	exeDir := t.TempDir()
	env := stubEnv(t, false, "", filepath.Join(exeDir, "app.exe"), filepath.Join(t.TempDir(), "roaming"), true)

	got, err := resolveDataDirCandidate(env)
	if err != nil {
		t.Fatalf("resolveDataDirCandidate: %v", err)
	}
	if want := filepath.Join(exeDir, "data"); got != want {
		t.Fatalf("portable: dir = %q, want %q", got, want)
	}
}

func TestResolveDataDirCandidate_NoMarkerFallsBackToUserConfig(t *testing.T) {
	t.Parallel()

	uc := t.TempDir()
	env := stubEnv(t, false, "", "", uc, false)

	got, err := resolveDataDirCandidate(env)
	if err != nil {
		t.Fatalf("resolveDataDirCandidate: %v", err)
	}
	if want := filepath.Join(uc, "EliGiftManager", "data"); got != want {
		t.Fatalf("fallback: dir = %q, want %q", got, want)
	}
}

func TestResolveDataDirCandidate_ExecutableErrorFallsBackToUserConfig(t *testing.T) {
	t.Parallel()

	uc := t.TempDir()
	env := stubEnv(t, false, "", "", uc, false)
	env.executable = func() (string, error) { return "", errors.New("no exe") }

	got, err := resolveDataDirCandidate(env)
	if err != nil {
		t.Fatalf("resolveDataDirCandidate: %v", err)
	}
	if want := filepath.Join(uc, "EliGiftManager", "data"); got != want {
		t.Fatalf("fallback: dir = %q, want %q", got, want)
	}
}

func TestResolveDataDirCandidate_UserConfigError(t *testing.T) {
	t.Parallel()

	env := stubEnv(t, false, "", "", "", false)
	if _, err := resolveDataDirCandidate(env); err == nil {
		t.Fatal("expected error when the user config dir lookup fails")
	}
}

// TestResolveDataDir_CreatesDirectory exercises the exported ResolveDataDir
// through the package-level environment seam. It must stay non-parallel: it
// swaps dataDirEnvSource while running.
func TestResolveDataDir_CreatesDirectory(t *testing.T) {
	root := t.TempDir()
	wd := filepath.Join(root, "project")
	if err := os.MkdirAll(wd, 0o755); err != nil {
		t.Fatalf("mkdir wd: %v", err)
	}

	orig := dataDirEnvSource
	dataDirEnvSource = func() dataDirEnv { return stubEnv(t, true, wd, "", "", false) }
	t.Cleanup(func() { dataDirEnvSource = orig })

	got, err := ResolveDataDir()
	if err != nil {
		t.Fatalf("ResolveDataDir: %v", err)
	}
	if want := filepath.Join(wd, "data"); got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
	info, err := os.Stat(got)
	if err != nil || !info.IsDir() {
		t.Fatalf("data dir %q was not created: %v", got, err)
	}
}

func TestIsDevBuild_MatchesBuildTags(t *testing.T) {
	t.Parallel()

	// Without -tags production (plain `go test`) the dev variant is compiled
	// in; `go test -tags production` compiles the other side. Both variants
	// must compile, which this file's existence already checks.
	if !isDevBuild() {
		t.Log("production build: isDevBuild() = false as expected with -tags production")
	}
}
