package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// revisionFrozenMarker tags an applied revision whose replaced lines still
// had frozen (factory) results. Home derives the warning count next to the
// pending-revisions bucket from this marker; no extra entity is kept for it.
const revisionFrozenMarker = `"revision_frozen_conflict":true`

// revisionMarkerExtraData replaces an applied revision's extra data once its
// original recipient snapshot has been copied onto the revised fact.
func revisionMarkerExtraData() string {
	return `{"revision_frozen_conflict":true}`
}

// ApplyRevision applies a pending revision to the fact it revises, inside one
// transaction. The revision's lines move onto the original fact (keeping
// their own line ids and any wave assignment), the original lines and their
// unfrozen results are dropped, the original fact's header content follows
// the revision, and waves that lost lines are recomputed. Frozen results
// survive as execution history: applying is not blocked, but the revision is
// marked with a frozen-conflict flag the home view surfaces as a warning.
func (ws *Workspace) ApplyRevision(ctx context.Context, factID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		tws := ws.withStore(tx)
		rev, err := tx.GetFact(ctx, factID)
		if err != nil {
			return err
		}
		if rev.RevisesID == nil {
			return ErrNotRevision
		}
		if rev.RevisionAppliedAt != nil {
			return ErrRevisionApplied
		}
		orig, err := tx.GetFact(ctx, *rev.RevisesID)
		if err != nil {
			return err
		}
		origLines, err := tx.ListFactLines(ctx, orig.ID)
		if err != nil {
			return err
		}
		revLines, err := tx.ListFactLines(ctx, rev.ID)
		if err != nil {
			return err
		}

		// Applying a revision moves lines and rebuilds results inside the
		// waves holding the original lines, so a closed wave refuses it: a
		// late input must wait for an explicit wave reopen.
		checkedWaves := map[uint]struct{}{}
		for _, ln := range origLines {
			if ln.WaveID == nil {
				continue
			}
			if _, seen := checkedWaves[*ln.WaveID]; seen {
				continue
			}
			checkedWaves[*ln.WaveID] = struct{}{}
			wave, err := tx.GetWave(ctx, *ln.WaveID)
			if err != nil {
				return err
			}
			if wave.CloseResult != string(domain.WaveCloseResultOpen) {
				return ErrWaveClosed
			}
		}

		// The fact's header content follows the revision; the original fact
		// id stays stable so existing results keep their InputFactID.
		orig.PlatformIdentityID = rev.PlatformIdentityID
		orig.CustomerProfileID = rev.CustomerProfileID
		orig.MembershipLevel = rev.MembershipLevel
		orig.SourceDocumentNo = rev.SourceDocumentNo
		orig.SourceCreatedAt = rev.SourceCreatedAt
		if rev.ExtraData != "" {
			orig.ExtraData = rev.ExtraData
		}

		pairOld, pairNew, removed, added := pairRevisionLines(origLines, revLines)

		frozenConflict := false
		affected := map[uint]struct{}{}
		dropOld := func(line domain.InputFactLine) error {
			conflict, err := tws.dropUnfrozenLineResults(ctx, tx, line)
			if err != nil {
				return err
			}
			if conflict {
				frozenConflict = true
			}
			return tx.DeleteFactLine(ctx, line.ID)
		}

		for i := range pairOld {
			oldLine, newLine := &pairOld[i], &pairNew[i]
			if oldLine.WaveID != nil {
				wid := *oldLine.WaveID
				if orig.Kind == string(domain.InputFactKindMembership) {
					if err := tws.retargetInstance(ctx, tx, wid, oldLine.ID, newLine.ID, rev); err != nil {
						return err
					}
				}
				newLine.WaveID = &wid
				affected[wid] = struct{}{}
			}
			newLine.FactID = orig.ID
			if err := tx.UpdateFactLine(ctx, newLine); err != nil {
				return err
			}
			if err := dropOld(*oldLine); err != nil {
				return err
			}
		}
		for _, line := range removed {
			if line.WaveID != nil {
				wid := *line.WaveID
				affected[wid] = struct{}{}
				if orig.Kind == string(domain.InputFactKindMembership) {
					if err := tws.dropInstance(ctx, tx, wid, line.ID); err != nil {
						return err
					}
				}
			}
			if err := dropOld(line); err != nil {
				return err
			}
		}
		for i := range added {
			line := added[i]
			line.FactID = orig.ID
			if err := tx.UpdateFactLine(ctx, &line); err != nil {
				return err
			}
		}

		now := ws.Now()
		rev.RevisionAppliedAt = &now
		if frozenConflict {
			rev.ExtraData = revisionMarkerExtraData()
		}
		if err := tx.UpdateFact(ctx, rev); err != nil {
			return err
		}
		if err := tx.UpdateFact(ctx, orig); err != nil {
			return err
		}

		for waveID := range affected {
			for i := range pairNew {
				line := pairNew[i]
				if line.WaveID == nil || *line.WaveID != waveID {
					continue
				}
				switch orig.Kind {
				case string(domain.InputFactKindRetailOrder):
					if err := tws.ensureRetailResult(ctx, waveID, orig, &line); err != nil {
						return err
					}
				case string(domain.InputFactKindOperatorGrant):
					if err := tws.ensureGrantResult(ctx, waveID, orig, &line); err != nil {
						return err
					}
				}
			}
			if orig.Kind == string(domain.InputFactKindMembership) {
				// Entitlement results are instance-backed; the rebuild reads
				// the retargeted instances and refreshed header.
				if err := recompute(ctx, tx, waveID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DismissRevision refuses a pending revision by deleting the revision fact
// and its lines. The tradeoff is deliberate: a refused revision carries no
// responsibility of its own, and the import document (with its raw payload)
// already records that the version arrived, so no audit entity is kept.
func (ws *Workspace) DismissRevision(ctx context.Context, factID uint) error {
	return ws.Store.WithTx(ctx, func(tx domain.Store) error {
		rev, err := tx.GetFact(ctx, factID)
		if err != nil {
			return err
		}
		if rev.RevisesID == nil {
			return ErrNotRevision
		}
		if rev.RevisionAppliedAt != nil {
			return ErrRevisionApplied
		}
		lines, err := tx.ListFactLines(ctx, rev.ID)
		if err != nil {
			return err
		}
		for _, ln := range lines {
			if err := tx.DeleteFactLine(ctx, ln.ID); err != nil {
				return err
			}
		}
		return tx.DeleteFact(ctx, rev.ID)
	})
}

// pairRevisionLines pairs original and revision lines by source line number;
// unpaired originals are removed and unpaired revisions are added. Pairing
// preserves line ids (and wave assignments) for content that persists across
// the revision.
func pairRevisionLines(oldLines, newLines []domain.InputFactLine) (pairOld, pairNew, removed, added []domain.InputFactLine) {
	used := make([]bool, len(newLines))
	for _, o := range oldLines {
		matched := false
		for i := range newLines {
			if used[i] || newLines[i].SourceLineNo != o.SourceLineNo {
				continue
			}
			used[i] = true
			pairOld = append(pairOld, o)
			pairNew = append(pairNew, newLines[i])
			matched = true
			break
		}
		if !matched {
			removed = append(removed, o)
		}
	}
	for i := range newLines {
		if !used[i] {
			added = append(added, newLines[i])
		}
	}
	return pairOld, pairNew, removed, added
}

// dropUnfrozenLineResults deletes the unfrozen results a line produced and
// reports whether any frozen result still claims the line.
func (ws *Workspace) dropUnfrozenLineResults(ctx context.Context, store domain.Store, line domain.InputFactLine) (bool, error) {
	if line.WaveID == nil {
		return false, nil
	}
	results, err := store.ListResults(ctx, *line.WaveID)
	if err != nil {
		return false, err
	}
	frozen := false
	for i := range results {
		r := &results[i]
		if r.InputFactLineID == nil || *r.InputFactLineID != line.ID {
			continue
		}
		if r.Frozen {
			frozen = true
			continue
		}
		if err := store.DeleteResult(ctx, r.ID); err != nil {
			return false, err
		}
	}
	return frozen, nil
}

// retargetInstance moves a membership entitlement instance from the replaced
// line to the surviving revision line and refreshes its header-derived
// fields, all inside the caller's transaction.
func (ws *Workspace) retargetInstance(ctx context.Context, store domain.Store, waveID, oldLineID, newLineID uint, rev *domain.InputFact) error {
	inst, err := store.GetInstanceByLine(ctx, waveID, oldLineID)
	if err == domain.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	inst.InputFactLineID = newLineID
	inst.CustomerProfileID = rev.CustomerProfileID
	inst.PlatformIdentityID = rev.PlatformIdentityID
	inst.MembershipLevel = rev.MembershipLevel
	return store.UpdateInstance(ctx, inst)
}

// dropInstance removes the entitlement instance of a line the revision
// removed entirely.
func (ws *Workspace) dropInstance(ctx context.Context, store domain.Store, waveID, lineID uint) error {
	inst, err := store.GetInstanceByLine(ctx, waveID, lineID)
	if err == domain.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	return store.DeleteInstance(ctx, inst.ID)
}
