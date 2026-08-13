package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/csvformula"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TrackingTemplateSource supplies optional profile→template resolution for
// tracking document exports. When nil (or repos missing), executors keep their
// hardcoded JSON/CSV fallback formats.
type TrackingTemplateSource struct {
	BindingRepo  domain.ProfileTemplateBindingRepository
	TemplateRepo domain.DocumentTemplateRepository
	// CarrierUC is optional; when set, export.carrier_code / tracking.carrier_code
	// are translated internal→external via ResolveCarrier (miss → warning + passthrough).
	CarrierUC CarrierMappingUseCase
}

// csvExportExecutor is a second real ChannelSyncExecutor that produces CSV
// tracking update files. It demonstrates the connector onboarding pattern
// beyond the initial local_export executor.
type csvExportExecutor struct {
	outputDir string
	templates *TrackingTemplateSource
}

// NewCSVExportExecutor returns a CapableExecutor for the "eli.csv_export"
// connector key under the document_export tracking_sync_mode.
// templates may be nil — fallback CSV columns are used when no default
// export_source_tracking_update template is bound to the profile.
func NewCSVExportExecutor(outputDir string, templates *TrackingTemplateSource) CapableExecutor {
	return &csvExportExecutor{outputDir: outputDir, templates: templates}
}

func (e *csvExportExecutor) ConnectorKey() string {
	return "eli.csv_export"
}

func (e *csvExportExecutor) Capabilities() ConnectorCapabilities {
	return ConnectorCapabilities{
		SupportsTrackingPush:    true,
		SupportsOrderExport:     false,
		SupportsStatusQuery:     false,
		RequiresCarrierMapping:  true,
		RequiresExternalOrderNo: false,
		SupportedDirections:     []string{"push_tracking"},
	}
}

func (e *csvExportExecutor) Execute(
	ctx context.Context,
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (*ChannelSyncExecutionResult, error) {
	if err := ValidateProfileDocumentType(profile, "export_source_tracking_update"); err != nil {
		return nil, fmt.Errorf("csv_export: %w", err)
	}
	generatedAt := time.Now().Format(time.RFC3339)

	if err := os.MkdirAll(e.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("csv_export: create output dir %q: %w", e.outputDir, err)
	}

	// Prefer template-driven columns when the profile binds a default tracking template.
	if filePath, warnings, ok, err := e.tryTemplateExport(ctx, job, items, profile); err != nil {
		return nil, err
	} else if ok {
		format := "csv"
		if strings.HasSuffix(strings.ToLower(filePath), ".xlsx") {
			format = "xlsx"
		}
		return trackingSuccessResult(items, filePath, format, generatedAt, warnings), nil
	}

	filename := fmt.Sprintf("tracking_update_%d_%s.csv", job.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(e.outputDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("csv_export: create file %q: %w", filePath, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(csvformula.SanitizeRow([]string{"fulfillment_line_id", "shipment_id", "tracking_no", "carrier_code", "external_document_no", "external_line_no"})); err != nil {
		return nil, fmt.Errorf("csv_export: write header: %w", err)
	}

	for _, it := range items {
		if err := w.Write(csvformula.SanitizeRow([]string{
			fmt.Sprintf("%d", it.FulfillmentLineID),
			fmt.Sprintf("%d", it.ShipmentID),
			it.TrackingNo,
			it.CarrierCode,
			it.ExternalDocumentNo,
			it.ExternalLineNo,
		})); err != nil {
			return nil, fmt.Errorf("csv_export: write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv_export: flush: %w", err)
	}

	return trackingSuccessResult(items, filePath, "csv", generatedAt), nil
}

func (e *csvExportExecutor) tryTemplateExport(
	ctx context.Context,
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (filePath string, warnings []string, ok bool, err error) {
	tmpl, rules, found := resolveTrackingTemplate(ctx, e.templates, profile)
	if !found {
		return "", nil, false, nil
	}

	format := strings.ToLower(strings.TrimSpace(tmpl.Format))
	renderer := newTrackingRenderer(ctx, e.templates, profile)
	var (
		data []byte
		ext  string
	)
	switch format {
	case "", "csv":
		csvText, renderErr := renderer.RenderTrackingExportCSV(items, rules)
		if renderErr != nil {
			return "", nil, false, fmt.Errorf("csv_export: render tracking template: %w", renderErr)
		}
		data = []byte(csvText)
		ext = "csv"
	case "xlsx":
		xlsxBytes, renderErr := renderer.RenderTrackingExportXLSX(items, rules)
		if renderErr != nil {
			return "", nil, false, fmt.Errorf("csv_export: render tracking xlsx: %w", renderErr)
		}
		data = xlsxBytes
		ext = "xlsx"
	case "xls":
		return "", nil, false, fmt.Errorf("csv_export: BIFF .xls output is not supported; update tracking template %d to format xlsx", tmpl.ID)
	default:
		// Non-tabular tracking template formats fall through to hardcoded CSV.
		return "", nil, false, nil
	}

	filename := fmt.Sprintf("tracking_update_%d_%s.%s", job.ID, time.Now().Format("20060102_150405"), ext)
	filePath = filepath.Join(e.outputDir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", nil, false, fmt.Errorf("csv_export: write file %q: %w", filePath, err)
	}
	return filePath, renderer.Warnings(), true, nil
}

// newTrackingRenderer builds a TemplatePayloadRenderer with optional carrier
// internal→external translation for the given profile.
func newTrackingRenderer(
	ctx context.Context,
	src *TrackingTemplateSource,
	profile *domain.IntegrationProfile,
) *TemplatePayloadRenderer {
	renderer := NewTemplatePayloadRenderer()
	if src == nil || src.CarrierUC == nil || profile == nil || profile.ID == 0 {
		return renderer
	}
	profileID := profile.ID
	carrierUC := src.CarrierUC
	return renderer.WithCarrierLookup(func(internal string) (string, bool) {
		ext, _, err := carrierUC.ResolveCarrier(ctx, profileID, internal)
		if err != nil || ext == "" {
			return "", false
		}
		return ext, true
	})
}

func resolveTrackingTemplate(
	ctx context.Context,
	src *TrackingTemplateSource,
	profile *domain.IntegrationProfile,
) (*domain.DocumentTemplate, *TemplateMappingRules, bool) {
	if src == nil || src.BindingRepo == nil || src.TemplateRepo == nil || profile == nil || profile.ID == 0 {
		return nil, nil, false
	}
	binding, err := src.BindingRepo.FindDefaultByProfileAndType(ctx, profile.ID, "export_source_tracking_update")
	if err != nil || binding == nil {
		return nil, nil, false
	}
	tmpl, err := src.TemplateRepo.FindByID(ctx, binding.TemplateID)
	if err != nil || tmpl == nil || tmpl.MappingRules == "" {
		return nil, nil, false
	}
	rules, err := ParseMappingRules(tmpl.MappingRules)
	if err != nil {
		return nil, nil, false
	}
	return tmpl, rules, true
}

func trackingSuccessResult(items []domain.ChannelSyncItem, filePath, format, generatedAt string, warnings ...[]string) *ChannelSyncExecutionResult {
	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{ItemID: it.ID, Status: "success"}
	}
	respMap := map[string]any{
		"status":       "ok",
		"output_file":  filePath,
		"format":       format,
		"item_count":   len(items),
		"generated_at": generatedAt,
	}
	if len(warnings) > 0 && len(warnings[0]) > 0 {
		respMap["warnings"] = warnings[0]
	}
	resp, _ := json.Marshal(respMap)
	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "success",
		RequestPayload:  filePath,
		ResponsePayload: string(resp),
	}
}
