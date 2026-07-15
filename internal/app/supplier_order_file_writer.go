package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// SupplierOrderFileWriterOptions carries optional repos used for template-driven
// export (csv/xlsx). When nil / unset, Generate falls back to the legacy JSON
// payload that embeds line IDs for reconciliation.
type SupplierOrderFileWriterOptions struct {
	FulfillRepo  domain.FulfillmentLineRepository
	ProductRepo  domain.ProductRepository
	AddressRepo  domain.CustomerAddressRepository
	TemplateRepo domain.DocumentTemplateRepository
}

type supplierOrderFileWriter struct {
	supplierRepo domain.SupplierOrderRepository
	outputDir    string
	opts         *SupplierOrderFileWriterOptions
}

// NewSupplierOrderFileWriter returns a SupplierOrderFileWriter that writes
// factory order files under outputDir (production wiring resolves this via
// service.ResolveExportsDir()). opts may be nil for the JSON-only path.
func NewSupplierOrderFileWriter(
	supplierRepo domain.SupplierOrderRepository,
	outputDir string,
	opts *SupplierOrderFileWriterOptions,
) SupplierOrderFileWriter {
	return &supplierOrderFileWriter{supplierRepo: supplierRepo, outputDir: outputDir, opts: opts}
}

// supplierOrderFilePayload is the on-disk shape of the generated factory
// file. FieldNames are deliberately explicit (not DTO reuse) since this is
// a persisted external-facing document contract, not an API response.
type supplierOrderFilePayload struct {
	SupplierOrderID  uint                           `json:"supplierOrderId"`
	WaveID           uint                           `json:"waveId"`
	SupplierPlatform string                         `json:"supplierPlatform"`
	BatchNo          string                         `json:"batchNo"`
	ExternalOrderNo  string                         `json:"externalOrderNo"`
	GeneratedAt      string                         `json:"generatedAt"`
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

	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return dto.SupplierOrderFileResultDTO{}, fmt.Errorf("supplier order %d file: create output directory %q: %w", orderID, w.outputDir, err)
	}

	// Prefer template-driven tabular export when the order carries a TemplateID
	// and MappingRules can be resolved. Headers come from the template column
	// mapping (no hardcoded bilibili/rouzao Chinese headers).
	if filePath, ok, renderErr := w.tryTemplateExport(ctx, order, lines, generatedAt); renderErr != nil {
		return dto.SupplierOrderFileResultDTO{}, renderErr
	} else if ok {
		return dto.SupplierOrderFileResultDTO{
			OrderID:     orderID,
			FilePath:    filePath,
			LineCount:   len(lines),
			GeneratedAt: generatedAt,
		}, nil
	}

	// Legacy JSON fallback: embeds line IDs for shipment-return reconciliation.
	// Surface a readable warning so callers know the template-driven path was skipped.
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
		Warnings: []string{
			"no export_supplier_order template bound or template export unavailable; wrote legacy JSON fallback",
		},
	}, nil
}

// tryTemplateExport attempts a MappingRules-driven csv/xlsx write.
// ok=false means "fall back to JSON" (no template / unsupported format).
func (w *supplierOrderFileWriter) tryTemplateExport(
	ctx context.Context,
	order *domain.SupplierOrder,
	lines []domain.SupplierOrderLine,
	generatedAt time.Time,
) (filePath string, ok bool, err error) {
	if w.opts == nil || w.opts.TemplateRepo == nil || order.TemplateID == "" {
		return "", false, nil
	}
	templateID, parseErr := strconv.ParseUint(order.TemplateID, 10, 64)
	if parseErr != nil || templateID == 0 {
		return "", false, nil
	}
	tmpl, err := w.opts.TemplateRepo.FindByID(ctx, uint(templateID))
	if err != nil || tmpl == nil || tmpl.MappingRules == "" {
		return "", false, nil
	}
	rules, err := ParseMappingRules(tmpl.MappingRules)
	if err != nil {
		// Malformed rules: fall back rather than hard-fail the whole download.
		return "", false, nil
	}

	format := strings.ToLower(strings.TrimSpace(tmpl.Format))
	if format == "" || format == "json" || format == "api_payload" {
		return "", false, nil
	}

	// Build optional context maps for export.* fields.
	linePtrs := make([]*domain.SupplierOrderLine, len(lines))
	fulfillMap := make(map[uint]*domain.FulfillmentLine, len(lines))
	productMap := make(map[uint]*domain.Product)
	addressMap := make(map[uint]*domain.CustomerAddress)

	for i := range lines {
		linePtrs[i] = &lines[i]
		if w.opts.FulfillRepo == nil {
			continue
		}
		fl, flErr := w.opts.FulfillRepo.FindByID(ctx, lines[i].FulfillmentLineID)
		if flErr != nil || fl == nil {
			continue
		}
		fulfillMap[fl.ID] = fl
		if fl.ProductID != nil && w.opts.ProductRepo != nil {
			if _, exists := productMap[*fl.ProductID]; !exists {
				if p, pErr := w.opts.ProductRepo.FindByID(ctx, *fl.ProductID); pErr == nil && p != nil {
					productMap[*fl.ProductID] = p
				}
			}
		}
		if fl.CustomerAddressID != nil && w.opts.AddressRepo != nil {
			if _, exists := addressMap[*fl.CustomerAddressID]; !exists {
				if a, aErr := w.opts.AddressRepo.FindByID(ctx, *fl.CustomerAddressID); aErr == nil && a != nil {
					addressMap[*fl.CustomerAddressID] = a
				}
			}
		}
	}

	renderer := NewTemplatePayloadRenderer()
	var (
		data     []byte
		ext      string
		filename string
	)
	switch format {
	case "csv":
		csvText, renderErr := renderer.RenderSupplierExportCSVWithContext(order, linePtrs, fulfillMap, productMap, addressMap, rules)
		if renderErr != nil {
			return "", false, fmt.Errorf("supplier order %d file: render csv: %w", order.ID, renderErr)
		}
		data = []byte(csvText)
		ext = "csv"
	case "xlsx":
		xlsxBytes, renderErr := renderer.RenderSupplierExportXLSX(order, linePtrs, fulfillMap, productMap, addressMap, rules)
		if renderErr != nil {
			return "", false, fmt.Errorf("supplier order %d file: render xlsx: %w", order.ID, renderErr)
		}
		data = xlsxBytes
		ext = "xlsx"
	default:
		// Unknown format → JSON fallback.
		return "", false, nil
	}

	filename = fmt.Sprintf("supplier_order_%d_%s.%s", order.ID, generatedAt.Format("20060102_150405"), ext)
	filePath = filepath.Join(w.outputDir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", false, fmt.Errorf("supplier order %d file: write file %q: %w", order.ID, filePath, err)
	}
	return filePath, true, nil
}
