package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

var validDocumentTypes = map[string]bool{
	"import_entitlement":            true,
	"import_sales_order":            true,
	"import_product_catalog":        true,
	"export_supplier_order":         true,
	"import_supplier_shipment":      true,
	"export_source_tracking_update": true,
	"import_carrier_mapping":        true,
}

var validDocumentFormats = map[string]bool{
	"csv":         true,
	"xlsx":        true,
	"xls":         true,
	"json":        true,
	"api_payload": true,
	"zip":         true, // product catalog archives (tabular sheet + images)
}

func validateDocumentFormat(docType, format string) error {
	if !validDocumentFormats[format] {
		return fmt.Errorf("invalid format: %q", format)
	}
	if format == "xls" && (docType == "export_supplier_order" || docType == "export_source_tracking_update") {
		return fmt.Errorf("format %q is read-only for imports; %s output supports xlsx (recommended) or csv, not BIFF .xls", format, docType)
	}
	return nil
}

// IsFactoryProfile reports whether the profile is a factory (supplier) surface.
func IsFactoryProfile(profile *domain.IntegrationProfile) bool {
	return profile != nil && profile.SourceSurface == string(domain.SourceSurfaceFactory)
}

// IsSourcePlatformProfile reports whether the profile is a source platform.
// membership and retail leftovers both count; they must not force a second row.
func IsSourcePlatformProfile(profile *domain.IntegrationProfile) bool {
	if profile == nil {
		return false
	}
	return profile.SourceSurface == string(domain.SourceSurfaceMembership) ||
		profile.SourceSurface == string(domain.SourceSurfaceRetail)
}

// DemandImportInterpretation is the document-type-owned reading of a demand
// import. Importers (controller / CSV, sibling lane) MUST use these values for
// DemandDocument.Kind, DemandDocument.SourceSurface, and customer identity.
// They must NOT treat IntegrationProfile.DemandKind or IdentityStrategy as the
// unique source of truth. Identity is not stored per-platform.
type DemandImportInterpretation struct {
	DocumentType     string
	DemandKind       string // DemandDocument.Kind
	SourceSurface    string // DemandDocument.SourceSurface
	IdentityStrategy string
}

// InterpretDemandImportDocumentType maps a demand-import document type onto
// DemandDocument kind/surface and the import-time identity strategy. It does
// not execute identity resolution.
func InterpretDemandImportDocumentType(docType string) (DemandImportInterpretation, error) {
	switch strings.TrimSpace(docType) {
	case "import_entitlement":
		return DemandImportInterpretation{
			DocumentType:     "import_entitlement",
			DemandKind:       string(domain.DemandKindMembershipEntitlement),
			SourceSurface:    string(domain.SourceSurfaceMembership),
			IdentityStrategy: "platform_uid",
		}, nil
	case "import_sales_order":
		return DemandImportInterpretation{
			DocumentType:     "import_sales_order",
			DemandKind:       string(domain.DemandKindRetailOrder),
			SourceSurface:    string(domain.SourceSurfaceRetail),
			IdentityStrategy: IdentityStrategyOrderScopedProvisional,
		}, nil
	default:
		return DemandImportInterpretation{}, fmt.Errorf("documentType %q is not a demand import type", strings.TrimSpace(docType))
	}
}

// ResolveDemandImportDocumentType verifies an explicit demand-import
// documentType against the profile. Callers must pass documentType; it is not
// inferred from IntegrationProfile.DemandKind.
func ResolveDemandImportDocumentType(profile *domain.IntegrationProfile, requested string) (string, error) {
	if profile == nil {
		return "", fmt.Errorf("integration profile is required")
	}

	docType := strings.TrimSpace(requested)
	if docType == "" {
		return "", fmt.Errorf("explicit documentType is required")
	}
	if docType != "import_entitlement" && docType != "import_sales_order" {
		return "", fmt.Errorf("documentType %q is not a demand import type", docType)
	}
	if err := ValidateProfileDocumentType(profile, docType); err != nil {
		return "", err
	}
	return docType, nil
}

// ValidateProfileDocumentType enforces legal document-template ownership.
// Demand imports are allowed on any source platform (not factory) regardless
// of leftover DemandKind. Factory document types require the matching
// capability; operational source-platform documents require their settings.
func ValidateProfileDocumentType(profile *domain.IntegrationProfile, docType string) error {
	if profile == nil {
		return fmt.Errorf("integration profile is required")
	}

	docType = strings.TrimSpace(docType)
	capability := ""
	allowed := false
	switch docType {
	case "import_entitlement", "import_sales_order":
		allowed = IsSourcePlatformProfile(profile)
		capability = "source_platform"
	case "import_product_catalog":
		allowed = IsFactoryProfile(profile) && profile.SupportsImportProductCatalog
		capability = "supportsImportProductCatalog"
	case "export_supplier_order":
		allowed = IsFactoryProfile(profile) && profile.SupportsExportSupplierOrder
		capability = "supportsExportSupplierOrder"
	case "import_supplier_shipment":
		allowed = IsFactoryProfile(profile) && profile.SupportsImportSupplierShipment
		capability = "supportsImportSupplierShipment"
	case "export_source_tracking_update":
		allowed = IsSourcePlatformProfile(profile) && profile.TrackingSyncMode == "document_export"
		capability = "trackingSyncMode=document_export"
	case "import_carrier_mapping":
		allowed = IsSourcePlatformProfile(profile) && profile.RequiresCarrierMapping
		capability = "requiresCarrierMapping"
	default:
		return fmt.Errorf("invalid documentType: %q", docType)
	}
	if !allowed {
		return fmt.Errorf(
			"documentType %q is not supported by profile %d (surface=%q capability=%q)",
			docType,
			profile.ID,
			profile.SourceSurface,
			capability,
		)
	}
	return nil
}

type templateManagementUseCase struct {
	templateRepo domain.DocumentTemplateRepository
	bindingRepo  domain.ProfileTemplateBindingRepository
	profileRepo  domain.IntegrationProfileRepository
	db           *gorm.DB
}

func NewTemplateManagementUseCase(
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileRepo domain.IntegrationProfileRepository,
	extra ...any,
) TemplateManagementUseCase {
	uc := &templateManagementUseCase{
		templateRepo: templateRepo,
		bindingRepo:  bindingRepo,
		profileRepo:  profileRepo,
	}
	for _, dep := range extra {
		if db, ok := dep.(*gorm.DB); ok && db != nil {
			uc.db = db
		}
	}
	return uc
}

type bindingDefaultMutator interface {
	ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error
	Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error
}

type gormBindingWriter struct {
	db *gorm.DB
}

func (w *gormBindingWriter) ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error {
	return w.db.WithContext(ctx).
		Model(&persistence.IntegrationProfileTemplateBinding{}).
		Where("integration_profile_id = ? AND document_type = ? AND is_default = ?", profileID, docType, true).
		Update("is_default", false).Error
}

func (w *gormBindingWriter) Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	p := persistence.ProfileTemplateBindingFromDomain(b)
	p.ID = b.ID
	if err := w.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*b = *persistence.ProfileTemplateBindingToDomain(p)
	return nil
}

func (uc *templateManagementUseCase) CreateDocumentTemplate(ctx context.Context, input dto.CreateDocumentTemplateInput) (*dto.DocumentTemplateDTO, error) {
	if input.TemplateKey == "" {
		return nil, fmt.Errorf("templateKey must not be empty")
	}
	if !validDocumentTypes[input.DocumentType] {
		return nil, fmt.Errorf("invalid documentType: %q", input.DocumentType)
	}
	if err := validateDocumentFormat(input.DocumentType, input.Format); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.MappingRules) != "" {
		rules, err := ParseMappingRules(input.MappingRules)
		if err != nil {
			return nil, fmt.Errorf("mappingRules: %w", err)
		}
		if err := ValidateMappingRulesConfig(input.DocumentType, rules); err != nil {
			return nil, fmt.Errorf("mappingRules: %w", err)
		}
	}

	t := &domain.DocumentTemplate{
		TemplateKey:  input.TemplateKey,
		DocumentType: input.DocumentType,
		Format:       input.Format,
		MappingRules: input.MappingRules,
		ExtraData:    input.ExtraData,
	}
	if err := uc.templateRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	return templateToDTO(t), nil
}

func (uc *templateManagementUseCase) ListDocumentTemplates(ctx context.Context) ([]dto.DocumentTemplateDTO, error) {
	templates, err := uc.templateRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.DocumentTemplateDTO, len(templates))
	for i := range templates {
		out[i] = *templateToDTO(&templates[i])
	}
	return out, nil
}

func (uc *templateManagementUseCase) UpdateDocumentTemplate(ctx context.Context, input dto.UpdateDocumentTemplateInput) (*dto.DocumentTemplateDTO, error) {
	if input.ID == 0 {
		return nil, fmt.Errorf("id must not be empty")
	}
	existing, err := uc.templateRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("look up template %d: %w", input.ID, err)
	}
	if existing == nil {
		return nil, fmt.Errorf("template %d not found", input.ID)
	}
	if err := validateDocumentFormat(existing.DocumentType, input.Format); err != nil {
		return nil, err
	}

	// TemplateKey and DocumentType are immutable — only Format/MappingRules/ExtraData may change.
	if strings.TrimSpace(input.MappingRules) != "" {
		rules, err := ParseMappingRules(input.MappingRules)
		if err != nil {
			return nil, fmt.Errorf("mappingRules: %w", err)
		}
		if err := ValidateMappingRulesConfig(existing.DocumentType, rules); err != nil {
			return nil, fmt.Errorf("mappingRules: %w", err)
		}
	}

	existing.Format = input.Format
	existing.MappingRules = input.MappingRules
	existing.ExtraData = input.ExtraData

	if err := uc.templateRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return templateToDTO(existing), nil
}

func (uc *templateManagementUseCase) DeleteDocumentTemplate(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("id must not be empty")
	}
	existing, err := uc.templateRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("look up template %d: %w", id, err)
	}
	if existing == nil {
		return fmt.Errorf("template %d not found", id)
	}

	// Refuse deletion while any profile binding still references this template.
	refs, err := uc.bindingRepo.ListByTemplateID(ctx, id)
	if err != nil {
		return fmt.Errorf("check bindings for template %d: %w", id, err)
	}
	if len(refs) > 0 {
		parts := make([]string, 0, len(refs))
		for _, b := range refs {
			parts = append(parts, fmt.Sprintf("bindingID=%d profileID=%d docType=%q", b.ID, b.IntegrationProfileID, b.DocumentType))
		}
		return fmt.Errorf("cannot delete template %d (%s): still referenced by %d binding(s): %s",
			id, existing.TemplateKey, len(refs), strings.Join(parts, "; "))
	}

	return uc.templateRepo.Delete(ctx, id)
}

func (uc *templateManagementUseCase) BindTemplateToProfile(ctx context.Context, input dto.BindTemplateToProfileInput) (*dto.ProfileTemplateBindingDTO, error) {
	// Validate document type
	if !validDocumentTypes[input.DocumentType] {
		return nil, fmt.Errorf("invalid documentType: %q", input.DocumentType)
	}

	// Validate template exists.
	t, err := uc.templateRepo.FindByID(ctx, input.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("look up template %d: %w", input.TemplateID, err)
	}
	if t == nil {
		return nil, fmt.Errorf("template %d not found", input.TemplateID)
	}

	// Validate template documentType matches binding documentType
	if t.DocumentType != input.DocumentType {
		return nil, fmt.Errorf("template %d has documentType %q, cannot bind as %q", input.TemplateID, t.DocumentType, input.DocumentType)
	}
	if err := validateDocumentFormat(t.DocumentType, t.Format); err != nil {
		return nil, fmt.Errorf("template %d: %w", t.ID, err)
	}

	// Validate profile exists.
	profile, err := uc.profileRepo.FindByID(ctx, input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("look up integration profile %d: %w", input.IntegrationProfileID, err)
	}
	if profile == nil {
		return nil, fmt.Errorf("integration profile %d not found", input.IntegrationProfileID)
	}
	if err := ValidateProfileDocumentType(profile, input.DocumentType); err != nil {
		return nil, err
	}

	// Enforce uniqueness: only one default binding per (profileID, documentType)
	if input.IsDefault {
		existing, err := uc.bindingRepo.FindDefaultByProfileAndType(ctx, input.IntegrationProfileID, input.DocumentType)
		if err != nil {
			return nil, fmt.Errorf("check existing default: %w", err)
		}
		if existing != nil {
			return nil, fmt.Errorf("default binding already exists for profile %d / type %q (binding ID %d)", input.IntegrationProfileID, input.DocumentType, existing.ID)
		}
	}

	b := &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: input.IntegrationProfileID,
		DocumentType:         input.DocumentType,
		TemplateID:           input.TemplateID,
		IsDefault:            input.IsDefault,
	}
	if err := uc.bindingRepo.Create(ctx, b); err != nil {
		return nil, err
	}
	return bindingToDTO(b), nil
}

func (uc *templateManagementUseCase) ListBindingsByProfile(ctx context.Context, profileID uint) ([]dto.ProfileTemplateBindingDTO, error) {
	bindings, err := uc.bindingRepo.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ProfileTemplateBindingDTO, len(bindings))
	for i := range bindings {
		out[i] = *bindingToDTO(&bindings[i])
	}
	return out, nil
}

func (uc *templateManagementUseCase) UnbindTemplate(ctx context.Context, bindingID uint) error {
	if bindingID == 0 {
		return fmt.Errorf("bindingID must not be empty")
	}
	existing, err := uc.bindingRepo.FindByID(ctx, bindingID)
	if err != nil {
		return fmt.Errorf("look up binding %d: %w", bindingID, err)
	}
	if existing == nil {
		return fmt.Errorf("binding %d not found", bindingID)
	}
	return uc.bindingRepo.Delete(ctx, bindingID)
}

func (uc *templateManagementUseCase) SetDefaultBinding(ctx context.Context, bindingID uint) error {
	if bindingID == 0 {
		return fmt.Errorf("bindingID must not be empty")
	}
	binding, err := uc.bindingRepo.FindByID(ctx, bindingID)
	if err != nil {
		return fmt.Errorf("look up binding %d: %w", bindingID, err)
	}
	if binding == nil {
		return fmt.Errorf("binding %d not found", bindingID)
	}
	if binding.IsDefault {
		return nil
	}

	profile, err := uc.profileRepo.FindByID(ctx, binding.IntegrationProfileID)
	if err != nil {
		return fmt.Errorf("look up integration profile %d: %w", binding.IntegrationProfileID, err)
	}
	if profile == nil {
		return fmt.Errorf("integration profile %d not found", binding.IntegrationProfileID)
	}
	if err := ValidateProfileDocumentType(profile, binding.DocumentType); err != nil {
		return err
	}
	t, err := uc.templateRepo.FindByID(ctx, binding.TemplateID)
	if err != nil {
		return fmt.Errorf("look up template %d: %w", binding.TemplateID, err)
	}
	if t == nil {
		return fmt.Errorf("template %d not found", binding.TemplateID)
	}
	if t.DocumentType != binding.DocumentType {
		return fmt.Errorf("template %d has documentType %q, cannot set binding as %q", binding.TemplateID, t.DocumentType, binding.DocumentType)
	}
	if err := validateDocumentFormat(t.DocumentType, t.Format); err != nil {
		return fmt.Errorf("template %d: %w", t.ID, err)
	}

	// Clear any existing default for the same (profile, documentType), then promote this row.
	promote := func(repo bindingDefaultMutator) error {
		if err := repo.ClearDefaultByProfileAndType(ctx, binding.IntegrationProfileID, binding.DocumentType); err != nil {
			return fmt.Errorf("clear existing default for profile %d / type %q: %w", binding.IntegrationProfileID, binding.DocumentType, err)
		}
		binding.IsDefault = true
		if err := repo.Update(ctx, binding); err != nil {
			return fmt.Errorf("set binding %d as default: %w", bindingID, err)
		}
		return nil
	}
	if uc.db == nil {
		return promote(uc.bindingRepo)
	}
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return promote(&gormBindingWriter{db: tx})
	})
}

func (uc *templateManagementUseCase) GetDefaultTemplateForProfile(ctx context.Context, profileID uint, docType string) (*dto.DocumentTemplateDTO, error) {
	binding, err := uc.bindingRepo.FindDefaultByProfileAndType(ctx, profileID, docType)
	if err != nil {
		return nil, fmt.Errorf("find default binding for profile %d / type %q: %w", profileID, docType, err)
	}
	if binding == nil {
		return nil, nil
	}

	t, err := uc.templateRepo.FindByID(ctx, binding.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("look up template %d: %w", binding.TemplateID, err)
	}
	if t == nil {
		return nil, fmt.Errorf("template %d referenced by binding not found", binding.TemplateID)
	}
	return templateToDTO(t), nil
}

// ---- helpers ----

func templateToDTO(t *domain.DocumentTemplate) *dto.DocumentTemplateDTO {
	return &dto.DocumentTemplateDTO{
		ID:           t.ID,
		TemplateKey:  t.TemplateKey,
		DocumentType: t.DocumentType,
		Format:       t.Format,
		MappingRules: t.MappingRules,
		ExtraData:    t.ExtraData,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func bindingToDTO(b *domain.IntegrationProfileTemplateBinding) *dto.ProfileTemplateBindingDTO {
	return &dto.ProfileTemplateBindingDTO{
		ID:                   b.ID,
		IntegrationProfileID: b.IntegrationProfileID,
		DocumentType:         b.DocumentType,
		TemplateID:           b.TemplateID,
		IsDefault:            b.IsDefault,
		CreatedAt:            b.CreatedAt,
	}
}
