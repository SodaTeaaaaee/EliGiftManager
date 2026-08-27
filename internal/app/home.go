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
		if !r.Assigned {
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
		blocked := 0
		unfrozen := 0
		for _, v := range views {
			if v.WorkState == domain.WorkStateBlocked {
				blocked++
			}
			if !v.Result.Frozen {
				unfrozen++
			}
		}
		out.BlockedResults += blocked
		if unfrozen > 0 {
			out.ResidualClose++
		}
	}
	return out, nil
}
