package service

import (
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestParseDispatchRecordCSVSuccess(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"批次号,会员ID,商品ID,数量,状态",
		"batch-20260423,101,501,3,ready",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportDispatchRecord,
		Name: "发货记录导入模板",
		MappingRules: `{
			"batch_name": "批次号",
			"member_id": "会员ID",
			"product_id": "商品ID",
			"quantity": "数量",
			"status": "状态"
		}`,
	}

	records, err := ParseDispatchRecordCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseDispatchRecordCSV returned unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 dispatch record, got %d", len(records))
	}

	record := records[0]
	if record.BatchName != "batch-20260423" || record.MemberID != 101 || record.ProductID != 501 {
		t.Fatalf("unexpected dispatch record content: %+v", record)
	}
	if record.Quantity != 3 || record.Status != "ready" {
		t.Fatalf("unexpected dispatch record values: %+v", record)
	}
}

func TestParseDispatchRecordCSVUsesDefaultValues(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"批次号,会员ID,商品ID,数量,状态",
		"batch-20260424,102,502,,",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportDispatchRecord,
		Name: "发货记录默认值模板",
		MappingRules: `{
			"batch_name": "批次号",
			"member_id": "会员ID",
			"product_id": "商品ID",
			"quantity": "数量",
			"status": "状态"
		}`,
	}

	records, err := ParseDispatchRecordCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseDispatchRecordCSV returned unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 dispatch record, got %d", len(records))
	}

	record := records[0]
	if record.Quantity != 1 {
		t.Fatalf("expected default Quantity to be 1, got %d", record.Quantity)
	}
	if record.Status != model.DispatchStatusPending {
		t.Fatalf("expected default Status to be pending, got %q", record.Status)
	}
}

func TestParseDispatchRecordCSVRejectsInvalidNumericField(t *testing.T) {
	t.Parallel()

	csvFile := writeTestCSVFile(t, []string{
		"批次号,会员ID,商品ID,数量",
		"batch-20260425,abc,502,2",
	})

	template := model.TemplateConfig{
		Type: model.TemplateTypeImportDispatchRecord,
		Name: "发货记录数值模板",
		MappingRules: `{
			"batch_name": "批次号",
			"member_id": "会员ID",
			"product_id": "商品ID",
			"quantity": "数量"
		}`,
	}

	_, err := ParseDispatchRecordCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "不是合法的无符号整数") {
		t.Fatalf("expected invalid numeric error, got %v", err)
	}
}
