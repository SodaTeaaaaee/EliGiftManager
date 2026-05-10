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
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

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
	product2 := model.Product{Platform: "douyin", Factory: "f2", FactorySKU: "sku-2", Name: "gift-b", ExtraData: "{}"}
	if err := db.Create(&product2).Error; err != nil {
		t.Fatalf("failed to seed product2: %v", err)
	}
	for _, record := range []model.DispatchRecord{
		{WaveID: wave.ID, MemberID: memberWithDispatches.ID, ProductID: product.ID, Quantity: 1, Status: model.DispatchStatusPending},
		{WaveID: wave.ID, MemberID: memberWithDispatches.ID, ProductID: product2.ID, Quantity: 2, Status: model.DispatchStatusPending},
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
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	member := model.Member{Platform: "douyin", PlatformUID: "uid-1", ExtraData: "{}"}
	wave := model.Wave{WaveNo: "TASK-TEST-001", Name: "wave-a", Status: "draft"}
	productWithDispatches := model.Product{Platform: "douyin", Factory: "factory-a", FactorySKU: "sku-1", Name: "gift-a", ExtraData: "{}"}
	productWithoutDispatches := model.Product{Platform: "douyin", Factory: "factory-b", FactorySKU: "sku-2", Name: "gift-b", ExtraData: "{}"}
	for _, record := range []any{&member, &wave, &productWithDispatches, &productWithoutDispatches} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("failed to seed test record: %v", err)
		}
	}
	member2 := model.Member{Platform: "douyin", PlatformUID: "uid-2", ExtraData: "{}"}
	if err := db.Create(&member2).Error; err != nil {
		t.Fatalf("failed to seed member2: %v", err)
	}
	for _, record := range []model.DispatchRecord{
		{WaveID: wave.ID, MemberID: member.ID, ProductID: productWithDispatches.ID, Quantity: 2, Status: model.DispatchStatusPending},
		{WaveID: wave.ID, MemberID: member2.ID, ProductID: productWithDispatches.ID, Quantity: 3, Status: model.DispatchStatusPending},
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
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
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

func TestRestoreDatabaseFromSourceReopensOriginalDBWhenRenameFailsAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "active.db")
	sourcePath := filepath.Join(tmpDir, "source.db")

	activeDB, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("init active db failed: %v", err)
	}
	sourceDB, err := database.InitDB(sourcePath)
	if err != nil {
		t.Fatalf("init source db failed: %v", err)
	}
	activeSQL, err := activeDB.DB()
	if err != nil {
		t.Fatalf("get active sql.DB failed: %v", err)
	}
	sourceSQL, err := sourceDB.DB()
	if err != nil {
		t.Fatalf("get source sql.DB failed: %v", err)
	}
	t.Cleanup(func() {
		_ = activeSQL.Close()
		_ = sourceSQL.Close()
		database.SetDefaultDB(nil)
	})

	if err := activeDB.Create(&model.Member{Platform: "douyin", PlatformUID: "active", ExtraData: "{}"}).Error; err != nil {
		t.Fatalf("seed active db failed: %v", err)
	}
	if err := sourceDB.Create(&model.Member{Platform: "douyin", PlatformUID: "source", ExtraData: "{}"}).Error; err != nil {
		t.Fatalf("seed source db failed: %v", err)
	}
	database.SetDefaultDB(activeDB)

	// Force rename(dbPath -> rollbackPath) to fail after the current connection is closed.
	if err := os.Mkdir(filepath.Join(tmpDir, "active.db.rollback"), 0o755); err != nil {
		t.Fatalf("create rollback blocker dir failed: %v", err)
	}

	err = restoreDatabaseFromSource(dbPath, sourcePath)
	if err == nil {
		t.Fatal("expected restoreDatabaseFromSource to fail when rollback path blocks rename")
	}

	currentDB := database.GetDB()
	if currentDB == nil {
		t.Fatal("expected defaultDB to be re-opened after restore failure")
	}
	if err := currentDB.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("expected reopened defaultDB to be usable, got: %v", err)
	}

	var count int64
	if err := currentDB.Model(&model.Member{}).Where("platform_uid = ?", "active").Count(&count).Error; err != nil {
		t.Fatalf("query reopened original DB failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected reopened DB to preserve original data, got count=%d", count)
	}
}

func TestReconcileWaveInvalidatesExportedStatusAfterAllocationChange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reconcile-export-invalidates.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	database.SetDefaultDB(db)
	t.Cleanup(func() { database.SetDefaultDB(nil) })

	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-1", ExtraData: "{}"}
	waveIDRef := uint(0)
	product := model.Product{Platform: "BILIBILI", Factory: "f", FactorySKU: "sku-1", Name: "gift", ExtraData: "{}"}
	wave := model.Wave{WaveNo: "TASK-RECON-001", Name: "wave", Status: model.DispatchStatusExported}
	for _, record := range []any{&member, &wave} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	waveIDRef = wave.ID
	product.WaveID = &waveIDRef
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("seed product failed: %v", err)
	}
	wm := model.WaveMember{WaveID: wave.ID, MemberID: member.ID, Platform: member.Platform, PlatformUID: member.PlatformUID, GiftLevel: "舰长", LatestNickname: "nick"}
	if err := db.Create(&wm).Error; err != nil {
		t.Fatalf("create wave member failed: %v", err)
	}
	tag := model.ProductTag{ProductID: product.ID, Platform: member.Platform, TagName: "舰长", TagType: "level", Quantity: 1}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create product tag failed: %v", err)
	}
	addr := model.MemberAddress{MemberID: member.ID, RecipientName: "Alice", Phone: "13800000000", Address: "Shanghai"}
	if err := db.Create(&addr).Error; err != nil {
		t.Fatalf("create address failed: %v", err)
	}
	dr := model.DispatchRecord{WaveID: wave.ID, MemberID: member.ID, ProductID: product.ID, MemberAddressID: &addr.ID, Quantity: 5, Status: model.DispatchStatusExported}
	if err := db.Create(&dr).Error; err != nil {
		t.Fatalf("create dispatch record failed: %v", err)
	}

	var wc WaveController
	allocated, err := wc.ReconcileWave(wave.ID)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}
	if allocated != 1 {
		t.Fatalf("expected allocated count 1, got %d", allocated)
	}

	var updatedRecord model.DispatchRecord
	if err := db.First(&updatedRecord, dr.ID).Error; err != nil {
		t.Fatalf("reload dispatch record failed: %v", err)
	}
	if updatedRecord.Status != model.DispatchStatusPending {
		t.Fatalf("expected dispatch status to fall back to pending after allocation change, got %q", updatedRecord.Status)
	}
	if updatedRecord.Quantity != 1 {
		t.Fatalf("expected reconciled quantity 1, got %d", updatedRecord.Quantity)
	}

	var updatedWave model.Wave
	if err := db.First(&updatedWave, wave.ID).Error; err != nil {
		t.Fatalf("reload wave failed: %v", err)
	}
	if updatedWave.Status != model.DispatchStatusPending {
		t.Fatalf("expected wave status to fall back to pending after allocation change, got %q", updatedWave.Status)
	}
}

func TestRemoveMemberFromWaveRecomputesWaveStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "remove-member-status.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	database.SetDefaultDB(db)
	t.Cleanup(func() { database.SetDefaultDB(nil) })

	wave := model.Wave{WaveNo: "TASK-RM-001", Name: "wave", Status: model.DispatchStatusPendingAddress}
	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-rm-1", ExtraData: "{}"}
	waveIDRef := uint(0)
	product := model.Product{Platform: "BILIBILI", Factory: "f", FactorySKU: "sku-rm-1", Name: "gift", ExtraData: "{}"}
	for _, record := range []any{&wave, &member} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	waveIDRef = wave.ID
	product.WaveID = &waveIDRef
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("seed product failed: %v", err)
	}
	wm := model.WaveMember{WaveID: wave.ID, MemberID: member.ID, Platform: member.Platform, PlatformUID: member.PlatformUID, GiftLevel: "舰长", LatestNickname: "nick"}
	if err := db.Create(&wm).Error; err != nil {
		t.Fatalf("create wave member failed: %v", err)
	}
	dr := model.DispatchRecord{WaveID: wave.ID, MemberID: member.ID, ProductID: product.ID, Quantity: 1, Status: model.DispatchStatusPendingAddress}
	if err := db.Create(&dr).Error; err != nil {
		t.Fatalf("create dispatch record failed: %v", err)
	}

	var mc MemberController
	if err := mc.RemoveMemberFromWave(wave.ID, wm.ID); err != nil {
		t.Fatalf("RemoveMemberFromWave failed: %v", err)
	}

	var remaining int64
	if err := db.Model(&model.DispatchRecord{}).Where("wave_id = ?", wave.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count dispatch records failed: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected 0 remaining dispatch records, got %d", remaining)
	}

	var updatedWave model.Wave
	if err := db.First(&updatedWave, wave.ID).Error; err != nil {
		t.Fatalf("reload wave failed: %v", err)
	}
	if updatedWave.Status != "draft" {
		t.Fatalf("expected wave status to recompute to draft after removing last member, got %q", updatedWave.Status)
	}
}
