package app

import (
	"context"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type HomeBuckets struct {
	Unassigned         int
	DuplicateAsk       int
	AlignmentConflict  int
	IdentityUnattached int
	PendingRevisions   int
	BlockedResults     int
	WritebackFailed    int
	ResidualClose      int
	// RevisionFrozenConflicts warns that applied revisions replaced lines
	// whose results were already frozen into factory orders.
	RevisionFrozenConflicts int
	RecentWaves             []domain.Wave
}

func (ws *Workspace) Home(ctx context.Context) (HomeBuckets, error) {
	var out HomeBuckets
	rows, err := ws.ListInboxRows(ctx)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		// Pending revision rows are not actionable assigns: their next action
		// is apply or dismiss, so they must not surface as unassigned work.
		if !r.Assigned && !r.RevisionPending {
			out.Unassigned++
		}
		if r.Unaligned {
			out.AlignmentConflict++
		}
		if r.Unattached {
			out.IdentityUnattached++
		}
	}
	dups, err := ws.Store.ListOpenDuplicates(ctx)
	if err != nil {
		return out, err
	}
	out.DuplicateAsk = len(dups)
	revisions, err := ws.Store.ListRevisionFacts(ctx)
	if err != nil {
		return out, err
	}
	for _, rev := range revisions {
		if rev.RevisionAppliedAt == nil {
			out.PendingRevisions++
			continue
		}
		if strings.Contains(rev.ExtraData, revisionFrozenMarker) {
			out.RevisionFrozenConflicts++
		}
	}
	failed, err := ws.Store.ListFailedWritebacks(ctx)
	if err != nil {
		return out, err
	}
	out.WritebackFailed = len(failed)
	waves, err := ws.Store.ListWaves(ctx)
	if err != nil {
		return out, err
	}
	if len(waves) > 8 {
		out.RecentWaves = waves[:8]
	} else {
		out.RecentWaves = waves
	}
	for _, w := range waves {
		// Only waves actually closed with residual count here. An open wave
		// with unfrozen results is work in progress, not a pending-home
		// concern.
		if w.CloseResult == string(domain.WaveCloseResultResidual) {
			out.ResidualClose++
		}
		if w.CloseResult != string(domain.WaveCloseResultOpen) {
			continue
		}
		views, err := ws.ListResultViews(ctx, w.ID)
		if err != nil {
			return out, err
		}
		for _, v := range views {
			if v.WorkState == domain.WorkStateBlocked {
				out.BlockedResults++
			}
		}
	}
	return out, nil
}
