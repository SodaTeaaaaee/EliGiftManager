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
	waveRepo domain.WaveRepository
}

func NewWaveUseCase(waveRepo domain.WaveRepository) WaveUseCase {
	return &waveUseCase{waveRepo: waveRepo}
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
}

func NewExportUseCase(supplierRepo domain.SupplierOrderRepository, fulfillRepo domain.FulfillmentLineRepository) ExportUseCase {
	return &exportUseCase{supplierRepo: supplierRepo, fulfillRepo: fulfillRepo}
}

func (uc *exportUseCase) ExportSupplierOrder(waveID uint) (*domain.SupplierOrder, error) {
	// Delete only existing draft orders for this wave (rebuild pattern for idempotency)
	if err := uc.supplierRepo.DeleteDraftsByWave(waveID); err != nil {
		return nil, err
	}

	// [V2-STUB] aggregate all FulfillmentLines for the wave into a SupplierOrder with lines
	fulfillLines, err := uc.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	order := &domain.SupplierOrder{
		WaveID:         waveID,
		Status:         "draft",
		SubmissionMode: "csv",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := uc.supplierRepo.Create(order); err != nil {
		return nil, err
	}

	for i := range fulfillLines {
		fl := &fulfillLines[i]
		line := &domain.SupplierOrderLine{
			SupplierOrderID:   order.ID,
			FulfillmentLineID: fl.ID,
			SubmittedQuantity: fl.Quantity,
			Status:            "draft",
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := uc.supplierRepo.CreateLine(line); err != nil {
			return nil, err
		}
	}

	return order, nil
}
