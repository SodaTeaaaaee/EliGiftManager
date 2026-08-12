package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ── UpsertAddressFromImport ──

func TestUpsertAddressFromImport_MatchByNamePhone_Updates(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 1,
		Label:             "Home",
		RecipientName:     "Hina",
		Phone:             "13800000000",
		AddressLine1:      "Old street",
		City:              "Gehenna",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	updated, err := uc.UpsertAddressFromImport(context.Background(), 1, RecipientAddressDraft{
		RecipientName: "Hina",
		Phone:         "13800000000",
		AddressLine1:  "New street",
		City:          "Millennium",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected hit on id=%d, got id=%d", created.ID, updated.ID)
	}
	if updated.AddressLine1 != "New street" || updated.City != "Millennium" {
		t.Fatalf("expected fields updated, got %+v", updated)
	}
	// Only one address should exist.
	list, err := uc.ListAddressesByProfile(context.Background(), 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 address after update, got %d", len(list))
	}
}

func TestUpsertAddressFromImport_MatchByNameAddressLine1_WhenPhoneEmpty(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	created, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 2,
		Label:             "Office",
		RecipientName:     "Sensei",
		Phone:             "",
		AddressLine1:      "Schale HQ",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	updated, err := uc.UpsertAddressFromImport(context.Background(), 2, RecipientAddressDraft{
		RecipientName: "Sensei",
		Phone:         "",
		AddressLine1:  "Schale HQ",
		City:          "Kivotos",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected match id=%d, got %d", created.ID, updated.ID)
	}
	if updated.City != "Kivotos" {
		t.Fatalf("city = %q, want Kivotos", updated.City)
	}
}

func TestUpsertAddressFromImport_CreatesWhenNoMatch(t *testing.T) {
	t.Parallel()
	addrRepo := newMockAddressRepo()
	fulfillRepo := newMockFulfillRepo()
	uc := NewAddressManagementUseCase(addrRepo, fulfillRepo)

	// Existing different address.
	if _, err := uc.CreateAddress(context.Background(), dto.CreateAddressInput{
		CustomerProfileID: 3,
		Label:             "A",
		RecipientName:     "Other",
		Phone:             "100",
		AddressLine1:      "Elsewhere",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	created, err := uc.UpsertAddressFromImport(context.Background(), 3, RecipientAddressDraft{
		RecipientName: "Hina",
		Phone:         "200",
		AddressLine1:  "HQ",
		Label:         "import",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected new id")
	}
	list, err := uc.ListAddressesByProfile(context.Background(), 3)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(list))
	}
}

// ── Reconcile 3-level match ──

type mockProductRepoForReconcile struct {
	mu       sync.Mutex
	products map[uint]*domain.Product
}

func (m *mockProductRepoForReconcile) Create(ctx context.Context, product *domain.Product) error {
	panic("not implemented")
}
func (m *mockProductRepoForReconcile) FindByID(ctx context.Context, id uint) (*domain.Product, error) {
	panic("not implemented")
}
func (m *mockProductRepoForReconcile) FindByWaveAndID(ctx context.Context, waveID uint, id uint) (*domain.Product, error) {
	panic("not implemented")
}
func (m *mockProductRepoForReconcile) ListByWave(ctx context.Context, waveID uint) ([]domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Product
	for _, p := range m.products {
		if p.WaveID == waveID {
			out = append(out, *p)
		}
	}
	return out, nil
}
func (m *mockProductRepoForReconcile) FindByWaveAndSKU(ctx context.Context, waveID uint, platform, sku string) (*domain.Product, error) {
	panic("not implemented")
}
func (m *mockProductRepoForReconcile) DeleteByWave(ctx context.Context, waveID uint) error {
	panic("not implemented")
}

func TestMatchReconcileCandidate_ThreeLevels(t *testing.T) {
	t.Parallel()

	pid1 := uint(10)
	addr1 := uint(20)
	flByID := &domain.FulfillmentLine{ID: 100, WaveID: 1, ProductID: &pid1, CustomerAddressID: &addr1}
	flSKUPhone := &domain.FulfillmentLine{ID: 101, WaveID: 1, ProductID: &pid1, CustomerAddressID: &addr1}
	flSKUName := &domain.FulfillmentLine{ID: 102, WaveID: 1, ProductID: &pid1, CustomerAddressID: &addr1}

	idx := &reconcileIndex{
		byFLID: map[uint]*reconcileCandidate{
			100: {line: flByID, factorySKU: "SKU-A", phone: "138", recipientName: "Hina"},
			101: {line: flSKUPhone, factorySKU: "SKU-B", phone: "139", recipientName: "Ako"},
			102: {line: flSKUName, factorySKU: "SKU-C", phone: "", recipientName: "Iori"},
		},
		bySKUPhone: map[string][]*reconcileCandidate{
			skuPhoneKey("SKU-B", "139"): {{line: flSKUPhone, factorySKU: "SKU-B", phone: "139", recipientName: "Ako"}},
		},
		bySKURecipient: map[string][]*reconcileCandidate{
			skuRecipientKey("SKU-C", "Iori"): {{line: flSKUName, factorySKU: "SKU-C", phone: "", recipientName: "Iori"}},
		},
		solByFulfillment: map[uint]*domain.SupplierOrderLine{},
	}

	// Level 1: FL ID
	c1, err := matchReconcileCandidate(idx, "100", "", "", "", "")
	if err != nil || c1.line.ID != 100 {
		t.Fatalf("level1: cand=%+v err=%v", c1, err)
	}

	// Level 2: sku+phone
	c2, err := matchReconcileCandidate(idx, "", "SKU-B", "", "139", "")
	if err != nil || c2.line.ID != 101 {
		t.Fatalf("level2: cand=%+v err=%v", c2, err)
	}

	// Level 3: sku+name
	c3, err := matchReconcileCandidate(idx, "", "SKU-C", "", "", "Iori")
	if err != nil || c3.line.ID != 102 {
		t.Fatalf("level3: cand=%+v err=%v", c3, err)
	}

	// Ambiguity
	idx.bySKUPhone[skuPhoneKey("SKU-B", "139")] = append(
		idx.bySKUPhone[skuPhoneKey("SKU-B", "139")],
		&reconcileCandidate{line: &domain.FulfillmentLine{ID: 999}, factorySKU: "SKU-B", phone: "139"},
	)
	if _, err := matchReconcileCandidate(idx, "", "SKU-B", "", "139", ""); err == nil {
		t.Fatal("expected ambiguity error")
	}

	// Zero hit
	if _, err := matchReconcileCandidate(idx, "", "NOPE", "", "000", "Nobody"); err == nil {
		t.Fatal("expected zero-hit error")
	}
}

func TestMatchReconcileCandidate_SupplierProductRefTiers(t *testing.T) {
	t.Parallel()

	flRefPhone := &domain.FulfillmentLine{ID: 201}
	flRefName := &domain.FulfillmentLine{ID: 202}
	idx := &reconcileIndex{
		byFLID: map[uint]*reconcileCandidate{},
		byRefPhone: map[string][]*reconcileCandidate{
			refPhoneKey("206068021", "138"): {{line: flRefPhone, supplierProductRef: "206068021", phone: "138"}},
		},
		byRefRecipient: map[string][]*reconcileCandidate{
			refRecipientKey("63098307", "Hina"): {{line: flRefName, supplierProductRef: "63098307", recipientName: "Hina"}},
		},
		bySKUPhone:       map[string][]*reconcileCandidate{},
		bySKURecipient:   map[string][]*reconcileCandidate{},
		solByFulfillment: map[uint]*domain.SupplierOrderLine{},
	}

	// Level 4: supplier_product_ref + phone
	c4, err := matchReconcileCandidate(idx, "", "", "206068021", "138", "")
	if err != nil || c4.line.ID != 201 {
		t.Fatalf("level4: cand=%+v err=%v", c4, err)
	}

	// Level 5: supplier_product_ref + recipient_name
	c5, err := matchReconcileCandidate(idx, "", "", "63098307", "", "Hina")
	if err != nil || c5.line.ID != 202 {
		t.Fatalf("level5: cand=%+v err=%v", c5, err)
	}

	// Ambiguity on ref+phone includes fl_ids detail
	idx.byRefPhone[refPhoneKey("206068021", "138")] = append(
		idx.byRefPhone[refPhoneKey("206068021", "138")],
		&reconcileCandidate{line: &domain.FulfillmentLine{ID: 299}, supplierProductRef: "206068021", phone: "138"},
	)
	err = nil
	_, err = matchReconcileCandidate(idx, "", "", "206068021", "138", "")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if !strings.Contains(err.Error(), "fl_ids=") {
		t.Fatalf("ambiguity error should include fl_ids detail, got %v", err)
	}
}

func TestParseSKUQuantityTokens(t *testing.T) {
	t.Parallel()

	// Self-contained synthetic multi-token shape; it is not paired with a
	// product-catalog fixture or any other sample document.
	raw := "206068021_设计师款透明单插立牌-底座可印刷15cm * 1|205721969_设计师款10x15cm单面印烫色明信片哑膜款 * 3|63098307_设计师款异形软磁冰箱贴-5cm * 1"
	tokens, err := ParseSKUQuantityTokens(raw)
	if err != nil {
		t.Fatalf("ParseSKUQuantityTokens: %v", err)
	}
	if len(tokens) != 3 {
		t.Fatalf("len=%d want 3", len(tokens))
	}
	want := []SKUQuantityToken{
		{SupplierProductRef: "206068021", Quantity: 1},
		{SupplierProductRef: "205721969", Quantity: 3},
		{SupplierProductRef: "63098307", Quantity: 1},
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Errorf("token[%d]=%+v want %+v", i, tokens[i], want[i])
		}
	}

	// Failures must not guess.
	for _, bad := range []string{
		"",
		"no_star_here",
		"_label * 1",           // empty digit prefix
		"ABC_label * 1",        // non-numeric prefix
		"206068021_label * x",  // non-int qty
		"206068021_label * 0",  // non-positive
		"206068021_label * 1|", // trailing empty segment
	} {
		if _, err := ParseSKUQuantityTokens(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestReconcileShipmentRow_MultiSKUExpandAndShippedAt(t *testing.T) {
	t.Parallel()

	fl1 := &domain.FulfillmentLine{ID: 301}
	fl2 := &domain.FulfillmentLine{ID: 302}
	sol1 := &domain.SupplierOrderLine{ID: 11, FulfillmentLineID: 301}
	sol2 := &domain.SupplierOrderLine{ID: 12, FulfillmentLineID: 302}
	idx := &reconcileIndex{
		byFLID: map[uint]*reconcileCandidate{},
		byRefPhone: map[string][]*reconcileCandidate{
			refPhoneKey("206068021", "13800000000"): {{line: fl1, supplierProductRef: "206068021", phone: "13800000000"}},
			refPhoneKey("63098307", "13800000000"):  {{line: fl2, supplierProductRef: "63098307", phone: "13800000000"}},
		},
		byRefRecipient:   map[string][]*reconcileCandidate{},
		bySKUPhone:       map[string][]*reconcileCandidate{},
		bySKURecipient:   map[string][]*reconcileCandidate{},
		solByFulfillment: map[uint]*domain.SupplierOrderLine{301: sol1, 302: sol2},
	}

	applied := map[string]string{
		"shipment.sku_quantity":         "206068021_standee * 1|63098307_magnet * 2",
		"shipment.phone":                "13800000000",
		"shipment.tracking_no":          "TRACK-M",
		"shipment.external_shipment_no": "EXT-M",
		"shipment.carrier_name":         "申通快递",
		"shipment.shipped_at":           "2026-05-10 09:17:20",
	}
	entries, err := reconcileShipmentRow(applied, idx, nil, context.Background(), 1)
	if err != nil {
		t.Fatalf("reconcileShipmentRow: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].FulfillmentLineID != 301 || entries[0].Quantity != 1 || entries[0].SupplierOrderLineID != 11 {
		t.Errorf("entry0 unexpected: %+v", entries[0])
	}
	if entries[1].FulfillmentLineID != 302 || entries[1].Quantity != 2 || entries[1].SupplierOrderLineID != 12 {
		t.Errorf("entry1 unexpected: %+v", entries[1])
	}
	if entries[0].TrackingNo != "TRACK-M" || entries[0].ExternalShipmentNo != "EXT-M" {
		t.Errorf("shared fields: %+v", entries[0])
	}
	if entries[0].ShippedAt == nil {
		t.Fatal("expected ShippedAt populated")
	}
	if got := entries[0].ShippedAt.Format("2006-01-02 15:04:05"); got != "2026-05-10 09:17:20" {
		t.Errorf("ShippedAt = %s, want 2026-05-10 09:17:20", got)
	}

	// Ambiguous multi-SKU token → whole row fails.
	idx.byRefPhone[refPhoneKey("206068021", "13800000000")] = append(
		idx.byRefPhone[refPhoneKey("206068021", "13800000000")],
		&reconcileCandidate{line: &domain.FulfillmentLine{ID: 399}, supplierProductRef: "206068021", phone: "13800000000"},
	)
	if _, err := reconcileShipmentRow(applied, idx, nil, context.Background(), 1); err == nil {
		t.Fatal("expected ambiguity error on multi-SKU expand")
	}
}

func TestMapAndReconcileShipments_ByExternalKey(t *testing.T) {
	t.Parallel()

	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()
	// buildImportFixture: FL 100/101, SOL 10/11 on wave 1.
	// Wire ProductID so index builds cleanly (not required for level-1 match).
	productRepo := &mockProductRepoForReconcile{products: map[uint]*domain.Product{}}
	addrRepo := newMockAddressRepo()

	// Seed template + binding via in-memory mapping service.
	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "p1", SourceSurface: string(domain.SourceSurfaceFactory), SupportsImportSupplierShipment: true,
	})
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "ship-return", DocumentType: "import_supplier_shipment", Format: "csv",
		MappingRules: `{
			"version": 2,
			"mode": "header",
			"hasHeader": true,
			"columns": {
				"shipment.third_party_order_no": "FL",
				"shipment.tracking_no": "Tracking",
				"shipment.external_shipment_no": "ExtNo",
				"shipment.carrier_code": "Carrier",
				"shipment.quantity": "Qty"
			},
			"transforms": {"shipment.tracking_no": ["strip_leading_quote"]}
		}`,
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
	uc = WithShipmentReconcileDeps(uc, mapping, productRepo, nil, addrRepo, nil)

	result, err := uc.MapAndReconcileShipments(context.Background(), dto.MapAndReconcileShipmentsInput{
		WaveID:               1,
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"FL": "100", "Tracking": "'TRACK-1", "ExtNo": "EXT-A", "Carrier": "SF", "Qty": "1"},
			{"FL": "101", "Tracking": "TRACK-2", "ExtNo": "EXT-B", "Carrier": "SF", "Qty": "1"},
		},
	})
	if err != nil {
		t.Fatalf("MapAndReconcileShipments: %v", err)
	}
	if result.SuccessCount != 2 || result.ErrorCount != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.CreatedShipments) != 2 {
		t.Fatalf("expected 2 shipments, got %d", len(result.CreatedShipments))
	}
	// strip_leading_quote applied.
	found := false
	for _, s := range result.CreatedShipments {
		if s.TrackingNo == "TRACK-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected TRACK-1 without leading quote, got %+v", result.CreatedShipments)
	}
}

// ── Carrier alias resolve + import ──

type mockCarrierMappingRepoFull struct {
	mu     sync.Mutex
	byID   map[uint]*domain.CarrierMapping
	lastID uint
}

func newMockCarrierMappingRepoFull() *mockCarrierMappingRepoFull {
	return &mockCarrierMappingRepoFull{byID: make(map[uint]*domain.CarrierMapping)}
}

func (m *mockCarrierMappingRepoFull) Create(ctx context.Context, mapping *domain.CarrierMapping) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	mapping.ID = m.lastID
	cp := *mapping
	m.byID[mapping.ID] = &cp
	return nil
}
func (m *mockCarrierMappingRepoFull) Update(ctx context.Context, mapping *domain.CarrierMapping) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *mapping
	m.byID[mapping.ID] = &cp
	return nil
}
func (m *mockCarrierMappingRepoFull) ListByProfile(ctx context.Context, profileID uint) ([]domain.CarrierMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.CarrierMapping
	for _, cm := range m.byID {
		if cm.IntegrationProfileID == profileID {
			out = append(out, *cm)
		}
	}
	return out, nil
}
func (m *mockCarrierMappingRepoFull) FindByProfileAndInternal(ctx context.Context, profileID uint, internalCode string) (*domain.CarrierMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cm := range m.byID {
		if cm.IntegrationProfileID == profileID && cm.InternalCarrierCode == internalCode {
			cp := *cm
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockCarrierMappingRepoFull) FindByProfileAndExternal(ctx context.Context, profileID uint, externalCode string) (*domain.CarrierMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cm := range m.byID {
		if cm.IntegrationProfileID == profileID && cm.ExternalCarrierCode == externalCode {
			cp := *cm
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockCarrierMappingRepoFull) Delete(ctx context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
	return nil
}

type mockIntegrationProfileRepoSimple struct {
	profiles map[uint]*domain.IntegrationProfile
}

func newMockIntegrationProfileRepoSimple() *mockIntegrationProfileRepoSimple {
	return &mockIntegrationProfileRepoSimple{profiles: make(map[uint]*domain.IntegrationProfile)}
}
func (m *mockIntegrationProfileRepoSimple) Create(ctx context.Context, p *domain.IntegrationProfile) error {
	if p.ID == 0 {
		p.ID = uint(len(m.profiles) + 1)
	}
	cp := *p
	m.profiles[p.ID] = &cp
	return nil
}
func (m *mockIntegrationProfileRepoSimple) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfile, error) {
	p, ok := m.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile %d not found", id)
	}
	cp := *p
	return &cp, nil
}
func (m *mockIntegrationProfileRepoSimple) FindByProfileKey(ctx context.Context, key string) (*domain.IntegrationProfile, error) {
	for _, p := range m.profiles {
		if p.ProfileKey == key {
			cp := *p
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("profile key %q not found", key)
}
func (m *mockIntegrationProfileRepoSimple) List(ctx context.Context) ([]domain.IntegrationProfile, error) {
	out := make([]domain.IntegrationProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		out = append(out, *p)
	}
	return out, nil
}
func (m *mockIntegrationProfileRepoSimple) Update(ctx context.Context, p *domain.IntegrationProfile) error {
	if p == nil || p.ID == 0 {
		return fmt.Errorf("profile ID is required")
	}
	cp := *p
	m.profiles[p.ID] = &cp
	return nil
}
func (m *mockIntegrationProfileRepoSimple) Delete(ctx context.Context, id uint) error {
	panic("not implemented")
}

func TestResolveByExternalOrAlias(t *testing.T) {
	t.Parallel()
	repo := newMockCarrierMappingRepoFull()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "p", SourceSurface: string(domain.SourceSurfaceRetail), RequiresCarrierMapping: true,
	})
	uc := NewCarrierMappingUseCase(repo, profileRepo)

	aliases, _ := json.Marshal([]string{"SFKY", "顺丰快运"})
	_, err := uc.CreateMapping(context.Background(), dto.CreateCarrierMappingInput{
		IntegrationProfileID: 1,
		InternalCarrierCode:  "SF",
		ExternalCarrierCode:  "shunfeng",
		ExternalCarrierName:  "顺丰",
		Aliases:              string(aliases),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Exact external
	internal, name, err := uc.ResolveByExternalOrAlias(context.Background(), 1, "shunfeng")
	if err != nil || internal != "SF" || name != "顺丰" {
		t.Fatalf("external: internal=%q name=%q err=%v", internal, name, err)
	}

	// Alias
	internal, _, err = uc.ResolveByExternalOrAlias(context.Background(), 1, "SFKY")
	if err != nil || internal != "SF" {
		t.Fatalf("alias: internal=%q err=%v", internal, err)
	}

	// External display name (factory returns often omit the code).
	internal, name, err = uc.ResolveByExternalOrAlias(context.Background(), 1, " 顺丰 ")
	if err != nil || internal != "SF" || name != "顺丰" {
		t.Fatalf("external name: internal=%q name=%q err=%v", internal, name, err)
	}

	// Exact internal still works via ResolveCarrier
	ext, _, err := uc.ResolveCarrier(context.Background(), 1, "SF")
	if err != nil || ext != "shunfeng" {
		t.Fatalf("internal resolve: ext=%q err=%v", ext, err)
	}

	_, err = uc.CreateMapping(context.Background(), dto.CreateCarrierMappingInput{
		IntegrationProfileID: 1,
		InternalCarrierCode:  "OTHER",
		ExternalCarrierCode:  "other-code",
		ExternalCarrierName:  "顺丰",
	})
	if err != nil {
		t.Fatalf("create ambiguous name mapping: %v", err)
	}
	if _, _, err := uc.ResolveByExternalOrAlias(context.Background(), 1, "顺丰"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous name error, got %v", err)
	}
}

func TestReconcileShipmentRowResolvesCarrierDisplayName(t *testing.T) {
	t.Parallel()
	carrierRepo := newMockCarrierMappingRepoFull()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{ID: 1, ProfileKey: "factory"})
	carrierUC := NewCarrierMappingUseCase(carrierRepo, profileRepo)
	if _, err := carrierUC.CreateMapping(context.Background(), dto.CreateCarrierMappingInput{
		IntegrationProfileID: 1,
		InternalCarrierCode:  "SF",
		ExternalCarrierCode:  "shunfeng",
		ExternalCarrierName:  "顺丰速运",
	}); err != nil {
		t.Fatalf("create mapping: %v", err)
	}

	fl := &domain.FulfillmentLine{ID: 42}
	idx := &reconcileIndex{
		byFLID: map[uint]*reconcileCandidate{42: {line: fl}},
		solByFulfillment: map[uint]*domain.SupplierOrderLine{
			42: {ID: 7, FulfillmentLineID: 42},
		},
	}
	entries, err := reconcileShipmentRow(map[string]string{
		"shipment.third_party_order_no": "42",
		"shipment.carrier_name":         "顺丰速运",
		"shipment.tracking_no":          "TRACK-1",
	}, idx, carrierUC, context.Background(), 1)
	if err != nil {
		t.Fatalf("reconcileShipmentRow: %v", err)
	}
	if len(entries) != 1 || entries[0].CarrierCode != "SF" || entries[0].CarrierName != "顺丰速运" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestImportCarrierMappings_UpsertByExternal(t *testing.T) {
	t.Parallel()
	repo := newMockCarrierMappingRepoFull()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "p", SourceSurface: string(domain.SourceSurfaceRetail), RequiresCarrierMapping: true,
	})

	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "carrier-map", DocumentType: "import_carrier_mapping", Format: "csv",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {
				"carrier.internal_carrier_code": "Internal",
				"carrier.external_carrier_code": "External",
				"carrier.external_carrier_name": "Name",
				"carrier.aliases": "Aliases"
			}
		}`,
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

	// Create
	r1, err := uc.ImportCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Internal": "SF", "External": "shunfeng", "Name": "顺丰", "Aliases": "SFKY,顺丰快运"},
		},
	})
	if err != nil {
		t.Fatalf("import1: %v", err)
	}
	if r1.CreatedCount != 1 || r1.UpdatedCount != 0 {
		t.Fatalf("create counts: %+v", r1)
	}

	// Upsert same external
	r2, err := uc.ImportCarrierMappings(context.Background(), dto.ImportCarrierMappingsInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Internal": "SF", "External": "shunfeng", "Name": "顺丰速运", "Aliases": `["SFKY"]`},
		},
	})
	if err != nil {
		t.Fatalf("import2: %v", err)
	}
	if r2.CreatedCount != 0 || r2.UpdatedCount != 1 {
		t.Fatalf("update counts: %+v", r2)
	}
	if r2.Mappings[0].ExternalCarrierName != "顺丰速运" {
		t.Fatalf("name not updated: %+v", r2.Mappings[0])
	}
}

// ── Catalog upsert ──

type mockProductMasterRepo struct {
	mu     sync.Mutex
	byKey  map[string]*domain.ProductMaster
	lastID uint
}

func newMockProductMasterRepo() *mockProductMasterRepo {
	return &mockProductMasterRepo{byKey: make(map[string]*domain.ProductMaster)}
}
func masterKey(platform, sku string) string { return platform + "\x00" + sku }

func (m *mockProductMasterRepo) Create(ctx context.Context, master *domain.ProductMaster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	master.ID = m.lastID
	cp := *master
	m.byKey[masterKey(master.SupplierPlatform, master.FactorySKU)] = &cp
	return nil
}
func (m *mockProductMasterRepo) FindByID(ctx context.Context, id uint) (*domain.ProductMaster, error) {
	panic("not implemented")
}
func (m *mockProductMasterRepo) List(ctx context.Context) ([]domain.ProductMaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.ProductMaster, 0, len(m.byKey))
	for _, master := range m.byKey {
		out = append(out, *master)
	}
	return out, nil
}
func (m *mockProductMasterRepo) FindByPlatformAndSKU(ctx context.Context, platform, sku string) (*domain.ProductMaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.byKey[masterKey(platform, sku)]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cp := *p
	return &cp, nil
}
func (m *mockProductMasterRepo) Update(ctx context.Context, master *domain.ProductMaster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *master
	m.byKey[masterKey(master.SupplierPlatform, master.FactorySKU)] = &cp
	return nil
}

func TestImportProductCatalog_Upsert(t *testing.T) {
	t.Parallel()
	masterRepo := newMockProductMasterRepo()
	profileRepo := newMockIntegrationProfileRepoSimple()
	_ = profileRepo.Create(context.Background(), &domain.IntegrationProfile{
		ID: 1, ProfileKey: "factory",
		SourceSurface:                string(domain.SourceSurfaceFactory),
		SupportsImportProductCatalog: true,
		FactorySupplierPlatform:      "rouzao",
		ConnectorKey:                 "factory-a",
	})

	templateRepo := newMockDocumentTemplateRepo()
	bindingRepo := newMockProfileTemplateBindingRepo()
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "catalog", DocumentType: "import_product_catalog", Format: "csv",
		MappingRules: `{
			"version": 2, "mode": "header", "hasHeader": true,
			"columns": {
				"product.factory_sku": "SKU",
				"product.name": "Name",
				"product.product_kind": "Kind"
			}
		}`,
	}
	if err := templateRepo.Create(context.Background(), tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if err := bindingRepo.Create(context.Background(), &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: 1, DocumentType: "import_product_catalog", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}
	mapping := NewTemplateMappingService(templateRepo, bindingRepo, profileRepo)

	uc := NewProductUseCase(masterRepo, nil, nil)
	uc = WithCatalogImportDeps(uc, mapping, profileRepo, nil)

	r1, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"SKU": "GOLD-1", "Name": "Gold Badge", "Kind": "badge"},
		},
	})
	if err != nil {
		t.Fatalf("import1: %v", err)
	}
	if r1.CreatedCount != 1 || r1.UpdatedCount != 0 {
		t.Fatalf("create: %+v", r1)
	}
	if r1.Masters[0].SupplierPlatform != "rouzao" {
		t.Fatalf("platform should prefer FactorySupplierPlatform, got %+v", r1.Masters[0])
	}

	r2, err := uc.ImportProductCatalog(context.Background(), dto.ImportProductCatalogInput{
		IntegrationProfileID: 1,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"SKU": "GOLD-1", "Name": "Gold Badge v2", "Kind": "badge"},
		},
	})
	if err != nil {
		t.Fatalf("import2: %v", err)
	}
	if r2.CreatedCount != 0 || r2.UpdatedCount != 1 {
		t.Fatalf("update: %+v", r2)
	}
	if r2.Masters[0].Name != "Gold Badge v2" {
		t.Fatalf("name not updated: %+v", r2.Masters[0])
	}
}

// ── Demand v2 namespaces ──

func TestMapDemandImportRow_Namespaces(t *testing.T) {
	t.Parallel()
	rules, err := ParseMappingRules(`{
		"version": 2,
		"mode": "header",
		"hasHeader": true,
		"columns": {
			"document.source_customer_ref": "UID",
			"document.display_name": "Nick",
			"line.external_title": "Title",
			"line.requested_quantity": "Qty",
			"recipient.name": "Recv",
			"recipient.phone": "Phone",
			"recipient.address_line1": "Addr"
		},
		"defaults": {"line.line_type": "sku_order"}
	}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	row, warnings, err := MapDemandImportRow(
		[]string{"u-1", "Hina", "Standee", "2", "日奈", "138", "HQ"},
		[]string{"UID", "Nick", "Title", "Qty", "Recv", "Phone", "Addr"},
		rules,
	)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if row.Line.ExternalTitle != "Standee" || row.Line.RequestedQuantity != 2 || row.Line.LineType != "sku_order" {
		t.Fatalf("line: %+v", row.Line)
	}
	if row.Document.SourceCustomerRef != "u-1" || row.Document.DisplayName != "Hina" {
		t.Fatalf("document: %+v", row.Document)
	}
	if row.Recipient == nil || row.Recipient.RecipientName != "日奈" || row.Recipient.Phone != "138" {
		t.Fatalf("recipient: %+v", row.Recipient)
	}
}
