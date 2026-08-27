package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// maxWritebackErrorRunes caps the stored error text so a runaway platform
// response cannot bloat the writeback row.
const maxWritebackErrorRunes = 500

// DefaultWritebackLayout is the built-in bilibili-shaped two-column writeback
// sheet used when the source platform has no writeback output template yet.
func DefaultWritebackLayout() alignment.LayoutConfig {
	return alignment.LayoutConfig{
		Version:     alignment.LayoutSchemaVersion,
		Format:      alignment.FormatCSV,
		ColumnOrder: []string{"source.document_no", "shipment.carrier_code"},
		HeaderNames: map[string]string{
			"source.document_no":    "订单号",
			"shipment.carrier_code": "快递公司编码",
		},
	}
}

// GenerateWritebacks collects the parcels behind a source fact into writeback
// items, one per shipment. Re-running it is safe: parcels that already have an
// item keep theirs untouched and only new parcels get new items. It returns
// every current item of the fact, old and new.
func (ws *Workspace) GenerateWritebacks(ctx context.Context, factID uint) ([]domain.ChannelWritebackItem, error) {
	var items []domain.ChannelWritebackItem
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		items, err = generateWritebacks(ctx, tx, factID)
		return err
	}); err != nil {
		return nil, err
	}
	return items, nil
}

// generateWritebacks walks the fact's fulfillment results to their supplier
// order lines and shipments, creating one writeback item per not-yet-recorded
// parcel with its rendered payload. Every read and write goes through the
// explicit store argument, so callers hand it the transaction they opened;
// every write commits or rolls back together.
func generateWritebacks(ctx context.Context, store domain.Store, factID uint) ([]domain.ChannelWritebackItem, error) {
	existing, err := store.ListWritebacksByFact(ctx, factID)
	if err != nil {
		return nil, err
	}
	items := append([]domain.ChannelWritebackItem(nil), existing...)
	seenShipment := make(map[uint]struct{}, len(existing))
	for _, wb := range existing {
		seenShipment[wb.ShipmentID] = struct{}{}
	}
	fact, err := store.GetFact(ctx, factID)
	if err != nil {
		return nil, err
	}
	layout, tplID, tplVersion, err := writebackLayout(ctx, store, fact.PlatformID)
	if err != nil {
		return nil, err
	}
	// The "order number" the source platform expects is the fact's stable
	// external id (falling back to the source document number when the stable
	// id is empty), not our internal tracking ids.
	orderNo := fact.StableExternalID
	if orderNo == "" {
		orderNo = fact.SourceDocumentNo
	}
	mappings, err := store.ListCarrierMappings(ctx, fact.PlatformID)
	if err != nil {
		return nil, err
	}
	externalCarrierCode := func(sh domain.Shipment) string {
		if sh.CarrierCode == "" {
			return ""
		}
		for _, m := range mappings {
			if m.InternalCode == sh.CarrierCode && m.ExternalCode != "" {
				return m.ExternalCode
			}
		}
		// No mapping to the platform's carrier vocabulary: stay conservative
		// and leave the cell empty rather than guess an external code.
		return ""
	}
	waves, err := store.ListWaves(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range waves {
		results, err := store.ListResults(ctx, w.ID)
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			if r.InputFactID == nil || *r.InputFactID != factID {
				continue
			}
			links, err := store.ListLinksByResult(ctx, r.ID)
			if err != nil {
				return nil, err
			}
			for _, link := range links {
				line, err := store.GetSupplierOrderLine(ctx, link.SupplierOrderLineID)
				if err != nil {
					return nil, err
				}
				if line.TrackingRetired {
					continue
				}
				ships, err := store.ListShipmentsByTracking(ctx, line.TrackingID)
				if err != nil {
					return nil, err
				}
				for _, sh := range ships {
					if _, ok := seenShipment[sh.ID]; ok {
						continue
					}
					seenShipment[sh.ID] = struct{}{}
					// Carrier code is the one value that needs translation:
					// templates receive the source platform's external code,
					// every other shipment field is passed through verbatim.
					payload, err := alignment.Render([]map[string]string{{
						"source.document_no":    orderNo,
						"shipment.carrier_code": externalCarrierCode(sh),
						"shipment.carrier_name": sh.CarrierName,
						"shipment.tracking_no":  sh.TrackingNo,
						"shipment.quantity":     strconv.Itoa(sh.Quantity),
					}}, layout)
					if err != nil {
						return nil, fmt.Errorf("writeback payload: %w", err)
					}
					item := &domain.ChannelWritebackItem{
						InputFactID:     factID,
						ShipmentID:      sh.ID,
						TrackingNo:      sh.TrackingNo,
						CarrierCode:     sh.CarrierCode,
						Quantity:        sh.Quantity,
						Status:          string(domain.WritebackPending),
						TemplateID:      tplID,
						TemplateVersion: tplVersion,
						Payload:         storeRenderedPayload(layout.Format, payload),
					}
					if err := store.CreateWriteback(ctx, item); err != nil {
						return nil, err
					}
					items = append(items, *item)
				}
			}
		}
	}
	return items, nil
}

// writebackLayout resolves the source platform's writeback output template
// and falls back to the built-in bilibili layout when no template is
// configured or its layout does not parse. The template id/version snapshot
// is only non-zero when the template layout is actually used.
func writebackLayout(ctx context.Context, store domain.Store, platformID uint) (alignment.LayoutConfig, uint, int, error) {
	layout := DefaultWritebackLayout()
	var tplID uint
	var tplVersion int
	tpl, err := findTemplate(ctx, store, platformID, domain.TemplateDirectionOutput, DocumentTypeWriteback)
	if err != nil {
		return layout, 0, 0, err
	}
	if tpl != nil {
		if parsed, perr := alignment.ParseLayoutConfig(tpl.LayoutJSON); perr == nil && len(parsed.ColumnOrder) > 0 {
			layout = parsed
			tplID, tplVersion = tpl.ID, tpl.Version
		}
	}
	return layout, tplID, tplVersion, nil
}

// storeRenderedPayload keeps rendered CSV payloads verbatim and base64-prefixes
// binary xlsx payloads, mirroring how factory-order exports are stored.
func storeRenderedPayload(format string, payload []byte) string {
	if format == alignment.FormatXLSX {
		return "base64:" + base64.StdEncoding.EncodeToString(payload)
	}
	return string(payload)
}

// MarkWritebackSent records that the parcel's writeback reached the source
// platform: status becomes sent and the error text of the last failed attempt
// is cleared. RetryCount is deliberately kept so the retry history survives a
// later success.
func (ws *Workspace) MarkWritebackSent(ctx context.Context, writebackID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wb, err := tx.GetWriteback(ctx, writebackID)
		if err != nil {
			return err
		}
		wb.Status = string(domain.WritebackSent)
		wb.ErrorMessage = ""
		return tx.UpdateWriteback(ctx, wb)
	})
}

// MarkWritebackFailed records a failed writeback attempt: status becomes
// failed, the retry counter advances, and the error text of this attempt is
// stored (truncated to maxWritebackErrorRunes runes).
func (ws *Workspace) MarkWritebackFailed(ctx context.Context, writebackID uint, errMsg string) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wb, err := tx.GetWriteback(ctx, writebackID)
		if err != nil {
			return err
		}
		wb.Status = string(domain.WritebackFailed)
		wb.RetryCount++
		wb.ErrorMessage = truncateRunes(errMsg, maxWritebackErrorRunes)
		return tx.UpdateWriteback(ctx, wb)
	})
}

// ListWritebacksByWave returns every writeback item of the facts assigned to
// the wave. Membership follows the fact lines' wave assignment — the same
// language the inbox uses — so listed items survive entitlement recomputes.
// Items are ordered by id for stable display.
func (ws *Workspace) ListWritebacksByWave(ctx context.Context, waveID uint) ([]domain.ChannelWritebackItem, error) {
	lines, err := ws.Store.ListFactLinesByWave(ctx, waveID)
	if err != nil {
		return nil, err
	}
	seenFact := make(map[uint]struct{}, len(lines))
	var out []domain.ChannelWritebackItem
	for _, l := range lines {
		if _, ok := seenFact[l.FactID]; ok {
			continue
		}
		seenFact[l.FactID] = struct{}{}
		items, err := ws.Store.ListWritebacksByFact(ctx, l.FactID)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// WritebackFileResult reports one writeback export: the written file's
// absolute path.
type WritebackFileResult struct {
	Path string
}

// ExportWritebackFile writes one writeback item's stored payload to
// <dataDir>/exports/writebacks/<id>-<timestamp>.<format> and returns the
// path. The payload was rendered and snapshotted on the item at generation
// time, so exporting is a pure bytes-to-file step: no re-render and no state
// change (reaching the platform and marking sent/failed are separate steps).
func (ws *Workspace) ExportWritebackFile(ctx context.Context, writebackID uint) (*WritebackFileResult, error) {
	wb, err := ws.Store.GetWriteback(ctx, writebackID)
	if err != nil {
		return nil, err
	}
	payload, format, err := writebackPayloadBytes(wb.Payload)
	if err != nil {
		return nil, err
	}
	path, err := ws.writeExportFile(fmt.Sprintf("writebacks/%d-%s", wb.ID, ws.Now().Format("20060102-150405")), format, payload)
	if err != nil {
		return nil, err
	}
	return &WritebackFileResult{Path: path}, nil
}

// writebackPayloadBytes decodes the stored payload back into file bytes and
// the file format: csv payloads are stored verbatim, xlsx payloads are stored
// base64-prefixed like every binary export payload.
func writebackPayloadBytes(stored string) ([]byte, string, error) {
	if rest, ok := strings.CutPrefix(stored, "base64:"); ok {
		data, err := base64.StdEncoding.DecodeString(rest)
		if err != nil {
			return nil, "", fmt.Errorf("writeback payload: decode xlsx: %w", err)
		}
		return data, alignment.FormatXLSX, nil
	}
	return []byte(stored), alignment.FormatCSV, nil
}

// truncateRunes shortens s to at most n runes without splitting a character.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
