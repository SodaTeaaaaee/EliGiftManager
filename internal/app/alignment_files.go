package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// Document type conventions used by the built-in templates and the file use
// cases. They ride on TemplateConfig.DocumentType.
const (
	DocumentTypeMembershipList = "membership_list"
	DocumentTypeOrderExport    = "order_export"
	DocumentTypeShipmentReturn = "shipment_return"
	DocumentTypeFactoryOrder   = "factory_order"
	DocumentTypeWriteback      = "writeback"
)

// DefaultFactoryOrderLayout is the rozao-shaped six-column order sheet used
// when the factory platform has no output template yet.
func DefaultFactoryOrderLayout() alignment.LayoutConfig {
	return alignment.LayoutConfig{
		Version:     alignment.LayoutSchemaVersion,
		Format:      alignment.FormatCSV,
		ColumnOrder: []string{"tracking.id", "recipient.name", "recipient.phone", "recipient.address_line1", "product.factory_sku", "quantity"},
		HeaderNames: map[string]string{
			"tracking.id":             "第三方订单号",
			"recipient.name":          "收件人",
			"recipient.phone":         "联系电话",
			"recipient.address_line1": "收件地址",
			"product.factory_sku":     "商家编码",
			"quantity":                "下单数量",
		},
	}
}

// DefaultShipmentReturnMapping is the rouzao 13-column shipment-return CSV
// used when the factory platform has no input template yet.
func DefaultShipmentReturnMapping() alignment.MappingConfig {
	return alignment.MappingConfig{
		Version:   alignment.MappingSchemaVersion,
		Mode:      alignment.ModeHeader,
		SheetName: "",
		Columns: map[string]string{
			"tracking.id":             "订单编号",
			"source.created_at":       "下单时间",
			"product.alias_id":        "商品编码",
			"product.alias_title":     "商品名称",
			"product.alias_spec":      "规格&数量",
			"recipient.name":          "收件人",
			"recipient.phone":         "电话",
			"recipient.address_line1": "收件信息",
			"shipment.carrier_name":   "物流公司",
			"shipment.tracking_no":    "物流单号",
			"shipment.shipped_at":     "打印快递时间",
			"shipment.quantity":       "规格&数量",
		},
		Transforms: map[string][]string{
			"tracking.id":          {"trim", "strip_quotes"},
			"shipment.tracking_no": {"trim", "strip_quotes"},
			"shipment.shipped_at":  {"parseDate"},
		},
		Required:    []string{"tracking.id", "shipment.tracking_no"},
		Fingerprint: []string{"tracking.id", "shipment.tracking_no"},
	}
}

// findTemplate returns the newest template for a platform, direction, and
// document type, or nil when none is configured.
func (ws *Workspace) findTemplate(ctx context.Context, platformID uint, direction domain.TemplateDirection, documentType string) *domain.TemplateConfig {
	templates, err := ws.Store.ListTemplates(ctx)
	if err != nil {
		return nil
	}
	var best *domain.TemplateConfig
	for i := range templates {
		t := templates[i]
		if t.PlatformID != platformID || t.Direction != string(direction) || t.DocumentType != documentType {
			continue
		}
		if best == nil || t.Version > best.Version {
			best = &templates[i]
		}
	}
	return best
}

func factKindForDocumentType(documentType string) string {
	switch documentType {
	case DocumentTypeMembershipList, string(domain.InputFactKindMembership):
		return string(domain.InputFactKindMembership)
	case string(domain.InputFactKindOperatorGrant):
		return string(domain.InputFactKindOperatorGrant)
	default:
		return string(domain.InputFactKindRetailOrder)
	}
}

// ImportFileResult reports what a file import produced: the document, the
// facts and lines that landed, duplicate observations, and parse issues.
type ImportFileResult struct {
	Document     domain.InputDocument
	FactsCreated int
	LinesCreated int
	Duplicates   []domain.DuplicateObservation
	Issues       []alignment.ParseIssue
}

// ImportFile reads a platform export file, parses it through the template's
// mapping config, and ingests the produced facts in one transaction.
func (ws *Workspace) ImportFile(ctx context.Context, platformID, tplID uint, filePath string) (*ImportFileResult, error) {
	tpl, err := ws.Store.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, err
	}
	if tpl.PlatformID != platformID {
		return nil, fmt.Errorf("import file: template %d does not belong to platform %d", tplID, platformID)
	}
	mapping, err := alignment.ParseMappingConfig(tpl.MappingJSON)
	if err != nil {
		return nil, fmt.Errorf("import file: template %q: %w", tpl.Name, err)
	}
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
	}
	kind := factKindForDocumentType(tpl.DocumentType)
	rows, issues, err := alignment.Parse(data, format, alignment.TemplateSpec{Mapping: mapping, Kind: kind})
	if err != nil {
		return nil, err
	}

	doc := &domain.InputDocument{
		PlatformID:      platformID,
		DocumentType:    tpl.DocumentType,
		Direction:       tpl.Direction,
		OriginalName:    filepath.Base(filePath),
		RawPayload:      string(data),
		TemplateID:      &tplID,
		TemplateVersion: tpl.Version,
	}
	facts := buildIngestFacts(kind, rows)

	result := &ImportFileResult{Issues: issues}
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		outDoc, dups, err := tws.ingestDocument(ctx, tx, doc, facts)
		if err != nil {
			return err
		}
		result.Document = *outDoc
		result.Duplicates = dups
		created, err := tx.ListFactsByDocument(ctx, outDoc.ID)
		if err != nil {
			return err
		}
		result.FactsCreated = len(created)
		for _, f := range created {
			lines, err := tx.ListFactLines(ctx, f.ID)
			if err != nil {
				return err
			}
			result.LinesCreated += len(lines)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if result.Duplicates == nil {
		result.Duplicates = []domain.DuplicateObservation{}
	}
	if result.Issues == nil {
		result.Issues = []alignment.ParseIssue{}
	}
	return result, nil
}

// buildIngestFacts groups parsed rows back into facts: one source row is one
// fact, and multi-SKU expansions become that fact's lines.
func buildIngestFacts(kind string, rows []alignment.ParsedRow) []IngestFactInput {
	bySource := map[int]*IngestFactInput{}
	order := []int{}
	for _, row := range rows {
		in, ok := bySource[row.SourceRow]
		if !ok {
			in = &IngestFactInput{
				Kind:            kind,
				IdentityType:    row.Values["identity.type"],
				IdentityValue:   row.Values["identity.value"],
				MembershipLevel: row.Values["membership.level"],
			}
			if docNo := row.Values["source.document_no"]; docNo != "" {
				in.SourceDocumentNo = docNo
				in.StableExternalID = docNo
			} else {
				in.StableExternalID = "fp:" + row.Fingerprint
			}
			if raw := row.Values["source.created_at"]; raw != "" {
				if t, ok := parseFlexibleTime(raw); ok {
					in.SourceCreatedAt = &t
				}
			}
			if extra := recipientExtraData(row.Values); extra != "" {
				in.ExtraData = extra
			}
			bySource[row.SourceRow] = in
			order = append(order, row.SourceRow)
		}
		qty := 1
		if n, err := strconv.Atoi(strings.TrimSpace(row.Values["quantity"])); err == nil && n > 0 {
			qty = n
		}
		in.Lines = append(in.Lines, IngestLine{
			SourceLineNo:  row.LineNo,
			ExternalSKU:   row.Values["product.alias_id"],
			ExternalTitle: row.Values["product.alias_title"],
			ExternalSpec:  row.Values["product.alias_spec"],
			Quantity:      qty,
		})
	}
	facts := make([]IngestFactInput, 0, len(order))
	for _, src := range order {
		facts = append(facts, *bySource[src])
	}
	return facts
}

// recipientExtraData snapshots recipient-ish keys onto the fact's extra data
// so imported addresses survive until the address-snapshot linkage lands.
func recipientExtraData(values map[string]string) string {
	snap := domain.AddressSnapshot{
		RecipientName: values["recipient.name"],
		Phone:         values["recipient.phone"],
		Country:       values["recipient.country"],
		Province:      values["recipient.province"],
		City:          values["recipient.city"],
		District:      values["recipient.district"],
		AddressLine1:  values["recipient.address_line1"],
		AddressLine2:  values["recipient.address_line2"],
		PostalCode:    values["recipient.postal_code"],
	}
	if snap.RecipientName == "" && snap.Phone == "" && snap.AddressLine1 == "" && snap.Province == "" && snap.City == "" {
		return ""
	}
	b, err := json.Marshal(map[string]domain.AddressSnapshot{"recipient": snap})
	if err != nil {
		return ""
	}
	return string(b)
}

func parseFlexibleTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ExportFileResult reports a rendered export: the updated order, the absolute
// file path, and the rows that were written.
type ExportFileResult struct {
	Order domain.SupplierOrder
	Path  string
	Rows  []map[string]string
}

// ExportFactoryOrderFile renders a generated supplier order into the factory
// platform's order-import file (one row per order line and distinct frozen
// address), stores the payload, and advances the order to exported. It is the
// file-producing layer on top of the generate-then-export state machine.
func (ws *Workspace) ExportFactoryOrderFile(ctx context.Context, orderID uint) (*ExportFileResult, error) {
	order, err := ws.Store.GetSupplierOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.Status == string(domain.SupplierOrderVoided) {
		return nil, ErrOrderNotGenerated
	}
	lines, err := ws.Store.ListSupplierOrderLines(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	layout := DefaultFactoryOrderLayout()
	var tplID uint
	var tplVersion int
	if tpl := ws.findTemplate(ctx, order.FactoryPlatformID, domain.TemplateDirectionOutput, DocumentTypeFactoryOrder); tpl != nil {
		if parsed, err := alignment.ParseLayoutConfig(tpl.LayoutJSON); err == nil && len(parsed.ColumnOrder) > 0 {
			layout = parsed
			tplID, tplVersion = tpl.ID, tpl.Version
		}
	}

	rows, err := ws.factoryOrderRows(ctx, lines)
	if err != nil {
		return nil, err
	}
	payload, err := alignment.Render(rows, layout)
	if err != nil {
		return nil, err
	}
	path, err := ws.writeExportFile(fmt.Sprintf("factory-orders/%d-%s", order.ID, ws.Now().Format("20060102-150405")), layout.Format, payload)
	if err != nil {
		return nil, err
	}

	stored := string(payload)
	if layout.Format == alignment.FormatXLSX {
		stored = "base64:" + base64.StdEncoding.EncodeToString(payload)
	}
	result := &ExportFileResult{Path: path, Rows: rows}
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		// Re-read inside the transaction so a concurrent void aborts the export.
		current, err := tx.GetSupplierOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if current.Status == string(domain.SupplierOrderVoided) {
			return ErrOrderNotGenerated
		}
		now := ws.Now()
		if current.ExportedAt == nil {
			current.ExportedAt = &now
		}
		current.Status = string(domain.SupplierOrderExported)
		current.TemplateID = tplID
		current.TemplateVersion = tplVersion
		current.ExportPayload = stored
		result.Order = *current
		return tx.UpdateSupplierOrder(ctx, current)
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// factoryOrderRows flattens order lines into rozao-style rows: one row per
// order line and distinct recipient, quantity summed over that recipient's
// linked results, so one source order can ship as multiple parcels.
func (ws *Workspace) factoryOrderRows(ctx context.Context, lines []domain.SupplierOrderLine) ([]map[string]string, error) {
	type key struct {
		line  uint
		name  string
		phone string
		addr  string
	}
	qty := map[key]int{}
	sku := map[key]string{}
	track := map[key]string{}
	var order []key
	for _, line := range lines {
		links, err := ws.Store.ListLinksByOrderLine(ctx, line.ID)
		if err != nil {
			return nil, err
		}
		for _, link := range links {
			res, err := ws.Store.GetResult(ctx, link.FulfillmentResultID)
			if err != nil {
				return nil, err
			}
			k := key{line: line.ID, name: res.Address.RecipientName, phone: res.Address.Phone, addr: res.Address.AddressLine1}
			if _, seen := qty[k]; !seen {
				order = append(order, k)
				sku[k] = line.FactorySKU
				track[k] = line.TrackingID
			}
			qty[k] += res.Quantity
		}
	}
	rows := make([]map[string]string, 0, len(order))
	for _, k := range order {
		rows = append(rows, map[string]string{
			"tracking.id":             track[k],
			"recipient.name":          k.name,
			"recipient.phone":         k.phone,
			"recipient.address_line1": k.addr,
			"product.factory_sku":     sku[k],
			"quantity":                strconv.Itoa(qty[k]),
		})
	}
	return rows, nil
}

// writeExportFile persists an export payload under
// <dataDir>/exports/<name>.<format> and returns the absolute path.
func (ws *Workspace) writeExportFile(name, format string, payload []byte) (string, error) {
	dir, err := ws.dataDir()
	if err != nil {
		return "", err
	}
	full := filepath.Join(dir, "exports", name+"."+format)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("export file: mkdir: %w", err)
	}
	if err := os.WriteFile(full, payload, 0o644); err != nil {
		return "", fmt.Errorf("export file: write %q: %w", full, err)
	}
	return full, nil
}

func (ws *Workspace) dataDir() (string, error) {
	if ws.ResolveDataDir != nil {
		return ws.ResolveDataDir()
	}
	return service.ResolveDataDir()
}

// SkippedShipment records why one shipment-return row did not import.
type SkippedShipment struct {
	LineNo     int
	TrackingID string
	Reason     string
}

// ImportShipmentFileResult summarizes a shipment-return import.
type ImportShipmentFileResult struct {
	Imported  int
	Skipped   []SkippedShipment
	Shipments []domain.Shipment
	Issues    []alignment.ParseIssue
}

// ImportShipmentFile reads a factory shipment return (one row per parcel),
// resolves each row's tracking id to a live supplier order line, and creates
// the shipments in one transaction. Unknown or retired tracking ids and
// already-recorded parcels are skipped with reasons; store failures abort.
func (ws *Workspace) ImportShipmentFile(ctx context.Context, platformID uint, filePath string) (*ImportShipmentFileResult, error) {
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
	}
	mapping := DefaultShipmentReturnMapping()
	if tpl := ws.findTemplate(ctx, platformID, domain.TemplateDirectionInput, DocumentTypeShipmentReturn); tpl != nil {
		if parsed, err := alignment.ParseMappingConfig(tpl.MappingJSON); err == nil {
			mapping = parsed
		}
	}
	rows, issues, err := alignment.Parse(data, format, alignment.TemplateSpec{Mapping: mapping})
	if err != nil {
		return nil, err
	}
	result := &ImportShipmentFileResult{Issues: issues}
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		for _, row := range rows {
			trackingID := row.Values["tracking.id"]
			trackingNo := row.Values["shipment.tracking_no"]
			skip := func(reason string) {
				result.Skipped = append(result.Skipped, SkippedShipment{LineNo: row.LineNo, TrackingID: trackingID, Reason: reason})
			}
			existing, err := tx.ListShipmentsByTracking(ctx, trackingID)
			if err != nil {
				return err
			}
			dup := false
			for _, sh := range existing {
				if sh.TrackingNo == trackingNo {
					dup = true
					break
				}
			}
			if dup {
				skip("shipment already imported")
				continue
			}
			// The 规格&数量 column is a "<sku>_<title> * n" blob, not a plain
			// number; one parcel may bundle several products. Empty (unmapped
			// column) falls back to 0 as before; an unparseable blob skips the
			// row with the reason recorded rather than landing Quantity=0.
			qty := 0
			if raw := row.Values["shipment.quantity"]; raw != "" {
				parsed, err := parseShipmentQuantity(raw)
				if err != nil {
					skip(err.Error())
					continue
				}
				qty = parsed
			}
			var shippedAt *time.Time
			if raw := row.Values["shipment.shipped_at"]; raw != "" {
				if t, ok := parseFlexibleTime(raw); ok {
					shippedAt = &t
				}
			}
			carrierName := row.Values["shipment.carrier_name"]
			carrierCode := resolveCarrierCode(ctx, tx, platformID, carrierName)
			sh, err := tws.ImportShipment(ctx, trackingID, trackingNo, carrierCode, carrierName, qty)
			if err != nil {
				if err == ErrUnknownTracking || err == ErrTrackingRetired {
					skip(err.Error())
					continue
				}
				return err
			}
			if shippedAt != nil {
				// ImportShipment stamps now; the printed shipping time from the
				// file is more truthful, so refine it.
				sh.ShippedAt = shippedAt
				if err := tx.UpdateShipmentShippedAt(ctx, sh.ID, shippedAt); err != nil {
					return err
				}
			}
			result.Imported++
			result.Shipments = append(result.Shipments, *sh)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if result.Skipped == nil {
		result.Skipped = []SkippedShipment{}
	}
	if result.Shipments == nil {
		result.Shipments = []domain.Shipment{}
	}
	if result.Issues == nil {
		result.Issues = []alignment.ParseIssue{}
	}
	return result, nil
}

// parseShipmentQuantity sums the parcel's total shipped quantity out of a
// rouzao-style "规格&数量" blob: pipe-separated segments like
// "206068021_标题 * 2", each contributing the positive integer after its last
// '*' (defaulting to 1 when no multiplier is written). Multi-segment blobs are
// one parcel carrying several products, so segment quantities sum.
func parseShipmentQuantity(blob string) (int, error) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return 0, fmt.Errorf("shipment quantity: empty value")
	}
	total := 0
	for _, seg := range strings.Split(blob, "|") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return 0, fmt.Errorf("shipment quantity: blob %q has an empty segment", blob)
		}
		n := 1
		if i := strings.LastIndex(seg, "*"); i >= 0 {
			rawQty := strings.TrimSpace(seg[i+1:])
			qty, err := strconv.Atoi(rawQty)
			if err != nil || qty <= 0 {
				return 0, fmt.Errorf("shipment quantity: segment %q has non-positive or non-numeric quantity %q", seg, rawQty)
			}
			n = qty
		}
		total += n
	}
	return total, nil
}

// resolveCarrierCode matches a carrier display name against the platform's
// carrier mappings; unmatched names carry no code.
func resolveCarrierCode(ctx context.Context, tx domain.Store, platformID uint, carrierName string) string {
	if carrierName == "" {
		return ""
	}
	mappings, err := tx.ListCarrierMappings(ctx, platformID)
	if err != nil {
		return ""
	}
	for _, m := range mappings {
		if m.InternalName == carrierName && m.InternalCode != "" {
			return m.InternalCode
		}
	}
	return ""
}

// readFileWithFormat reads a file and derives its tabular format from the
// extension.
func readFileWithFormat(filePath string) ([]byte, string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("alignment file: read %q: %w", filePath, err)
	}
	format := alignment.FormatFromExtension(filepath.Ext(filePath))
	if format == "" {
		return nil, "", fmt.Errorf("alignment file: unsupported extension %q", filepath.Ext(filePath))
	}
	return data, format, nil
}

// PreviewTemplate runs a template's mapping against a sample file without
// touching any store, returning the first limit rows and every issue.
func (ws *Workspace) PreviewTemplate(ctx context.Context, tplID uint, filePath string, limit int) (alignment.TemplatePreview, error) {
	tpl, err := ws.Store.GetTemplate(ctx, tplID)
	if err != nil {
		return alignment.TemplatePreview{}, err
	}
	mapping, err := alignment.ParseMappingConfig(tpl.MappingJSON)
	if err != nil {
		return alignment.TemplatePreview{}, fmt.Errorf("preview: template %q: %w", tpl.Name, err)
	}
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return alignment.TemplatePreview{}, err
	}
	if limit <= 0 {
		limit = 20
	}
	return alignment.TestTemplate(data, format, alignment.TemplateSpec{Mapping: mapping, Kind: factKindForDocumentType(tpl.DocumentType)}, limit)
}
