package main

import (
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/tabular"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
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
	docType := input.DocumentType
	if docType == "" {
		docType = "import_entitlement"
	}

	// Optional path-based read: resolve rules first so HasHeader is authoritative.
	var orderedRows [][]string
	var headers []string
	headerRows := input.Rows
	totalHint := len(input.Rows)

	if input.FilePath != "" {
		_, rules, resolveErr := c.templateMapping.ResolveTemplateAndRules(ctx, profile.ID, docType)
		if resolveErr != nil {
			return dto.ImportDemandCSVResult{}, fmt.Errorf("template pipeline: %w", resolveErr)
		}
		sheet, readErr := tabular.ReadTabularFile(input.FilePath, tabular.ReadOptions{
			HasHeader: rules.HasHeader,
			Encoding:  "auto",
		})
		if readErr != nil {
			return dto.ImportDemandCSVResult{}, fmt.Errorf("read tabular file: %w", readErr)
		}
		orderedRows = sheet.Rows
		headers = sheet.Headers
		headerRows = nil
		totalHint = len(sheet.Rows)
	}

	_, mapped, rowErrs, rowWarnings, err := c.templateMapping.BuildDemandImportPipelineWithMode(
		ctx, profile.ID, docType, headerRows, orderedRows, headers, mode,
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

	// reject_all: any row error means nothing gets persisted at all.
	if mode == "reject_all" && len(rowErrs) > 0 {
		result.SuccessCount = 0
		return result, nil
	}
	// skip_invalid with zero successfully-mapped rows: nothing to persist.
	if len(mapped) == 0 {
		return result, nil
	}

	// Identity is resolved per distinct source_customer_ref (not first-wins batch fold).
	// Bilibili membership sheets map one UID per row; each distinct ref becomes its own
	// DemandDocument bound to the matching CustomerProfile. Same-UID rows share one doc.
	type importGroup struct {
		ref  string
		rows []app.DemandImportMappedRow
	}
	groupIndex := map[string]int{}
	var groups []importGroup
	for _, row := range mapped {
		ref := strings.TrimSpace(row.Document.SourceCustomerRef)
		if ref == "" {
			ref = strings.TrimSpace(input.SourceCustomerRef)
		}
		if idx, ok := groupIndex[ref]; ok {
			groups[idx].rows = append(groups[idx].rows, row)
			continue
		}
		groupIndex[ref] = len(groups)
		groups = append(groups, importGroup{ref: ref, rows: []app.DemandImportMappedRow{row}})
	}

	identityType := app.ResolveIdentityStrategy(profile.IdentityStrategy)
	// Cache resolved profile IDs so repeated refs (already grouped) and display-name
	// updates stay O(distinct refs).
	resolvedProfiles := map[string]uint{}
	var firstDoc *domain.DemandDocument
	persistedLines := 0

	for _, group := range groups {
		var customerProfileID *uint
		if group.ref != "" && profile.SourceChannel != "" {
			pid, ok := resolvedProfiles[group.ref]
			if !ok {
				var resolveErr error
				pid, resolveErr = c.identityResolution.ResolveOrCreateProfile(ctx, profile.SourceChannel, group.ref, identityType)
				if resolveErr != nil {
					return dto.ImportDemandCSVResult{}, fmt.Errorf("identity resolution for %q: %w", group.ref, resolveErr)
				}
				resolvedProfiles[group.ref] = pid
			}
			customerProfileID = &pid

			// document.display_name belongs to this ref's rows — first non-empty wins per group.
			displayName := ""
			for _, row := range group.rows {
				if strings.TrimSpace(row.Document.DisplayName) != "" {
					displayName = strings.TrimSpace(row.Document.DisplayName)
					break
				}
			}
			if displayName != "" && c.profileRepo != nil {
				if cp, findErr := c.profileRepo.FindByID(ctx, pid); findErr == nil && cp != nil {
					if cp.DisplayName != displayName {
						cp.DisplayName = displayName
						_ = c.profileRepo.Update(ctx, cp)
					}
				}
			}
		}

		// document.source_document_no: input override, else first non-empty in this group.
		sourceDocumentNo := strings.TrimSpace(input.SourceDocumentNo)
		if sourceDocumentNo == "" {
			for _, row := range group.rows {
				if strings.TrimSpace(row.Document.SourceDocumentNo) != "" {
					sourceDocumentNo = strings.TrimSpace(row.Document.SourceDocumentNo)
					break
				}
			}
		}

		// Hybrid address upsert — each recipient row binds to this group's profile.
		if customerProfileID != nil && c.addressUC != nil {
			for _, row := range group.rows {
				if row.Recipient == nil {
					continue
				}
				if strings.TrimSpace(row.Recipient.RecipientName) == "" {
					continue
				}
				if _, upsertErr := c.addressUC.UpsertAddressFromImport(ctx, *customerProfileID, *row.Recipient); upsertErr != nil {
					// Address upsert failure is a row-level soft error: surface but do not
					// abort demand-line persistence (address can be bound later via defaults).
					result.Errors = append(result.Errors, dto.DemandCSVImportError{
						RowIndex: -1,
						Reason:   fmt.Sprintf("address upsert: %v", upsertErr),
					})
					result.ErrorCount++
				}
			}
		}

		lines := make([]*domain.DemandLine, len(group.rows))
		for i := range group.rows {
			lines[i] = group.rows[i].Line
		}

		doc := domain.DemandDocument{
			Kind:                 profile.DemandKind,
			CaptureMode:          "document_import",
			SourceChannel:        profile.SourceChannel,
			SourceSurface:        profile.SourceSurface,
			SourceDocumentNo:     sourceDocumentNo,
			SourceCustomerRef:    group.ref,
			CustomerProfileID:    customerProfileID,
			IntegrationProfileID: &profile.ID,
		}
		if err := c.intakeUC.ImportDemand(ctx, &doc, lines); err != nil {
			return dto.ImportDemandCSVResult{}, err
		}
		persistedLines += len(lines)
		if firstDoc == nil {
			firstDoc = &doc
		}
	}

	result.SuccessCount = persistedLines
	if firstDoc != nil {
		docDTO := domainToDemandDTO(firstDoc)
		result.Document = &docDTO
	}
	return result, nil
}
