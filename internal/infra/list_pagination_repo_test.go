package infra

import (
	"context"
	"fmt"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupListPaginationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:list-pagination-%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&persistence.CustomerProfile{}, &persistence.CustomerIdentity{}, &persistence.CustomerAddress{},
		&persistence.ProductMaster{}, &persistence.DemandDocument{}, &persistence.DemandLine{},
		&persistence.WaveDemandAssignment{}, &persistence.SupplierOrder{}, &persistence.Shipment{},
		&persistence.ShipmentLine{},
		&persistence.CustomerNameObservation{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestListPaginationRepository_Customers(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()
	profiles := []persistence.CustomerProfile{
		{DisplayName: "Zulu", ProfileType: "buyer"},
		{DisplayName: "Alpha", ProfileType: "member"},
		{DisplayName: "Middle", ProfileType: "manual"},
		{DisplayName: "Deleted", ProfileType: "member"},
	}
	for i := range profiles {
		if err := db.Create(&profiles[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	identities := []persistence.CustomerIdentity{
		{CustomerProfileID: profiles[0].ID, IdentityPlatform: "Patreon", IdentityValue: "z-user", IdentityType: "platform_uid"},
		{CustomerProfileID: profiles[1].ID, IdentityPlatform: "Shop", IdentityValue: "needle", IdentityType: "platform_uid"},
		{CustomerProfileID: profiles[2].ID, IdentityPlatform: "Patreon", IdentityValue: "m-user", IdentityType: "platform_uid"},
		{CustomerProfileID: profiles[3].ID, IdentityPlatform: "DeletedPlatform", IdentityValue: "old-user", IdentityType: "platform_uid"},
	}
	for i := range identities {
		if err := db.Create(&identities[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	address := persistence.CustomerAddress{CustomerProfileID: profiles[0].ID, RecipientName: "Zulu"}
	if err := db.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	deletedAddress := persistence.CustomerAddress{CustomerProfileID: profiles[1].ID, RecipientName: "Old"}
	if err := db.Create(&deletedAddress).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&deletedAddress).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&profiles[3]).Error; err != nil {
		t.Fatal(err)
	}

	page1, total, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, SortBy: "displayName", SortDir: "asc"}})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(page1) != 2 || page1[0].DisplayName != "Alpha" || page1[1].DisplayName != "Middle" {
		t.Fatalf("unexpected first page: total=%d rows=%+v", total, page1)
	}
	page2, total2, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, Offset: 2, SortBy: "displayName", SortDir: "asc"}})
	if err != nil {
		t.Fatal(err)
	}
	if total2 != 3 || len(page2) != 1 || page2[0].ID == page1[0].ID || page2[0].ID == page1[1].ID {
		t.Fatalf("pages overlap or total changed: %+v", page2)
	}
	out, total3, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, Offset: 99}})
	if err != nil || total3 != 3 || len(out) != 0 {
		t.Fatalf("out-of-range page: total=%d rows=%d err=%v", total3, len(out), err)
	}

	filtered, _, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50}, Keyword: "NEEDLE", Platform: "shop", MissingAddressOnly: true})
	if err != nil || len(filtered) != 1 || filtered[0].ID != profiles[1].ID {
		t.Fatalf("customer filters failed: %+v, %v", filtered, err)
	}
	children, err := repo.ListCustomerIdentitiesByProfileIDs(ctx, []uint{profiles[0].ID, profiles[1].ID})
	if err != nil || len(children) != 2 {
		t.Fatalf("bulk identities: %d, %v", len(children), err)
	}
	addresses, err := repo.ListCustomerAddressesByProfileIDs(ctx, []uint{profiles[0].ID, profiles[1].ID})
	if err != nil || len(addresses) != 1 {
		t.Fatalf("bulk addresses should exclude soft deletes: %d, %v", len(addresses), err)
	}
	platforms, err := repo.ListCustomerIdentityPlatforms(ctx)
	if err != nil || len(platforms) != 2 {
		t.Fatalf("platforms: %+v, %v", platforms, err)
	}

	fallback, _, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50, SortBy: "DROP TABLE customer_profiles", SortDir: "desc"}})
	if err != nil || fallback[0].ID != profiles[2].ID {
		t.Fatalf("invalid sort should fall back to id desc: %+v, %v", fallback, err)
	}
	for _, field := range []string{"displayName", "profileType", "createdAt"} {
		if _, _, err := repo.ListCustomerProfilesPage(ctx, domain.CustomerProfilePageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1, SortBy: field}}); err != nil {
			t.Fatalf("valid sort %s: %v", field, err)
		}
	}
}

func TestListPaginationRepository_Products(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()
	products := []persistence.ProductMaster{
		{Name: "Zulu", SupplierPlatform: "B", FactorySKU: "SKU-2", SupplierProductRef: "REF-2", ProductKind: "gift"},
		{Name: "Alpha", SupplierPlatform: "A", FactorySKU: "SKU-1", SupplierProductRef: "REF-1", ProductKind: "book", Archived: true},
		{Name: "Middle", SupplierPlatform: "A", FactorySKU: "SKU-3", SupplierProductRef: "REF-3", ProductKind: "gift"},
		{Name: "Deleted", SupplierPlatform: "D", FactorySKU: "SKU-4", ProductKind: "gift"},
	}
	for i := range products {
		if err := db.Create(&products[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Delete(&products[3]).Error; err != nil {
		t.Fatal(err)
	}
	page, total, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, SortBy: "name"}})
	if err != nil || total != 3 || len(page) != 2 || page[0].Name != "Alpha" {
		t.Fatalf("product page: total=%d rows=%+v err=%v", total, page, err)
	}
	page2, total2, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, Offset: 2, SortBy: "name"}})
	if err != nil || total2 != 3 || len(page2) != 1 {
		t.Fatalf("product page 2: total=%d rows=%+v err=%v", total2, page2, err)
	}
	filtered, total3, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50}, Keyword: "sku-1", ProductKinds: []string{"book"}, ArchivedOnly: true})
	if err != nil || total3 != 1 || len(filtered) != 1 || filtered[0].ID != products[1].ID {
		t.Fatalf("product filters: %+v total=%d err=%v", filtered, total3, err)
	}
	out, total4, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 2, Offset: 99}})
	if err != nil || total4 != 3 || len(out) != 0 {
		t.Fatalf("product out of range: %d %d %v", total4, len(out), err)
	}
	fallback, _, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50, SortBy: "invalid"}})
	if err != nil || fallback[0].ID != products[0].ID {
		t.Fatalf("product fallback: %+v %v", fallback, err)
	}
	for _, field := range []string{"name", "supplierPlatform", "factorySku", "supplierProductRef", "productKind", "archived"} {
		if _, _, err := repo.ListProductMastersPage(ctx, domain.ProductMasterPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1, SortBy: field}}); err != nil {
			t.Fatalf("valid sort %s: %v", field, err)
		}
	}
}

func TestListPaginationRepository_DemandInbox(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()
	profileID := uint(7)
	docs := []persistence.DemandDocument{
		{Kind: "membership", CaptureMode: "api", SourceChannel: "B", SourceDocumentNo: "Z", IntegrationProfileID: &profileID},
		{Kind: "shop", CaptureMode: "manual", SourceChannel: "A", SourceDocumentNo: "A"},
		{Kind: "membership", CaptureMode: "api", SourceChannel: "C", SourceDocumentNo: "M", IntegrationProfileID: &profileID},
		{Kind: "deleted", CaptureMode: "api"},
	}
	for i := range docs {
		if err := db.Create(&docs[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	assignments := []persistence.WaveDemandAssignment{{WaveID: 10, DemandDocumentID: docs[0].ID}, {WaveID: 20, DemandDocumentID: docs[2].ID}}
	for i := range assignments {
		if err := db.Create(&assignments[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	deletedAssignment := persistence.WaveDemandAssignment{WaveID: 10, DemandDocumentID: docs[1].ID}
	if err := db.Create(&deletedAssignment).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&deletedAssignment).Error; err != nil {
		t.Fatal(err)
	}
	lines := []persistence.DemandLine{{DemandDocumentID: docs[0].ID, LineType: "gift"}, {DemandDocumentID: docs[2].ID, LineType: "gift"}}
	for i := range lines {
		if err := db.Create(&lines[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Delete(&docs[3]).Error; err != nil {
		t.Fatal(err)
	}

	page, total, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1}})
	if err != nil || total != 3 || len(page) != 1 {
		t.Fatalf("demand page: total=%d rows=%+v err=%v", total, page, err)
	}
	page2, total2, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1, Offset: 1}})
	if err != nil || total2 != 3 || len(page2) != 1 || page2[0].ID == page[0].ID {
		t.Fatalf("demand page 2: %+v %v", page2, err)
	}
	waveID := uint(10)
	filtered, total3, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50}, Assignment: "assigned", DemandKind: "membership", IntegrationProfileID: &profileID, WaveID: &waveID})
	if err != nil || total3 != 1 || len(filtered) != 1 || filtered[0].ID != docs[0].ID {
		t.Fatalf("demand filters: %+v total=%d err=%v", filtered, total3, err)
	}
	unassigned, _, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50}, Assignment: "unassigned"})
	if err != nil || len(unassigned) != 1 || unassigned[0].ID != docs[1].ID {
		t.Fatalf("soft-deleted assignment counted: %+v %v", unassigned, err)
	}
	bulkAssignments, err := repo.ListDemandAssignmentsByDocumentIDs(ctx, []uint{docs[0].ID, docs[1].ID, docs[2].ID})
	if err != nil || len(bulkAssignments) != 2 {
		t.Fatalf("bulk assignments: %d %v", len(bulkAssignments), err)
	}
	bulkLines, err := repo.ListDemandLinesByDocumentIDs(ctx, []uint{docs[0].ID, docs[2].ID})
	if err != nil || len(bulkLines) != 2 {
		t.Fatalf("bulk lines: %d %v", len(bulkLines), err)
	}
	out, total4, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1, Offset: 99}})
	if err != nil || total4 != 3 || len(out) != 0 {
		t.Fatalf("demand out of range: %d %d %v", total4, len(out), err)
	}
	fallback, _, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 50, SortBy: "invalid"}})
	if err != nil || fallback[0].ID != docs[0].ID {
		t.Fatalf("demand fallback: %+v %v", fallback, err)
	}
	for _, field := range []string{"kind", "captureMode", "sourceChannel", "sourceDocumentNo", "createdAt"} {
		if _, _, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{ListPageQuery: domain.ListPageQuery{Limit: 1, SortBy: field}}); err != nil {
			t.Fatalf("valid sort %s: %v", field, err)
		}
	}
}

func TestListPaginationRepository_Shipments(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()
	orders := []persistence.SupplierOrder{{WaveID: 1}, {WaveID: 2}, {WaveID: 1}}
	for i := range orders {
		if err := db.Create(&orders[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	shipments := []persistence.Shipment{
		{SupplierOrderID: orders[0].ID, ShipmentNo: "Z", SupplierPlatform: "B", ExternalShipmentNo: "E2", CarrierName: "Zulu", TrackingNo: "T2", Status: "shipped"},
		{SupplierOrderID: orders[0].ID, ShipmentNo: "A", SupplierPlatform: "A", ExternalShipmentNo: "E1", CarrierName: "Alpha", TrackingNo: "T1", Status: "pending"},
		{SupplierOrderID: orders[1].ID, ShipmentNo: "Other"},
		{SupplierOrderID: orders[2].ID, ShipmentNo: "Deleted order"},
		{SupplierOrderID: orders[0].ID, ShipmentNo: "Deleted shipment"},
	}
	for i := range shipments {
		if err := db.Create(&shipments[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Delete(&orders[2]).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&shipments[4]).Error; err != nil {
		t.Fatal(err)
	}
	lines := []persistence.ShipmentLine{{ShipmentID: shipments[0].ID, Quantity: 1}, {ShipmentID: shipments[1].ID, Quantity: 2}}
	for i := range lines {
		if err := db.Create(&lines[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	page, total, err := repo.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{WaveID: 1, ListPageQuery: domain.ListPageQuery{Limit: 1, SortBy: "shipmentNo"}})
	if err != nil || total != 2 || len(page) != 1 || page[0].ShipmentNo != "A" {
		t.Fatalf("shipment page: total=%d rows=%+v err=%v", total, page, err)
	}
	page2, total2, err := repo.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{WaveID: 1, ListPageQuery: domain.ListPageQuery{Limit: 1, Offset: 1, SortBy: "shipmentNo"}})
	if err != nil || total2 != 2 || len(page2) != 1 || page2[0].ID == page[0].ID {
		t.Fatalf("shipment page 2: %+v %v", page2, err)
	}
	out, total3, err := repo.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{WaveID: 1, ListPageQuery: domain.ListPageQuery{Limit: 1, Offset: 99}})
	if err != nil || total3 != 2 || len(out) != 0 {
		t.Fatalf("shipment out of range: %d %d %v", total3, len(out), err)
	}
	bulk, err := repo.ListShipmentLinesByShipmentIDs(ctx, []uint{shipments[0].ID, shipments[1].ID})
	if err != nil || len(bulk) != 2 {
		t.Fatalf("bulk shipment lines: %d %v", len(bulk), err)
	}
	fallback, _, err := repo.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{WaveID: 1, ListPageQuery: domain.ListPageQuery{Limit: 50, SortBy: "invalid"}})
	if err != nil || fallback[0].ID != shipments[0].ID {
		t.Fatalf("shipment fallback: %+v %v", fallback, err)
	}
	for _, field := range []string{"shipmentNo", "supplierPlatform", "externalShipmentNo", "carrier", "trackingNo", "status", "shippedAt"} {
		if _, _, err := repo.ListShipmentsPage(ctx, domain.ShipmentByWavePageQuery{WaveID: 1, ListPageQuery: domain.ListPageQuery{Limit: 1, SortBy: field}}); err != nil {
			t.Fatalf("valid sort %s: %v", field, err)
		}
	}
}

func TestListPaginationRepository_DemandDocumentsRoutingDispositionFilter(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()

	docA := persistence.DemandDocument{Kind: "membership_entitlement", SourceDocumentNo: "A"}
	docB := persistence.DemandDocument{Kind: "retail_order", SourceDocumentNo: "B"}
	if err := db.Create(&docA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&docB).Error; err != nil {
		t.Fatal(err)
	}

	lines := []persistence.DemandLine{
		{DemandDocumentID: docA.ID, RoutingDisposition: "pending_intake"},
		{DemandDocumentID: docA.ID, RoutingDisposition: "accepted"},
		{DemandDocumentID: docB.ID, RoutingDisposition: "accepted"},
	}
	for i := range lines {
		if err := db.Create(&lines[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	docs, total, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		RoutingDispositions: []string{"pending_intake"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(docs) != 1 || docs[0].ID != docA.ID {
		t.Fatalf("want only docA (has pending_intake line), got total=%d docs=%v", total, docs)
	}

	docs, total, err = repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		DemandKinds: []string{"retail_order"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(docs) != 1 || docs[0].ID != docB.ID {
		t.Fatalf("want only docB (retail_order), got total=%d docs=%v", total, docs)
	}
}
