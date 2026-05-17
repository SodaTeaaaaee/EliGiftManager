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
	intakeUC           app.DemandIntakeUseCase
	demandRepo         domain.DemandDocumentRepository
	profileRepo        domain.CustomerProfileRepository
	integrationProfile domain.IntegrationProfileRepository
	assignmentRepo     domain.WaveDemandAssignmentRepository
	waveRepo           domain.WaveRepository
}

func NewDemandController() *DemandController {
	gdb := db.GetDB()
	demandRepo := infra.NewDemandRepository(gdb)
	profileRepo := infra.NewProfileRepository(gdb)
	integrationProfileRepo := infra.NewIntegrationProfileRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	waveRepo := infra.NewWaveRepository(gdb)
	return &DemandController{
		intakeUC:           app.NewDemandIntakeUseCase(demandRepo),
		demandRepo:         demandRepo,
		profileRepo:        profileRepo,
		integrationProfile: integrationProfileRepo,
		assignmentRepo:     assignmentRepo,
		waveRepo:           waveRepo,
	}
}

// ImportDemandDocument imports a DemandDocument with its DemandLines.
func (c *DemandController) ImportDemandDocument(input dto.CreateDemandInput) (dto.DemandDocumentDTO, error) {
	if input.CustomerProfileID != nil {
		if _, err := c.profileRepo.FindByID(*input.CustomerProfileID); err != nil {
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

	if input.IntegrationProfileID != nil {
		profile, err := c.integrationProfile.FindByID(*input.IntegrationProfileID)
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
	}

	doc := domain.DemandDocument{
		Kind:                 effectiveKind,
		CaptureMode:          input.CaptureMode,
		SourceChannel:        effectiveSourceChannel,
		SourceSurface:        effectiveSourceSurface,
		SourceDocumentNo:     input.SourceDocumentNo,
		SourceCustomerRef:    input.SourceCustomerRef,
		CustomerProfileID:    input.CustomerProfileID,
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
			EligibilityContextRef:  l.EligibilityContextRef,
			EntitlementCode:       l.EntitlementCode,
			GiftLevelSnapshot:     l.GiftLevelSnapshot,
			ProductMasterID:       l.ProductMasterID,
			RecipientInputPayload: l.RecipientInputPayload,
			ExternalTitle:         l.ExternalTitle,
			RequestedQuantity:     l.RequestedQuantity,
		}
	}
	if err := c.intakeUC.ImportDemand(&doc, lines); err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(&doc), nil
}

// ListDemandDocuments lists all demand documents.
func (c *DemandController) ListDemandDocuments() ([]dto.DemandDocumentDTO, error) {
	docs, err := c.demandRepo.List()
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
	docs, err := c.demandRepo.ListUnassigned()
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
	lines, err := c.demandRepo.ListLinesByDocument(documentID)
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
	doc, err := c.demandRepo.FindByID(id)
	if err != nil {
		return dto.DemandDocumentDTO{}, err
	}
	return domainToDemandDTO(doc), nil
}

func (c *DemandController) ListDemandInboxRows(input dto.DemandInboxFilterInput) ([]dto.DemandInboxRowDTO, error) {
	docs, err := c.demandRepo.List()
	if err != nil {
		return nil, err
	}
	waves, err := c.waveRepo.List()
	if err != nil {
		return nil, err
	}
	waveMap := make(map[uint]domain.Wave, len(waves))
	for _, w := range waves {
		waveMap[w.ID] = w
	}

	rows := make([]dto.DemandInboxRowDTO, 0, len(docs))
	for _, doc := range docs {
		if input.DemandKind != "" && doc.Kind != input.DemandKind {
			continue
		}

		assignments, err := c.assignmentRepo.ListByDemandDocument(doc.ID)
		if err != nil {
			return nil, err
		}
		assigned := len(assignments) > 0
		if input.Assignment == "assigned" && !assigned {
			continue
		}
		if input.Assignment == "unassigned" && assigned {
			continue
		}

		lines, err := c.demandRepo.ListLinesByDocument(doc.ID)
		if err != nil {
			return nil, err
		}

		row := dto.DemandInboxRowDTO{
			DemandDocumentID:     doc.ID,
			Kind:                 doc.Kind,
			CaptureMode:          doc.CaptureMode,
			SourceChannel:        doc.SourceChannel,
			SourceSurface:        doc.SourceSurface,
			SourceDocumentNo:     doc.SourceDocumentNo,
			CustomerProfileID:    doc.CustomerProfileID,
			IntegrationProfileID: doc.IntegrationProfileID,
			Assigned:             assigned,
			CreatedAt:            doc.CreatedAt,
		}
		if doc.IntegrationProfileID != nil {
			if profile, profileErr := c.integrationProfile.FindByID(*doc.IntegrationProfileID); profileErr == nil && profile != nil {
				row.IntegrationProfileLabel = fmt.Sprintf("%s (%s)", profile.ProfileKey, profile.SourceChannel)
			}
		}
		if assigned {
			waveID := assignments[0].WaveID
			row.AssignedWaveID = &waveID
			if wave, ok := waveMap[waveID]; ok {
				row.AssignedWaveLabel = fmt.Sprintf("%s — %s", wave.WaveNo, wave.Name)
			}
		}
		for _, line := range lines {
			row.TotalLineCount++
			switch line.RoutingDisposition {
			case "accepted":
				row.AcceptedCount++
				if line.RecipientInputState == "ready" || line.RecipientInputState == "not_required" {
					row.ReadyAcceptedCount++
				}
				if line.RecipientInputState == "waiting_for_input" || line.RecipientInputState == "partially_collected" {
					row.WaitingInputCount++
				}
			case "deferred":
				row.DeferredCount++
			case "excluded_manual", "excluded_duplicate", "excluded_revoked":
				row.ExcludedCount++
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
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
		EligibilityContextRef:  line.EligibilityContextRef,
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
