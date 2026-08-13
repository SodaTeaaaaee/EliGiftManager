package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsService_SaveWritesValidJSONAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "settings.json")
	svc := &SettingsService{filePath: dest}

	want := &SystemSettings{
		AutoMergeCrossPlatform: true,
		AutoMergeByEmail:       true,
		AutoMergeByPhone:       false,
	}
	if err := svc.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("settings.json is not valid JSON: %s", raw)
	}

	got, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}

	assertNoSettingsTmp(t, dir)
}

func TestSettingsService_SaveOverwritesExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(dest, []byte(`{"autoMergeByPhone":true}`), 0o644); err != nil {
		t.Fatalf("seed dest: %v", err)
	}

	svc := &SettingsService{filePath: dest}
	want := &SystemSettings{AutoMergeCrossPlatform: true}
	if err := svc.Save(want); err != nil {
		t.Fatalf("Save overwrite: %v", err)
	}

	got, err := svc.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
	assertNoSettingsTmp(t, dir)
}

func TestWriteFileAtomic_SameDirTempThenRename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "settings.json")
	payload := []byte("{\n  \"autoMergeByEmail\": true\n}")

	if err := writeFileAtomic(dest, payload, 0o644); err != nil {
		t.Fatalf("writeFileAtomic: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("dest = %q, want %q", got, payload)
	}
	if !json.Valid(got) {
		t.Fatalf("dest is not valid JSON: %s", got)
	}
	assertNoSettingsTmp(t, dir)

	// Second write must replace in place (Windows rename-over-existing).
	next := []byte("{\n  \"autoMergeByPhone\": true\n}")
	if err := writeFileAtomic(dest, next, 0o644); err != nil {
		t.Fatalf("writeFileAtomic replace: %v", err)
	}
	got, err = os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest after replace: %v", err)
	}
	if string(got) != string(next) {
		t.Fatalf("replaced dest = %q, want %q", got, next)
	}
	assertNoSettingsTmp(t, dir)
}

func TestSettingsService_SaveNilRejected(t *testing.T) {
	t.Parallel()

	svc := &SettingsService{filePath: filepath.Join(t.TempDir(), "settings.json")}
	if err := svc.Save(nil); err == nil {
		t.Fatal("expected error saving nil settings")
	}
}

func assertNoSettingsTmp(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".settings-") && strings.HasSuffix(name, ".tmp") {
			t.Fatalf("leftover temp file after atomic save: %s", name)
		}
	}
}
