package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestHomeBucketUnits(t *testing.T) {
	f := newRevisionFixture(t)
	ctx := f.ctx

	// One unassigned, unaligned, unattached retail line.
	in := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "HOME-ORD-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-HOME",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "UNKNOWN-SKU", Quantity: 1}},
	}
	f.ingest(t, in)

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
	if home.Unassigned != 1 {
		t.Fatalf("Unassigned = %d, want 1", home.Unassigned)
	}
	if home.AlignmentConflict != 1 {
		t.Fatalf("AlignmentConflict = %d, want 1", home.AlignmentConflict)
	}
	// The unattached identity bucket counts inbox rows (lines), not the
	// platform-identity row on its own.
	if home.IdentityUnattached != 1 {
		t.Fatalf("IdentityUnattached = %d, want 1 row", home.IdentityUnattached)
	}
	if home.PendingRevisions != 0 {
		t.Fatalf("PendingRevisions = %d, want 0", home.PendingRevisions)
	}
}
