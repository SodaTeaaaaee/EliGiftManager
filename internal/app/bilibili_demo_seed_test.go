package app

import (
	"context"
	"testing"
)

func TestBilibiliExportTrackingMappingRules_ParseableAndLegal(t *testing.T) {
	t.Parallel()

	rules, err := ParseMappingRules(BilibiliExportTrackingMappingRules)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if rules.Version != 2 || rules.Mode != "header" || !rules.HasHeader {
		t.Fatalf("unexpected rules meta: version=%d mode=%q hasHeader=%v", rules.Version, rules.Mode, rules.HasHeader)
	}
	wantCols := map[string]string{
		"export.external_document_no": "订单号*",
		"export.carrier_code":         "快递公司编码*（请在网页中查看快递编码）",
		"export.tracking_no":          "物流单号*",
	}
	for dest, src := range wantCols {
		if got := rules.Columns[dest]; got != src {
			t.Errorf("columns[%q] = %q, want %q", dest, got, src)
		}
	}
	if err := ValidateMappingRulesConfig(BilibiliExportTrackingDocType, rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}
}

func TestBilibiliImportCarrierMappingRules_ParseableAndLegal(t *testing.T) {
	t.Parallel()

	rules, err := ParseMappingRules(BilibiliImportCarrierMappingRules)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	wantCols := map[string]string{
		"carrier.external_carrier_code": "快递公司编码",
		"carrier.external_carrier_name": "快递公司名称",
	}
	for dest, src := range wantCols {
		if got := rules.Columns[dest]; got != src {
			t.Errorf("columns[%q] = %q, want %q", dest, got, src)
		}
	}
	if err := ValidateMappingRulesConfig(BilibiliImportCarrierDocType, rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}
}

func TestSeedBilibiliDemo_Idempotent(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	p1, err := SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("first SeedBilibiliDemo: %v", err)
	}
	if p1 == nil || p1.ProfileKey != BilibiliDemoProfileKey {
		t.Fatalf("profile = %+v", p1)
	}
	if !p1.RequiresCarrierMapping {
		t.Error("RequiresCarrierMapping = false, want true")
	}
	if p1.SourceChannel != "bilibili" {
		t.Errorf("SourceChannel = %q, want bilibili", p1.SourceChannel)
	}

	exportTmpl, err := templateRepo.FindByKey(ctx, BilibiliExportTrackingTemplateKey)
	if err != nil || exportTmpl == nil {
		t.Fatalf("export template: tmpl=%v err=%v", exportTmpl, err)
	}
	if exportTmpl.Format != "xlsx" {
		t.Errorf("export format = %q, want xlsx", exportTmpl.Format)
	}
	if exportTmpl.DocumentType != BilibiliExportTrackingDocType {
		t.Errorf("export docType = %q", exportTmpl.DocumentType)
	}

	carrierTmpl, err := templateRepo.FindByKey(ctx, BilibiliImportCarrierTemplateKey)
	if err != nil || carrierTmpl == nil {
		t.Fatalf("carrier template: tmpl=%v err=%v", carrierTmpl, err)
	}
	if carrierTmpl.DocumentType != BilibiliImportCarrierDocType {
		t.Errorf("carrier docType = %q", carrierTmpl.DocumentType)
	}

	bExport, err := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, BilibiliExportTrackingDocType)
	if err != nil || bExport == nil {
		t.Fatalf("export binding: b=%v err=%v", bExport, err)
	}
	bCarrier, err := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, BilibiliImportCarrierDocType)
	if err != nil || bCarrier == nil {
		t.Fatalf("carrier binding: b=%v err=%v", bCarrier, err)
	}

	// Second seed must not create duplicates.
	p2, err := SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("second SeedBilibiliDemo: %v", err)
	}
	if p2.ID != p1.ID {
		t.Errorf("second seed profile ID = %d, want %d", p2.ID, p1.ID)
	}
	exportTmpl2, _ := templateRepo.FindByKey(ctx, BilibiliExportTrackingTemplateKey)
	if exportTmpl2 == nil || exportTmpl2.ID != exportTmpl.ID {
		t.Errorf("export template not idempotent: first=%v second=%v", exportTmpl, exportTmpl2)
	}
}

func TestSeedBilibiliDemo_RequiresProfileRepo(t *testing.T) {
	t.Parallel()
	if _, err := SeedBilibiliDemo(context.Background(), nil, nil, nil); err == nil {
		t.Fatal("expected error when profileRepo is nil")
	}
}
