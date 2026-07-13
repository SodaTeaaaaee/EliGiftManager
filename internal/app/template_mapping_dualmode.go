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
	t, err := s.ResolveImportTemplate(ctx, profileID, documentType)
	if err != nil {
		return nil, nil, nil, err
	}
	rules, err := ParseMappingRules(t.MappingRules)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("template %s: %w", t.TemplateKey, err)
	}

	now := time.Now()
	lines := make([]*domain.DemandLine, 0, len(rows))
	var rowErrors []RowMappingError
	for i, row := range rows {
		line, mapErr := MapCSVRowToDemandLine(row, rules)
		if mapErr != nil {
			rowErrors = append(rowErrors, RowMappingError{RowIndex: i, Reason: mapErr.Error()})
			if importMode == "reject_all" {
				break
			}
			continue
		}
		if line.CreatedAt.IsZero() {
			line.CreatedAt = now
		}
		if line.UpdatedAt.IsZero() {
			line.UpdatedAt = now
		}
		lines = append(lines, line)
	}

	return t, lines, rowErrors, nil
}
