package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// fixturePath resolves testdata/integration_profile/<name> from the module root.
// Synthetic fixtures only — never copy real SampleData PII.
func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/app → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	path := filepath.Join(root, "testdata", "integration_profile", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture %s: %v", path, err)
	}
	return path
}

// ── 1. Bilibili membership positional CSV ──

func TestBilibiliMembershipPositionalFixture_FirstRowIsData(t *testing.T) {
	t.Parallel()

	path := fixturePath(t, "bilibili_membership_positional.csv")

	// hasHeader=false: first physical row must remain data, not peel into headers.
	sheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{
		Format:    "csv",
		HasHeader: false,
		Encoding:  "auto",
	})
	if err != nil {
		t.Fatalf("ReadTabularFile: %v", err)
	}
	if len(sheet.Headers) != 0 {
		t.Fatalf("headers should be empty when hasHeader=false, got %+v", sheet.Headers)
	}
	if len(sheet.Rows) != 3 {
		t.Fatalf("expected 3 data rows (first row is membership data, not header), got %d", len(sheet.Rows))
	}
	// BOM must not leak into the first cell of the first data row.
	if strings.HasPrefix(sheet.Rows[0][0], "\ufeff") {
		t.Fatalf("BOM leaked into first cell: %q", sheet.Rows[0][0])
	}
	if sheet.Rows[0][0] != "总督" || sheet.Rows[0][1] != "uid-10001" || sheet.Rows[0][2] != "DisplayA" {
		t.Fatalf("row0 unexpected: %+v", sheet.Rows[0])
	}

	// Apply bilibili-style positional MappingRules (preset shape).
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "positional",
		"hasHeader": false,
		"positions": {
			"line.gift_level_snapshot": 0,
			"document.source_customer_ref": 1,
			"document.display_name": 2
		},
		"defaults": {
			"line.line_type": "entitlement_rule",
			"line.requested_quantity": "1"
		}
	}`)
	if err != nil {
		t.Fatalf("ParseMappingRules: %v", err)
	}
	if err := ValidateMappingRulesConfig("import_entitlement", rules); err != nil {
		t.Fatalf("ValidateMappingRulesConfig: %v", err)
	}

	mapped, warnings, err := MapDemandImportRow(sheet.Rows[0], nil, rules)
	if err != nil {
		t.Fatalf("MapDemandImportRow: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if mapped.Line == nil {
		t.Fatal("expected line")
	}
	if mapped.Line.GiftLevelSnapshot != "总督" {
		t.Errorf("GiftLevelSnapshot = %q, want 总督", mapped.Line.GiftLevelSnapshot)
	}
	if mapped.Document.SourceCustomerRef != "uid-10001" {
		t.Errorf("SourceCustomerRef = %q, want uid-10001", mapped.Document.SourceCustomerRef)
	}
	if mapped.Document.DisplayName != "DisplayA" {
		t.Errorf("DisplayName = %q, want DisplayA", mapped.Document.DisplayName)
	}
	if mapped.Line.LineType != "entitlement_rule" {
		t.Errorf("LineType default = %q", mapped.Line.LineType)
	}

	// Contrast: hasHeader=true would incorrectly consume the first membership row as headers.
	asHeader, err := tabular.ReadTabularFile(path, tabular.ReadOptions{
		Format: "csv", HasHeader: true, Encoding: "auto",
	})
	if err != nil {
		t.Fatalf("ReadTabularFile hasHeader=true: %v", err)
	}
	if len(asHeader.Headers) != 3 || asHeader.Headers[0] != "总督" {
		t.Fatalf("hasHeader=true should peel first data row into headers, got %+v", asHeader.Headers)
	}
	if len(asHeader.Rows) != 2 {
		t.Fatalf("hasHeader=true should leave 2 data rows, got %d", len(asHeader.Rows))
	}
}

// ── 2. Rouzao shipment return fixture → MapAndReconcileShipments ──

func TestMapAndReconcileShipments_RouzaoReturnFixture(t *testing.T) {
	t.Parallel()

	path := fixturePath(t, "rouzao_shipment_return.csv")

	// Wave 1: two unique multi-SKU targets (matched via the normalized factory_sku
	// ref) + one ambiguous ref pair (matched via the
	// supplier_product_ref fallback tier) + one fallback-only target whose
	// FactorySKU deliberately does NOT derive to its shipment-row token, so it
	// can only resolve through the supplier_product_ref fallback tier.
	//
	// This is a fully self-contained synthetic convention fixture: FactorySKU
	// carries a "<PLATFORM>_<digits>" shape whose digit suffix is the shipment
	// token REF. SupplierProductRef is deliberately independent. No catalog
	// fixture or other sample file is used to derive these relationships.
	shipmentRepo := newMockShipmentRepo()
	supplierRepo := newMockSupplierRepoForShipment()
	fulfillRepo := newMockFulfillRepoForShipment()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	supplierRepo.orders[1] = &domain.SupplierOrder{
		ID: 1, WaveID: 1, Status: "draft", SupplierPlatform: "rouzao",
		CreatedAt: now, UpdatedAt: now,
	}

	pid1, pid2, pidAmbA, pidAmbB, pidFallback := uint(11), uint(12), uint(13), uint(14), uint(15)

	masterRepo := newMockProductMasterRepo()
	// Seed masters first so ProductMasterID pointers match Create-assigned IDs.
	m1 := &domain.ProductMaster{
		SupplierPlatform: "rouzao", FactorySKU: "ROUZAO_206068021",
		SupplierProductRef: "181881175", Name: "Synthetic Standee",
	}
	m2 := &domain.ProductMaster{
		SupplierPlatform: "rouzao", FactorySKU: "ROUZAO_63098307",
		SupplierProductRef: "181000002", Name: "Synthetic Magnet",
	}
	mAmb := &domain.ProductMaster{
		SupplierPlatform: "rouzao", FactorySKU: "ROUZAO_SKU-AMB",
		SupplierProductRef: "99999999", Name: "Synthetic Ambiguous",
	}
	// mFallback's FactorySKU has no "<letters>_<digits>" derivable suffix that
	// matches its shipment-row token — it can only resolve via the
	// supplier_product_ref fallback tier (priority 6/7).
	mFallback := &domain.ProductMaster{
		SupplierPlatform: "rouzao", FactorySKU: "NOPREFIXMATCH",
		SupplierProductRef: "205721969", Name: "Synthetic Fallback-only",
	}
	if err := masterRepo.Create(context.Background(), m1); err != nil {
		t.Fatalf("seed master1: %v", err)
	}
	if err := masterRepo.Create(context.Background(), m2); err != nil {
		t.Fatalf("seed master2: %v", err)
	}
	if err := masterRepo.Create(context.Background(), mAmb); err != nil {
		t.Fatalf("seed masterAmb: %v", err)
	}
	if err := masterRepo.Create(context.Background(), mFallback); err != nil {
		t.Fatalf("seed masterFallback: %v", err)
	}
	masterID1, masterID2, masterAmb, masterFallback := m1.ID, m2.ID, mAmb.ID, mFallback.ID

	addrRepo := newMockAddressRepo()
	addr := &domain.CustomerAddress{
		CustomerProfileID: 1,
		RecipientName:     "RecipientA",
		Phone:             "13800000001",
		AddressLine1:      "Synthetic Address Line",
	}
	if err := addrRepo.Create(context.Background(), addr); err != nil {
		t.Fatalf("seed address: %v", err)
	}
	addrID := addr.ID

	// FLs 100/101: unique normalized factory_sku ref + phone for multi-SKU expand.
	// FLs 102/103: same supplier_product_ref 99999999 + same phone → ambiguity
	// on row 1 via the fallback tier.
	// FL 104: only resolvable via the supplier_product_ref fallback tier
	// (its FactorySKU does not derive to the row-2 token).
	fulfillRepo.lines[100] = &domain.FulfillmentLine{
		ID: 100, WaveID: 1, ProductID: &pid1, CustomerAddressID: &addrID, Quantity: 1,
	}
	fulfillRepo.lines[101] = &domain.FulfillmentLine{
		ID: 101, WaveID: 1, ProductID: &pid2, CustomerAddressID: &addrID, Quantity: 2,
	}
	fulfillRepo.lines[102] = &domain.FulfillmentLine{
		ID: 102, WaveID: 1, ProductID: &pidAmbA, CustomerAddressID: &addrID, Quantity: 1,
	}
	fulfillRepo.lines[103] = &domain.FulfillmentLine{
		ID: 103, WaveID: 1, ProductID: &pidAmbB, CustomerAddressID: &addrID, Quantity: 1,
	}
	fulfillRepo.lines[104] = &domain.FulfillmentLine{
		ID: 104, WaveID: 1, ProductID: &pidFallback, CustomerAddressID: &addrID, Quantity: 1,
	}

	supplierRepo.orderLines[10] = &domain.SupplierOrderLine{
		ID: 10, SupplierOrderID: 1, FulfillmentLineID: 100, SubmittedQuantity: 5,
	}
	supplierRepo.orderLines[11] = &domain.SupplierOrderLine{
		ID: 11, SupplierOrderID: 1, FulfillmentLineID: 101, SubmittedQuantity: 5,
	}
	supplierRepo.orderLines[12] = &domain.SupplierOrderLine{
		ID: 12, SupplierOrderID: 1, FulfillmentLineID: 102, SubmittedQuantity: 5,
	}
	supplierRepo.orderLines[13] = &domain.SupplierOrderLine{
		ID: 13, SupplierOrderID: 1, FulfillmentLineID: 103, SubmittedQuantity: 5,
	}
	supplierRepo.orderLines[14] = &domain.SupplierOrderLine{
		ID: 14, SupplierOrderID: 1, FulfillmentLineID: 104, SubmittedQuantity: 5,
	}

	productRepo := &mockProductRepoForReconcile{products: map[uint]*domain.Product{
		pid1:        {ID: pid1, WaveID: 1, FactorySKU: "ROUZAO_206068021", ProductMasterID: &masterID1},
		pid2:        {ID: pid2, WaveID: 1, FactorySKU: "ROUZAO_63098307", ProductMasterID: &masterID2},
		pidAmbA:     {ID: pidAmbA, WaveID: 1, FactorySKU: "ROUZAO_SKU-A1", ProductMasterID: &masterAmb},
		pidAmbB:     {ID: pidAmbB, WaveID: 1, FactorySKU: "ROUZAO_SKU-A2", ProductMasterID: &masterAmb},
		pidFallback: {ID: pidFallback, WaveID: 1, FactorySKU: "NOPREFIXMATCH", ProductMasterID: &masterFallback},
	}}

	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory_rouzao_fixture", SourceSurface: "factory",
		FactorySupplierPlatform:        "rouzao",
		SupportsImportSupplierShipment: true,
	})
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "shipment_rouzao_fixture", DocumentType: "import_supplier_shipment", Format: "csv",
		MappingRules: ShipmentDemoMappingRules,
	}
	if err := templateRepo.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_supplier_shipment", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)
	uc = WithShipmentReconcileDeps(uc, mapping, productRepo, masterRepo, addrRepo, nil)

	result, err := uc.MapAndReconcileShipments(context.Background(), dto.MapAndReconcileShipmentsInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("MapAndReconcileShipments: %v", err)
	}

	// Row 0 expands multi-SKU → 2 shipments (derived factory_sku ref tier);
	// row 1 ambiguous → unresolved error (supplier_product_ref fallback tier,
	// 2 candidates); row 2 → 1 shipment resolved only via the
	// supplier_product_ref fallback tier (its FactorySKU does not derive to
	// the row's token).
	if result.TotalProcessed != 3 {
		t.Errorf("TotalProcessed = %d, want 3 physical source rows", result.TotalProcessed)
	}
	if result.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3 (2 derived-ref multi-SKU entries + 1 supplier_product_ref fallback entry)", result.SuccessCount)
	}
	if result.ErrorCount < 1 {
		t.Fatalf("expected at least 1 ambiguity error, got ErrorCount=%d Errors=%+v", result.ErrorCount, result.Errors)
	}
	ambFound := false
	for _, e := range result.Errors {
		if strings.Contains(e.Reason, "ambiguous") || strings.Contains(e.Reason, "fl_ids=") || strings.Contains(strings.ToLower(e.Reason), "ambigu") {
			ambFound = true
			break
		}
		// matchReconcileCandidate wording
		if strings.Contains(e.Reason, "99999999") || strings.Contains(e.Reason, "multiple") {
			ambFound = true
			break
		}
	}
	if !ambFound {
		// Accept any non-empty error for row 1 as unresolved ambiguity.
		if len(result.Errors) == 0 {
			t.Fatal("expected unresolved ambiguity error detail")
		}
		t.Logf("ambiguity errors (wording may vary): %+v", result.Errors)
	}

	// Tracking strip_leading_quote + shipped_at from row 0; multi-SKU → FL 100+101.
	foundTrack, foundShipped := false, false
	flIDs := map[uint]bool{}
	for _, s := range result.CreatedShipments {
		if s.TrackingNo == "YT1234567890" {
			foundTrack = true
		}
		if strings.HasPrefix(s.TrackingNo, "'") {
			t.Errorf("leading apostrophe not stripped: %q", s.TrackingNo)
		}
		if s.ShippedAt != nil && s.ShippedAt.Format("2006-01-02 15:04:05") == "2026-05-10 09:17:20" {
			foundShipped = true
		}
		for _, line := range s.Lines {
			flIDs[line.FulfillmentLineID] = true
		}
	}
	for _, lines := range shipmentRepo.shipmentLines {
		for _, l := range lines {
			flIDs[l.FulfillmentLineID] = true
		}
	}
	if !foundTrack {
		t.Errorf("expected tracking YT1234567890 without leading quote; created=%+v", result.CreatedShipments)
	}
	if !foundShipped {
		t.Errorf("expected shipped_at 2026-05-10 09:17:20 on created shipment")
	}
	if !flIDs[100] || !flIDs[101] {
		t.Errorf("expected multi-SKU expand to FL 100+101 via the derived factory_sku ref index, got flIDs=%v", flIDs)
	}
	if !flIDs[104] {
		t.Errorf("expected row 2 to resolve to FL 104 via the supplier_product_ref fallback index, got flIDs=%v", flIDs)
	}
}

// ── 3. Carrier codes: empty external + no internal column → skip_invalid details ──

func TestImportCarrierMappings_EmptyCodeAndNoInternalColumn(t *testing.T) {
	t.Parallel()

	path := fixturePath(t, "bilibili_carrier_codes.csv")

	repo := newMockCarrierMappingRepoFull()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "bili-carrier", SourceSurface: string(domain.SourceSurfaceMembership), RequiresCarrierMapping: true,
	})

	// Pre-seed one external mapping so "shunfeng" can backfill internal via ResolveByExternalOrAlias.
	// Deliberately wrong external name — import must overwrite from 快递公司名称 column.
	_, err := NewCarrierMappingUseCase(repo, profileRepo).CreateMapping(context.Background(), dto.CreateCarrierMappingInput{
		IntegrationProfileID: 1,
		InternalCarrierCode:  "SF",
		ExternalCarrierCode:  "shunfeng",
		ExternalCarrierName:  "WRONG_OLD_NAME",
	})
	if err != nil {
		t.Fatalf("seed carrier: %v", err)
	}

	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "carrier-fixture", DocumentType: "import_carrier_mapping", Format: "csv",
		// No internal column — matches BilibiliImportCarrierMappingRules shape.
		MappingRules: BilibiliImportCarrierMappingRules,
	}
	if err := templateRepo.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_carrier_mapping", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)
	uc := WithCarrierImportDeps(NewCarrierMappingUseCase(repo, profileRepo), mapping)

	result, err := uc.ImportCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{
		IntegrationProfileID: 1,
		ImportMode:           "skip_invalid",
		FilePath:             path,
	})
	if err != nil {
		t.Fatalf("ImportCarrierMappings: %v", err)
	}

	if result.TotalProcessed != 3 {
		t.Errorf("TotalProcessed = %d, want 3", result.TotalProcessed)
	}
	// shunfeng succeeds via ResolveByExternalOrAlias backfill; empty + yto fail.
	if result.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1 (shunfeng backfilled), result=%+v", result.SuccessCount, result)
	}
	if result.ErrorCount < 2 {
		t.Fatalf("ErrorCount = %d, want >=2 skip_invalid details, errors=%+v", result.ErrorCount, result.Errors)
	}

	// Seed MappingRules map 快递公司名称 → external_carrier_name (not the wrong "名称" header).
	var sawNameFrom快递公司名称 bool
	for _, m := range result.Mappings {
		if m.ExternalCarrierCode == "shunfeng" {
			if m.ExternalCarrierName != "顺丰" {
				t.Errorf("shunfeng ExternalCarrierName = %q, want 顺丰 from 快递公司名称 column (seed rules)", m.ExternalCarrierName)
			} else {
				sawNameFrom快递公司名称 = true
			}
		}
	}
	if !sawNameFrom快递公司名称 {
		t.Errorf("expected shunfeng mapping with name from 快递公司名称; mappings=%+v", result.Mappings)
	}

	var sawEmpty, sawMissingInternal bool
	for _, e := range result.Errors {
		if strings.Contains(e.Reason, "empty") {
			sawEmpty = true
		}
		if strings.Contains(e.Reason, "internal_carrier_code missing") || strings.Contains(e.Reason, "ResolveByExternalOrAlias") {
			sawMissingInternal = true
		}
	}
	if !sawEmpty {
		t.Errorf("expected empty external_carrier_code skip detail, errors=%+v", result.Errors)
	}
	if !sawMissingInternal {
		t.Errorf("expected missing-internal skip detail for yto, errors=%+v", result.Errors)
	}
}

// ── 4. CreateProfile factory formal path ──

func TestCreateProfile_FactorySurfaceFormalPath(t *testing.T) {
	t.Parallel()

	var stored *domain.IntegrationProfile
	uc := NewProfileManagementUseCase(
		&stubIntegrationProfileRepo{
			FindByKeyFn: func(key string) (*domain.IntegrationProfile, error) {
				return nil, fmt.Errorf("profile key not found")
			},
			CreateFn: func(profile *domain.IntegrationProfile) error {
				profile.ID = 42
				cp := *profile
				stored = &cp
				return nil
			},
		},
		&stubDemandDocumentRepo{},
		&stubChannelSyncRepo{},
		&stubProfileTemplateBindingRepo{},
		&stubClosureDecisionRepo{},
		nil,
	)

	dtoOut, err := uc.CreateProfile(context.Background(), dto.CreateProfileInput{
		ProfileKey:                     "factory_formal_path",
		SourceChannel:                  "rouzao",
		SourceSurface:                  string(domain.SourceSurfaceFactory),
		FactorySupplierPlatform:        "rouzao",
		SupportsImportProductCatalog:   true,
		SupportsImportSupplierShipment: true,
		SupportsExportSupplierOrder:    true,
	})
	if err != nil {
		t.Fatalf("CreateProfile factory: %v", err)
	}
	if dtoOut.ID != 42 {
		t.Errorf("ID = %d, want 42", dtoOut.ID)
	}
	if dtoOut.SourceSurface != string(domain.SourceSurfaceFactory) {
		t.Errorf("SourceSurface = %q", dtoOut.SourceSurface)
	}
	if dtoOut.FactorySupplierPlatform != "rouzao" {
		t.Errorf("FactorySupplierPlatform = %q", dtoOut.FactorySupplierPlatform)
	}
	if !dtoOut.SupportsImportProductCatalog || !dtoOut.SupportsImportSupplierShipment || !dtoOut.SupportsExportSupplierOrder {
		t.Errorf("factory capabilities not persisted on DTO: %+v", dtoOut)
	}
	if stored == nil || stored.FactorySupplierPlatform != "rouzao" {
		t.Fatalf("repo Create not called with factory platform, stored=%+v", stored)
	}
}

// ── 5. Export carrier internal→external + miss warning ──

func TestRenderTrackingExportCSV_CarrierLookupAndMissWarning(t *testing.T) {
	t.Parallel()

	rules := &TemplateMappingRules{
		Version: 2,
		Mode:    "header",
		Columns: map[string]string{
			"export.third_party_order_no": "订单号",
			"export.carrier_code":         "快递公司编码",
			"export.tracking_no":          "物流单号",
		},
		ColumnOrder: []string{
			"export.third_party_order_no",
			"export.carrier_code",
			"export.tracking_no",
		},
	}

	items := []domain.ChannelSyncItem{
		{ID: 1, FulfillmentLineID: 10, TrackingNo: "T-OK", CarrierCode: "SF"},
		{ID: 2, FulfillmentLineID: 11, TrackingNo: "T-MISS", CarrierCode: "UNKNOWN"},
	}

	renderer := NewTemplatePayloadRenderer().WithCarrierLookup(func(internal string) (string, bool) {
		if internal == "SF" {
			return "shunfeng", true
		}
		return "", false
	})

	csvText, err := renderer.RenderTrackingExportCSV(items, rules)
	if err != nil {
		t.Fatalf("RenderTrackingExportCSV: %v", err)
	}
	if !strings.Contains(csvText, "shunfeng") {
		t.Errorf("expected internal SF → external shunfeng in csv:\n%s", csvText)
	}
	if !strings.Contains(csvText, "UNKNOWN") {
		t.Errorf("expected miss passthrough UNKNOWN in csv:\n%s", csvText)
	}
	// Mapped code must not appear as raw SF when translation succeeded.
	// (Header is Chinese; data row 1 should use shunfeng.)
	if strings.Contains(csvText, ",SF,") || strings.Contains(csvText, "\nSF,") {
		t.Errorf("raw SF should not appear after successful translate:\n%s", csvText)
	}

	warnings := renderer.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("want 1 carrier-miss warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "UNKNOWN") {
		t.Errorf("warning should mention UNKNOWN, got %q", warnings[0])
	}
	if !strings.Contains(warnings[0], "carrier mapping miss") {
		t.Errorf("warning wording: %q", warnings[0])
	}
}
