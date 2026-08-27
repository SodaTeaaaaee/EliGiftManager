package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type GenerateFactoryOrderResult struct {
	Order domain.SupplierOrder
	Lines []domain.SupplierOrderLine
}

// GenerateFactoryOrder submits every eligible result of the wave to the
// factory platform. It is the whole-wave form of
// GenerateFactoryOrderForResults.
func (ws *Workspace) GenerateFactoryOrder(ctx context.Context, waveID, factoryID uint) (*domain.SupplierOrder, []domain.SupplierOrderLine, error) {
	return ws.GenerateFactoryOrderForResults(ctx, waveID, factoryID, nil)
}

// GenerateFactoryOrderForResults submits only the results whose ids appear in
// resultIDs (nil or empty means the whole wave). Unselected results are left
// untouched; the one-open-order-per-(wave, factory) slot constraint is
// unchanged, so a partial submission blocks further submissions to the same
// factory until that order is exported or voided.
func (ws *Workspace) GenerateFactoryOrderForResults(ctx context.Context, waveID, factoryID uint, resultIDs []uint) (*domain.SupplierOrder, []domain.SupplierOrderLine, error) {
	var (
		order *domain.SupplierOrder
		lines []domain.SupplierOrderLine
	)
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		o, l, err := ws.withStore(tx).generateFactoryOrderForResults(ctx, waveID, factoryID, resultIDs)
		if err != nil {
			return err
		}
		order, lines = o, l
		return nil
	}); err != nil {
		return nil, nil, err
	}
	return order, lines, nil
}

// generateFactoryOrderForResults creates the supplier order, its lines, the
// execution links, and freezes the covered results. resultIDs nil selects the
// whole wave. It must run on a workspace bound to the surrounding transaction
// so every write commits or rolls back together.
func (ws *Workspace) generateFactoryOrderForResults(ctx context.Context, waveID, factoryID uint, resultIDs []uint) (*domain.SupplierOrder, []domain.SupplierOrderLine, error) {
	if _, err := ws.Store.FindOpenSupplierOrder(ctx, waveID, factoryID); err == nil {
		return nil, nil, ErrOrderAlreadyOpen
	} else if err != domain.ErrNotFound {
		return nil, nil, err
	}
	views, err := ws.ListResultViews(ctx, waveID)
	if err != nil {
		return nil, nil, err
	}
	var selected map[uint]struct{}
	if resultIDs != nil {
		selected = make(map[uint]struct{}, len(resultIDs))
		for _, id := range resultIDs {
			selected[id] = struct{}{}
		}
	}
	type group struct {
		product domain.ProductItem
		results []domain.FulfillmentResult
		qty     int
	}
	groups := map[uint]*group{}
	for _, v := range views {
		if selected != nil {
			if _, ok := selected[v.Result.ID]; !ok {
				continue
			}
		}
		if v.Result.Frozen || v.WorkState != domain.WorkStateReady || v.Result.ProductItemID == nil {
			continue
		}
		p, err := ws.Store.GetProduct(ctx, *v.Result.ProductItemID)
		if err != nil {
			return nil, nil, err
		}
		if p.FactoryPlatformID != factoryID {
			continue
		}
		g, ok := groups[p.ID]
		if !ok {
			g = &group{product: *p}
			groups[p.ID] = g
		}
		g.results = append(g.results, v.Result)
		g.qty += v.Result.Quantity
	}
	if len(groups) == 0 {
		return nil, nil, ErrNothingToSubmit
	}
	// Iterate product groups in a stable order (name, then factory SKU) so the
	// created order lines — and their tracking ids — do not shuffle between
	// otherwise identical runs.
	ordered := make([]*group, 0, len(groups))
	for _, g := range groups {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].product.Name != ordered[j].product.Name {
			return ordered[i].product.Name < ordered[j].product.Name
		}
		return ordered[i].product.FactorySKU < ordered[j].product.FactorySKU
	})
	// Snapshot the factory order output template version into the execution
	// links; 0 means no output template was configured.
	configVersion := 0
	if tpl := findTemplate(ctx, ws.Store, factoryID, domain.TemplateDirectionOutput, DocumentTypeFactoryOrder); tpl != nil {
		configVersion = tpl.Version
	}
	order := &domain.SupplierOrder{WaveID: waveID, FactoryPlatformID: factoryID, Status: string(domain.SupplierOrderGenerated)}
	if err := ws.Store.CreateSupplierOrder(ctx, order); err != nil {
		return nil, nil, err
	}
	var lines []domain.SupplierOrderLine
	for _, g := range ordered {
		tid, err := ws.NewTrackingID()
		if err != nil {
			return nil, nil, err
		}
		retired, err := ws.Store.TrackingIDRetired(ctx, tid)
		if err != nil {
			return nil, nil, err
		}
		if retired {
			return nil, nil, ErrTrackingRetired
		}
		line := &domain.SupplierOrderLine{SupplierOrderID: order.ID, ProductItemID: g.product.ID, FactorySKU: g.product.FactorySKU, Quantity: g.qty, TrackingID: tid}
		if err := ws.Store.CreateSupplierOrderLine(ctx, line); err != nil {
			return nil, nil, err
		}
		for i := range g.results {
			r := g.results[i]
			if err := ws.Store.CreateLink(ctx, &domain.ExecutionQuantityLink{FulfillmentResultID: r.ID, SupplierOrderLineID: line.ID, Quantity: r.Quantity, ConfigVersion: configVersion}); err != nil {
				return nil, nil, err
			}
			r.Frozen = true
			if err := ws.Store.UpdateResult(ctx, &r); err != nil {
				return nil, nil, err
			}
		}
		lines = append(lines, *line)
	}
	return order, lines, nil
}

func (ws *Workspace) ExportFactoryOrder(ctx context.Context, orderID uint) (*domain.SupplierOrder, error) {
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
	payload, err := json.Marshal(lines)
	if err != nil {
		return nil, err
	}
	now := ws.Now()
	if order.ExportedAt == nil {
		order.ExportedAt = &now
	}
	order.Status = string(domain.SupplierOrderExported)
	order.ExportPayload = string(payload)
	if err := ws.Store.UpdateSupplierOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (ws *Workspace) VoidFactoryOrder(ctx context.Context, orderID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		return ws.withStore(tx).voidFactoryOrder(ctx, orderID)
	})
}

// voidFactoryOrder unfreezes the linked results, retires tracking IDs, drops
// the links, and marks the order voided. It must run on a workspace bound to
// the surrounding transaction so every write commits or rolls back together.
func (ws *Workspace) voidFactoryOrder(ctx context.Context, orderID uint) error {
	order, err := ws.Store.GetSupplierOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == string(domain.SupplierOrderExported) || order.ExportedAt != nil {
		return ErrOrderExported
	}
	if order.Status != string(domain.SupplierOrderGenerated) {
		return ErrOrderNotGenerated
	}
	lines, err := ws.Store.ListSupplierOrderLines(ctx, order.ID)
	if err != nil {
		return err
	}
	for _, line := range lines {
		links, err := ws.Store.ListLinksByOrderLine(ctx, line.ID)
		if err != nil {
			return err
		}
		for _, link := range links {
			r, err := ws.Store.GetResult(ctx, link.FulfillmentResultID)
			if err != nil {
				return err
			}
			r.Frozen = false
			if err := ws.Store.UpdateResult(ctx, r); err != nil {
				return err
			}
		}
		if line.TrackingID != "" {
			if err := ws.Store.RetireTrackingID(ctx, line.TrackingID, order.WaveID); err != nil {
				return err
			}
		}
		line.TrackingRetired = true
		if err := ws.Store.UpdateSupplierOrderLine(ctx, &line); err != nil {
			return err
		}
	}
	if err := ws.Store.DeleteLinksByOrder(ctx, order.ID); err != nil {
		return err
	}
	now := ws.Now()
	order.Status = string(domain.SupplierOrderVoided)
	order.VoidedAt = &now
	return ws.Store.UpdateSupplierOrder(ctx, order)
}

func (ws *Workspace) ImportShipment(ctx context.Context, trackingID, trackingNo, carrierCode, carrierName string, qty int) (*domain.Shipment, error) {
	if trackingID == "" {
		return nil, fmt.Errorf("tracking id is required")
	}
	retired, err := ws.Store.TrackingIDRetired(ctx, trackingID)
	if err != nil {
		return nil, err
	}
	if retired {
		return nil, ErrTrackingRetired
	}
	if _, err := ws.Store.GetSupplierOrderLineByTracking(ctx, trackingID); err != nil {
		if err == domain.ErrNotFound {
			return nil, ErrUnknownTracking
		}
		return nil, err
	}
	now := ws.Now()
	sh := &domain.Shipment{TrackingID: trackingID, TrackingNo: trackingNo, CarrierCode: carrierCode, CarrierName: carrierName, Quantity: qty, ShippedAt: &now}
	if err := ws.Store.CreateShipment(ctx, sh); err != nil {
		return nil, err
	}
	return sh, nil
}

func (ws *Workspace) GenerateWritebacks(ctx context.Context, factID uint) ([]domain.ChannelWritebackItem, error) {
	var created []domain.ChannelWritebackItem
	if err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		items, err := generateWritebacks(ctx, tx, factID)
		if err != nil {
			return err
		}
		created = items
		return nil
	}); err != nil {
		return nil, err
	}
	return created, nil
}

// generateWritebacks collects the shipments behind a source fact and records
// one writeback item per shipment. Every read and write goes through the
// explicit store argument, so callers hand it the transaction they opened;
// every write commits or rolls back together.
func generateWritebacks(ctx context.Context, store domain.Store, factID uint) ([]domain.ChannelWritebackItem, error) {
	existing, err := store.ListWritebacksByFact(ctx, factID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing, nil
	}
	waves, err := store.ListWaves(ctx)
	if err != nil {
		return nil, err
	}
	seenShipment := map[uint]struct{}{}
	var created []domain.ChannelWritebackItem
	// Snapshot the writeback output template (0 when none is configured).
	var wbTplID uint
	var wbTplVersion int
	fact, err := store.GetFact(ctx, factID)
	if err != nil {
		return nil, err
	}
	if tpl := findTemplate(ctx, store, fact.PlatformID, domain.TemplateDirectionOutput, DocumentTypeWriteback); tpl != nil {
		wbTplID, wbTplVersion = tpl.ID, tpl.Version
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
					item := &domain.ChannelWritebackItem{
						InputFactID:     factID,
						ShipmentID:      sh.ID,
						TrackingNo:      sh.TrackingNo,
						CarrierCode:     sh.CarrierCode,
						Status:          string(domain.WritebackPending),
						TemplateID:      wbTplID,
						TemplateVersion: wbTplVersion,
						Payload:         strings.Join([]string{sh.TrackingNo, sh.CarrierCode}, ","),
					}
					if err := store.CreateWriteback(ctx, item); err != nil {
						return nil, err
					}
					created = append(created, *item)
				}
			}
		}
	}
	return created, nil
}

func (ws *Workspace) ListSupplierOrders(ctx context.Context, waveID uint) ([]domain.SupplierOrder, error) {
	return ws.Store.ListSupplierOrders(ctx, waveID)
}

func (ws *Workspace) ListSupplierOrderLines(ctx context.Context, orderID uint) ([]domain.SupplierOrderLine, error) {
	return ws.Store.ListSupplierOrderLines(ctx, orderID)
}
