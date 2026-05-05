package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

type TemplateController struct {
	db       *gorm.DB
	presetFS embed.FS
}

func (c *TemplateController) CreateTemplate(platform, templateType, name, mappingRules string) (TemplateItem, error) {
	platform = strings.TrimSpace(platform)
	templateType = strings.TrimSpace(templateType)
	name = strings.TrimSpace(name)
	mappingRules = strings.TrimSpace(mappingRules)
	if platform == "" || templateType == "" || name == "" {
		return TemplateItem{}, fmt.Errorf("platform, type and name are required")
	}
	if mappingRules == "" {
		mappingRules = "{}"
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(mappingRules), &probe); err != nil {
		return TemplateItem{}, fmt.Errorf("mapping rules must be valid JSON: %w", err)
	}
	if validation, err := service.ValidateTemplateRules(templateType, mappingRules); err != nil {
		return TemplateItem{}, err
	} else if !validation.Valid {
		return TemplateItem{}, fmt.Errorf("template validation failed: %s", strings.Join(validation.Errors, "; "))
	}
	db := c.db
	template := model.TemplateConfig{Platform: platform, Type: templateType, Name: name, MappingRules: mappingRules}
	if err := db.Create(&template).Error; err != nil {
		return TemplateItem{}, err
	}
	return templateItemFromModel(template), nil
}

func (c *TemplateController) ListTemplates() ([]TemplateItem, error) {
	db := c.db
	var templates []model.TemplateConfig
	if err := db.Order("platform ASC, type ASC, updated_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}
	items := make([]TemplateItem, 0, len(templates))
	for _, template := range templates {
		items = append(items, templateItemFromModel(template))
	}
	return items, nil
}

// ListBuiltinPresets returns the built-in (embedded) preset templates.
func (c *TemplateController) ListBuiltinPresets() ([]service.PresetInfo, error) {
	return service.ListBuiltinPresets(c.presetFS)
}

// ListUserPresets returns user-created presets from the data directory.
func (c *TemplateController) ListUserPresets() ([]service.PresetInfo, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return nil, err
	}
	return service.ListUserPresets(dataDir)
}

// GetPresetContent returns the full content of a preset.
func (c *TemplateController) GetPresetContent(source, id string) (*service.PresetContent, error) {
	dataDir, _ := service.ResolveDataDir()
	return service.ReadPresetContent(c.presetFS, dataDir, source, id)
}

// AddFromPreset creates a template from a preset and returns the created TemplateItem.
func (c *TemplateController) AddFromPreset(source, id string) (TemplateItem, error) {
	dataDir, err := service.ResolveDataDir()
	if err != nil {
		return TemplateItem{}, err
	}
	content, err := service.ReadPresetContent(c.presetFS, dataDir, source, id)
	if err != nil {
		return TemplateItem{}, err
	}
	mappingJSON, err := json.Marshal(content.MappingRules)
	if err != nil {
		return TemplateItem{}, fmt.Errorf("marshal preset mapping rules: %w", err)
	}
	return c.CreateTemplate(content.Platform, content.Type, content.Name, string(mappingJSON))
}

func (c *TemplateController) UpdateTemplate(id uint, platform, templateType, name, mappingRules string) error {
	platform = strings.TrimSpace(platform)
	templateType = strings.TrimSpace(templateType)
	name = strings.TrimSpace(name)
	mappingRules = strings.TrimSpace(mappingRules)
	if platform == "" || templateType == "" || name == "" {
		return fmt.Errorf("platform, type and name are required")
	}
	if mappingRules == "" {
		mappingRules = "{}"
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(mappingRules), &probe); err != nil {
		return fmt.Errorf("mapping rules must be valid JSON: %w", err)
	}
	if validation, err := service.ValidateTemplateRules(templateType, mappingRules); err != nil {
		return err
	} else if !validation.Valid {
		return fmt.Errorf("template validation failed: %s", strings.Join(validation.Errors, "; "))
	}
	db := c.db
	result := db.Model(&model.TemplateConfig{}).Where("id = ?", id).Updates(map[string]any{
		"platform":      platform,
		"type":          templateType,
		"name":          name,
		"mapping_rules": mappingRules,
	})
	if result.Error != nil {
		return fmt.Errorf("update template failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}


// ValidateTemplate checks the mapping rules for a template type without saving.
func (c *TemplateController) ValidateTemplate(templateType, mappingRules string) (*service.TemplateValidationResult, error) {
	return service.ValidateTemplateRules(templateType, mappingRules)
}

func (c *TemplateController) DeleteTemplate(id uint) error {
	result := c.db.Delete(&model.TemplateConfig{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete template failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template not found")
	}
	return nil
}
