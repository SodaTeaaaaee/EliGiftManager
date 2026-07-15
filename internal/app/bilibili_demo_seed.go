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

	BilibiliExportTrackingTemplateKey = "bilibili_export_source_tracking_update"
	BilibiliExportTrackingDocType     = "export_source_tracking_update"

	BilibiliImportCarrierTemplateKey = "bilibili_import_carrier_mapping"
	BilibiliImportCarrierDocType     = "import_carrier_mapping"
)

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

// SeedBilibiliDemo ensures a bilibili membership demo profile plus default
// bindings for export_source_tracking_update (xlsx) and import_carrier_mapping.
//
// Idempotent:
//   - profile: skip create when ProfileKey already exists
//   - each template: skip create when TemplateKey already exists
//   - each binding: skip create when a default binding already exists for
//     (profileID, documentType)
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
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile.ID, exportTmpl.ID, BilibiliExportTrackingDocType); err != nil {
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
	if err := ensureBilibiliBinding(ctx, bindingRepo, profile.ID, carrierTmpl.ID, BilibiliImportCarrierDocType); err != nil {
		return nil, err
	}

	return profile, nil
}

func ensureBilibiliDemoProfile(
	ctx context.Context,
	profileRepo domain.IntegrationProfileRepository,
) (*domain.IntegrationProfile, error) {
	existing, err := profileRepo.FindByProfileKey(ctx, BilibiliDemoProfileKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Route through the same validation path as CreateProfile — no repo.Create
	// backdoor that could seed an invalid membership profile.
	input := dto.CreateProfileInput{
		ProfileKey:                BilibiliDemoProfileKey,
		SourceChannel:             "bilibili",
		SourceSurface:             string(domain.SourceSurfaceMembership),
		DemandKind:                "membership_entitlement",
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
		FactorySupplierPlatform:   "bilibili",
	}
	if err := validateProfileEnums(input); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", BilibiliDemoProfileKey, err)
	}
	// Seed path does not hold the runtime executor registry; nil provider only
	// enforces non-empty connector_key for document_export.
	if err := validateExecutionReadiness(input, nil); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", BilibiliDemoProfileKey, err)
	}

	profile := profileFromCreateInput(input)
	if err := profileRepo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("seed bilibili demo profile %q: %w", BilibiliDemoProfileKey, err)
	}
	return profile, nil
}

func ensureBilibiliTemplate(
	ctx context.Context,
	templateRepo domain.DocumentTemplateRepository,
	templateKey, docType, format, mappingRules string,
) (*domain.DocumentTemplate, error) {
	existing, err := templateRepo.FindByKey(ctx, templateKey)
	if err != nil {
		return nil, fmt.Errorf("seed bilibili demo template lookup %q: %w", templateKey, err)
	}
	if existing != nil {
		return existing, nil
	}

	rules, err := ParseMappingRules(mappingRules)
	if err != nil {
		return nil, fmt.Errorf("seed bilibili demo mapping rules invalid for %q: %w", templateKey, err)
	}
	if err := ValidateMappingRulesConfig(docType, rules); err != nil {
		return nil, fmt.Errorf("seed bilibili demo mapping dest validation failed for %q: %w", templateKey, err)
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
	profileID, templateID uint,
	docType string,
) error {
	existing, err := bindingRepo.FindDefaultByProfileAndType(ctx, profileID, docType)
	if err != nil {
		// Some test stubs return an error for "not found"; treat that as absent.
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
		return fmt.Errorf("seed bilibili demo binding profile=%d template=%d type=%s: %w", profileID, templateID, docType, err)
	}
	return nil
}
