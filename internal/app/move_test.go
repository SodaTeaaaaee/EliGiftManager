package app

import (
	"context"
	"errors"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type moveFixture struct {
	ws      *Workspace
	ctx     context.Context
	source  *domain.Platform
	factory *domain.Platform
	cust    *domain.CustomerProfile
	product *domain.ProductItem
	waveA   *domain.Wave
	waveB   *domain.Wave
}

func newMoveFixture(t *testing.T) *moveFixture {
	t.Helper()
	f := &moveFixture{ws: newTestWorkspace(t), ctx: context.Background()}
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
	f.cust = &domain.CustomerProfile{DisplayName: "Move Cust"}
	if err := f.ws.CreateCustomer(f.ctx, f.cust); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: f.cust.ID, RecipientName: "Move Recipient", Phone: "13800000000", AddressLine1: "1 Move St", IsDefault: true}
	if err := f.ws.CreateAddress(f.ctx, addr); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	f.product = &domain.ProductItem{Name: "Move Product", FactoryPlatformID: f.factory.ID, FactorySKU: "MOVE-SKU-1"}
	if err := f.ws.CreateProduct(f.ctx, f.product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	alias := &domain.ProductAlias{ProductItemID: f.product.ID, PlatformID: f.source.ID, ExternalProductID: "MOVE-ALIAS"}
	if err := f.ws.CreateAlias(f.ctx, alias); err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}
	for i, name := range []string{"wave A", "wave B"} {
		w, err := f.ws.CreateWave(f.ctx, name, "")
		if err != nil {
			t.Fatalf("CreateWave %s: %v", name, err)
		}
		if i == 0 {
			f.waveA = w
		} else {
			f.waveB = w
		}
	}
	return f
}

func (f *moveFixture) ingestRetail(t *testing.T, stableID string, qty int) domain.InputFactLine {
	t.Helper()
	in := IngestFactInput{
		Kind:             string(domain.InputFactKindRetailOrder),
		StableExternalID: stableID,
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-" + stableID,
		Lines:            []IngestLine{{SourceLineNo: 1, ExternalSKU: "MOVE-ALIAS", Quantity: qty}},
	}
	f.ingest(t, in)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	var fact domain.InputFact
	for _, fc := range facts {
		if fc.StableExternalID == stableID {
			fact = fc
		}
	}
	lines, err := f.ws.Store.ListFactLines(f.ctx, fact.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("ListFactLines(%s): %v (%d)", stableID, err, len(lines))
	}
	return lines[0]
}

func (f *moveFixture) ingest(t *testing.T, in IngestFactInput) {
	t.Helper()
	doc := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, doc, []IngestFactInput{in}); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	} else if len(dups) != 0 {
		t.Fatalf("IngestDocument produced duplicates: %+v", dups)
	}
}

func TestMoveLinesWithoutResultsMovesDirectly(t *testing.T) {
	f := newMoveFixture(t)
	// Membership line assigned without rules: an instance exists, no results.
	in := IngestFactInput{
		Kind:             string(domain.InputFactKindMembership),
		StableExternalID: "MOVE-MEM-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-MOVE-MEM",
		MembershipLevel:  "captain",
	}
	f.ingest(t, in)
	facts := mustFactsByPlatform(t, f.ws, f.source.ID)
	lines, err := f.ws.Store.ListFactLines(f.ctx, facts[0].ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("membership lines: %v (%d)", err, len(lines))
	}
	if err := f.ws.AssignLines(f.ctx, f.waveA.ID, []uint{lines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}

	if err := f.ws.MoveLines(f.ctx, []uint{lines[0].ID}, f.waveB.ID); err != nil {
		t.Fatalf("MoveLines: %v", err)
	}
	line, err := f.ws.Store.GetFactLine(f.ctx, lines[0].ID)
	if err != nil {
		t.Fatalf("GetFactLine: %v", err)
	}
	if line.WaveID == nil || *line.WaveID != f.waveB.ID {
		t.Fatalf("line must sit in wave B, got %+v", line.WaveID)
	}
	if _, err := f.ws.Store.GetInstanceByLine(f.ctx, f.waveA.ID, line.ID); err != domain.ErrNotFound {
		t.Fatalf("instance must leave wave A, got %v", err)
	}
	if _, err := f.ws.Store.GetInstanceByLine(f.ctx, f.waveB.ID, line.ID); err != nil {
		t.Fatalf("instance must follow into wave B: %v", err)
	}

	// Moving an unassigned line into a wave works like a fresh assignment.
	other := f.ingestRetail(t, "MOVE-NORES-1", 2)
	if err := f.ws.MoveLines(f.ctx, []uint{other.ID}, f.waveB.ID); err != nil {
		t.Fatalf("MoveLines unassigned: %v", err)
	}
	results, err := f.ws.Store.ListResults(f.ctx, f.waveB.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	found := false
	for _, r := range results {
		if r.InputFactLineID != nil && *r.InputFactLineID == other.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("unassigned retail line must gain a result in the target wave")
	}
}

func TestMoveLinesRevokesAndRebuildsUnfrozenResults(t *testing.T) {
	f := newMoveFixture(t)
	// Retail line with an unfrozen result in wave A plus a membership
	// entitlement so both result kinds are exercised.
	retailLine := f.ingestRetail(t, "MOVE-RET-1", 3)
	ident, err := f.ws.Store.FindIdentity(f.ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-MOVE-RET-1"))
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if err := f.ws.AttachIdentity(f.ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.waveA.ID, []uint{retailLine.ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	if _, err := f.ws.CreateGrant(f.ctx, f.waveA.ID, f.cust.ID, f.product.ID, 1); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	before, err := f.ws.Store.ListResults(f.ctx, f.waveA.ID)
	if err != nil {
		t.Fatalf("ListResults A: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("expected retail + grant results in wave A, got %d", len(before))
	}

	if err := f.ws.MoveLines(f.ctx, []uint{retailLine.ID}, f.waveB.ID); err != nil {
		t.Fatalf("MoveLines: %v", err)
	}
	afterA, err := f.ws.Store.ListResults(f.ctx, f.waveA.ID)
	if err != nil {
		t.Fatalf("ListResults A after: %v", err)
	}
	if len(afterA) != 1 {
		t.Fatalf("wave A must keep only the grant result, got %d", len(afterA))
	}
	afterB, err := f.ws.Store.ListResults(f.ctx, f.waveB.ID)
	if err != nil {
		t.Fatalf("ListResults B after: %v", err)
	}
	if len(afterB) != 1 || afterB[0].Quantity != 3 || afterB[0].InputFactLineID == nil || *afterB[0].InputFactLineID != retailLine.ID {
		t.Fatalf("wave B must hold the rebuilt retail result, got %+v", afterB)
	}

	// Membership entitlement move: instance-backed results follow the
	// instance into the target wave.
	mem := IngestFactInput{
		Kind:             string(domain.InputFactKindMembership),
		StableExternalID: "MOVE-MEM-2",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "UID-MOVE-MEM2",
		MembershipLevel:  "captain",
	}
	f.ingest(t, mem)
	memFacts := mustFactsByPlatform(t, f.ws, f.source.ID)
	var memFact domain.InputFact
	for _, fc := range memFacts {
		if fc.StableExternalID == "MOVE-MEM-2" {
			memFact = fc
		}
	}
	memLines, err := f.ws.Store.ListFactLines(f.ctx, memFact.ID)
	if err != nil || len(memLines) != 1 {
		t.Fatalf("membership lines: %v", err)
	}
	memIdent, err := f.ws.Store.FindIdentity(f.ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-MOVE-MEM2"))
	if err != nil {
		t.Fatalf("FindIdentity mem: %v", err)
	}
	if err := f.ws.AttachIdentity(f.ctx, memIdent.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity mem: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.waveA.ID, []uint{memLines[0].ID}); err != nil {
		t.Fatalf("AssignLines mem: %v", err)
	}
	rule := &domain.EntitlementRule{WaveID: f.waveA.ID, ProductID: f.product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 2, Active: true}
	if err := f.ws.UpsertRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}
	ruleB := &domain.EntitlementRule{WaveID: f.waveB.ID, ProductID: f.product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 5, Active: true}
	if err := f.ws.UpsertRule(f.ctx, ruleB); err != nil {
		t.Fatalf("UpsertRule B: %v", err)
	}
	if err := f.ws.MoveLines(f.ctx, []uint{memLines[0].ID}, f.waveB.ID); err != nil {
		t.Fatalf("MoveLines membership: %v", err)
	}
	entA, err := f.ws.Store.ListResults(f.ctx, f.waveA.ID)
	if err != nil {
		t.Fatalf("ListResults A after mem move: %v", err)
	}
	for _, r := range entA {
		if r.SourceKind == string(domain.SourceEntitlementInstance) {
			t.Fatal("entitlement result must leave wave A with its instance")
		}
	}
	entB, err := f.ws.Store.ListResults(f.ctx, f.waveB.ID)
	if err != nil {
		t.Fatalf("ListResults B after mem move: %v", err)
	}
	foundEnt := false
	for _, r := range entB {
		// Rules are wave-scoped: the rebuilt entitlement follows wave B's rule
		// (quantity 5), not the wave A rule the instance used to sit under.
		if r.SourceKind == string(domain.SourceEntitlementInstance) && r.Quantity == 5 {
			foundEnt = true
		}
	}
	if !foundEnt {
		t.Fatalf("entitlement result must be rebuilt in wave B under its rules, got %+v", entB)
	}
}

func TestMoveLinesRefusesFrozenLines(t *testing.T) {
	f := newMoveFixture(t)
	line := f.ingestRetail(t, "MOVE-FROZEN-1", 2)
	ident, err := f.ws.Store.FindIdentity(f.ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-MOVE-FROZEN-1"))
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if err := f.ws.AttachIdentity(f.ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	if err := f.ws.AssignLines(f.ctx, f.waveA.ID, []uint{line.ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	if _, _, err := f.ws.GenerateFactoryOrder(f.ctx, f.waveA.ID, f.factory.ID); err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}

	if err := f.ws.MoveLines(f.ctx, []uint{line.ID}, f.waveB.ID); !errors.Is(err, ErrLineFrozen) {
		t.Fatalf("MoveLines on frozen line = %v, want ErrLineFrozen", err)
	}
	after, err := f.ws.Store.GetFactLine(f.ctx, line.ID)
	if err != nil {
		t.Fatalf("GetFactLine: %v", err)
	}
	if after.WaveID == nil || *after.WaveID != f.waveA.ID {
		t.Fatal("frozen line must stay in wave A after the refused move")
	}
	results, err := f.ws.Store.ListResults(f.ctx, f.waveA.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 || !results[0].Frozen {
		t.Fatalf("frozen result must be untouched, got %+v", results)
	}
	bResults, err := f.ws.Store.ListResults(f.ctx, f.waveB.ID)
	if err != nil {
		t.Fatalf("ListResults B: %v", err)
	}
	if len(bResults) != 0 {
		t.Fatalf("wave B must stay empty after the refused move, got %d", len(bResults))
	}

	// A closed target wave refuses any move.
	if err := f.ws.CloseWave(f.ctx, f.waveB.ID, string(domain.WaveCloseResultClean), ""); err != nil {
		t.Fatalf("CloseWave: %v", err)
	}
	other := f.ingestRetail(t, "MOVE-CLOSED-1", 1)
	if err := f.ws.MoveLines(f.ctx, []uint{other.ID}, f.waveB.ID); !errors.Is(err, ErrWaveClosed) {
		t.Fatalf("MoveLines into closed wave = %v, want ErrWaveClosed", err)
	}
}

func TestMoveLinesRefusesClosedSourceWave(t *testing.T) {
	f := newMoveFixture(t)
	// Wave B plays the open target; wave A gets closed as the source.
	line := f.ingestRetail(t, "MOVE-SRC-CLOSED-1", 2)
	if err := f.ws.AssignLines(f.ctx, f.waveA.ID, []uint{line.ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	if err := f.ws.CloseWave(f.ctx, f.waveA.ID, string(domain.WaveCloseResultClean), ""); err != nil {
		t.Fatalf("CloseWave: %v", err)
	}

	if err := f.ws.MoveLines(f.ctx, []uint{line.ID}, f.waveB.ID); !errors.Is(err, ErrWaveClosed) {
		t.Fatalf("MoveLines out of closed wave = %v, want ErrWaveClosed", err)
	}
	after, err := f.ws.Store.GetFactLine(f.ctx, line.ID)
	if err != nil {
		t.Fatalf("GetFactLine: %v", err)
	}
	if after.WaveID == nil || *after.WaveID != f.waveA.ID {
		t.Fatalf("line must stay in the closed source wave, got %+v", after.WaveID)
	}
	if results, err := f.ws.Store.ListResults(f.ctx, f.waveB.ID); err != nil || len(results) != 0 {
		t.Fatalf("target wave must stay empty after the refused move, got %v (%d)", err, len(results))
	}
}
