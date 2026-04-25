package service

import (
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestParseDispatchRecordCSVSuccess(t *testing.T) {
	t.Parallel()
	csvFile := writeTestCSVFile(t, []string{"WaveID,MemberID,ProductID,Quantity,Status", "7,101,501,3,ready"})
	template := model.TemplateConfig{Type: model.TemplateTypeImportDispatchRecord, Name: "dispatch record import", MappingRules: `{"wave_id":"WaveID","member_id":"MemberID","product_id":"ProductID","quantity":"Quantity","status":"Status"}`}
	records, err := ParseDispatchRecordCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseDispatchRecordCSV returned unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 dispatch record, got %d", len(records))
	}
	record := records[0]
	if record.WaveID != 7 || record.MemberID != 101 || record.ProductID != 501 {
		t.Fatalf("unexpected dispatch record content: %+v", record)
	}
	if record.Quantity != 3 || record.Status != "ready" {
		t.Fatalf("unexpected dispatch record values: %+v", record)
	}
}

func TestParseDispatchRecordCSVUsesDefaultValues(t *testing.T) {
	t.Parallel()
	csvFile := writeTestCSVFile(t, []string{"MemberID,ProductID,Quantity,Status", "102,502,,"})
	template := model.TemplateConfig{Type: model.TemplateTypeImportDispatchRecord, Name: "dispatch record defaults", MappingRules: `{"member_id":"MemberID","product_id":"ProductID","quantity":"Quantity","status":"Status"}`}
	records, err := ParseDispatchRecordCSV(csvFile, template)
	if err != nil {
		t.Fatalf("ParseDispatchRecordCSV returned unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 dispatch record, got %d", len(records))
	}
	if records[0].Quantity != 1 {
		t.Fatalf("expected default Quantity to be 1, got %d", records[0].Quantity)
	}
	if records[0].Status != model.DispatchStatusPending {
		t.Fatalf("expected default Status to be pending, got %q", records[0].Status)
	}
}

func TestParseDispatchRecordCSVRejectsInvalidNumericField(t *testing.T) {
	t.Parallel()
	csvFile := writeTestCSVFile(t, []string{"MemberID,ProductID,Quantity", "abc,502,2"})
	template := model.TemplateConfig{Type: model.TemplateTypeImportDispatchRecord, Name: "dispatch record numeric", MappingRules: `{"member_id":"MemberID","product_id":"ProductID","quantity":"Quantity"}`}
	_, err := ParseDispatchRecordCSV(csvFile, template)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsigned integer") {
		t.Fatalf("expected invalid numeric error, got %v", err)
	}
}
