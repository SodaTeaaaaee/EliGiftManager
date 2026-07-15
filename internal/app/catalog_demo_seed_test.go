package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestCatalogDemoMappingRules_Parseable(t *testing.T) {
	t.Parallel()

	rules, err := ParseMappingRules(CatalogDemoMappingRules)
	if err != nil {
		t.Fatalf("ParseMappingRules(CatalogDemoMappingRules): %v", err)
	}
	if rules.Version != 2 {
		t.Errorf("version = %d, want 2", rules.Version)
	}
	if rules.Mode != "header" {
		t.Errorf("mode = %q, want header", rules.Mode)
	}
	if !rules.HasHeader {
		t.Error("hasHeader want true")
	}

	wantCols := map[string]string{
		"product.factory_sku":          "商家编码",
		"product.name":                 "商品名称",
		"product.supplier_product_ref": "商品ID",
	}
	for dest, src := range wantCols {
		if got := rules.Columns[dest]; got != src {
			t.Errorf("columns[%q] = %q, want %q", dest, got, src)
		}
	}

	if rules.ImageLayout == nil {
		t.Fatal("expected ImageLayout")
	}
	l := rules.ImageLayout
	if !l.Enabled {
		t.Error("imageLayout.enabled want true")
	}
	if l.CoverDir != "主图" {
		t.Errorf("coverDir = %q, want 主图", l.CoverDir)
	}
	if l.DetailDir != "详情图" {
		t.Errorf("detailDir = %q, want 详情图", l.DetailDir)
	}
	if l.MatchField != "product.name" {
		t.Errorf("matchField = %q, want product.name", l.MatchField)
	}
	if l.NamePattern != "{match}#{nn}" {
		t.Errorf("namePattern = %q, want {match}#{nn}", l.NamePattern)
	}
	if l.TabularGlob != "*.csv" {
		t.Errorf("tabularGlob = %q, want *.csv", l.TabularGlob)
	}
}

func TestShipmentDemoMappingRules_ParseableAndLegal(t *testing.T) {
	t.Parallel()

	rules, err := ParseMappingRules(ShipmentDemoMappingRules)
	if err != nil {
		t.Fatalf("ParseMappingRules(ShipmentDemoMappingRules): %v", err)
	}
	if err := ValidateMappingRulesConfig(ShipmentDemoDocType, rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}

	wantCols := map[string]string{
		"shipment.external_shipment_no": "订单编号",
		"shipment.sku_quantity":         "规格&数量",
		"shipment.recipient_name":       "收件人",
		"shipment.phone":                "电话",
		"shipment.carrier_name":         "物流公司",
		"shipment.tracking_no":          "物流单号",
		"shipment.shipped_at":           "打印快递时间",
	}
	for dest, src := range wantCols {
		if got := rules.Columns[dest]; got != src {
			t.Errorf("columns[%q] = %q, want %q", dest, got, src)
		}
	}
	transforms := rules.Transforms["shipment.tracking_no"]
	foundStrip := false
	for _, tr := range transforms {
		if tr == "strip_leading_quote" {
			foundStrip = true
		}
	}
	if !foundStrip {
		t.Errorf("tracking_no transforms = %v, want strip_leading_quote", transforms)
	}
}

func TestSeedCatalogDemo_Idempotent(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	// First seed creates profile + template + binding.
	p1, err := SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("first SeedCatalogDemo: %v", err)
	}
	if p1 == nil {
		t.Fatal("first seed returned nil profile")
	}
	if p1.ProfileKey != CatalogDemoProfileKey {
		t.Errorf("ProfileKey = %q, want %q", p1.ProfileKey, CatalogDemoProfileKey)
	}
	if p1.FactorySupplierPlatform != "rouzao" {
		t.Errorf("FactorySupplierPlatform = %q, want rouzao", p1.FactorySupplierPlatform)
	}
	if p1.SourceSurface != "factory" {
		t.Errorf("SourceSurface = %q, want factory", p1.SourceSurface)
	}
	if !p1.SupportsImportProductCatalog {
		t.Error("SupportsImportProductCatalog = false, want true")
	}
	if p1.ID == 0 {
		t.Error("expected non-zero profile ID after create")
	}

	tmpl1, err := templateRepo.FindByKey(ctx, CatalogDemoTemplateKey)
	if err != nil || tmpl1 == nil {
		t.Fatalf("template after first seed: tmpl=%v err=%v", tmpl1, err)
	}
	if tmpl1.DocumentType != CatalogDemoDocType {
		t.Errorf("DocumentType = %q, want %q", tmpl1.DocumentType, CatalogDemoDocType)
	}
	if tmpl1.Format != "zip" {
		t.Errorf("Format = %q, want zip", tmpl1.Format)
	}
	if tmpl1.MappingRules != CatalogDemoMappingRules {
		t.Error("MappingRules mismatch after first seed")
	}

	b1, err := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, CatalogDemoDocType)
	if err != nil || b1 == nil {
		t.Fatalf("binding after first seed: b=%v err=%v", b1, err)
	}
	if b1.TemplateID != tmpl1.ID {
		t.Errorf("binding.TemplateID = %d, want %d", b1.TemplateID, tmpl1.ID)
	}
	if !b1.IsDefault {
		t.Error("binding.IsDefault want true")
	}

	// Shipment demo template + binding.
	if !p1.SupportsImportSupplierShipment {
		t.Error("SupportsImportSupplierShipment = false, want true")
	}
	shipTmpl1, err := templateRepo.FindByKey(ctx, ShipmentDemoTemplateKey)
	if err != nil || shipTmpl1 == nil {
		t.Fatalf("shipment template after first seed: tmpl=%v err=%v", shipTmpl1, err)
	}
	if shipTmpl1.DocumentType != ShipmentDemoDocType {
		t.Errorf("shipment DocumentType = %q, want %q", shipTmpl1.DocumentType, ShipmentDemoDocType)
	}
	if shipTmpl1.Format != "csv" {
		t.Errorf("shipment Format = %q, want csv", shipTmpl1.Format)
	}
	if shipTmpl1.MappingRules != ShipmentDemoMappingRules {
		t.Error("shipment MappingRules mismatch after first seed")
	}
	shipB1, err := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, ShipmentDemoDocType)
	if err != nil || shipB1 == nil {
		t.Fatalf("shipment binding after first seed: b=%v err=%v", shipB1, err)
	}
	if shipB1.TemplateID != shipTmpl1.ID {
		t.Errorf("shipment binding.TemplateID = %d, want %d", shipB1.TemplateID, shipTmpl1.ID)
	}

	// Snapshot IDs before second seed.
	profileID := p1.ID
	templateID := tmpl1.ID
	bindingID := b1.ID
	shipTemplateID := shipTmpl1.ID
	shipBindingID := shipB1.ID

	// Second seed must not create duplicates.
	p2, err := SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("second SeedCatalogDemo: %v", err)
	}
	if p2.ID != profileID {
		t.Errorf("second seed profile ID = %d, want %d (idempotent)", p2.ID, profileID)
	}

	// Count profiles with the demo key.
	allProfiles, err := profileRepo.List(ctx)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	var demoCount int
	for i := range allProfiles {
		if allProfiles[i].ProfileKey == CatalogDemoProfileKey {
			demoCount++
		}
	}
	if demoCount != 1 {
		t.Errorf("demo profile count = %d, want 1", demoCount)
	}

	allTemplates, err := templateRepo.List(ctx)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	var tmplCount, shipTmplCount int
	for i := range allTemplates {
		if allTemplates[i].TemplateKey == CatalogDemoTemplateKey {
			tmplCount++
		}
		if allTemplates[i].TemplateKey == ShipmentDemoTemplateKey {
			shipTmplCount++
		}
	}
	if tmplCount != 1 {
		t.Errorf("demo template count = %d, want 1", tmplCount)
	}
	if shipTmplCount != 1 {
		t.Errorf("shipment demo template count = %d, want 1", shipTmplCount)
	}

	bindings, err := bindingRepo.ListByProfile(ctx, profileID)
	if err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	var defaultCatalogBindings, defaultShipBindings int
	for i := range bindings {
		if bindings[i].DocumentType == CatalogDemoDocType && bindings[i].IsDefault {
			defaultCatalogBindings++
			if bindings[i].ID != bindingID {
				t.Errorf("default binding ID changed: got %d want %d", bindings[i].ID, bindingID)
			}
			if bindings[i].TemplateID != templateID {
				t.Errorf("default binding TemplateID changed: got %d want %d", bindings[i].TemplateID, templateID)
			}
		}
		if bindings[i].DocumentType == ShipmentDemoDocType && bindings[i].IsDefault {
			defaultShipBindings++
			if bindings[i].ID != shipBindingID {
				t.Errorf("shipment default binding ID changed: got %d want %d", bindings[i].ID, shipBindingID)
			}
			if bindings[i].TemplateID != shipTemplateID {
				t.Errorf("shipment default binding TemplateID changed: got %d want %d", bindings[i].TemplateID, shipTemplateID)
			}
		}
	}
	if defaultCatalogBindings != 1 {
		t.Errorf("default catalog binding count = %d, want 1", defaultCatalogBindings)
	}
	if defaultShipBindings != 1 {
		t.Errorf("default shipment binding count = %d, want 1", defaultShipBindings)
	}
}

func TestSeedCatalogDemo_RequiresProfileRepo(t *testing.T) {
	t.Parallel()

	_, err := SeedCatalogDemo(context.Background(), nil, nil, nil)
	if err == nil {
		t.Fatal("expected error when profileRepo is nil")
	}
}

func TestSeedCatalogDemo_ProfileOnlyWhenTemplateReposNil(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	ctx := context.Background()

	p, err := SeedCatalogDemo(ctx, profileRepo, nil, nil)
	if err != nil {
		t.Fatalf("SeedCatalogDemo profile-only: %v", err)
	}
	if p.ProfileKey != CatalogDemoProfileKey {
		t.Errorf("ProfileKey = %q", p.ProfileKey)
	}

	// Re-seed still idempotent for profile.
	p2, err := SeedCatalogDemo(ctx, profileRepo, nil, nil)
	if err != nil {
		t.Fatalf("second profile-only seed: %v", err)
	}
	if p2.ID != p.ID {
		t.Errorf("profile ID changed on re-seed: %d -> %d", p.ID, p2.ID)
	}
}

// Ensure mockIntegrationProfileRepoSimple is available (defined in import_reconcile_test.go).
var _ domain.IntegrationProfileRepository = (*mockIntegrationProfileRepoSimple)(nil)
