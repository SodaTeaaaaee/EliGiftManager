package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/xuri/excelize/v2"
)

func TestSupplierOrderDemoTemplate_RendersExactWorkbookContract(t *testing.T) {
	t.Parallel()

	rules, err := ParseMappingRules(SupplierOrderDemoMappingRules)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if err := ValidateMappingRulesConfig(SupplierOrderDemoDocType, rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}
	if rules.SheetName != "工作表1" {
		t.Fatalf("SheetName = %q, want 工作表1", rules.SheetName)
	}

	productID, addressID := uint(21), uint(31)
	line := &domain.SupplierOrderLine{SupplierLineNo: 1, FulfillmentLineID: 11, SupplierSKU: "RZ-1", SubmittedQuantity: 2}
	fl := &domain.FulfillmentLine{ID: 11, ProductID: &productID, CustomerAddressID: &addressID}
	data, err := NewTemplatePayloadRenderer().RenderSupplierExportXLSX(
		&domain.SupplierOrder{},
		[]*domain.SupplierOrderLine{line},
		map[uint]*domain.FulfillmentLine{11: fl},
		map[uint]*domain.Product{productID: {ID: productID, FactorySKU: " RZ-1 "}},
		map[uint]*domain.CustomerAddress{addressID: {
			ID: addressID, RecipientName: " Receiver ", Phone: " 13800000000 ",
			Province: "上海市", City: "上海市", District: "浦东新区", AddressLine1: "测试路1号",
		}},
		rules,
	)
	if err != nil {
		t.Fatalf("RenderSupplierExportXLSX: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()
	if sheets := f.GetSheetList(); len(sheets) != 1 || sheets[0] != "工作表1" {
		t.Fatalf("sheets = %v, want [工作表1]", sheets)
	}
	rows, err := f.GetRows("工作表1")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	wantHeader := []string{"第三方订单号", "收件人", "联系电话", "收件地址", "商家编码", "下单数量"}
	if len(rows) != 2 || len(rows[0]) != len(wantHeader) {
		t.Fatalf("rows = %#v", rows)
	}
	for i := range wantHeader {
		if rows[0][i] != wantHeader[i] {
			t.Fatalf("header[%d] = %q, want %q", i, rows[0][i], wantHeader[i])
		}
	}
	if rows[1][0] != "11" || rows[1][1] != "Receiver" || rows[1][2] != "13800000000" || rows[1][4] != "RZ-1" || rows[1][5] != "2" {
		t.Fatalf("data row = %#v", rows[1])
	}
}

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

	if err := ValidateMappingRulesConfig(CatalogDemoDocType, rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
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
	if !p1.SupportsExportSupplierOrder {
		t.Error("SupportsExportSupplierOrder = false, want true")
	}
	if p1.DemandKind != "" {
		t.Errorf("DemandKind = %q, want empty leftover (documentType is explicit)", p1.DemandKind)
	}
	if p1.IdentityStrategy != "" {
		t.Errorf("IdentityStrategy = %q, want empty leftover", p1.IdentityStrategy)
	}
	for _, docType := range []string{CatalogDemoDocType, ShipmentDemoDocType, SupplierOrderDemoDocType} {
		if err := ValidateProfileDocumentType(p1, docType); err != nil {
			t.Errorf("ValidateProfileDocumentType(%q): %v", docType, err)
		}
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
	orderTmpl1, err := templateRepo.FindByKey(ctx, SupplierOrderDemoTemplateKey)
	if err != nil || orderTmpl1 == nil {
		t.Fatalf("supplier order template after first seed: tmpl=%v err=%v", orderTmpl1, err)
	}
	if orderTmpl1.DocumentType != SupplierOrderDemoDocType || orderTmpl1.Format != "xlsx" {
		t.Errorf("supplier order template = %+v", orderTmpl1)
	}
	orderB1, err := bindingRepo.FindDefaultByProfileAndType(ctx, p1.ID, SupplierOrderDemoDocType)
	if err != nil || orderB1 == nil || orderB1.TemplateID != orderTmpl1.ID {
		t.Fatalf("supplier order binding: binding=%v err=%v", orderB1, err)
	}

	// Snapshot IDs before second seed.
	profileID := p1.ID
	templateID := tmpl1.ID
	bindingID := b1.ID
	shipTemplateID := shipTmpl1.ID
	shipBindingID := shipB1.ID
	orderTemplateID := orderTmpl1.ID
	orderBindingID := orderB1.ID

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
	var tmplCount, shipTmplCount, orderTmplCount int
	for i := range allTemplates {
		if allTemplates[i].TemplateKey == CatalogDemoTemplateKey {
			tmplCount++
		}
		if allTemplates[i].TemplateKey == ShipmentDemoTemplateKey {
			shipTmplCount++
		}
		if allTemplates[i].TemplateKey == SupplierOrderDemoTemplateKey {
			orderTmplCount++
		}
	}
	if tmplCount != 1 {
		t.Errorf("demo template count = %d, want 1", tmplCount)
	}
	if shipTmplCount != 1 {
		t.Errorf("shipment demo template count = %d, want 1", shipTmplCount)
	}
	if orderTmplCount != 1 {
		t.Errorf("supplier order demo template count = %d, want 1", orderTmplCount)
	}

	bindings, err := bindingRepo.ListByProfile(ctx, profileID)
	if err != nil {
		t.Fatalf("list bindings: %v", err)
	}
	if len(bindings) != 3 {
		t.Fatalf("profile binding total = %d, want 3 (catalog + shipment + supplier-order)", len(bindings))
	}
	var defaultCatalogBindings, defaultShipBindings, defaultOrderBindings int
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
		if bindings[i].DocumentType == SupplierOrderDemoDocType && bindings[i].IsDefault {
			defaultOrderBindings++
			if bindings[i].ID != orderBindingID || bindings[i].TemplateID != orderTemplateID {
				t.Errorf("supplier order default binding changed: %+v", bindings[i])
			}
		}
	}
	if defaultCatalogBindings != 1 {
		t.Errorf("default catalog binding count = %d, want 1", defaultCatalogBindings)
	}
	if defaultShipBindings != 1 {
		t.Errorf("default shipment binding count = %d, want 1", defaultShipBindings)
	}
	if defaultOrderBindings != 1 {
		t.Errorf("default supplier order binding count = %d, want 1", defaultOrderBindings)
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
	if p.DemandKind != "" {
		t.Errorf("DemandKind = %q, want empty leftover", p.DemandKind)
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

func TestSeedCatalogDemo_DoesNotOverwriteExistingMappingsOrCapabilities(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	ctx := context.Background()

	const operatorRules = `{"version":2,"mode":"header","hasHeader":true,"columns":{"product.factory_sku":"SKU"}}`
	stale := &domain.IntegrationProfile{
		ProfileKey:                     CatalogDemoProfileKey,
		SourceSurface:                  string(domain.SourceSurfaceFactory),
		FactorySupplierPlatform:        "rouzao",
		DemandKind:                     string(domain.DemandKindRetailOrder),
		SupportsImportProductCatalog:   true,
		SupportsImportSupplierShipment: true,
		// Operator left export capability off — seed must not flip it.
		SupportsExportSupplierOrder: false,
	}
	if err := profileRepo.Create(ctx, stale); err != nil {
		t.Fatalf("create stale profile: %v", err)
	}
	existingTmpl := &domain.DocumentTemplate{
		TemplateKey:  CatalogDemoTemplateKey,
		DocumentType: CatalogDemoDocType,
		Format:       "csv",
		MappingRules: operatorRules,
	}
	if err := templateRepo.Create(ctx, existingTmpl); err != nil {
		t.Fatalf("create operator template: %v", err)
	}

	profile, err := SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo)
	if err != nil {
		t.Fatalf("SeedCatalogDemo: %v", err)
	}
	if profile.SupportsExportSupplierOrder {
		t.Fatal("seed overwrote operator SupportsExportSupplierOrder")
	}
	if !profile.SupportsImportProductCatalog || !profile.SupportsImportSupplierShipment {
		t.Fatalf("catalog/shipment capabilities changed: %+v", profile)
	}
	if profile.DemandKind != "" {
		t.Fatalf("leftover DemandKind = %q, want empty (documentType is explicit)", profile.DemandKind)
	}

	storedTmpl, err := templateRepo.FindByKey(ctx, CatalogDemoTemplateKey)
	if err != nil || storedTmpl == nil {
		t.Fatalf("catalog template: tmpl=%v err=%v", storedTmpl, err)
	}
	if storedTmpl.MappingRules != operatorRules || storedTmpl.Format != "csv" {
		t.Fatalf("seed clobbered operator MappingRules/Format: %+v", storedTmpl)
	}

	shipTmpl, err := templateRepo.FindByKey(ctx, ShipmentDemoTemplateKey)
	if err != nil || shipTmpl == nil {
		t.Fatalf("missing shipment template was not created: tmpl=%v err=%v", shipTmpl, err)
	}
	orderTmpl, err := templateRepo.FindByKey(ctx, SupplierOrderDemoTemplateKey)
	if err != nil || orderTmpl == nil {
		t.Fatalf("missing supplier-order template was not created: tmpl=%v err=%v", orderTmpl, err)
	}

	catalogBinding, err := bindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, CatalogDemoDocType)
	if err != nil || catalogBinding == nil || catalogBinding.TemplateID != storedTmpl.ID {
		t.Fatalf("catalog default binding: binding=%v err=%v", catalogBinding, err)
	}
	shipBinding, err := bindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, ShipmentDemoDocType)
	if err != nil || shipBinding == nil || shipBinding.TemplateID != shipTmpl.ID {
		t.Fatalf("shipment default binding: binding=%v err=%v", shipBinding, err)
	}
	orderBinding, err := bindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, SupplierOrderDemoDocType)
	if err != nil {
		t.Fatalf("export binding lookup: %v", err)
	}
	if orderBinding != nil {
		t.Fatalf("export default binding created despite missing capability: %+v", orderBinding)
	}

	if _, err := SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	again, _ := templateRepo.FindByKey(ctx, CatalogDemoTemplateKey)
	if again.MappingRules != operatorRules {
		t.Fatalf("second seed clobbered MappingRules: %+v", again)
	}
}

func TestSeedCatalogDemo_PropagatesBindingLookupError(t *testing.T) {
	t.Parallel()

	profileRepo := newMockIntegrationProfileRepoSimple()
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := &failLookupBindingRepo{mockProfileTemplateBindingRepo: newMockProfileTemplateBindingRepo()}
	_, err := SeedCatalogDemo(context.Background(), profileRepo, templateRepo, bindingRepo)
	if err == nil {
		t.Fatal("expected binding lookup error, got nil")
	}
	if !strings.Contains(err.Error(), "db unavailable") {
		t.Fatalf("error = %v, want binding lookup failure", err)
	}
}

type failLookupBindingRepo struct {
	*mockProfileTemplateBindingRepo
}

func (m *failLookupBindingRepo) FindDefaultByProfileAndType(context.Context, uint, string) (*domain.IntegrationProfileTemplateBinding, error) {
	return nil, fmt.Errorf("db unavailable")
}

// Ensure mockIntegrationProfileRepoSimple is available (defined in import_reconcile_test.go).
var _ domain.IntegrationProfileRepository = (*mockIntegrationProfileRepoSimple)(nil)
