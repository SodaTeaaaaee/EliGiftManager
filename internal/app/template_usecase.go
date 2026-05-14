package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

var validDocumentTypes = map[string]bool{
	"import_entitlement":          true,
	"import_sales_order":          true,
	"import_product_catalog":      true,
	"export_supplier_order":       true,
	"import_supplier_shipment":    true,
	"export_source_tracking_update": true,
}

var validDocumentFormats = map[string]bool{
	"csv":         true,
	"xlsx":        true,
	"json":        true,
	"api_payload": true,
}

type templateManagementUseCase struct {
	templateRepo domain.DocumentTemplateRepository
	bindingRepo  domain.ProfileTemplateBindingRepository
	profileRepo  domain.IntegrationProfileRepository
}

func NewTemplateManagementUseCase(
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileRepo domain.IntegrationProfileRepository,
) TemplateManagementUseCase {
	return &templateManagementUseCase{
		templateRepo: templateRepo,
		bindingRepo:  bindingRepo,
		profileRepo:  profileRepo,
	}
}

func (uc *templateManagementUseCase) CreateDocumentTemplate(input dto.CreateDocumentTemplateInput) (*dto.DocumentTemplateDTO, error) {
	if input.TemplateKey == "" {
		return nil, fmt.Errorf("templateKey must not be empty")
	}
	if !validDocumentTypes[input.DocumentType] {
		return nil, fmt.Errorf("invalid documentType: %q", input.DocumentType)
	}
	if !validDocumentFormats[input.Format] {
		return nil, fmt.Errorf("invalid format: %q", input.Format)
	}

	t := &domain.DocumentTemplate{
		TemplateKey:  input.TemplateKey,
		DocumentType: input.DocumentType,
		Format:       input.Format,
		MappingRules: input.MappingRules,
		ExtraData:    input.ExtraData,
	}
	if err := uc.templateRepo.Create(t); err != nil {
		return nil, err
	}
	return templateToDTO(t), nil
}

func (uc *templateManagementUseCase) ListDocumentTemplates() ([]dto.DocumentTemplateDTO, error) {
	templates, err := uc.templateRepo.List()
	if err != nil {
		return nil, err
	}
	out := make([]dto.DocumentTemplateDTO, len(templates))
	for i := range templates {
		out[i] = *templateToDTO(&templates[i])
	}
	return out, nil
}

func (uc *templateManagementUseCase) BindTemplateToProfile(input dto.BindTemplateToProfileInput) (*dto.ProfileTemplateBindingDTO, error) {
	// Validate document type
	if !validDocumentTypes[input.DocumentType] {
		return nil, fmt.Errorf("invalid documentType: %q", input.DocumentType)
	}

	// Validate template exists.
	t, err := uc.templateRepo.FindByID(input.TemplateID)
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

	// Validate profile exists.
	profile, err := uc.profileRepo.FindByID(input.IntegrationProfileID)
	if err != nil {
		return nil, fmt.Errorf("look up integration profile %d: %w", input.IntegrationProfileID, err)
	}
	if profile == nil {
		return nil, fmt.Errorf("integration profile %d not found", input.IntegrationProfileID)
	}

	// Enforce uniqueness: only one default binding per (profileID, documentType)
	if input.IsDefault {
		existing, err := uc.bindingRepo.FindDefaultByProfileAndType(input.IntegrationProfileID, input.DocumentType)
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
	if err := uc.bindingRepo.Create(b); err != nil {
		return nil, err
	}
	return bindingToDTO(b), nil
}

func (uc *templateManagementUseCase) ListBindingsByProfile(profileID uint) ([]dto.ProfileTemplateBindingDTO, error) {
	bindings, err := uc.bindingRepo.ListByProfile(profileID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ProfileTemplateBindingDTO, len(bindings))
	for i := range bindings {
		out[i] = *bindingToDTO(&bindings[i])
	}
	return out, nil
}

func (uc *templateManagementUseCase) GetDefaultTemplateForProfile(profileID uint, docType string) (*dto.DocumentTemplateDTO, error) {
	binding, err := uc.bindingRepo.FindDefaultByProfileAndType(profileID, docType)
	if err != nil {
		return nil, fmt.Errorf("find default binding for profile %d / type %q: %w", profileID, docType, err)
	}
	if binding == nil {
		return nil, nil
	}

	t, err := uc.templateRepo.FindByID(binding.TemplateID)
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
