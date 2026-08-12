package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestAssembleDemandInboxRowsCountsPendingIntake(t *testing.T) {
	docs := []domain.DemandDocument{{ID: 1, Kind: "membership_entitlement"}}
	lines := []domain.DemandLine{
		{ID: 1, DemandDocumentID: 1, RoutingDisposition: "pending_intake"},
		{ID: 2, DemandDocumentID: 1, RoutingDisposition: "accepted", RecipientInputState: "ready"},
		{ID: 3, DemandDocumentID: 1, RoutingDisposition: "deferred"},
		{ID: 4, DemandDocumentID: 1, RoutingDisposition: "excluded_manual"},
	}
	rows := AssembleDemandInboxRows(docs, nil, lines, nil, nil)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.PendingIntakeCount != 1 || row.TotalLineCount != 4 ||
		row.AcceptedCount != 1 || row.ReadyAcceptedCount != 1 ||
		row.DeferredCount != 1 || row.ExcludedCount != 1 {
		t.Fatalf("unexpected counts: %+v", row)
	}
}
