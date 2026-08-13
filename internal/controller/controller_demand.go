package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// DemandController exposes demand-intake Wails bindings.
type DemandController struct {
	gdb                  *gorm.DB
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
		gdb:                  gdb,
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

func (c *DemandController) requireCustomerResolutionWrites(ctx context.Context) error {
	if gate, ok := c.profileRepo.(domain.CustomerResolutionFeatureGate); ok && gate != nil {
		return gate.RequireFeature(ctx, domain.CustomerResolutionFeatureWrites)
	}
	if c.gdb != nil {
		return infra.NewCustomerResolutionFeaturePolicyRepository(c.gdb).RequireFeature(ctx, domain.CustomerResolutionFeatureWrites)
	}
	return &domain.FeaturePolicyError{Code: domain.FeaturePolicyCodeUnavailable,
		Feature: domain.CustomerResolutionFeatureWrites, Detail: "customer resolution feature gate is not configured"}
}

// ImportDemandDocument imports a DemandDocument with its DemandLines.
func (c *DemandController) ImportDemandDocument(input dto.CreateDemandInput) (dto.DemandDocumentDTO, error) {
	ctx := appContext
	if err := c.requireCustomerResolutionWrites(ctx); err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	// Kind / SourceSurface / IdentityStrategy come from the resolved
	// documentType. The integration profile is the platform (SourceChannel);
	// it is not authority for persisted Kind/Surface or customer-resolution
	// strategy.
	effectiveKind := input.Kind
	effectiveSourceChannel := input.SourceChannel
	effectiveSourceSurface := input.SourceSurface
	identityStrategy := ""

	requestedDocType := requestedDemandDocumentType(input.DocumentType)
	if requestedDocType == "" {
		requestedDocType = requestedDemandDocumentType(input.Kind)
	}

	var integrationProfile *domain.IntegrationProfile
	if input.IntegrationProfileID != nil {
		profile, err := c.integrationProfile.FindByID(ctx, *input.IntegrationProfileID)
		if err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("integration profile %d does not exist", *input.IntegrationProfileID)
		}
		docType, err := app.ResolveDemandImportDocumentType(profile, requestedDocType)
		if err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("resolve demand document type: %w", err)
		}
		interp, err := app.InterpretDemandImportDocumentType(docType)
		if err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("interpret demand document type: %w", err)
		}
		effectiveKind, effectiveSourceSurface, identityStrategy = interp.DemandKind, interp.SourceSurface, interp.IdentityStrategy
		if profile.SourceChannel != "" {
			effectiveSourceChannel = profile.SourceChannel
		}
		integrationProfile = profile
	} else if requestedDocType == "import_entitlement" || requestedDocType == "import_sales_order" {
		interp, err := app.InterpretDemandImportDocumentType(requestedDocType)
		if err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("interpret demand document type: %w", err)
		}
		effectiveKind, effectiveSourceSurface = interp.DemandKind, interp.SourceSurface
	}
	if c.gdb == nil {
		return dto.DemandDocumentDTO{}, fmt.Errorf("demand import database is not configured")
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

	var imported domain.DemandDocument
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		profileRepo := infra.NewProfileRepository(tx)
		if input.CustomerProfileID != nil {
			if _, err := profileRepo.FindByID(ctx, *input.CustomerProfileID); err != nil {
				return fmt.Errorf("customer profile %d does not exist", *input.CustomerProfileID)
			}
		}

		effectiveCustomerProfileID := input.CustomerProfileID
		originID := (*uint)(nil)
		resolver := app.NewDemandCustomerResolutionService(profileRepo, infra.NewCustomerProfileOriginRepository(tx))
		// An explicit customerProfileId must not be replaced by order-scoped
		// provisional resolution (or any other strategy).
		if integrationProfile != nil && input.CustomerProfileID == nil {
			resolved, resolveErr := resolver.Resolve(ctx, app.DemandCustomerResolutionInput{
				IntegrationProfileID: integrationProfile.ID,
				IdentityStrategy:     identityStrategy,
				SourceChannel:        effectiveSourceChannel,
				SourceDocumentNo:     input.SourceDocumentNo,
				SourceCustomerRef:    input.SourceCustomerRef,
				DisplayName:          input.SourceCustomerRef,
				ObservedAt:           time.Now().UTC(),
			})
			if resolveErr != nil {
				return fmt.Errorf("customer resolution: %w", resolveErr)
			}
			effectiveCustomerProfileID = resolved.CustomerProfileID
			originID = resolved.OriginID
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
		if err := app.NewDemandIntakeUseCase(infra.NewDemandRepository(tx)).ImportDemand(ctx, &doc, lines); err != nil {
			return err
		}
		if originID != nil {
			if err := resolver.AttachOriginDocument(ctx, *originID, doc.ID); err != nil {
				return err
			}
		}
		imported = doc
		return nil
	})
	if err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(&imported), nil
}

// ImportDemandFromCSV is the leftover single-document CSV entry point. It delegates
// to ImportDemandCSV (reject_all) so mapping, grouping, address, and evidence stay
// aligned with the dual-mode path.
func (c *DemandController) ImportDemandFromCSV(input dto.ImportDemandTemplateInput) (dto.DemandDocumentDTO, error) {
	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: input.IntegrationProfileID,
		DocumentType:         input.DocumentType,
		SourceDocumentNo:     input.SourceDocumentNo,
		SourceCustomerRef:    input.SourceCustomerRef,
		ImportMode:           "reject_all",
		Rows:                 input.Rows,
	})
	if err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	if result.Document == nil {
		if len(result.Errors) > 0 {
			return dto.DemandDocumentDTO{}, fmt.Errorf("demand csv import rejected: %s", result.Errors[0].Reason)
		}
		return dto.DemandDocumentDTO{}, fmt.Errorf("demand csv import produced no document")
	}
	return *result.Document, nil
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

// requestedDemandDocumentType maps a manual Kind (or an already-resolved
// import_* documentType) into the documentType passed to
// ResolveDemandImportDocumentType. Empty stays empty; Resolve then fails
// because documentType must be explicit (not inferred from profile.DemandKind).
func requestedDemandDocumentType(kind string) string {
	switch strings.TrimSpace(kind) {
	case "import_entitlement", string(domain.DemandKindMembershipEntitlement):
		return "import_entitlement"
	case "import_sales_order", string(domain.DemandKindRetailOrder):
		return "import_sales_order"
	default:
		return strings.TrimSpace(kind)
	}
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
