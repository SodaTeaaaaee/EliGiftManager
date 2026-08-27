package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type splitFixture struct {
	ws      *Workspace
	ctx     context.Context
	source  *domain.Platform
	factory *domain.Platform
	cust    *domain.CustomerProfile
	prodA   *domain.ProductItem
	prodB   *domain.ProductItem
	wave    *domain.Wave
}

func newSplitFixture(t *testing.T) *splitFixture {
	t.Helper()
	f := &splitFixture{ws: newTestWorkspace(t), ctx: context.Background()}
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
	f.cust = &domain.CustomerProfile{DisplayName: "Split Cust"}
	if err := f.ws.CreateCustomer(f.ctx, f.cust); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: f.cust.ID, RecipientName: "Split Recipient", Phone: "13800000000", AddressLine1: "1 Split St", IsDefault: true}
	if err := f.ws.CreateAddress(f.ctx, addr); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	f.prodA = &domain.ProductItem{Name: "Split A", FactoryPlatformID: f.factory.ID, FactorySKU: "SPLIT-A"}
	f.prodB = &domain.ProductItem{Name: "Split B", FactoryPlatformID: f.factory.ID, FactorySKU: "SPLIT-B"}
	for _, p := range []*domain.ProductItem{f.prodA, f.prodB} {
		if err := f.ws.CreateProduct(f.ctx, p); err != nil {
			t.Fatalf("CreateProduct: %v", err)
		}
	}
	wave, err := f.ws.CreateWave(f.ctx, "split wave", "")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	f.wave = wave
	return f
}

// ingestSplitLine ingests and assigns a retail line of the given quantity on
// the given SKU and returns its line id.
func (f *splitFixture) ingestSplitLine(t *testing.T, stableID, sku string, qty int) uint {
	t.Helper()
	in := IngestFactInput{
		Kind:            string(domain.InputFactKindRetailOrder),
		StableExternalID: stableID,
		IdentityType:    string(domain.IdentityTypePlatformUID),
		IdentityValue:   "UID-" + stableID,
		Lines:           []IngestLine{{SourceLineNo: 1, ExternalSKU: sku, Quantity: qty}},
	}
	doc := &domain.InputDocument{PlatformID: f.source.ID, DocumentType: "retail"}
	if _, dups, err := f.ws.IngestDocument(f.ctx, doc, []IngestFactInput{in}); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	} else if len(dups) != 0 {
		t.Fatalf("IngestDocument produced duplicates: %+v", dups)
	}
	ident, err := f.ws.Store.FindIdentity(f.ctx, f.source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-"+stableID))
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if err := f.ws.AttachIdentity(f.ctx, ident.ID, f.cust.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
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
	if err := f.ws.AssignLines(f.ctx, f.wave.ID, []uint{lines[0].ID}); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}
	return lines[0].ID
}

func TestQuantitySplitExpandsWhenComponentsSum(t *testing.T) {
	f := newSplitFixture(t)
	lineID := f.ingestSplitLine(t, "SPLIT-ORD-1", "SPLIT-SKU", 5)

	// Before the rule: one plain (unaligned) result.
	views, err := f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil || len(views) != 1 {
		t.Fatalf("plain results: %v (%d)", err, len(views))
	}

	rule := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "SPLIT-SKU",
		Components: []domain.QuantitySplitComponent{
			{ProductItemID: f.prodA.ID, Quantity: 2},
			{ProductItemID: f.prodB.ID, Quantity: 3},
		},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertQuantitySplitRule: %v", err)
	}
	if rule.ID == 0 || len(rule.Components) != 2 {
		t.Fatalf("upsert must persist the rule with components, got %+v", rule)
	}

	views, err = f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected 2 split results, got %d", len(views))
	}
	qtyByProduct := map[uint]int{}
	for _, v := range views {
		if v.WorkState != domain.WorkStateReady {
			t.Fatalf("split result %d state = %s blocks = %v", v.Result.ID, v.WorkState, v.Blocks)
		}
		qtyByProduct[*v.Result.ProductItemID] = v.Result.Quantity
	}
	if qtyByProduct[f.prodA.ID] != 2 || qtyByProduct[f.prodB.ID] != 3 {
		t.Fatalf("split quantities = %v, want A:2 B:3", qtyByProduct)
	}

	listed, err := f.ws.ListQuantitySplitRules(f.ctx, f.wave.ID)
	if err != nil || len(listed) != 1 || len(listed[0].Components) != 2 {
		t.Fatalf("ListQuantitySplitRules: %v %+v", err, listed)
	}

	// Updating the rule replaces components wholesale and re-derives results.
	rule.Components = []domain.QuantitySplitComponent{
		{ProductItemID: f.prodA.ID, Quantity: 5},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertQuantitySplitRule update: %v", err)
	}
	views, err = f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after update: %v", err)
	}
	if len(views) != 1 || *views[0].Result.ProductItemID != f.prodA.ID || views[0].Result.Quantity != 5 {
		t.Fatalf("updated split results = %+v", views)
	}

	// Deleting the rule falls back to plain alignment.
	if err := f.ws.DeleteQuantitySplitRule(f.ctx, rule.ID); err != nil {
		t.Fatalf("DeleteQuantitySplitRule: %v", err)
	}
	views, err = f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after delete: %v", err)
	}
	if len(views) != 1 || views[0].Result.ProductItemID != nil {
		t.Fatalf("delete must fall back to the plain unaligned result, got %+v", views)
	}
	_ = lineID
}

func TestQuantitySplitNotSummingBlocks(t *testing.T) {
	f := newSplitFixture(t)
	f.ingestSplitLine(t, "SPLIT-ORD-2", "SPLIT-SKU", 5)

	rule := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "SPLIT-SKU",
		Components: []domain.QuantitySplitComponent{
			{ProductItemID: f.prodA.ID, Quantity: 2},
			{ProductItemID: f.prodB.ID, Quantity: 2},
		},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertQuantitySplitRule: %v", err)
	}

	views, err := f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("not-summing must keep exactly one placeholder result, got %d", len(views))
	}
	v := views[0]
	if v.WorkState != domain.WorkStateBlocked {
		t.Fatalf("placeholder state = %s, want blocked", v.WorkState)
	}
	if len(v.Blocks) != 1 || v.Blocks[0] != domain.BlockQuantitySplitNotSumming {
		t.Fatalf("blocks = %v, want [quantity_split_not_summing]", v.Blocks)
	}
	if v.Result.Quantity != 5 {
		t.Fatalf("placeholder must carry the line quantity 5, got %d", v.Result.Quantity)
	}

	// Blocked results must not enter a factory order.
	if _, _, err := f.ws.GenerateFactoryOrder(f.ctx, f.wave.ID, f.factory.ID); err != ErrNothingToSubmit {
		t.Fatalf("GenerateFactoryOrder on blocked wave = %v, want ErrNothingToSubmit", err)
	}

	// Fixing the components to sum unblocks without touching the line.
	rule.Components = []domain.QuantitySplitComponent{
		{ProductItemID: f.prodA.ID, Quantity: 1},
		{ProductItemID: f.prodB.ID, Quantity: 4},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertQuantitySplitRule fix: %v", err)
	}
	views, err = f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after fix: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("expected 2 split results after the fix, got %d", len(views))
	}
	for _, v := range views {
		if v.WorkState != domain.WorkStateReady {
			t.Fatalf("result %d must be ready after the fix, got %s blocks %v", v.Result.ID, v.WorkState, v.Blocks)
		}
	}
}

func TestQuantitySplitScopedToWaveAndKey(t *testing.T) {
	f := newSplitFixture(t)
	f.ingestSplitLine(t, "SPLIT-ORD-3", "SPLIT-SKU", 5)
	otherWave, err := f.ws.CreateWave(f.ctx, "split wave 2", "")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	rule := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "SPLIT-SKU",
		Components: []domain.QuantitySplitComponent{{ProductItemID: f.prodA.ID, Quantity: 5}},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, rule); err != nil {
		t.Fatalf("UpsertQuantitySplitRule: %v", err)
	}
	// A different key is not covered.
	ruleOther := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "OTHER-SKU",
		Components: []domain.QuantitySplitComponent{{ProductItemID: f.prodB.ID, Quantity: 9}},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, ruleOther); err != nil {
		t.Fatalf("UpsertQuantitySplitRule other: %v", err)
	}
	views, err := f.ws.ListResultViews(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	if len(views) != 1 || views[0].Result.ProductItemID == nil || *views[0].Result.ProductItemID != f.prodA.ID {
		t.Fatalf("only the matching key may expand, got %+v", views)
	}

	// Upserting the same key twice in the same wave replaces, not duplicates.
	dup := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "SPLIT-SKU",
		Components: []domain.QuantitySplitComponent{{ProductItemID: f.prodB.ID, Quantity: 5}},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, dup); err != nil {
		t.Fatalf("UpsertQuantitySplitRule dup: %v", err)
	}
	rules, err := f.ws.ListQuantitySplitRules(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListQuantitySplitRules: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules (SPLIT-SKU replaced + OTHER-SKU), got %d", len(rules))
	}
	if dup.ID != rule.ID {
		t.Fatalf("same (wave, platform, key) must reuse the rule row: got %d want %d", dup.ID, rule.ID)
	}

	// A closed wave refuses rule changes.
	if err := f.ws.CloseWave(f.ctx, f.wave.ID, string(domain.WaveCloseResultClean), ""); err != nil {
		t.Fatalf("CloseWave: %v", err)
	}
	closed := &domain.QuantitySplitRule{
		WaveID: f.wave.ID, PlatformID: f.source.ID, ExternalKey: "CLOSED-SKU",
		Components: []domain.QuantitySplitComponent{{ProductItemID: f.prodA.ID, Quantity: 1}},
	}
	if err := f.ws.UpsertQuantitySplitRule(f.ctx, closed); err == nil {
		t.Fatal("expected refusal on closed wave")
	}
	_ = otherWave
}
