package app

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestExportSupplierOrder_UsesFactorySupplierPlatform(t *testing.T) {
	t.Parallel()

	waveID := uint(77)
	profileID := uint(9)
	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	productID := uint(401)

	doc := &domain.DemandDocument{IntegrationProfileID: &profileID}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatal(err)
	}
	if err := fulfillRepo.Create(context.Background(), &domain.FulfillmentLine{
		WaveID:           waveID,
		Quantity:         2,
		DemandDocumentID: &doc.ID,
		ProductID:        &productID,
	}); err != nil {
		t.Fatal(err)
	}

	profileRepo := newMockProfileRepoForExport(&domain.IntegrationProfile{
		ID:                          profileID,
		SourceSurface:               string(domain.SourceSurfaceFactory),
		SupportsExportSupplierOrder: true,
		FactorySupplierPlatform:     "rouzao",
	})
	bindingRepo := &mockBindingRepo{defaultTemplateByProfile: map[uint]uint{profileID: 51}}
	productRepo := &mockProductRepoForExport{products: map[uint]*domain.Product{
		productID: {ID: productID, WaveID: waveID, SupplierPlatform: "rouzao", FactorySKU: "RZ-1"},
	}}
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, demandRepo, profileRepo, bindingRepo, productRepo)

	orders, err := exportUC.ExportSupplierOrderForProfile(context.Background(), waveID, profileID)
	if err != nil {
		t.Fatalf("ExportSupplierOrder: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].SupplierPlatform != "rouzao" {
		t.Fatalf("SupplierPlatform = %q, want rouzao (FactorySupplierPlatform must win)", orders[0].SupplierPlatform)
	}
}

func TestExportSupplierOrder_RejectsMissingFactorySupplierPlatform(t *testing.T) {
	t.Parallel()

	waveID := uint(78)
	profileID := uint(10)
	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	doc := &domain.DemandDocument{IntegrationProfileID: &profileID}
	if err := demandRepo.Create(context.Background(), doc); err != nil {
		t.Fatal(err)
	}
	if err := fulfillRepo.Create(context.Background(), &domain.FulfillmentLine{
		WaveID:           waveID,
		Quantity:         1,
		DemandDocumentID: &doc.ID,
	}); err != nil {
		t.Fatal(err)
	}

	profileRepo := newMockProfileRepoForExport(&domain.IntegrationProfile{
		ID:                          profileID,
		SourceSurface:               string(domain.SourceSurfaceFactory),
		SupportsExportSupplierOrder: true,
	})
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, demandRepo, profileRepo, &mockBindingRepo{}, &mockProductRepoForExport{products: map[uint]*domain.Product{}})

	_, err := exportUC.ExportSupplierOrderForProfile(context.Background(), waveID, profileID)
	if err == nil || !strings.Contains(err.Error(), "factorySupplierPlatform") {
		t.Fatalf("expected missing factorySupplierPlatform error, got %v", err)
	}
}

// mockProductRepoForExport is a minimal ProductRepository for export SKU tests.
type mockProductRepoForExport struct {
	products map[uint]*domain.Product
}

func (m *mockProductRepoForExport) Create(ctx context.Context, product *domain.Product) error {
	return nil
}
func (m *mockProductRepoForExport) FindByID(ctx context.Context, id uint) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, fmt.Errorf("product %d not found", id)
	}
	cp := *p
	return &cp, nil
}
func (m *mockProductRepoForExport) FindByWaveAndID(ctx context.Context, waveID uint, id uint) (*domain.Product, error) {
	return m.FindByID(ctx, id)
}
func (m *mockProductRepoForExport) ListByWave(ctx context.Context, waveID uint) ([]domain.Product, error) {
	return nil, nil
}
func (m *mockProductRepoForExport) FindByWaveAndSKU(ctx context.Context, waveID uint, platform, sku string) (*domain.Product, error) {
	return nil, nil
}
func (m *mockProductRepoForExport) DeleteByWave(ctx context.Context, waveID uint) error { return nil }

func TestExportSupplierOrder_FillsSupplierSKUFromFactorySKU(t *testing.T) {
	t.Parallel()

	waveID := uint(79)
	productID := uint(55)
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()

	if err := fulfillRepo.Create(context.Background(), &domain.FulfillmentLine{
		WaveID:    waveID,
		Quantity:  3,
		ProductID: &productID,
	}); err != nil {
		t.Fatal(err)
	}

	productRepo := &mockProductRepoForExport{
		products: map[uint]*domain.Product{
			productID: {ID: productID, WaveID: waveID, SupplierPlatform: "rouzao", FactorySKU: "RZ-SKU-001"},
		},
	}
	profileID := uint(11)
	profileRepo := newMockProfileRepoForExport(&domain.IntegrationProfile{
		ID: profileID, SourceSurface: string(domain.SourceSurfaceFactory),
		SupportsExportSupplierOrder: true, FactorySupplierPlatform: "rouzao",
	})
	bindingRepo := &mockBindingRepo{defaultTemplateByProfile: map[uint]uint{profileID: 52}}
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, nil, profileRepo, bindingRepo, productRepo)
	orders, err := exportUC.ExportSupplierOrderForProfile(context.Background(), waveID, profileID)
	if err != nil {
		t.Fatalf("ExportSupplierOrder: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	lines, err := supplierRepo.ListLinesByOrder(context.Background(), orders[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].SupplierSKU != "RZ-SKU-001" {
		t.Fatalf("SupplierSKU = %q, want RZ-SKU-001", lines[0].SupplierSKU)
	}
}

func TestExportSupplierOrderForProfileFiltersOtherFactoryPlatforms(t *testing.T) {
	t.Parallel()
	waveID, profileID := uint(80), uint(12)
	productA, productB := uint(61), uint(62)
	fulfillRepo := newMockFulfillRepo()
	supplierRepo := newMockSupplierRepo()
	for _, line := range []*domain.FulfillmentLine{
		{WaveID: waveID, Quantity: 2, ProductID: &productA},
		{WaveID: waveID, Quantity: 3, ProductID: &productB},
	} {
		if err := fulfillRepo.Create(context.Background(), line); err != nil {
			t.Fatal(err)
		}
	}
	profileRepo := newMockProfileRepoForExport(&domain.IntegrationProfile{
		ID: profileID, SourceSurface: string(domain.SourceSurfaceFactory),
		SupportsExportSupplierOrder: true, FactorySupplierPlatform: "factory-a",
	})
	bindingRepo := &mockBindingRepo{defaultTemplateByProfile: map[uint]uint{profileID: 53}}
	productRepo := &mockProductRepoForExport{products: map[uint]*domain.Product{
		productA: {ID: productA, SupplierPlatform: "factory-a", FactorySKU: "A-1"},
		productB: {ID: productB, SupplierPlatform: "factory-b", FactorySKU: "B-1"},
	}}
	exportUC := NewExportUseCase(supplierRepo, fulfillRepo, nil, nil, profileRepo, bindingRepo, productRepo)
	orders, err := exportUC.ExportSupplierOrderForProfile(context.Background(), waveID, profileID)
	if err != nil {
		t.Fatalf("ExportSupplierOrderForProfile: %v", err)
	}
	lines, err := supplierRepo.ListLinesByOrder(context.Background(), orders[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].SupplierSKU != "A-1" || lines[0].SubmittedQuantity != 2 {
		t.Fatalf("filtered order lines = %+v", lines)
	}
}

func TestRenderTrackingExportCSV_UsesColumnOrder(t *testing.T) {
	t.Parallel()

	rules := &TemplateMappingRules{
		Version: 2,
		Mode:    "header",
		Columns: map[string]string{
			"export.tracking_no":          "物流单号",
			"export.carrier_code":         "快递公司",
			"export.third_party_order_no": "第三方订单号",
			"export.shipment_id":          "发货单",
		},
		ColumnOrder: []string{
			"export.third_party_order_no",
			"export.carrier_code",
			"export.tracking_no",
		},
	}

	items := []domain.ChannelSyncItem{
		{
			ID:                1,
			FulfillmentLineID: 42,
			ShipmentID:        7,
			TrackingNo:        "SF123",
			CarrierCode:       "SF",
		},
	}

	csvText, err := NewTemplatePayloadRenderer().RenderTrackingExportCSV(items, rules)
	if err != nil {
		t.Fatalf("RenderTrackingExportCSV: %v", err)
	}

	r := csv.NewReader(strings.NewReader(csvText))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header + 1 data row, got %d", len(rows))
	}
	// columnOrder first three + remainder (shipment_id) alphabetically after.
	wantHeaders := []string{"第三方订单号", "快递公司", "物流单号", "发货单"}
	if len(rows[0]) != len(wantHeaders) {
		t.Fatalf("headers len=%d want %d: %v", len(rows[0]), len(wantHeaders), rows[0])
	}
	for i, h := range wantHeaders {
		if rows[0][i] != h {
			t.Fatalf("header[%d]=%q want %q (full=%v)", i, rows[0][i], h, rows[0])
		}
	}
	if rows[1][0] != "42" || rows[1][1] != "SF" || rows[1][2] != "SF123" || rows[1][3] != "7" {
		t.Fatalf("data row = %v", rows[1])
	}
}

func TestCSVExportExecutor_UsesTrackingTemplateColumnOrder(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	profileID := uint(3)
	tmplID := uint(11)

	tmplRepo := &mockTemplateRepoForExport{
		templates: map[uint]*domain.DocumentTemplate{
			tmplID: {
				ID:           tmplID,
				TemplateKey:  "tracking_csv",
				DocumentType: "export_source_tracking_update",
				Format:       "csv",
				MappingRules: `{
					"version":2,
					"mode":"header",
					"columns":{
						"export.tracking_no":"物流单号",
						"export.third_party_order_no":"第三方订单号",
						"export.carrier_code":"快递"
					},
					"columnOrder":["export.third_party_order_no","export.tracking_no","export.carrier_code"]
				}`,
			},
		},
	}
	bindingRepo := &mockBindingRepoWithDefault{
		binding: &domain.IntegrationProfileTemplateBinding{
			ID:                   1,
			IntegrationProfileID: profileID,
			DocumentType:         "export_source_tracking_update",
			TemplateID:           tmplID,
			IsDefault:            true,
		},
	}

	exec := NewCSVExportExecutor(outputDir, &TrackingTemplateSource{
		BindingRepo:  bindingRepo,
		TemplateRepo: tmplRepo,
	})

	job := &domain.ChannelSyncJob{ID: 100, IntegrationProfileID: profileID}
	items := []domain.ChannelSyncItem{{
		ID: 1, FulfillmentLineID: 9, TrackingNo: "YT999", CarrierCode: "YTO",
	}}
	profile := &domain.IntegrationProfile{
		ID: profileID, ConnectorKey: "eli.csv_export",
		SourceSurface: string(domain.SourceSurfaceRetail), TrackingSyncMode: "document_export",
	}

	result, err := exec.Execute(context.Background(), job, items, profile)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.AggregateStatus != "success" {
		t.Fatalf("status=%q", result.AggregateStatus)
	}

	// Find the written CSV.
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 export file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(outputDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatalf("rows=%v", rows)
	}
	if rows[0][0] != "第三方订单号" || rows[0][1] != "物流单号" || rows[0][2] != "快递" {
		t.Fatalf("headers=%v", rows[0])
	}
	if rows[1][0] != "9" || rows[1][1] != "YT999" || rows[1][2] != "YTO" {
		t.Fatalf("data=%v", rows[1])
	}
}

func TestExportFieldValue_Sources(t *testing.T) {
	t.Parallel()

	flID := uint(88)
	productID := uint(5)
	addrID := uint(6)
	fl := &domain.FulfillmentLine{
		ID:                flID,
		ProductID:         &productID,
		CustomerAddressID: &addrID,
		Quantity:          4,
	}
	line := &domain.SupplierOrderLine{
		FulfillmentLineID: flID,
		SupplierLineNo:    1,
		SupplierSKU:       "FALLBACK-SKU",
		SubmittedQuantity: 4,
	}
	product := &domain.Product{ID: productID, FactorySKU: "FACT-001"}
	addr := &domain.CustomerAddress{
		ID:            addrID,
		RecipientName: "张三",
		Phone:         "13800000000",
		Province:      "浙江",
		City:          "杭州",
		AddressLine1:  "西湖路1号",
	}

	r := NewTemplatePayloadRenderer()
	ctx := ExportRowContext{Line: line, Fulfill: fl, Product: product, Address: addr}

	cases := map[string]string{
		"export.third_party_order_no": "88",
		"export.recipient":            "张三",
		"export.phone":                "13800000000",
		"export.address":              "浙江 杭州 西湖路1号",
		"export.factory_sku":          "FACT-001",
		"export.quantity":             "4",
	}
	for field, want := range cases {
		if got := r.exportFieldValue(ctx, field); got != want {
			t.Errorf("field %q = %q, want %q", field, got, want)
		}
	}
}

// ---- test doubles for template resolution ----

type mockTemplateRepoForExport struct {
	templates map[uint]*domain.DocumentTemplate
}

func (m *mockTemplateRepoForExport) Create(ctx context.Context, t *domain.DocumentTemplate) error {
	return nil
}
func (m *mockTemplateRepoForExport) FindByID(ctx context.Context, id uint) (*domain.DocumentTemplate, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, fmt.Errorf("template %d not found", id)
	}
	cp := *t
	return &cp, nil
}
func (m *mockTemplateRepoForExport) FindByKey(ctx context.Context, key string) (*domain.DocumentTemplate, error) {
	return nil, nil
}
func (m *mockTemplateRepoForExport) List(ctx context.Context) ([]domain.DocumentTemplate, error) {
	return nil, nil
}
func (m *mockTemplateRepoForExport) ListByDocumentType(ctx context.Context, docType string) ([]domain.DocumentTemplate, error) {
	return nil, nil
}
func (m *mockTemplateRepoForExport) Update(ctx context.Context, t *domain.DocumentTemplate) error {
	return nil
}
func (m *mockTemplateRepoForExport) Delete(ctx context.Context, id uint) error {
	return nil
}

type mockBindingRepoWithDefault struct {
	binding *domain.IntegrationProfileTemplateBinding
}

func (m *mockBindingRepoWithDefault) Create(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	return nil
}
func (m *mockBindingRepoWithDefault) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (m *mockBindingRepoWithDefault) ListByProfile(ctx context.Context, profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (m *mockBindingRepoWithDefault) ListByTemplateID(ctx context.Context, templateID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	return nil, nil
}
func (m *mockBindingRepoWithDefault) FindDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	if m.binding == nil || m.binding.IntegrationProfileID != profileID || m.binding.DocumentType != docType {
		return nil, fmt.Errorf("no default binding for profile %d type %s", profileID, docType)
	}
	cp := *m.binding
	return &cp, nil
}
func (m *mockBindingRepoWithDefault) ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error {
	return nil
}
func (m *mockBindingRepoWithDefault) Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	return nil
}
func (m *mockBindingRepoWithDefault) Delete(ctx context.Context, id uint) error { return nil }
func (m *mockBindingRepoWithDefault) CountByProfileID(ctx context.Context, profileID uint) (int64, error) {
	return 0, nil
}
