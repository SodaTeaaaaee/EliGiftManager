package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	PreflightCarrierMappings(ctx context.Context, input dto.ImportCarrierMappingsInput) (*CarrierImportPlan, error)
	ExecuteCarrierImportPlan(ctx context.Context, plan *CarrierImportPlan) (*dto.ImportCarrierMappingsResult, error)
	DeleteMapping(ctx context.Context, id uint) error
}

type carrierMappingUseCase struct {
	mappingRepo      domain.CarrierMappingRepository
	profileRepo      domain.IntegrationProfileRepository
	templateMapping  *TemplateMappingService
	evidence         *ImportEvidenceUseCase
	externalRegistry *ExternalCarrierUseCase
	conflictAudit    *ExternalCarrierUseCase
}

type carrierImportPending struct {
	idx          int
	mapping      *domain.CarrierMapping
	canonicalKey string
	nameKey      string
}

// CarrierImportPlan is an immutable full-file preflight result. It contains no
// database mutations; ExecuteCarrierImportPlan applies only its validated rows.
type CarrierImportPlan struct {
	input           dto.ImportCarrierMappingsInput
	mode            string
	evidenceRun     *domain.ImportRun
	evidenceRecords []domain.ImportRawRecord
	pendingRows     []carrierImportPending
	invalidRows     map[int]struct{}
	reviewReasons   map[uint]string
	result          *dto.ImportCarrierMappingsResult
}

func (p *CarrierImportPlan) RejectsBusinessWrites() bool {
	return p != nil && p.mode == "reject_all" && p.result != nil && p.result.ErrorCount > 0
}

func WithCarrierImportEvidence(uc CarrierMappingUseCase, evidence *ImportEvidenceUseCase) CarrierMappingUseCase {
	c, ok := uc.(*carrierMappingUseCase)
	if ok {
		c.evidence = evidence
	}
	return uc
}

func WithExternalCarrierRegistry(uc CarrierMappingUseCase, registry *ExternalCarrierUseCase) CarrierMappingUseCase {
	c, ok := uc.(*carrierMappingUseCase)
	if ok {
		c.externalRegistry = registry
	}
	return uc
}

// WithCarrierConflictAudit attaches the base-database writer used to durably
// persist conflicts before the business transaction starts.
func WithCarrierConflictAudit(uc CarrierMappingUseCase, audit *ExternalCarrierUseCase) CarrierMappingUseCase {
	c, ok := uc.(*carrierMappingUseCase)
	if ok {
		c.conflictAudit = audit
	}
	return uc
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
	if err := requireCustomerResolutionFeature(ctx, uc.mappingRepo, domain.CustomerResolutionFeatureCarrierRegistry); err != nil {
		return nil, err
	}
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
// external names and Aliases JSON arrays on the profile's mappings. Factory
// shipment returns commonly provide only the display name, so name resolution is
// part of the reverse-mapping contract rather than a caller-side special case.
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
	var matched *domain.CarrierMapping
	for i := range mappings {
		m := &mappings[i]
		isMatch := strings.EqualFold(strings.TrimSpace(m.ExternalCarrierName), code)
		aliases, parseErr := parseAliases(m.Aliases)
		if parseErr == nil {
			for _, a := range aliases {
				if strings.EqualFold(strings.TrimSpace(a), code) {
					isMatch = true
					break
				}
			}
		}
		if !isMatch {
			continue
		}
		if matched != nil && matched.InternalCarrierCode != m.InternalCarrierCode {
			return "", "", fmt.Errorf("ambiguous carrier name/alias %q on profile %d", code, profileID)
		}
		if matched == nil {
			matched = m
		}
	}
	if matched != nil {
		return matched.InternalCarrierCode, matched.ExternalCarrierName, nil
	}
	return "", "", fmt.Errorf("no carrier mapping for external/alias %q on profile %d", code, profileID)
}

// ImportCarrierMappings performs full-file preflight before applying any
// registry or mapping writes. Controllers that own a business transaction call
// PreflightCarrierMappings outside it and ExecuteCarrierImportPlan inside it.
func (uc *carrierMappingUseCase) ImportCarrierMappings(ctx context.Context, input dto.ImportCarrierMappingsInput) (*dto.ImportCarrierMappingsResult, error) {
	plan, err := uc.PreflightCarrierMappings(ctx, input)
	if err != nil {
		return nil, err
	}
	return uc.ExecuteCarrierImportPlan(ctx, plan)
}

func (uc *carrierMappingUseCase) PreflightCarrierMappings(ctx context.Context, input dto.ImportCarrierMappingsInput) (*CarrierImportPlan, error) {
	if err := requireCustomerResolutionFeature(ctx, uc.mappingRepo, domain.CustomerResolutionFeatureCarrierRegistry); err != nil {
		return nil, err
	}
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
	var evidenceRun *domain.ImportRun
	var evidenceRecords []domain.ImportRawRecord
	if uc.evidence != nil {
		evidenceRows, unmapped := importEvidenceRows(orderedRows, headers, headerRows)
		parserMetadata := fmt.Sprintf(`{"hasHeader":%t,"sheetName":%q}`, rules.HasHeader, rules.SheetName)
		evidenceRun, evidenceRecords, err = uc.evidence.StartImportEvidence(ctx, "carrier_mapping", input.IntegrationProfileID, mode, input.FilePath, parserMetadata, evidenceRows, unmapped, nil)
		if err != nil {
			return nil, fmt.Errorf("start carrier import evidence: %w", err)
		}
	}

	plan := &CarrierImportPlan{
		input: input, mode: mode, evidenceRun: evidenceRun, evidenceRecords: evidenceRecords,
		invalidRows: make(map[int]struct{}), reviewReasons: make(map[uint]string),
	}
	rowReasons := make(map[int][]string)
	conflictRows := make(map[int]struct{})
	addRowError := func(index int, reason string) {
		for _, existing := range rowReasons[index] {
			if existing == reason {
				return
			}
		}
		rowReasons[index] = append(rowReasons[index], reason)
		plan.invalidRows[index] = struct{}{}
	}
	addConflict := func(index int, reason string) {
		addRowError(index, reason)
		conflictRows[index] = struct{}{}
	}
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
			markImportEvidenceFailure(evidenceRecords, i, "mapping_error", mapErr.Error(), warnings)
			addRowError(i, mapErr.Error())
			continue
		}
		m, buildErr := buildCarrierMappingFromApplied(applied, input.IntegrationProfileID)
		if buildErr != nil {
			markImportEvidenceFailure(evidenceRecords, i, "validation_error", buildErr.Error(), warnings)
			addRowError(i, buildErr.Error())
			continue
		}
		// Cold-start sheets may omit internal codes. Existing manual bindings are
		// reused when present; otherwise the external registry keeps a provisional
		// carrier for later explicit BindInternalCarrier.
		if m.InternalCarrierCode == "" {
			resolved, _, rErr := uc.ResolveByExternalOrAlias(ctx, input.IntegrationProfileID, m.ExternalCarrierCode)
			if rErr == nil {
				m.InternalCarrierCode = strings.TrimSpace(resolved)
			} else if uc.externalRegistry == nil {
				reason := fmt.Sprintf("carrier.internal_carrier_code missing and no external registry is configured for %q: %v", m.ExternalCarrierCode, rErr)
				addRowError(i, reason)
				markImportEvidenceFailure(evidenceRecords, i, "internal_carrier_unresolved", reason, warnings)
				continue
			}
		}
		key, keyErr := ExternalCarrierCanonicalKey(m.ExternalCarrierCode, m.ExternalCarrierName)
		if keyErr != nil {
			addRowError(i, keyErr.Error())
			markImportEvidenceFailure(evidenceRecords, i, "validation_error", keyErr.Error(), warnings)
			continue
		}
		plan.pendingRows = append(plan.pendingRows, carrierImportPending{idx: i, mapping: m, canonicalKey: key, nameKey: normalizeExternalCarrierName(m.ExternalCarrierName)})
	}

	keyRows := make(map[string][]carrierImportPending)
	nameRows := make(map[string][]carrierImportPending)
	for _, pending := range plan.pendingRows {
		keyRows[pending.canonicalKey] = append(keyRows[pending.canonicalKey], pending)
		if pending.nameKey != "" {
			nameRows[pending.nameKey] = append(nameRows[pending.nameKey], pending)
		}
	}
	for _, rows := range keyRows {
		if len(rows) < 2 {
			continue
		}
		// Exact duplicate observations are idempotent. They remain separate
		// successful input rows, while execution discovers the registry carrier
		// once from the lowest source row and preserves that first provenance.
		reason := ""
		base := rows[0].mapping
		baseName := strings.TrimSpace(base.ExternalCarrierName)
		for _, row := range rows[1:] {
			if !strings.EqualFold(baseName, strings.TrimSpace(row.mapping.ExternalCarrierName)) {
				reason = "same external carrier code has different names in import"
				break
			}
			if !sameCarrierImportMapping(base, row.mapping) {
				reason = "duplicate external carrier has conflicting mapping fields in import"
				break
			}
		}
		if reason == "" {
			continue
		}
		for _, row := range rows {
			addConflict(row.idx, reason)
		}
	}
	for _, rows := range nameRows {
		keys := make(map[string]struct{})
		for _, row := range rows {
			keys[row.canonicalKey] = struct{}{}
		}
		if len(keys) < 2 {
			continue
		}
		for _, row := range rows {
			addConflict(row.idx, "same external carrier name has different codes in import")
		}
	}

	var existing []domain.ExternalCarrier
	if uc.externalRegistry != nil {
		var listErr error
		existing, listErr = uc.externalRegistry.listDomainByProfile(ctx, input.IntegrationProfileID)
		if listErr != nil {
			return nil, fmt.Errorf("list external carriers for preflight: %w", listErr)
		}
		byKey := make(map[string]domain.ExternalCarrier, len(existing))
		byName := make(map[string][]domain.ExternalCarrier, len(existing))
		for _, carrier := range existing {
			byKey[carrier.CanonicalKey] = carrier
			nameKey := normalizeExternalCarrierName(carrier.ExternalCarrierName)
			if nameKey != "" {
				byName[nameKey] = append(byName[nameKey], carrier)
			}
		}
		for _, row := range plan.pendingRows {
			if carrier, ok := byKey[row.canonicalKey]; ok {
				exactCode := strings.EqualFold(strings.TrimSpace(carrier.ExternalCarrierCode), strings.TrimSpace(row.mapping.ExternalCarrierCode))
				exactName := strings.EqualFold(strings.TrimSpace(carrier.ExternalCarrierName), strings.TrimSpace(row.mapping.ExternalCarrierName))
				if !exactCode || !exactName {
					reason := "same external carrier code has different names"
					addConflict(row.idx, reason)
					plan.reviewReasons[carrier.ID] = reason
				}
			}
			for _, carrier := range byName[row.nameKey] {
				if row.nameKey != "" && carrier.CanonicalKey != row.canonicalKey {
					reason := "same external carrier name has different codes"
					addConflict(row.idx, reason)
					plan.reviewReasons[carrier.ID] = reason
				}
			}
		}
	}

	rowIndexes := make([]int, 0, len(rowReasons))
	for index := range rowReasons {
		rowIndexes = append(rowIndexes, index)
	}
	sort.Ints(rowIndexes)
	rowErrors := make([]dto.ImportCarrierMappingError, 0, len(rowIndexes))
	conflicts := make([]domain.ExternalCarrierConflict, 0, len(rowIndexes))
	for _, index := range rowIndexes {
		reason := strings.Join(rowReasons[index], "; ")
		rowErrors = append(rowErrors, dto.ImportCarrierMappingError{RowIndex: index, Reason: reason})
		_, isConflict := conflictRows[index]
		if isConflict && index >= 0 && index < len(evidenceRecords) {
			evidenceRecords[index].Outcome = "quarantined"
			evidenceRecords[index].ErrorCode = "carrier_conflict"
			evidenceRecords[index].ErrorMessage = reason
		}
		for _, pending := range plan.pendingRows {
			if pending.idx != index || !isConflict {
				continue
			}
			payload, _ := json.Marshal(map[string]string{"reason": reason})
			conflict := domain.ExternalCarrierConflict{IntegrationProfileID: input.IntegrationProfileID, CanonicalKey: pending.canonicalKey, ConflictKind: "import_preflight_conflict", ExternalCarrierCode: pending.mapping.ExternalCarrierCode, ExternalCarrierName: pending.mapping.ExternalCarrierName, InternalCarrierCode: pending.mapping.InternalCarrierCode, Payload: string(payload), CreatedAt: time.Now().UTC()}
			if evidenceRun != nil {
				conflict.SourceImportRunID = &evidenceRun.ID
				if index < len(evidenceRecords) {
					conflict.SourceRawRecordID = &evidenceRecords[index].ID
				}
			}
			conflicts = append(conflicts, conflict)
			break
		}
	}
	audit := uc.conflictAudit
	if audit == nil {
		audit = uc.externalRegistry
	}
	if len(conflicts) > 0 {
		if audit == nil {
			return nil, fmt.Errorf("carrier conflict audit is not configured")
		}
		if err := audit.RecordConflicts(ctx, conflicts); err != nil {
			return nil, fmt.Errorf("persist carrier conflict audit: %w", err)
		}
	}

	plan.result = &dto.ImportCarrierMappingsResult{
		ImportRunID:      importEvidenceRunID(evidenceRun),
		EvidenceDisabled: uc.evidence != nil && evidenceRun == nil,
		TotalProcessed:   total,
		ErrorCount:       len(rowErrors),
		Errors:           rowErrors,
		Warnings:         rowWarnings.warnings(),
	}
	if mode == "reject_all" && len(rowErrors) > 0 {
		if evidenceRun != nil {
			if err := uc.evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, "rejected"); err != nil {
				return nil, err
			}
		}
		return plan, nil
	}
	return plan, nil
}

func (uc *carrierMappingUseCase) ExecuteCarrierImportPlan(ctx context.Context, plan *CarrierImportPlan) (*dto.ImportCarrierMappingsResult, error) {
	if plan == nil || plan.result == nil {
		return nil, fmt.Errorf("carrier import plan is required")
	}
	result := plan.result
	if plan.mode == "reject_all" && result.ErrorCount > 0 {
		return result, nil
	}
	for id, reason := range plan.reviewReasons {
		if uc.externalRegistry == nil {
			return nil, fmt.Errorf("external carrier registry is not configured")
		}
		if err := uc.externalRegistry.MarkReview(ctx, id, reason); err != nil {
			return nil, fmt.Errorf("mark external carrier %d for review: %w", id, err)
		}
	}
	now := time.Now()
	pendingRows := append([]carrierImportPending(nil), plan.pendingRows...)
	sort.SliceStable(pendingRows, func(i, j int) bool { return pendingRows[i].idx < pendingRows[j].idx })
	observedByCanonicalKey := make(map[string]*dto.ExternalCarrierDTO)
	for _, p := range pendingRows {
		if _, invalid := plan.invalidRows[p.idx]; invalid {
			continue
		}
		var external *dto.ExternalCarrierDTO
		if uc.externalRegistry != nil {
			external = observedByCanonicalKey[p.canonicalKey]
			if external == nil {
				observation := carrierImportObservation(plan, p)
				observed, observeErr := uc.externalRegistry.ObserveExternalCarrierWithProvenance(ctx, observation)
				if observeErr != nil {
					return nil, fmt.Errorf("persist external carrier row %d: %w", p.idx, observeErr)
				}
				external = observed
				observedByCanonicalKey[p.canonicalKey] = observed
				result.ExternalCarriers = append(result.ExternalCarriers, *observed)
			}
		}
		if p.mapping.InternalCarrierCode == "" {
			if external != nil {
				markImportEvidenceSuccess(plan.evidenceRecords, p.idx, "external_carrier", external.ID)
			}
			result.SuccessCount++
			continue
		}
		existing, findErr := uc.mappingRepo.FindByProfileAndExternal(ctx, plan.input.IntegrationProfileID, p.mapping.ExternalCarrierCode)
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
				return nil, fmt.Errorf("update carrier mapping row %d: %w", p.idx, err)
			}
			result.UpdatedCount++
			result.SuccessCount++
			result.Mappings = append(result.Mappings, toCarrierMappingDTO(existing))
			markImportEvidenceSuccess(plan.evidenceRecords, p.idx, "carrier_mapping", existing.ID)
			continue
		}

		p.mapping.CreatedAt = now
		p.mapping.UpdatedAt = now
		if err := uc.mappingRepo.Create(ctx, p.mapping); err != nil {
			return nil, fmt.Errorf("create carrier mapping row %d: %w", p.idx, err)
		}
		result.CreatedCount++
		result.SuccessCount++
		result.Mappings = append(result.Mappings, toCarrierMappingDTO(p.mapping))
		markImportEvidenceSuccess(plan.evidenceRecords, p.idx, "carrier_mapping", p.mapping.ID)
	}
	if plan.evidenceRun != nil {
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		if err := uc.evidence.CompleteImportEvidence(ctx, plan.evidenceRun, plan.evidenceRecords, status); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func sameCarrierImportMapping(left, right *domain.CarrierMapping) bool {
	if left == nil || right == nil {
		return left == right
	}
	return strings.EqualFold(strings.TrimSpace(left.InternalCarrierCode), strings.TrimSpace(right.InternalCarrierCode)) &&
		strings.EqualFold(strings.TrimSpace(left.ExternalCarrierCode), strings.TrimSpace(right.ExternalCarrierCode)) &&
		strings.EqualFold(strings.TrimSpace(left.ExternalCarrierName), strings.TrimSpace(right.ExternalCarrierName)) &&
		strings.TrimSpace(left.Aliases) == strings.TrimSpace(right.Aliases) &&
		left.IsDefault == right.IsDefault
}

func carrierImportObservation(plan *CarrierImportPlan, pending carrierImportPending) ExternalCarrierObservationInput {
	input := ExternalCarrierObservationInput{
		IntegrationProfileID: plan.input.IntegrationProfileID,
		ExternalCarrierCode:  pending.mapping.ExternalCarrierCode,
		ExternalCarrierName:  pending.mapping.ExternalCarrierName,
	}
	if plan.evidenceRun == nil || plan.evidenceRun.ID == 0 || pending.idx < 0 || pending.idx >= len(plan.evidenceRecords) {
		return input
	}
	rawRecordID := plan.evidenceRecords[pending.idx].ID
	if rawRecordID == 0 {
		return input
	}
	input.SourceImportRunID = copyOptionalUint(&plan.evidenceRun.ID)
	input.SourceRawRecordID = copyOptionalUint(&rawRecordID)
	return input
}

func (uc *carrierMappingUseCase) DeleteMapping(ctx context.Context, id uint) error {
	if err := requireCustomerResolutionFeature(ctx, uc.mappingRepo, domain.CustomerResolutionFeatureCarrierRegistry); err != nil {
		return err
	}
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

	// Name-key fallback is explicit for platforms that omit external codes.
	if external == "" && name == "" {
		return nil, fmt.Errorf("carrier external code or name is required")
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
