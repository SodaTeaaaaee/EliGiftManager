package app

import (
	"context"
	"errors"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// revisionFixture bundles the shared setup for revision tests: a source
// platform with an aligned alias, a customer, and a wave.
type revisionFixture struct {
	ws      *Workspace
	ctx     context.Context
	source  *domain.Platform
	factory *domain.Platform
	cust    *domain.CustomerProfile
	product *domain.ProductItem
	wave    *domain.Wave
}

func newRevisionFixture(t *testing.T) *revisionFixture {
	t.Helper()
	f := &revisionFixture{ws: newTestWorkspace(t), ctx: context.Background()}
	if err := f.ws.EnsureBuiltinPlatforms(f.ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	for _, p := range mustListPlatforms(t, f.ws) {
		pc := p
		switch p.Kind {
		case string(domain.PlatformKindSource):
			f.source = &pc
		case string(domain.PlatformKindFactory):
			f.factory = &pc
		}
	}
	f.cust = &domain.CustomerProfile{DisplayName: "Rev Cust"}
	if err := f.ws.CreateCustomer(f.ctx, f.cust); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	f.product = &domain.ProductItem{Name: "Rev Product", FactoryPlatformID: f.factory.ID, FactorySKU: "REV-SKU-1"}
	if err := f.ws.CreateProduct(f.ctx, f.product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	alias := &domain.ProductAlias{ProductItemID: f.product.ID, PlatformID: f.source.ID, ExternalProductID: "REV-ALIAS"}
	if err := f.ws.CreateAlias(f.ctx, alias); err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}
	wave, err := f.ws.CreateWave(f.ctx, "rev wave", "")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	f.wave = wave
	return f
}

func (f *revisionFixture) ingest(t *testing.T, in IngestFactInput) *domain.InputDocument {
	t.Helper()
	doc := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, doc, []IngestFactInput{in}); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	} else if len(dups) != 0 {
		t.Fatalf("IngestDocument produced duplicates: %+v", dups)
	}
	return doc
}

func TestRevisionCreatedWhenStableIDContentChanges(t *testing.T) {
	f := newRevisionFixture(t)
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "REV-ORD-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-REV",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	}
	f.ingest(t, base)

	// Identical content + stable id: plain duplicate, no revision.
	dup := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, dup, []IngestFactInput{base}); err != nil {
		t.Fatalf("re-ingest identical: %v", err)
	} else if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateRecordOnly) {
		t.Fatalf("identical re-import must be record_only, got %+v", dups)
	}

	// Same stable id, corrected quantity: a pending revision.
	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 5}}
	f.ingest(t, revIn)

	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil {
		t.Fatalf("ListRevisionFacts: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision fact, got %d", len(revisions))
	}
	if revisions[0].RevisesID == nil || revisions[0].RevisionAppliedAt != nil {
		t.Fatalf("revision must be pending and anchored, got %+v", revisions[0])
	}

	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	var origFact domain.InputFact
	for _, fact := range facts {
		if fact.ID == *revisions[0].RevisesID {
			origFact = fact
		}
	}
	revLines, err := f.ws.Store.ListFactLines(f.ctx, revisions[0].ID)
	if err != nil {
		t.Fatalf("ListFactLines: %v", err)
	}
	if len(revLines) != 1 || revLines[0].Quantity != 5 {
		t.Fatalf("revision lines = %+v", revLines)
	}

	// Inbox marks the revision rows pending; assignment is refused.
	rows, err := f.ws.ListInboxRows(f.ctx)
	if err != nil {
		t.Fatalf("ListInboxRows: %v", err)
	}
	pendingRows := 0
	for _, r := range rows {
		if r.Fact.ID == revisions[0].ID {
			pendingRows++
			if !r.RevisionPending {
				t.Fatal("revision row must be marked RevisionPending")
			}
			if r.Line.ProductItemID == nil || *r.Line.ProductItemID != f.product.ID {
				t.Fatalf("revision line must still align through the alias, got %+v", r.Line.ProductItemID)
			}
		}
	}
	if pendingRows != 1 {
		t.Fatalf("expected 1 pending revision row, got %d", pendingRows)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{revLines[0].ID}); !errors.Is(err, ErrRevisionPending) {
		t.Fatalf("AssignLines on pending revision = %v, want ErrRevisionPending", err)
	}

	home, err := f.ws.Home(f.ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.PendingRevisions != 1 {
		t.Fatalf("PendingRevisions = %d, want 1", home.PendingRevisions)
	}

	// Same stable id + same content again while a pending revision exists:
	// still a duplicate of the established fact, not a revision of it.
	dup2 := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, dup2, []IngestFactInput{base}); err != nil {
		t.Fatalf("re-ingest identical after revision: %v", err)
	} else if len(dups) != 1 {
		t.Fatalf("identical re-import must stay a duplicate, got %+v", dups)
	}
	_ = origFact
}

func TestApplyRevisionReplacesRetailLinesAndResults(t *testing.T) {
	f := newRevisionFixture(t)
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "REV-ORD-2",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-REV2",
		Lines: []IngestLine{
			{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2},
			{SourceLineNo: 2, ExternalSKU: "REV-ALIAS", Quantity: 3},
		},
	}
	f.ingest(t, base)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	orig := facts[0]
	origLines, err := f.ws.Store.ListFactLines(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("ListFactLines: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{origLines[0].ID, origLines[1].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	before, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("expected 2 retail results, got %d", len(before))
	}

	// Revision corrects line 1's quantity and drops line 2.
	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 7}}
	f.ingest(t, revIn)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("ListRevisionFacts: %v (%d)", err, len(revisions))
	}
	if err := f.ws.ApplyRevision(f.ctx, revisions[0].ID); err != nil {
		t.Fatalf("ApplyRevision: %v", err)
	}

	applied, err := f.ws.Store.GetFact(f.ctx, revisions[0].ID)
	if err != nil {
		t.Fatalf("GetFact revision: %v", err)
	}
	if applied.RevisionAppliedAt == nil {
		t.Fatal("revision must be marked applied")
	}
	revLines, err := f.ws.Store.ListFactLines(f.ctx, revisions[0].ID)
	if err != nil {
		t.Fatalf("ListFactLines revision after apply: %v", err)
	}
	if len(revLines) != 0 {
		t.Fatalf("revision fact must hand its lines over, got %+v", revLines)
	}
	afterLines, err := f.ws.Store.ListFactLines(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("ListFactLines after apply: %v", err)
	}
	if len(afterLines) != 1 {
		t.Fatalf("expected 1 surviving line on the original fact, got %+v", afterLines)
	}
	if afterLines[0].ID == origLines[0].ID {
		t.Fatalf("original line %d must be deleted; the revision line keeps its own id", origLines[0].ID)
	}
	if afterLines[0].Quantity != 7 {
		t.Fatalf("surviving line quantity = %d, want 7", afterLines[0].Quantity)
	}
	if afterLines[0].WaveID == nil || *afterLines[0].WaveID != f.wave.ID {
		t.Fatalf("surviving line must take over the wave assignment, got %+v", afterLines[0].WaveID)
	}

	after, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults after apply: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("expected 1 rebuilt retail result, got %d", len(after))
	}
	if after[0].Quantity != 7 || after[0].InputFactLineID == nil || *after[0].InputFactLineID != afterLines[0].ID {
		t.Fatalf("rebuilt result = %+v", after[0])
	}

	home, err := f.ws.Home(f.ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.PendingRevisions != 0 || home.RevisionFrozenConflicts != 0 {
		t.Fatalf("home after apply = pending %d conflicts %d, want 0/0", home.PendingRevisions, home.RevisionFrozenConflicts)
	}

	// Applying twice is refused.
	if err := f.ws.ApplyRevision(f.ctx, revisions[0].ID); !errors.Is(err, ErrRevisionApplied) {
		t.Fatalf("second apply = %v, want ErrRevisionApplied", err)
	}
}

func TestApplyRevisionRetargetsMembershipInstance(t *testing.T) {
	f := newRevisionFixture(t)
	ident := &domain.PlatformIdentity{PlatformID: f.source.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "UID-MEM-REV", NormalizedValue: NormalizeIdentity("UID-MEM-REV"), CustomerProfileID: &f.cust.ID}
	if err := f.ws.Store.CreateIdentity(f.ctx, ident); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindMembership),
		StableExternalID: "REV-MEM-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-MEM-REV",
		MembershipLevel:  "captain",
		Lines:            []IngestLine{{SourceLineNo: 1, Quantity: 1}},
	}
	f.ingest(t, base)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	orig := facts[0]
	origLines, err := f.ws.Store.ListFactLines(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("ListFactLines: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{origLines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	rule := &domain.EntitlementRule{WaveID: f.wave.ID, ProductID: f.product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 2, Active: true}
	if err := f.ws.UpsertRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}

	// The revision corrects the source document number and the stub line's
	// quantity. Membership stubs have no SKU to vary, so the header-level
	// order number (part of the fingerprint) plus the quantity keep the
	// revision detection honest.
	revIn := base
	revIn.SourceDocumentNo = "REVISED-DOC-1"
	revIn.Lines = []IngestLine{{SourceLineNo: 1, Quantity: 4}}
	f.ingest(t, revIn)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("ListRevisionFacts: %v (%d)", err, len(revisions))
	}

	if err := f.ws.ApplyRevision(f.ctx, revisions[0].ID); err != nil {
		t.Fatalf("ApplyRevision: %v", err)
	}

	afterLines, err := f.ws.Store.ListFactLines(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("ListFactLines after apply: %v", err)
	}
	if len(afterLines) != 1 || afterLines[0].ID == origLines[0].ID {
		t.Fatalf("the revision line must replace the original line by its own id, got %+v", afterLines)
	}
	inst, err := f.ws.Store.GetInstanceByLine(f.ctx, f.wave.ID, afterLines[0].ID)
	if err != nil {
		t.Fatalf("GetInstanceByLine after apply: %v", err)
	}
	if inst.InputFactLineID != afterLines[0].ID {
		t.Fatalf("instance must be retargeted to the surviving line, got %d", inst.InputFactLineID)
	}
	origAfter, err := f.ws.Store.GetFact(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("GetFact orig: %v", err)
	}
	if origAfter.SourceDocumentNo != "REVISED-DOC-1" {
		t.Fatalf("header must follow the revision, doc no = %q", origAfter.SourceDocumentNo)
	}

	results, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 rebuilt entitlement result, got %d", len(results))
	}
	if results[0].InputFactLineID == nil || *results[0].InputFactLineID != afterLines[0].ID {
		t.Fatalf("rebuilt result must reference the surviving line, got %+v", results[0].InputFactLineID)
	}
	if results[0].CustomerProfileID == nil || *results[0].CustomerProfileID != f.cust.ID {
		t.Fatalf("rebuilt result customer = %+v", results[0].CustomerProfileID)
	}
}

func TestApplyRevisionRefusesClosedWave(t *testing.T) {
	f := newRevisionFixture(t)
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "REV-ORD-CLOSED",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-REV-CLOSED",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	}
	f.ingest(t, base)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	origLines, err := f.ws.Store.ListFactLines(f.ctx, facts[0].ID)
	if err != nil {
		t.Fatalf("ListFactLines: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{origLines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}

	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 4}}
	f.ingest(t, revIn)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("ListRevisionFacts: %v (%d)", err, len(revisions))
	}

	if err := f.ws.CloseWave(f.ctx, f.wave.ID, string(domain.WaveCloseResultClean), ""); err != nil {
		t.Fatalf("CloseWave: %v", err)
	}
	if err := f.ws.ApplyRevision(f.ctx, revisions[0].ID); !errors.Is(err, ErrWaveClosed) {
		t.Fatalf("ApplyRevision on closed wave = %v, want ErrWaveClosed", err)
	}
	// The refusal leaves everything in place: the revision stays pending and
	// the original line keeps its assignment.
	rev, err := f.ws.Store.GetFact(f.ctx, revisions[0].ID)
	if err != nil {
		t.Fatalf("GetFact revision: %v", err)
	}
	if rev.RevisionAppliedAt != nil {
		t.Fatal("refused apply must leave the revision pending")
	}
	line, err := f.ws.Store.GetFactLine(f.ctx, origLines[0].ID)
	if err != nil {
		t.Fatalf("GetFactLine: %v", err)
	}
	if line.WaveID == nil || *line.WaveID != f.wave.ID {
		t.Fatalf("original line must keep its wave assignment, got %+v", line.WaveID)
	}
	if line.Quantity != 2 {
		t.Fatalf("original line must keep its pre-revision quantity, got %d", line.Quantity)
	}
}

func TestApplyRevisionRecordsFrozenConflict(t *testing.T) {
	f := newRevisionFixture(t)
	addr := &domain.RecipientAddress{CustomerProfileID: f.cust.ID, RecipientName: "Rev Recipient", Phone: "13800000000", AddressLine1: "1 Rev St", IsDefault: true}
	if err := f.ws.CreateAddress(f.ctx, addr); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "REV-ORD-3",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-REV3",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	}
	f.ingest(t, base)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	orig := facts[0]
	origLines, err := f.ws.Store.ListFactLines(f.ctx, orig.ID)
	if err != nil {
		t.Fatalf("ListFactLines: %v", err)
	}
	// Attach identity to the customer so the retail result gets a usable address.
	ident, err := f.ws.Store.FindIdentity(f.ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-REV3"))
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if err := f.ws.AttachIdentity(f.ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{origLines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	if _, _, err := f.ws.GenerateFactoryOrder(f.ctx, f.wave.ID, f.factory.ID); err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}

	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 9}}
	f.ingest(t, revIn)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("ListRevisionFacts: %v (%d)", err, len(revisions))
	}
	if err := f.ws.ApplyRevision(f.ctx, revisions[0].ID); err != nil {
		t.Fatalf("ApplyRevision must not be blocked by frozen results: %v", err)
	}

	home, err := f.ws.Home(f.ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.RevisionFrozenConflicts != 1 {
		t.Fatalf("RevisionFrozenConflicts = %d, want 1", home.RevisionFrozenConflicts)
	}

	// The frozen result survives as execution history.
	results, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	frozen := 0
	for _, r := range results {
		if r.Frozen {
			frozen++
		}
	}
	if frozen != 1 {
		t.Fatalf("frozen result must survive the revision, got %d", frozen)
	}
}

func TestDismissRevisionDeletesRevision(t *testing.T) {
	f := newRevisionFixture(t)
	base := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: "REV-ORD-4",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-REV4",
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 2}},
	}
	f.ingest(t, base)
	revIn := base
	revIn.Lines = []IngestLine{{SourceLineNo: 1, ExternalSKU: "REV-ALIAS", Quantity: 3}}
	f.ingest(t, revIn)
	revisions, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("ListRevisionFacts: %v (%d)", err, len(revisions))
	}

	if err := f.ws.DismissRevision(f.ctx, revisions[0].ID); err != nil {
		t.Fatalf("DismissRevision: %v", err)
	}
	after, err := f.ws.Store.ListRevisionFacts(f.ctx)
	if err != nil {
		t.Fatalf("ListRevisionFacts after dismiss: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("dismissed revision must be gone, got %d", len(after))
	}
	if _, err := f.ws.Store.GetFact(f.ctx, revisions[0].ID); err != domain.ErrNotFound {
		t.Fatalf("revision fact must be deleted, got %v", err)
	}
	// The established fact is untouched.
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	if len(facts) != 1 {
		t.Fatalf("expected the original fact to survive, got %d", len(facts))
	}
	// Dismissing a non-revision fact is refused.
	if err := f.ws.DismissRevision(f.ctx, facts[0].ID); !errors.Is(err, ErrNotRevision) {
		t.Fatalf("DismissRevision on non-revision = %v, want ErrNotRevision", err)
	}
}
