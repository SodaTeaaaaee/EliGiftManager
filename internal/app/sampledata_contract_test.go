package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
)

const (
	sampleMembership = "需求平台——哔哩哔哩/从需求平台导出-会员列表.csv"
	sampleSales      = "需求平台——哔哩哔哩/从需求平台导出-单个订单数据.xls"
	sampleCarrier    = "需求平台——哔哩哔哩/从需求平台导出-快递编码映射关系.xls"
	sampleTracking   = "需求平台——哔哩哔哩/需要导入需求平台-订单快递跟踪.xls"
	sampleCatalog    = "工厂平台——柔造/从工厂平台导出-商品列表.zip"
	sampleShipment   = "工厂平台——柔造/从工厂平台导出-快递订单数据.csv"
	sampleOrder      = "工厂平台——柔造/需要导入工厂平台-批量下单表格.xlsx"
)

func sampleDataPath(t *testing.T, relative string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("SampleData contract skipped: repository root unavailable")
	}
	path := filepath.Join(filepath.Dir(source), "..", "..", "SampleData", filepath.FromSlash(relative))
	if _, err := os.Stat(path); err != nil {
		t.Skipf("SampleData fixture missing: %s", relative)
	}
	return path
}

func sampleContractFailure(t *testing.T, stage string) {
	t.Helper()
	t.Fatalf("SampleData contract failed during %s (row contents intentionally redacted)", stage)
}

func seedBilibiliSampleContracts(t *testing.T, gdb *gorm.DB) *domain.IntegrationProfile {
	t.Helper()
	ctx := context.Background()
	profiles := infra.NewIntegrationProfileRepository(gdb)
	if _, err := SeedBilibiliDemo(ctx, profiles, infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb)); err != nil {
		sampleContractFailure(t, "Bilibili profile/template seed")
	}
	membership, err := profiles.FindByProfileKey(ctx, BilibiliDemoProfileKey)
	if err != nil || membership == nil {
		sampleContractFailure(t, "membership profile lookup")
	}
	return membership
}

func seedCatalogSampleContracts(t *testing.T, gdb *gorm.DB) *domain.IntegrationProfile {
	t.Helper()
	profile, err := SeedCatalogDemo(context.Background(), infra.NewIntegrationProfileRepository(gdb), infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb))
	if err != nil || profile == nil {
		sampleContractFailure(t, "factory profile/template seed")
	}
	return profile
}

func runDemandSampleContract(t *testing.T, path, profileKey, documentType string) {
	t.Helper()
	gdb := durableEvidenceDB(t)
	seedBilibiliSampleContracts(t, gdb)
	ctx := context.Background()
	profile, err := infra.NewIntegrationProfileRepository(gdb).FindByProfileKey(ctx, profileKey)
	if err != nil || profile == nil {
		sampleContractFailure(t, "demand profile lookup")
	}
	mapping := NewTemplateMappingService(infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb), infra.NewIntegrationProfileRepository(gdb))
	_, rules, err := mapping.ResolveDemandImportTemplateAndRules(ctx, profile.ID, documentType, "")
	if err != nil {
		sampleContractFailure(t, "demand template resolution")
	}
	sheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{HasHeader: rules.HasHeader, Encoding: "auto", SheetName: rules.SheetName})
	if err != nil || len(sheet.Rows) == 0 {
		sampleContractFailure(t, "demand source parse")
	}
	_, mapped, rowErrors, _, err := mapping.BuildDemandImportPipelineWithMode(ctx, profile.ID, documentType, nil, sheet.Rows, sheet.Headers, "skip_invalid")
	if err != nil {
		sampleContractFailure(t, "demand row mapping")
	}
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	rawRows, unmapped := BuildImportEvidenceRows(sheet.Rows, sheet.Headers, nil)
	run, records, err := evidence.StartImportEvidence(ctx, "demand", profile.ID, "skip_invalid", path, `{"contract":"sampledata"}`, rawRows, unmapped, nil)
	if err != nil || run == nil {
		sampleContractFailure(t, "demand RAW persistence")
	}
	for _, rowErr := range rowErrors {
		MarkImportEvidenceFailure(records, rowErr.RowIndex, "mapping_error", rowErr.Reason, nil)
	}
	intake := NewDemandIntakeUseCase(infra.NewDemandRepository(gdb))
	for i := range mapped {
		doc := &domain.DemandDocument{Kind: profile.DemandKind, CaptureMode: "sample_contract", SourceChannel: profile.SourceChannel, SourceSurface: profile.SourceSurface, IntegrationProfileID: &profile.ID, SourceDocumentNo: mapped[i].Document.SourceDocumentNo, SourceCustomerRef: mapped[i].Document.SourceCustomerRef}
		line := *mapped[i].Line
		if err := intake.ImportDemand(ctx, doc, []*domain.DemandLine{&line}); err != nil {
			sampleContractFailure(t, "demand persistence")
		}
		MarkImportEvidenceSuccess(records, line.SourceLineNo-1, "demand_document", doc.ID)
	}
	status := "completed"
	if len(rowErrors) > 0 {
		status = "partial_success"
	}
	if err := evidence.CompleteImportEvidence(ctx, run, records, status); err != nil {
		sampleContractFailure(t, "demand evidence staging")
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		sampleContractFailure(t, "demand evidence finalize")
	}
	var persisted int64
	if err := gdb.Table("demand_lines").Count(&persisted).Error; err != nil || int(persisted) != len(mapped) {
		sampleContractFailure(t, "demand persisted-row count")
	}
	detail, err := NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)).GetRunDetail(ctx, run.ID)
	if err != nil || detail.Run.RecordCount != len(sheet.Rows) || detail.Run.SuccessCount+detail.Run.FailureCount+detail.Run.QuarantinedCount != detail.Run.RecordCount {
		sampleContractFailure(t, "demand RAW arithmetic")
	}
}

func TestSampleDataMembershipDemandReadOnlyContract(t *testing.T) {
	runDemandSampleContract(t, sampleDataPath(t, sampleMembership), BilibiliDemoProfileKey, BilibiliImportEntitlementDocType)
}

func TestSampleDataSalesDemandReadOnlyContract(t *testing.T) {
	runDemandSampleContract(t, sampleDataPath(t, sampleSales), BilibiliDemoProfileKey, BilibiliImportSalesOrderDocType)
}

func TestSampleDataCatalogReadOnlyContract(t *testing.T) {
	path := sampleDataPath(t, sampleCatalog)
	gdb := durableEvidenceDB(t)
	profile := seedCatalogSampleContracts(t, gdb)
	ctx := context.Background()
	mapping := NewTemplateMappingService(infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb), infra.NewIntegrationProfileRepository(gdb))
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	uc := NewProductUseCase(infra.NewProductMasterRepository(gdb), infra.NewProductRepository(gdb), infra.NewWaveRepository(gdb))
	uc = WithCatalogImportDeps(uc, mapping, infra.NewIntegrationProfileRepository(gdb), service.NewAssetStoreAt(filepath.Join(t.TempDir(), "assets")))
	uc = WithCatalogImportEvidence(uc, evidence)
	result, err := uc.ImportProductCatalog(ctx, dto.ImportProductCatalogInput{IntegrationProfileID: profile.ID, ImportMode: "skip_invalid", FilePath: path})
	if err != nil || result.SuccessCount == 0 || result.SuccessCount+result.ErrorCount != result.TotalProcessed {
		sampleContractFailure(t, "catalog parse/map/persist")
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		sampleContractFailure(t, "catalog evidence finalize")
	}
	detail, err := NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb)).GetRunDetail(ctx, result.ImportRunID)
	if err != nil || detail.Run.RecordCount != result.TotalProcessed || len(detail.Records) == 0 || detail.Records[0].AssetMembers == "[]" || !strings.Contains(detail.Records[0].AssetMembers, "sha256") {
		sampleContractFailure(t, "catalog RAW asset metadata")
	}
}

func TestSampleDataCarrierReadOnlyContract(t *testing.T) {
	path := sampleDataPath(t, sampleCarrier)
	gdb := durableEvidenceDB(t)
	membership := seedBilibiliSampleContracts(t, gdb)
	ctx := context.Background()
	mappingRepo := infra.NewCarrierMappingRepository(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	mapping := NewTemplateMappingService(infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb), profileRepo)
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	preflight := newCarrierPreflightUC(gdb, mapping, mappingRepo, profileRepo, evidence)
	plan, err := preflight.PreflightCarrierMappings(ctx, dto.ImportCarrierMappingsInput{IntegrationProfileID: membership.ID, ImportMode: "skip_invalid", FilePath: path})
	if err != nil {
		sampleContractFailure(t, "carrier full-file preflight")
	}
	var result *dto.ImportCarrierMappingsResult
	err = gdb.Transaction(func(tx *gorm.DB) error {
		uc := NewCarrierMappingUseCase(infra.NewCarrierMappingRepository(tx), infra.NewIntegrationProfileRepository(tx))
		uc = WithCarrierImportEvidence(uc, evidence)
		uc = WithExternalCarrierRegistry(uc, NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(tx)))
		var executeErr error
		result, executeErr = uc.ExecuteCarrierImportPlan(ctx, plan)
		return executeErr
	})
	if err != nil || result == nil {
		sampleContractFailure(t, "carrier valid-row persistence")
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		sampleContractFailure(t, "carrier evidence finalize")
	}
	if result.SuccessCount == 0 || result.ErrorCount == 0 || result.SuccessCount+result.ErrorCount != result.TotalProcessed {
		sampleContractFailure(t, "carrier result arithmetic")
	}
	var carriers, conflicts, raw int64
	if err := gdb.Table("external_carriers").Count(&carriers).Error; err != nil {
		sampleContractFailure(t, "carrier registry count")
	}
	if err := gdb.Table("external_carrier_conflicts").Count(&conflicts).Error; err != nil {
		sampleContractFailure(t, "carrier conflict count")
	}
	if err := gdb.Table("import_raw_records").Count(&raw).Error; err != nil {
		sampleContractFailure(t, "carrier RAW count")
	}
	if int(carriers) != result.SuccessCount || conflicts == 0 || int(raw) != result.TotalProcessed {
		sampleContractFailure(t, "carrier registry/conflict/RAW contract")
	}
}

func TestSampleDataShipmentReadOnlyContract(t *testing.T) {
	path := sampleDataPath(t, sampleShipment)
	gdb := durableEvidenceDB(t)
	profile := seedCatalogSampleContracts(t, gdb)
	ctx := context.Background()
	rules, err := ParseMappingRules(ShipmentDemoMappingRules)
	if err != nil {
		sampleContractFailure(t, "shipment rule parse")
	}
	sheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{HasHeader: rules.HasHeader, Encoding: "auto"})
	if err != nil || len(sheet.Rows) == 0 {
		sampleContractFailure(t, "shipment source parse")
	}
	applied, _, err := ApplyRow(sheet.Rows[0], sheet.Headers, rules)
	if err != nil {
		sampleContractFailure(t, "shipment first-row mapping")
	}
	tokens, err := ParseSKUQuantityTokens(applied["shipment.sku_quantity"])
	if err != nil || len(tokens) == 0 {
		sampleContractFailure(t, "shipment multi-SKU parse")
	}
	wave := &domain.Wave{WaveNo: "sample-contract", Name: "sample-contract", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(gdb).Create(ctx, wave); err != nil {
		sampleContractFailure(t, "shipment wave seed")
	}
	masterRepo := infra.NewProductMasterRepository(gdb)
	productRepo := infra.NewProductRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	customer := &domain.CustomerProfile{DisplayName: "sample-contract", ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := infra.NewProfileRepository(gdb).Create(ctx, customer); err != nil {
		sampleContractFailure(t, "shipment customer seed")
	}
	addressRepo := infra.NewAddressRepository(gdb)
	address := &domain.CustomerAddress{CustomerProfileID: customer.ID, RecipientName: applied["shipment.recipient_name"], Phone: applied["shipment.phone"], AddressLine1: "sample-contract", ValidationStatus: "unverified"}
	if err := addressRepo.Create(ctx, address); err != nil {
		sampleContractFailure(t, "shipment address seed")
	}
	order := &domain.SupplierOrder{WaveID: wave.ID, FactoryIntegrationProfileID: &profile.ID, SupplierPlatform: profile.FactorySupplierPlatform, BatchNo: "sample-contract", Status: "submitted"}
	if err := supplierRepo.Create(ctx, order); err != nil {
		sampleContractFailure(t, "shipment supplier-order seed")
	}
	for i, token := range tokens {
		master := &domain.ProductMaster{SupplierPlatform: profile.FactorySupplierPlatform, FactorySKU: "SAMPLE_" + token.SupplierProductRef, SupplierProductRef: token.SupplierProductRef, Name: "sample-contract"}
		if err := masterRepo.Create(ctx, master); err != nil {
			sampleContractFailure(t, "shipment product-master seed")
		}
		product := &domain.Product{WaveID: wave.ID, ProductMasterID: &master.ID, SupplierPlatform: master.SupplierPlatform, FactorySKU: master.FactorySKU, Name: master.Name}
		if err := productRepo.Create(ctx, product); err != nil {
			sampleContractFailure(t, "shipment product seed")
		}
		line := &domain.FulfillmentLine{WaveID: wave.ID, CustomerProfileID: &customer.ID, ProductID: &product.ID, CustomerAddressID: &address.ID, Quantity: token.Quantity, AllocationState: "allocated", SupplierState: "submitted", ChannelSyncState: "pending"}
		if err := fulfillRepo.Create(ctx, line); err != nil {
			sampleContractFailure(t, "shipment fulfillment seed")
		}
		orderLine := &domain.SupplierOrderLine{SupplierOrderID: order.ID, FulfillmentLineID: line.ID, SupplierLineNo: i + 1, SupplierSKU: master.FactorySKU, SubmittedQuantity: token.Quantity, Status: "submitted"}
		if err := supplierRepo.CreateLine(ctx, orderLine); err != nil {
			sampleContractFailure(t, "shipment supplier-order-line seed")
		}
	}
	mapping := NewTemplateMappingService(infra.NewDocumentTemplateRepository(gdb), infra.NewProfileTemplateBindingRepository(gdb), infra.NewIntegrationProfileRepository(gdb))
	evidence := NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))
	uc := NewShipmentImportUseCase(infra.NewShipmentRepository(gdb), supplierRepo, fulfillRepo, nil)
	uc = WithShipmentReconcileDeps(uc, mapping, productRepo, masterRepo, addressRepo, nil)
	uc = WithShipmentImportEvidence(uc, evidence)
	result, err := uc.MapAndReconcileShipments(ctx, dto.MapAndReconcileShipmentsInput{WaveID: wave.ID, IntegrationProfileID: profile.ID, ImportMode: "skip_invalid", FilePath: path})
	if err != nil || result.SuccessCount < len(tokens) {
		sampleContractFailure(t, "shipment map/reconcile/persist")
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		sampleContractFailure(t, "shipment evidence finalize")
	}
	var shipments, raw int64
	_ = gdb.Table("shipments").Count(&shipments).Error
	_ = gdb.Table("import_raw_records").Count(&raw).Error
	if shipments == 0 || int(raw) != len(sheet.Rows) {
		sampleContractFailure(t, "shipment persisted/RAW count")
	}
}

func TestSampleDataSupplierOrderReadOnlyContract(t *testing.T) {
	path := sampleDataPath(t, sampleOrder)
	gdb := durableEvidenceDB(t)
	profile := seedCatalogSampleContracts(t, gdb)
	ctx := context.Background()
	rules, err := ParseMappingRules(SupplierOrderDemoMappingRules)
	if err != nil {
		sampleContractFailure(t, "supplier-order rule parse")
	}
	sampleSheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{HasHeader: true, SheetName: rules.SheetName})
	if err != nil || len(sampleSheet.Headers) != len(rules.ColumnOrder) {
		sampleContractFailure(t, "supplier-order sample parse")
	}
	wave := &domain.Wave{WaveNo: "sample-order", Name: "sample-order", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(gdb).Create(ctx, wave); err != nil {
		sampleContractFailure(t, "supplier-order wave seed")
	}
	master := &domain.ProductMaster{SupplierPlatform: profile.FactorySupplierPlatform, FactorySKU: "sample-sku", Name: "sample-contract"}
	if err := infra.NewProductMasterRepository(gdb).Create(ctx, master); err != nil {
		sampleContractFailure(t, "supplier-order product-master seed")
	}
	product := &domain.Product{WaveID: wave.ID, ProductMasterID: &master.ID, SupplierPlatform: master.SupplierPlatform, FactorySKU: master.FactorySKU, Name: master.Name}
	if err := infra.NewProductRepository(gdb).Create(ctx, product); err != nil {
		sampleContractFailure(t, "supplier-order product seed")
	}
	fulfill := &domain.FulfillmentLine{WaveID: wave.ID, ProductID: &product.ID, Quantity: 1, AllocationState: "allocated", SupplierState: "draft", ChannelSyncState: "pending"}
	if err := infra.NewFulfillmentRepository(gdb).Create(ctx, fulfill); err != nil {
		sampleContractFailure(t, "supplier-order fulfillment seed")
	}
	template, err := infra.NewDocumentTemplateRepository(gdb).FindByKey(ctx, SupplierOrderDemoTemplateKey)
	if err != nil || template == nil {
		sampleContractFailure(t, "supplier-order template lookup")
	}
	order := &domain.SupplierOrder{WaveID: wave.ID, FactoryIntegrationProfileID: &profile.ID, SupplierPlatform: profile.FactorySupplierPlatform, TemplateID: strconv.FormatUint(uint64(template.ID), 10), BatchNo: "sample-contract", Status: "draft"}
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	if err := supplierRepo.Create(ctx, order); err != nil {
		sampleContractFailure(t, "supplier-order persistence")
	}
	if err := supplierRepo.CreateLine(ctx, &domain.SupplierOrderLine{SupplierOrderID: order.ID, FulfillmentLineID: fulfill.ID, SupplierLineNo: 1, SupplierSKU: master.FactorySKU, SubmittedQuantity: 1, Status: "draft"}); err != nil {
		sampleContractFailure(t, "supplier-order line persistence")
	}
	outputDir := t.TempDir()
	writer := NewSupplierOrderFileWriter(supplierRepo, outputDir, &SupplierOrderFileWriterOptions{FulfillRepo: infra.NewFulfillmentRepository(gdb), ProductRepo: infra.NewProductRepository(gdb), AddressRepo: infra.NewAddressRepository(gdb), TemplateRepo: infra.NewDocumentTemplateRepository(gdb)})
	generated, err := writer.GenerateSupplierOrderFile(ctx, order.ID)
	if err != nil || filepath.Ext(generated.FilePath) != ".xlsx" {
		sampleContractFailure(t, "supplier-order render")
	}
	generatedSheet, err := tabular.ReadTabularFile(generated.FilePath, tabular.ReadOptions{HasHeader: true, SheetName: rules.SheetName})
	if err != nil || len(generatedSheet.Rows) != 1 || strings.Join(generatedSheet.Headers, "\x00") != strings.Join(sampleSheet.Headers, "\x00") {
		sampleContractFailure(t, "supplier-order rendered workbook contract")
	}
}

func TestSampleDataTrackingReadOnlyContract(t *testing.T) {
	path := sampleDataPath(t, sampleTracking)
	gdb := durableEvidenceDB(t)
	profile := seedBilibiliSampleContracts(t, gdb)
	ctx := context.Background()
	rules, err := ParseMappingRules(BilibiliExportTrackingMappingRules)
	if err != nil {
		sampleContractFailure(t, "tracking rule parse")
	}
	sampleSheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{HasHeader: true})
	if err != nil || len(sampleSheet.Headers) != len(rules.ColumnOrder) {
		sampleContractFailure(t, "tracking sample parse")
	}
	wave := &domain.Wave{WaveNo: "sample-tracking", Name: "sample-tracking", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(gdb).Create(ctx, wave); err != nil {
		sampleContractFailure(t, "tracking wave seed")
	}
	job := &domain.ChannelSyncJob{WaveID: wave.ID, IntegrationProfileID: profile.ID, Direction: "push_tracking", Status: "pending"}
	if err := infra.NewChannelSyncRepository(gdb).CreateJob(ctx, job); err != nil {
		sampleContractFailure(t, "tracking job persistence")
	}
	items := []domain.ChannelSyncItem{{ChannelSyncJobID: job.ID, ExternalDocumentNo: "sample-order", CarrierCode: "sample-carrier", TrackingNo: "sample-tracking"}}
	data, err := NewTemplatePayloadRenderer().RenderTrackingExportXLSX(items, rules)
	if err != nil {
		sampleContractFailure(t, "tracking render")
	}
	generatedPath := filepath.Join(t.TempDir(), "tracking.xlsx")
	if err := os.WriteFile(generatedPath, data, 0o600); err != nil {
		sampleContractFailure(t, "tracking rendered-file write")
	}
	generatedSheet, err := tabular.ReadTabularFile(generatedPath, tabular.ReadOptions{HasHeader: true})
	if err != nil || len(generatedSheet.Rows) != 1 || strings.Join(generatedSheet.Headers, "\x00") != strings.Join(sampleSheet.Headers, "\x00") {
		sampleContractFailure(t, "tracking rendered workbook contract")
	}
}
