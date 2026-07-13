package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// SupplierOrderFileWriter generates the downloadable factory order file
// (plan 3.3.4): file generation moved up to the factory step, and the file
// embeds each line's ID and batch number so a later shipment-return CSV can
// reconcile against it without requiring the factory to echo back our DB
// IDs verbatim.
type SupplierOrderFileWriter interface {
	// GenerateSupplierOrderFile writes the factory order file for orderID
	// under the resolved exports directory and returns its result (path +
	// line count).
	GenerateSupplierOrderFile(ctx context.Context, orderID uint) (dto.SupplierOrderFileResultDTO, error)
}

type supplierOrderFileWriter struct {
	supplierRepo domain.SupplierOrderRepository
	outputDir    string
}

// NewSupplierOrderFileWriter returns a SupplierOrderFileWriter that writes
// factory order files under outputDir (production wiring resolves this via
// service.ResolveExportsDir()).
func NewSupplierOrderFileWriter(supplierRepo domain.SupplierOrderRepository, outputDir string) SupplierOrderFileWriter {
	return &supplierOrderFileWriter{supplierRepo: supplierRepo, outputDir: outputDir}
}

// supplierOrderFilePayload is the on-disk shape of the generated factory
// file. FieldNames are deliberately explicit (not DTO reuse) since this is
// a persisted external-facing document contract, not an API response.
type supplierOrderFilePayload struct {
	SupplierOrderID  uint                          `json:"supplierOrderId"`
	WaveID           uint                          `json:"waveId"`
	SupplierPlatform string                        `json:"supplierPlatform"`
	BatchNo          string                        `json:"batchNo"`
	ExternalOrderNo  string                        `json:"externalOrderNo"`
	GeneratedAt      string                        `json:"generatedAt"`
	Lines            []supplierOrderFilePayloadLine `json:"lines"`
}

// supplierOrderFilePayloadLine embeds the reconciliation key described in
// plan 3.3.4: the line's own DB ID, plus the batch number + supplier line
// number combination, so a shipment-return CSV can match back to this line
// without the factory needing to know about our internal IDs.
type supplierOrderFilePayloadLine struct {
	LineID            uint   `json:"lineId"`
	BatchNo           string `json:"batchNo"`
	SupplierLineNo    int    `json:"supplierLineNo"`
	SupplierSKU       string `json:"supplierSku"`
	SubmittedQuantity int    `json:"submittedQuantity"`
}

func (w *supplierOrderFileWriter) GenerateSupplierOrderFile(ctx context.Context, orderID uint) (dto.SupplierOrderFileResultDTO, error) {
	if orderID == 0 {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("order_id is required")
	}

	order, err := w.supplierRepo.FindByID(ctx, orderID)
	if err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d lookup failed: %w", orderID, err)
	}
	if order == nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d not found", orderID)
	}

	lines, err := w.supplierRepo.ListLinesByOrder(ctx, orderID)
	if err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d lines lookup failed: %w", orderID, err)
	}

	generatedAt := time.Now()

	payload := supplierOrderFilePayload{
		SupplierOrderID:  order.ID,
		WaveID:           order.WaveID,
		SupplierPlatform: order.SupplierPlatform,
		BatchNo:          order.BatchNo,
		ExternalOrderNo:  order.ExternalOrderNo,
		GeneratedAt:      generatedAt.Format(time.RFC3339),
		Lines:            make([]supplierOrderFilePayloadLine, len(lines)),
	}
	for i, line := range lines {
		payload.Lines[i] = supplierOrderFilePayloadLine{
			LineID:            line.ID,
			BatchNo:           order.BatchNo,
			SupplierLineNo:    line.SupplierLineNo,
			SupplierSKU:       line.SupplierSKU,
			SubmittedQuantity: line.SubmittedQuantity,
		}
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d file: marshal payload: %w", orderID, err)
	}

	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d file: create output directory %q: %w", orderID, w.outputDir, err)
	}

	filename := fmt.Sprintf("supplier_order_%d_%s.json", orderID, generatedAt.Format("20060102_150405"))
	filePath := filepath.Join(w.outputDir, filename)

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d file: write file %q: %w", orderID, filePath, err)
	}

	return dto.SupplierOrderFileResultDTO{
		OrderID:     orderID,
		FilePath:    filePath,
		LineCount:   len(lines),
		GeneratedAt: generatedAt,
	}, nil
}
