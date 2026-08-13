package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/csvformula"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/xuri/excelize/v2"
)

// TemplatePayloadRenderer generates supplier export and tracking document payloads
// from fulfillment data using a DocumentTemplate's MappingRules.
// Locale-aware: respects the IntegrationProfile's default_locale for formatting.
type TemplatePayloadRenderer struct {
	locale string
	// carrierLookup maps an internal carrier code to the platform external code.
	// nil means passthrough (no translation). On miss, callers should return
	// ( "", false ) so the renderer emits the original code and records a warning.
	carrierLookup func(internal string) (external string, found bool)
	// warnings accumulates non-fatal notices from the last render (e.g. carrier miss).
	warnings []string
	// seenCarrierMisses de-dupes carrier-miss warnings within one render.
	seenCarrierMisses map[string]struct{}
}

func NewTemplatePayloadRenderer() *TemplatePayloadRenderer {
	return &TemplatePayloadRenderer{locale: "en_US"}
}

// WithLocale sets the locale for formatting (e.g. "zh_CN", "ja_JP").
func (r *TemplatePayloadRenderer) WithLocale(locale string) *TemplatePayloadRenderer {
	if locale != "" {
		r.locale = locale
	}
	return r
}

// WithCarrierLookup attaches an internal→external carrier translator for tracking
// exports. Returns a shallow copy so the original renderer stays reusable.
func (r *TemplatePayloadRenderer) WithCarrierLookup(fn func(internal string) (external string, found bool)) *TemplatePayloadRenderer {
	clone := *r
	clone.carrierLookup = fn
	clone.warnings = nil
	clone.seenCarrierMisses = nil
	return &clone
}

// Locale returns the current formatting locale.
func (r *TemplatePayloadRenderer) Locale() string {
	return r.locale
}

// Warnings returns non-fatal notices collected during the last render.
func (r *TemplatePayloadRenderer) Warnings() []string {
	if len(r.warnings) == 0 {
		return nil
	}
	out := make([]string, len(r.warnings))
	copy(out, r.warnings)
	return out
}

func (r *TemplatePayloadRenderer) noteCarrierMiss(internal string) {
	if r.seenCarrierMisses == nil {
		r.seenCarrierMisses = make(map[string]struct{})
	}
	if _, seen := r.seenCarrierMisses[internal]; seen {
		return
	}
	r.seenCarrierMisses[internal] = struct{}{}
	r.warnings = append(r.warnings, fmt.Sprintf(
		"carrier mapping miss for internal code %q; exporting as-is", internal,
	))
}

// ExportRowContext carries optional lookups used when resolving export.* fields.
type ExportRowContext struct {
	Line    *domain.SupplierOrderLine
	Fulfill *domain.FulfillmentLine
	Product *domain.Product
	Address *domain.CustomerAddress
}

// TrackingRowContext carries fields available for tracking export rows.
type TrackingRowContext struct {
	Item *domain.ChannelSyncItem
}

// RenderSupplierExportCSV produces a CSV payload from supplier order lines and
// the template's MappingRules. The rules define which fulfillment fields map to
// which output columns.
func (r *TemplatePayloadRenderer) RenderSupplierExportCSV(
	order *domain.SupplierOrder,
	lines []*domain.SupplierOrderLine,
	fulfillLines map[uint]*domain.FulfillmentLine,
	rules *TemplateMappingRules,
) (string, error) {
	return r.RenderSupplierExportCSVWithContext(order, lines, fulfillLines, nil, nil, rules)
}

// RenderSupplierExportCSVWithContext is like RenderSupplierExportCSV but also
// resolves product/address-backed export fields when the maps are provided.
func (r *TemplatePayloadRenderer) RenderSupplierExportCSVWithContext(
	_ *domain.SupplierOrder,
	lines []*domain.SupplierOrderLine,
	fulfillLines map[uint]*domain.FulfillmentLine,
	products map[uint]*domain.Product,
	addresses map[uint]*domain.CustomerAddress,
	rules *TemplateMappingRules,
) (string, error) {
	if len(lines) == 0 {
		return "", fmt.Errorf("no lines to export")
	}

	columns := r.orderedColumns(rules)
	if len(columns) == 0 {
		return "", fmt.Errorf("template has no export column mapping")
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.output
	}
	if err := w.Write(csvformula.SanitizeRow(headers)); err != nil {
		return "", err
	}

	for _, line := range lines {
		fl := fulfillLines[line.FulfillmentLineID]
		rowCtx := r.buildExportRowContext(line, fl, products, addresses)
		row, err := r.renderMappedRow(columns, rules, func(dest string) string {
			return r.exportFieldValue(rowCtx, dest)
		})
		if err != nil {
			return "", fmt.Errorf("supplier export line %d: %w", line.SupplierLineNo, err)
		}
		if err := w.Write(csvformula.SanitizeRow(row)); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
}

// RenderSupplierExportXLSX produces an xlsx workbook (bytes) using the same
// columnOrder-driven mapping as the CSV renderer.
func (r *TemplatePayloadRenderer) RenderSupplierExportXLSX(
	_ *domain.SupplierOrder,
	lines []*domain.SupplierOrderLine,
	fulfillLines map[uint]*domain.FulfillmentLine,
	products map[uint]*domain.Product,
	addresses map[uint]*domain.CustomerAddress,
	rules *TemplateMappingRules,
) ([]byte, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("no lines to export")
	}
	columns := r.orderedColumns(rules)
	if len(columns) == 0 {
		return nil, fmt.Errorf("template has no export column mapping")
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet, err := configureWorkbookSheet(f, rules)
	if err != nil {
		return nil, err
	}

	for i, col := range columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetCellValue(sheet, cell, col.output); err != nil {
			return nil, err
		}
	}
	for rowIdx, line := range lines {
		fl := fulfillLines[line.FulfillmentLineID]
		rowCtx := r.buildExportRowContext(line, fl, products, addresses)
		row, rowErr := r.renderMappedRow(columns, rules, func(dest string) string {
			return r.exportFieldValue(rowCtx, dest)
		})
		if rowErr != nil {
			return nil, fmt.Errorf("supplier export line %d: %w", line.SupplierLineNo, rowErr)
		}
		for colIdx := range columns {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return nil, err
			}
			if err := f.SetCellValue(sheet, cell, row[colIdx]); err != nil {
				return nil, err
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderSupplierExportJSON produces a JSON payload from supplier order lines.
func (r *TemplatePayloadRenderer) RenderSupplierExportJSON(
	order *domain.SupplierOrder,
	lines []*domain.SupplierOrderLine,
	fulfillLines map[uint]*domain.FulfillmentLine,
) (string, error) {
	type lineObj struct {
		SupplierLineNo int    `json:"supplierLineNo"`
		SupplierSKU    string `json:"supplierSku"`
		Quantity       int    `json:"quantity"`
		ProductRef     string `json:"productRef,omitempty"`
	}
	type payload struct {
		BatchNo         string    `json:"batchNo"`
		SupplierOrderID uint      `json:"supplierOrderId"`
		WaveID          uint      `json:"waveId"`
		Lines           []lineObj `json:"lines"`
	}

	p := payload{
		BatchNo:         order.BatchNo,
		SupplierOrderID: order.ID,
		WaveID:          order.WaveID,
		Lines:           make([]lineObj, len(lines)),
	}
	for i, line := range lines {
		productRef := ""
		if fl, ok := fulfillLines[line.FulfillmentLineID]; ok && fl.ProductID != nil {
			productRef = fmt.Sprintf("product-%d", *fl.ProductID)
		}
		p.Lines[i] = lineObj{
			SupplierLineNo: line.SupplierLineNo,
			SupplierSKU:    line.SupplierSKU,
			Quantity:       line.SubmittedQuantity,
			ProductRef:     productRef,
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RenderTrackingExportCSV produces a CSV payload for channel-sync tracking items
// using the template's MappingRules columnOrder.
func (r *TemplatePayloadRenderer) RenderTrackingExportCSV(
	items []domain.ChannelSyncItem,
	rules *TemplateMappingRules,
) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no tracking items to export")
	}
	columns := r.orderedColumns(rules)
	if len(columns) == 0 {
		return "", fmt.Errorf("template has no tracking column mapping")
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.output
	}
	if err := w.Write(csvformula.SanitizeRow(headers)); err != nil {
		return "", err
	}

	for i := range items {
		rowCtx := TrackingRowContext{Item: &items[i]}
		row, err := r.renderMappedRow(columns, rules, func(dest string) string {
			return r.trackingFieldValue(rowCtx, dest)
		})
		if err != nil {
			return "", fmt.Errorf("tracking export item %d: %w", items[i].ID, err)
		}
		if err := w.Write(csvformula.SanitizeRow(row)); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
}

// RenderTrackingExportXLSX produces an xlsx workbook for tracking items.
func (r *TemplatePayloadRenderer) RenderTrackingExportXLSX(
	items []domain.ChannelSyncItem,
	rules *TemplateMappingRules,
) ([]byte, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no tracking items to export")
	}
	columns := r.orderedColumns(rules)
	if len(columns) == 0 {
		return nil, fmt.Errorf("template has no tracking column mapping")
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet, err := configureWorkbookSheet(f, rules)
	if err != nil {
		return nil, err
	}

	for i, col := range columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetCellValue(sheet, cell, col.output); err != nil {
			return nil, err
		}
	}
	for rowIdx := range items {
		rowCtx := TrackingRowContext{Item: &items[rowIdx]}
		row, rowErr := r.renderMappedRow(columns, rules, func(dest string) string {
			return r.trackingFieldValue(rowCtx, dest)
		})
		if rowErr != nil {
			return nil, fmt.Errorf("tracking export item %d: %w", items[rowIdx].ID, rowErr)
		}
		for colIdx := range columns {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			if err != nil {
				return nil, err
			}
			if err := f.SetCellValue(sheet, cell, row[colIdx]); err != nil {
				return nil, err
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type exportColumn struct {
	src    string
	output string
}

// renderMappedRow applies the same transforms/defaults/required contract used
// by import ApplyRow to values produced by an export renderer. Keeping the two
// paths aligned prevents a template from validating successfully while its
// export-time transforms or required fields are silently ignored.
func (r *TemplatePayloadRenderer) renderMappedRow(
	columns []exportColumn,
	rules *TemplateMappingRules,
	value func(dest string) string,
) ([]string, error) {
	values := make(map[string]string, len(columns))
	for _, col := range columns {
		v, err := applyTransforms(value(col.src), rules.Transforms[col.src])
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", col.src, err)
		}
		values[col.src] = v
	}
	for dest, defaultValue := range rules.Defaults {
		v, err := applyTransforms(defaultValue, rules.Transforms[dest])
		if err != nil {
			return nil, fmt.Errorf("default %q: %w", dest, err)
		}
		values[dest] = v
	}
	for _, dest := range rules.Required {
		if strings.TrimSpace(values[dest]) == "" {
			return nil, fmt.Errorf("required field %q is missing or empty", dest)
		}
	}

	row := make([]string, len(columns))
	for i, col := range columns {
		row[i] = values[col.src]
	}
	return row, nil
}

func configureWorkbookSheet(f *excelize.File, rules *TemplateMappingRules) (string, error) {
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return "", fmt.Errorf("xlsx workbook has no default worksheet")
	}
	if rules == nil {
		return sheet, nil
	}
	desired := strings.TrimSpace(rules.SheetName)
	if desired == "" || desired == sheet {
		return sheet, nil
	}
	if err := f.SetSheetName(sheet, desired); err != nil {
		return "", fmt.Errorf("set xlsx sheet name %q: %w", desired, err)
	}
	return desired, nil
}

// orderedColumns returns export columns in a stable order.
// Prefer rules.ColumnOrder when present; otherwise sort dest keys alphabetically.
// Never relies on map range order.
func (r *TemplatePayloadRenderer) orderedColumns(rules *TemplateMappingRules) []exportColumn {
	if rules == nil || len(rules.Columns) == 0 {
		return nil
	}

	if len(rules.ColumnOrder) > 0 {
		cols := make([]exportColumn, 0, len(rules.ColumnOrder))
		seen := make(map[string]bool, len(rules.ColumnOrder))
		for _, dest := range rules.ColumnOrder {
			src, ok := rules.Columns[dest]
			if !ok || seen[dest] {
				continue
			}
			seen[dest] = true
			cols = append(cols, exportColumn{src: dest, output: src})
		}
		// Append any columns not listed in columnOrder, still alphabetically stable.
		if len(cols) < len(rules.Columns) {
			rest := make([]string, 0, len(rules.Columns)-len(cols))
			for dest := range rules.Columns {
				if !seen[dest] {
					rest = append(rest, dest)
				}
			}
			sort.Strings(rest)
			for _, dest := range rest {
				cols = append(cols, exportColumn{src: dest, output: rules.Columns[dest]})
			}
		}
		return cols
	}

	keys := make([]string, 0, len(rules.Columns))
	for dest := range rules.Columns {
		keys = append(keys, dest)
	}
	sort.Strings(keys)
	cols := make([]exportColumn, 0, len(keys))
	for _, dest := range keys {
		cols = append(cols, exportColumn{src: dest, output: rules.Columns[dest]})
	}
	return cols
}

func (r *TemplatePayloadRenderer) buildExportRowContext(
	line *domain.SupplierOrderLine,
	fl *domain.FulfillmentLine,
	products map[uint]*domain.Product,
	addresses map[uint]*domain.CustomerAddress,
) ExportRowContext {
	ctx := ExportRowContext{Line: line, Fulfill: fl}
	if fl == nil {
		return ctx
	}
	if fl.ProductID != nil && products != nil {
		ctx.Product = products[*fl.ProductID]
	}
	if fl.CustomerAddressID != nil && addresses != nil {
		ctx.Address = addresses[*fl.CustomerAddressID]
	}
	return ctx
}

// normalizeFieldKey strips known namespace prefixes so templates can use either
// "export.factory_sku" or bare "factory_sku" dest keys.
func normalizeFieldKey(field string) string {
	field = strings.TrimSpace(field)
	for _, prefix := range []string{"export.", "tracking.", "shipment.", "line."} {
		if strings.HasPrefix(field, prefix) {
			return strings.TrimPrefix(field, prefix)
		}
	}
	return field
}

func (r *TemplatePayloadRenderer) exportFieldValue(ctx ExportRowContext, field string) string {
	key := normalizeFieldKey(field)
	line := ctx.Line
	fl := ctx.Fulfill

	switch key {
	case "supplier_line_no":
		if line == nil {
			return ""
		}
		return strconv.Itoa(line.SupplierLineNo)
	case "supplier_sku", "factory_sku":
		// Prefer Product.FactorySKU when available; fall back to line snapshot.
		if ctx.Product != nil && ctx.Product.FactorySKU != "" {
			return ctx.Product.FactorySKU
		}
		if line != nil {
			return line.SupplierSKU
		}
		return ""
	case "quantity":
		if line == nil {
			return ""
		}
		return strconv.Itoa(line.SubmittedQuantity)
	case "product_id":
		if fl != nil && fl.ProductID != nil {
			return fmt.Sprintf("%d", *fl.ProductID)
		}
		return ""
	case "fulfillment_line_id", "third_party_order_no":
		// FL.ID is the stable third-party order number for factory exports.
		if fl != nil {
			return fmt.Sprintf("%d", fl.ID)
		}
		if line != nil {
			return fmt.Sprintf("%d", line.FulfillmentLineID)
		}
		return ""
	case "recipient", "recipient_name":
		if ctx.Address != nil {
			return ctx.Address.RecipientName
		}
		return ""
	case "phone":
		if ctx.Address != nil {
			return ctx.Address.Phone
		}
		return ""
	case "address":
		return formatCustomerAddress(ctx.Address)
	default:
		return ""
	}
}

func (r *TemplatePayloadRenderer) trackingFieldValue(ctx TrackingRowContext, field string) string {
	key := normalizeFieldKey(field)
	if ctx.Item == nil {
		return ""
	}
	it := ctx.Item
	switch key {
	case "item_id":
		return fmt.Sprintf("%d", it.ID)
	case "fulfillment_line_id", "third_party_order_no":
		return fmt.Sprintf("%d", it.FulfillmentLineID)
	case "shipment_id":
		return fmt.Sprintf("%d", it.ShipmentID)
	case "tracking_no":
		return it.TrackingNo
	case "carrier_code":
		code := it.CarrierCode
		if r.carrierLookup != nil && strings.TrimSpace(code) != "" {
			if ext, ok := r.carrierLookup(code); ok && ext != "" {
				return ext
			}
			r.noteCarrierMiss(code)
		}
		return code
	case "external_document_no":
		return it.ExternalDocumentNo
	case "external_line_no":
		return it.ExternalLineNo
	default:
		return ""
	}
}

// fieldValue is retained for older call sites / tests that only have line+fulfillment.
func (r *TemplatePayloadRenderer) fieldValue(line *domain.SupplierOrderLine, fl *domain.FulfillmentLine, field string) string {
	return r.exportFieldValue(ExportRowContext{Line: line, Fulfill: fl}, field)
}

func formatCustomerAddress(addr *domain.CustomerAddress) string {
	if addr == nil {
		return ""
	}
	parts := make([]string, 0, 7)
	for _, p := range []string{
		addr.Country,
		addr.Province,
		addr.City,
		addr.District,
		addr.AddressLine1,
		addr.AddressLine2,
		addr.PostalCode,
	} {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}
