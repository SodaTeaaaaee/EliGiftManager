package controller

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// demandTabularMaxBytes caps on-disk spreadsheet reads so ParseTabularFile /
// ImportDemandCSV never follow an unbounded os.ReadFile of a user-picked path.
const demandTabularMaxBytes int64 = tabular.MaxFileBytes

func demandTabularAllowedExts() map[string]struct{} {
	return map[string]struct{}{
		".csv":  {},
		".xlsx": {},
		".xls":  {},
	}
}

// validateDemandImportFilePath rejects NUL, `..` segments, disallowed extensions,
// missing files, directories, and files larger than maxBytes. Users may pick files
// outside the app data dir (e.g. Downloads); this is a local content guard only.
func validateDemandImportFilePath(path string, maxBytes int64, allowedExts map[string]struct{}) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("tabular file path is empty")
	}
	if strings.IndexByte(path, 0) >= 0 {
		return fmt.Errorf("tabular file path contains a NUL byte")
	}
	for _, seg := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if seg == ".." {
			return fmt.Errorf("tabular file path must not contain '..' segments")
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := allowedExts[ext]; !ok {
		return fmt.Errorf("unsupported tabular extension %q", ext)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat tabular file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("tabular path is a directory")
	}
	if info.Size() > maxBytes {
		return fmt.Errorf("tabular file exceeds max size of %d bytes", maxBytes)
	}
	return nil
}

func demandImportEvidenceRowIndex(row app.DemandImportMappedRow) int {
	if row.Line != nil && row.Line.SourceLineNo > 0 {
		return row.Line.SourceLineNo - 1
	}
	return -1
}

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
	if err := validateDemandImportFilePath(path, demandTabularMaxBytes, demandTabularAllowedExts()); err != nil {
		return dto.CSVFilePreviewDTO{}, err
	}
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
// mode requires ordered cells). Kind / SourceSurface / IdentityStrategy are derived from
// the resolved documentType; the integration profile supplies platform SourceChannel and
// template binding.
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
	interp, err := app.InterpretDemandImportDocumentType(docType)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("interpret demand document type: %w", err)
	}
	kind, sourceSurface, identityStrategy := interp.DemandKind, interp.SourceSurface, interp.IdentityStrategy

	// Optional path-based read: resolve rules first so HasHeader is authoritative.
	var orderedRows [][]string
	var headers []string
	headerRows := input.Rows
	totalHint := len(input.Rows)

	if input.FilePath != "" {
		if err := validateDemandImportFilePath(input.FilePath, demandTabularMaxBytes, demandTabularAllowedExts()); err != nil {
			return dto.ImportDemandCSVResult{}, err
		}
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
		Documents:      []dto.DemandDocumentDTO{},
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
			return result, err
		}
		return result, nil
	}
	// skip_invalid with zero successfully-mapped rows: nothing to persist.
	if len(mapped) == 0 {
		if err := completeEvidenceOnly("failed"); err != nil {
			return result, err
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
		if identityStrategy == app.IdentityStrategyOrderScopedProvisional {
			groupKey = sourceDocumentNo
		}
		// Empty grouping keys must not collapse unrelated rows into one document
		// (membership empty source_customer_ref, or retail empty order number).
		if groupKey == "" {
			groupKey = fmt.Sprintf("\x00row-%d", rowIndex)
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

	importedDocs := make([]dto.DemandDocumentDTO, 0, len(groups))
	persistedLines := 0
	importedAt := time.Now().UTC()
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		for _, group := range groups {
			var saved domain.DemandDocument
			groupErr := tx.Transaction(func(groupTx *gorm.DB) error {
				demandRepo := infra.NewDemandRepository(groupTx)
				profileRepo := infra.NewProfileRepository(groupTx)
				originRepo := infra.NewCustomerProfileOriginRepository(groupTx)
				observationRepo := infra.NewCustomerNameObservationRepository(groupTx)
				eventRepo := infra.NewCustomerNameEventRepository(groupTx)
				customerResolver := app.NewDemandCustomerResolutionService(profileRepo, originRepo)
				nameService := app.NewCustomerNameObservationService(profileRepo, observationRepo, eventRepo)
				intakeUC := app.NewDemandIntakeUseCase(demandRepo)

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

				var customerProfileID, identityID, originID *uint
				// Membership/entitlement always resolves identity. Empty
				// source_customer_ref is an error and must not persist a
				// CustomerProfileID=nil document. Retail order-scoped
				// resolution still requires sourceDocumentNo.
				needsIdentity := identityStrategy != app.IdentityStrategyOrderScopedProvisional ||
					strings.TrimSpace(group.sourceDocumentNo) != ""
				if needsIdentity {
					resolved, resolveErr := customerResolver.Resolve(ctx, app.DemandCustomerResolutionInput{
						IntegrationProfileID: profile.ID,
						IdentityStrategy:     identityStrategy,
						SourceChannel:        profile.SourceChannel,
						SourceDocumentNo:     group.sourceDocumentNo,
						SourceCustomerRef:    group.ref,
						DisplayName:          displayName,
						ObservedAt:           importedAt,
					})
					if resolveErr != nil {
						resolveLabel := strings.TrimSpace(group.sourceDocumentNo)
						if resolveLabel == "" {
							resolveLabel = strings.TrimSpace(group.ref)
						}
						if resolveLabel == "" {
							resolveLabel = "(empty source_customer_ref)"
						}
						return fmt.Errorf("customer resolution for %q: %w", resolveLabel, resolveErr)
					}
					customerProfileID, identityID, originID = resolved.CustomerProfileID, resolved.IdentityID, resolved.OriginID
					if customerProfileID == nil {
						return fmt.Errorf("customer resolution for %q produced no customer profile", group.ref)
					}
				}

				lines := make([]*domain.DemandLine, len(group.rows))
				for i := range group.rows {
					lines[i] = group.rows[i].Line
				}
				doc := domain.DemandDocument{
					Kind: kind, CaptureMode: "document_import", SourceChannel: profile.SourceChannel,
					SourceSurface: sourceSurface, SourceDocumentNo: group.sourceDocumentNo,
					SourceCustomerRef: group.ref, CustomerProfileID: customerProfileID, IntegrationProfileID: &profile.ID,
				}
				if err := intakeUC.ImportDemand(ctx, &doc, lines); err != nil {
					return fmt.Errorf("persist demand group %q: %w", group.sourceDocumentNo, err)
				}

				if originID != nil {
					if err := customerResolver.AttachOriginDocument(ctx, *originID, doc.ID); err != nil {
						return err
					}
				}

				if customerProfileID != nil && displayName != "" {
					nameKind := domain.CustomerNameKindStableIdentityNickname
					if identityStrategy == app.IdentityStrategyOrderScopedProvisional {
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
					addressUC := app.NewAddressManagementUseCase(infra.NewAddressRepository(groupTx), infra.NewFulfillmentRepository(groupTx))
					for _, row := range group.rows {
						if row.Recipient == nil || strings.TrimSpace(row.Recipient.RecipientName) == "" {
							continue
						}
						if _, err := addressUC.UpsertAddressFromImport(ctx, *customerProfileID, *row.Recipient); err != nil {
							return fmt.Errorf("address upsert: %w", err)
						}
					}
				}

				saved = doc
				return nil
			})
			if groupErr != nil {
				if mode == "reject_all" {
					return groupErr
				}
				for _, row := range group.rows {
					rowIdx := demandImportEvidenceRowIndex(row)
					reason := groupErr.Error()
					result.Errors = append(result.Errors, dto.DemandCSVImportError{RowIndex: rowIdx, Reason: reason})
					result.ErrorCount++
					app.MarkImportEvidenceFailure(evidenceRecords, rowIdx, "persist_error", reason, nil)
				}
				continue
			}
			docDTO := domainToDemandDTO(&saved)
			importedDocs = append(importedDocs, docDTO)
			for _, row := range group.rows {
				app.MarkImportEvidenceSuccess(evidenceRecords, demandImportEvidenceRowIndex(row), "demand_document", saved.ID)
			}
			persistedLines += len(group.rows)
		}
		if persistedLines == 0 {
			return evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, "failed")
		}
		status := "completed"
		if result.ErrorCount > 0 {
			status = "partial_success"
		}
		return evidence.CompleteImportEvidence(ctx, evidenceRun, evidenceRecords, status)
	})
	if err != nil {
		return result, errors.Join(err, evidence.FinalizeFailure(ctx, "failed", err))
	}
	if err := evidence.FinalizePending(ctx); err != nil {
		return result, err
	}

	result.SuccessCount = persistedLines
	result.Documents = importedDocs
	if len(importedDocs) > 0 {
		docDTO := importedDocs[0]
		result.Document = &docDTO
	}
	return result, nil
}
