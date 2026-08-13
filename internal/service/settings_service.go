package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type SystemSettings struct {
	AutoMergeCrossPlatform bool `json:"autoMergeCrossPlatform"`
	AutoMergeByEmail       bool `json:"autoMergeByEmail"`
	AutoMergeByPhone       bool `json:"autoMergeByPhone"`
}

type SettingsService struct {
	mu       sync.RWMutex
	filePath string // optional test override; empty uses ResolveDataDir
}

func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

func (s *SettingsService) settingsFilePath() (string, error) {
	if s != nil && s.filePath != "" {
		return s.filePath, nil
	}
	dataDir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "settings.json"), nil
}

func (s *SettingsService) Load() (*SystemSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, err := s.settingsFilePath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return &SystemSettings{
			AutoMergeCrossPlatform: false,
			AutoMergeByEmail:       false,
			AutoMergeByPhone:       false,
		}, nil
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var settings SystemSettings
	if err := json.Unmarshal(b, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *SettingsService) Save(settings *SystemSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if settings == nil {
		return errors.New("cannot save nil settings")
	}

	p, err := s.settingsFilePath()
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return writeFileAtomic(p, b, 0o644)
}

// writeFileAtomic writes data to path by creating a temp file in the same
// directory, fsyncing it, then renaming into place so a crash cannot leave
// a truncated dest file. On Windows, rename-over-existing falls back to
// remove-then-rename if needed.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	closed := false
	success := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if perm != 0 {
		_ = f.Chmod(perm)
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		closed = true
		return err
	}
	closed = true

	if err := renameOver(tmpName, path); err != nil {
		return err
	}
	success = true
	return nil
}

func renameOver(tmpName, dest string) error {
	if err := os.Rename(tmpName, dest); err == nil {
		return nil
	} else if rmErr := os.Remove(dest); rmErr != nil && !os.IsNotExist(rmErr) {
		return err
	}
	return os.Rename(tmpName, dest)
}
