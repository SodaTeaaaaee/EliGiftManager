package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// Bilibili demo seed keys — idempotent by ProfileKey / TemplateKey.
// Mapping header literals are locked to SampleData/需求平台——哔哩哔哩
// (no real PII is copied into the seed).
const (
	BilibiliDemoProfileKey = "bilibili_membership_demo"
	// BilibiliRetailDemoProfileKey is no longer seeded as a standalone profile.
	// Kept only so existing callers of the old symbol still compile.
	BilibiliRetailDemoProfileKey = "bilibili_retail_demo"

	BilibiliImportEntitlementTemplateKey = "bilibili_import_entitlement"
	BilibiliImportEntitlementDocType     = "import_entitlement"

	BilibiliImportSalesOrderTemplateKey = "bilibili_import_sales_order"
	BilibiliImportSalesOrderDocType     = "import_sales_order"

	BilibiliExportTrackingTemplateKey = "bilibili_export_source_tracking_update"
	BilibiliExportTrackingDocType     = "export_source_tracking_update"

	BilibiliImportCarrierTemplateKey = "bilibili_import_carrier_mapping"
	BilibiliImportCarrierDocType     = "import_carrier_mapping"
)

// BilibiliImportEntitlementMappingRules maps the headerless three-column
// membership export (level / UID / display name). Each sample file is treated
// independently; these rules do not derive any value from another sample.
const BilibiliImportEntitlementMappingRules = `{
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
    "line.requested_quantity": "1",
    "line.obligation_trigger_kind": "periodic_membership",
    "line.entitlement_authority": "upstream_platform",
    "line.recipient_input_state": "not_required",
    "line.routing_disposition": "accepted"
  },
  "required": [
    "line.gift_level_snapshot",
    "document.source_customer_ref"
  ]
}`

// BilibiliImportSalesOrderMappingRules maps one Bilibili retail-order export.
// The buyer nickname is provenance and a display-name observation only. It is
// never treated as a stable external buyer identity.
const BilibiliImportSalesOrderMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "columns": {
    "line.external_title": "商品名称",
    "line.requested_quantity": "数量",
    "recipient.name": "收货人姓名",
    "recipient.phone": "联系电话",
    "recipient.address_line1": "收货地址",
    "document.source_document_no": "订单号",
    "document.source_customer_ref": "买家昵称",
    "document.display_name": "买家昵称"
  },
  "defaults": {
    "line.line_type": "sku_order",
    "line.entitlement_authority": "upstream_platform",
    "line.recipient_input_state": "ready",
    "line.routing_disposition": "accepted"
  },
  "required": [
    "document.source_document_no"
  ]
}`

// BilibiliExportTrackingMappingRules is the v2 MappingRules JSON for bilibili
// source-tracking re-import (SampleData 需要导入需求平台-订单快递跟踪).
// Headers are locked verbatim (including the * markers and parenthetical note).
const BilibiliExportTrackingMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "columns": {
    "export.external_document_no": "订单号*",
    "export.carrier_code": "快递公司编码*（请在网页中查看快递编码）",
    "export.tracking_no": "物流单号*"
  },
  "columnOrder": [
    "export.external_document_no",
    "export.carrier_code",
    "export.tracking_no"
  ],
  "required": [
    "export.external_document_no",
    "export.carrier_code",
    "export.tracking_no"
  ]
}`

// BilibiliImportCarrierMappingRules is the v2 MappingRules JSON for bilibili
// carrier-code mapping import (SampleData 从需求平台导出-快递编码映射关系).
// Platform sheets expose external code + name only; internal is backfilled via
// ResolveByExternalOrAlias on import when already known.
const BilibiliImportCarrierMappingRules = `{
  "version": 2,
  "mode": "header",
  "hasHeader": true,
  "columns": {
    "carrier.external_carrier_code": "快递公司编码",
    "carrier.external_carrier_name": "快递公司名称"
  }
}`

// SeedBilibiliDemo ensures one Bilibili demand profile (bilibili_membership_demo)
// plus four document templates and four default bindings on that same profile:
// import_entitlement, import_sales_order, export_source_tracking_update, and
// import_carrier_mapping.
//
// Idempotent (same pattern as SeedCatalogDemo):
//   - profile: skip create when ProfileKey already exists; leftover DemandKind
//     is cleared to empty (documentType is explicit). Do not overwrite
//     operator mappings or capabilities
//   - template: skip create when TemplateKey already exists; do not clobber
//     MappingRules. Creating missing templates is OK
//   - binding: skip create when a default binding already exists for
//     (profileID, document_type). Missing defaults are created only when
//     ValidateProfileDocumentType allows the document type on the profile
//
// templateRepo / bindingRepo may be nil only when the caller does not need
// template+binding seeding (no-op for those parts). profileRepo is required.
func SeedBilibiliDemo(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
) (*domain.IntegrationProfile, error) {
	if profileRepo == nil {
		return nil, fmt.Errorf("seed bilibili demo: profile repository is required")
	}

	profile, err := ensureBilibiliDemoProfile(ctx, profileRepo)
	if err != nil {
		return nil, err
	}

	if templateRepo == nil || bindingRepo == nil {
		return profile, nil
	}

	entitlementTmpl, err := ensureBilibiliTemplate(
		ctx,
		templateRepo,
		BilibiliImportEntitlementTemplateKey,
		BilibiliImportEntitlementDocType,
		"csv",
		BilibiliImportEntitlementMappingRules,
	)
	if err != nil {
		return nil, err
	}
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile, entitlementTmpl.ID, BilibiliImportEntitlementDocType); err != nil {
		return nil, err
	}

	salesOrderTmpl, err := ensureBilibiliTemplate(
		ctx,
		templateRepo,
		BilibiliImportSalesOrderTemplateKey,
		BilibiliImportSalesOrderDocType,
		"xls",
		BilibiliImportSalesOrderMappingRules,
	)
	if err != nil {
		return nil, err
	}
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile, salesOrderTmpl.ID, BilibiliImportSalesOrderDocType); err != nil {
		return nil, err
	}

	exportTmpl, err := ensureBilibiliTemplate(
		ctx,
		templateRepo,
		BilibiliExportTrackingTemplateKey,
		BilibiliExportTrackingDocType,
		"xlsx",
		BilibiliExportTrackingMappingRules,
	)
	if err != nil {
		return nil, err
	}
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile, exportTmpl.ID, BilibiliExportTrackingDocType); err != nil {
		return nil, err
	}

	carrierTmpl, err := ensureBilibiliTemplate(
		ctx,
		templateRepo,
		BilibiliImportCarrierTemplateKey,
		BilibiliImportCarrierDocType,
		"xls",
		BilibiliImportCarrierMappingRules,
	)
	if err != nil {
		return nil, err
	}
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile, carrierTmpl.ID, BilibiliImportCarrierDocType); err != nil {
		return nil, err
	}

	return profile, nil
}

func ensureBilibiliDemoProfile(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
) (*domain.IntegrationProfile, error) {
	return ensureBilibiliProfile(ctx, profileRepo, dto.CreateProfileInput{
		ProfileKey:    BilibiliDemoProfileKey,
		SourceChannel: "bilibili",
		SourceSurface: string(domain.SourceSurfaceMembership),
		// DemandKind is leftover and is not the unique document type. Empty so
		// both import_entitlement and import_sales_order can bind; importers
		// must not infer documentType from this field.
		DemandKind:                "",
		InitialAllocationStrategy: "policy_driven",
		IdentityStrategy:          "platform_uid",
		EntitlementAuthorityMode:  "upstream_platform",
		RecipientInputMode:        "none",
		ReferenceStrategy:         "member_level",
		TrackingSyncMode:          "document_export",
		ClosurePolicy:             "close_after_sync",
		SupportsPartialShipment:   true,
		RequiresCarrierMapping:    true,
		RequiresExternalOrderNo:   false,
		AllowsManualClosure:       false,
		ConnectorKey:              "eli.local_export",
	})
}

func ensureBilibiliProfile(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
	input dto.CreateProfileInput,
) (*domain.IntegrationProfile, error) {
	existing, findErr := profileRepo.FindByProfileKey(ctx, input.ProfileKey)
	if findErr == nil && existing != nil {
		// DemandKind is leftover and is not the unique document type. Empty so
		// both import_entitlement and import_sales_order can bind. Do not touch
		// operator capabilities or template MappingRules.
		if existing.DemandKind != "" {
			existing.DemandKind = ""
			if err := profileRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("seed bilibili demo profile %q leftover DemandKind: %w", input.ProfileKey, err)
			}
		}
		return existing, nil
	}

	// Route through the same validation path as CreateProfile — no repo.Create
	// backdoor that could seed an invalid demand profile.
	if err := validateProfileEnums(input); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", input.ProfileKey, err)
	}
	// Seed path does not hold the runtime executor registry; nil provider only
	// enforces non-empty connector_key for document_export.
	if err := validateExecutionReadiness(input, nil); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", input.ProfileKey, err)
	}

	desired := profileFromCreateInput(input)
	if err := profileRepo.Create(ctx, desired); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", input.ProfileKey, err)
	}
	return desired, nil
}

func ensureBilibiliTemplate(
	ctx context.Context,
	templateRepo domain.DocumentTemplateRepository,
	templateKey, docType, format, mappingRules string,
) (*domain.DocumentTemplate, error) {
	rules, err := ParseMappingRules(mappingRules)
	if err != nil {
		return nil, fmt.Errorf("seed bilibili demo mapping rules invalid for %q: %w", templateKey, err)
	}
	if err := ValidateMappingRulesConfig(docType, rules); err != nil {
		return nil, fmt.Errorf("seed bilibili demo mapping dest validation failed for %q: %w", templateKey, err)
	}

	existing, err := templateRepo.FindByKey(ctx, templateKey)
	if err != nil {
		return nil, fmt.Errorf("seed bilibili demo template lookup %q: %w", templateKey, err)
	}
	if existing != nil {
		return existing, nil
	}

	tmpl := &domain.DocumentTemplate{
		TemplateKey:  templateKey,
		DocumentType: docType,
		Format:       format,
		MappingRules: mappingRules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("seed bilibili demo template %q: %w", templateKey, err)
	}
	return tmpl, nil
}

func ensureBilibiliBinding(
	ctx context.Context,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profile *domain.IntegrationProfile,
	templateID uint,
	docType string,
) error {
	return ensureDemoBinding(ctx, bindingRepo, profile, templateID, docType, "bilibili")
}
