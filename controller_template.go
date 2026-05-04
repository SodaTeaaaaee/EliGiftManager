package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type TemplateController struct {
	db *gorm.DB
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

// ListDefaultTemplates returns the hardcoded preset templates that users can
// choose to add to their database. Does not write to the database.
func (c *TemplateController) ListDefaultTemplates() ([]TemplateItem, error) {
	presets := []struct {
		Platform     string
		Type         string
		Name         string
		MappingRules string
	}{
		{
			Platform:     "柔造",
			Type:         model.TemplateTypeImportProduct,
			Name:         "柔造 商品导入",
			MappingRules: `{"format":"zip","hasHeader":true,"mapping":{"name":{"sourceColumn":"商品名称","required":true},"factory_sku":{"sourceColumn":"商家编码","required":true}},"extraData":{"strategy":"catch_all"}}`,
		},
		{
			Platform:     "BILIBILI",
			Type:         model.TemplateTypeImportDispatchRecord,
			Name:         "BILIBILI 会员导入",
			MappingRules: `{"format":"csv","hasHeader":false,"mapping":{"gift_level":{"columnIndex":0,"required":true},"platform_uid":{"columnIndex":1,"required":true},"nickname":{"columnIndex":2},"platform":{"columnIndex":-1}},"extraData":{"strategy":"catch_all"}}`,
		},
		{
			Platform:     "柔造",
			Type:         model.TemplateTypeExportOrder,
			Name:         "柔造 工厂导出",
			MappingRules: `{"format":"csv","hasHeader":true,"columns":[{"headerName":"订单号","valueType":"order_no","prefix":"ROUZAO-"},{"headerName":"收件人","valueType":"recipient"},{"headerName":"手机号","valueType":"phone"},{"headerName":"收件地址","valueType":"address"},{"headerName":"SKU","valueType":"sku"},{"headerName":"数量","valueType":"quantity"}]}`,
		},
	}
	items := make([]TemplateItem, 0, len(presets))
	for _, p := range presets {
		items = append(items, TemplateItem{
			Platform:     p.Platform,
			Type:         p.Type,
			Name:         p.Name,
			MappingRules: p.MappingRules,
		})
	}
	return items, nil
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
