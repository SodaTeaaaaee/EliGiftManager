package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type ProductTotal struct {
	ProductID uint
	Name      string
	Quantity  int
}

func (ws *Workspace) CreateWave(ctx context.Context, name, notes string) (*domain.Wave, error) {
	if name == "" {
		return nil, fmt.Errorf("wave name is required")
	}
	no, err := ws.Store.NextWaveNo(ctx)
	if err != nil {
		return nil, err
	}
	w := &domain.Wave{WaveNo: no, Name: name, Notes: notes, CloseResult: string(domain.WaveCloseResultOpen)}
	if err := ws.Store.CreateWave(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (ws *Workspace) ListWaves(ctx context.Context) ([]domain.Wave, error) {
	return ws.Store.ListWaves(ctx)
}

func (ws *Workspace) GetWave(ctx context.Context, id uint) (*domain.Wave, error) {
	return ws.Store.GetWave(ctx, id)
}

func (ws *Workspace) CloseWave(ctx context.Context, id uint, result, note string) error {
	if result != string(domain.WaveCloseResultClean) && result != string(domain.WaveCloseResultResidual) {
		return fmt.Errorf("close result must be clean or residual")
	}
	w, err := ws.Store.GetWave(ctx, id)
	if err != nil {
		return err
	}
	now := ws.Now()
	w.CloseResult = result
	w.CloseNote = note
	w.ClosedAt = &now
	return ws.Store.UpdateWave(ctx, w)
}

func (ws *Workspace) ReopenWave(ctx context.Context, id uint) error {
	w, err := ws.Store.GetWave(ctx, id)
	if err != nil {
		return err
	}
	if w.CloseResult == string(domain.WaveCloseResultOpen) {
		return nil
	}
	now := ws.Now()
	w.CloseResult = string(domain.WaveCloseResultOpen)
	w.ReopenedAt = &now
	return ws.Store.UpdateWave(ctx, w)
}

func (ws *Workspace) UpsertRule(ctx context.Context, rule *domain.EntitlementRule) error {
	if err := validateSelector(rule.Selector); err != nil {
		return err
	}
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wave, err := tx.GetWave(ctx, rule.WaveID)
		if err != nil {
			return err
		}
		if wave.CloseResult != string(domain.WaveCloseResultOpen) {
			return ErrWaveClosed
		}
		if rule.ID == 0 {
			if err := tx.CreateRule(ctx, rule); err != nil {
				return err
			}
		} else {
			old, err := tx.GetRule(ctx, rule.ID)
			if err != nil {
				return err
			}
			// Callers may hand in a rule built from scratch (e.g. an edit
			// form), so backfill the audit timestamps before the full-field
			// Save in UpdateRule clobbers them with zero values.
			rule.CreatedAt, rule.UpdatedAt = old.CreatedAt, old.UpdatedAt
			if err := tx.UpdateRule(ctx, rule); err != nil {
				return err
			}
		}
		return recompute(ctx, tx, rule.WaveID)
	})
}

func (ws *Workspace) DeleteRule(ctx context.Context, id uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		rule, err := tx.GetRule(ctx, id)
		if err != nil {
			return err
		}
		if err := tx.DeleteRule(ctx, id); err != nil {
			return err
		}
		return recompute(ctx, tx, rule.WaveID)
	})
}

func (ws *Workspace) AddException(ctx context.Context, e *domain.EntitlementException) error {
	if e.InstanceID == 0 {
		return fmt.Errorf("exception must name an entitlement instance")
	}
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.CreateException(ctx, e); err != nil {
			return err
		}
		return recompute(ctx, tx, e.WaveID)
	})
}

func (ws *Workspace) DeleteException(ctx context.Context, id uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		e, err := tx.GetException(ctx, id)
		if err != nil {
			return err
		}
		if err := tx.DeleteException(ctx, id); err != nil {
			return err
		}
		return recompute(ctx, tx, e.WaveID)
	})
}

// ExceptionView is one entitlement exception joined with the display fields
// the wave screen needs: the member behind the instance and the product the
// exception grants.
type ExceptionView struct {
	ID            uint
	InstanceID    uint
	CustomerName  string
	ProductItemID uint
	ProductName   string
	Quantity      int
	Note          string
}

// ListExceptions returns the wave's entitlement exceptions with customer and
// product display names resolved, so the UI never has to hand-type instance
// or product ids.
func (ws *Workspace) ListExceptions(ctx context.Context, waveID uint) ([]ExceptionView, error) {
	excs, err := ws.Store.ListExceptions(ctx, waveID)
	if err != nil {
		return nil, err
	}
	out := make([]ExceptionView, 0, len(excs))
	for _, e := range excs {
		view := ExceptionView{ID: e.ID, InstanceID: e.InstanceID, ProductItemID: e.ProductID, Quantity: e.Quantity, Note: e.Note}
		inst, err := ws.Store.GetInstance(ctx, e.InstanceID)
		if err != nil {
			return nil, err
		}
		view.CustomerName = instanceCustomerName(ctx, ws.Store, *inst)
		p, err := ws.Store.GetProduct(ctx, e.ProductID)
		if err != nil {
			return nil, err
		}
		view.ProductName = p.Name
		out = append(out, view)
	}
	return out, nil
}

// InstanceView is one entitlement instance with the member-facing display
// fields: customer name and a platform identity summary.
type InstanceView struct {
	ID               uint
	CustomerName     string
	PlatformIdentity string
	MembershipLevel  string
}

// ListEntitlementInstances returns the wave's membership instances with
// customer and identity display fields resolved.
func (ws *Workspace) ListEntitlementInstances(ctx context.Context, waveID uint) ([]InstanceView, error) {
	insts, err := ws.Store.ListInstances(ctx, waveID)
	if err != nil {
		return nil, err
	}
	out := make([]InstanceView, 0, len(insts))
	for _, inst := range insts {
		view := InstanceView{ID: inst.ID, MembershipLevel: inst.MembershipLevel}
		view.CustomerName = instanceCustomerName(ctx, ws.Store, inst)
		if inst.PlatformIdentityID != nil {
			ident, err := ws.Store.GetIdentity(ctx, *inst.PlatformIdentityID)
			if err != nil {
				return nil, err
			}
			view.PlatformIdentity = ident.IdentityType + ":" + ident.IdentityValue
		}
		out = append(out, view)
	}
	return out, nil
}

// instanceCustomerName resolves the member behind an instance: the instance's
// own customer link first, then the identity's attachment. Instances without
// either resolve to an empty name. Every read goes through the explicit store
// argument so callers control which transaction or connection the lookup
// joins.
func instanceCustomerName(ctx context.Context, store domain.Store, inst domain.EntitlementInstance) string {
	resolve := func(id uint) string {
		c, err := store.GetCustomer(ctx, id)
		if err != nil {
			return ""
		}
		return c.DisplayName
	}
	if inst.CustomerProfileID != nil {
		return resolve(*inst.CustomerProfileID)
	}
	if inst.PlatformIdentityID != nil {
		ident, err := store.GetIdentity(ctx, *inst.PlatformIdentityID)
		if err != nil {
			return ""
		}
		if ident.CustomerProfileID != nil {
			return resolve(*ident.CustomerProfileID)
		}
	}
	return ""
}

func (ws *Workspace) CreateGrant(ctx context.Context, waveID, customerID, productID uint, qty int) (*domain.FulfillmentResult, error) {
	var grant *domain.FulfillmentResult
	err := ws.Store.WithTx(ctx, func(tx domain.Store) error {
		wave, err := tx.GetWave(ctx, waveID)
		if err != nil {
			return err
		}
		if wave.CloseResult != string(domain.WaveCloseResultOpen) {
			return ErrWaveClosed
		}
		fact := &domain.InputFact{Kind: string(domain.InputFactKindOperatorGrant), CustomerProfileID: &customerID}
		if err := tx.CreateFact(ctx, fact); err != nil {
			return err
		}
		line := &domain.InputFactLine{FactID: fact.ID, SourceLineNo: 1, ProductItemID: &productID, Quantity: qty, WaveID: &waveID}
		if err := tx.CreateFactLine(ctx, line); err != nil {
			return err
		}
		if err := ws.withStore(tx).ensureGrantResult(ctx, waveID, fact, line); err != nil {
			return err
		}
		results, err := tx.ListResults(ctx, waveID)
		if err != nil {
			return err
		}
		for i := range results {
			if results[i].InputFactLineID != nil && *results[i].InputFactLineID == line.ID {
				grant = &results[i]
				return nil
			}
		}
		return fmt.Errorf("grant result not created")
	})
	if err != nil {
		return nil, err
	}
	return grant, nil
}

func validateSelector(sel domain.EntitlementSelector) error {
	switch sel.Type {
	case string(domain.SelectorWaveAll):
		return nil
	case string(domain.SelectorPlatformLevel):
		if sel.PlatformID == 0 || sel.Level == "" {
			return fmt.Errorf("platform_level selector needs platform and level")
		}
		return nil
	case string(domain.SelectorInstance):
		if sel.InstanceID == nil {
			return fmt.Errorf("instance selector needs instance id")
		}
		return nil
	default:
		return ErrInvalidSelector
	}
}

func (ws *Workspace) RecomputeEntitlements(ctx context.Context, waveID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		return recompute(ctx, tx, waveID)
	})
}

// recompute deletes and rebuilds the unfrozen entitlement results of a wave.
// Every read and write goes through the explicit store argument, so callers
// hand it the transaction they opened; public entry points wrap it in their
// own WithTx instead of nesting RecomputeEntitlements.
func recompute(ctx context.Context, store domain.Store, waveID uint) error {
	type key struct {
		inst uint
		prod uint
	}
	old, err := store.ListUnfrozenEntitlementResults(ctx, waveID)
	if err != nil {
		return err
	}
	// Operator-pinned address snapshots must survive the rebuild: collect them
	// by (instance, product) before deleting so the recreated results keep the
	// manual address instead of falling back to the customer default.
	pinned := map[key]domain.AddressSnapshot{}
	for _, r := range old {
		if r.AddressPinned && r.EntitlementInstanceID != nil && r.ProductItemID != nil {
			pinned[key{*r.EntitlementInstanceID, *r.ProductItemID}] = r.Address
		}
	}
	for _, r := range old {
		if err := store.DeleteResult(ctx, r.ID); err != nil {
			return err
		}
	}
	instances, err := store.ListInstances(ctx, waveID)
	if err != nil {
		return err
	}
	rules, err := store.ListRules(ctx, waveID)
	if err != nil {
		return err
	}
	exceptions, err := store.ListExceptions(ctx, waveID)
	if err != nil {
		return err
	}
	qty := map[key]int{}
	cust := map[uint]*uint{}
	existing, err := store.ListResults(ctx, waveID)
	if err != nil {
		return err
	}
	frozen := map[key]struct{}{}
	for _, r := range existing {
		if !r.Frozen || r.SourceKind != string(domain.SourceEntitlementInstance) || r.EntitlementInstanceID == nil || r.ProductItemID == nil {
			continue
		}
		frozen[key{*r.EntitlementInstanceID, *r.ProductItemID}] = struct{}{}
	}
	for _, inst := range instances {
		customerID := inst.CustomerProfileID
		if customerID == nil && inst.PlatformIdentityID != nil {
			if ident, err := store.GetIdentity(ctx, *inst.PlatformIdentityID); err == nil && ident.CustomerProfileID != nil {
				customerID = ident.CustomerProfileID
			}
		}
		cust[inst.ID] = customerID
		for _, rule := range rules {
			if !rule.Active {
				continue
			}
			ok, err := selectorMatches(ctx, store, rule.Selector, inst)
			if err != nil {
				return err
			}
			if ok {
				k := key{inst.ID, rule.ProductID}
				qty[k] += rule.Quantity
			}
		}
		for _, ex := range exceptions {
			if ex.InstanceID == inst.ID {
				k := key{inst.ID, ex.ProductID}
				qty[k] += ex.Quantity
			}
		}
	}
	for k, n := range qty {
		if n == 0 {
			continue
		}
		if _, ok := frozen[k]; ok {
			continue
		}
		instID := k.inst
		prodID := k.prod
		addr, err := defaultSnapshot(ctx, store, cust[k.inst])
		if err != nil {
			return err
		}
		addressPinned := false
		if snap, ok := pinned[k]; ok {
			addr = snap
			addressPinned = true
		}
		res := &domain.FulfillmentResult{
			WaveID:                waveID,
			SourceKind:            string(domain.SourceEntitlementInstance),
			EntitlementInstanceID: &instID,
			CustomerProfileID:     cust[k.inst],
			ProductItemID:         &prodID,
			Quantity:              n,
			Address:               addr,
			AddressPinned:         addressPinned,
		}
		if inst, err := store.GetInstance(ctx, instID); err == nil {
			if line, err := store.GetFactLine(ctx, inst.InputFactLineID); err == nil {
				fact, ferr := store.GetFact(ctx, line.FactID)
				if ferr == nil {
					fid := fact.ID
					res.InputFactID = &fid
					lid := line.ID
					res.InputFactLineID = &lid
				}
			}
		}
		if err := store.CreateResult(ctx, res); err != nil {
			return err
		}
	}
	return nil
}

// selectorMatches reports whether an entitlement instance falls under a rule
// selector. Every read goes through the explicit store argument so callers
// control which transaction or connection the lookup joins.
func selectorMatches(ctx context.Context, store domain.Store, sel domain.EntitlementSelector, inst domain.EntitlementInstance) (bool, error) {
	switch sel.Type {
	case string(domain.SelectorWaveAll):
		return true, nil
	case string(domain.SelectorInstance):
		return sel.InstanceID != nil && *sel.InstanceID == inst.ID, nil
	case string(domain.SelectorPlatformLevel):
		if inst.MembershipLevel != sel.Level {
			return false, nil
		}
		if inst.PlatformIdentityID == nil {
			return false, nil
		}
		ident, err := store.GetIdentity(ctx, *inst.PlatformIdentityID)
		if err != nil {
			return false, err
		}
		return ident.PlatformID == sel.PlatformID, nil
	default:
		return false, ErrInvalidSelector
	}
}

func (ws *Workspace) ListResultViews(ctx context.Context, waveID uint) ([]ResultView, error) {
	results, err := ws.Store.ListResults(ctx, waveID)
	if err != nil {
		return nil, err
	}
	out := make([]ResultView, 0, len(results))
	for _, r := range results {
		v, err := ws.inspectResult(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (ws *Workspace) ProductTotals(ctx context.Context, waveID uint) ([]ProductTotal, error) {
	views, err := ws.ListResultViews(ctx, waveID)
	if err != nil {
		return nil, err
	}
	sum := map[uint]int{}
	for _, v := range views {
		if v.Result.Frozen || v.Result.ProductItemID == nil {
			continue
		}
		sum[*v.Result.ProductItemID] += v.Result.Quantity
	}
	var out []ProductTotal
	for id, n := range sum {
		p, err := ws.Store.GetProduct(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, ProductTotal{ProductID: id, Name: p.Name, Quantity: n})
	}
	return out, nil
}

func (ws *Workspace) SetResultAddress(ctx context.Context, resultID, addressID uint) error {
	r, err := ws.Store.GetResult(ctx, resultID)
	if err != nil {
		return err
	}
	if r.Frozen {
		return fmt.Errorf("address is frozen after factory generate")
	}
	addr, err := ws.Store.GetAddress(ctx, addressID)
	if err != nil {
		return err
	}
	r.Address = snapshotFromAddress(*addr)
	r.AddressPinned = true
	return ws.Store.UpdateResult(ctx, r)
}

func (ws *Workspace) ListRules(ctx context.Context, waveID uint) ([]domain.EntitlementRule, error) {
	return ws.Store.ListRules(ctx, waveID)
}
