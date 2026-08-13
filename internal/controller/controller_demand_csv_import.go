package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// ParseTabularFile reads a locally-picked spreadsheet (CSV / XLSX / XLS; path typically
// obtained via App.PickTabularFile) and returns headers plus every data row as a
// header-keyed map. Format is detected from the file extension. The frontend reuses the
// same parsed rows for preview and import so the file is only read once.
//
// hasHeader controls whether the first physical row is treated as column headers.
// Pass false for headerless sheets (e.g. bilibili membership positional CSV) so row0
// remains data. When hasHeader is false and the sheet has no headers, synthetic
// col_0..col_N names are generated so Rows stay addressable as maps.
func (c *DemandController) ParseTabularFile(path string, hasHeader bool) (dto.CSVFilePreviewDTO, error) {
	sheet, err := tabular.ReadTabularFile(path, tabular.ReadOptions{
		// Empty Format → extension-based detection (csv / xlsx / xls).
		Format:    "",
		HasHeader: hasHeader,
		Encoding:  "auto",
	})
	if err != nil {
		return dto.CSVFilePreviewDTO{}, err
	}
	headers := sheet.Headers
	// Headerless sheets leave Headers empty; synthesize stable col_N keys so the
	// preview DTO still carries cell values via HeaderKeyedRows.
	if !hasHeader && len(headers) == 0 {
		maxCols := 0
		for _, record := range sheet.Rows {
			if len(record) > maxCols {
				maxCols = len(record)
			}
		}
		headers = make([]string, maxCols)
		for i := range headers {
			headers[i] = fmt.Sprintf("col_%d", i)
		}
		sheet.Headers = headers
	}
	return dto.CSVFilePreviewDTO{
		Headers: headers,
		Rows:    sheet.HeaderKeyedRows(),
	}, nil
}

// ParseCSVFile is the legacy entry point; it delegates to ParseTabularFile so CSV and
// multi-format callers share one code path. Always treats the first row as headers.
func (c *DemandController) ParseCSVFile(path string) (dto.CSVFilePreviewDTO, error) {
	return c.ParseTabularFile(path, true)
}

// ImportDemandCSV performs a dual-mode (reject_all / skip_invalid) template-driven demand CSV
// import. Supports v2 dest namespaces (document.* / line.* / recipient.*). When FilePath is
// set the backend re-reads the tabular file with hasHeader from mapping rules (positional
// mode requires ordered cells). Profile-driven DemandKind / SourceChannel / SourceSurface
// are always forced from the integration profile.
func (c *DemandController) ImportDemandCSV(input dto.ImportDemandCSVInput) (dto.ImportDemandCSVResult, error) {
	ctx := appContext
	if err := c.requireCustomerResolutionWrites(ctx); err != nil {
		return dto.ImportDemandCSVResult{}, err
	}

	mode := input.ImportMode
	if mode == "" {
		mode = "skip_invalid"
	}
	if mode != "reject_all" && mode != "skip_invalid" {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("invalid importMode %q: must be \"reject_all\" or \"skip_invalid\"", mode)
	}

	profile, err := c.integrationProfile.FindByID(ctx, input.IntegrationProfileID)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("integration profile %d not found: %w", input.IntegrationProfileID, err)
	}
	if profile == nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("integration profile %d not found", input.IntegrationProfileID)
	}
	docType, err := app.ResolveDemandImportDocumentType(profile, input.DocumentType)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("resolve demand document type: %w", err)
	}

	// Optional path-based read: resolve rules first so HasHeader is authoritative.
	var orderedRows [][]string
	var headers []string
	headerRows := input.Rows
	totalHint := len(input.Rows)

	if input.FilePath != "" {
		_, rules, resolveErr := c.templateMapping.ResolveDemandImportTemplateAndRules(ctx, profile.ID, docType, input.MappingRules)
		if resolveErr != nil {
			return dto.ImportDemandCSVResult{}, fmt.Errorf("template pipeline: %w", resolveErr)
		}
		sheet, readErr := tabular.ReadTabularFile(input.FilePath, tabular.ReadOptions{
			HasHeader: rules.HasHeader,
			Encoding:  "auto",
			SheetName: rules.SheetName,
		})
		if readErr != nil {
			return dto.ImportDemandCSVResult{}, fmt.Errorf("read tabular file: %w", readErr)
		}
		orderedRows = sheet.Rows
		headers = sheet.Headers
		headerRows = nil
		totalHint = len(sheet.Rows)
	}

	_, mapped, rowErrs, rowWarnings, err := c.templateMapping.BuildDemandImportPipelineWithModeAndOverride(
		ctx, profile.ID, docType, headerRows, orderedRows, headers, mode, input.MappingRules,
	)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("template pipeline: %w", err)
	}

	result := dto.ImportDemandCSVResult{
		TotalProcessed: totalHint,
		SuccessCount:   len(mapped),
		ErrorCount:     len(rowErrs),
		Errors:         make([]dto.DemandCSVImportError, len(rowErrs)),
		Warnings:       rowWarnings,
	}
	for i, re := range rowErrs {
		result.Errors[i] = dto.DemandCSVImportError{RowIndex: re.RowIndex, Reason: re.Reason}
	}
	evidenceRows, evidenceUnmapped := app.BuildImportEvidenceRows(orderedRows, headers, headerRows)
	if c.gdb == nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("demand import database is not configured")
	}
	evidence := app.NewDeferredImportEvidenceUseCase(infra.NewImportEvidenceRepository(c.gdb))
	evidenceRun, evidenceRecords, err := evidence.StartImportEvidence(ctx, "demand", profile.ID, mode, input.FilePath, `{"pipeline":"demand_mapping_v2"}`, evidenceRows, evidenceUnmapped, nil)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("start demand import evidence: %w", err)
	}
	if evidenceRun != nil {
		result.ImportRunID = evidenceRun.ID
	} else {
		result.EvidenceDisabled = true
	}
	for i := range evidenceRecords {
		app.MarkImportEvidenceSuccess(evidenceRecords, i, "demand_document", 0)
	}
	for _, rowErr := range rowErrs {
		app.MarkImportEvidenceFailure(evidenceRecords, rowErr.RowIndex, "mapping_error", rowErr.Reason, nil)
	}
	completeEvidenceOnly := func(status string) error {
		if status == "rejected" {
			return evidence.FinalizeFailure(ctx, status, nil)
		}
		if err := evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, status); err != nil {
			return err
		}
		return evidence.FinalizePending(ctx)
	}

	// reject_all: any row error means nothing gets persisted at all.
	if mode == "reject_all" && len(rowErrs) > 0 {
		result.SuccessCount = 0
		if err := completeEvidenceOnly("rejected"); err != nil {
			return dto.ImportDemandCSVResult{}, err
		}
		return result, nil
	}
	// skip_invalid with zero successfully-mapped rows: nothing to persist.
	if len(mapped) == 0 {
		if err := completeEvidenceOnly("failed"); err != nil {
			return dto.ImportDemandCSVResult{}, err
		}
		return result, nil
	}

	// Membership inputs group by source_customer_ref because each member entitlement is
	// customer-scoped. Retail inputs group by source_document_no because one customer can
	// place multiple independent orders; customer ref remains attached to each order group.
	type importGroup struct {
		ref              string
		sourceDocumentNo string
		rows             []app.DemandImportMappedRow
	}
	groupIndex := map[string]int{}
	groups := make([]importGroup, 0, len(mapped))
	for rowIndex, row := range mapped {
		ref := strings.TrimSpace(row.Document.SourceCustomerRef)
		if ref == "" {
			ref = strings.TrimSpace(input.SourceCustomerRef)
		}
		sourceDocumentNo := strings.TrimSpace(input.SourceDocumentNo)
		if sourceDocumentNo == "" {
			sourceDocumentNo = strings.TrimSpace(row.Document.SourceDocumentNo)
		}

		groupKey := ref
		if profile.IdentityStrategy == app.IdentityStrategyOrderScopedProvisional {
			groupKey = sourceDocumentNo
			// A custom retail template may omit an order number even when the profile
			// does not require one. Never merge such unrelated rows by customer ref.
			if groupKey == "" {
				groupKey = fmt.Sprintf("\x00retail-row-%d", rowIndex)
			}
		}
		if idx, ok := groupIndex[groupKey]; ok {
			if groups[idx].ref == "" {
				groups[idx].ref = ref
			}
			if groups[idx].sourceDocumentNo == "" {
				groups[idx].sourceDocumentNo = sourceDocumentNo
			}
			groups[idx].rows = append(groups[idx].rows, row)
			continue
		}
		groupIndex[groupKey] = len(groups)
		groups = append(groups, importGroup{
			ref:              ref,
			sourceDocumentNo: sourceDocumentNo,
			rows:             []app.DemandImportMappedRow{row},
		})
	}

	var firstDoc *domain.DemandDocument
	persistedLines := 0
	importedAt := time.Now().UTC()
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		demandRepo := infra.NewDemandRepository(tx)
		profileRepo := infra.NewProfileRepository(tx)
		originRepo := infra.NewCustomerProfileOriginRepository(tx)
		observationRepo := infra.NewCustomerNameObservationRepository(tx)
		eventRepo := infra.NewCustomerNameEventRepository(tx)
		customerResolver := app.NewDemandCustomerResolutionService(profileRepo, originRepo)
		nameService := app.NewCustomerNameObservationService(profileRepo, observationRepo, eventRepo)
		intakeUC := app.NewDemandIntakeUseCase(demandRepo)

		for _, group := range groups {
			displayName := ""
			for _, row := range group.rows {
				if candidate := strings.TrimSpace(row.Document.DisplayName); candidate != "" {
					displayName = candidate
					break
				}
			}
			if displayName == "" {
				displayName = group.ref
			}

			resolved, resolveErr := customerResolver.Resolve(ctx, app.DemandCustomerResolutionInput{
				IntegrationProfileID: profile.ID,
				IdentityStrategy:     profile.IdentityStrategy,
				SourceChannel:        profile.SourceChannel,
				SourceDocumentNo:     group.sourceDocumentNo,
				SourceCustomerRef:    group.ref,
				DisplayName:          displayName,
				ObservedAt:           importedAt,
			})
			if resolveErr != nil {
				return fmt.Errorf("customer resolution for %q: %w", group.sourceDocumentNo, resolveErr)
			}
			customerProfileID, identityID, originID := resolved.CustomerProfileID, resolved.IdentityID, resolved.OriginID

			lines := make([]*domain.DemandLine, len(group.rows))
			for i := range group.rows {
				lines[i] = group.rows[i].Line
			}
			doc := domain.DemandDocument{
				Kind: profile.DemandKind, CaptureMode: "document_import", SourceChannel: profile.SourceChannel,
				SourceSurface: profile.SourceSurface, SourceDocumentNo: group.sourceDocumentNo,
				SourceCustomerRef: group.ref, CustomerProfileID: customerProfileID, IntegrationProfileID: &profile.ID,
			}
			if err := intakeUC.ImportDemand(ctx, &doc, lines); err != nil {
				return fmt.Errorf("persist demand group %q: %w", group.sourceDocumentNo, err)
			}
			for _, row := range group.rows {
				app.MarkImportEvidenceSuccess(evidenceRecords, row.Line.SourceLineNo-1, "demand_document", doc.ID)
			}

			if originID != nil {
				if err := customerResolver.AttachOriginDocument(ctx, *originID, doc.ID); err != nil {
					return err
				}
			}

			if customerProfileID != nil && displayName != "" {
				nameKind := domain.CustomerNameKindStableIdentityNickname
				if profile.IdentityStrategy == app.IdentityStrategyOrderScopedProvisional {
					nameKind = domain.CustomerNameKindTrustedNickname
				}
				if _, observeErr := nameService.Observe(ctx, app.ObserveCustomerNameInput{
					CustomerProfileID: *customerProfileID, Name: displayName, NameKind: nameKind,
					Authority: profile.SourceChannel, SourceEventKey: fmt.Sprintf("demand-document:%d:name", doc.ID),
					SourceIntegrationProfileID: &profile.ID, SourceDocumentID: &doc.ID,
					SourceIdentityID: identityID, ObservedAt: importedAt,
				}); observeErr != nil {
					return fmt.Errorf("observe customer name: %w", observeErr)
				}
			}

			if customerProfileID != nil {
				for _, row := range group.rows {
					if row.Recipient == nil || strings.TrimSpace(row.Recipient.RecipientName) == "" {
						continue
					}
					upsertErr := tx.Transaction(func(addressTx *gorm.DB) error {
						addressUC := app.NewAddressManagementUseCase(infra.NewAddressRepository(addressTx), infra.NewFulfillmentRepository(addressTx))
						_, err := addressUC.UpsertAddressFromImport(ctx, *customerProfileID, *row.Recipient)
						return err
					})
					if upsertErr != nil {
						result.Errors = append(result.Errors, dto.DemandCSVImportError{RowIndex: -1, Reason: fmt.Sprintf("address upsert: %v", upsertErr)})
						result.ErrorCount++
					}
				}
			}

			persistedLines += len(lines)
			if firstDoc == nil {
				copy := doc
				firstDoc = &copy
			}
		}
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		return evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, status)
	})
	if err != nil {
		return dto.ImportDemandCSVResult{}, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return dto.ImportDemandCSVResult{}, err
	}

	result.SuccessCount = persistedLines
	if firstDoc != nil {
		docDTO := domainToDemandDTO(firstDoc)
		result.Document = &docDTO
	}
	return result, nil
}
