package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// Catalog demo seed keys — idempotent by ProfileKey / TemplateKey.
const (
	CatalogDemoProfileKey  = "factory_rouzao_demo"
	CatalogDemoTemplateKey = "catalog_rouzao_zip_demo"
	CatalogDemoDocType     = "import_product_catalog"

	// Shipment return template for the same factory_rouzao_demo profile.
	ShipmentDemoTemplateKey = "shipment_rouzao_csv_demo"
	ShipmentDemoDocType     = "import_supplier_shipment"

	// Supplier-order export contract for the independent Rouzao batch-order
	// workbook sample. This template is deliberately independent of the catalog
	// and shipment-return samples.
	SupplierOrderDemoTemplateKey = "supplier_order_rouzao_xlsx_demo"
	SupplierOrderDemoDocType     = "export_supplier_order"
)

// SupplierOrderDemoMappingRules defines the exact six-column worksheet emitted
// for the Rouzao batch-order workbook. columnOrder is authoritative; SheetName
// is consumed by the xlsx renderer.
const SupplierOrderDemoMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "sheetName": "工作表1",
  "columns": {
    "export.third_party_order_no": "第三方订单号",
    "export.recipient_name": "收件人",
    "export.phone": "联系电话",
    "export.address": "收件地址",
    "export.factory_sku": "商家编码",
    "export.quantity": "下单数量"
  },
  "columnOrder": [
    "export.third_party_order_no",
    "export.recipient_name",
    "export.phone",
    "export.address",
    "export.factory_sku",
    "export.quantity"
  ],
  "transforms": {
    "export.recipient_name": ["trim"],
    "export.phone": ["trim"],
    "export.address": ["trim"],
    "export.factory_sku": ["trim"]
  },
  "required": [
    "export.third_party_order_no",
    "export.factory_sku",
    "export.quantity"
  ]
}`

// CatalogDemoMappingRules is the v2 MappingRules JSON for the rouzao product-list
// sample (SampleData/工厂平台——柔造). Chinese header and directory names are
// configuration data only — import code must not special-case platform==rouzao.
//
// Headers (SampleData CSV): 商品ID, 商品名称, 商家编码, 规格名称
// Image dirs in the sample zip: 主图/, 详情图/
const CatalogDemoMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "columns": {
    "product.factory_sku": "商家编码",
    "product.name": "商品名称",
    "product.supplier_product_ref": "商品ID"
  },
  "imageLayout": {
    "enabled": true,
    "coverDir": "主图",
    "detailDir": "详情图",
    "matchField": "product.name",
    "namePattern": "{match}#{nn}",
    "tabularGlob": "*.csv"
  }
}`

// ShipmentDemoMappingRules is the v2 MappingRules JSON for the rouzao express-
// order sample (SampleData/工厂平台——柔造/从工厂平台导出-快递订单数据.csv).
//
// 13 headers (synthetic seed uses the same names):
//
//	订单编号,下单时间,商品编码,商品名称,规格&数量,应付金额,收件人,电话,
//	收件信息,物流公司,物流单号,打印快递时间,订单状态
//
// Multi-SKU cells live in 规格&数量 → shipment.sku_quantity (pipe tokens;
// see ParseSKUQuantityTokens). 物流单号 uses strip_leading_quote for the
// leading apostrophe Excel emits on long numerics.
const ShipmentDemoMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "columns": {
    "shipment.external_shipment_no": "订单编号",
    "shipment.sku_quantity": "规格&数量",
    "shipment.recipient_name": "收件人",
    "shipment.phone": "电话",
    "shipment.carrier_name": "物流公司",
    "shipment.tracking_no": "物流单号",
    "shipment.shipped_at": "打印快递时间"
  },
  "transforms": {
    "shipment.tracking_no": ["trim", "strip_leading_quote"],
    "shipment.external_shipment_no": ["trim"],
    "shipment.sku_quantity": ["trim"],
    "shipment.phone": ["trim"],
    "shipment.recipient_name": ["trim"],
    "shipment.shipped_at": ["trim"]
  }
}`

// SeedCatalogDemo ensures a factory-side demo profile, an import_product_catalog
// DocumentTemplate (zip-friendly MappingRules with imageLayout), an
// import_supplier_shipment DocumentTemplate (rouzao 13-column express return),
// an export_supplier_order DocumentTemplate (exact six-column xlsx contract),
// and default IntegrationProfileTemplateBindings for all three document types.
//
// Idempotent:
//   - profile: skip create when ProfileKey already exists
//   - template: skip create when TemplateKey already exists
//   - binding: skip create when a default binding already exists for
//     (profileID, document_type)
//
// templateRepo / bindingRepo may be nil only when the caller does not need
// template+binding seeding (no-op for those parts). profileRepo is required.
func SeedCatalogDemo(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
) (*domain.IntegrationProfile, error) {
	if profileRepo == nil {
		return nil, fmt.Errorf("seed catalog demo: profile repository is required")
	}

	profile, err := ensureCatalogDemoProfile(ctx, profileRepo)
	if err != nil {
		return nil, err
	}

	if templateRepo == nil || bindingRepo == nil {
		return profile, nil
	}

	tmpl, err := ensureCatalogDemoTemplate(ctx, templateRepo)
	if err != nil {
		return nil, err
	}

	if err := ensureCatalogDemoBinding(ctx, bindingRepo, profile.ID, tmpl.ID); err != nil {
		return nil, err
	}

	shipTmpl, err := ensureShipmentDemoTemplate(ctx, templateRepo)
	if err != nil {
		return nil, err
	}

	if err := ensureShipmentDemoBinding(ctx, bindingRepo, profile.ID, shipTmpl.ID); err != nil {
		return nil, err
	}

	orderTmpl, err := ensureSupplierOrderDemoTemplate(ctx, templateRepo)
	if err != nil {
		return nil, err
	}

	if err := ensureSupplierOrderDemoBinding(ctx, bindingRepo, profile.ID, orderTmpl.ID); err != nil {
		return nil, err
	}

	return profile, nil
}

func ensureCatalogDemoProfile(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
) (*domain.IntegrationProfile, error) {
	existing, err := profileRepo.FindByProfileKey(ctx, CatalogDemoProfileKey)
	if err == nil && existing != nil {
		if !existing.SupportsExportSupplierOrder {
			existing.SupportsExportSupplierOrder = true
			if err := profileRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("seed catalog demo profile %q capability update: %w", CatalogDemoProfileKey, err)
			}
		}
		return existing, nil
	}

	// Route through the same validation path as CreateProfile — no repo.Create
	// backdoor that could seed an invalid factory profile.
	input := dto.CreateProfileInput{
		ProfileKey:                     CatalogDemoProfileKey,
		SourceChannel:                  "rouzao",
		SourceSurface:                  string(domain.SourceSurfaceFactory),
		FactorySupplierPlatform:        "rouzao",
		SupportsImportProductCatalog:   true,
		SupportsImportSupplierShipment: true,
		SupportsExportSupplierOrder:    true,
		// Factory demo does not channel-sync; keep readiness green without a connector.
		TrackingSyncMode: "unsupported",
	}
	if err := validateProfileEnums(input); err != nil {
		return nil, fmt.Errorf("seed catalog demo profile %q: %w", CatalogDemoProfileKey, err)
	}
	if err := validateExecutionReadiness(input, nil); err != nil {
		return nil, fmt.Errorf("seed catalog demo profile %q: %w", CatalogDemoProfileKey, err)
	}

	profile := profileFromCreateInput(input)
	if err := profileRepo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("seed catalog demo profile %q: %w", CatalogDemoProfileKey, err)
	}
	return profile, nil
}

func ensureCatalogDemoTemplate(
	ctx context.Context,
	templateRepo domain.DocumentTemplateRepository,
) (*domain.DocumentTemplate, error) {
	existing, err := templateRepo.FindByKey(ctx, CatalogDemoTemplateKey)
	if err != nil {
		return nil, fmt.Errorf("seed catalog demo template lookup %q: %w", CatalogDemoTemplateKey, err)
	}
	if existing != nil {
		return existing, nil
	}

	// Validate MappingRules at seed time so a bad constant fails loudly in tests
	// and on first seed rather than at first import.
	if _, err := ParseMappingRules(CatalogDemoMappingRules); err != nil {
		return nil, fmt.Errorf("seed catalog demo mapping rules invalid: %w", err)
	}

	tmpl := &domain.DocumentTemplate{
		TemplateKey:  CatalogDemoTemplateKey,
		DocumentType: CatalogDemoDocType,
		// zip: sample catalog ships as archive (tabular CSV + 主图/详情图).
		Format:       "zip",
		MappingRules: CatalogDemoMappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("seed catalog demo template %q: %w", CatalogDemoTemplateKey, err)
	}
	return tmpl, nil
}

func ensureCatalogDemoBinding(
	ctx context.Context,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID, templateID uint,
) error {
	return ensureDemoBinding(ctx, bindingRepo, profileID, templateID, CatalogDemoDocType, "catalog")
}

func ensureShipmentDemoTemplate(
	ctx context.Context,
	templateRepo domain.DocumentTemplateRepository,
) (*domain.DocumentTemplate, error) {
	existing, err := templateRepo.FindByKey(ctx, ShipmentDemoTemplateKey)
	if err != nil {
		return nil, fmt.Errorf("seed shipment demo template lookup %q: %w", ShipmentDemoTemplateKey, err)
	}
	if existing != nil {
		return existing, nil
	}

	rules, err := ParseMappingRules(ShipmentDemoMappingRules)
	if err != nil {
		return nil, fmt.Errorf("seed shipment demo mapping rules invalid: %w", err)
	}
	// Config-time dest check so an illegal dest fails at seed, not first import.
	if err := ValidateMappingRulesConfig(ShipmentDemoDocType, rules); err != nil {
		return nil, fmt.Errorf("seed shipment demo mapping rules illegal: %w", err)
	}

	tmpl := &domain.DocumentTemplate{
		TemplateKey:  ShipmentDemoTemplateKey,
		DocumentType: ShipmentDemoDocType,
		Format:       "csv",
		MappingRules: ShipmentDemoMappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("seed shipment demo template %q: %w", ShipmentDemoTemplateKey, err)
	}
	return tmpl, nil
}

func ensureShipmentDemoBinding(
	ctx context.Context,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID, templateID uint,
) error {
	return ensureDemoBinding(ctx, bindingRepo, profileID, templateID, ShipmentDemoDocType, "shipment")
}

func ensureSupplierOrderDemoTemplate(
	ctx context.Context,
	templateRepo domain.DocumentTemplateRepository,
) (*domain.DocumentTemplate, error) {
	existing, err := templateRepo.FindByKey(ctx, SupplierOrderDemoTemplateKey)
	if err != nil {
		return nil, fmt.Errorf("seed supplier order demo template lookup %q: %w", SupplierOrderDemoTemplateKey, err)
	}
	if existing != nil {
		return existing, nil
	}

	rules, err := ParseMappingRules(SupplierOrderDemoMappingRules)
	if err != nil {
		return nil, fmt.Errorf("seed supplier order demo mapping rules invalid: %w", err)
	}
	if err := ValidateMappingRulesConfig(SupplierOrderDemoDocType, rules); err != nil {
		return nil, fmt.Errorf("seed supplier order demo mapping rules illegal: %w", err)
	}

	tmpl := &domain.DocumentTemplate{
		TemplateKey:  SupplierOrderDemoTemplateKey,
		DocumentType: SupplierOrderDemoDocType,
		Format:       "xlsx",
		MappingRules: SupplierOrderDemoMappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("seed supplier order demo template %q: %w", SupplierOrderDemoTemplateKey, err)
	}
	return tmpl, nil
}

func ensureSupplierOrderDemoBinding(
	ctx context.Context,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID, templateID uint,
) error {
	return ensureDemoBinding(ctx, bindingRepo, profileID, templateID, SupplierOrderDemoDocType, "supplier order")
}

func ensureDemoBinding(
	ctx context.Context,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileID, templateID uint,
	docType, label string,
) error {
	existing, err := bindingRepo.FindDefaultByProfileAndType(ctx, profileID, docType)
	if err != nil {
		// Some test stubs return an error for "not found"; treat that as absent.
		// Real infra returns (nil, nil) when no default binding exists.
		existing = nil
	}
	if existing != nil {
		return nil
	}

	b := &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profileID,
		DocumentType:         docType,
		TemplateID:           templateID,
		IsDefault:            true,
	}
	if err := bindingRepo.Create(ctx, b); err != nil {
		return fmt.Errorf("seed %s demo binding profile=%d template=%d: %w", label, profileID, templateID, err)
	}
	return nil
}
