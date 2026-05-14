package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// documentExportExecutor is the first real ChannelSyncExecutor.
// It serialises a ChannelSyncJob + Items into a structured JSON file
// under a configurable output directory.
type documentExportExecutor struct {
	outputDir string
}

// NewDocumentExportExecutor returns a real executor for the document_export
// tracking_sync_mode. outputDir is resolved by the production wiring layer
// (e.g. data/exports/ under the app data directory).
func NewDocumentExportExecutor(outputDir string) ChannelSyncExecutor {
	return &documentExportExecutor{outputDir: outputDir}
}

type exportPayload struct {
	JobID               uint               `json:"job_id"`
	WaveID              uint               `json:"wave_id"`
	IntegrationProfileID uint               `json:"integration_profile_id"`
	Direction           string             `json:"direction"`
	GeneratedAt         string             `json:"generated_at"`
	Items               []exportPayloadItem `json:"items"`
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
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (*ChannelSyncExecutionResult, error) {
	generatedAt := time.Now().Format(time.RFC3339)

	payload := exportPayload{
		JobID:               job.ID,
		WaveID:              job.WaveID,
		IntegrationProfileID: job.IntegrationProfileID,
		Direction:           job.Direction,
		GeneratedAt:         generatedAt,
		Items:               make([]exportPayloadItem, len(items)),
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

	if err := os.MkdirAll(e.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("document_export: create output directory %q for job %d: %w", e.outputDir, job.ID, err)
	}

	filename := fmt.Sprintf("channel_sync_%d_%s.json", job.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(e.outputDir, filename)

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("document_export: write file %q for job %d: %w", filePath, job.ID, err)
	}

	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{
			ItemID: it.ID,
			Status: "success",
		}
	}

	resp := map[string]interface{}{
		"status":       "ok",
		"output_file":  filePath,
		"item_count":   len(items),
		"generated_at": generatedAt,
	}
	respBytes, _ := json.Marshal(resp)

	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "success",
		RequestPayload:  string(data),
		ResponsePayload: string(respBytes),
	}, nil
}
