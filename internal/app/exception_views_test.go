package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// setupMembershipInstance ingests one membership fact, attaches its identity
// to a customer, and assigns the line into a fresh wave so the wave holds one
// entitlement instance.
func setupMembershipInstance(t *testing.T) (*Workspace, context.Context, domain.Wave, domain.ProductItem) {
	t.Helper()
	ws := newTestWorkspace(t)
	ctx := context.Background()
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatal(err)
	}
	source := platformByKind(t, ws, domain.PlatformKindSource)
	factory := platformByKind(t, ws, domain.PlatformKindFactory)

	customer := &domain.CustomerProfile{DisplayName: "Coco"}
	if err := ws.CreateCustomer(ctx, customer); err != nil {
		t.Fatal(err)
	}
	product := &domain.ProductItem{Name: "Fan Pin", FactoryPlatformID: factory.ID, FactorySKU: "EXC-SKU-1"}
	if err := ws.CreateProduct(ctx, product); err != nil {
		t.Fatal(err)
	}

	doc := &domain.InputDocument{PlatformID: source.ID, DocumentType: DocumentTypeMembershipList}
	if _, _, err := ws.IngestDocument(ctx, doc, []IngestFactInput{{
		Kind:             string(domain.InputFactKindMembership),
		StableExternalID: "MEM-EXC-1",
		IdentityType:     string(domain.IdentityTypePlatformUID),
		IdentityValue:    "uid-exc",
		MembershipLevel:  "captain",
		Lines:            []IngestLine{{SourceLineNo: 1, Quantity: 1}},
	}}); err != nil {
		t.Fatal(err)
	}
	facts, err := ws.Store.ListFactsByDocument(ctx, doc.ID)
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts = %v err = %v", facts, err)
	}
	ident, err := ws.Store.FindIdentity(ctx, source.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("uid-exc"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AttachIdentity(ctx, ident.ID, customer.ID); err != nil {
		t.Fatal(err)
	}
	lines, err := ws.Store.ListFactLines(ctx, facts[0].ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("lines = %v err = %v", lines, err)
	}
	wave, err := ws.CreateWave(ctx, "exceptions wave", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.AssignLines(ctx, wave.ID, []uint{lines[0].ID}); err != nil {
		t.Fatal(err)
	}
	return ws, ctx, *wave, *product
}

// TestListEntitlementInstancesView checks the instance list the exception
// editor needs: one row per member with the resolved customer name, the
// platform identity summary, and the membership level.
func TestListEntitlementInstancesView(t *testing.T) {
	ws, ctx, wave, _ := setupMembershipInstance(t)
	views, err := ws.ListEntitlementInstances(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListEntitlementInstances: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("instance views = %d, want 1", len(views))
	}
	v := views[0]
	if v.CustomerName != "Coco" {
		t.Fatalf("customer name = %q, want Coco (resolved through the identity attachment)", v.CustomerName)
	}
	if v.PlatformIdentity != string(domain.IdentityTypePlatformUID)+":uid-exc" {
		t.Fatalf("platform identity = %q", v.PlatformIdentity)
	}
	if v.MembershipLevel != "captain" {
		t.Fatalf("membership level = %q, want captain", v.MembershipLevel)
	}
}

// TestListExceptionsView checks the exception list: display fields (customer
// and product names) are joined in the app layer so the frontend never needs
// raw instance or product ids.
func TestListExceptionsView(t *testing.T) {
	ws, ctx, wave, product := setupMembershipInstance(t)
	insts, err := ws.ListEntitlementInstances(ctx, wave.ID)
	if err != nil || len(insts) != 1 {
		t.Fatalf("instances = %v err = %v", insts, err)
	}
	if err := ws.AddException(ctx, &domain.EntitlementException{
		WaveID:     wave.ID,
		ProductID:  product.ID,
		InstanceID: insts[0].ID,
		Quantity:   2,
		Note:       "fan club bonus",
	}); err != nil {
		t.Fatalf("AddException: %v", err)
	}
	views, err := ws.ListExceptions(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListExceptions: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("exception views = %d, want 1", len(views))
	}
	v := views[0]
	if v.InstanceID != insts[0].ID || v.ProductItemID != product.ID || v.Quantity != 2 || v.Note != "fan club bonus" {
		t.Fatalf("view = %+v", v)
	}
	if v.CustomerName != "Coco" {
		t.Fatalf("customer name = %q, want Coco", v.CustomerName)
	}
	if v.ProductName != "Fan Pin" {
		t.Fatalf("product name = %q, want Fan Pin", v.ProductName)
	}
}
