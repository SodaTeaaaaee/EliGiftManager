package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type factoryOrderFixture struct {
	ws      *Workspace
	ctx     context.Context
	factory *domain.Platform
	cust    *domain.CustomerProfile
	wave    *domain.Wave
}

func newFactoryOrderFixture(t *testing.T) *factoryOrderFixture {
	t.Helper()
	f := &factoryOrderFixture{ws: newTestWorkspace(t), ctx: context.Background()}
	if err := f.ws.EnsureBuiltinPlatforms(f.ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	for _, p := range mustListPlatforms(t, f.ws) {
		pc := p
		if p.Kind == string(domain.PlatformKindFactory) {
			f.factory = &pc
		}
	}
	f.cust = &domain.CustomerProfile{DisplayName: "Factory Order Cust"}
	if err := f.ws.CreateCustomer(f.ctx, f.cust); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	addr := &domain.RecipientAddress{CustomerProfileID: f.cust.ID, RecipientName: "Factory Recipient", Phone: "13800000001", AddressLine1: "1 Factory St", IsDefault: true}
	if err := f.ws.CreateAddress(f.ctx, addr); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	wave, err := f.ws.CreateWave(f.ctx, "factory order wave", "")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	f.wave = wave
	return f
}

func (f *factoryOrderFixture) createProduct(t *testing.T, name, sku string) *domain.ProductItem {
	t.Helper()
	p := &domain.ProductItem{Name: name, FactoryPlatformID: f.factory.ID, FactorySKU: sku}
	if err := f.ws.CreateProduct(f.ctx, p); err != nil {
		t.Fatalf("CreateProduct %s: %v", name, err)
	}
	return p
}

func TestGenerateFactoryOrderForResultsFreezesOnlySelection(t *testing.T) {
	f := newFactoryOrderFixture(t)
	product := f.createProduct(t, "Factory Product", "FACT-SKU-1")
	first, err := f.ws.CreateGrant(f.ctx, f.wave.ID, f.cust.ID, product.ID, 2)
	if err != nil {
		t.Fatalf("CreateGrant first: %v", err)
	}
	second, err := f.ws.CreateGrant(f.ctx, f.wave.ID, f.cust.ID, product.ID, 3)
	if err != nil {
		t.Fatalf("CreateGrant second: %v", err)
	}

	order, lines, err := f.ws.GenerateFactoryOrderForResults(f.ctx, f.wave.ID, f.factory.ID, []uint{first.ID})
	if err != nil {
		t.Fatalf("GenerateFactoryOrderForResults: %v", err)
	}
	if order.Status != string(domain.SupplierOrderGenerated) {
		t.Fatalf("order status = %s, want generated", order.Status)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 order line for the selected subset, got %d", len(lines))
	}
	if lines[0].Quantity != 2 {
		t.Fatalf("subset line quantity = %d, want 2 (only the selected result)", lines[0].Quantity)
	}

	results, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	for _, r := range results {
		wantFrozen := r.ID == first.ID
		if r.Frozen != wantFrozen {
			t.Fatalf("result %d frozen = %v, want %v (unselected results must stay untouched)", r.ID, r.Frozen, wantFrozen)
		}
	}

	// The one-open-order slot per (wave, factory) still applies after a
	// partial submission: the leftover result cannot enter a second order
	// until the first is exported or voided.
	if _, _, err := f.ws.GenerateFactoryOrderForResults(f.ctx, f.wave.ID, f.factory.ID, []uint{second.ID}); !errors.Is(err, ErrOrderAlreadyOpen) {
		t.Fatalf("second submission = %v, want ErrOrderAlreadyOpen", err)
	}
}

func TestGenerateFactoryOrderForResultsEmptySelectionSubmitsNothing(t *testing.T) {
	f := newFactoryOrderFixture(t)
	product := f.createProduct(t, "Factory Product", "FACT-SKU-1")
	if _, err := f.ws.CreateGrant(f.ctx, f.wave.ID, f.cust.ID, product.ID, 2); err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}

	// An explicitly empty (non-nil) selection is a submission of nothing,
	// while nil keeps the whole-wave semantics covered by the other tests.
	if _, _, err := f.ws.GenerateFactoryOrderForResults(f.ctx, f.wave.ID, f.factory.ID, []uint{}); !errors.Is(err, ErrNothingToSubmit) {
		t.Fatalf("empty selection = %v, want ErrNothingToSubmit", err)
	}
	results, err := f.ws.Store.ListResults(f.ctx, f.wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 || results[0].Frozen {
		t.Fatalf("result must stay unfrozen after a refused empty submission, got %+v", results)
	}
}

func TestGenerateFactoryOrderLinesSortedByProduct(t *testing.T) {
	f := newFactoryOrderFixture(t)
	zeta := f.createProduct(t, "Zeta Pin", "FACT-Z")
	alpha := f.createProduct(t, "Alpha Pin", "FACT-A")
	for _, p := range []*domain.ProductItem{zeta, alpha} {
		if _, err := f.ws.CreateGrant(f.ctx, f.wave.ID, f.cust.ID, p.ID, 1); err != nil {
			t.Fatalf("CreateGrant %s: %v", p.Name, err)
		}
	}

	_, lines, err := f.ws.GenerateFactoryOrder(f.ctx, f.wave.ID, f.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 order lines, got %d", len(lines))
	}
	if lines[0].FactorySKU != alpha.FactorySKU || lines[1].FactorySKU != zeta.FactorySKU {
		t.Fatalf("order lines must be sorted by product name/sku, got [%s, %s]", lines[0].FactorySKU, lines[1].FactorySKU)
	}
}

// TestImportShipmentOverageWarning pins the shipment-to-order reconciliation:
// parcels within the linked order quantity carry no marker; once the running
// total exceeds the linked quantity (sum of the execution quantity links) the
// shipment records an overage warning in its extra data instead of blocking.
func TestImportShipmentOverageWarning(t *testing.T) {
	p := setupReadyRetailPath(t, "BILI-SKU-OVR")
	ws, ctx := p.ws, p.ctx
	// The retail line's quantity is 2, so the generated order line links 2.
	order, lines, err := ws.GenerateFactoryOrder(ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	links, err := ws.Store.ListLinksByOrderLine(ctx, lines[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].Quantity != 2 {
		t.Fatalf("links = %+v, want one link of quantity 2", links)
	}
	_ = order

	normal, err := ws.ImportShipment(ctx, lines[0].TrackingID, "OVR-1", "SF", "顺丰速运", 2)
	if err != nil {
		t.Fatalf("ImportShipment normal: %v", err)
	}
	if normal.ExtraData != "" {
		t.Fatalf("normal shipment ExtraData = %q, want empty", normal.ExtraData)
	}

	overage, err := ws.ImportShipment(ctx, lines[0].TrackingID, "OVR-2", "SF", "顺丰速运", 2)
	if err != nil {
		t.Fatalf("ImportShipment overage: %v", err)
	}
	// Running total 4 against linked 2 leaves an overage of 2.
	if !strings.Contains(overage.ExtraData, `"overage":2`) {
		t.Fatalf("overage shipment ExtraData = %q, want overage 2 marker", overage.ExtraData)
	}
}
