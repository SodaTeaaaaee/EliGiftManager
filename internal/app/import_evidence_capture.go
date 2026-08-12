package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type PrepareTemplateImportEvidenceInput struct {
	ImportKind           string
	DocumentType         string
	IntegrationProfileID uint
	ImportMode           string
	FilePath             string
	Rows                 []map[string]string
	IncludeZIPAssets     bool
}

// PrepareTemplateImportEvidence parses the complete source and durably records
// its RAW evidence before a caller opens any business transaction.
func PrepareTemplateImportEvidence(ctx context.Context, evidence *ImportEvidenceUseCase, mapping *TemplateMappingService, input PrepareTemplateImportEvidenceInput) error {
	if evidence == nil {
		return nil
	}
	enabled, err := evidence.Enabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	_, rules, err := mapping.ResolveTemplateAndRules(ctx, input.IntegrationProfileID, input.DocumentType)
	if err != nil {
		return fmt.Errorf("template pipeline: %w", err)
	}
	ordered, headers, headerRows, _, _, cleanup, err := loadImportRows(input.FilePath, input.Rows, rules, nil)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}
	rows, unmapped := importEvidenceRows(ordered, headers, headerRows)
	assets := make([][]map[string]string, len(rows))
	if input.IncludeZIPAssets && strings.EqualFold(filepath.Ext(input.FilePath), ".zip") && len(assets) > 0 {
		metadata, metadataErr := zipAssetMetadata(input.FilePath)
		if metadataErr != nil {
			return metadataErr
		}
		assets[0] = metadata
	}
	parserMetadata := fmt.Sprintf(`{"hasHeader":%t,"sheetName":%q}`, rules.HasHeader, rules.SheetName)
	_, _, err = evidence.StartImportEvidence(ctx, input.ImportKind, input.IntegrationProfileID, normalizedImportMode(input.ImportMode), input.FilePath, parserMetadata, rows, unmapped, assets)
	return err
}

func PrepareShipmentEntryImportEvidence(ctx context.Context, evidence *ImportEvidenceUseCase, input dto.ImportShipmentInput) error {
	if evidence == nil {
		return nil
	}
	enabled, err := evidence.Enabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	rows := make([]any, len(input.Entries))
	unmapped := make([]map[string]string, len(input.Entries))
	for i, entry := range input.Entries {
		rows[i] = entry
		unmapped[i] = map[string]string{}
	}
	_, _, err = evidence.StartImportEvidence(ctx, "supplier_shipment", input.IntegrationProfileID, normalizedImportMode(input.ImportMode), "", `{"source":"mapped_entries"}`, rows, unmapped, nil)
	return err
}

func normalizedImportMode(mode string) string {
	if mode == "" {
		return "skip_invalid"
	}
	return mode
}

func importEvidenceRows(ordered [][]string, headers []string, headerRows []map[string]string) ([]any, []map[string]string) {
	if len(ordered) > 0 {
		rows := make([]any, len(ordered))
		unmapped := make([]map[string]string, len(ordered))
		for i, row := range ordered {
			rows[i] = append([]string(nil), row...)
			unmapped[i] = map[string]string{}
			for col, header := range headers {
				if col < len(row) {
					unmapped[i][header] = row[col]
				}
			}
		}
		return rows, unmapped
	}
	rows := make([]any, len(headerRows))
	for i := range headerRows {
		rows[i] = headerRows[i]
	}
	return rows, headerRows
}

func BuildImportEvidenceRows(ordered [][]string, headers []string, headerRows []map[string]string) ([]any, []map[string]string) {
	return importEvidenceRows(ordered, headers, headerRows)
}

func MarkImportEvidenceFailure(records []domain.ImportRawRecord, index int, code, message string, warnings []string) {
	markImportEvidenceFailure(records, index, code, message, warnings)
}

func MarkImportEvidenceSuccess(records []domain.ImportRawRecord, index int, resultType string, resultID uint) {
	markImportEvidenceSuccess(records, index, resultType, resultID)
}

func importEvidenceRunID(run *domain.ImportRun) uint {
	if run == nil {
		return 0
	}
	return run.ID
}
func markImportEvidenceFailure(records []domain.ImportRawRecord, index int, code, message string, warnings []string) {
	if index < 0 || index >= len(records) {
		return
	}
	raw, _ := json.Marshal(warnings)
	records[index].Outcome = "failed"
	records[index].ErrorCode = code
	records[index].ErrorMessage = message
	records[index].WarningCodes = string(raw)
}
func markImportEvidenceSuccess(records []domain.ImportRawRecord, index int, resultType string, resultID uint) {
	if index < 0 || index >= len(records) {
		return
	}
	records[index].Outcome = "success"
	records[index].ResultType = resultType
	if resultID != 0 {
		records[index].ResultID = &resultID
	}
}

// zipAssetMetadata records member path/hash/size without retaining binary
// payloads. Tabular members are excluded because their logical rows are stored.
func zipAssetMetadata(path string) ([]map[string]string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open zip for evidence metadata: %w", err)
	}
	defer reader.Close()
	out := make([]map[string]string, 0)
	for _, member := range reader.File {
		if member.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(member.Name))
		if ext == ".csv" || ext == ".xlsx" || ext == ".xls" {
			continue
		}
		r, err := member.Open()
		if err != nil {
			return nil, err
		}
		sum := sha256.New()
		_, copyErr := io.Copy(sum, r)
		closeErr := r.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		out = append(out, map[string]string{"member": member.Name, "path": filepath.ToSlash(member.Name), "sha256": hex.EncodeToString(sum.Sum(nil)), "size": fmt.Sprint(member.UncompressedSize64)})
	}
	return out, nil
}
