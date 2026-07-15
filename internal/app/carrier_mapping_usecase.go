package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// CarrierMappingUseCase manages carrier code translations for integration profiles.
type CarrierMappingUseCase interface {
	CreateMapping(ctx context.Context, input dto.CreateCarrierMappingInput) (*dto.CarrierMappingDTO, error)
	ListMappingsByProfile(ctx context.Context, profileID uint) ([]dto.CarrierMappingDTO, error)
	// ResolveCarrier maps an internal carrier code to its external code/name.
	ResolveCarrier(ctx context.Context, profileID uint, internalCode string) (externalCode string, externalName string, err error)
	// ResolveByExternalOrAlias maps an external code or alias back to the internal carrier code.
	ResolveByExternalOrAlias(ctx context.Context, profileID uint, externalOrAlias string) (internalCode string, externalName string, err error)
	ImportCarrierMappings(ctx context.Context, input dto.ImportCarrierMappingsInput) (*dto.ImportCarrierMappingsResult, error)
	DeleteMapping(ctx context.Context, id uint) error
}

type carrierMappingUseCase struct {
	mappingRepo     domain.CarrierMappingRepository
	profileRepo     domain.IntegrationProfileRepository
	templateMapping *TemplateMappingService
}

// NewCarrierMappingUseCase returns a CarrierMappingUseCase.
func NewCarrierMappingUseCase(
	mappingRepo domain.CarrierMappingRepository,
	profileRepo domain.IntegrationProfileRepository,
) CarrierMappingUseCase {
	return &carrierMappingUseCase{
		mappingRepo: mappingRepo,
		profileRepo: profileRepo,
	}
}

// WithCarrierImportDeps attaches the template mapping service needed for ImportCarrierMappings.
func WithCarrierImportDeps(uc CarrierMappingUseCase, mapping *TemplateMappingService) CarrierMappingUseCase {
	c, ok := uc.(*carrierMappingUseCase)
	if !ok {
		return uc
	}
	c.templateMapping = mapping
	return c
}

func (uc *carrierMappingUseCase) CreateMapping(ctx context.Context, input dto.CreateCarrierMappingInput) (*dto.CarrierMappingDTO, error) {
	if input.IntegrationProfileID == 0 {
		return nil, fmt.Errorf("create carrier mapping: integrationProfileId is required")
	}
	if input.InternalCarrierCode == "" {
		return nil, fmt.Errorf("create carrier mapping: internalCarrierCode must not be empty")
	}
	if input.ExternalCarrierCode == "" {
		return nil, fmt.Errorf("create carrier mapping: externalCarrierCode must not be empty")
	}
	if err := validateAliasesJSON(input.Aliases); err != nil {
		return nil, fmt.Errorf("create carrier mapping: %w", err)
	}

	// Validate profile exists.
	if _, err := uc.profileRepo.FindByID(ctx, input.IntegrationProfileID); err != nil {
		return nil, fmt.Errorf("create carrier mapping: integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	mapping := &domain.CarrierMapping{
		IntegrationProfileID: input.IntegrationProfileID,
		InternalCarrierCode:  input.InternalCarrierCode,
		ExternalCarrierCode:  input.ExternalCarrierCode,
		ExternalCarrierName:  input.ExternalCarrierName,
		Aliases:              input.Aliases,
		IsDefault:            input.IsDefault,
	}
	if err := uc.mappingRepo.Create(ctx, mapping); err != nil {
		return nil, fmt.Errorf("create carrier mapping: %w", err)
	}
	result := toCarrierMappingDTO(mapping)
	return &result, nil
}

func (uc *carrierMappingUseCase) ListMappingsByProfile(ctx context.Context, profileID uint) ([]dto.CarrierMappingDTO, error) {
	mappings, err := uc.mappingRepo.ListByProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("list carrier mappings for profile %d: %w", profileID, err)
	}
	result := make([]dto.CarrierMappingDTO, len(mappings))
	for i, m := range mappings {
		result[i] = toCarrierMappingDTO(&m)
	}
	return result, nil
}

func (uc *carrierMappingUseCase) ResolveCarrier(ctx context.Context, profileID uint, internalCode string) (string, string, error) {
	if internalCode == "" {
		return "", "", fmt.Errorf("resolve carrier: internalCode must not be empty")
	}
	mapping, err := uc.mappingRepo.FindByProfileAndInternal(ctx, profileID, internalCode)
	if err != nil {
		return "", "", fmt.Errorf("resolve carrier %q for profile %d: %w", internalCode, profileID, err)
	}
	return mapping.ExternalCarrierCode, mapping.ExternalCarrierName, nil
}

// ResolveByExternalOrAlias looks up by exact ExternalCarrierCode first, then scans
// Aliases JSON arrays on the profile's mappings. Returns internal code + external name.
func (uc *carrierMappingUseCase) ResolveByExternalOrAlias(ctx context.Context, profileID uint, externalOrAlias string) (string, string, error) {
	code := strings.TrimSpace(externalOrAlias)
	if code == "" {
		return "", "", fmt.Errorf("resolve by external/alias: code must not be empty")
	}

	// Exact external code.
	if mapping, err := uc.mappingRepo.FindByProfileAndExternal(ctx, profileID, code); err == nil && mapping != nil {
		return mapping.InternalCarrierCode, mapping.ExternalCarrierName, nil
	}

	// Alias scan (in-memory — alias sets are small per profile).
	mappings, err := uc.mappingRepo.ListByProfile(ctx, profileID)
	if err != nil {
		return "", "", fmt.Errorf("list carrier mappings for alias scan: %w", err)
	}
	for i := range mappings {
		m := &mappings[i]
		aliases, parseErr := parseAliases(m.Aliases)
		if parseErr != nil {
			continue
		}
		for _, a := range aliases {
			if strings.EqualFold(strings.TrimSpace(a), code) {
				return m.InternalCarrierCode, m.ExternalCarrierName, nil
			}
		}
	}
	return "", "", fmt.Errorf("no carrier mapping for external/alias %q on profile %d", code, profileID)
}

// ImportCarrierMappings upserts CarrierMapping rows from a template-mapped sheet
// (document type typically reuses a dedicated binding; dest keys carrier.*).
// Upsert key: external_carrier_code within the profile.
func (uc *carrierMappingUseCase) ImportCarrierMappings(ctx context.Context, input dto.ImportCarrierMappingsInput) (*dto.ImportCarrierMappingsResult, error) {
	if uc.templateMapping == nil {
		return nil, fmt.Errorf("import carrier mappings: template mapping deps not configured")
	}
	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	if mode != "reject_all" && mode != "skip_invalid" {
		return nil, fmt.Errorf("invalid importMode %q", mode)
	}
	if input.IntegrationProfileID == 0 {
		return nil, fmt.Errorf("integrationProfileId is required")
	}
	if _, err := uc.profileRepo.FindByID(ctx, input.IntegrationProfileID); err != nil {
		return nil, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}

	// Prefer a dedicated document type; fall back is caller's responsibility via binding.
	_, rules, err := uc.templateMapping.ResolveTemplateAndRules(ctx, input.IntegrationProfileID, "import_carrier_mapping")
	if err != nil {
		// Fallback: try import_supplier_shipment-adjacent type is NOT used; surface error.
		return nil, fmt.Errorf("template pipeline: %w", err)
	}

	orderedRows, headers, headerRows, total, _, cleanup, err := loadImportRows(input.FilePath, input.Rows, rules, nil)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}

	type pending struct {
		idx     int
		mapping *domain.CarrierMapping
	}
	var pendingRows []pending
	var rowErrors []dto.ImportCarrierMappingError
	var rowWarnings rowWarningCollector

	for i := 0; i < total; i++ {
		var applied map[string]string
		var mapErr error
		var warnings []string
		if len(orderedRows) > 0 {
			applied, warnings, mapErr = ApplyRow(orderedRows[i], headers, rules)
		} else {
			applied, warnings, mapErr = applyHeaderMap(headerRows[i], rules)
		}
		rowWarnings.add(i, warnings)
		if mapErr != nil {
			rowErrors = append(rowErrors, dto.ImportCarrierMappingError{RowIndex: i, Reason: mapErr.Error()})
			if mode == "reject_all" {
				break
			}
			continue
		}
		m, buildErr := buildCarrierMappingFromApplied(applied, input.IntegrationProfileID)
		if buildErr != nil {
			rowErrors = append(rowErrors, dto.ImportCarrierMappingError{RowIndex: i, Reason: buildErr.Error()})
			if mode == "reject_all" {
				break
			}
			continue
		}
		// Platform carrier sheets often omit internal codes. When missing, try
		// ResolveByExternalOrAlias to backfill from an existing mapping/alias.
		// Failure → skip_invalid detail (or reject_all break).
		if m.InternalCarrierCode == "" {
			resolved, _, rErr := uc.ResolveByExternalOrAlias(ctx, input.IntegrationProfileID, m.ExternalCarrierCode)
			if rErr != nil || strings.TrimSpace(resolved) == "" {
				reason := fmt.Sprintf(
					"carrier.internal_carrier_code missing and ResolveByExternalOrAlias failed for %q: %v",
					m.ExternalCarrierCode, rErr,
				)
				rowErrors = append(rowErrors, dto.ImportCarrierMappingError{RowIndex: i, Reason: reason})
				if mode == "reject_all" {
					break
				}
				continue
			}
			m.InternalCarrierCode = resolved
		}
		pendingRows = append(pendingRows, pending{idx: i, mapping: m})
	}

	result := &dto.ImportCarrierMappingsResult{
		TotalProcessed: total,
		ErrorCount:     len(rowErrors),
		Errors:         rowErrors,
		Warnings:       rowWarnings.warnings(),
	}
	if mode == "reject_all" && len(rowErrors) > 0 {
		return result, nil
	}

	now := time.Now()
	for _, p := range pendingRows {
		existing, findErr := uc.mappingRepo.FindByProfileAndExternal(ctx, input.IntegrationProfileID, p.mapping.ExternalCarrierCode)
		if findErr == nil && existing != nil {
			existing.InternalCarrierCode = p.mapping.InternalCarrierCode
			if p.mapping.ExternalCarrierName != "" {
				existing.ExternalCarrierName = p.mapping.ExternalCarrierName
			}
			if p.mapping.Aliases != "" {
				existing.Aliases = p.mapping.Aliases
			}
			existing.IsDefault = p.mapping.IsDefault
			existing.UpdatedAt = now
			if err := uc.mappingRepo.Update(ctx, existing); err != nil {
				result.Errors = append(result.Errors, dto.ImportCarrierMappingError{
					RowIndex: p.idx, Reason: fmt.Sprintf("update: %v", err),
				})
				result.ErrorCount++
				continue
			}
			result.UpdatedCount++
			result.SuccessCount++
			result.Mappings = append(result.Mappings, toCarrierMappingDTO(existing))
			continue
		}

		p.mapping.CreatedAt = now
		p.mapping.UpdatedAt = now
		if err := uc.mappingRepo.Create(ctx, p.mapping); err != nil {
			result.Errors = append(result.Errors, dto.ImportCarrierMappingError{
				RowIndex: p.idx, Reason: fmt.Sprintf("create: %v", err),
			})
			result.ErrorCount++
			continue
		}
		result.CreatedCount++
		result.SuccessCount++
		result.Mappings = append(result.Mappings, toCarrierMappingDTO(p.mapping))
	}

	return result, nil
}

func (uc *carrierMappingUseCase) DeleteMapping(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("delete carrier mapping: id is required")
	}
	return uc.mappingRepo.Delete(ctx, id)
}

func buildCarrierMappingFromApplied(applied map[string]string, profileID uint) (*domain.CarrierMapping, error) {
	get := func(field string) string {
		if v, ok := applied["carrier."+field]; ok {
			return strings.TrimSpace(v)
		}
		return strings.TrimSpace(applied[field])
	}

	internal := get("internal_carrier_code")
	if internal == "" {
		internal = get("internal_code")
	}
	external := get("external_carrier_code")
	if external == "" {
		external = get("external_code")
	}
	name := get("external_carrier_name")
	if name == "" {
		name = get("external_name")
	}
	aliasesRaw := get("aliases")
	isDefault := parseBoolish(get("is_default"))

	// Empty external code rows are skip_invalid (caller records the reason).
	if external == "" {
		return nil, fmt.Errorf("carrier.external_carrier_code is empty")
	}
	// internal may be empty — ImportCarrierMappings backfills via ResolveByExternalOrAlias.

	// Normalise aliases: accept JSON array or comma-separated list.
	aliasesJSON := ""
	if aliasesRaw != "" {
		if err := validateAliasesJSON(aliasesRaw); err == nil {
			aliasesJSON = aliasesRaw
		} else {
			parts := strings.Split(aliasesRaw, ",")
			cleaned := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					cleaned = append(cleaned, p)
				}
			}
			b, _ := json.Marshal(cleaned)
			aliasesJSON = string(b)
		}
	}

	return &domain.CarrierMapping{
		IntegrationProfileID: profileID,
		InternalCarrierCode:  internal,
		ExternalCarrierCode:  external,
		ExternalCarrierName:  name,
		Aliases:              aliasesJSON,
		IsDefault:            isDefault,
	}, nil
}

func validateAliasesJSON(raw string) error {
	if raw == "" {
		return nil
	}
	var aliases []string
	if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
		return fmt.Errorf("aliases must be a JSON string array: %w", err)
	}
	return nil
}

func parseAliases(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var aliases []string
	if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
		return nil, err
	}
	return aliases, nil
}

func toCarrierMappingDTO(m *domain.CarrierMapping) dto.CarrierMappingDTO {
	return dto.CarrierMappingDTO{
		ID:                   m.ID,
		IntegrationProfileID: m.IntegrationProfileID,
		InternalCarrierCode:  m.InternalCarrierCode,
		ExternalCarrierCode:  m.ExternalCarrierCode,
		ExternalCarrierName:  m.ExternalCarrierName,
		Aliases:              m.Aliases,
		IsDefault:            m.IsDefault,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}
