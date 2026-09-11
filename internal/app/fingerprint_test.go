package app

import (
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestFactFingerprintIdentityTypeOnlyParticipatesWithValue(t *testing.T) {
	// An empty identity value drops the type from the fingerprint, whatever
	// the input claimed: stored identity-less facts never carry a type, so an
	// explicit type on an empty value must not look like different content.
	emptyWithType := IngestFactInput{
		Kind:         string(domain.InputFactKindRetailOrder),
		IdentityType: string(domain.IdentityTypeEmail),
		Lines:        []IngestLine{{Quantity: 1}},
	}
	emptyWithoutType := emptyWithType
	emptyWithoutType.IdentityType = ""
	if a, b := emptyWithType.ingestFingerprints(), emptyWithoutType.ingestFingerprints(); a[0] != b[0] {
		t.Fatalf("empty identity value must hash the same with and without a type: %q vs %q", a[0], b[0])
	}

	// With a value present, the type does discriminate, and an omitted type
	// defaults to platform uid.
	uid := IngestFactInput{
		Kind:          string(domain.InputFactKindRetailOrder),
		IdentityType:  string(domain.IdentityTypePlatformUID),
		IdentityValue: "U-1",
		Lines:         []IngestLine{{Quantity: 1}},
	}
	uidDefaulted := uid
	uidDefaulted.IdentityType = ""
	phone := uid
	phone.IdentityType = string(domain.IdentityTypeEmail)
	if a, b := uid.ingestFingerprints(), uidDefaulted.ingestFingerprints(); a[0] != b[0] {
		t.Fatalf("omitted type must default to platform uid: %q vs %q", a[0], b[0])
	}
	if a, b := uid.ingestFingerprints(), phone.ingestFingerprints(); a[0] == b[0] {
		t.Fatal("different identity types on the same value must produce different fingerprints")
	}
}

func TestFactFingerprintIncludesSourceDocumentNo(t *testing.T) {
	day := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	base := func(docNo string) string {
		return factFingerprint(string(domain.IdentityTypePlatformUID), "u-1", "SKU-1", docNo, 2, &day)
	}
	if base("ORD-1") == base("ORD-2") {
		t.Fatal("different source order numbers must produce different fingerprints")
	}
	if base("ORD-1") != base("ORD-1") {
		t.Fatal("same order number must produce the same fingerprint")
	}
	// Membership and grant facts carry no order number: the empty slot adds
	// no discrimination on its own.
	if base("") != base("") {
		t.Fatal("empty order numbers must produce the same fingerprint")
	}
}

// TestSameContentFingerprintsComparesMultisets pins the duplicate/revision
// boundary on repeated lines: two identical lines are not the same content as
// two different lines, even though both fingerprint sets have the same members.
func TestSameContentFingerprintsComparesMultisets(t *testing.T) {
	f := newRevisionFixture(t)

	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "FP-MULTI-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-FP-MULTI",
		Lines: []IngestLine{
			{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2},
			{SourceLineNo: 2, ExternalSKU: "REV-ALIAS", Quantity: 2},
		},
	}
	f.ingest(t, base)

	// Identical multiset [X, X]: plain duplicate.
	dup := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, dup, []IngestFactInput{base}); err != nil {
		t.Fatalf("identical re-import: %v", err)
	} else if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateRecordOnly) {
		t.Fatalf("identical multiset must be record_only, got %+v", dups)
	}

	// Same set of fingerprint values but [X, Y]: corrected content, a revision.
	changed := base
	changed.Lines = []IngestLine{
		{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2},
		{SourceLineNo: 2, ExternalSKU: "OTHER-ALIAS", Quantity: 2},
	}
	f.ingest(t, changed)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil {
		t.Fatalf("ListRevisionFacts: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("[X, X] vs [X, Y] must be a revision, not a duplicate; got %d revisions", len(revisions))
	}
}
