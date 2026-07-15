package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// RowMappingError records a single CSV row that failed template-driven mapping,
// paired with its zero-based index in the original rows slice.
type RowMappingError struct {
	RowIndex int
	Reason   string
}

// BuildImportPipelineWithMode resolves the template for a profile + document type ONCE
// (reusing ResolveImportTemplate + ParseMappingRules), then maps rows to DemandLines using
// the same per-row logic as MapCSVRowToDemandLine, according to importMode:
//
//   - "reject_all": aborts mapping at the first row that fails — matching BuildImportPipeline's
//     existing abort-on-first-error behavior. The failing row is still reported as a single
//     RowMappingError (rather than a bare top-level error) so callers can treat template
//     resolution failures and row-mapping failures uniformly as data, not exceptions.
//   - any other value (including "skip_invalid"): maps every row independently, accumulating
//     failures into the returned []RowMappingError and continuing, so a bad row does not
//     block the good rows in the same batch.
//
// A failure to resolve the template itself (missing binding, missing template, invalid
// mapping rules) is always a hard top-level error regardless of importMode.
func (s *TemplateMappingService) BuildImportPipelineWithMode(ctx context.Context, profileID uint, documentType string, rows []map[string]string, importMode string) (*domain.DocumentTemplate, []*domain.DemandLine, []RowMappingError, error) {
	// Row-level mapping warnings are not part of this legacy wrapper's return shape
	// (no production caller remains — see BuildDemandImportPipelineWithMode, which is
	// used directly by the dual-mode demand CSV import controller and does surface them).
	t, mapped, rowErrors, _, err := s.BuildDemandImportPipelineWithMode(ctx, profileID, documentType, rows, nil, nil, importMode)
	if err != nil {
		return nil, nil, nil, err
	}
	lines := make([]*domain.DemandLine, len(mapped))
	for i := range mapped {
		lines[i] = mapped[i].Line
	}
	return t, lines, rowErrors, nil
}

// BuildDemandImportPipelineWithMode is the v2-aware demand import pipeline.
// Prefer orderedRows+headers (from tabular.ReadTabularFile) when available so
// positional mapping works; otherwise fall back to header-keyed maps.
//
// Returns rich per-row results (line + document.* + recipient.*) so the controller
// can upsert addresses and fill document fields without re-applying mapping rules.
//
// The fourth return value is the deduplicated, row-prefixed set of non-blocking
// mapping warnings (e.g. mapping dests outside the global legal vocabulary)
// accumulated across every row — including rows that otherwise failed mapping
// with a hard error, since a dest-vocabulary warning and a required-field error
// are independent signals.
func (s *TemplateMappingService) BuildDemandImportPipelineWithMode(
	ctx context.Context,
	profileID uint,
	documentType string,
	headerRows []map[string]string,
	orderedRows [][]string,
	headers []string,
	importMode string,
) (*domain.DocumentTemplate, []DemandImportMappedRow, []RowMappingError, []string, error) {
	t, err := s.ResolveImportTemplate(ctx, profileID, documentType)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	rules, err := ParseMappingRules(t.MappingRules)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("template %s: %w", t.TemplateKey, err)
	}

	now := time.Now()
	var mapped []DemandImportMappedRow
	var rowErrors []RowMappingError
	var rowWarnings rowWarningCollector

	useOrdered := len(orderedRows) > 0
	n := len(headerRows)
	if useOrdered {
		n = len(orderedRows)
	}

	for i := 0; i < n; i++ {
		var row *DemandImportMappedRow
		var mapErr error
		var warnings []string
		if useOrdered {
			row, warnings, mapErr = MapDemandImportRow(orderedRows[i], headers, rules)
		} else {
			row, warnings, mapErr = MapDemandImportRowFromHeaderMap(headerRows[i], rules)
		}
		rowWarnings.add(i, warnings)
		if mapErr != nil {
			rowErrors = append(rowErrors, RowMappingError{RowIndex: i, Reason: mapErr.Error()})
			if importMode == "reject_all" {
				break
			}
			continue
		}
		if row.Line.CreatedAt.IsZero() {
			row.Line.CreatedAt = now
		}
		if row.Line.UpdatedAt.IsZero() {
			row.Line.UpdatedAt = now
		}
		mapped = append(mapped, *row)
	}

	return t, mapped, rowErrors, rowWarnings.warnings(), nil
}

// ResolveTemplateAndRules loads the default template binding and parses MappingRules.
// Shared by catalog / shipment / carrier import entry points.
func (s *TemplateMappingService) ResolveTemplateAndRules(ctx context.Context, profileID uint, documentType string) (*domain.DocumentTemplate, *TemplateMappingRules, error) {
	t, err := s.ResolveImportTemplate(ctx, profileID, documentType)
	if err != nil {
		return nil, nil, err
	}
	rules, err := ParseMappingRules(t.MappingRules)
	if err != nil {
		return nil, nil, fmt.Errorf("template %s: %w", t.TemplateKey, err)
	}
	return t, rules, nil
}
