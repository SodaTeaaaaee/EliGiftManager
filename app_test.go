package main

import (
	"os"
	"path/filepath"
	"testing"

	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
)

func TestBuildMemberItemsAggregatesDispatchCounts(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "app-test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	memberWithDispatches := model.Member{Platform: "douyin", PlatformUID: "uid-1", ExtraData: "{}"}
	memberWithoutDispatches := model.Member{Platform: "kuaishou", PlatformUID: "uid-2", ExtraData: "{}"}
	product := model.Product{Platform: "douyin", Factory: "factory-a", FactorySKU: "sku-1", Name: "gift-a", ExtraData: "{}"}
	wave := model.Wave{WaveNo: "TASK-TEST-001", Name: "wave-a", Status: "draft"}
	for _, record := range []any{&memberWithDispatches, &memberWithoutDispatches, &product, &wave} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to seed test record: %v", err)
		}
	}
	if err := db.Create(&model.MemberNickname{MemberID: memberWithDispatches.ID, Nickname: "latest-nick"}).Error; err != nil {
		t.Fatalf("failed to seed nickname: %v", err)
	}
	defaultAddress := model.MemberAddress{MemberID: memberWithDispatches.ID, RecipientName: "Alice", Phone: "13800000000", Address: "Shanghai", IsDefault: true}
	deletedAddress := model.MemberAddress{MemberID: memberWithDispatches.ID, RecipientName: "Old", Phone: "13900000000", Address: "Old address", IsDeleted: true}
	for _, address := range []model.MemberAddress{defaultAddress, deletedAddress} {
		address := address
		if err := db.Create(&address).Error; err != nil {
			t.Fatalf("failed to seed address: %v", err)
		}
	}
	for _, record := range []model.DispatchRecord{
		{WaveID: wave.ID, MemberID: memberWithDispatches.ID, ProductID: product.ID, Quantity: 1, Status: model.DispatchStatusPending},
		{WaveID: wave.ID, MemberID: memberWithDispatches.ID, ProductID: product.ID, Quantity: 2, Status: model.DispatchStatusPending},
	} {
		record := record
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("failed to seed dispatch record: %v", err)
		}
	}

	var members []model.Member
	if err := db.
		Preload("Nicknames").
		Preload("Addresses").
		Order("id ASC").
		Find(&members).Error; err != nil {
		t.Fatalf("failed to query members: %v", err)
	}

	items, err := buildMemberItems(db, members)
	if err != nil {
		t.Fatalf("buildMemberItems returned unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0]
	if first.ID != memberWithDispatches.ID {
		t.Fatalf("expected first item member id %d, got %d", memberWithDispatches.ID, first.ID)
	}
	if first.DispatchCount != 2 {
		t.Fatalf("expected dispatch count 2, got %d", first.DispatchCount)
	}
	if first.ActiveAddressCount != 1 {
		t.Fatalf("expected active address count 1, got %d", first.ActiveAddressCount)
	}
	if first.LatestRecipient != "Alice" || first.LatestPhone != "13800000000" || first.LatestAddress != "Shanghai" {
		t.Fatalf("unexpected latest address payload: %+v", first)
	}
	if first.LatestNickname != "latest-nick" {
		t.Fatalf("expected latest nickname latest-nick, got %q", first.LatestNickname)
	}

	second := items[1]
	if second.ID != memberWithoutDispatches.ID {
		t.Fatalf("expected second item member id %d, got %d", memberWithoutDispatches.ID, second.ID)
	}
	if second.DispatchCount != 0 {
		t.Fatalf("expected dispatch count 0, got %d", second.DispatchCount)
	}
	if second.ActiveAddressCount != 0 {
		t.Fatalf("expected active address count 0, got %d", second.ActiveAddressCount)
	}
}

func TestBuildProductItemsAggregatesDispatchStats(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "app-test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	member := model.Member{Platform: "douyin", PlatformUID: "uid-1", ExtraData: "{}"}
	wave := model.Wave{WaveNo: "TASK-TEST-001", Name: "wave-a", Status: "draft"}
	productWithDispatches := model.Product{Platform: "douyin", Factory: "factory-a", FactorySKU: "sku-1", Name: "gift-a", ExtraData: "{}"}
	productWithoutDispatches := model.Product{Platform: "douyin", Factory: "factory-b", FactorySKU: "sku-2", Name: "gift-b", ExtraData: "{}"}
	for _, record := range []any{&member, &wave, &productWithDispatches, &productWithoutDispatches} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to seed test record: %v", err)
		}
	}
	for _, record := range []model.DispatchRecord{
		{WaveID: wave.ID, MemberID: member.ID, ProductID: productWithDispatches.ID, Quantity: 2, Status: model.DispatchStatusPending},
		{WaveID: wave.ID, MemberID: member.ID, ProductID: productWithDispatches.ID, Quantity: 3, Status: model.DispatchStatusPending},
	} {
		record := record
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("failed to seed dispatch record: %v", err)
		}
	}

	var products []model.Product
	if err := db.Order("id ASC").Find(&products).Error; err != nil {
		t.Fatalf("failed to query products: %v", err)
	}

	items, err := buildProductItems(db, products)
	if err != nil {
		t.Fatalf("buildProductItems returned unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0]
	if first.ID != productWithDispatches.ID {
		t.Fatalf("expected first item product id %d, got %d", productWithDispatches.ID, first.ID)
	}
	if first.DispatchCount != 2 {
		t.Fatalf("expected dispatch count 2, got %d", first.DispatchCount)
	}
	if first.TotalQuantity != 5 {
		t.Fatalf("expected total quantity 5, got %d", first.TotalQuantity)
	}

	second := items[1]
	if second.ID != productWithoutDispatches.ID {
		t.Fatalf("expected second item product id %d, got %d", productWithoutDispatches.ID, second.ID)
	}
	if second.DispatchCount != 0 {
		t.Fatalf("expected dispatch count 0, got %d", second.DispatchCount)
	}
	if second.TotalQuantity != 0 {
		t.Fatalf("expected total quantity 0, got %d", second.TotalQuantity)
	}
}

func TestValidateDatabaseFile(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "app-test.db")
	if _, err := database.InitDB(dbPath); err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	if err := validateDatabaseFile(dbPath); err != nil {
		t.Fatalf("validateDatabaseFile returned unexpected error: %v", err)
	}
}

func TestValidateDatabaseFileRejectsInvalidSQLiteFile(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "invalid.db")
	if err := os.WriteFile(dbPath, []byte("not-a-sqlite-database"), 0o644); err != nil {
		t.Fatalf("failed to write invalid database file: %v", err)
	}
	if err := validateDatabaseFile(dbPath); err == nil {
		t.Fatal("expected validateDatabaseFile to reject invalid sqlite file")
	}
}
