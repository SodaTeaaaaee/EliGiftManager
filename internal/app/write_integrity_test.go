package app

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// TestSaveUpdatesPreserveCreatedAt guards the Save-based Update methods: the
// FromDomain mappers must carry CreatedAt/UpdatedAt so a full-field Save does
// not zero the audit timestamps of existing rows.
func TestSaveUpdatesPreserveCreatedAt(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	store := infra.NewGormStore(gdb)
	past := time.Now().Add(-96 * time.Hour).Truncate(time.Second)

	must := func(err error, op string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
	}
	rewind := func(table string, id uint) {
		t.Helper()
		if err := gdb.Exec("UPDATE "+table+" SET created_at = ? WHERE id = ?", past, id).Error; err != nil {
			t.Fatalf("rewind %s: %v", table, err)
		}
	}

	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	must(store.CreatePlatform(ctx, factory), "create factory platform")
	wave := &domain.Wave{WaveNo: "W-000001", Name: "w", CloseResult: string(domain.WaveCloseResultOpen)}
	must(store.CreateWave(ctx, wave), "create wave")
	customer := &domain.CustomerProfile{DisplayName: "C"}
	must(store.CreateCustomer(ctx, customer), "create customer")
	product := &domain.ProductItem{Name: "P", FactoryPlatformID: factory.ID, FactorySKU: "SKU-P"}
	must(store.CreateProduct(ctx, product), "create product")

	t.Run("wave", func(t *testing.T) {
		stored, err := store.GetWave(ctx, wave.ID)
		must(err, "get wave")
		rewind("waves", wave.ID)
		stored, _ = store.GetWave(ctx, wave.ID)
		stored.Notes = "updated"
		must(store.UpdateWave(ctx, stored), "update wave")
		after, err := store.GetWave(ctx, wave.ID)
		must(err, "re-read wave")
		if after.Notes != "updated" {
			t.Fatalf("notes = %q, want updated", after.Notes)
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("wave created_at = %v, want %v", after.CreatedAt, past)
		}
	})

	t.Run("address", func(t *testing.T) {
		addr := &domain.RecipientAddress{CustomerProfileID: customer.ID, RecipientName: "A", AddressLine1: "1 St"}
		must(store.CreateAddress(ctx, addr), "create address")
		rewind("recipient_addresses", addr.ID)
		stored, err := store.GetAddress(ctx, addr.ID)
		must(err, "get address")
		stored.Label = "updated"
		must(store.UpdateAddress(ctx, stored), "update address")
		after, err := store.GetAddress(ctx, addr.ID)
		must(err, "re-read address")
		if after.Label != "updated" {
			t.Fatalf("label = %q, want updated", after.Label)
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("address created_at = %v, want %v", after.CreatedAt, past)
		}
	})

	t.Run("template", func(t *testing.T) {
		tmpl := &domain.TemplateConfig{PlatformID: factory.ID, DocumentType: "csv", Direction: "input", Name: "t", Version: 1}
		must(store.CreateTemplate(ctx, tmpl), "create template")
		rewind("template_configs", tmpl.ID)
		stored, err := store.GetTemplate(ctx, tmpl.ID)
		must(err, "get template")
		stored.Notes = "updated"
		must(store.UpdateTemplate(ctx, stored), "update template")
		after, err := store.GetTemplate(ctx, tmpl.ID)
		must(err, "re-read template")
		if after.Notes != "updated" {
			t.Fatalf("notes = %q, want updated", after.Notes)
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("template created_at = %v, want %v", after.CreatedAt, past)
		}
	})

	t.Run("fact and fact line", func(t *testing.T) {
		fact := &domain.InputFact{PlatformID: factory.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: "mem-a"}
		must(store.CreateFact(ctx, fact), "create fact")
		rewind("input_facts", fact.ID)
		storedFact, err := store.GetFact(ctx, fact.ID)
		must(err, "get fact")
		storedFact.MembershipLevel = "captain"
		must(store.UpdateFact(ctx, storedFact), "update fact")
		afterFact, err := store.GetFact(ctx, fact.ID)
		must(err, "re-read fact")
		if afterFact.MembershipLevel != "captain" {
			t.Fatalf("membership level = %q, want captain", afterFact.MembershipLevel)
		}
		if !afterFact.CreatedAt.Equal(past) {
			t.Fatalf("fact created_at = %v, want %v", afterFact.CreatedAt, past)
		}

		line := &domain.InputFactLine{FactID: fact.ID, SourceLineNo: 1, Quantity: 1}
		must(store.CreateFactLine(ctx, line), "create fact line")
		rewind("input_fact_lines", line.ID)
		storedLine, err := store.GetFactLine(ctx, line.ID)
		must(err, "get fact line")
		storedLine.Quantity = 3
		must(store.UpdateFactLine(ctx, storedLine), "update fact line")
		afterLine, err := store.GetFactLine(ctx, line.ID)
		must(err, "re-read fact line")
		if afterLine.Quantity != 3 {
			t.Fatalf("quantity = %d, want 3", afterLine.Quantity)
		}
		if !afterLine.CreatedAt.Equal(past) {
			t.Fatalf("fact line created_at = %v, want %v", afterLine.CreatedAt, past)
		}
	})

	t.Run("rule", func(t *testing.T) {
		rule := &domain.EntitlementRule{WaveID: wave.ID, ProductID: product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 1, Active: true}
		must(store.CreateRule(ctx, rule), "create rule")
		rewind("entitlement_rules", rule.ID)
		stored, err := store.GetRule(ctx, rule.ID)
		must(err, "get rule")
		stored.Quantity = 7
		must(store.UpdateRule(ctx, stored), "update rule")
		after, err := store.GetRule(ctx, rule.ID)
		must(err, "re-read rule")
		if after.Quantity != 7 {
			t.Fatalf("quantity = %d, want 7", after.Quantity)
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("rule created_at = %v, want %v", after.CreatedAt, past)
		}
	})

	t.Run("result", func(t *testing.T) {
		res := &domain.FulfillmentResult{WaveID: wave.ID, SourceKind: string(domain.SourceEntitlementInstance), Quantity: 1, Address: domain.AddressSnapshot{RecipientName: "A", AddressLine1: "1 St"}}
		must(store.CreateResult(ctx, res), "create result")
		rewind("fulfillment_results", res.ID)
		stored, err := store.GetResult(ctx, res.ID)
		must(err, "get result")
		stored.Quantity = 5
		must(store.UpdateResult(ctx, stored), "update result")
		after, err := store.GetResult(ctx, res.ID)
		must(err, "re-read result")
		if after.Quantity != 5 {
			t.Fatalf("quantity = %d, want 5", after.Quantity)
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("result created_at = %v, want %v", after.CreatedAt, past)
		}
	})

	t.Run("supplier order and line", func(t *testing.T) {
		order := &domain.SupplierOrder{WaveID: wave.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderGenerated)}
		must(store.CreateSupplierOrder(ctx, order), "create order")
		rewind("supplier_orders", order.ID)
		storedOrder, err := store.GetSupplierOrder(ctx, order.ID)
		must(err, "get order")
		storedOrder.ExtraData = "updated"
		must(store.UpdateSupplierOrder(ctx, storedOrder), "update order")
		afterOrder, err := store.GetSupplierOrder(ctx, order.ID)
		must(err, "re-read order")
		if afterOrder.ExtraData != "updated" {
			t.Fatalf("extra data = %q, want updated", afterOrder.ExtraData)
		}
		if !afterOrder.CreatedAt.Equal(past) {
			t.Fatalf("order created_at = %v, want %v", afterOrder.CreatedAt, past)
		}

		line := &domain.SupplierOrderLine{SupplierOrderID: order.ID, ProductItemID: product.ID, FactorySKU: product.FactorySKU, Quantity: 1, TrackingID: "trk-pin-1"}
		must(store.CreateSupplierOrderLine(ctx, line), "create order line")
		rewind("supplier_order_lines", line.ID)
		storedLine, err := store.GetSupplierOrderLine(ctx, line.ID)
		must(err, "get order line")
		storedLine.Quantity = 4
		must(store.UpdateSupplierOrderLine(ctx, storedLine), "update order line")
		afterLine, err := store.GetSupplierOrderLine(ctx, line.ID)
		must(err, "re-read order line")
		if afterLine.Quantity != 4 {
			t.Fatalf("quantity = %d, want 4", afterLine.Quantity)
		}
		if !afterLine.CreatedAt.Equal(past) {
			t.Fatalf("order line created_at = %v, want %v", afterLine.CreatedAt, past)
		}
	})

	t.Run("result via SetResultAddress", func(t *testing.T) {
		ws := NewWorkspace(store)
		res := &domain.FulfillmentResult{WaveID: wave.ID, SourceKind: string(domain.SourceEntitlementInstance), Quantity: 1}
		must(store.CreateResult(ctx, res), "create result")
		rewind("fulfillment_results", res.ID)
		addr := &domain.RecipientAddress{CustomerProfileID: customer.ID, RecipientName: "Pinned", AddressLine1: "2 St"}
		must(store.CreateAddress(ctx, addr), "create address")
		must(ws.SetResultAddress(ctx, res.ID, addr.ID), "set result address")
		after, err := store.GetResult(ctx, res.ID)
		must(err, "re-read result")
		if after.Address.RecipientName != "Pinned" {
			t.Fatalf("recipient name = %q, want Pinned", after.Address.RecipientName)
		}
		if !after.AddressPinned {
			t.Fatal("expected AddressPinned after SetResultAddress")
		}
		if !after.CreatedAt.Equal(past) {
			t.Fatalf("result created_at = %v, want %v", after.CreatedAt, past)
		}
	})
}

// TestGenerateFactoryOrderRollsBackAtomically fails the tracking ID source on
// the second product group and asserts the whole generation (order, lines,
// links, result freezing) rolls back.
func TestGenerateFactoryOrderRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	store := ws.Store

	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory: %v", err)
	}
	customer := &domain.CustomerProfile{DisplayName: "C"}
	if err := store.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: customer.ID, RecipientName: "Default", AddressLine1: "1 St", IsDefault: true}
	if err := store.CreateAddress(ctx, addr); err != nil {
		t.Fatalf("create address: %v", err)
	}
	p1 := &domain.ProductItem{Name: "P1", FactoryPlatformID: factory.ID, FactorySKU: "SKU-1"}
	p2 := &domain.ProductItem{Name: "P2", FactoryPlatformID: factory.ID, FactorySKU: "SKU-2"}
	if err := store.CreateProduct(ctx, p1); err != nil {
		t.Fatalf("create product 1: %v", err)
	}
	if err := store.CreateProduct(ctx, p2); err != nil {
		t.Fatalf("create product 2: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatalf("create wave: %v", err)
	}
	for i, p := range []*domain.ProductItem{p1, p2} {
		if _, err := ws.CreateGrant(ctx, wave.ID, customer.ID, p.ID, i+1); err != nil {
			t.Fatalf("create grant %d: %v", i+1, err)
		}
	}

	calls := 0
	ws.NewTrackingID = func() (string, error) {
		calls++
		if calls == 1 {
			return strings.Repeat("a", 32), nil
		}
		return "", fmt.Errorf("tracking id source exhausted")
	}

	_, _, err = ws.GenerateFactoryOrder(ctx, wave.ID, factory.ID)
	if err == nil {
		t.Fatal("expected GenerateFactoryOrder to fail on the second tracking id")
	}
	if !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("unexpected error: %v", err)
	}

	var orderCount, lineCount, linkCount int64
	if err := gdb.Table("supplier_orders").Where("wave_id = ?", wave.ID).Count(&orderCount).Error; err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if err := gdb.Table("supplier_order_lines").Count(&lineCount).Error; err != nil {
		t.Fatalf("count order lines: %v", err)
	}
	if err := gdb.Table("execution_quantity_links").Count(&linkCount).Error; err != nil {
		t.Fatalf("count links: %v", err)
	}
	if orderCount != 0 || lineCount != 0 || linkCount != 0 {
		t.Fatalf("expected atomic rollback, got orders=%d lines=%d links=%d", orderCount, lineCount, linkCount)
	}

	results, err := store.ListResults(ctx, wave.ID)
	if err != nil {
		t.Fatalf("list results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 grant results after rollback, got %d", len(results))
	}
	for _, r := range results {
		if r.Frozen {
			t.Fatalf("result %d must stay unfrozen after rollback", r.ID)
		}
		links, err := store.ListLinksByResult(ctx, r.ID)
		if err != nil {
			t.Fatalf("list links: %v", err)
		}
		if len(links) != 0 {
			t.Fatalf("result %d must have no links after rollback, got %d", r.ID, len(links))
		}
	}
}

// TestRecomputeKeepsPinnedAddress verifies that a manually set (pinned)
// address snapshot survives entitlement recompute while unpinned results fall
// back to the (possibly changed) customer default address.
func TestRecomputeKeepsPinnedAddress(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	store := ws.Store

	source := &domain.Platform{Key: "bilibili", Name: "B", Kind: string(domain.PlatformKindSource)}
	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, source); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory: %v", err)
	}
	customer := &domain.CustomerProfile{DisplayName: "C"}
	if err := store.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	addrDefault := &domain.RecipientAddress{CustomerProfileID: customer.ID, Label: "home", RecipientName: "Default Name", AddressLine1: "1 Default St", Phone: "138", IsDefault: true}
	addrOther := &domain.RecipientAddress{CustomerProfileID: customer.ID, Label: "office", RecipientName: "Other Name", AddressLine1: "9 Other Ave", Phone: "139"}
	if err := store.CreateAddress(ctx, addrDefault); err != nil {
		t.Fatalf("create default address: %v", err)
	}
	if err := store.CreateAddress(ctx, addrOther); err != nil {
		t.Fatalf("create other address: %v", err)
	}
	product := &domain.ProductItem{Name: "P", FactoryPlatformID: factory.ID, FactorySKU: "SKU-P"}
	if err := store.CreateProduct(ctx, product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatalf("create wave: %v", err)
	}

	newMemberLine := func(tag string) uint {
		t.Helper()
		fact := &domain.InputFact{PlatformID: source.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: tag, CustomerProfileID: &customer.ID}
		if err := store.CreateFact(ctx, fact); err != nil {
			t.Fatalf("create fact %s: %v", tag, err)
		}
		line := &domain.InputFactLine{FactID: fact.ID, SourceLineNo: 1, Quantity: 1}
		if err := store.CreateFactLine(ctx, line); err != nil {
			t.Fatalf("create line %s: %v", tag, err)
		}
		return line.ID
	}
	lineA, lineB := newMemberLine("mem-a"), newMemberLine("mem-b")
	if err := ws.AssignLines(ctx, wave.ID, []uint{lineA, lineB}); err != nil {
		t.Fatalf("assign lines: %v", err)
	}

	rule := &domain.EntitlementRule{WaveID: wave.ID, ProductID: product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 1, Active: true}
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("upsert rule: %v", err)
	}

	entitlementByInstance := func() map[uint]domain.FulfillmentResult {
		t.Helper()
		instA, err := store.GetInstanceByLine(ctx, wave.ID, lineA)
		if err != nil {
			t.Fatalf("get instance A: %v", err)
		}
		instB, err := store.GetInstanceByLine(ctx, wave.ID, lineB)
		if err != nil {
			t.Fatalf("get instance B: %v", err)
		}
		results, err := store.ListResults(ctx, wave.ID)
		if err != nil {
			t.Fatalf("list results: %v", err)
		}
		byInstance := map[uint]domain.FulfillmentResult{}
		for _, r := range results {
			if r.SourceKind == string(domain.SourceEntitlementInstance) && r.EntitlementInstanceID != nil {
				byInstance[*r.EntitlementInstanceID] = r
			}
		}
		if _, ok := byInstance[instA.ID]; !ok {
			t.Fatalf("missing entitlement result for instance A (%d)", instA.ID)
		}
		if _, ok := byInstance[instB.ID]; !ok {
			t.Fatalf("missing entitlement result for instance B (%d)", instB.ID)
		}
		return byInstance
	}

	before := entitlementByInstance()
	instA, err := store.GetInstanceByLine(ctx, wave.ID, lineA)
	if err != nil {
		t.Fatalf("get instance A: %v", err)
	}
	instB, err := store.GetInstanceByLine(ctx, wave.ID, lineB)
	if err != nil {
		t.Fatalf("get instance B: %v", err)
	}
	if got := before[instA.ID].Address.RecipientName; got != "Default Name" {
		t.Fatalf("instance A initial address = %q, want default snapshot", got)
	}

	if err := ws.SetResultAddress(ctx, before[instA.ID].ID, addrOther.ID); err != nil {
		t.Fatalf("set result address: %v", err)
	}

	// Change the default address, then recompute via a rule quantity change.
	addrDefault.RecipientName = "Default V2"
	if err := store.UpdateAddress(ctx, addrDefault); err != nil {
		t.Fatalf("update default address: %v", err)
	}
	rule.Quantity = 2
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("upsert rule after pin: %v", err)
	}

	after := entitlementByInstance()
	resA, resB := after[instA.ID], after[instB.ID]
	if resA.Address.RecipientName != "Other Name" {
		t.Fatalf("pinned result address = %q, want Other Name", resA.Address.RecipientName)
	}
	if !resA.AddressPinned {
		t.Fatal("pinned result must stay AddressPinned across recompute")
	}
	if resA.Quantity != 2 {
		t.Fatalf("pinned result quantity = %d, want 2", resA.Quantity)
	}
	if resB.Address.RecipientName != "Default V2" {
		t.Fatalf("unpinned result address = %q, want refreshed default Default V2", resB.Address.RecipientName)
	}
	if resB.AddressPinned {
		t.Fatal("unpinned result must not become pinned")
	}
	if resB.Quantity != 2 {
		t.Fatalf("unpinned result quantity = %d, want 2", resB.Quantity)
	}
}

// TestNextWaveNoFromMaxSuffix ensures wave numbers come from the max suffix,
// not the row count, so deleting a wave never re-issues its number.
func TestNextWaveNoFromMaxSuffix(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	store := ws.Store

	if no, err := store.NextWaveNo(ctx); err != nil {
		t.Fatalf("NextWaveNo on empty table: %v", err)
	} else if no != "W-000001" {
		t.Fatalf("NextWaveNo on empty table = %q, want W-000001", no)
	}

	for i := 1; i <= 3; i++ {
		w, err := ws.CreateWave(ctx, fmt.Sprintf("w%d", i), "")
		if err != nil {
			t.Fatalf("create wave %d: %v", i, err)
		}
		if want := fmt.Sprintf("W-%06d", i); w.WaveNo != want {
			t.Fatalf("wave %d number = %q, want %q", i, w.WaveNo, want)
		}
	}

	if err := gdb.Exec("DELETE FROM waves WHERE wave_no = 'W-000002'").Error; err != nil {
		t.Fatalf("delete middle wave: %v", err)
	}
	if no, err := store.NextWaveNo(ctx); err != nil {
		t.Fatalf("NextWaveNo after delete: %v", err)
	} else if no != "W-000004" {
		t.Fatalf("NextWaveNo after delete = %q, want W-000004", no)
	}

	w, err := ws.CreateWave(ctx, "w4", "")
	if err != nil {
		t.Fatalf("create wave after delete: %v", err)
	}
	if w.WaveNo != "W-000004" {
		t.Fatalf("created wave number = %q, want W-000004", w.WaveNo)
	}
}

// failingUpdateStore wraps a store and fails the Nth UpdateSupplierOrderLine
// call. It also wraps WithTx so the injection reaches inside transactions.
type failingUpdateStore struct {
	domain.Store
	calls  *int
	failOn int
}

func (s *failingUpdateStore) UpdateSupplierOrderLine(ctx context.Context, l *domain.SupplierOrderLine) error {
	*s.calls++
	if *s.calls == s.failOn {
		return fmt.Errorf("injected update failure %d", *s.calls)
	}
	return s.Store.UpdateSupplierOrderLine(ctx, l)
}

func (s *failingUpdateStore) WithTx(ctx context.Context, fn func(domain.Store) error) error {
	return s.Store.WithTx(ctx, func(tx domain.Store) error {
		return fn(&failingUpdateStore{Store: tx, calls: s.calls, failOn: s.failOn})
	})
}

// TestVoidFactoryOrderRollsBackAtomically fails the second order-line update
// mid-void and asserts the whole void rolls back: results stay frozen, tracking
// IDs stay unretired, links stay in place, and the order keeps its status.
func TestVoidFactoryOrderRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	base := infra.NewGormStore(gdb)
	ws := NewWorkspace(base)

	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := base.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory: %v", err)
	}
	customer := &domain.CustomerProfile{DisplayName: "C"}
	if err := base.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: customer.ID, RecipientName: "Default", Phone: "138", AddressLine1: "1 St", IsDefault: true}
	if err := base.CreateAddress(ctx, addr); err != nil {
		t.Fatalf("create address: %v", err)
	}
	p1 := &domain.ProductItem{Name: "P1", FactoryPlatformID: factory.ID, FactorySKU: "SKU-1"}
	p2 := &domain.ProductItem{Name: "P2", FactoryPlatformID: factory.ID, FactorySKU: "SKU-2"}
	if err := base.CreateProduct(ctx, p1); err != nil {
		t.Fatalf("create product 1: %v", err)
	}
	if err := base.CreateProduct(ctx, p2); err != nil {
		t.Fatalf("create product 2: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatalf("create wave: %v", err)
	}
	for i, p := range []*domain.ProductItem{p1, p2} {
		if _, err := ws.CreateGrant(ctx, wave.ID, customer.ID, p.ID, i+1); err != nil {
			t.Fatalf("create grant %d: %v", i+1, err)
		}
	}
	order, lines, err := ws.GenerateFactoryOrder(ctx, wave.ID, factory.ID)
	if err != nil {
		t.Fatalf("generate factory order: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 order lines, got %d", len(lines))
	}

	calls := 0
	failing := &failingUpdateStore{Store: base, calls: &calls, failOn: 2}
	voidWs := NewWorkspace(failing)
	if err := voidWs.VoidFactoryOrder(ctx, order.ID); err == nil || !strings.Contains(err.Error(), "injected update failure") {
		t.Fatalf("expected injected failure on second line update, got %v", err)
	}

	after, err := base.GetSupplierOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order after failed void: %v", err)
	}
	if after.Status != string(domain.SupplierOrderGenerated) {
		t.Fatalf("order status = %q, want %q", after.Status, domain.SupplierOrderGenerated)
	}
	if after.VoidedAt != nil {
		t.Fatalf("order voided_at = %v, want nil after rollback", after.VoidedAt)
	}

	results, err := base.ListResults(ctx, wave.ID)
	if err != nil {
		t.Fatalf("list results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Frozen {
			t.Fatalf("result %d must stay frozen after void rollback", r.ID)
		}
		links, err := base.ListLinksByResult(ctx, r.ID)
		if err != nil {
			t.Fatalf("list links for result %d: %v", r.ID, err)
		}
		if len(links) != 1 {
			t.Fatalf("result %d must keep its link after void rollback, got %d", r.ID, len(links))
		}
	}

	var retired int64
	if err := gdb.Table("retired_tracking_ids").Count(&retired).Error; err != nil {
		t.Fatalf("count retired tracking ids: %v", err)
	}
	if retired != 0 {
		t.Fatalf("expected 0 retired tracking ids after rollback, got %d", retired)
	}

	for _, ln := range lines {
		stored, err := base.GetSupplierOrderLine(ctx, ln.ID)
		if err != nil {
			t.Fatalf("get order line %d: %v", ln.ID, err)
		}
		if stored.TrackingRetired {
			t.Fatalf("order line %d must not be marked retired after rollback", ln.ID)
		}
	}
}

// TestRecomputeKeepsFrozenResult covers the pinned-and-frozen coexistence: a
// result pinned before factory order generation is frozen with the pinned
// address, address changes are rejected once frozen, and recompute leaves the
// frozen row untouched instead of rebuilding it.
func TestRecomputeKeepsFrozenResult(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	store := ws.Store

	source := &domain.Platform{Key: "bilibili", Name: "B", Kind: string(domain.PlatformKindSource)}
	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, source); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory: %v", err)
	}
	customer := &domain.CustomerProfile{DisplayName: "C"}
	if err := store.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	addrDefault := &domain.RecipientAddress{CustomerProfileID: customer.ID, Label: "home", RecipientName: "Default Name", Phone: "138", AddressLine1: "1 Default St", IsDefault: true}
	addrOther := &domain.RecipientAddress{CustomerProfileID: customer.ID, Label: "office", RecipientName: "Other Name", Phone: "139", AddressLine1: "9 Other Ave"}
	if err := store.CreateAddress(ctx, addrDefault); err != nil {
		t.Fatalf("create default address: %v", err)
	}
	if err := store.CreateAddress(ctx, addrOther); err != nil {
		t.Fatalf("create other address: %v", err)
	}
	product := &domain.ProductItem{Name: "P", FactoryPlatformID: factory.ID, FactorySKU: "SKU-P"}
	if err := store.CreateProduct(ctx, product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatalf("create wave: %v", err)
	}

	ident := &domain.PlatformIdentity{PlatformID: source.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "uid-1", NormalizedValue: NormalizeIdentity("uid-1"), CustomerProfileID: &customer.ID}
	if err := store.CreateIdentity(ctx, ident); err != nil {
		t.Fatalf("create identity: %v", err)
	}
	fact := &domain.InputFact{PlatformID: source.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: "mem-fz", CustomerProfileID: &customer.ID, PlatformIdentityID: &ident.ID, MembershipLevel: "captain"}
	if err := store.CreateFact(ctx, fact); err != nil {
		t.Fatalf("create fact: %v", err)
	}
	line := &domain.InputFactLine{FactID: fact.ID, SourceLineNo: 1, Quantity: 1}
	if err := store.CreateFactLine(ctx, line); err != nil {
		t.Fatalf("create fact line: %v", err)
	}
	if err := ws.AssignLines(ctx, wave.ID, []uint{line.ID}); err != nil {
		t.Fatalf("assign lines: %v", err)
	}
	inst, err := store.GetInstanceByLine(ctx, wave.ID, line.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}

	rule := &domain.EntitlementRule{WaveID: wave.ID, ProductID: product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 1, Active: true}
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("upsert rule: %v", err)
	}

	entitlementFor := func() domain.FulfillmentResult {
		t.Helper()
		results, err := store.ListResults(ctx, wave.ID)
		if err != nil {
			t.Fatalf("list results: %v", err)
		}
		var found *domain.FulfillmentResult
		count := 0
		for i := range results {
			if results[i].SourceKind == string(domain.SourceEntitlementInstance) && results[i].EntitlementInstanceID != nil && *results[i].EntitlementInstanceID == inst.ID {
				found = &results[i]
				count++
			}
		}
		if found == nil || count != 1 {
			t.Fatalf("expected exactly 1 entitlement result for instance %d, got %d", inst.ID, count)
		}
		return *found
	}

	before := entitlementFor()
	if got := before.Address.RecipientName; got != "Default Name" {
		t.Fatalf("initial address = %q, want default snapshot", got)
	}

	// Pin the address first, then freeze via factory order generation.
	if err := ws.SetResultAddress(ctx, before.ID, addrOther.ID); err != nil {
		t.Fatalf("set result address before generate: %v", err)
	}
	if _, _, err := ws.GenerateFactoryOrder(ctx, wave.ID, factory.ID); err != nil {
		t.Fatalf("generate factory order: %v", err)
	}
	frozen := entitlementFor()
	if !frozen.Frozen || !frozen.AddressPinned {
		t.Fatal("result must be both frozen and pinned after generation")
	}
	if got := frozen.Address.RecipientName; got != "Other Name" {
		t.Fatalf("frozen address = %q, want pinned Other Name", got)
	}
	if frozen.ID != before.ID {
		t.Fatalf("freeze must keep the same result row, got %d want %d", frozen.ID, before.ID)
	}

	// Address changes are rejected once frozen.
	if err := ws.SetResultAddress(ctx, frozen.ID, addrDefault.ID); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("expected frozen rejection, got %v", err)
	}

	// Recompute via a rule quantity change: the frozen row survives untouched.
	rule.Quantity = 4
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("upsert rule after freeze: %v", err)
	}
	after := entitlementFor()
	if after.ID != frozen.ID {
		t.Fatalf("recompute must not rebuild the frozen result, got id %d want %d", after.ID, frozen.ID)
	}
	if !after.Frozen || !after.AddressPinned {
		t.Fatal("recomputed result must stay frozen and pinned")
	}
	if after.Quantity != frozen.Quantity {
		t.Fatalf("frozen result quantity = %d, want unchanged %d", after.Quantity, frozen.Quantity)
	}
	if got := after.Address.RecipientName; got != "Other Name" {
		t.Fatalf("frozen result address = %q, want unchanged Other Name", got)
	}
}

// TestUpsertRuleEditPreservesTimestampsAndUpdateRefreshesUpdatedAt guards the
// rule edit path: a caller-constructed rule without audit timestamps must not
// zero created_at (backfill), and the Save-based update must still refresh
// updated_at beyond the rewound original.
func TestUpsertRuleEditPreservesTimestampsAndUpdateRefreshesUpdatedAt(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	ws := NewWorkspace(infra.NewGormStore(gdb))
	store := ws.Store
	past := time.Now().Add(-96 * time.Hour).Truncate(time.Second)

	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory: %v", err)
	}
	product := &domain.ProductItem{Name: "P", FactoryPlatformID: factory.ID, FactorySKU: "SKU-P"}
	if err := store.CreateProduct(ctx, product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "w", "")
	if err != nil {
		t.Fatalf("create wave: %v", err)
	}

	rule := &domain.EntitlementRule{WaveID: wave.ID, ProductID: product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 1, Active: true}
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("upsert rule: %v", err)
	}
	if err := gdb.Exec("UPDATE entitlement_rules SET created_at = ?, updated_at = ? WHERE id = ?", past, past, rule.ID).Error; err != nil {
		t.Fatalf("rewind rule timestamps: %v", err)
	}

	// Edit as a UI would: a fresh rule struct with only the edited fields.
	edited := &domain.EntitlementRule{ID: rule.ID, WaveID: wave.ID, ProductID: product.ID, Selector: domain.EntitlementSelector{Type: string(domain.SelectorWaveAll)}, Quantity: 3, Active: true}
	if err := ws.UpsertRule(ctx, edited); err != nil {
		t.Fatalf("upsert edited rule: %v", err)
	}

	after, err := store.GetRule(ctx, rule.ID)
	if err != nil {
		t.Fatalf("get rule: %v", err)
	}
	if after.Quantity != 3 {
		t.Fatalf("quantity = %d, want 3", after.Quantity)
	}
	if !after.CreatedAt.Equal(past) {
		t.Fatalf("rule created_at = %v, want backfilled %v", after.CreatedAt, past)
	}
	if !after.UpdatedAt.After(past) {
		t.Fatalf("rule updated_at = %v, want refreshed after %v", after.UpdatedAt, past)
	}
}
