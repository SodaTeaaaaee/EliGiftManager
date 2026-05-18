package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TemplateMappingService applies a template's MappingRules to convert raw rows
// into structured DemandLines.
type TemplateMappingService struct {
	templateRepo    domain.DocumentTemplateRepository
	bindingRepo     domain.ProfileTemplateBindingRepository
	profileRepo     domain.IntegrationProfileRepository
}

func NewTemplateMappingService(
	templateRepo domain.DocumentTemplateRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	profileRepo domain.IntegrationProfileRepository,
) *TemplateMappingService {
	return &TemplateMappingService{
		templateRepo: templateRepo,
		bindingRepo:  bindingRepo,
		profileRepo:  profileRepo,
	}
}

// TemplateMappingRules defines the column mapping contract stored in DocumentTemplate.MappingRules.
type TemplateMappingRules struct {
	Columns  map[string]string `json:"columns"`
	Defaults map[string]string `json:"defaults"`
}

// ParseMappingRules unmarshals a template's MappingRules JSON.
func ParseMappingRules(raw string) (*TemplateMappingRules, error) {
	if raw == "" {
		return nil, fmt.Errorf("template has no mapping rules")
	}
	var rules TemplateMappingRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil, fmt.Errorf("invalid mapping rules: %w", err)
	}
	if len(rules.Columns) == 0 {
		return nil, fmt.Errorf("template mapping rules must define at least one column")
	}
	return &rules, nil
}

// ResolveImportTemplate finds the default template binding for a profile and document type.
func (s *TemplateMappingService) ResolveImportTemplate(profileID uint, documentType string) (*domain.DocumentTemplate, error) {
	binding, err := s.bindingRepo.FindDefaultByProfileAndType(profileID, documentType)
	if err != nil {
		return nil, fmt.Errorf("resolve template binding for profile %d / type %s: %w", profileID, documentType, err)
	}
	if binding == nil {
		return nil, fmt.Errorf("no default template binding for profile %d / type %s", profileID, documentType)
	}
	t, err := s.templateRepo.FindByID(binding.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template %d not found: %w", binding.TemplateID, err)
	}
	return t, nil
}

// MapCSVRowToDemandLine converts a single CSV row (column → value map) into a
// DemandLine using the template's MappingRules.
func MapCSVRowToDemandLine(row map[string]string, rules *TemplateMappingRules) (*domain.DemandLine, error) {
	line := &domain.DemandLine{}

	// Apply column mappings
	for destField, srcColumn := range rules.Columns {
		val, ok := row[srcColumn]
		if !ok {
			continue // optional column — skip
		}
		if err := setDemandLineField(line, destField, val); err != nil {
			return nil, fmt.Errorf("field %q (column %q): %w", destField, srcColumn, err)
		}
	}

	// Apply defaults
	for field, val := range rules.Defaults {
		if err := setDemandLineField(line, field, val); err != nil {
			return nil, fmt.Errorf("default %q: %w", field, err)
		}
	}

	return line, nil
}

// setDemandLineField sets a single field on a DemandLine by field name.
func setDemandLineField(line *domain.DemandLine, field, value string) error {
	switch field {
	case "line_type":
		line.LineType = value
	case "obligation_trigger_kind":
		line.ObligationTriggerKind = value
	case "entitlement_authority":
		line.EntitlementAuthority = value
	case "recipient_input_state":
		line.RecipientInputState = value
	case "routing_disposition":
		line.RoutingDisposition = value
	case "routing_reason_code":
		line.RoutingReasonCode = value
	case "eligibility_context_ref":
		line.EligibilityContextRef = value
	case "entitlement_code":
		line.EntitlementCode = value
	case "gift_level_snapshot":
		line.GiftLevelSnapshot = value
	case "recipient_input_payload":
		line.RecipientInputPayload = value
	case "external_title":
		line.ExternalTitle = value
	case "requested_quantity":
		qty, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid quantity %q", value)
		}
		line.RequestedQuantity = qty
	default:
		return fmt.Errorf("unknown field %q", field)
	}
	return nil
}

// BuildImportPipeline resolves the template for a profile + document type,
// then maps all CSV rows to DemandLines. Returns parsed lines and the resolved template.
func (s *TemplateMappingService) BuildImportPipeline(profileID uint, documentType string, rows []map[string]string) (*domain.DocumentTemplate, []*domain.DemandLine, error) {
	t, err := s.ResolveImportTemplate(profileID, documentType)
	if err != nil {
		return nil, nil, err
	}

	rules, err := ParseMappingRules(t.MappingRules)
	if err != nil {
		return nil, nil, fmt.Errorf("template %s: %w", t.TemplateKey, err)
	}

	now := time.Now().Format(time.RFC3339)
	lines := make([]*domain.DemandLine, 0, len(rows))
	for i, row := range rows {
		line, err := MapCSVRowToDemandLine(row, rules)
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		if line.CreatedAt == "" {
			line.CreatedAt = now
		}
		if line.UpdatedAt == "" {
			line.UpdatedAt = now
		}
		lines = append(lines, line)
	}

	return t, lines, nil
}
