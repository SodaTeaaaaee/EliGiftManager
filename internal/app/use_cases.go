package app

import (
	"fmt"
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

func (uc *demandIntakeUseCase) ImportDemand(doc *domain.DemandDocument, lines []*domain.DemandLine) error {
	// [V2-STUB] demand-driven: create DemandDocument then all DemandLines
	now := time.Now().Format(time.RFC3339)
	if doc.CreatedAt == "" {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	if err := uc.demandRepo.Create(doc); err != nil {
		return err
	}

	for _, line := range lines {
		if line == nil {
			continue
		}
		line.DemandDocumentID = doc.ID
		if line.CreatedAt == "" {
			line.CreatedAt = now
		}
		line.UpdatedAt = now
		if err := uc.demandRepo.CreateLine(line); err != nil {
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

func (uc *waveUseCase) CreateWave(wave *domain.Wave) error {
	// [V2-STUB] generate WaveNo (WAVE-YYYYMMDD-NNN), set defaults, persist
	datePrefix := time.Now().Format("20060102")
	existing, err := uc.waveRepo.List()
	if err != nil {
		return err
	}

	count := 0
	prefix := "WAVE-" + datePrefix + "-"
	for _, w := range existing {
		if strings.HasPrefix(w.WaveNo, prefix) {
			count++
		}
	}
	wave.WaveNo = fmt.Sprintf("WAVE-%s-%03d", datePrefix, count+1)

	if wave.LifecycleStage == "" {
		wave.LifecycleStage = "intake"
	}

	now := time.Now().Format(time.RFC3339)
	if wave.CreatedAt == "" {
		wave.CreatedAt = now
	}
	wave.UpdatedAt = now

	return uc.waveRepo.Create(wave)
}

func (uc *waveUseCase) ListWaves() ([]domain.Wave, error) {
	return uc.waveRepo.List()
}

func (uc *waveUseCase) GetWave(id uint) (*domain.Wave, error) {
	return uc.waveRepo.FindByID(id)
}

func (uc *waveUseCase) GenerateParticipants(waveID uint) (int, error) {
	// Get demand documents assigned to this wave
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return 0, err
	}

	// Get existing participants for idempotency check
	existingSnaps, err := uc.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return 0, err
	}
	existingProfiles := make(map[uint]bool, len(existingSnaps))
	for _, snap := range existingSnaps {
		existingProfiles[snap.CustomerProfileID] = true
	}

	// Track profiles we generate in this run (dedup within batch)
	generatedProfiles := make(map[uint]bool)
	count := 0
	skippedNoProfile := 0

	for docIdx := range docs {
		doc := &docs[docIdx]

		// Documents without a CustomerProfileID cannot generate participant snapshots
		if doc.CustomerProfileID == nil {
			skippedNoProfile++
			continue
		}
		profileID := *doc.CustomerProfileID

		// Skip if already exists or already generated in this batch
		if existingProfiles[profileID] || generatedProfiles[profileID] {
			continue
		}

		// Get demand lines for this document
		lines, err := uc.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return count, err
		}

		// Find first accepted line to extract GiftLevelSnapshot
		var giftLevel string
		hasAccepted := false
		for lineIdx := range lines {
			if lines[lineIdx].RoutingDisposition == "accepted" {
				giftLevel = lines[lineIdx].GiftLevelSnapshot
				hasAccepted = true
				break
			}
		}

		// Only generate snapshot if there's at least one accepted line
		if !hasAccepted {
			continue
		}

		// Determine snapshot type based on demand document kind
		snapshotType := "member"
		if doc.Kind == "retail_order" {
			snapshotType = "buyer"
		}

		snap := domain.WaveParticipantSnapshot{
			WaveID:             waveID,
			CustomerProfileID:  profileID,
			SnapshotType:       snapshotType,
			IdentityPlatform:   doc.SourceChannel,
			IdentityValue:      doc.SourceCustomerRef,
			DisplayName:        "",
			GiftLevel:          giftLevel,
			SourceDocumentRefs: fmt.Sprintf("%d", doc.ID),
			SourceProfileRefs:  "",
			CreatedAt:          time.Now().Format(time.RFC3339),
		}

		if err := uc.waveRepo.AddParticipant(&snap); err != nil {
			return count, err
		}

		generatedProfiles[profileID] = true
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
}

func NewDemandMappingUseCase(demandRepo domain.DemandDocumentRepository, fulfillRepo domain.FulfillmentLineRepository, assignmentRepo domain.WaveDemandAssignmentRepository, waveRepo domain.WaveRepository, productRepo domain.ProductRepository) DemandMappingUseCase {
	return &demandMappingUseCase{demandRepo: demandRepo, fulfillRepo: fulfillRepo, assignmentRepo: assignmentRepo, waveRepo: waveRepo, productRepo: productRepo}
}

// isEligibleForFulfillment checks the unified execution-eligibility rule:
// routing_disposition = accepted AND recipient_input_state in (ready, not_required).
func isEligibleForFulfillment(dl *domain.DemandLine) bool {
	if dl.RoutingDisposition != "accepted" {
		return false
	}
	return dl.RecipientInputState == "ready" || dl.RecipientInputState == "not_required"
}

func (uc *demandMappingUseCase) MapDemandToFulfillment(waveID uint) (*dto.DemandMappingResult, error) {
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return nil, err
	}

	// Build profileID → snapshotID lookup for participant association
	var profileToSnapshot map[uint]uint
	if uc.waveRepo != nil {
		participants, err := uc.waveRepo.ListParticipantsByWave(waveID)
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
		waveProducts, err := uc.productRepo.ListByWave(waveID)
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
		hasEligible, err := uc.docHasEligibleLines(doc.ID)
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
	if err := uc.fulfillRepo.DeleteByWaveAndGeneratedBy(waveID, "allocation_demand_driven"); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	var createdLines []domain.FulfillmentLine
	var blockedLines []dto.DemandMappingBlockedLine

	for docIdx := range docs {
		doc := &docs[docIdx]
		if doc.Kind != "retail_order" || doc.CustomerProfileID == nil {
			continue
		}

		snapID := profileToSnapshot[*doc.CustomerProfileID]

		demandLines, err := uc.demandRepo.ListLinesByDocument(doc.ID)
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

			if err := uc.fulfillRepo.Create(&fl); err != nil {
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

func (uc *demandMappingUseCase) docHasEligibleLines(docID uint) (bool, error) {
	demandLines, err := uc.demandRepo.ListLinesByDocument(docID)
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
}

func NewExportUseCase(supplierRepo domain.SupplierOrderRepository, fulfillRepo domain.FulfillmentLineRepository, basisStamp *BasisStampService) ExportUseCase {
	return &exportUseCase{supplierRepo: supplierRepo, fulfillRepo: fulfillRepo, basisStamp: basisStamp}
}

func (uc *exportUseCase) ExportSupplierOrder(waveID uint) (*domain.SupplierOrder, error) {
	// Delete only existing draft orders for this wave (rebuild pattern for idempotency)
	if err := uc.supplierRepo.DeleteDraftsByWave(waveID); err != nil {
		return nil, err
	}

	// Resolve basis stamp before persisting
	var basisNodeID, basisHash string
	var pinNodeID uint
	if uc.basisStamp != nil {
		var err error
		basisNodeID, basisHash, err = uc.basisStamp.ResolveBasis(waveID)
		if err != nil {
			return nil, fmt.Errorf("resolve basis for supplier order: %w", err)
		}
		if basisNodeID != "" {
			fmt.Sscanf(basisNodeID, "%d", &pinNodeID)
		}
	}

	// [V2-STUB] aggregate all FulfillmentLines for the wave into a SupplierOrder with lines
	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	order := &domain.SupplierOrder{
		WaveID:              waveID,
		Status:              "draft",
		SubmissionMode:      "csv",
		BasisHistoryNodeID:  basisNodeID,
		BasisProjectionHash: basisHash,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	lines := make([]*domain.SupplierOrderLine, len(fulfillLines))
	for i := range fulfillLines {
		fl := &fulfillLines[i]
		lines[i] = &domain.SupplierOrderLine{
			FulfillmentLineID: fl.ID,
			SubmittedQuantity: fl.Quantity,
			Status:            "draft",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	var pin *domain.BasisPinParam
	if pinNodeID != 0 {
		pin = &domain.BasisPinParam{
			HistoryNodeID: pinNodeID,
			PinKind:       "supplier_order_basis",
			RefType:       "supplier_order",
		}
	}

	if err := uc.supplierRepo.AtomicCreateSupplierOrder(order, lines, pin); err != nil {
		return nil, err
	}

	// Project supplier_state from the newly created supplier order onto each
	// referenced FulfillmentLine. This is a fact-driven write-path projection.
	uc.projectSupplierStateFromOrder(order, lines)

	return order, nil
}

// projectSupplierStateFromOrder maps a SupplierOrder.Status to the corresponding
// SupplierState and bulk-updates the referenced FulfillmentLines.
func (uc *exportUseCase) projectSupplierStateFromOrder(order *domain.SupplierOrder, lines []*domain.SupplierOrderLine) {
	projected := supplierOrderStatusToState(order.Status)
	if projected == "" {
		return
	}
	updates := make([]domain.FulfillmentLineStateUpdate, 0, len(lines))
	for _, l := range lines {
		updates = append(updates, domain.FulfillmentLineStateUpdate{
			ID:            l.FulfillmentLineID,
			SupplierState: projected,
		})
	}
	if len(updates) > 0 {
		_ = uc.fulfillRepo.BulkUpdateStates(updates)
	}
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
