package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ---- DemandIntake ----

type demandIntakeUseCase struct {
	demandRepo domain.DemandDocumentRepository
}

func NewDemandIntakeUseCase(demandRepo domain.DemandDocumentRepository) DemandIntakeUseCase {
	return &demandIntakeUseCase{demandRepo: demandRepo}
}

func (uc *demandIntakeUseCase) ImportDemand(ctx context.Context, doc *domain.DemandDocument, lines []*domain.DemandLine) error {
	// NOTE: This operation is not fully transactional. DemandDocumentRepository has
	// no transaction API, so this usecase cannot begin or join a DB transaction
	// itself (and must not take a *gorm.DB). ImportDemandDocument in
	// controller_demand.go already wraps this call in gorm.DB.Transaction, which
	// provides atomic rollback for document+lines. Other callers that need
	// atomicity should wrap similarly. If a line creation fails after the document
	// is persisted outside a transaction, an orphaned document may remain because
	// the repository does not expose a Delete method.
	now := time.Now()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	if doc.SourceDocumentNo != "" && doc.IntegrationProfileID != nil {
		existing, err := uc.demandRepo.List(ctx)
		if err != nil {
			return err
		}
		for i := range existing {
			other := &existing[i]
			if other.IntegrationProfileID == nil {
				continue
			}
			if *other.IntegrationProfileID == *doc.IntegrationProfileID && other.SourceDocumentNo == doc.SourceDocumentNo {
				return fmt.Errorf("duplicate demand document: integration profile %d already has source_document_no %q", *doc.IntegrationProfileID, doc.SourceDocumentNo)
			}
		}
	}

	if err := uc.demandRepo.Create(ctx, doc); err != nil {
		return err
	}

	for _, line := range lines {
		if line == nil {
			continue
		}
		line.DemandDocumentID = doc.ID
		if line.CreatedAt.IsZero() {
			line.CreatedAt = now
		}
		line.UpdatedAt = now
		if err := uc.demandRepo.CreateLine(ctx, line); err != nil {
			return err
		}
	}
	return nil
}

// ---- Wave ----

type waveUseCase struct {
	waveRepo       domain.WaveRepository
	demandRepo     domain.DemandDocumentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
}

func NewWaveUseCase(waveRepo domain.WaveRepository, demandRepo domain.DemandDocumentRepository, assignmentRepo domain.WaveDemandAssignmentRepository) WaveUseCase {
	return &waveUseCase{waveRepo: waveRepo, demandRepo: demandRepo, assignmentRepo: assignmentRepo}
}

func (uc *waveUseCase) CreateWave(ctx context.Context, wave *domain.Wave) error {
	// generate WaveNo (WAVE-YYYYMMDD-NNN), set defaults, persist.
	datePrefix := time.Now().Format("20060102")
	prefix := "WAVE-" + datePrefix + "-"

	if wave.LifecycleStage == "" {
		wave.LifecycleStage = "intake"
	}
	now := time.Now()
	if wave.CreatedAt.IsZero() {
		wave.CreatedAt = now
	}
	wave.UpdatedAt = now

	// The WaveNo column has a UNIQUE index. Counting existing rows then inserting
	// is a read-then-write race: two concurrent creates can derive the same
	// number. We treat the unique index as the source of truth and retry on a
	// collision (recomputing the count) so the caller gets a unique WaveNo
	// instead of a hard failure.
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		count, err := uc.waveRepo.CountByDatePrefix(ctx, prefix)
		if err != nil {
			return err
		}
		wave.WaveNo = fmt.Sprintf("WAVE-%s-%03d", datePrefix, count+1)

		err = uc.waveRepo.Create(ctx, wave)
		if err == nil {
			return nil
		}
		if !isUniqueConstraintErr(err) {
			return err
		}
	}
	return fmt.Errorf("could not allocate a unique WaveNo for %s after %d attempts", prefix, maxAttempts)
}

// isUniqueConstraintErr reports whether err is a SQLite UNIQUE constraint
// violation. GORM's typed ErrDuplicatedKey requires error translation to be
// enabled, so we match on the driver message, consistent with the infra layer.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint")
}

func (uc *waveUseCase) ListWaves(ctx context.Context) ([]domain.Wave, error) {
	return uc.waveRepo.List(ctx)
}

func (uc *waveUseCase) ListWavesPaginated(ctx context.Context, offset, limit int) ([]domain.Wave, int64, error) {
	return uc.waveRepo.ListPaginated(ctx, offset, limit)
}

func (uc *waveUseCase) GetWave(ctx context.Context, id uint) (*domain.Wave, error) {
	return uc.waveRepo.FindByID(ctx, id)
}

func (uc *waveUseCase) GenerateParticipants(ctx context.Context, waveID uint) (int, error) {
	// Get demand documents assigned to this wave
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(ctx, waveID)
	if err != nil {
		return 0, err
	}

	// Get existing participants for idempotency check
	existingSnaps, err := uc.waveRepo.ListParticipantsByWave(ctx, waveID)
	if err != nil {
		return 0, err
	}
	existingProfiles := make(map[uint]bool, len(existingSnaps))
	for _, snap := range existingSnaps {
		existingProfiles[snap.CustomerProfileID] = true
	}

	// Group assigned documents by CustomerProfileID (skip nil profile).
	grouped := make(map[uint][]domain.DemandDocument)
	profileOrder := make([]uint, 0)
	skippedNoProfile := 0
	for docIdx := range docs {
		doc := docs[docIdx]
		if doc.CustomerProfileID == nil {
			skippedNoProfile++
			continue
		}
		profileID := *doc.CustomerProfileID
		if _, seen := grouped[profileID]; !seen {
			profileOrder = append(profileOrder, profileID)
		}
		grouped[profileID] = append(grouped[profileID], doc)
	}

	count := 0
	for _, profileID := range profileOrder {
		if existingProfiles[profileID] {
			continue
		}
		profileDocs := grouped[profileID]

		hasNonRetail := false
		var firstNonRetail, firstRetail *domain.DemandDocument
		docIDs := make([]uint, 0, len(profileDocs))
		for i := range profileDocs {
			d := &profileDocs[i]
			docIDs = append(docIDs, d.ID)
			if d.Kind != "retail_order" {
				hasNonRetail = true
				if firstNonRetail == nil {
					firstNonRetail = d
				}
			} else if firstRetail == nil {
				firstRetail = d
			}
		}

		var giftFromMembership, giftFromAny string
		sawMembershipAccepted, sawAnyAccepted := false, false
		for i := range profileDocs {
			d := &profileDocs[i]
			lines, lineErr := uc.demandRepo.ListLinesByDocument(ctx, d.ID)
			if lineErr != nil {
				return count, lineErr
			}
			for lineIdx := range lines {
				if lines[lineIdx].RoutingDisposition != "accepted" {
					continue
				}
				if !sawAnyAccepted {
					giftFromAny = lines[lineIdx].GiftLevelSnapshot
					sawAnyAccepted = true
				}
				if d.Kind != "retail_order" && !sawMembershipAccepted {
					giftFromMembership = lines[lineIdx].GiftLevelSnapshot
					sawMembershipAccepted = true
				}
				break
			}
		}

		// Only generate snapshot if there's at least one accepted line on any doc
		if !sawAnyAccepted {
			continue
		}

		giftLevel := giftFromAny
		if sawMembershipAccepted {
			giftLevel = giftFromMembership
		}

		snapshotType := "buyer"
		if hasNonRetail {
			snapshotType = "member"
		}

		identityDoc := firstRetail
		if firstNonRetail != nil {
			identityDoc = firstNonRetail
		}
		var identityPlatform, identityValue string
		if identityDoc != nil {
			identityPlatform = identityDoc.SourceChannel
			identityValue = identityDoc.SourceCustomerRef
		}

		sort.Slice(docIDs, func(i, j int) bool { return docIDs[i] < docIDs[j] })
		refParts := make([]string, len(docIDs))
		for i, id := range docIDs {
			refParts[i] = fmt.Sprintf("%d", id)
		}

		snap := domain.WaveParticipantSnapshot{
			WaveID:             waveID,
			CustomerProfileID:  profileID,
			SnapshotType:       snapshotType,
			IdentityPlatform:   identityPlatform,
			IdentityValue:      identityValue,
			DisplayName:        "",
			GiftLevel:          giftLevel,
			SourceDocumentRefs: strings.Join(refParts, ","),
			SourceProfileRefs:  "",
			CreatedAt:          time.Now(),
		}

		if err := uc.waveRepo.AddParticipant(ctx, &snap); err != nil {
			return count, err
		}

		count++
	}

	// If documents were assigned but all lacked CustomerProfileID, signal explicitly
	if count == 0 && skippedNoProfile > 0 {
		return 0, fmt.Errorf("all %d assigned demand documents lack a CustomerProfileID; cannot generate participant snapshots", skippedNoProfile)
	}

	return count, nil
}

// ---- DemandMapping ----

type demandMappingUseCase struct {
	demandRepo     domain.DemandDocumentRepository
	fulfillRepo    domain.FulfillmentLineRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	waveRepo       domain.WaveRepository
	productRepo    domain.ProductRepository
	addressRepo    domain.CustomerAddressRepository
}

func NewDemandMappingUseCase(demandRepo domain.DemandDocumentRepository, fulfillRepo domain.FulfillmentLineRepository, assignmentRepo domain.WaveDemandAssignmentRepository, waveRepo domain.WaveRepository, productRepo domain.ProductRepository, addressRepo domain.CustomerAddressRepository) DemandMappingUseCase {
	return &demandMappingUseCase{demandRepo: demandRepo, fulfillRepo: fulfillRepo, assignmentRepo: assignmentRepo, waveRepo: waveRepo, productRepo: productRepo, addressRepo: addressRepo}
}

// isEligibleForFulfillment checks the unified execution-eligibility rule:
// routing_disposition = accepted AND recipient_input_state in (ready, not_required).
func isEligibleForFulfillment(dl *domain.DemandLine) bool {
	if dl.RoutingDisposition != "accepted" {
		return false
	}
	return dl.RecipientInputState == "ready" || dl.RecipientInputState == "not_required"
}

func (uc *demandMappingUseCase) MapDemandToFulfillment(ctx context.Context, waveID uint) (*dto.DemandMappingResult, error) {
	// NOTE: This method deletes then re-creates fulfillment lines. The controller
	// (MapDemandLines) already wraps this call in gorm.DB.Transaction for atomicity.
	// If called outside a transaction, a failure mid-loop leaves partial data.
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(ctx, waveID)
	if err != nil {
		return nil, err
	}

	// Build profileID → snapshotID lookup for participant association
	var profileToSnapshot map[uint]uint
	if uc.waveRepo != nil {
		participants, err := uc.waveRepo.ListParticipantsByWave(ctx, waveID)
		if err != nil {
			return nil, err
		}
		profileToSnapshot = make(map[uint]uint, len(participants))
		for i := range participants {
			profileToSnapshot[participants[i].CustomerProfileID] = participants[i].ID
		}
	}

	// Build FK → wave-scoped ProductID lookup for demand-line product mapping
	productMasterToWaveProduct := make(map[uint]uint)
	if uc.productRepo != nil {
		waveProducts, err := uc.productRepo.ListByWave(ctx, waveID)
		if err != nil {
			return nil, err
		}
		for _, wp := range waveProducts {
			if wp.ProductMasterID != nil {
				productMasterToWaveProduct[*wp.ProductMasterID] = wp.ID
			}
		}
	}

	// Pre-check: every retail_order with eligible lines must be associable to a snapshot.
	var missingProfileDocs []uint
	var missingSnapshotProfiles []uint
	for docIdx := range docs {
		doc := &docs[docIdx]
		if doc.Kind != "retail_order" {
			continue
		}
		hasEligible, err := uc.docHasEligibleLines(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		if !hasEligible {
			continue
		}
		if doc.CustomerProfileID == nil {
			missingProfileDocs = append(missingProfileDocs, doc.ID)
			continue
		}
		if profileToSnapshot != nil {
			if _, ok := profileToSnapshot[*doc.CustomerProfileID]; !ok {
				missingSnapshotProfiles = append(missingSnapshotProfiles, *doc.CustomerProfileID)
			}
		}
	}
	if len(missingProfileDocs) > 0 {
		return nil, fmt.Errorf("retail demand documents %v have eligible lines but no CustomerProfileID; cannot generate fulfillment lines", missingProfileDocs)
	}
	if len(missingSnapshotProfiles) > 0 {
		return nil, fmt.Errorf("no participant snapshots found for customer profiles %v; run GenerateParticipants first", missingSnapshotProfiles)
	}

	// Pre-check passed — safe to rebuild
	if err := uc.fulfillRepo.DeleteByWaveAndGeneratedBy(ctx, waveID, "allocation_demand_driven"); err != nil {
		return nil, err
	}

	now := time.Now()
	// Pre-allocate empty (non-nil) slices so DemandMappingResult JSON encodes
	// createdLines/blockedLines as [] rather than null when nothing matched.
	createdLines := make([]domain.FulfillmentLine, 0)
	blockedLines := make([]dto.DemandMappingBlockedLine, 0)

	for docIdx := range docs {
		doc := &docs[docIdx]
		if doc.Kind != "retail_order" || doc.CustomerProfileID == nil {
			continue
		}

		snapID := profileToSnapshot[*doc.CustomerProfileID]

		demandLines, err := uc.demandRepo.ListLinesByDocument(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		for lineIdx := range demandLines {
			dl := &demandLines[lineIdx]
			if !isEligibleForFulfillment(dl) {
				continue
			}

			// Resolve ProductID via ProductMasterID → wave-scoped Product lookup.
			// Lines that require a product reference but cannot resolve it are
			// blocked — they are NOT silently admitted with ProductID=nil.
			var productID *uint
			if dl.ProductMasterID != nil {
				if waveProductID, ok := productMasterToWaveProduct[*dl.ProductMasterID]; ok {
					pid := waveProductID
					productID = &pid
				} else {
					blockedLines = append(blockedLines, dto.DemandMappingBlockedLine{
						DemandLineID:    dl.ID,
						DemandLineTitle: dl.ExternalTitle,
						Reason:          "wave_product_missing",
					})
					continue
				}
			}

			// Address readiness gate: when addressRepo is wired, block lines whose
			// profile has no valid addresses. A ListByProfile error fails the mapping.
			if uc.addressRepo != nil && doc.CustomerProfileID != nil {
				addrs, addrErr := uc.addressRepo.ListByProfile(ctx, *doc.CustomerProfileID)
				if addrErr != nil {
					return nil, addrErr
				}
				if len(addrs) == 0 {
					blockedLines = append(blockedLines, dto.DemandMappingBlockedLine{
						DemandLineID:    dl.ID,
						DemandLineTitle: dl.ExternalTitle,
						Reason:          "address_unavailable",
					})
					continue
				}
			}
			docID := doc.ID
			lineID := dl.ID
			fl := domain.FulfillmentLine{
				WaveID:                    waveID,
				DemandDocumentID:          &docID,
				DemandLineID:              &lineID,
				CustomerProfileID:         doc.CustomerProfileID,
				WaveParticipantSnapshotID: &snapID,
				ProductID:                 productID,
				Quantity:                  dl.RequestedQuantity,
				AllocationState:           "ready",
				AddressState:              "missing",
				SupplierState:             "not_submitted",
				ChannelSyncState:          "not_required",
				LineReason:                "retail_order",
				GeneratedBy:               "allocation_demand_driven",
				CreatedAt:                 now,
				UpdatedAt:                 now,
			}

			if err := uc.fulfillRepo.Create(ctx, &fl); err != nil {
				return nil, err
			}
			createdLines = append(createdLines, fl)
		}
	}

	lineDTOs := make([]dto.FulfillmentLineDTO, len(createdLines))
	for i := range createdLines {
		lineDTOs[i] = domainToFulfillmentLineDTO(&createdLines[i])
	}
	return &dto.DemandMappingResult{
		CreatedLines: lineDTOs,
		BlockedLines: blockedLines,
	}, nil
}

// domainToFulfillmentLineDTO mirrors the controller-level converter.
func domainToFulfillmentLineDTO(fl *domain.FulfillmentLine) dto.FulfillmentLineDTO {
	if fl == nil {
		return dto.FulfillmentLineDTO{}
	}
	return dto.FulfillmentLineDTO{
		ID:                        fl.ID,
		WaveID:                    fl.WaveID,
		CustomerProfileID:         fl.CustomerProfileID,
		WaveParticipantSnapshotID: fl.WaveParticipantSnapshotID,
		ProductID:                 fl.ProductID,
		DemandDocumentID:          fl.DemandDocumentID,
		DemandLineID:              fl.DemandLineID,
		CustomerAddressID:         fl.CustomerAddressID,
		Quantity:                  fl.Quantity,
		AllocationState:           fl.AllocationState,
		AddressState:              fl.AddressState,
		SupplierState:             fl.SupplierState,
		ChannelSyncState:          fl.ChannelSyncState,
		LineReason:                fl.LineReason,
		GeneratedBy:               fl.GeneratedBy,
		ExtraData:                 fl.ExtraData,
		CreatedAt:                 fl.CreatedAt,
		UpdatedAt:                 fl.UpdatedAt,
	}
}

func (uc *demandMappingUseCase) docHasEligibleLines(ctx context.Context, docID uint) (bool, error) {
	demandLines, err := uc.demandRepo.ListLinesByDocument(ctx, docID)
	if err != nil {
		return false, err
	}
	for i := range demandLines {
		if isEligibleForFulfillment(&demandLines[i]) {
			return true, nil
		}
	}
	return false, nil
}

// ---- Export ----

type exportUseCase struct {
	supplierRepo domain.SupplierOrderRepository
	fulfillRepo  domain.FulfillmentLineRepository
	basisStamp   *BasisStampService
	demandRepo   domain.DemandDocumentRepository
	profileRepo  domain.IntegrationProfileRepository
	bindingRepo  domain.ProfileTemplateBindingRepository
	productRepo  domain.ProductRepository
}

func NewExportUseCase(
	supplierRepo domain.SupplierOrderRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	basisStamp *BasisStampService,
	demandRepo domain.DemandDocumentRepository,
	profileRepo domain.IntegrationProfileRepository,
	bindingRepo domain.ProfileTemplateBindingRepository,
	productRepo domain.ProductRepository,
) ExportUseCase {
	return &exportUseCase{
		supplierRepo: supplierRepo,
		fulfillRepo:  fulfillRepo,
		basisStamp:   basisStamp,
		demandRepo:   demandRepo,
		profileRepo:  profileRepo,
		bindingRepo:  bindingRepo,
		productRepo:  productRepo,
	}
}

// ExportSupplierOrder is the compatibility entry point. It auto-selects a
// factory profile only when exactly one valid profile can execute every
// fulfillment line in the wave. Ambiguity is an error; callers that know the
// intended factory should use ExportSupplierOrderForProfile.
func (uc *exportUseCase) ExportSupplierOrder(ctx context.Context, waveID uint) ([]*domain.SupplierOrder, error) {
	if waveID == 0 {
		return nil, fmt.Errorf("waveID is required")
	}
	if uc.profileRepo == nil || uc.bindingRepo == nil || uc.productRepo == nil || uc.fulfillRepo == nil {
		return nil, fmt.Errorf("auto-select factory profile: profile, binding, product, and fulfillment repositories are required")
	}
	fulfillLines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("auto-select factory profile: list wave %d fulfillment lines: %w", waveID, err)
	}
	if len(fulfillLines) == 0 {
		return nil, fmt.Errorf("wave %d has no fulfillment lines to export", waveID)
	}
	profiles, err := uc.profileRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("auto-select factory profile: list profiles: %w", err)
	}

	type candidate struct {
		profile *domain.IntegrationProfile
		binding *domain.IntegrationProfileTemplateBinding
	}
	var candidates []candidate
	for i := range profiles {
		profile := &profiles[i]
		if ValidateProfileDocumentType(profile, "export_supplier_order") != nil || strings.TrimSpace(profile.FactorySupplierPlatform) == "" {
			continue
		}
		binding, bindErr := uc.bindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, "export_supplier_order")
		if bindErr != nil || validateSupplierOrderBinding(profile.ID, binding) != nil {
			continue
		}
		selected, selectErr := uc.selectFactoryFulfillmentLines(ctx, profile, fulfillLines)
		if selectErr != nil || len(selected) != len(fulfillLines) {
			continue
		}
		candidates = append(candidates, candidate{profile: profile, binding: binding})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("wave %d has no uniquely executable factory profile; select a factory profile explicitly and verify product platforms and export_supplier_order binding", waveID)
	}
	if len(candidates) > 1 {
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].profile.ID < candidates[j].profile.ID })
		parts := make([]string, len(candidates))
		for i := range candidates {
			parts[i] = fmt.Sprintf("%d(%s)", candidates[i].profile.ID, candidates[i].profile.ProfileKey)
		}
		return nil, fmt.Errorf("wave %d factory profile is ambiguous (%s); call ExportSupplierOrderForProfile with an explicit profile ID", waveID, strings.Join(parts, ", "))
	}
	return uc.exportSupplierOrderWithProfile(ctx, waveID, candidates[0].profile, candidates[0].binding, fulfillLines)
}

// ExportSupplierOrderForProfile explicitly selects the factory execution
// profile and records that selection on the resulting SupplierOrder.
func (uc *exportUseCase) ExportSupplierOrderForProfile(ctx context.Context, waveID, factoryProfileID uint) ([]*domain.SupplierOrder, error) {
	if waveID == 0 {
		return nil, fmt.Errorf("waveID is required")
	}
	if factoryProfileID == 0 {
		return nil, fmt.Errorf("factoryProfileID is required")
	}
	if uc.profileRepo == nil || uc.bindingRepo == nil || uc.productRepo == nil || uc.fulfillRepo == nil {
		return nil, fmt.Errorf("export supplier order: profile, binding, product, and fulfillment repositories are required")
	}
	profile, err := uc.profileRepo.FindByID(ctx, factoryProfileID)
	if err != nil {
		return nil, fmt.Errorf("factory profile %d not found: %w", factoryProfileID, err)
	}
	if profile == nil {
		return nil, fmt.Errorf("factory profile %d not found", factoryProfileID)
	}
	if err := ValidateProfileDocumentType(profile, "export_supplier_order"); err != nil {
		return nil, fmt.Errorf("factory profile %d cannot export supplier orders: %w", factoryProfileID, err)
	}
	if strings.TrimSpace(profile.FactorySupplierPlatform) == "" {
		return nil, fmt.Errorf("factory profile %d has no factorySupplierPlatform", factoryProfileID)
	}
	binding, err := uc.bindingRepo.FindDefaultByProfileAndType(ctx, factoryProfileID, "export_supplier_order")
	if err != nil {
		return nil, fmt.Errorf("factory profile %d export_supplier_order binding lookup: %w", factoryProfileID, err)
	}
	if err := validateSupplierOrderBinding(factoryProfileID, binding); err != nil {
		return nil, err
	}
	fulfillLines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return nil, fmt.Errorf("list wave %d fulfillment lines: %w", waveID, err)
	}
	if len(fulfillLines) == 0 {
		return nil, fmt.Errorf("wave %d has no fulfillment lines to export", waveID)
	}
	selectedLines, err := uc.selectFactoryFulfillmentLines(ctx, profile, fulfillLines)
	if err != nil {
		return nil, err
	}
	if len(selectedLines) == 0 {
		return nil, fmt.Errorf("wave %d has no fulfillment lines for factory profile %d platform %q", waveID, profile.ID, profile.FactorySupplierPlatform)
	}
	return uc.exportSupplierOrderWithProfile(ctx, waveID, profile, binding, selectedLines)
}

func validateSupplierOrderBinding(profileID uint, binding *domain.IntegrationProfileTemplateBinding) error {
	if binding == nil {
		return fmt.Errorf("factory profile %d has no default export_supplier_order binding", profileID)
	}
	if binding.IntegrationProfileID != profileID || binding.DocumentType != "export_supplier_order" || binding.TemplateID == 0 || !binding.IsDefault {
		return fmt.Errorf("factory profile %d has an invalid default export_supplier_order binding", profileID)
	}
	return nil
}

func (uc *exportUseCase) selectFactoryFulfillmentLines(ctx context.Context, profile *domain.IntegrationProfile, lines []domain.FulfillmentLine) ([]domain.FulfillmentLine, error) {
	productCache := make(map[uint]*domain.Product)
	selected := make([]domain.FulfillmentLine, 0, len(lines))
	for i := range lines {
		fl := &lines[i]
		if fl.ProductID == nil {
			return nil, fmt.Errorf("fulfillment line %d has no product; cannot route to factory profile %d", fl.ID, profile.ID)
		}
		product, ok := productCache[*fl.ProductID]
		if !ok {
			found, err := uc.productRepo.FindByID(ctx, *fl.ProductID)
			if err != nil {
				return nil, fmt.Errorf("fulfillment line %d product %d lookup failed: %w", fl.ID, *fl.ProductID, err)
			}
			if found == nil {
				return nil, fmt.Errorf("fulfillment line %d product %d not found", fl.ID, *fl.ProductID)
			}
			product = found
			productCache[*fl.ProductID] = found
		}
		if product.SupplierPlatform != profile.FactorySupplierPlatform {
			continue
		}
		if strings.TrimSpace(product.FactorySKU) == "" {
			return nil, fmt.Errorf("fulfillment line %d product %d has no factory SKU", fl.ID, product.ID)
		}
		selected = append(selected, *fl)
	}
	return selected, nil
}

func (uc *exportUseCase) exportSupplierOrderWithProfile(
	ctx context.Context,
	waveID uint,
	profile *domain.IntegrationProfile,
	binding *domain.IntegrationProfileTemplateBinding,
	fulfillLines []domain.FulfillmentLine,
) ([]*domain.SupplierOrder, error) {
	// Rebuild only after all routing checks succeed, so an invalid profile
	// selection cannot delete a valid existing draft.
	type factoryScopedDraftRebuilder interface {
		DeleteDraftsByWaveAndFactoryProfile(context.Context, uint, uint) error
	}
	var rebuildErr error
	if scoped, ok := uc.supplierRepo.(factoryScopedDraftRebuilder); ok {
		rebuildErr = scoped.DeleteDraftsByWaveAndFactoryProfile(ctx, waveID, profile.ID)
	} else {
		// Test doubles and legacy repository implementations retain the original
		// whole-wave rebuild behavior. Production repositories are profile-scoped.
		rebuildErr = uc.supplierRepo.DeleteDraftsByWave(ctx, waveID)
	}
	if rebuildErr != nil {
		return nil, rebuildErr
	}

	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		var err error
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(ctx, waveID)
		if err != nil {
			return nil, fmt.Errorf("resolve basis for supplier order: %w", err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	now := time.Now()
	profileID := profile.ID
	submissionMode := "csv"
	if profile.SupportsAPIExport {
		submissionMode = "api"
	}
	order := &domain.SupplierOrder{
		WaveID:                      waveID,
		FactoryIntegrationProfileID: &profileID,
		SupplierPlatform:            profile.FactorySupplierPlatform,
		TemplateID:                  fmt.Sprintf("%d", binding.TemplateID),
		BatchNo:                     fmt.Sprintf("WAVE-%d-FACTORY-%d", waveID, profile.ID),
		Status:                      "draft",
		SubmissionMode:              submissionMode,
		BasisHistoryNodeID:          basisNodeID,
		BasisProjectionHash:         basisHash,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
	productCache := make(map[uint]*domain.Product)
	lines := make([]*domain.SupplierOrderLine, len(fulfillLines))
	for i := range fulfillLines {
		fl := &fulfillLines[i]
		sku, skuErr := resolveSupplierSKU(ctx, fl, uc.productRepo, productCache)
		if skuErr != nil {
			return nil, fmt.Errorf("fulfillment line %d: %w", fl.ID, skuErr)
		}
		lines[i] = &domain.SupplierOrderLine{
			FulfillmentLineID: fl.ID,
			SupplierLineNo:    i + 1,
			SupplierSKU:       sku,
			SubmittedQuantity: fl.Quantity,
			Status:            "draft",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}
	var pin *domain.BasisPinParam
	if pinNodeID != 0 {
		pin = &domain.BasisPinParam{HistoryNodeID: pinNodeID, PinKind: "supplier_order_basis", RefType: "supplier_order"}
	}
	if err := uc.supplierRepo.AtomicCreateSupplierOrder(ctx, order, lines, pin); err != nil {
		return nil, err
	}
	if err := uc.projectSupplierStateFromOrder(ctx, order, lines); err != nil {
		return nil, err
	}
	return []*domain.SupplierOrder{order}, nil
}

// resolveSupplierSKU fills SupplierOrderLine.SupplierSKU from the wave Product
// snapshot's FactorySKU. An empty SKU is never returned; callers must treat
// lookup and missing-SKU failures as errors.
func resolveSupplierSKU(
	ctx context.Context,
	fl *domain.FulfillmentLine,
	productRepo domain.ProductRepository,
	cache map[uint]*domain.Product,
) (string, error) {
	if fl == nil || fl.ProductID == nil {
		return "", fmt.Errorf("fulfillment line has no product")
	}
	if productRepo == nil {
		return "", fmt.Errorf("product repository is required to resolve supplier SKU")
	}
	pid := *fl.ProductID
	p, ok := cache[pid]
	if !ok {
		found, err := productRepo.FindByID(ctx, pid)
		if err != nil {
			return "", fmt.Errorf("product %d lookup failed: %w", pid, err)
		}
		if found == nil {
			return "", fmt.Errorf("product %d not found", pid)
		}
		p = found
		if cache != nil {
			cache[pid] = found
		}
	}
	if p == nil {
		return "", fmt.Errorf("product %d not found", pid)
	}
	if strings.TrimSpace(p.FactorySKU) == "" {
		return "", fmt.Errorf("product %d has no factory SKU", pid)
	}
	return p.FactorySKU, nil
}

// projectSupplierStateFromOrder maps a SupplierOrder.Status to the corresponding
// SupplierState and bulk-updates the referenced FulfillmentLines.
func (uc *exportUseCase) projectSupplierStateFromOrder(ctx context.Context, order *domain.SupplierOrder, lines []*domain.SupplierOrderLine) error {
	projected := supplierOrderStatusToState(order.Status)
	if projected == "" {
		return nil
	}
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(lines))
	for _, l := range lines {
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:            l.FulfillmentLineID,
			SupplierState: projected,
		})
	}
	if len(updates) > 0 {
		if err := uc.fulfillRepo.BulkUpdateStates(ctx, updates); err != nil {
			return err
		}
	}
	return nil
}

// supplierOrderStatusToState maps SupplierOrder.Status → FulfillmentLine.SupplierState.
func supplierOrderStatusToState(status string) string {
	switch status {
	case "draft":
		return "not_submitted"
	case "submitted":
		return "submitted"
	case "accepted":
		return "accepted"
	case "partially_shipped":
		return "partially_shipped"
	case "shipped":
		return "shipped"
	case "canceled":
		return "canceled"
	default:
		return ""
	}
}
