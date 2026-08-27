package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// MoveLines transfers fact lines into another wave, following the three
// attribution-move rules: lines without fulfillment results (or whose results
// are all unfrozen) move directly — their unfrozen results are revoked and
// rebuilt in the target wave; lines whose results entered a factory order
// (frozen) are refused with ErrLineFrozen. Everything runs in one transaction.
func (ws *Workspace) MoveLines(ctx context.Context, lineIDs []uint, targetWaveID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		target, err := tx.GetWave(ctx, targetWaveID)
		if err != nil {
			return err
		}
		if target.CloseResult != string(domain.WaveCloseResultOpen) {
			return ErrWaveClosed
		}
		sourceWaves := map[uint]struct{}{}
		moved := make([]domain.InputFactLine, 0, len(lineIDs))
		for _, id := range lineIDs {
			line, err := tx.GetFactLine(ctx, id)
			if err != nil {
				return err
			}
			if line.WaveID != nil && *line.WaveID == targetWaveID {
				continue
			}
			fact, err := tx.GetFact(ctx, line.FactID)
			if err != nil {
				return err
			}
			if fact.RevisesID != nil && fact.RevisionAppliedAt == nil {
				return ErrRevisionPending
			}
			if line.WaveID == nil {
				// Unassigned lines enter the target wave like a fresh
				// assignment (instance + source results).
				if err := tws.assignLines(ctx, targetWaveID, []uint{id}); err != nil {
					return err
				}
				moved = append(moved, *line)
				continue
			}
			sourceWaveID := *line.WaveID

			// Revoke the line's results in the source wave; a frozen result
			// means the line entered a factory order and cannot move.
			results, err := tx.ListResults(ctx, sourceWaveID)
			if err != nil {
				return err
			}
			for i := range results {
				r := &results[i]
				if r.InputFactLineID == nil || *r.InputFactLineID != line.ID {
					continue
				}
				if r.Frozen {
					return ErrLineFrozen
				}
				if err := tx.DeleteResult(ctx, r.ID); err != nil {
					return err
				}
			}
			if fact.Kind == string(domain.InputFactKindMembership) {
				inst, err := tx.GetInstanceByLine(ctx, sourceWaveID, line.ID)
				if err == domain.ErrNotFound {
					inst = nil
				} else if err != nil {
					return err
				}
				if inst != nil {
					inst.WaveID = targetWaveID
					if err := tx.UpdateInstance(ctx, inst); err != nil {
						return err
					}
				}
			}
			wid := targetWaveID
			line.WaveID = &wid
			if err := tx.UpdateFactLine(ctx, line); err != nil {
				return err
			}
			sourceWaves[sourceWaveID] = struct{}{}
			moved = append(moved, *line)
		}
		for i := range moved {
			line := moved[i]
			if line.WaveID == nil {
				continue
			}
			fact, err := tx.GetFact(ctx, line.FactID)
			if err != nil {
				return err
			}
			switch fact.Kind {
			case string(domain.InputFactKindRetailOrder):
				if err := tws.ensureRetailResult(ctx, targetWaveID, fact, &moved[i]); err != nil {
					return err
				}
			case string(domain.InputFactKindOperatorGrant):
				if err := tws.ensureGrantResult(ctx, targetWaveID, fact, &moved[i]); err != nil {
					return err
				}
			}
		}
		// Membership entitlements are instance-backed: rebuild both sides so
		// the source wave loses the moved instance's results and the target
		// wave gains them.
		for waveID := range sourceWaves {
			if err := recompute(ctx, tx, waveID); err != nil {
				return err
			}
		}
		return recompute(ctx, tx, targetWaveID)
	})
}
