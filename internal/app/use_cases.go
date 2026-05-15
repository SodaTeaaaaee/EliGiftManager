package app

import (
	"fmt"
	"strings"
	"time"

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

// ---- Allocation ----

type allocationUseCase struct {
	demandRepo     domain.DemandDocumentRepository
	ruleRepo       domain.AllocationPolicyRuleRepository
	fulfillRepo    domain.FulfillmentLineRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
}

func NewAllocationUseCase(demandRepo domain.DemandDocumentRepository, ruleRepo domain.AllocationPolicyRuleRepository, fulfillRepo domain.FulfillmentLineRepository, assignmentRepo domain.WaveDemandAssignmentRepository) AllocationUseCase {
	return &allocationUseCase{demandRepo: demandRepo, ruleRepo: ruleRepo, fulfillRepo: fulfillRepo, assignmentRepo: assignmentRepo}
}

func (uc *allocationUseCase) ApplyRules(waveID uint) ([]domain.FulfillmentLine, error) {
	// Delete existing allocation_demand_driven fulfillment lines for this wave (rebuild pattern for idempotency)
	if err := uc.fulfillRepo.DeleteByWaveAndGeneratedBy(waveID, "allocation_demand_driven"); err != nil {
		return nil, err
	}

	// Use assigned demands only (wave-demand linkage)
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(waveID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	var lines []domain.FulfillmentLine

	for docIdx := range docs {
		doc := &docs[docIdx]
		demandLines, err := uc.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return nil, err
		}
		for lineIdx := range demandLines {
			dl := &demandLines[lineIdx]
			if dl.RoutingDisposition != "accepted" {
				continue
			}

			// Derive LineReason from the DemandDocument's Kind
			lineReason := "retail_order"
			if doc.Kind == "membership_entitlement" {
				lineReason = "entitlement"
			}

			docID := doc.ID
			lineID := dl.ID
			fl := domain.FulfillmentLine{
				WaveID:           waveID,
				DemandDocumentID: &docID,
				DemandLineID:     &lineID,
				Quantity:         dl.RequestedQuantity,
				AllocationState:  "allocated",
				LineReason:       lineReason,
				GeneratedBy:      "allocation_demand_driven",
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if doc.CustomerProfileID != nil {
				fl.CustomerProfileID = doc.CustomerProfileID
			}

			if err := uc.fulfillRepo.Create(&fl); err != nil {
				return nil, err
			}
			lines = append(lines, fl)
		}
	}

	return lines, nil
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

	return order, nil
}
