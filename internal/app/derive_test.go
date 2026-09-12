package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// TestListResultViewsBlocksNonNil pins the wire contract for block-free
// result views: ListResultViews must return a non-nil empty Blocks slice so
// serialization emits [] rather than null, matching the generated TS types
// and the bridge facade that model Blocks as an array.
func TestListResultViewsBlocksNonNil(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()

	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	platforms, err := ws.ListPlatforms(ctx)
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	var factoryPlatform *domain.Platform
	for i := range platforms {
		if platforms[i].Kind == string(domain.PlatformKindFactory) {
			factoryPlatform = &platforms[i]
			break
		}
	}
	if factoryPlatform == nil {
		t.Fatal("expected a builtin factory platform")
	}

	customer := &domain.CustomerProfile{DisplayName: "Blocks Contract"}
	if err := ws.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	address := &domain.RecipientAddress{
		CustomerProfileID: customer.ID,
		RecipientName:     "Blocks Contract",
		AddressLine1:      "1 Contract Street",
		IsDefault:         true,
	}
	if err := ws.CreateAddress(ctx, address); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	product := &domain.ProductItem{
		Name:              "Blocks Contract Medal",
		FactoryPlatformID: factoryPlatform.ID,
		FactorySKU:        "BLOCKS-SKU-001",
	}
	if err := ws.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	wave, err := ws.CreateWave(ctx, "blocks-contract", "")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}

	// An operator grant with an aligned product and the customer's usable
	// default address yields exactly one result view with zero block reasons.
	grant, err := ws.CreateGrant(ctx, wave.ID, customer.ID, product.ID, 1)
	if err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	views, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	var view *ResultView
	for i := range views {
		if views[i].Result.ID == grant.ID {
			view = &views[i]
			break
		}
	}
	if view == nil {
		t.Fatalf("grant result %d not found among %d views", grant.ID, len(views))
	}
	if view.WorkState != domain.WorkStateReady {
		t.Fatalf("expected the block-free grant view to be ready, got %s", view.WorkState)
	}
	if view.Blocks == nil {
		t.Fatal("expected Blocks to be a non-nil empty slice for a block-free view, got nil")
	}
	if len(view.Blocks) != 0 {
		t.Fatalf("expected Blocks to be empty for a block-free view, got %v", view.Blocks)
	}
}
