package service

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PresetInfo is the lightweight summary of a preset template.
type PresetInfo struct {
	ID       string `json:"id"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
	Name     string `json:"name"`
}

// PresetContent is the full preset template, including mapping rules as a JSON object.
type PresetContent struct {
	ID           string          `json:"id"`
	Platform     string          `json:"platform"`
	Type         string          `json:"type"`
	Name         string          `json:"name"`
	MappingRules json.RawMessage `json:"mappingRules"`
}

// ListBuiltinPresets returns preset summaries from the embedded filesystem.
func ListBuiltinPresets(presetFS embed.FS) ([]PresetInfo, error) {
	return listPresetsFromFS(presetFS, "presets/templates")
}

// ListUserPresets returns preset summaries from the on-disk user presets directory.
func ListUserPresets(dataDir string) ([]PresetInfo, error) {
	dir := filepath.Join(dataDir, "presets", "user")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []PresetInfo{}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []PresetInfo{}, nil
	}
	infos := make([]PresetInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var content PresetContent
		if err := json.Unmarshal(raw, &content); err != nil {
			continue
		}
		infos = append(infos, PresetInfo{
			ID:       strings.TrimSuffix(entry.Name(), ".json"),
			Platform: content.Platform,
			Type:     content.Type,
			Name:     content.Name,
		})
	}
	return infos, nil
}

// ReadPresetContent returns the full content of a preset.
// source: "builtin" or "user".
func ReadPresetContent(presetFS embed.FS, dataDir, source, id string) (*PresetContent, error) {
	var raw []byte
	var err error

	switch source {
	case "builtin":
		raw, err = fs.ReadFile(presetFS, "presets/templates/"+id+".json")
		if err != nil {
			return nil, fmt.Errorf("read builtin preset %q: %w", id, err)
		}
	case "user":
		userDir := filepath.Join(dataDir, "presets", "user")
		raw, err = os.ReadFile(filepath.Join(userDir, id+".json"))
		if err != nil {
			return nil, fmt.Errorf("read user preset %q: %w", id, err)
		}
	default:
		return nil, fmt.Errorf("unknown preset source: %s", source)
	}

	var content PresetContent
	if err := json.Unmarshal(raw, &content); err != nil {
		return nil, fmt.Errorf("parse preset %q: %w", id, err)
	}
	content.ID = id
	return &content, nil
}

// WriteUserPreset saves a user-created preset to disk.
func WriteUserPreset(dataDir string, id string, content *PresetContent) error {
	userDir := filepath.Join(dataDir, "presets", "user")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("create user preset dir: %w", err)
	}
	raw, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal user preset: %w", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, id+".json"), raw, 0644); err != nil {
		return fmt.Errorf("write user preset: %w", err)
	}
	return nil
}

// DeleteUserPreset removes a user-created preset from disk.
func DeleteUserPreset(dataDir, id string) error {
	userDir := filepath.Join(dataDir, "presets", "user")
	path := filepath.Join(userDir, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete user preset: %w", err)
	}
	return nil
}

func listPresetsFromFS(src fs.ReadDirFS, root string) ([]PresetInfo, error) {
	entries, err := fs.ReadDir(src, root)
	if err != nil {
		// If the directory doesn't exist (e.g., user presets empty), return empty.
		return []PresetInfo{}, nil
	}

	infos := make([]PresetInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(src, filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		var content PresetContent
		if err := json.Unmarshal(raw, &content); err != nil {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		infos = append(infos, PresetInfo{
			ID:       id,
			Platform: content.Platform,
			Type:     content.Type,
			Name:     content.Name,
		})
	}
	return infos, nil
}
