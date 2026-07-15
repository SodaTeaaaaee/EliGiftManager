package main

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// DemandController exposes demand-intake Wails bindings.
type DemandController struct {
	intakeUC             app.DemandIntakeUseCase
	entitlementRoutingUC app.EntitlementRoutingUseCase
	demandRepo           domain.DemandDocumentRepository
	profileRepo          domain.CustomerProfileRepository
	integrationProfile   domain.IntegrationProfileRepository
	assignmentRepo       domain.WaveDemandAssignmentRepository
	waveRepo             domain.WaveRepository
	identityResolution   *app.IdentityResolutionService
	templateMapping      *app.TemplateMappingService
	inboxAssignmentRepo  domain.DemandInboxAssignmentRepository
	inboxLineRepo        domain.DemandInboxLineRepository
	addressUC            app.AddressManagementUseCase
}

func NewDemandController() *DemandController {
	gdb := db.GetDB()
	demandRepo := infra.NewDemandRepository(gdb)
	profileRepo := infra.NewProfileRepository(gdb)
	integrationProfileRepo := infra.NewIntegrationProfileRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	addressRepo := infra.NewAddressRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	return &DemandController{
		intakeUC:             app.NewDemandIntakeUseCase(demandRepo),
		entitlementRoutingUC: app.NewEntitlementRoutingUseCase(demandRepo, assignmentRepo),
		demandRepo:           demandRepo,
		profileRepo:          profileRepo,
		integrationProfile:   integrationProfileRepo,
		assignmentRepo:       assignmentRepo,
		waveRepo:             waveRepo,
		identityResolution:   app.NewIdentityResolutionService(profileRepo),
		templateMapping:      app.NewTemplateMappingService(templateRepo, bindingRepo, integrationProfileRepo),
		inboxAssignmentRepo:  infra.NewDemandInboxAssignmentRepository(gdb),
		inboxLineRepo:        infra.NewDemandInboxLineRepository(gdb),
		addressUC:            app.NewAddressManagementUseCase(addressRepo, fulfillRepo),
	}
}

// ImportDemandDocument imports a DemandDocument with its DemandLines.
func (c *DemandController) ImportDemandDocument(input dto.CreateDemandInput) (dto.DemandDocumentDTO, error) {
	ctx := appContext
	if input.CustomerProfileID != nil {
		if _, err := c.profileRepo.FindByID(ctx, *input.CustomerProfileID); err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("customer profile %d does not exist", *input.CustomerProfileID)
		}
	}

	// Silent override — backend is the final arbiter for profile-driven fields.
	// When an integration profile is selected, DemandKind / SourceChannel /
	// SourceSurface are dictated by the profile configuration; any values the
	// frontend submitted for these fields are intentionally discarded. This
	// ensures data consistency regardless of frontend state or user edits.
	// Frontend validation is purely UX guidance and does NOT constitute authority.
	effectiveKind := input.Kind
	effectiveSourceChannel := input.SourceChannel
	effectiveSourceSurface := input.SourceSurface

	var resolvedProfileID *uint
	if input.IntegrationProfileID != nil {
		profile, err := c.integrationProfile.FindByID(ctx, *input.IntegrationProfileID)
		if err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("integration profile %d does not exist", *input.IntegrationProfileID)
		}
		if profile.DemandKind != "" {
			effectiveKind = profile.DemandKind
		}
		if profile.SourceChannel != "" {
			effectiveSourceChannel = profile.SourceChannel
		}
		if profile.SourceSurface != "" {
			effectiveSourceSurface = profile.SourceSurface
		}

		// Auto-resolve CustomerProfile via identity when SourceCustomerRef is provided.
		if input.CustomerProfileID == nil && input.SourceCustomerRef != "" && effectiveSourceChannel != "" {
			identityType := app.ResolveIdentityStrategy(profile.IdentityStrategy)
			pid, resolveErr := c.identityResolution.ResolveOrCreateProfile(ctx, effectiveSourceChannel, input.SourceCustomerRef, identityType)
			if resolveErr != nil {
				return dto.DemandDocumentDTO{}, fmt.Errorf("identity resolution failed: %w", resolveErr)
			}
			resolvedProfileID = &pid
		}
	}

	effectiveCustomerProfileID := input.CustomerProfileID
	if resolvedProfileID != nil {
		effectiveCustomerProfileID = resolvedProfileID
	}

	doc := domain.DemandDocument{
		Kind:                 effectiveKind,
		CaptureMode:          input.CaptureMode,
		SourceChannel:        effectiveSourceChannel,
		SourceSurface:        effectiveSourceSurface,
		SourceDocumentNo:     input.SourceDocumentNo,
		SourceCustomerRef:    input.SourceCustomerRef,
		CustomerProfileID:    effectiveCustomerProfileID,
		IntegrationProfileID: input.IntegrationProfileID,
	}
	lines := make([]*domain.DemandLine, len(input.Lines))
	for i, l := range input.Lines {
		lines[i] = &domain.DemandLine{
			LineType:              l.LineType,
			ObligationTriggerKind: l.ObligationTriggerKind,
			EntitlementAuthority:  l.EntitlementAuthority,
			RecipientInputState:   l.RecipientInputState,
			RoutingDisposition:    l.RoutingDisposition,
			RoutingReasonCode:     l.RoutingReasonCode,
			EligibilityContextRef: l.EligibilityContextRef,
			EntitlementCode:       l.EntitlementCode,
			GiftLevelSnapshot:     l.GiftLevelSnapshot,
			ProductMasterID:       l.ProductMasterID,
			RecipientInputPayload: l.RecipientInputPayload,
			ExternalTitle:         l.ExternalTitle,
			RequestedQuantity:     l.RequestedQuantity,
		}
	}
	if err := c.intakeUC.ImportDemand(ctx, &doc, lines); err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(&doc), nil
}

// ImportDemandFromCSV imports a demand document using a template-driven CSV pipeline.
func (c *DemandController) ImportDemandFromCSV(input dto.ImportDemandTemplateInput) (dto.DemandDocumentDTO, error) {
	ctx := appContext
	profile, err := c.integrationProfile.FindByID(ctx, input.IntegrationProfileID)
	if err != nil {
		return dto.DemandDocumentDTO{}, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}
	docType := input.DocumentType
	if docType == "" {
		docType = "import_entitlement"
	}
	_, mappedLines, err := c.templateMapping.BuildImportPipeline(ctx, profile.ID, docType, input.Rows)
	if err != nil {
		return dto.DemandDocumentDTO{}, fmt.Errorf("template pipeline: %w", err)
	}
	var customerProfileID *uint
	if input.SourceCustomerRef != "" && profile.SourceChannel != "" {
		identityType := app.ResolveIdentityStrategy(profile.IdentityStrategy)
		pid, resolveErr := c.identityResolution.ResolveOrCreateProfile(ctx, profile.SourceChannel, input.SourceCustomerRef, identityType)
		if resolveErr != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("identity resolution: %w", resolveErr)
		}
		customerProfileID = &pid
	}
	doc := domain.DemandDocument{
		Kind:                 profile.DemandKind,
		CaptureMode:          "document_import",
		SourceChannel:        profile.SourceChannel,
		SourceSurface:        profile.SourceSurface,
		SourceDocumentNo:     input.SourceDocumentNo,
		SourceCustomerRef:    input.SourceCustomerRef,
		CustomerProfileID:    customerProfileID,
		IntegrationProfileID: &profile.ID,
	}
	if err := c.intakeUC.ImportDemand(ctx, &doc, mappedLines); err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(&doc), nil
}

// ListDemandDocuments lists all demand documents.
func (c *DemandController) ListDemandDocuments() ([]dto.DemandDocumentDTO, error) {
	ctx := appContext
	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DemandDocumentDTO, len(docs))
	for i, doc := range docs {
		result[i] = domainToDemandDTO(&doc)
	}
	return result, nil
}

// ListUnassignedDemandDocuments returns demand documents not assigned to any wave.
func (c *DemandController) ListUnassignedDemandDocuments() ([]dto.DemandDocumentDTO, error) {
	ctx := appContext
	docs, err := c.demandRepo.ListUnassigned(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DemandDocumentDTO, len(docs))
	for i, doc := range docs {
		result[i] = domainToDemandDTO(&doc)
	}
	return result, nil
}

// ListDemandLines returns all demand lines for a given document.
func (c *DemandController) ListDemandLines(documentID uint) ([]dto.DemandLineDTO, error) {
	ctx := appContext
	lines, err := c.demandRepo.ListLinesByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.DemandLineDTO, len(lines))
	for i, line := range lines {
		result[i] = domainToDemandLineDTO(&line)
	}
	return result, nil
}

// GetDemandDocument returns a single demand document by ID.
func (c *DemandController) GetDemandDocument(id uint) (dto.DemandDocumentDTO, error) {
	ctx := appContext
	doc, err := c.demandRepo.FindByID(ctx, id)
	if err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(doc), nil
}

// ListDemandInboxRows returns a paginated, server-filtered page of demand inbox rows.
//
// Prefetch strategy (fixes the former N+1 pattern, which issued one
// assignmentRepo.ListByDemandDocument + one demandRepo.ListLinesByDocument +
// one integrationProfile.FindByID call PER document inside the loop):
//  1. Load all documents, all waves, and all integration profiles once; index waves/profiles
//     into maps (same pattern as the pre-existing waveMap prefetch below).
//  2. Apply doc-level filters (demandKind, integrationProfileId) in memory.
//  3. Bulk-fetch assignment state for the filtered doc IDs in a single query, to evaluate the
//     assignment filter and compute TotalCount for pagination.
//  4. Slice the filtered+assignment-filtered set to the requested page.
//  5. Bulk-fetch demand lines for ONLY the current page's doc IDs in a single query, then
//     assemble rows from the prefetched maps — no further per-row DB calls.
func (c *DemandController) ListDemandInboxRows(input dto.DemandInboxFilterInput, pageInput dto.PaginationInput) (dto.DemandInboxRowListDTO, error) {
	ctx := appContext
	pageInput = dto.NormalizePagination(pageInput)

	docs, err := c.demandRepo.List(ctx)
	if err != nil {
		return dto.DemandInboxRowListDTO{}, err
	}
	waves, err := c.waveRepo.List(ctx)
	if err != nil {
		return dto.DemandInboxRowListDTO{}, err
	}
	profiles, err := c.integrationProfile.List(ctx)
	if err != nil {
		return dto.DemandInboxRowListDTO{}, err
	}

	// Stage 1: doc-level filters (demandKind, server-side integrationProfileId filter).
	filtered := make([]domain.DemandDocument, 0, len(docs))
	for _, doc := range docs {
		if input.DemandKind != "" && doc.Kind != input.DemandKind {
			continue
		}
		if input.IntegrationProfileID != nil {
			if doc.IntegrationProfileID == nil || *doc.IntegrationProfileID != *input.IntegrationProfileID {
				continue
			}
		}
		filtered = append(filtered, doc)
	}
	filteredIDs := make([]uint, len(filtered))
	for i, doc := range filtered {
		filteredIDs[i] = doc.ID
	}

	// Stage 2: bulk-fetch assignment state for the filtered set (single query, replaces the
	// former per-document ListByDemandDocument call).
	assignments, err := c.inboxAssignmentRepo.ListByDemandDocumentIDs(ctx, filteredIDs)
	if err != nil {
		return dto.DemandInboxRowListDTO{}, err
	}
	assignmentsByDoc := make(map[uint][]domain.WaveDemandAssignment, len(filteredIDs))
	for _, a := range assignments {
		assignmentsByDoc[a.DemandDocumentID] = append(assignmentsByDoc[a.DemandDocumentID], a)
	}

	// Stage 3: assignment-state filter.
	final := make([]domain.DemandDocument, 0, len(filtered))
	for _, doc := range filtered {
		docAssignments := assignmentsByDoc[doc.ID]
		assigned := len(docAssignments) > 0
		if input.Assignment == "assigned" && !assigned {
			continue
		}
		if input.Assignment == "unassigned" && assigned {
			continue
		}
		if input.WaveID != nil {
			matchesWave := false
			for _, assignment := range docAssignments {
				if assignment.WaveID == *input.WaveID {
					matchesWave = true
					break
				}
			}
			if !matchesWave {
				continue
			}
		}
		final = append(final, doc)
	}

	result := dto.DemandInboxRowListDTO{
		Pagination: dto.PaginationResult{
			Page:       pageInput.Page,
			PageSize:   pageInput.PageSize,
			TotalCount: len(final),
		},
	}
	result.Pagination.ComputePages()

	start := (pageInput.Page - 1) * pageInput.PageSize
	if start >= len(final) {
		result.Rows = []dto.DemandInboxRowDTO{}
		return result, nil
	}
	end := start + pageInput.PageSize
	if end > len(final) {
		end = len(final)
	}
	pageSlice := final[start:end]
	pageIDs := make([]uint, len(pageSlice))
	for i, doc := range pageSlice {
		pageIDs[i] = doc.ID
	}

	// Stage 4: bulk-fetch demand lines for only the current page's documents (single query,
	// replaces the former per-document ListLinesByDocument call).
	lines, err := c.inboxLineRepo.ListLinesByDocumentIDs(ctx, pageIDs)
	if err != nil {
		return dto.DemandInboxRowListDTO{}, err
	}
	result.Rows = app.AssembleDemandInboxRows(pageSlice, assignments, lines, waves, profiles)
	return result, nil
}

// domainToDemandDTO converts a domain DemandDocument to a DTO.
func domainToDemandDTO(doc *domain.DemandDocument) dto.DemandDocumentDTO {
	if doc == nil {
		return dto.DemandDocumentDTO{}
	}
	return dto.DemandDocumentDTO{
		ID:                   doc.ID,
		Kind:                 doc.Kind,
		CaptureMode:          doc.CaptureMode,
		SourceChannel:        doc.SourceChannel,
		SourceSurface:        doc.SourceSurface,
		IntegrationProfileID: doc.IntegrationProfileID,
		SourceDocumentNo:     doc.SourceDocumentNo,
		SourceCustomerRef:    doc.SourceCustomerRef,
		CustomerProfileID:    doc.CustomerProfileID,
		SourceCreatedAt:      doc.SourceCreatedAt,
		SourcePaidAt:         doc.SourcePaidAt,
		Currency:             doc.Currency,
		AuthoritySnapshotAt:  doc.AuthoritySnapshotAt,
		RawPayload:           doc.RawPayload,
		ExtraData:            doc.ExtraData,
		CreatedAt:            doc.CreatedAt,
		UpdatedAt:            doc.UpdatedAt,
	}
}

// domainToDemandLineDTO converts a domain DemandLine to a DTO.
func domainToDemandLineDTO(line *domain.DemandLine) dto.DemandLineDTO {
	if line == nil {
		return dto.DemandLineDTO{}
	}
	return dto.DemandLineDTO{
		ID:                    line.ID,
		DemandDocumentID:      line.DemandDocumentID,
		SourceLineNo:          intPtr(line.SourceLineNo),
		LineType:              line.LineType,
		ObligationTriggerKind: line.ObligationTriggerKind,
		EntitlementAuthority:  line.EntitlementAuthority,
		RecipientInputState:   line.RecipientInputState,
		RoutingDisposition:    line.RoutingDisposition,
		RoutingReasonCode:     line.RoutingReasonCode,
		EligibilityContextRef: line.EligibilityContextRef,
		ProductMasterID:       line.ProductMasterID,
		ExternalTitle:         line.ExternalTitle,
		RequestedQuantity:     line.RequestedQuantity,
		EntitlementCode:       line.EntitlementCode,
		GiftLevelSnapshot:     line.GiftLevelSnapshot,
		RecipientInputPayload: line.RecipientInputPayload,
		RawPayload:            line.RawPayload,
		ExtraData:             line.ExtraData,
		CreatedAt:             line.CreatedAt,
		UpdatedAt:             line.UpdatedAt,
	}
}

func intPtr(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func domainLineSliceToPtrs(lines []domain.DemandLine) []*domain.DemandLine {
	ptrs := make([]*domain.DemandLine, len(lines))
	for i := range lines {
		ptrs[i] = &lines[i]
	}
	return ptrs
}

// UpdateDemandLineRouting updates routing disposition, recipient input state, and reason code
// for a single demand line.
func (c *DemandController) UpdateDemandLineRouting(input dto.UpdateDemandLineRoutingInput) error {
	ctx := appContext
	return c.entitlementRoutingUC.UpdateDemandLineRouting(ctx, input)
}

// BatchUpdateDemandLineRouting applies routing updates to multiple demand lines in one call.
func (c *DemandController) BatchUpdateDemandLineRouting(input dto.BatchUpdateDemandLineRoutingInput) (dto.BatchUpdateDemandLineRoutingResult, error) {
	ctx := appContext
	result, err := c.entitlementRoutingUC.BatchUpdateDemandLineRouting(ctx, input)
	if err != nil {
		return dto.BatchUpdateDemandLineRoutingResult{}, err
	}
	return *result, nil
}

// GetWaveRoutingStats returns routing disposition counts for all demand lines in a wave.
func (c *DemandController) GetWaveRoutingStats(waveID uint) (dto.WaveRoutingStatsDTO, error) {
	ctx := appContext
	stats, err := c.entitlementRoutingUC.GetWaveRoutingStats(ctx, waveID)
	if err != nil {
		return dto.WaveRoutingStatsDTO{}, err
	}
	return *stats, nil
}
