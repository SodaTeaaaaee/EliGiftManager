package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// ParseCSVFile reads a locally-picked CSV file (path typically obtained via App.PickCSVFile)
// and returns its parsed headers plus every data row as a header-keyed map. The frontend
// slices the first few rows for preview and reuses the same parsed rows for the real import,
// so the file is only ever read once.
func (c *DemandController) ParseCSVFile(path string) (dto.CSVFilePreviewDTO, error) {
	f, err := os.Open(path)
	if err != nil {
		return dto.CSVFilePreviewDTO{}, fmt.Errorf("open csv file %q: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	// Ragged rows (fewer/more fields than the header row) must not panic or hard-fail —
	// disable the built-in fixed-width check and guard manually below.
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return dto.CSVFilePreviewDTO{}, fmt.Errorf("read csv file %q: %w", path, err)
	}
	if len(records) == 0 {
		return dto.CSVFilePreviewDTO{}, fmt.Errorf("csv file %q is empty", path)
	}

	headers := records[0]
	if len(headers) == 0 {
		return dto.CSVFilePreviewDTO{}, fmt.Errorf("csv file %q has no headers", path)
	}
	// Strip a UTF-8 BOM from the first header cell, common in Excel-exported CSVs.
	headers[0] = strings.TrimPrefix(headers[0], "\ufeff")

	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(headers))
		n := len(headers)
		if len(record) < n {
			n = len(record)
		}
		for i := 0; i < n; i++ {
			row[headers[i]] = record[i]
		}
		rows = append(rows, row)
	}

	return dto.CSVFilePreviewDTO{Headers: headers, Rows: rows}, nil
}

// ImportDemandCSV performs a dual-mode (reject_all / skip_invalid) template-driven demand CSV
// import. Unlike ImportDemandFromCSV, a "skip_invalid" run persists only the successfully
// mapped rows as a single DemandDocument and surfaces the failed rows as data (not a Go error).
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

	_, mappedLines, rowErrs, err := c.templateMapping.BuildImportPipelineWithMode(ctx, profile.ID, docType, input.Rows, mode)
	if err != nil {
		return dto.ImportDemandCSVResult{}, fmt.Errorf("template pipeline: %w", err)
	}

	result := dto.ImportDemandCSVResult{
		TotalProcessed: len(input.Rows),
		SuccessCount:   len(mappedLines),
		ErrorCount:     len(rowErrs),
		Errors:         make([]dto.DemandCSVImportError, len(rowErrs)),
	}
	for i, re := range rowErrs {
		result.Errors[i] = dto.DemandCSVImportError{RowIndex: re.RowIndex, Reason: re.Reason}
	}

	// reject_all: any row error means nothing gets persisted at all.
	if mode == "reject_all" && len(rowErrs) > 0 {
		result.SuccessCount = 0
		return result, nil
	}
	// skip_invalid with zero successfully-mapped rows: nothing to persist, don't create an
	// empty document.
	if len(mappedLines) == 0 {
		return result, nil
	}

	var customerProfileID *uint
	if input.SourceCustomerRef != "" && profile.SourceChannel != "" {
		identityType := app.ResolveIdentityStrategy(profile.IdentityStrategy)
		pid, resolveErr := c.identityResolution.ResolveOrCreateProfile(ctx, profile.SourceChannel, input.SourceCustomerRef, identityType)
		if resolveErr != nil {
			return dto.ImportDemandCSVResult{}, fmt.Errorf("identity resolution: %w", resolveErr)
		}
		customerProfileID = &pid
	}

	doc := domain.DemandDocument{
		Kind:                 profile.DemandKind,
		CaptureMode:          "document_import",
		SourceChannel:        profile.SourceChannel,
		SourceSurface:        profile.SourceSurface,
		SourceDocumentNo:     input.SourceDocumentNo,
		SourceCustomerRef:    input.SourceCustomerRef,
		CustomerProfileID:    customerProfileID,
		IntegrationProfileID: &profile.ID,
	}
	if err := c.intakeUC.ImportDemand(ctx, &doc, mappedLines); err != nil {
		return dto.ImportDemandCSVResult{}, err
	}
	docDTO := domainToDemandDTO(&doc)
	result.Document = &docDTO
	return result, nil
}
