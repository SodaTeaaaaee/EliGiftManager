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
}

func NewDemandController() *DemandController {
	gdb := db.GetDB()
	demandRepo := infra.NewDemandRepository(gdb)
	profileRepo := infra.NewProfileRepository(gdb)
	integrationProfileRepo := infra.NewIntegrationProfileRepository(gdb)
	return &DemandController{
		intakeUC:           app.NewDemandIntakeUseCase(demandRepo),
		demandRepo:         demandRepo,
		profileRepo:        profileRepo,
		integrationProfile: integrationProfileRepo,
	}
}

// ImportDemandDocument imports a DemandDocument with its DemandLines.
func (c *DemandController) ImportDemandDocument(input dto.CreateDemandInput) (dto.DemandDocumentDTO, error) {
	if input.CustomerProfileID != nil {
		if _, err := c.profileRepo.FindByID(*input.CustomerProfileID); err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("customer profile %d does not exist", *input.CustomerProfileID)
		}
	}
	if input.IntegrationProfileID != nil {
		if _, err := c.integrationProfile.FindByID(*input.IntegrationProfileID); err != nil {
			return dto.DemandDocumentDTO{}, fmt.Errorf("integration profile %d does not exist", *input.IntegrationProfileID)
		}
	}
	doc := domain.DemandDocument{
		Kind:                 input.Kind,
		CaptureMode:          input.CaptureMode,
		SourceChannel:        input.SourceChannel,
		SourceSurface:        input.SourceSurface,
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
			RoutingDisposition:    l.RoutingDisposition,
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
