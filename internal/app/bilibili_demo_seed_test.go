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
	if len(rules.Required) != 3 {
		t.Fatalf("tracking required fields = %v, want all three output columns", rules.Required)
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

func TestBilibiliDemandImportMappingRules_ParseableAndLegal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		docType string
		raw     string
	}{
		{name: "membership", docType: BilibiliImportEntitlementDocType, raw: BilibiliImportEntitlementMappingRules},
		{name: "retail", docType: BilibiliImportSalesOrderDocType, raw: BilibiliImportSalesOrderMappingRules},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rules, err := ParseMappingRules(tc.raw)
			if err != nil {
				t.Fatalf("ParseMappingRules: %v", err)
			}
			if err := ValidateMappingRulesConfig(tc.docType, rules); err != nil {
				t.Fatalf("ValidateMappingRulesConfig: %v", err)
			}
		})
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
	retail, err := profileRepo.FindByProfileKey(ctx, BilibiliRetailDemoProfileKey)
	if retail != nil {
		t.Fatalf("second demand profile %q must not be seeded: %+v (err=%v)", BilibiliRetailDemoProfileKey, retail, err)
	}

	entitlementTmpl, err := templateRepo.FindByKey(ctx, BilibiliImportEntitlementTemplateKey)
	if err != nil || entitlementTmpl == nil {
		t.Fatalf("entitlement template: tmpl=%v err=%v", entitlementTmpl, err)
	}
	salesTmpl, err := templateRepo.FindByKey(ctx, BilibiliImportSalesOrderTemplateKey)
	if err != nil || salesTmpl == nil {
		t.Fatalf("sales template: tmpl=%v err=%v", salesTmpl, err)
	}
	exportTmpl, err := templateRepo.FindByKey(ctx, BilibiliExportTrackingTemplateKey)
	if err != nil || exportTmpl == nil {
		t.Fatalf("export template: tmpl=%v err=%v", exportTmpl, err)
	}
	carrierTmpl, err := templateRepo.FindByKey(ctx, BilibiliImportCarrierTemplateKey)
	if err != nil || carrierTmpl == nil {
		t.Fatalf("carrier template: tmpl=%v err=%v", carrierTmpl, err)
	}
	if entitlementTmpl.Format != "csv" || salesTmpl.Format != "xls" || exportTmpl.Format != "xlsx" || carrierTmpl.Format != "xls" {
		t.Fatalf("template formats: entitlement=%q sales=%q tracking=%q carrier=%q",
			entitlementTmpl.Format, salesTmpl.Format, exportTmpl.Format, carrierTmpl.Format)
	}
	if exportTmpl.DocumentType != BilibiliExportTrackingDocType {
		t.Errorf("export docType = %q", exportTmpl.DocumentType)
	}
	if carrierTmpl.DocumentType != BilibiliImportCarrierDocType {
		t.Errorf("carrier docType = %q", carrierTmpl.DocumentType)
	}

	for _, check := range []struct {
		docType  string
		template uint
	}{
		{docType: BilibiliImportEntitlementDocType, template: entitlementTmpl.ID},
		{docType: BilibiliImportSalesOrderDocType, template: salesTmpl.ID},
		{docType: BilibiliExportTrackingDocType, template: exportTmpl.ID},
		{docType: BilibiliImportCarrierDocType, template: carrierTmpl.ID},
	} {
		binding, findErr := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, check.docType)
		if findErr != nil || binding == nil || binding.TemplateID != check.template {
			t.Errorf("binding profile=%d type=%s: binding=%v err=%v", p1.ID, check.docType, binding, findErr)
		}
	}

	// Second seed must not create duplicates.
	p2, err := SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("second SeedBilibiliDemo: %v", err)
	}
	if p2.ID != p1.ID {
		t.Errorf("second seed profile ID = %d, want %d", p2.ID, p1.ID)
	}
	entitlementTmpl2, _ := templateRepo.FindByKey(ctx, BilibiliImportEntitlementTemplateKey)
	salesTmpl2, _ := templateRepo.FindByKey(ctx, BilibiliImportSalesOrderTemplateKey)
	exportTmpl2, _ := templateRepo.FindByKey(ctx, BilibiliExportTrackingTemplateKey)
	carrierTmpl2, _ := templateRepo.FindByKey(ctx, BilibiliImportCarrierTemplateKey)
	if entitlementTmpl2 == nil || entitlementTmpl2.ID != entitlementTmpl.ID {
		t.Errorf("entitlement template not idempotent: first=%v second=%v", entitlementTmpl, entitlementTmpl2)
	}
	if salesTmpl2 == nil || salesTmpl2.ID != salesTmpl.ID {
		t.Errorf("sales template not idempotent: first=%v second=%v", salesTmpl, salesTmpl2)
	}
	if exportTmpl2 == nil || exportTmpl2.ID != exportTmpl.ID {
		t.Errorf("export template not idempotent: first=%v second=%v", exportTmpl, exportTmpl2)
	}
	if carrierTmpl2 == nil || carrierTmpl2.ID != carrierTmpl.ID {
		t.Errorf("carrier template not idempotent: first=%v second=%v", carrierTmpl, carrierTmpl2)
	}
	profiles, _ := profileRepo.List(ctx)
	templates, _ := templateRepo.List(ctx)
	bindings, _ := bindingRepo.ListByProfile(ctx, p1.ID)
	if len(profiles) != 1 || len(templates) != 4 || len(bindings) != 4 {
		t.Fatalf("seed duplicated records: profiles=%d templates=%d bindings=%d",
			len(profiles), len(templates), len(bindings))
	}
}

func TestSeedBilibiliDemo_ConvergesTemplateAndDefaultBinding(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	membership, err := SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("first seed: %v", err)
	}
	entitlement, _ := templateRepo.FindByKey(ctx, BilibiliImportEntitlementTemplateKey)
	carrier, _ := templateRepo.FindByKey(ctx, BilibiliImportCarrierTemplateKey)
	if entitlement == nil || carrier == nil {
		t.Fatal("seed did not create templates")
	}

	entitlement.Format = "json"
	entitlement.MappingRules = `{"columns":{"line.external_title":"wrong"}}`
	if err := templateRepo.Update(ctx, entitlement); err != nil {
		t.Fatalf("drift template: %v", err)
	}
	binding, _ := bindingRepo.FindDefaultByProfileAndType(ctx, membership.ID, BilibiliImportEntitlementDocType)
	binding.TemplateID = carrier.ID
	if err := bindingRepo.Update(ctx, binding); err != nil {
		t.Fatalf("drift binding: %v", err)
	}

	if _, err := SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo); err != nil {
		t.Fatalf("converging seed: %v", err)
	}
	gotTemplate, _ := templateRepo.FindByKey(ctx, BilibiliImportEntitlementTemplateKey)
	if gotTemplate.Format != "csv" || gotTemplate.MappingRules != BilibiliImportEntitlementMappingRules {
		t.Fatalf("template did not converge: %+v", gotTemplate)
	}
	gotBinding, _ := bindingRepo.FindDefaultByProfileAndType(ctx, membership.ID, BilibiliImportEntitlementDocType)
	if gotBinding == nil || gotBinding.TemplateID != gotTemplate.ID {
		t.Fatalf("binding did not converge: %+v", gotBinding)
	}
}

func TestSeedBilibiliDemo_RequiresProfileRepo(t *testing.T) {
	t.Parallel()
	if _, err := SeedBilibiliDemo(context.Background(), nil, nil, nil); err == nil {
		t.Fatal("expected error when profileRepo is nil")
	}
}
