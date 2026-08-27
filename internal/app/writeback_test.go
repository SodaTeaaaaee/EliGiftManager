package app

import (
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// shippedRetailFact drives one retail fact through factory order, shipment,
// and writeback generation, returning the pending writeback items.
func shippedRetailFact(t *testing.T, sku string) (*readyPath, []domain.ChannelWritebackItem) {
	t.Helper()
	p := setupReadyRetailPath(t, sku)
	_, lines, err := p.ws.GenerateFactoryOrder(p.ctx, p.wave.ID, p.factory.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder: %v", err)
	}
	if _, err := p.ws.ImportShipment(p.ctx, lines[0].TrackingID, "SF-WB-0001", "SF", "顺丰速运", 2); err != nil {
		t.Fatalf("ImportShipment: %v", err)
	}
	items, err := p.ws.GenerateWritebacks(p.ctx, p.fact.ID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("writeback items = %d, want 1", len(items))
	}
	return p, items
}

// TestMarkWritebackFailed_MakesWorkStateAndHomeReachable pins the whole
// writeback_failed chain: after a failed mark the home bucket counts the item
// and the linked result views report WorkStateWritebackFailed, the states that
// were previously unreachable because nothing ever wrote the failed status.
func TestMarkWritebackFailed_MakesWorkStateAndHomeReachable(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB1")
	ws, ctx := p.ws, p.ctx

	if err := ws.MarkWritebackFailed(ctx, items[0].ID, "bilibili rejected: order locked"); err != nil {
		t.Fatalf("MarkWritebackFailed: %v", err)
	}

	home, err := ws.Home(ctx)
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if home.WritebackFailed != 1 {
		t.Fatalf("home WritebackFailed = %d, want 1", home.WritebackFailed)
	}

	views, err := ws.ListResultViews(ctx, p.wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	found := false
	for _, v := range views {
		if v.Result.InputFactID != nil && *v.Result.InputFactID == p.fact.ID {
			if v.WorkState != domain.WorkStateWritebackFailed {
				t.Fatalf("retail result work state = %s, want writeback_failed", v.WorkState)
			}
			if !v.WritebackFailed {
				t.Fatal("expected WritebackFailed flag on the result view")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected a view for the retail fact")
	}
}

// TestWritebackStateTransitions covers the retry-history contract: failures
// advance RetryCount and store (truncated) error text, a success clears the
// error but keeps the counter, and the result view falls back to shipped once
// every writeback item is no longer failed.
func TestWritebackStateTransitions(t *testing.T) {
	p, items := shippedRetailFact(t, "BILI-SKU-WB2")
	ws, ctx := p.ws, p.ctx

	if err := ws.MarkWritebackFailed(ctx, items[0].ID, "attempt 1 failed"); err != nil {
		t.Fatalf("MarkWritebackFailed 1: %v", err)
	}
	long := strings.Repeat("错", 600) + "tail"
	if err := ws.MarkWritebackFailed(ctx, items[0].ID, long); err != nil {
		t.Fatalf("MarkWritebackFailed 2: %v", err)
	}
	stored, err := ws.Store.GetWriteback(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("GetWriteback: %v", err)
	}
	if stored.Status != string(domain.WritebackFailed) {
		t.Fatalf("status = %s, want failed", stored.Status)
	}
	if stored.RetryCount != 2 {
		t.Fatalf("retry count = %d, want 2", stored.RetryCount)
	}
	got := []rune(stored.ErrorMessage)
	if len(got) != 500 {
		t.Fatalf("error message rune count = %d, want 500 (truncated)", len(got))
	}

	if err := ws.MarkWritebackSent(ctx, items[0].ID); err != nil {
		t.Fatalf("MarkWritebackSent: %v", err)
	}
	sent, err := ws.Store.GetWriteback(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if sent.Status != string(domain.WritebackSent) {
		t.Fatalf("status = %s, want sent", sent.Status)
	}
	if sent.ErrorMessage != "" {
		t.Fatalf("error message = %q, want cleared after sent", sent.ErrorMessage)
	}
	if sent.RetryCount != 2 {
		t.Fatalf("retry count = %d, want 2 kept after success", sent.RetryCount)
	}
	// The parcel shipped, so once no writeback is failed the result state is
	// shipped again rather than writeback_failed.
	views, err := ws.ListResultViews(ctx, p.wave.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.Result.InputFactID != nil && *v.Result.InputFactID == p.fact.ID {
			if v.WorkState != domain.WorkStateShipped {
				t.Fatalf("retail result work state = %s, want shipped after sent", v.WorkState)
			}
		}
	}
	home, err := ws.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if home.WritebackFailed != 0 {
		t.Fatalf("home WritebackFailed = %d, want 0 after sent", home.WritebackFailed)
	}
}
