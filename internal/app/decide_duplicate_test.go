package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestDecideDuplicateReplaysNewResponsibility(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.Local)
	day := now.Add(-24 * time.Hour)
	ws, gdb, source := setupDuplicateWindowWorkspace(t)
	ws.Now = func() time.Time { return now }
	ctx := context.Background()

	in := IngestFactInput{
		Kind:            string(domain.InputFactKindRetailOrder),
		IdentityType:    string(domain.IdentityTypePlatformUID),
		IdentityValue:   "UID-DECIDE",
		SourceCreatedAt: &day,
		Lines:           []IngestLine{{SourceLineNo: 1, ExternalSKU: "SKU-DECIDE", Quantity: 2}},
	}
	firstDoc := &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "1.csv"}
	if _, dups, err := ws.IngestDocument(ctx, firstDoc, []IngestFactInput{in}); err != nil {
		t.Fatalf("first ingest: %v", err)
	} else if len(dups) != 0 {
		t.Fatalf("first ingest produced duplicates: %v", dups)
	}
	// Rewind into the ask window.
	facts := mustFactsByPlatform(t, ws, source.ID)
	rewind := func(factID uint, to time.Time) {
		t.Helper()
		if err := gdb.Exec("UPDATE input_facts SET created_at = ? WHERE id = ?", to, factID).Error; err != nil {
			t.Fatalf("rewind: %v", err)
		}
	}
	rewind(facts[0].ID, now.Add(-48*time.Hour))

	secondDoc := &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "2.csv"}
	_, dups, err := ws.IngestDocument(ctx, secondDoc, []IngestFactInput{in})
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if len(dups) != 1 || dups[0].Verdict != string(domain.DuplicateAskOperator) {
		t.Fatalf("expected ask_operator observation, got %+v", dups)
	}
	obsID := dups[0].ID

	open, err := ws.Store.ListOpenDuplicates(ctx)
	if err != nil || len(open) != 1 {
		t.Fatalf("ListOpenDuplicates: %v (%d)", err, len(open))
	}

	// Accepting the duplicate keeps record_only: no fact is created.
	if err := ws.DecideDuplicate(ctx, obsID, true); err != nil {
		t.Fatalf("DecideDuplicate accept: %v", err)
	}
	if facts := mustFactsByPlatform(t, ws, source.ID); len(facts) != 1 {
		t.Fatalf("accept must not create a fact, got %d", len(facts))
	}

	// A second, identical ask observation: refusing it replays the input.
	rewind(facts[0].ID, now.Add(-48*time.Hour))
	thirdDoc := &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "3.csv"}
	_, dups, err = ws.IngestDocument(ctx, thirdDoc, []IngestFactInput{in})
	if err != nil {
		t.Fatalf("third ingest: %v", err)
	}
	if len(dups) != 1 {
		t.Fatalf("expected another ask observation, got %+v", dups)
	}
	if err := ws.DecideDuplicate(ctx, dups[0].ID, false); err != nil {
		t.Fatalf("DecideDuplicate refuse: %v", err)
	}
	after := mustFactsByPlatform(t, ws, source.ID)
	if len(after) != 2 {
		t.Fatalf("refuse must replay the fact, got %d facts", len(after))
	}
	var replayed domain.InputFact
	for _, f := range after {
		if f.ID != facts[0].ID {
			replayed = f
		}
	}
	if replayed.ID == 0 || replayed.StableExternalID != "" || replayed.Kind != string(domain.InputFactKindRetailOrder) {
		t.Fatalf("replayed fact = %+v", replayed)
	}
	if replayed.DocumentID == nil || *replayed.DocumentID != thirdDoc.ID {
		t.Fatalf("replayed fact must sit under the asking document %d, got %+v", thirdDoc.ID, replayed.DocumentID)
	}
	lines, err := ws.Store.ListFactLines(ctx, replayed.ID)
	if err != nil || len(lines) != 1 || lines[0].ExternalSKU != "SKU-DECIDE" || lines[0].Quantity != 2 {
		t.Fatalf("replayed lines = %+v (%v)", lines, err)
	}
	// The identity link came along.
	if replayed.PlatformIdentityID == nil {
		t.Fatal("replayed fact must link the identity")
	}

	// The observation is decided and leaves the open list.
	open, err = ws.Store.ListOpenDuplicates(ctx)
	if err != nil || len(open) != 0 {
		t.Fatalf("open duplicates after decisions: %v (%d)", err, len(open))
	}

	// Deciding an unknown observation is ErrNotFound.
	if err := ws.DecideDuplicate(ctx, 999999, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("DecideDuplicate unknown = %v, want ErrNotFound", err)
	}

	// The replayed observation's snapshot decodes losslessly.
	var snap duplicateInputSnapshot
	if err := json.Unmarshal([]byte(dups[0].ExtraData), &snap); err != nil {
		t.Fatalf("snapshot decode: %v", err)
	}
	if snap.Fact.IdentityValue != "UID-DECIDE" {
		t.Fatalf("snapshot identity = %q", snap.Fact.IdentityValue)
	}
}

// TestDecideDuplicateRefusesZeroLineReplay pins the replay defense: an ask
// observation whose snapshot carries no lines cannot rebuild a meaningful
// retail fact, so claiming a new responsibility from it is refused instead of
// creating a zero-line fact. Membership (and grant) inputs legitimately carry
// no lines and still replay through the single-line stub.
func TestDecideDuplicateRefusesZeroLineReplay(t *testing.T) {
	ws, _, source := setupDuplicateWindowWorkspace(t)
	ctx := context.Background()
	doc := &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail", OriginalName: "zero.csv"}
	if err := ws.Store.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}

	mkObs := func(kind string) uint {
		t.Helper()
		snap, err := json.Marshal(duplicateInputSnapshot{Fact: IngestFactInput{Kind: kind, IdentityValue: "UID-ZERO"}})
		if err != nil {
			t.Fatalf("marshal snapshot: %v", err)
		}
		obs := &domain.DuplicateObservation{
			DocumentID:     doc.ID,
			ExistingFactID: 1,
			Verdict:        string(domain.DuplicateAskOperator),
			Reason:         "within_ask_window",
			ExtraData:      string(snap),
		}
		if err := ws.Store.CreateDuplicate(ctx, obs); err != nil {
			t.Fatalf("CreateDuplicate: %v", err)
		}
		return obs.ID
	}

	retailID := mkObs(string(domain.InputFactKindRetailOrder))
	err := ws.DecideDuplicate(ctx, retailID, false)
	if err == nil || !strings.Contains(err.Error(), "zero-line") {
		t.Fatalf("zero-line retail replay must be refused by the zero-line guard, got %v", err)
	}
	if facts := mustFactsByPlatform(t, ws, source.ID); len(facts) != 0 {
		t.Fatalf("refused replay must not create a fact, got %d", len(facts))
	}
	// The observation stays undecided so the operator is not silently past it.
	open, oerr := ws.Store.ListOpenDuplicates(ctx)
	if oerr != nil || len(open) != 1 {
		t.Fatalf("open duplicates after refusal: %v (%d)", oerr, len(open))
	}

	memID := mkObs(string(domain.InputFactKindMembership))
	if err := ws.DecideDuplicate(ctx, memID, false); err != nil {
		t.Fatalf("membership zero-line replay must still work: %v", err)
	}
	facts := mustFactsByPlatform(t, ws, source.ID)
	if len(facts) != 1 || facts[0].Kind != string(domain.InputFactKindMembership) {
		t.Fatalf("membership replay must create one membership fact, got %+v", facts)
	}
	lines, err := ws.Store.ListFactLines(ctx, facts[0].ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("membership replay must derive the single stub line, got %v (%d)", err, len(lines))
	}
}

func TestAttachIdentityTriggersWaveRecompute(t *testing.T) {
	f := newRevisionFixture(t)
	addr := &domain.RecipientAddress{CustomerProfileID: f.cust.ID, RecipientName: "Att Recipient", Phone: "13800000000", AddressLine1: "1 Att St", IsDefault: true}
	if err := f.ws.CreateAddress(f.ctx, addr); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	ident := &domain.PlatformIdentity{PlatformID: f.source.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "UID-ATT", NormalizedValue: NormalizeIdentity("UID-ATT")}
	if err := f.ws.Store.CreateIdentity(f.ctx, ident); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	in := IngestFactInput{
		Kind:             string(domain.InputFactKindMembership),
		StableExternalID: "ATT-MEM-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-ATT",
		MembershipLevel:  "captain",
	}
	f.ingest(t, in)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	lines, err := f.ws.Store.ListFactLines(f.ctx, facts[0].ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("membership lines: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{lines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	rule := &domain.EntitlementRule{WaveID: f.wave.ID, ProductID: f.product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 1, Active: true}
	if err := f.ws.UpsertRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}

	// Before attaching, the entitlement result blocks on identity_unattached.
	views, err := f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil || len(views) != 1 {
		t.Fatalf("ListResultViews: %v (%d)", err, len(views))
	}
	if views[0].WorkState != domain.WorkStateBlocked {
		t.Fatalf("pre-attach state = %s, want blocked", views[0].WorkState)
	}

	// AttachIdentity alone (no explicit recompute) must clear the block.
	if err := f.ws.AttachIdentity(f.ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	views, err = f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil || len(views) != 1 {
		t.Fatalf("ListResultViews after attach: %v (%d)", err, len(views))
	}
	if views[0].WorkState != domain.WorkStateReady {
		t.Fatalf("post-attach state = %s blocks = %v, want ready", views[0].WorkState, views[0].Blocks)
	}
	if views[0].Result.CustomerProfileID == nil || *views[0].Result.CustomerProfileID != f.cust.ID {
		t.Fatalf("recomputed result must resolve the customer, got %+v", views[0].Result.CustomerProfileID)
	}
}
