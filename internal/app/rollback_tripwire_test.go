package app

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// failingCreateStore wraps a store and fails the Nth Create call of the
// chosen kind. WithTx re-wraps the transaction so the injection reaches
// inside WithTx callbacks.
type failingCreateStore struct {
	domain.Store
	calls  *int
	failOn int
	kind   string
}

func (s *failingCreateStore) inject(kind string) error {
	if kind != s.kind {
		return nil
	}
	*s.calls++
	if *s.calls == s.failOn {
		return fmt.Errorf("injected %s failure %d", kind, *s.calls)
	}
	return nil
}

func (s *failingCreateStore) CreateFactLine(ctx context.Context, l *domain.InputFactLine) error {
	if err := s.inject("CreateFactLine"); err != nil {
		return err
	}
	return s.Store.CreateFactLine(ctx, l)
}

func (s *failingCreateStore) CreateWriteback(ctx context.Context, w *domain.ChannelWritebackItem) error {
	if err := s.inject("CreateWriteback"); err != nil {
		return err
	}
	return s.Store.CreateWriteback(ctx, w)
}

func (s *failingCreateStore) WithTx(ctx context.Context, fn func(domain.Store) error) error {
	return s.Store.WithTx(ctx, func(tx domain.Store) error {
		return fn(&failingCreateStore{Store: tx, calls: s.calls, failOn: s.failOn, kind: s.kind})
	})
}

// TestIngestDocumentRollsBackAtomically fails the second fact-line creation
// mid-import and asserts the whole document import (document, facts, lines)
// rolls back.
func TestIngestDocumentRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	base := infra.NewGormStore(gdb)
	ws := NewWorkspace(base)
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	var source *domain.Platform
	for _, p := range mustListPlatforms(t, ws) {
		if p.Kind == string(domain.PlatformKindSource) {
			pc := p
			source = &pc
		}
	}

	facts := []IngestFactInput{
		{
			Kind: string(domain.InputFactKindRetailOrder), StableExternalID: "RB-ORD-1",
			IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "UID-RB-1",
			Lines: []IngestLine{{SourceLineNo: 1, ExternalSKU: "RB-SKU", Quantity: 1}},
		},
		{
			Kind: string(domain.InputFactKindRetailOrder), StableExternalID: "RB-ORD-2",
			IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "UID-RB-2",
			Lines: []IngestLine{{SourceLineNo: 1, ExternalSKU: "RB-SKU", Quantity: 2}},
		},
	}

	calls := 0
	failing := &failingCreateStore{Store: base, calls: &calls, failOn: 2, kind: "CreateFactLine"}
	failWs := NewWorkspace(failing)
	_, _, err := failWs.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail"}, facts)
	if err == nil || !strings.Contains(err.Error(), "injected CreateFactLine failure") {
		t.Fatalf("expected injected failure on the second line creation, got %v", err)
	}

	assertCount := func(table string, want int64) {
		t.Helper()
		var n int64
		if err := gdb.Table(table).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != want {
			t.Fatalf("%s count = %d, want %d", table, n, want)
		}
	}
	assertCount("input_documents", 0)
	assertCount("input_facts", 0)
	assertCount("input_fact_lines", 0)

	// The healthy store still works afterwards (nothing half-committed).
	if _, dups, err := ws.IngestDocument(ctx, &domain.InputDocument{PlatformID: source.ID, DocumentType: "retail"}, facts); err != nil {
		t.Fatalf("clean retry: %v", err)
	} else if len(dups) != 0 {
		t.Fatalf("clean retry duplicates: %+v", dups)
	}
	assertCount("input_documents", 1)
	assertCount("input_facts", 2)
	assertCount("input_fact_lines", 2)
}

// TestGenerateWritebacksRollsBackAtomically fails the first writeback insert
// and asserts nothing lands.
func TestGenerateWritebacksRollsBackAtomically(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	base := infra.NewGormStore(gdb)
	ws := NewWorkspace(base)
	seedBuiltins(t, ws)
	var source, factory *domain.Platform
	for _, p := range mustListPlatforms(t, ws) {
		pc := p
		switch p.Kind {
		case string(domain.PlatformKindSource):
			source = &pc
		case string(domain.PlatformKindFactory):
			factory = &pc
		}
	}
	cloneBuiltinTemplate(t, ws, source.ID, DocumentTypeWriteback)

	fact := &domain.InputFact{PlatformID: source.ID, Kind: string(domain.InputFactKindRetailOrder), StableExternalID: "WB-RB-1"}
	if err := base.CreateFact(ctx, fact); err != nil {
		t.Fatalf("CreateFact: %v", err)
	}
	wave := &domain.Wave{WaveNo: "W-000001", Name: "w", CloseResult: string(domain.WaveCloseResultOpen)}
	if err := base.CreateWave(ctx, wave); err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	customer := &domain.CustomerProfile{DisplayName: "WB RB"}
	if err := base.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	product := &domain.ProductItem{Name: "P", FactoryPlatformID: factory.ID, FactorySKU: "WB-SKU-1"}
	if err := base.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	fid := fact.ID
	result := &domain.FulfillmentResult{WaveID: wave.ID, SourceKind: string(domain.SourceRetailLine), InputFactID: &fid, CustomerProfileID: &customer.ID, ProductItemID: &product.ID, Quantity: 1, Frozen: true}
	if err := base.CreateResult(ctx, result); err != nil {
		t.Fatalf("CreateResult: %v", err)
	}
	order := &domain.SupplierOrder{WaveID: wave.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderExported)}
	if err := base.CreateSupplierOrder(ctx, order); err != nil {
		t.Fatalf("CreateSupplierOrder: %v", err)
	}
	sol := &domain.SupplierOrderLine{SupplierOrderID: order.ID, ProductItemID: product.ID, FactorySKU: product.FactorySKU, Quantity: 1, TrackingID: "trk-wb-rb-1"}
	if err := base.CreateSupplierOrderLine(ctx, sol); err != nil {
		t.Fatalf("CreateSupplierOrderLine: %v", err)
	}
	if err := base.CreateLink(ctx, &domain.ExecutionQuantityLink{FulfillmentResultID: result.ID, SupplierOrderLineID: sol.ID, Quantity: 1}); err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if err := base.CreateShipment(ctx, &domain.Shipment{TrackingID: sol.TrackingID, TrackingNo: "SF-RB-1", CarrierCode: "SF", CarrierName: "顺丰"}); err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}

	calls := 0
	failing := &failingCreateStore{Store: base, calls: &calls, failOn: 1, kind: "CreateWriteback"}
	failWs := NewWorkspace(failing)
	if _, err := failWs.GenerateWritebacks(ctx, fact.ID); err == nil || !strings.Contains(err.Error(), "injected CreateWriteback failure") {
		t.Fatalf("expected injected writeback failure, got %v", err)
	}
	var n int64
	if err := gdb.Table("channel_writeback_items").Count(&n).Error; err != nil {
		t.Fatalf("count writebacks: %v", err)
	}
	if n != 0 {
		t.Fatalf("writeback count = %d, want 0 after rollback", n)
	}

	// The healthy run still produces the item.
	items, err := ws.GenerateWritebacks(ctx, fact.ID)
	if err != nil {
		t.Fatalf("clean GenerateWritebacks: %v", err)
	}
	if len(items) != 1 || items[0].TrackingNo != "SF-RB-1" {
		t.Fatalf("clean run items = %+v", items)
	}
}
