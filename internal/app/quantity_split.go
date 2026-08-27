package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// UpsertQuantitySplitRule stores a wave-scoped quantity split (replacing its
// components wholesale) and re-derives the retail results of every wave line
// the rule covers, in one transaction. Frozen results are left untouched —
// execution history is not rewritten.
func (ws *Workspace) UpsertQuantitySplitRule(ctx context.Context, rule *domain.QuantitySplitRule) error {
	if rule.WaveID == 0 || rule.PlatformID == 0 || rule.ExternalKey == "" {
		return fmt.Errorf("quantity split rule needs wave, platform, and external key")
	}
	if len(rule.Components) == 0 {
		return fmt.Errorf("quantity split rule needs at least one component")
	}
	for _, c := range rule.Components {
		if c.ProductItemID == 0 || c.Quantity <= 0 {
			return fmt.Errorf("quantity split components need a product and a positive quantity")
		}
	}
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		wave, err := tx.GetWave(ctx, rule.WaveID)
		if err != nil {
			return err
		}
		if wave.CloseResult != string(domain.WaveCloseResultOpen) {
			return ErrWaveClosed
		}
		if err := tx.UpsertQuantitySplitRule(ctx, rule); err != nil {
			return err
		}
		return tws.rebuildSplitCoveredLines(ctx, tx, rule.WaveID, rule.PlatformID, rule.ExternalKey)
	})
}

// DeleteQuantitySplitRule removes a rule (and its components) and re-derives
// the covered lines so they fall back to plain (or bundle) alignment.
func (ws *Workspace) DeleteQuantitySplitRule(ctx context.Context, id uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		rule, err := tx.GetQuantitySplitRule(ctx, id)
		if err != nil {
			return err
		}
		if err := tx.DeleteQuantitySplitRule(ctx, id); err != nil {
			return err
		}
		return ws.withStore(tx).rebuildSplitCoveredLines(ctx, tx, rule.WaveID, rule.PlatformID, rule.ExternalKey)
	})
}

func (ws *Workspace) ListQuantitySplitRules(ctx context.Context, waveID uint) ([]domain.QuantitySplitRule, error) {
	return ws.Store.ListQuantitySplitRules(ctx, waveID)
}

// rebuildSplitCoveredLines deletes the unfrozen retail results of every wave
// line matching (platform, external key) and re-runs ensureRetailResult so
// the current split state (hit, miss, or not-summing) is reflected.
func (ws *Workspace) rebuildSplitCoveredLines(ctx context.Context, tx domain.Store, waveID, platformID uint, externalKey string) error {
	lines, err := tx.ListFactLinesByWave(ctx, waveID)
	if err != nil {
		return err
	}
	for i := range lines {
		line := &lines[i]
		if line.ExternalSKU != externalKey {
			continue
		}
		fact, err := tx.GetFact(ctx, line.FactID)
		if err != nil {
			return err
		}
		if fact.PlatformID != platformID || fact.Kind != string(domain.InputFactKindRetailOrder) {
			continue
		}
		results, err := tx.ListResults(ctx, waveID)
		if err != nil {
			return err
		}
		for j := range results {
			r := &results[j]
			if r.InputFactLineID == nil || *r.InputFactLineID != line.ID || r.SourceKind != string(domain.SourceRetailLine) {
				continue
			}
			if r.Frozen {
				continue
			}
			if err := tx.DeleteResult(ctx, r.ID); err != nil {
				return err
			}
		}
		if err := ws.ensureRetailResult(ctx, waveID, fact, line); err != nil {
			return err
		}
	}
	return nil
}

// applyQuantitySplit resolves the split rule covering a retail line, if any.
// It runs before bundle mapping: the wave-scoped exception explains this
// specific line, overriding the alias's global bundle composition.
//
// When the components sum to the line quantity the line expands into one
// result per component with the component's absolute quantity. When they do
// not sum, no results are generated; instead a placeholder result carries the
// quantity_split_not_summing block so the workbench shows why nothing ships.
func (ws *Workspace) applyQuantitySplit(ctx context.Context, waveID uint, fact *domain.InputFact, line *domain.InputFactLine) (bool, error) {
	if line.ExternalSKU == "" {
		return false, nil
	}
	rule, err := ws.Store.FindQuantitySplitRule(ctx, waveID, fact.PlatformID, line.ExternalSKU)
	if err == domain.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	sum := 0
	for _, comp := range rule.Components {
		sum += comp.Quantity
	}
	fid := fact.ID
	lid := line.ID
	custID := retailCustomerID(ctx, ws.Store, fact)
	addr, err := defaultSnapshot(ctx, ws.Store, custID)
	if err != nil {
		return true, err
	}
	if sum != line.Quantity {
		res := &domain.FulfillmentResult{
			WaveID:            waveID,
			SourceKind:        string(domain.SourceRetailLine),
			InputFactLineID:   &lid,
			InputFactID:       &fid,
			CustomerProfileID: custID,
			Quantity:          line.Quantity,
			Address:           addr,
			ExtraData:         fmt.Sprintf(`{"quantity_split_not_summing":true,"quantity_split_rule_id":%d}`, rule.ID),
		}
		return true, ws.Store.CreateResult(ctx, res)
	}
	for _, comp := range rule.Components {
		pid := comp.ProductItemID
		res := &domain.FulfillmentResult{
			WaveID:            waveID,
			SourceKind:        string(domain.SourceRetailLine),
			InputFactLineID:   &lid,
			InputFactID:       &fid,
			CustomerProfileID: custID,
			ProductItemID:     &pid,
			Quantity:          comp.Quantity,
			Address:           addr,
		}
		if err := ws.Store.CreateResult(ctx, res); err != nil {
			return true, err
		}
	}
	return true, nil
}
