package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// csvExportExecutor is a second real ChannelSyncExecutor that produces CSV
// tracking update files. It demonstrates the connector onboarding pattern
// beyond the initial local_export executor.
type csvExportExecutor struct {
	outputDir string
}

// NewCSVExportExecutor returns a CapableExecutor for the "eli.csv_export"
// connector key under the document_export tracking_sync_mode.
func NewCSVExportExecutor(outputDir string) CapableExecutor {
	return &csvExportExecutor{outputDir: outputDir}
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
	job *domain.ChannelSyncJob,
	items []domain.ChannelSyncItem,
	profile *domain.IntegrationProfile,
) (*ChannelSyncExecutionResult, error) {
	generatedAt := time.Now().Format(time.RFC3339)

	if err := os.MkdirAll(e.outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("csv_export: create output dir %q: %w", e.outputDir, err)
	}

	filename := fmt.Sprintf("tracking_update_%d_%s.csv", job.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(e.outputDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("csv_export: create file %q: %w", filePath, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"fulfillment_line_id", "shipment_id", "tracking_no", "carrier_code", "external_document_no", "external_line_no"}); err != nil {
		return nil, fmt.Errorf("csv_export: write header: %w", err)
	}

	for _, it := range items {
		if err := w.Write([]string{
			fmt.Sprintf("%d", it.FulfillmentLineID),
			fmt.Sprintf("%d", it.ShipmentID),
			it.TrackingNo,
			it.CarrierCode,
			it.ExternalDocumentNo,
			it.ExternalLineNo,
		}); err != nil {
			return nil, fmt.Errorf("csv_export: write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv_export: flush: %w", err)
	}

	results := make([]ChannelSyncItemResult, len(items))
	for i, it := range items {
		results[i] = ChannelSyncItemResult{ItemID: it.ID, Status: "success"}
	}

	resp, _ := json.Marshal(map[string]any{
		"status":       "ok",
		"output_file":  filePath,
		"format":       "csv",
		"item_count":   len(items),
		"generated_at": generatedAt,
	})

	return &ChannelSyncExecutionResult{
		Items:           results,
		AggregateStatus: "success",
		RequestPayload:  filePath,
		ResponsePayload: string(resp),
	}, nil
}
