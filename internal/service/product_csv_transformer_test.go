package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestParseProductCSVSuccess(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"工厂名,商品SKU,商品名称,主图路径,颜色,尺码",
		"华东工厂,sku-100,保温杯,E:\\images\\cup.png,白色,大号",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportProduct,
		Name: "商品导入模板",
		MappingRules: `{
			"factory": "工厂名",
			"factory_sku": "商品SKU",
			"name": "商品名称",
			"image_path": "主图路径"
		}`,
	}

	products, err := ParseProductCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseProductCSV returned unexpected error: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}

	product := products[0]
	if product.Factory != "华东工厂" || product.FactorySKU != "sku-100" || product.Name != "保温杯" {
		t.Fatalf("unexpected product content: %+v", product)
	}
	if product.ImagePath != "E:\\images\\cup.png" {
		t.Fatalf("expected ImagePath to be E:\\images\\cup.png, got %q", product.ImagePath)
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

	csvFile := writeTestCSVFile(t, []string{
		"工厂名,商品SKU,商品名称",
		"华东工厂,,保温杯",
	})

	template := model.TemplateConfig{
		Type:         model.TemplateTypeImportProduct,
		Name:         "商品导入模板",
		MappingRules: `{"factory":"工厂名","factory_sku":"商品SKU","name":"商品名称"}`,
	}

	_, err := ParseProductCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "缺少必填字段 FactorySKU") {
		t.Fatalf("expected missing FactorySKU error, got %v", err)
	}
}
