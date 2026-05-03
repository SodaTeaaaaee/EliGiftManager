package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestParseProductCSVSuccess(t *testing.T) {
	t.Parallel()
	csvFile := writeTestCSVFile(t, []string{"平台,工厂,商品SKU,商品名称,主图路径,颜色,尺码", "抖音,华东工厂,sku-100,保温杯,E:\\images\\cup.png,白色,大号"})
	template := model.TemplateConfig{Platform: "抖音", Type: model.TemplateTypeImportProduct, Name: "礼物导入模板", MappingRules: `{"hasHeader":true,"mapping":{"platform":{"sourceColumn":"平台"},"factory":{"sourceColumn":"工厂"},"factory_sku":{"sourceColumn":"商品SKU","required":true},"name":{"sourceColumn":"商品名称","required":true},"cover_image":{"sourceColumn":"主图路径"}},"extraData":{"strategy":"catch_all"}}`}
	products, err := ParseProductCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseProductCSV returned unexpected error: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	product := products[0]
	if product.Platform != "抖音" || product.Factory != "华东工厂" || product.FactorySKU != "sku-100" || product.Name != "保温杯" {
		t.Fatalf("unexpected product content: %+v", product)
	}
	if product.CoverImage != "E:\\images\\cup.png" {
		t.Fatalf("expected CoverImage path, got %q", product.CoverImage)
	}
	var extraData map[string]string
	if err := json.Unmarshal([]byte(product.ExtraData), &extraData); err != nil {
		t.Fatalf("failed to unmarshal ExtraData: %v", err)
	}
	if extraData["颜色"] != "白色" || extraData["尺码"] != "大号" {
		t.Fatalf("unexpected ExtraData content: %+v", extraData)
	}
}

func TestParseProductCSVRejectsMissingRequiredField(t *testing.T) {
	t.Parallel()
	csvFile := writeTestCSVFile(t, []string{"平台,工厂,商品SKU,商品名称", "抖音,华东工厂,,保温杯"})
	template := model.TemplateConfig{Platform: "抖音", Type: model.TemplateTypeImportProduct, Name: "礼物导入模板", MappingRules: `{"hasHeader":true,"mapping":{"platform":{"sourceColumn":"平台"},"factory":{"sourceColumn":"工厂"},"factory_sku":{"sourceColumn":"商品SKU","required":true},"name":{"sourceColumn":"商品名称","required":true}},"extraData":{"strategy":"catch_all"}}`}
	_, err := ParseProductCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "factory_sku") {
		t.Fatalf("expected missing factory_sku error, got %v", err)
	}
}
