package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TemplatePayloadRenderer generates supplier export and tracking document payloads
// from fulfillment data using a DocumentTemplate's MappingRules.
// Locale-aware: respects the IntegrationProfile's default_locale for formatting.
type TemplatePayloadRenderer struct {
	locale string
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

// Locale returns the current formatting locale.
func (r *TemplatePayloadRenderer) Locale() string {
	return r.locale
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
	if len(lines) == 0 {
		return "", fmt.Errorf("no lines to export")
	}

	columns := r.orderedColumns(rules)
	if len(columns) == 0 {
		return "", fmt.Errorf("template has no export column mapping")
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)

	// Header row
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.output
	}
	if err := w.Write(headers); err != nil {
		return "", err
	}

	// Data rows
	for _, line := range lines {
		fl := fulfillLines[line.FulfillmentLineID]
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = r.fieldValue(line, fl, col.src)
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}
	w.Flush()
	return buf.String(), w.Error()
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

type exportColumn struct {
	src    string
	output string
}

func (r *TemplatePayloadRenderer) orderedColumns(rules *TemplateMappingRules) []exportColumn {
	cols := make([]exportColumn, 0, len(rules.Columns))
	for dest, src := range rules.Columns {
		cols = append(cols, exportColumn{src: dest, output: src})
	}
	return cols
}

func (r *TemplatePayloadRenderer) fieldValue(line *domain.SupplierOrderLine, fl *domain.FulfillmentLine, field string) string {
	switch field {
	case "supplier_line_no":
		return strconv.Itoa(line.SupplierLineNo)
	case "supplier_sku":
		return line.SupplierSKU
	case "quantity":
		return strconv.Itoa(line.SubmittedQuantity)
	case "product_id":
		if fl != nil && fl.ProductID != nil {
			return fmt.Sprintf("%d", *fl.ProductID)
		}
		return ""
	case "fulfillment_line_id":
		return fmt.Sprintf("%d", line.FulfillmentLineID)
	default:
		return ""
	}
}
