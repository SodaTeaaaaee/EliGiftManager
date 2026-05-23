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
	mu sync.RWMutex
}

func NewSettingsService() *SettingsService {
	return &SettingsService{}
}

func (s *SettingsService) settingsFilePath() (string, error) {
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

	return os.WriteFile(p, b, 0o644)
}
