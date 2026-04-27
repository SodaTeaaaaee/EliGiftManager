package service

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

// defaultTemplates defines the seed templates guaranteed to exist on startup.
var defaultTemplates = []model.TemplateConfig{
	{
		Platform:     "柔造",
		Type:         model.TemplateTypeImportProduct,
		Name:         "柔造 商品导入",
		MappingRules: `{"format": "zip", "csvPattern": "*.csv", "imageDir": "主图", "mapping": {"name": "商品名称", "factorySku": "商家编码"}}`,
	},
	{
		Platform:     "BILIBILI",
		Type:         model.TemplateTypeImportDispatchRecord,
		Name:         "BILIBILI 会员导入",
		MappingRules: `{"hasHeader": false, "mapping": {"giftName": {"columnIndex": 0}, "platformUid": {"columnIndex": 1, "required": true}, "nickname": {"columnIndex": 2}}}`,
	},
	{
		Platform:     "柔造",
		Type:         model.TemplateTypeExportOrder,
		Name:         "柔造 工厂导出",
		MappingRules: `{"headers": ["第三方订单号", "收件人", "联系电话", "收件地址", "商家编码", "下单数量"], "prefix": "ROUZAO-"}`,
	},
}

// EnsureDefaultTemplates seeds the default templates.
// Each is inserted only when no row with the same (platform, type, name) exists.
func EnsureDefaultTemplates(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("EnsureDefaultTemplates: db is nil")
	}
	for i := range defaultTemplates {
		t := &defaultTemplates[i]
		if err := db.Where("platform = ? AND type = ? AND name = ?", t.Platform, t.Type, t.Name).
			FirstOrCreate(t).Error; err != nil {
			return fmt.Errorf("EnsureDefaultTemplates: %q: %w", t.Name, err)
		}
	}
	return nil
}
