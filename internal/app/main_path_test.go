package app

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestMainPath(t *testing.T) {
	ws := newTestWorkspace(t)
	ctx := context.Background()

	// 1. Ensure source (bilibili) and factory (rozao) platforms exist.
	if err := ws.EnsureBuiltinPlatforms(ctx); err != nil {
		t.Fatalf("EnsureBuiltinPlatforms: %v", err)
	}
	platforms, err := ws.ListPlatforms(ctx)
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	var sourcePlatform, factoryPlatform *domain.Platform
	for i := range platforms {
		if platforms[i].Key == "bilibili" && platforms[i].Kind == string(domain.PlatformKindSource) {
			sourcePlatform = &platforms[i]
		}
		if platforms[i].Key == "rozao" && platforms[i].Kind == string(domain.PlatformKindFactory) {
			factoryPlatform = &platforms[i]
		}
	}
	if sourcePlatform == nil || factoryPlatform == nil {
		t.Fatalf("expected both bilibili source and rozao factory platforms, got source=%v factory=%v", sourcePlatform, factoryPlatform)
	}

	// 2. Create customer, default usable address, product bound to factory SKU, and alias on source platform.
	customer := &domain.CustomerProfile{
		DisplayName: "Alice",
		Notes:       "VIP member",
	}
	if err := ws.CreateCustomer(ctx, customer); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	if customer.ID == 0 {
		t.Fatal("expected non-zero customer ID")
	}

	address := &domain.RecipientAddress{
		CustomerProfileID: customer.ID,
		RecipientName:     "Alice Zhang",
		Phone:             "13800000000",
		Country:           "CN",
		Province:          "Shanghai",
		City:              "Shanghai",
		District:          "Pudong",
		AddressLine1:      "100 Century Avenue",
		IsDefault:         true,
	}
	if err := ws.CreateAddress(ctx, address); err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	if address.ID == 0 {
		t.Fatal("expected non-zero address ID")
	}

	product := &domain.ProductItem{
		Name:              "Commemorative Medal",
		FactoryPlatformID: factoryPlatform.ID,
		FactorySKU:        "ROZAO-MEDAL-001",
		Notes:             "Annual gift",
	}
	if err := ws.CreateProduct(ctx, product); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if product.ID == 0 {
		t.Fatal("expected non-zero product ID")
	}

	alias := &domain.ProductAlias{
		ProductItemID:     product.ID,
		PlatformID:        sourcePlatform.ID,
		ExternalProductID: "BILI-SKU-MEDAL",
		Title:             "Bilibili 2026 Medal",
		Spec:              "Gold Edition",
	}
	if err := ws.CreateAlias(ctx, alias); err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}
	if alias.ID == 0 {
		t.Fatal("expected non-zero alias ID")
	}

	// 3. Ingest membership document: 1 fact, exactly 1 line, identity value and membership level, stable external ID.
	memDoc := &domain.InputDocument{
		PlatformID:   sourcePlatform.ID,
		DocumentType: "membership_roster",
		OriginalName: "members_2026_08.xlsx",
	}
	memFacts := []IngestFactInput{
		{
			Kind:             string(domain.InputFactKindMembership),
			StableExternalID: "MEM-FACT-1001",
			IdentityType:     string(domain.IdentityTypePlatformUID),
			IdentityValue:    "UID-99001",
			MembershipLevel:  "captain",
			Lines: []IngestLine{
				{
					SourceLineNo: 1,
					Quantity:     1,
				},
			},
		},
	}
	if _, dups, err := ws.IngestDocument(ctx, memDoc, memFacts); err != nil {
		t.Fatalf("IngestDocument membership: %v", err)
	} else if len(dups) > 0 {
		t.Fatalf("expected 0 duplicates for initial membership ingestion, got %d", len(dups))
	}

	memFactList, err := ws.Store.ListFactsByDocument(ctx, memDoc.ID)
	if err != nil {
		t.Fatalf("ListFactsByDocument membership: %v", err)
	}
	if len(memFactList) != 1 {
		t.Fatalf("expected 1 membership fact, got %d", len(memFactList))
	}
	memLines, err := ws.Store.ListFactLines(ctx, memFactList[0].ID)
	if err != nil {
		t.Fatalf("ListFactLines membership: %v", err)
	}
	if len(memLines) != 1 {
		t.Fatalf("expected exactly 1 membership line, got %d", len(memLines))
	}

	// 4. Ingest retail order document: 1 fact, 2 lines (two quantities), stable external ID, same identity.
	retailDoc := &domain.InputDocument{
		PlatformID:   sourcePlatform.ID,
		DocumentType: "retail_order_export",
		OriginalName: "retail_orders_2026_08.xlsx",
	}
	retailFacts := []IngestFactInput{
		{
			Kind:             string(domain.InputFactKindRetailOrder),
			StableExternalID: "RETAIL-FACT-2001",
			IdentityType:     string(domain.IdentityTypePlatformUID),
			IdentityValue:    "UID-99001",
			SourceDocumentNo: "ORD-20260826-001",
			Lines: []IngestLine{
				{
					SourceLineNo: 1,
					ExternalSKU:  "BILI-SKU-MEDAL",
					Quantity:     2,
				},
				{
					SourceLineNo: 2,
					ExternalSKU:  "BILI-SKU-MEDAL",
					Quantity:     3,
				},
			},
		},
	}
	if _, dups, err := ws.IngestDocument(ctx, retailDoc, retailFacts); err != nil {
		t.Fatalf("IngestDocument retail: %v", err)
	} else if len(dups) > 0 {
		t.Fatalf("expected 0 duplicates for initial retail ingestion, got %d", len(dups))
	}

	retailFactList, err := ws.Store.ListFactsByDocument(ctx, retailDoc.ID)
	if err != nil {
		t.Fatalf("ListFactsByDocument retail: %v", err)
	}
	if len(retailFactList) != 1 {
		t.Fatalf("expected 1 retail fact, got %d", len(retailFactList))
	}
	retailFactID := retailFactList[0].ID
	retailLines, err := ws.Store.ListFactLines(ctx, retailFactID)
	if err != nil {
		t.Fatalf("ListFactLines retail: %v", err)
	}
	if len(retailLines) != 2 {
		t.Fatalf("expected 2 retail lines, got %d", len(retailLines))
	}

	// 5. Check unattached identity behavior before AttachIdentity.
	ident, err := ws.Store.FindIdentity(ctx, sourcePlatform.ID, string(domain.IdentityTypePlatformUID), NormalizeIdentity("UID-99001"))
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if ident.CustomerProfileID != nil {
		t.Fatalf("expected unattached identity before attach, got customer ID %v", *ident.CustomerProfileID)
	}

	// 6. Create wave and assign membership + retail lines into that wave.
	wave, err := ws.CreateWave(ctx, "August 2026 Wave", "Main test wave")
	if err != nil {
		t.Fatalf("CreateWave: %v", err)
	}
	if wave.WaveNo == "" {
		t.Fatal("expected assigned WaveNo")
	}

	lineIDs := []uint{memLines[0].ID, retailLines[0].ID, retailLines[1].ID}
	if err := ws.AssignLines(ctx, wave.ID, lineIDs); err != nil {
		t.Fatalf("AssignLines: %v", err)
	}

	// 7. UpsertRule on the wave with selector type platform_level (bilibili + captain) for the product.
	rule := &domain.EntitlementRule{
		WaveID:    wave.ID,
		ProductID: product.ID,
		Selector: domain.EntitlementSelector{
			Type:       string(domain.SelectorPlatformLevel),
			PlatformID: sourcePlatform.ID,
			Level:      "captain",
		},
		Quantity: 1,
		Active:   true,
	}
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}

	// Before attaching identity, entitlement result view reports identity_unattached block.
	viewsBeforeAttach, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews before attach: %v", err)
	}
	foundUnattachedBlock := false
	for _, v := range viewsBeforeAttach {
		if v.Result.SourceKind == string(domain.SourceEntitlementInstance) {
			for _, b := range v.Blocks {
				if b == domain.BlockIdentityUnattached {
					foundUnattachedBlock = true
					break
				}
			}
		}
	}
	if !foundUnattachedBlock {
		t.Fatal("expected BlockIdentityUnattached before AttachIdentity on entitlement result")
	}

	// Attach identity to the customer profile and recompute.
	if err := ws.AttachIdentity(ctx, ident.ID, customer.ID); err != nil {
		t.Fatalf("AttachIdentity: %v", err)
	}
	if err := ws.RecomputeEntitlements(ctx, wave.ID); err != nil {
		t.Fatalf("RecomputeEntitlements: %v", err)
	}

	// Assert source-level FulfillmentResult leaves exist (entitlement_instance and retail_line).
	results, err := ws.Store.ListResults(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	var entitlementResults, retailResults []domain.FulfillmentResult
	for _, r := range results {
		switch r.SourceKind {
		case string(domain.SourceEntitlementInstance):
			entitlementResults = append(entitlementResults, r)
		case string(domain.SourceRetailLine):
			retailResults = append(retailResults, r)
		}
	}
	if len(entitlementResults) != 1 {
		t.Fatalf("expected 1 entitlement result, got %d", len(entitlementResults))
	}
	if entitlementResults[0].Quantity != 1 {
		t.Fatalf("expected entitlement result quantity 1, got %d", entitlementResults[0].Quantity)
	}
	if len(retailResults) != 2 {
		t.Fatalf("expected 2 retail results, got %d", len(retailResults))
	}

	// 8. Change rule quantity and call UpsertRule again. Assert live recomputation.
	rule.Quantity = 5
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("UpsertRule updated quantity: %v", err)
	}
	updatedResults, err := ws.Store.ListResults(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResults after rule update: %v", err)
	}
	var recomputedEntitlement *domain.FulfillmentResult
	for i := range updatedResults {
		if updatedResults[i].SourceKind == string(domain.SourceEntitlementInstance) {
			recomputedEntitlement = &updatedResults[i]
			break
		}
	}
	if recomputedEntitlement == nil {
		t.Fatal("expected entitlement result after rule update")
	}
	if recomputedEntitlement.Quantity != 5 {
		t.Fatalf("expected recomputed quantity 5, got %d", recomputedEntitlement.Quantity)
	}

	// 9. CreateGrant(wave, customer, product, qty) as operator grant (no input document).
	grantResult, err := ws.CreateGrant(ctx, wave.ID, customer.ID, product.ID, 4)
	if err != nil {
		t.Fatalf("CreateGrant: %v", err)
	}
	if grantResult.SourceKind != string(domain.SourceOperatorGrant) {
		t.Fatalf("expected source_kind operator_grant, got %s", grantResult.SourceKind)
	}
	if grantResult.Quantity != 4 {
		t.Fatalf("expected grant quantity 4, got %d", grantResult.Quantity)
	}
	if grantResult.ProductItemID == nil || *grantResult.ProductItemID != product.ID {
		t.Fatalf("expected grant product item ID %d, got %v", product.ID, grantResult.ProductItemID)
	}

	// 10. SetResultAddress on results that need a usable snapshot if still blocking on unusable_address.
	views, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews: %v", err)
	}
	for _, v := range views {
		for _, b := range v.Blocks {
			if b == domain.BlockUnusableAddress {
				if err := ws.SetResultAddress(ctx, v.Result.ID, address.ID); err != nil {
					t.Fatalf("SetResultAddress: %v", err)
				}
			}
		}
	}

	viewsAfterAddress, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after address setup: %v", err)
	}
	for _, v := range viewsAfterAddress {
		if v.WorkState != domain.WorkStateReady {
			t.Fatalf("expected result %d to be in ready state, got %s with blocks %v", v.Result.ID, v.WorkState, v.Blocks)
		}
	}

	// 11. GenerateFactoryOrder(wave, factory):
	// - one SupplierOrderLine per product group
	// - TrackingID is random 32-char hex from RandomTrackingID / NewTrackingID
	// - TrackingID is unique per line
	// - TrackingID != fmt of FulfillmentResult.ID
	// - corresponding results Frozen == true
	order1, lines1, err := ws.GenerateFactoryOrder(ctx, wave.ID, factoryPlatform.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder order1: %v", err)
	}
	if order1.Status != string(domain.SupplierOrderGenerated) {
		t.Fatalf("expected order status generated, got %s", order1.Status)
	}
	if len(lines1) != 1 {
		t.Fatalf("expected 1 supplier order line for single product group, got %d", len(lines1))
	}
	expectedTotalQty := 5 + 2 + 3 + 4 // 14
	if lines1[0].Quantity != expectedTotalQty {
		t.Fatalf("expected aggregated order line quantity %d, got %d", expectedTotalQty, lines1[0].Quantity)
	}

	firstTrackingID := lines1[0].TrackingID
	if len(firstTrackingID) != 32 {
		t.Fatalf("expected 32-char hex tracking id, got length %d: %q", len(firstTrackingID), firstTrackingID)
	}
	if _, err := hex.DecodeString(firstTrackingID); err != nil {
		t.Fatalf("tracking id is not valid hex: %v", err)
	}

	frozenViews, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after generate: %v", err)
	}
	var frozenEntitlement domain.FulfillmentResult
	entitlementCount := 0
	for _, v := range frozenViews {
		if !v.Result.Frozen {
			t.Fatalf("expected result %d to be frozen after order generation", v.Result.ID)
		}
		if firstTrackingID == fmt.Sprintf("%d", v.Result.ID) {
			t.Fatalf("tracking id must not equal fulfillment result id %d", v.Result.ID)
		}
		if v.Result.SourceKind == string(domain.SourceEntitlementInstance) {
			entitlementCount++
			frozenEntitlement = v.Result
		}
	}
	if entitlementCount != 1 {
		t.Fatalf("expected 1 frozen entitlement result, got %d", entitlementCount)
	}

	rule.Quantity = 9
	if err := ws.UpsertRule(ctx, rule); err != nil {
		t.Fatalf("UpsertRule after freeze: %v", err)
	}
	afterFreezeEdit, err := ws.Store.ListResults(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResults after freeze+rule edit: %v", err)
	}
	entitlementAfterEdit := 0
	unfrozenEntitlement := 0
	for _, r := range afterFreezeEdit {
		if r.SourceKind != string(domain.SourceEntitlementInstance) {
			continue
		}
		entitlementAfterEdit++
		if !r.Frozen {
			unfrozenEntitlement++
		}
		if r.ID != frozenEntitlement.ID {
			t.Fatalf("expected frozen entitlement result %d to remain, found %d", frozenEntitlement.ID, r.ID)
		}
		if r.Quantity != frozenEntitlement.Quantity {
			t.Fatalf("frozen entitlement quantity changed from %d to %d", frozenEntitlement.Quantity, r.Quantity)
		}
	}
	if entitlementAfterEdit != 1 {
		t.Fatalf("expected no cloned entitlement result after freeze+rule edit, got %d", entitlementAfterEdit)
	}
	if unfrozenEntitlement != 0 {
		t.Fatalf("expected no unfrozen entitlement clone after freeze+rule edit, got %d", unfrozenEntitlement)
	}

	// 12. VoidFactoryOrder on unexported order: results unfrozen, tracking ids retired.
	if err := ws.VoidFactoryOrder(ctx, order1.ID); err != nil {
		t.Fatalf("VoidFactoryOrder order1: %v", err)
	}
	voidedOrder, err := ws.Store.GetSupplierOrder(ctx, order1.ID)
	if err != nil {
		t.Fatalf("GetSupplierOrder voided: %v", err)
	}
	if voidedOrder.Status != string(domain.SupplierOrderVoided) || voidedOrder.VoidedAt == nil {
		t.Fatalf("expected voided status and voided_at set, got status=%s voided_at=%v", voidedOrder.Status, voidedOrder.VoidedAt)
	}

	retired, err := ws.Store.TrackingIDRetired(ctx, firstTrackingID)
	if err != nil {
		t.Fatalf("TrackingIDRetired check: %v", err)
	}
	if !retired {
		t.Fatalf("expected tracking id %s to be retired", firstTrackingID)
	}

	voidedLine, err := ws.Store.GetSupplierOrderLine(ctx, lines1[0].ID)
	if err != nil {
		t.Fatalf("GetSupplierOrderLine voided line: %v", err)
	}
	if !voidedLine.TrackingRetired {
		t.Fatal("expected supplier order line TrackingRetired to be true")
	}

	unfrozenViews, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after void: %v", err)
	}
	for _, v := range unfrozenViews {
		if v.Result.Frozen {
			t.Fatalf("expected result %d to be unfrozen after voiding order", v.Result.ID)
		}
	}

	// 13. GenerateFactoryOrder again, ExportFactoryOrder, ExportFactoryOrder a second time: tracking IDs unchanged.
	order2, lines2, err := ws.GenerateFactoryOrder(ctx, wave.ID, factoryPlatform.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder order2: %v", err)
	}
	if order2.ID == order1.ID {
		t.Fatalf("expected new order ID, got same %d", order2.ID)
	}
	if len(lines2) != 1 {
		t.Fatalf("expected 1 order line in order2, got %d", len(lines2))
	}
	secondTrackingID := lines2[0].TrackingID
	if secondTrackingID == firstTrackingID {
		t.Fatalf("expected new tracking id on re-generation, got retired id %s", secondTrackingID)
	}
	retiredSecond, err := ws.Store.TrackingIDRetired(ctx, secondTrackingID)
	if err != nil {
		t.Fatalf("TrackingIDRetired check on second tracking ID: %v", err)
	}
	if retiredSecond {
		t.Fatalf("new tracking id %s must not be retired", secondTrackingID)
	}

	exp1, err := ws.ExportFactoryOrder(ctx, order2.ID)
	if err != nil {
		t.Fatalf("ExportFactoryOrder first: %v", err)
	}
	if exp1.Status != string(domain.SupplierOrderExported) || exp1.ExportedAt == nil {
		t.Fatalf("expected exported status, got status=%s exported_at=%v", exp1.Status, exp1.ExportedAt)
	}

	exp2, err := ws.ExportFactoryOrder(ctx, order2.ID)
	if err != nil {
		t.Fatalf("ExportFactoryOrder second: %v", err)
	}
	if exp2.Status != string(domain.SupplierOrderExported) {
		t.Fatalf("expected exported status on repeat export, got %s", exp2.Status)
	}

	linesAfterRepeatExport, err := ws.Store.ListSupplierOrderLines(ctx, order2.ID)
	if err != nil {
		t.Fatalf("ListSupplierOrderLines after repeat export: %v", err)
	}
	if len(linesAfterRepeatExport) != 1 || linesAfterRepeatExport[0].TrackingID != secondTrackingID {
		t.Fatalf("expected unchanged tracking ID %s after repeat export, got %v", secondTrackingID, linesAfterRepeatExport)
	}

	// 14. VoidFactoryOrder on exported order must fail with ErrOrderExported.
	if err := ws.VoidFactoryOrder(ctx, order2.ID); !errors.Is(err, ErrOrderExported) {
		t.Fatalf("expected ErrOrderExported on voiding exported order, got %v", err)
	}

	grantAfterExport, err := ws.CreateGrant(ctx, wave.ID, customer.ID, product.ID, 2)
	if err != nil {
		t.Fatalf("CreateGrant after export: %v", err)
	}
	viewsAfterGrant, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after post-export grant: %v", err)
	}
	for _, v := range viewsAfterGrant {
		if v.Result.ID != grantAfterExport.ID {
			continue
		}
		for _, b := range v.Blocks {
			if b == domain.BlockUnusableAddress {
				if err := ws.SetResultAddress(ctx, v.Result.ID, address.ID); err != nil {
					t.Fatalf("SetResultAddress on post-export grant: %v", err)
				}
			}
		}
	}
	order3, lines3, err := ws.GenerateFactoryOrder(ctx, wave.ID, factoryPlatform.ID)
	if err != nil {
		t.Fatalf("GenerateFactoryOrder after export: %v", err)
	}
	if order3.ID == order2.ID {
		t.Fatalf("expected a new factory order after export, got %d", order3.ID)
	}
	if len(lines3) == 0 {
		t.Fatal("expected supplier lines on second generated order after export")
	}

	// 15. ImportShipment by tracking id: import two shipments / parcels for secondTrackingID.
	shipment1, err := ws.ImportShipment(ctx, secondTrackingID, "SF20260826001", "SF", "顺丰速运", 7)
	if err != nil {
		t.Fatalf("ImportShipment 1: %v", err)
	}
	if shipment1.ID == 0 {
		t.Fatal("expected non-zero shipment1 ID")
	}

	shipment2, err := ws.ImportShipment(ctx, secondTrackingID, "SF20260826002", "SF", "顺丰速运", 7)
	if err != nil {
		t.Fatalf("ImportShipment 2: %v", err)
	}
	if shipment2.ID == 0 {
		t.Fatal("expected non-zero shipment2 ID")
	}

	shippedViews, err := ws.ListResultViews(ctx, wave.ID)
	if err != nil {
		t.Fatalf("ListResultViews after shipments: %v", err)
	}
	for _, v := range shippedViews {
		if v.Result.ID == grantAfterExport.ID {
			if v.WorkState != domain.WorkStateInFactory {
				t.Fatalf("post-export grant result %d work state = %s, want in_factory", v.Result.ID, v.WorkState)
			}
			continue
		}
		if v.WorkState != domain.WorkStateShipped {
			t.Fatalf("expected result %d to be in shipped state, got %s", v.Result.ID, v.WorkState)
		}
	}

	// 16. GenerateWritebacks(retailFactID): grouped by InputFact, returns 2 items for the two parcels.
	writebackItems, err := ws.GenerateWritebacks(ctx, retailFactID)
	if err != nil {
		t.Fatalf("GenerateWritebacks: %v", err)
	}
	if len(writebackItems) != 2 {
		t.Fatalf("expected 2 writeback items for 2 shipments on the retail fact, got %d", len(writebackItems))
	}
	foundTrackingNos := map[string]bool{}
	for _, item := range writebackItems {
		if item.InputFactID != retailFactID {
			t.Fatalf("expected writeback InputFactID %d, got %d", retailFactID, item.InputFactID)
		}
		if item.Status != string(domain.WritebackPending) {
			t.Fatalf("expected writeback status pending, got %s", item.Status)
		}
		foundTrackingNos[item.TrackingNo] = true
	}
	if !foundTrackingNos["SF20260826001"] || !foundTrackingNos["SF20260826002"] {
		t.Fatalf("expected both tracking numbers in writebacks, got %v", foundTrackingNos)
	}

	// 17. Home() buckets are populated and RecentWaves includes the created wave.
	homeBuckets, err := ws.Home(ctx)
	if err != nil {
		t.Fatalf("Home(): %v", err)
	}
	foundWaveInHome := false
	for _, w := range homeBuckets.RecentWaves {
		if w.ID == wave.ID {
			foundWaveInHome = true
			break
		}
	}
	if !foundWaveInHome {
		t.Fatalf("expected RecentWaves in HomeBuckets to include wave ID %d, got %v", wave.ID, homeBuckets.RecentWaves)
	}
}
