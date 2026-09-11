package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestHomeBucketUnits(t *testing.T) {
	f := newRevisionFixture(t)
	ctx := f.ctx

	// One unassigned, unaligned retail line. Its identity is new, so import
	// auto-creates a profile and attaches it: nothing is left unattached.
	in := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "HOME-ORD-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-HOME",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "UNKNOWN-SKU", Quantity: 1}},
	}
	f.ingest(t, in)
	// A hand-made identity without a profile still counts as unattached
	// work once a fact points at it.
	orphan := &domain.PlatformIdentity{PlatformID: f.source.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "UID-ORPHAN", NormalizedValue: NormalizeIdentity("UID-ORPHAN")}
	if err := f.ws.Store.CreateIdentity(ctx, orphan); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	orphanDoc := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if err := f.ws.Store.CreateDocument(ctx, orphanDoc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	orphanFact := &domain.InputFact{DocumentID: &orphanDoc.ID, PlatformID: f.source.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: "HOME-ORPHAN", PlatformIdentityID: &orphan.ID}
	if err := f.ws.Store.CreateFact(ctx, orphanFact); err != nil {
		t.Fatalf("CreateFact: %v", err)
	}
	if err := f.ws.Store.CreateFactLine(ctx, &domain.InputFactLine{FactID: orphanFact.ID, SourceLineNo: 1, Quantity: 1}); err != nil {
		t.Fatalf("CreateFactLine: %v", err)
	}

	// An open wave with unfrozen results must NOT count as residual close.
	grantRes, err := f.ws.CreateGrant(ctx, f.wave.ID, f.cust.ID, f.product.ID, 2)
	if err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if grantRes.ID == 0 {
		t.Fatal("grant result missing")
	}
	// A separate wave closed with residual counts once.
	residualWave, err := f.ws.CreateWave(ctx, "residual wave", "")
	if err != nil {
		t.Fatalf("CreateWave residual: %v", err)
	}
	if err := f.ws.CloseWave(ctx, residualWave.ID, string(domain.WaveCloseResultResidual), "leftovers"); err != nil {
		t.Fatalf("CloseWave residual: %v", err)
	}

	home, err := f.ws.Home(ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.ResidualClose != 1 {
		t.Fatalf("ResidualClose = %d, want 1 (residual-closed wave only)", home.ResidualClose)
	}
	if home.Unassigned != 2 {
		t.Fatalf("Unassigned = %d, want 2 (imported retail line + orphan membership line)", home.Unassigned)
	}
	if home.AlignmentConflict != 1 {
		t.Fatalf("AlignmentConflict = %d, want 1", home.AlignmentConflict)
	}
	// The unattached identity bucket counts inbox rows (lines), not the
	// platform-identity row on its own; the imported line auto-attached, only
	// the orphan line counts.
	if home.IdentityUnattached != 1 {
		t.Fatalf("IdentityUnattached = %d, want 1 row", home.IdentityUnattached)
	}
	if home.PendingRevisions != 0 {
		t.Fatalf("PendingRevisions = %d, want 0", home.PendingRevisions)
	}
}

// TestHomeUnassignedExcludesPendingRevisions pins the bucket combination: a
// pending revision row is not actionable assign work, so while it is the only
// unassigned inbox content the Unassigned bucket stays empty even though
// PendingRevisions counts it.
func TestHomeUnassignedExcludesPendingRevisions(t *testing.T) {
	f := newRevisionFixture(t)
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "HOME-REV-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-HOME-REV",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	}
	f.ingest(t, base)
	// The established fact's line is assigned, so the revision row is the
	// only unassigned inbox content left.
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	origLines, err := f.ws.Store.ListFactLines(f.ctx, facts[0].ID)
	if err != nil || len(origLines) != 1 {
		t.Fatalf("ListFactLines: %v (%d)", err, len(origLines))
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{origLines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 5}}
	f.ingest(t, revIn)

	home, err := f.ws.Home(f.ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.PendingRevisions != 1 {
		t.Fatalf("PendingRevisions = %d, want 1", home.PendingRevisions)
	}
	if home.Unassigned != 0 {
		t.Fatalf("Unassigned = %d, want 0 while only pending-revision rows are unassigned", home.Unassigned)
	}
	if home.AlignmentConflict != 0 {
		t.Fatalf("aligned revision rows must not count as alignment conflicts, got %d", home.AlignmentConflict)
	}
}
