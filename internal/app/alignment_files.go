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
// mapping config, and ingests the produced facts in one transaction. The
// template must be an active input template of the platform that produces
// facts (membership_list or order_export); built-in rows are refused (copy
// them first) and shipment returns go through ImportShipmentFile.
func (ws *Workspace) ImportFile(ctx context.Context, platformID, tplID uint, filePath string) (*ImportFileResult, error) {
	tpl, err := ws.Store.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, err
	}
	if err := checkUsableTemplate(tpl, platformID, domain.TemplateDirectionInput, ""); err != nil {
		return nil, fmt.Errorf("import file: %w", err)
	}
	if tpl.DocumentType != DocumentTypeMembershipList && tpl.DocumentType != DocumentTypeOrderExport {
		return nil, fmt.Errorf("import file: %w: template %d %q is a %s template; only membership_list and order_export templates ingest facts", ErrTemplateMismatch, tpl.ID, tpl.Name, tpl.DocumentType)
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

	// Binary xlsx originals are stored base64-prefixed like exported xlsx
	// payloads; csv/xls stay verbatim. Nothing decodes the audit copy.
	rawPayload := string(data)
	if format == alignment.FormatXLSX {
		rawPayload = "base64:" + base64.StdEncoding.EncodeToString(data)
	}
	doc := &domain.InputDocument{
		PlatformID:      platformID,
		DocumentType:    tpl.DocumentType,
		Direction:       tpl.Direction,
		OriginalName:    filepath.Base(filePath),
		RawPayload:      rawPayload,
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
				DisplayName:     strings.TrimSpace(row.Values["customer.display_name"]),
				Recipient:       recipientSnapshot(row.Values),
			}
			if docNo := row.Values["source.document_no"]; docNo != "" {
				in.SourceDocumentNo = docNo
				in.StableExternalID = docNo
			} else {
				// Known tradeoff: the "fp:" stable id embeds the template's
				// Fingerprint key set at import time. Changing a template's
				// Fingerprint keys makes future imports compute different
				// "fp:" ids, so rows already imported under the old keys stop
				// matching (duplicate detection misses them against old
				// files). Accepted for now; do not retune template
				// fingerprints casually.
				in.StableExternalID = "fp:" + row.Fingerprint
			}
			if raw := row.Values["source.created_at"]; raw != "" {
				if t, ok := parseFlexibleTime(raw); ok {
					in.SourceCreatedAt = &t
				}
			}
			if in.Recipient != nil {
				in.ExtraData = recipientExtraData(*in.Recipient)
			}
			bySource[row.SourceRow] = in
			order = append(order, row.SourceRow)
		}
		qty := 1
		if n, err := strconv.Atoi(strings.TrimSpace(row.Values["quantity"])); err == nil && n > 0 {
			qty = n
		}
		lineNo := row.LineNo
		if n, err := strconv.Atoi(strings.TrimSpace(row.Values["source.line_no"])); err == nil && n > 0 {
			lineNo = n
		}
		in.Lines = append(in.Lines, IngestLine{
			SourceLineNo:  lineNo,
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

// recipientSnapshot collects the recipient.* keys of a parsed row, or nil when
// the row carries no recipient data at all.
func recipientSnapshot(values map[string]string) *domain.AddressSnapshot {
	snap := domain.AddressSnapshot{
		RecipientName: strings.TrimSpace(values["recipient.name"]),
		Phone:         strings.TrimSpace(values["recipient.phone"]),
		Country:       strings.TrimSpace(values["recipient.country"]),
		Province:      strings.TrimSpace(values["recipient.province"]),
		City:          strings.TrimSpace(values["recipient.city"]),
		District:      strings.TrimSpace(values["recipient.district"]),
		AddressLine1:  strings.TrimSpace(values["recipient.address_line1"]),
		AddressLine2:  strings.TrimSpace(values["recipient.address_line2"]),
		PostalCode:    strings.TrimSpace(values["recipient.postal_code"]),
	}
	if snap.RecipientName == "" && snap.Phone == "" && snap.AddressLine1 == "" && snap.Province == "" && snap.City == "" {
		return nil
	}
	return &snap
}

// recipientExtraData keeps the recipient data as read from the file on the
// fact's extra data, an audit copy independent of the profile address it also
// lands in.
func recipientExtraData(snap domain.AddressSnapshot) string {
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
// file-producing layer on top of the generate-then-export state machine. The
// layout comes from the factory platform's active factory_order template;
// without one the export fails with ErrNoActiveTemplate.
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

	tpl, err := requireActiveTemplate(ctx, ws.Store, order.FactoryPlatformID, domain.TemplateDirectionOutput, DocumentTypeFactoryOrder)
	if err != nil {
		return nil, fmt.Errorf("export factory order: %w", err)
	}
	layout, err := alignment.ParseLayoutConfig(tpl.LayoutJSON)
	if err != nil {
		return nil, fmt.Errorf("export factory order: template %q: %w", tpl.Name, err)
	}
	tplID, tplVersion := tpl.ID, tpl.Version

	rows, err := ws.factoryOrderRows(ctx, lines)
	if err != nil {
		return nil, err
	}
	payload, err := alignment.Render(rows, layout)
	if err != nil {
		return nil, err
	}

	stored := string(payload)
	if layout.Format == alignment.FormatXLSX {
		stored = "base64:" + base64.StdEncoding.EncodeToString(payload)
	}
	result := &ExportFileResult{Rows: rows}
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		// Re-read inside the transaction so a concurrent void aborts the export.
		current, err := tx.GetSupplierOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if current.Status == string(domain.SupplierOrderVoided) {
			return ErrOrderNotGenerated
		}
		// The file write lives inside the transaction and before the status
		// flip: a write failure rolls the order back to generated, so an
		// exported order always has its file. The residual failure mode is the
		// reverse — a written file with a rolled-back transaction (for example
		// the update fails) — which leaves a harmless orphan file and no dirty
		// state.
		path, err := ws.writeExportFile(fmt.Sprintf("factory-orders/%d-%s", order.ID, ws.Now().Format("20060102-150405")), layout.Format, payload)
		if err != nil {
			return err
		}
		result.Path = path
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

// ImportShipmentFile reads a factory shipment return (one row per parcel)
// through the given active shipment_return template of the factory platform,
// resolves each row's tracking id to a live supplier order line, and creates
// the shipments in one transaction. Unknown or retired tracking ids and
// already-recorded parcels are skipped with reasons; store failures abort.
// Carrier names are kept as the file spells them; translating them into a
// source platform's carrier id happens at writeback time.
func (ws *Workspace) ImportShipmentFile(ctx context.Context, platformID, tplID uint, filePath string) (*ImportShipmentFileResult, error) {
	tpl, err := ws.Store.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, err
	}
	if err := checkUsableTemplate(tpl, platformID, domain.TemplateDirectionInput, DocumentTypeShipmentReturn); err != nil {
		return nil, fmt.Errorf("import shipment file: %w", err)
	}
	mapping, err := alignment.ParseMappingConfig(tpl.MappingJSON)
	if err != nil {
		return nil, fmt.Errorf("import shipment file: template %q: %w", tpl.Name, err)
	}
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
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
				parsed, segIssues, err := parseShipmentQuantity(raw)
				if err != nil {
					skip(err.Error())
					continue
				}
				for i := range segIssues {
					segIssues[i].LineNo = row.LineNo
					result.Issues = append(result.Issues, segIssues[i])
				}
				qty = parsed
			}
			var shippedAt *time.Time
			if raw := row.Values["shipment.shipped_at"]; raw != "" {
				if t, ok := parseFlexibleTime(raw); ok {
					shippedAt = &t
				}
			}
			carrierName := strings.TrimSpace(row.Values["shipment.carrier_name"])
			sh, err := tws.ImportShipment(ctx, trackingID, trackingNo, "", carrierName, qty)
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
// one parcel carrying several products, so segment quantities sum. Segments
// that carry text but no '*' multiplier still count as 1, but are reported as
// issues so the operator sees the conservative default instead of a silent
// guess.
func parseShipmentQuantity(blob string) (int, []alignment.ParseIssue, error) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return 0, nil, fmt.Errorf("shipment quantity: empty value")
	}
	var issues []alignment.ParseIssue
	total := 0
	for _, seg := range strings.Split(blob, "|") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return 0, nil, fmt.Errorf("shipment quantity: blob %q has an empty segment", blob)
		}
		n := 1
		if i := strings.LastIndex(seg, "*"); i >= 0 {
			rawQty := strings.TrimSpace(seg[i+1:])
			qty, err := strconv.Atoi(rawQty)
			if err != nil || qty <= 0 {
				return 0, nil, fmt.Errorf("shipment quantity: segment %q has non-positive or non-numeric quantity %q", seg, rawQty)
			}
			n = qty
		} else {
			issues = append(issues, alignment.ParseIssue{
				Key:     "shipment.quantity",
				Message: fmt.Sprintf("segment %q has no '*' multiplier; counting it as 1", seg),
			})
		}
		total += n
	}
	return total, issues, nil
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

// defaultPreviewLimit caps preview and inspection output when the caller
// passes no positive limit.
const defaultPreviewLimit = 20

// PreviewTemplate runs a saved template's mapping against a sample file
// without touching any store, returning the first limit rows and every issue.
// Built-in templates may be previewed: testing before copying is fine.
func (ws *Workspace) PreviewTemplate(ctx context.Context, tplID uint, filePath string, limit int) (*alignment.TemplatePreview, error) {
	tpl, err := ws.Store.GetTemplate(ctx, tplID)
	if err != nil {
		return nil, err
	}
	if tpl.Direction != string(domain.TemplateDirectionInput) {
		return nil, fmt.Errorf("preview: template %q is an output template; only input templates parse sample files", tpl.Name)
	}
	preview, err := previewMapping(tpl.MappingJSON, tpl.DocumentType, filePath, limit)
	if err != nil {
		return nil, fmt.Errorf("preview: template %q: %w", tpl.Name, err)
	}
	return preview, nil
}

// PreviewMapping runs an unsaved mapping config against a sample file, so a
// template can be tuned before it is stored. documentType picks the fact kind
// the rows would ingest as; no store is touched.
func (ws *Workspace) PreviewMapping(ctx context.Context, mappingJSON, documentType, filePath string, limit int) (*alignment.TemplatePreview, error) {
	preview, err := previewMapping(mappingJSON, documentType, filePath, limit)
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}
	return preview, nil
}

func previewMapping(mappingJSON, documentType, filePath string, limit int) (*alignment.TemplatePreview, error) {
	mapping, err := alignment.ParseMappingConfig(mappingJSON)
	if err != nil {
		return nil, err
	}
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultPreviewLimit
	}
	preview, err := alignment.TestTemplate(data, format, alignment.TemplateSpec{Mapping: mapping, Kind: factKindForDocumentType(documentType)}, limit)
	if err != nil {
		return nil, err
	}
	if preview.Rows == nil {
		preview.Rows = []alignment.PreviewRow{}
	}
	return &preview, nil
}

// SampleFileInfo describes a tabular file before any mapping is applied: its
// format, the selectable xlsx sheets, the first raw records (the first one is
// the candidate header row), and the total record count.
type SampleFileInfo struct {
	Format  string
	Sheets  []string
	Records [][]string
	Total   int
}

// InspectSampleFile reads a sample file's raw records so a template author can
// see the headers and values before mapping them. sheetName selects the xlsx
// sheet (empty means the first); it is ignored for csv and xls. limit <= 0
// returns the first 20 records. No store is touched.
func (ws *Workspace) InspectSampleFile(ctx context.Context, filePath, sheetName string, limit int) (*SampleFileInfo, error) {
	data, format, err := readFileWithFormat(filePath)
	if err != nil {
		return nil, err
	}
	sheets, err := alignment.ListSheets(data, format)
	if err != nil {
		return nil, err
	}
	records, err := alignment.ReadRows(data, format, sheetName)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultPreviewLimit
	}
	info := &SampleFileInfo{Format: format, Sheets: sheets, Total: len(records)}
	if len(records) > limit {
		records = records[:limit]
	}
	info.Records = make([][]string, 0, len(records))
	for _, r := range records {
		if r == nil {
			r = []string{}
		}
		info.Records = append(info.Records, r)
	}
	return info, nil
}
