package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// documentExportExecutor is the first real ChannelSyncExecutor.
// It serialises a ChannelSyncJob + Items into a structured JSON file
// under a configurable output directory, or a template-driven tabular
// tracking file when the profile binds export_source_tracking_update.
type documentExportExecutor struct {
	outputDir string
	templates *TrackingTemplateSource
}

// NewDocumentExportExecutor returns a CapableExecutor for the document_export
// tracking_sync_mode under the "eli.local_export" connector key.
// templates may be nil — fallback JSON payload is used when no default
// export_source_tracking_update template is bound to the profile.
func NewDocumentExportExecutor(outputDir string, templates *TrackingTemplateSource) CapableExecutor {
	return &documentExportExecutor{outputDir: outputDir, templates: templates}
}

// ConnectorKey implements CapableExecutor.
func (e *documentExportExecutor) ConnectorKey() string {
	return "eli.local_export"
}

// Capabilities implements CapableExecutor.
func (e *documentExportExecutor) Capabilities() ConnectorCapabilities {
	return ConnectorCapabilities{
		SupportsTrackingPush:    true,
		SupportsOrderExport:     true,
		SupportsStatusQuery:     false,
		RequiresCarrierMapping:  false,
		RequiresExternalOrderNo: false,
		SupportedDirections:     []string{"push_tracking"},
	}
}

type exportPayload struct {
	JobID                uint                `json:"job_id"`
	WaveID               uint                `json:"wave_id"`
	IntegrationProfileID uint                `json:"integration_profile_id"`
	Direction            string              `json:"direction"`
	GeneratedAt          string              `json:"generated_at"`
	Items                []exportPayloadItem `json:"items"`
}

type exportPayloadItem struct {
	ItemID             uint   `json:"item_id"`
	FulfillmentLineID  uint   `json:"fulfillment_line_id"`
	ShipmentID         uint   `json:"shipment_id"`
	ExternalDocumentNo string `json:"external_document_no"`
	ExternalLineNo     string `json:"external_line_no"`
	TrackingNo         string `json:"tracking_no"`
	CarrierCode        string `json:"carrier_code"`
}

func (e *documentExportExecutor) Execute(
	ctx context.Context,
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (*ChannelSyncExecutionResult, error) {
	if err := ValidateProfileDocumentType(profile, "export_source_tracking_update"); err != nil {
		return nil, fmt.Errorf("document_export: %w", err)
	}
	generatedAt := time.Now().Format(time.RFC3339)

	if err := os.MkdirAll(e.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("document_export: create output directory %q for job %d: %w", e.outputDir, job.ID, err)
	}

	// Prefer template-driven columns when the profile binds a default tracking template.
	if filePath, data, warnings, ok, err := e.tryTemplateExport(ctx, job, items, profile); err != nil {
		return nil, err
	} else if ok {
		return documentSuccessResult(items, filePath, string(data), generatedAt, warnings), nil
	}

	payload := exportPayload{
		JobID:                job.ID,
		WaveID:               job.WaveID,
		IntegrationProfileID: job.IntegrationProfileID,
		Direction:            job.Direction,
		GeneratedAt:          generatedAt,
		Items:                make([]exportPayloadItem, len(items)),
	}
	for i, it := range items {
		payload.Items[i] = exportPayloadItem{
			ItemID:             it.ID,
			FulfillmentLineID:  it.FulfillmentLineID,
			ShipmentID:         it.ShipmentID,
			ExternalDocumentNo: it.ExternalDocumentNo,
			ExternalLineNo:     it.ExternalLineNo,
			TrackingNo:         it.TrackingNo,
			CarrierCode:        it.CarrierCode,
		}
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("document_export: marshal payload for job %d: %w", job.ID, err)
	}

	filename := fmt.Sprintf("channel_sync_%d_%s.json", job.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(e.outputDir, filename)

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("document_export: write file %q for job %d: %w", filePath, job.ID, err)
	}

	return documentSuccessResult(items, filePath, string(data), generatedAt), nil
}

func (e *documentExportExecutor) tryTemplateExport(
	ctx context.Context,
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (filePath string, data []byte, warnings []string, ok bool, err error) {
	tmpl, rules, found := resolveTrackingTemplate(ctx, e.templates, profile)
	if !found {
		return "", nil, nil, false, nil
	}

	format := strings.ToLower(strings.TrimSpace(tmpl.Format))
	renderer := newTrackingRenderer(ctx, e.templates, profile)
	var ext string
	switch format {
	case "csv":
		csvText, renderErr := renderer.RenderTrackingExportCSV(items, rules)
		if renderErr != nil {
			return "", nil, nil, false, fmt.Errorf("document_export: render tracking csv: %w", renderErr)
		}
		data = []byte(csvText)
		ext = "csv"
	case "xlsx":
		xlsxBytes, renderErr := renderer.RenderTrackingExportXLSX(items, rules)
		if renderErr != nil {
			return "", nil, nil, false, fmt.Errorf("document_export: render tracking xlsx: %w", renderErr)
		}
		data = xlsxBytes
		ext = "xlsx"
	case "xls":
		return "", nil, nil, false, fmt.Errorf("document_export: BIFF .xls output is not supported; update tracking template %d to format xlsx", tmpl.ID)
	default:
		// JSON / api_payload / empty → keep the structured JSON fallback.
		return "", nil, nil, false, nil
	}

	filename := fmt.Sprintf("channel_sync_%d_%s.%s", job.ID, time.Now().Format("20060102_150405"), ext)
	filePath = filepath.Join(e.outputDir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", nil, nil, false, fmt.Errorf("document_export: write file %q for job %d: %w", filePath, job.ID, err)
	}
	return filePath, data, renderer.Warnings(), true, nil
}

func documentSuccessResult(items []domain.ChannelSyncItem, filePath, requestPayload, generatedAt string, warnings ...[]string) *ChannelSyncExecutionResult {
	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{
			ItemID: it.ID,
			Status: "success",
		}
	}
	resp := map[string]any{
		"status":       "ok",
		"output_file":  filePath,
		"item_count":   len(items),
		"generated_at": generatedAt,
	}
	if len(warnings) > 0 && len(warnings[0]) > 0 {
		resp["warnings"] = warnings[0]
	}
	respBytes, _ := json.Marshal(resp)
	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "success",
		RequestPayload:  requestPayload,
		ResponsePayload: string(respBytes),
	}
}
