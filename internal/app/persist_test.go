package app

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

func TestUniqueKeysAndWaveNo(t *testing.T) {
	ctx := context.Background()
	ws := newTestWorkspace(t)
	store := ws.Store

	src := &domain.Platform{Key: "bilibili", Name: "Bilibili", Kind: string(domain.PlatformKindSource)}
	if err := store.CreatePlatform(ctx, src); err != nil {
		t.Fatalf("create source platform: %v", err)
	}
	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("create factory platform: %v", err)
	}

	if err := store.CreatePlatform(ctx, &domain.Platform{Key: "bilibili", Name: "dup", Kind: string(domain.PlatformKindSource)}); err == nil {
		t.Fatal("expected unique platform key")
	}

	ident := &domain.PlatformIdentity{PlatformID: src.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "U-1", NormalizedValue: "u-1"}
	if err := store.CreateIdentity(ctx, ident); err != nil {
		t.Fatalf("create identity: %v", err)
	}
	if err := store.CreateIdentity(ctx, &domain.PlatformIdentity{PlatformID: src.ID, IdentityType: string(domain.IdentityTypePlatformUID), IdentityValue: "U-1-again", NormalizedValue: "u-1"}); err == nil {
		t.Fatal("expected unique identity (platform, type, normalized)")
	}

	p := &domain.ProductItem{Name: "金徽章", FactoryPlatformID: factory.ID, FactorySKU: "GOLD-01"}
	if err := store.CreateProduct(ctx, p); err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := store.CreateProduct(ctx, &domain.ProductItem{Name: "dup", FactoryPlatformID: factory.ID, FactorySKU: "GOLD-01"}); err == nil {
		t.Fatal("expected unique product factory+sku")
	}

	alias := &domain.ProductAlias{ProductItemID: p.ID, PlatformID: src.ID, ExternalProductID: "ext-gold"}
	if err := store.CreateAlias(ctx, alias); err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if err := store.CreateAlias(ctx, &domain.ProductAlias{ProductItemID: p.ID, PlatformID: src.ID, ExternalProductID: "ext-gold"}); err == nil {
		t.Fatal("expected unique alias platform+external id")
	}

	no, err := store.NextWaveNo(ctx)
	if err != nil {
		t.Fatalf("NextWaveNo: %v", err)
	}
	w := &domain.Wave{WaveNo: no, Name: "2026-08", CloseResult: string(domain.WaveCloseResultOpen)}
	if err := store.CreateWave(ctx, w); err != nil {
		t.Fatalf("create wave: %v", err)
	}
	if err := store.CreateWave(ctx, &domain.Wave{WaveNo: no, Name: "dup", CloseResult: string(domain.WaveCloseResultOpen)}); err == nil {
		t.Fatal("expected unique wave no")
	}

	fact := &domain.InputFact{PlatformID: src.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: "mem-1"}
	if err := store.CreateFact(ctx, fact); err != nil {
		t.Fatalf("create fact: %v", err)
	}
	if err := store.CreateFact(ctx, &domain.InputFact{PlatformID: src.ID, Kind: string(domain.InputFactKindMembership), StableExternalID: "mem-1"}); err == nil {
		t.Fatal("expected unique input fact platform+stable id")
	}

	order := &domain.SupplierOrder{WaveID: w.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderGenerated)}
	if err := store.CreateSupplierOrder(ctx, order); err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	line := &domain.SupplierOrderLine{SupplierOrderID: order.ID, ProductItemID: p.ID, FactorySKU: p.FactorySKU, Quantity: 1, TrackingID: "trk-aaa"}
	if err := store.CreateSupplierOrderLine(ctx, line); err != nil {
		t.Fatalf("create supplier line: %v", err)
	}
	if err := store.CreateSupplierOrderLine(ctx, &domain.SupplierOrderLine{SupplierOrderID: order.ID, ProductItemID: p.ID, FactorySKU: p.FactorySKU, Quantity: 1, TrackingID: "trk-aaa"}); err == nil {
		t.Fatal("expected unique tracking id")
	}

	tid, err := RandomTrackingID()
	if err != nil {
		t.Fatalf("RandomTrackingID: %v", err)
	}
	if len(tid) != 32 {
		t.Fatalf("tracking id length = %d, want 32 hex chars", len(tid))
	}
	if tid == strconv.FormatUint(uint64(line.ID), 10) {
		t.Fatalf("tracking id must not equal supplier line id, got %s", tid)
	}
}

func TestOpenSupplierOrderUniquePerWaveFactory(t *testing.T) {
	ctx := context.Background()
	store := newTestWorkspace(t).Store
	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("platform: %v", err)
	}
	w := &domain.Wave{WaveNo: "W-000001", Name: "w", CloseResult: string(domain.WaveCloseResultOpen)}
	if err := store.CreateWave(ctx, w); err != nil {
		t.Fatalf("wave: %v", err)
	}
	if err := store.CreateSupplierOrder(ctx, &domain.SupplierOrder{WaveID: w.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderGenerated)}); err != nil {
		t.Fatalf("first order: %v", err)
	}
	err := store.CreateSupplierOrder(ctx, &domain.SupplierOrder{WaveID: w.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderGenerated)})
	if err == nil {
		t.Fatal("expected unique open order per wave+factory")
	}
	if !isConstraint(err) {
		t.Fatalf("constraint error, got %v", err)
	}
}

func TestExportedSupplierOrderDoesNotOccupyOpenSlot(t *testing.T) {
	ctx := context.Background()
	store := newTestWorkspace(t).Store
	factory := &domain.Platform{Key: "rozao", Name: "柔造", Kind: string(domain.PlatformKindFactory)}
	if err := store.CreatePlatform(ctx, factory); err != nil {
		t.Fatalf("platform: %v", err)
	}
	w := &domain.Wave{WaveNo: "W-000001", Name: "w", CloseResult: string(domain.WaveCloseResultOpen)}
	if err := store.CreateWave(ctx, w); err != nil {
		t.Fatalf("wave: %v", err)
	}
	if err := store.CreateSupplierOrder(ctx, &domain.SupplierOrder{WaveID: w.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderExported)}); err != nil {
		t.Fatalf("exported order: %v", err)
	}
	if _, err := store.FindOpenSupplierOrder(ctx, w.ID, factory.ID); err != domain.ErrNotFound {
		t.Fatalf("FindOpenSupplierOrder after export = %v, want ErrNotFound", err)
	}
	if err := store.CreateSupplierOrder(ctx, &domain.SupplierOrder{WaveID: w.ID, FactoryPlatformID: factory.ID, Status: string(domain.SupplierOrderGenerated)}); err != nil {
		t.Fatalf("generated after exported: %v", err)
	}
}

func isConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint") || errors.Is(err, domain.ErrNotFound)
}

// TestFindFactByStableIDAnchorsEstablishedFact pins the lookup to the partial
// unique index semantics: revision facts share the stable external id of the
// fact they revise, and only the established fact (revises_id IS NULL) owns
// the anchor slot — with concurrent pending revisions and regardless of the
// revision chain's id ordering.
func TestFindFactByStableIDAnchorsEstablishedFact(t *testing.T) {
	ctx := context.Background()
	gdb := openTestDB(t)
	store := infra.NewGormStore(gdb)

	src := &domain.Platform{Key: "bilibili", Name: "Bilibili", Kind: string(domain.PlatformKindSource)}
	if err := store.CreatePlatform(ctx, src); err != nil {
		t.Fatalf("create source platform: %v", err)
	}

	orig := &domain.InputFact{PlatformID: src.ID, Kind: string(domain.InputFactKindRetailOrder), StableExternalID: "ANCHOR-1"}
	if err := store.CreateFact(ctx, orig); err != nil {
		t.Fatalf("create original: %v", err)
	}
	// Two pending revisions of the same established fact share its stable id.
	for i := 0; i < 2; i++ {
		rev := &domain.InputFact{PlatformID: src.ID, Kind: string(domain.InputFactKindRetailOrder), StableExternalID: "ANCHOR-1", RevisesID: &orig.ID}
		if err := store.CreateFact(ctx, rev); err != nil {
			t.Fatalf("create revision %d: %v", i, err)
		}
	}
	found, err := store.FindFactByStableID(ctx, src.ID, "ANCHOR-1")
	if err != nil {
		t.Fatalf("FindFactByStableID: %v", err)
	}
	if found.ID != orig.ID {
		t.Fatalf("lookup must anchor the established fact %d, got %d", orig.ID, found.ID)
	}

	// Reorder the chain so a revision carries a lower id than the original:
	// the anchor must not move. (Only the revision's own id changes, so the
	// revises_id reference stays intact.)
	var revID uint
	if err := gdb.Raw("SELECT id FROM input_facts WHERE revises_id = ? ORDER BY id LIMIT 1", orig.ID).Scan(&revID).Error; err != nil {
		t.Fatalf("pick revision: %v", err)
	}
	if err := gdb.Exec("UPDATE input_facts SET id = 0 WHERE id = ?", revID).Error; err != nil {
		t.Fatalf("reorder revision id: %v", err)
	}
	found, err = store.FindFactByStableID(ctx, src.ID, "ANCHOR-1")
	if err != nil {
		t.Fatalf("FindFactByStableID after reorder: %v", err)
	}
	if found.ID != orig.ID || found.RevisesID != nil {
		t.Fatalf("lookup must still anchor the established fact %d, got fact %d (revises %v)", orig.ID, found.ID, found.RevisesID)
	}
}
