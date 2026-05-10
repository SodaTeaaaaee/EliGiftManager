package service

import (
	"os"
	"path/filepath"
	"testing"

	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------------------------------------------------------------------
// Scenario 1: duplicate ProductMaster dedup + reference merge
//
// Tests the dedup SQL from autoMigrateTables step (c):
//   - Two product_masters share (platform, factory_sku)
//   - Two products each point to a different master
//   - After dedup: single canonical master, both products point to it
// ---------------------------------------------------------------------------

func TestProductMasterDedupAndReferenceMerge(t *testing.T) {
	db := newServiceTestDB(t)

	// Drop the unique index so we can insert duplicates.
	db.Exec("DROP INDEX IF EXISTS idx_product_master_platform_sku")

	// Insert two duplicate product_masters with same (platform, factory_sku).
	if err := db.Exec(
		`INSERT INTO product_masters (id, platform, factory, factory_sku, name, cover_image, extra_data, created_at, updated_at)
		 VALUES (1, 'taobao', 'factory-A', 'sku-X', 'Gift Alpha', '', '{}', datetime('now'), datetime('now'))`,
	).Error; err != nil {
		t.Fatalf("insert master 1 failed: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO product_masters (id, platform, factory, factory_sku, name, cover_image, extra_data, created_at, updated_at)
		 VALUES (2, 'taobao', 'factory-A', 'sku-X', 'Gift Alpha Dup', '', '{}', datetime('now'), datetime('now'))`,
	).Error; err != nil {
		t.Fatalf("insert master 2 failed: %v", err)
	}

	// Insert two products, each pointing to a different master.
	if err := db.Exec(
		`INSERT INTO products (id, platform, factory, factory_sku, name, product_master_id, extra_data, created_at, updated_at)
		 VALUES (1, 'taobao', 'factory-A', 'sku-X', 'Prod-1', 1, '{}', datetime('now'), datetime('now'))`,
	).Error; err != nil {
		t.Fatalf("insert product 1 failed: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO products (id, platform, factory, factory_sku, name, product_master_id, extra_data, created_at, updated_at)
		 VALUES (2, 'taobao', 'factory-A', 'sku-X', 'Prod-2', 2, '{}', datetime('now'), datetime('now'))`,
	).Error; err != nil {
		t.Fatalf("insert product 2 failed: %v", err)
	}

	// --- Execute the production dedup SQL from autoMigrateTables step (c) ---

	// Step 1: re-point products.product_master_id to canonical.
	if err := db.Exec(`
		UPDATE products SET product_master_id = (
			SELECT MIN(pm2.id) FROM product_masters pm2
			WHERE pm2.platform  = (SELECT pm3.platform   FROM product_masters pm3 WHERE pm3.id = products.product_master_id)
			  AND pm2.factory_sku = (SELECT pm3.factory_sku FROM product_masters pm3 WHERE pm3.id = products.product_master_id)
		) WHERE product_master_id IS NOT NULL
	`).Error; err != nil {
		t.Fatalf("re-point products to canonical master failed: %v", err)
	}

	// Step 2: re-point product_master_images (trivial in this test but exercised).
	if err := db.Exec(`
		UPDATE product_master_images SET product_master_id = (
			SELECT MIN(pm2.id) FROM product_masters pm2
			WHERE pm2.platform  = (SELECT pm3.platform   FROM product_masters pm3 WHERE pm3.id = product_master_images.product_master_id)
			  AND pm2.factory_sku = (SELECT pm3.factory_sku FROM product_masters pm3 WHERE pm3.id = product_master_images.product_master_id)
		)
	`).Error; err != nil {
		t.Fatalf("re-point product_master_images failed: %v", err)
	}

	// Step 3: delete non-canonical product_masters.
	if err := db.Exec(`
		DELETE FROM product_masters WHERE id NOT IN (
			SELECT MIN(id) FROM product_masters GROUP BY platform, factory_sku
		)
	`).Error; err != nil {
		t.Fatalf("delete duplicate product_masters failed: %v", err)
	}

	// --- Verify ---

	// Only one product_master remains.
	var masterCount int64
	db.Model(&model.ProductMaster{}).Where("platform = ? AND factory_sku = ?", "taobao", "sku-X").Count(&masterCount)
	if masterCount != 1 {
		t.Fatalf("expected 1 product_master after dedup, got %d", masterCount)
	}

	// Canonical master keeps the data from id=1 (smallest ID).
	var canonicalID uint
	db.Model(&model.ProductMaster{}).Where("platform = ? AND factory_sku = ?", "taobao", "sku-X").Select("id").Scan(&canonicalID)
	if canonicalID != 1 {
		t.Fatalf("expected canonical master id=1, got %d", canonicalID)
	}

	// Both products now point to the canonical master.
	var p1MasterID, p2MasterID uint
	db.Model(&model.Product{}).Where("id = 1").Select("product_master_id").Scan(&p1MasterID)
	db.Model(&model.Product{}).Where("id = 2").Select("product_master_id").Scan(&p2MasterID)
	if p1MasterID != canonicalID {
		t.Fatalf("product 1 should point to master %d, got %d", canonicalID, p1MasterID)
	}
	if p2MasterID != canonicalID {
		t.Fatalf("product 2 should point to master %d, got %d", canonicalID, p2MasterID)
	}

	// Non-canonical master (id=2) is deleted.
	var orphanCount int64
	db.Model(&model.ProductMaster{}).Where("id = 2").Count(&orphanCount)
	if orphanCount != 0 {
		t.Fatalf("expected orphan master id=2 to be deleted, found %d", orphanCount)
	}

	// Re-create unique index to confirm data is clean.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_product_master_platform_sku ON product_masters(platform, factory_sku)").Error; err != nil {
		t.Fatalf("re-create unique index after dedup failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: duplicate snapshot merge + references not lost
//
// Tests the product dedup SQL from autoMigrateTables step (d):
//   - Same wave, two products with same (wave_id, platform, factory_sku)
//   - DispatchRecord, ProductTag, ProductImage attached to each
//   - After dedup: single canonical product, all references survive
// ---------------------------------------------------------------------------

func TestProductSnapshotDedupKeepsAllReferences(t *testing.T) {
	db := newServiceTestDB(t)

	// Create a wave that the products belong to.
	wave := model.Wave{WaveNo: "TASK-20260101-001", Name: "DedupWave", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}

	// Create two members for the dispatch records (avoiding pre-clean conflict).
	member1 := model.Member{Platform: "douyin", PlatformUID: "uid-10", ExtraData: "{}"}
	member2 := model.Member{Platform: "douyin", PlatformUID: "uid-20", ExtraData: "{}"}
	if err := db.Create(&member1).Error; err != nil {
		t.Fatalf("create member1 failed: %v", err)
	}
	if err := db.Create(&member2).Error; err != nil {
		t.Fatalf("create member2 failed: %v", err)
	}

	// Drop the product unique index so we can insert duplicates.
	db.Exec("DROP INDEX IF EXISTS idx_product_wave_platform_sku")

	// Find free IDs for the two products.
	var prod1ID, prod2ID uint
	db.Raw("SELECT COALESCE(MAX(id), 0) + 1 FROM products").Scan(&prod1ID)
	prod2ID = prod1ID + 1

	if err := db.Exec(
		`INSERT INTO products (id, platform, factory, factory_sku, name, wave_id, extra_data, created_at, updated_at)
		 VALUES (?, 'douyin', 'f-A', 'sku-Y', 'Cup', ?, '{}', datetime('now'), datetime('now'))`,
		prod1ID, wave.ID,
	).Error; err != nil {
		t.Fatalf("insert product 1 failed: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO products (id, platform, factory, factory_sku, name, wave_id, extra_data, created_at, updated_at)
		 VALUES (?, 'douyin', 'f-A', 'sku-Y', 'Cup Dup', ?, '{}', datetime('now'), datetime('now'))`,
		prod2ID, wave.ID,
	).Error; err != nil {
		t.Fatalf("insert product 2 failed: %v", err)
	}

	// Attach DispatchRecord to product 1 (member 1).
	if err := db.Exec(
		`INSERT INTO dispatch_records (wave_id, member_id, product_id, quantity, status, created_at, updated_at)
		 VALUES (?, ?, ?, 1, 'pending', datetime('now'), datetime('now'))`,
		wave.ID, member1.ID, prod1ID,
	).Error; err != nil {
		t.Fatalf("insert dispatch_record 1 failed: %v", err)
	}

	// Attach DispatchRecord to product 2 (member 2 — different member, no conflict).
	if err := db.Exec(
		`INSERT INTO dispatch_records (wave_id, member_id, product_id, quantity, status, created_at, updated_at)
		 VALUES (?, ?, ?, 3, 'pending', datetime('now'), datetime('now'))`,
		wave.ID, member2.ID, prod2ID,
	).Error; err != nil {
		t.Fatalf("insert dispatch_record 2 failed: %v", err)
	}

	// Attach ProductTag to product 1.
	if err := db.Exec(
		`INSERT INTO product_tags (product_id, platform, tag_name, tag_type, quantity, created_at, updated_at)
		 VALUES (?, 'douyin', 'commander', 'level', 2, datetime('now'), datetime('now'))`,
		prod1ID,
	).Error; err != nil {
		t.Fatalf("insert product_tag 1 failed: %v", err)
	}

	// Attach ProductTag to product 2.
	if err := db.Exec(
		`INSERT INTO product_tags (product_id, platform, tag_name, tag_type, quantity, created_at, updated_at)
		 VALUES (?, 'douyin', 'captain', 'level', 1, datetime('now'), datetime('now'))`,
		prod2ID,
	).Error; err != nil {
		t.Fatalf("insert product_tag 2 failed: %v", err)
	}

	// Attach ProductImage to product 1 (no updated_at column in this model).
	if err := db.Exec(
		`INSERT INTO product_images (product_id, path, sort_order, created_at)
		 VALUES (?, 'ab/cup-front.png', 0, datetime('now'))`,
		prod1ID,
	).Error; err != nil {
		t.Fatalf("insert product_image 1 failed: %v", err)
	}

	// Attach ProductImage to product 2.
	if err := db.Exec(
		`INSERT INTO product_images (product_id, path, sort_order, created_at)
		 VALUES (?, 'cd/cup-back.png', 1, datetime('now'))`,
		prod2ID,
	).Error; err != nil {
		t.Fatalf("insert product_image 2 failed: %v", err)
	}

	// Count references before dedup.
	var preDR, preTags, preImgs int64
	db.Model(&model.DispatchRecord{}).Count(&preDR)
	db.Model(&model.ProductTag{}).Count(&preTags)
	db.Model(&model.ProductImage{}).Count(&preImgs)

	// --- Execute the production product dedup SQL from autoMigrateTables step (d) ---

	// Pre-clean: delete dispatch_records that would conflict after re-point.
	if err := db.Exec(`
		DELETE FROM dispatch_records WHERE id IN (
			SELECT dr1.id FROM dispatch_records dr1
			INNER JOIN products p_dup ON p_dup.id = dr1.product_id
			WHERE p_dup.wave_id IS NOT NULL
			  AND EXISTS (
			    SELECT 1 FROM products p_other
			    WHERE p_other.wave_id = p_dup.wave_id
			      AND p_other.platform = p_dup.platform
			      AND p_other.factory_sku = p_dup.factory_sku
			      AND p_other.id < p_dup.id
			  )
			  AND EXISTS (
			    SELECT 1 FROM dispatch_records dr2
			    WHERE dr2.wave_id = dr1.wave_id
			      AND dr2.member_id = dr1.member_id
			      AND dr2.product_id = (
			        SELECT MIN(p_min.id) FROM products p_min
			        WHERE p_min.wave_id = p_dup.wave_id
			          AND p_min.platform = p_dup.platform
			          AND p_min.factory_sku = p_dup.factory_sku
			      )
			  )
		)
	`).Error; err != nil {
		t.Fatalf("pre-clean conflicting dispatch_records failed: %v", err)
	}

	// Step 1: re-point dispatch_records.product_id to canonical.
	if err := db.Exec(`
		UPDATE dispatch_records SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = dispatch_records.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = dispatch_records.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = dispatch_records.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("re-point dispatch_records failed: %v", err)
	}

	// Step 2: re-point product_tags.product_id to canonical.
	if err := db.Exec(`
		UPDATE product_tags SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = product_tags.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = product_tags.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = product_tags.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("re-point product_tags failed: %v", err)
	}

	// Step 3: re-point product_images.product_id to canonical.
	if err := db.Exec(`
		UPDATE product_images SET product_id = (
			SELECT MIN(p2.id) FROM products p2
			WHERE p2.wave_id     = (SELECT p3.wave_id     FROM products p3 WHERE p3.id = product_images.product_id)
			  AND p2.platform    = (SELECT p3.platform    FROM products p3 WHERE p3.id = product_images.product_id)
			  AND p2.factory_sku = (SELECT p3.factory_sku FROM products p3 WHERE p3.id = product_images.product_id)
		)
		WHERE product_id IN (
			SELECT p1.id FROM products p1 WHERE p1.wave_id IS NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("re-point product_images failed: %v", err)
	}

	// Step 4: delete non-canonical products.
	if err := db.Exec(`
		DELETE FROM products WHERE id NOT IN (
			SELECT MIN(id) FROM products WHERE wave_id IS NOT NULL
			GROUP BY wave_id, platform, factory_sku
		) AND wave_id IS NOT NULL
	`).Error; err != nil {
		t.Fatalf("delete duplicate products failed: %v", err)
	}

	// --- Verify ---

	// Only one product snapshot remains in this wave.
	var productCount int64
	db.Model(&model.Product{}).Where("wave_id = ? AND platform = ? AND factory_sku = ?", wave.ID, "douyin", "sku-Y").Count(&productCount)
	if productCount != 1 {
		t.Fatalf("expected 1 product snapshot after dedup, got %d", productCount)
	}

	// The canonical product is the one with the smallest ID.
	var canonicalProductID uint
	db.Model(&model.Product{}).Where("wave_id = ? AND platform = ? AND factory_sku = ?", wave.ID, "douyin", "sku-Y").Select("id").Scan(&canonicalProductID)
	if canonicalProductID != prod1ID {
		t.Fatalf("expected canonical product id=%d, got %d", prod1ID, canonicalProductID)
	}

	// All references survive and point to canonical.

	// DispatchRecord for member1 (was on prod1) — should still exist.
	var drOnProd1 int64
	db.Model(&model.DispatchRecord{}).Where("product_id = ? AND member_id = ?", canonicalProductID, member1.ID).Count(&drOnProd1)
	if drOnProd1 != 1 {
		t.Fatalf("dispatch_record for member1 should exist on canonical product, found %d", drOnProd1)
	}

	// DispatchRecord for member2 (was on prod2) — should be re-pointed.
	var drOnProd2 int64
	db.Model(&model.DispatchRecord{}).Where("product_id = ? AND member_id = ?", canonicalProductID, member2.ID).Count(&drOnProd2)
	if drOnProd2 != 1 {
		t.Fatalf("dispatch_record for member2 should be re-pointed to canonical product, found %d", drOnProd2)
	}

	// ProductTag "commander" (was on prod1) — should still exist on canonical.
	var tagCommander int64
	db.Model(&model.ProductTag{}).Where("product_id = ? AND tag_name = 'commander'", canonicalProductID).Count(&tagCommander)
	if tagCommander != 1 {
		t.Fatalf("product_tag 'commander' should survive on canonical product, found %d", tagCommander)
	}

	// ProductTag "captain" (was on prod2) — should be re-pointed.
	var tagCaptain int64
	db.Model(&model.ProductTag{}).Where("product_id = ? AND tag_name = 'captain'", canonicalProductID).Count(&tagCaptain)
	if tagCaptain != 1 {
		t.Fatalf("product_tag 'captain' should be re-pointed to canonical product, found %d", tagCaptain)
	}

	// ProductImage "cup-front.png" (was on prod1) — should survive.
	var imgFront int64
	db.Model(&model.ProductImage{}).Where("product_id = ? AND path = 'ab/cup-front.png'", canonicalProductID).Count(&imgFront)
	if imgFront != 1 {
		t.Fatalf("product_image 'cup-front.png' should survive on canonical product, found %d", imgFront)
	}

	// ProductImage "cup-back.png" (was on prod2) — should be re-pointed.
	var imgBack int64
	db.Model(&model.ProductImage{}).Where("product_id = ? AND path = 'cd/cup-back.png'", canonicalProductID).Count(&imgBack)
	if imgBack != 1 {
		t.Fatalf("product_image 'cup-back.png' should be re-pointed to canonical product, found %d", imgBack)
	}

	// Total reference counts are preserved (no data lost).
	var totalDR, totalTags, totalImages int64
	db.Model(&model.DispatchRecord{}).Count(&totalDR)
	db.Model(&model.ProductTag{}).Count(&totalTags)
	db.Model(&model.ProductImage{}).Count(&totalImages)
	if totalDR != preDR {
		t.Fatalf("dispatch_record count changed: was %d, now %d", preDR, totalDR)
	}
	if totalTags != preTags {
		t.Fatalf("product_tag count changed: was %d, now %d", preTags, totalTags)
	}
	if totalImages != preImgs {
		t.Fatalf("product_image count changed: was %d, now %d", preImgs, totalImages)
	}

	// Re-create unique index to confirm data is clean.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_product_wave_platform_sku ON products(wave_id, platform, factory_sku)").Error; err != nil {
		t.Fatalf("re-create product unique index after dedup failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: DeleteWave does not orphan ProductMaster
// ---------------------------------------------------------------------------

func TestDeleteWavePreservesProductMaster(t *testing.T) {
	db := newServiceTestDB(t)

	// Create a ProductMaster.
	master := model.ProductMaster{
		Platform:   "kuaishou",
		Factory:    "f-B",
		FactorySKU: "sku-Z",
		Name:       "Mug",
		ExtraData:  "{}",
	}
	if err := db.Create(&master).Error; err != nil {
		t.Fatalf("create product_master failed: %v", err)
	}

	// Create a Wave.
	wave := model.Wave{WaveNo: "TASK-20260101-003", Name: "DeleteWaveTest", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}

	// Create a Product snapshot linked to this wave and master.
	product := model.Product{
		Platform:        "kuaishou",
		Factory:         "f-B",
		FactorySKU:      "sku-Z",
		Name:            "Mug",
		WaveID:          &wave.ID,
		ProductMasterID: &master.ID,
		ExtraData:       "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product snapshot failed: %v", err)
	}

	// Verify product exists in wave before delete.
	var preCount int64
	db.Model(&model.Product{}).Where("wave_id = ?", wave.ID).Count(&preCount)
	if preCount != 1 {
		t.Fatalf("expected 1 product in wave before delete, got %d", preCount)
	}

	// Simulate DeleteWave: cascade order per controller_wave.go.
	db.Where("wave_id = ?", wave.ID).Delete(&model.DispatchRecord{})
	db.Where("wave_id = ?", wave.ID).Delete(&model.WaveMember{})
	db.Where("wave_id = ?", wave.ID).Delete(&model.Product{})
	db.Delete(&model.Wave{}, wave.ID)

	// Verify products for this wave are deleted.
	var postProductCount int64
	db.Model(&model.Product{}).Where("wave_id = ?", wave.ID).Count(&postProductCount)
	if postProductCount != 0 {
		t.Fatalf("expected 0 products after DeleteWave, got %d", postProductCount)
	}

	// Verify ProductMaster STILL EXISTS.
	var masterCount int64
	db.Model(&model.ProductMaster{}).Where("id = ?", master.ID).Count(&masterCount)
	if masterCount != 1 {
		t.Fatalf("expected ProductMaster to survive DeleteWave, but count=%d", masterCount)
	}

	// Verify the master still has its original attributes.
	var reloaded model.ProductMaster
	if err := db.First(&reloaded, master.ID).Error; err != nil {
		t.Fatalf("reload ProductMaster failed: %v", err)
	}
	if reloaded.Platform != "kuaishou" || reloaded.FactorySKU != "sku-Z" {
		t.Fatalf("ProductMaster data corrupted after DeleteWave: %+v", reloaded)
	}
}

// ---------------------------------------------------------------------------
// Scenario 4: RouZao re-import — upsertProductsToWave returns only batch-scoped IDs
// ---------------------------------------------------------------------------

func TestUpsertProductsToWaveReturnsOnlyBatchIDs(t *testing.T) {
	db := newServiceTestDB(t)

	wave := model.Wave{WaveNo: "TASK-20260101-004", Name: "RouZaoScopeTest", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}

	// Replica of upsertProductsToWave logic.
	upsertBatch := func(products []model.Product) []uint {
		var ids []uint
		err := db.Transaction(func(tx *gorm.DB) error {
			for i := range products {
				if products[i].ExtraData == "" {
					products[i].ExtraData = "{}"
				}
				products[i].WaveID = &wave.ID

				master := model.ProductMaster{
					Platform:   products[i].Platform,
					Factory:    products[i].Factory,
					FactorySKU: products[i].FactorySKU,
					Name:       products[i].Name,
					ExtraData:  products[i].ExtraData,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "platform"}, {Name: "factory_sku"}},
					DoUpdates: clause.AssignmentColumns([]string{"name", "cover_image", "factory", "extra_data", "updated_at"}),
				}).Create(&master).Error; err != nil {
					return err
				}
				if master.ID == 0 {
					if err := tx.Where("platform = ? AND factory_sku = ?",
						master.Platform, master.FactorySKU).First(&master).Error; err != nil {
						return err
					}
				}

				products[i].ProductMasterID = &master.ID
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "wave_id"}, {Name: "platform"}, {Name: "factory_sku"}},
					DoUpdates: clause.AssignmentColumns([]string{"name", "cover_image", "factory", "extra_data", "product_master_id", "updated_at"}),
				}).Create(&products[i]).Error; err != nil {
					return err
				}
				if products[i].ID == 0 {
					if err := tx.Where("wave_id = ? AND platform = ? AND factory_sku = ?",
						wave.ID, products[i].Platform, products[i].FactorySKU).First(&products[i]).Error; err != nil {
						return err
					}
				}
				ids = append(ids, products[i].ID)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("upsert batch failed: %v", err)
		}
		return ids
	}

	// Batch 1: product A.
	batch1 := []model.Product{
		{Platform: "柔造", Factory: "factory-C", FactorySKU: "sku-A", Name: "Product A"},
	}
	batch1IDs := upsertBatch(batch1)

	// Batch 2: product B (different).
	batch2 := []model.Product{
		{Platform: "柔造", Factory: "factory-C", FactorySKU: "sku-B", Name: "Product B"},
	}
	batch2IDs := upsertBatch(batch2)

	if len(batch2IDs) != 1 {
		t.Fatalf("expected batch2 to return 1 ID, got %d", len(batch2IDs))
	}

	// batch2 must NOT include batch1's IDs.
	batch1Set := make(map[uint]bool)
	for _, id := range batch1IDs {
		batch1Set[id] = true
	}
	for _, id := range batch2IDs {
		if batch1Set[id] {
			t.Fatalf("batch2 returned product ID %d which belongs to batch1", id)
		}
	}

	// Re-import same product should return SAME ID (upsert, not insert).
	batch1Repeat := []model.Product{
		{Platform: "柔造", Factory: "factory-C", FactorySKU: "sku-A", Name: "Product A Updated"},
	}
	batch1RepeatIDs := upsertBatch(batch1Repeat)
	if len(batch1RepeatIDs) != 1 {
		t.Fatalf("expected re-import to return 1 ID, got %d", len(batch1RepeatIDs))
	}
	if batch1RepeatIDs[0] != batch1IDs[0] {
		t.Fatalf("re-import should return same ID (%d), got %d", batch1IDs[0], batch1RepeatIDs[0])
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: ProductMaster cover_image backfill
// ---------------------------------------------------------------------------

func TestProductMasterCoverImageBackfill(t *testing.T) {
	db := newServiceTestDB(t)

	// Create a ProductMaster with EMPTY cover_image.
	master := model.ProductMaster{
		Platform:   "bilibili",
		Factory:    "f-D",
		FactorySKU: "sku-Cover",
		Name:       "Coverless Gift",
		CoverImage: "",
		ExtraData:  "{}",
	}
	if err := db.Create(&master).Error; err != nil {
		t.Fatalf("create product_master failed: %v", err)
	}

	// Create a Product snapshot linked to the master, also empty cover.
	wave := model.Wave{WaveNo: "TASK-20260101-005", Name: "CoverTest", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}
	product := model.Product{
		Platform:        "bilibili",
		Factory:         "f-D",
		FactorySKU:      "sku-Cover",
		Name:            "Coverless Gift",
		WaveID:          &wave.ID,
		ProductMasterID: &master.ID,
		CoverImage:      "",
		ExtraData:       "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	// Simulate ProcessCoverImages: image matched, fill cover_image.
	coverPath := "ef/cover-image-hash.png"

	// Update Product cover_image.
	if err := db.Model(&product).Update("cover_image", coverPath).Error; err != nil {
		t.Fatalf("update product cover_image failed: %v", err)
	}

	// Update ProductMaster cover_image (only if empty — matches production code).
	result := db.Model(&model.ProductMaster{}).Where("id = ? AND cover_image = ''", master.ID).
		Update("cover_image", coverPath)
	if result.Error != nil {
		t.Fatalf("update master cover_image failed: %v", result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("expected 1 row affected when backfilling empty cover, got %d", result.RowsAffected)
	}

	// Verify ProductMaster.cover_image is now filled.
	var updatedMaster model.ProductMaster
	if err := db.First(&updatedMaster, master.ID).Error; err != nil {
		t.Fatalf("reload master failed: %v", err)
	}
	if updatedMaster.CoverImage == "" {
		t.Fatal("expected ProductMaster.cover_image to be backfilled, but it is still empty")
	}
	if updatedMaster.CoverImage != coverPath {
		t.Fatalf("expected cover_image %q, got %q", coverPath, updatedMaster.CoverImage)
	}

	// Idempotency: existing cover_image MUST NOT be overwritten.
	newCoverPath := "gh/newer-image.png"
	result2 := db.Model(&model.ProductMaster{}).Where("id = ? AND cover_image = ''", master.ID).
		Update("cover_image", newCoverPath)
	if result2.Error != nil {
		t.Fatalf("second cover update failed: %v", result2.Error)
	}
	if result2.RowsAffected != 0 {
		t.Fatalf("expected 0 rows affected (cover already set), got %d", result2.RowsAffected)
	}

	var finalMaster model.ProductMaster
	if err := db.First(&finalMaster, master.ID).Error; err != nil {
		t.Fatalf("reload master final failed: %v", err)
	}
	if finalMaster.CoverImage != coverPath {
		t.Fatalf("cover_image was overwritten from %q to %q", coverPath, finalMaster.CoverImage)
	}
}

// ---------------------------------------------------------------------------
// Scenario 6: global product catalog queries ProductMaster, not Product
// ---------------------------------------------------------------------------

func TestGlobalCatalogUsesProductMaster(t *testing.T) {
	db := newServiceTestDB(t)

	master := model.ProductMaster{
		Platform:   "taobao",
		Factory:    "f-E",
		FactorySKU: "sku-Global",
		Name:       "Global Product",
		CoverImage: "ab/global-cover.png",
		ExtraData:  `{"color":"red"}`,
	}
	if err := db.Create(&master).Error; err != nil {
		t.Fatalf("create product_master failed: %v", err)
	}

	masterImg := model.ProductMasterImage{
		ProductMasterID: master.ID,
		Path:            "cd/master-img-1.png",
		SortOrder:       0,
		SourceDir:       "主图",
	}
	if err := db.Create(&masterImg).Error; err != nil {
		t.Fatalf("create product_master_image failed: %v", err)
	}

	// Also create Product + ProductImage for contrast.
	wave := model.Wave{WaveNo: "TASK-20260101-006", Name: "CatalogTest", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}
	product := model.Product{
		Platform:        "taobao",
		Factory:         "f-E",
		FactorySKU:      "sku-Global",
		Name:            "Global Product",
		WaveID:          &wave.ID,
		ProductMasterID: &master.ID,
		ExtraData:       `{"color":"red"}`,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	productImg := model.ProductImage{
		ProductID: product.ID,
		Path:      "ef/product-only-img.png",
		SortOrder: 0,
	}
	if err := db.Create(&productImg).Error; err != nil {
		t.Fatalf("create product_image failed: %v", err)
	}

	// --- Simulate ListProductMasters ---
	var masters []model.ProductMaster
	if err := db.Model(&model.ProductMaster{}).Find(&masters).Error; err != nil {
		t.Fatalf("list product_masters failed: %v", err)
	}
	if len(masters) == 0 {
		t.Fatal("expected at least 1 ProductMaster, got 0")
	}
	foundMaster := masters[len(masters)-1] // ours is the last inserted
	if foundMaster.Platform != "taobao" || foundMaster.FactorySKU != "sku-Global" {
		t.Fatalf("ListProductMasters returned wrong data: %+v", foundMaster)
	}
	if foundMaster.CoverImage != "ab/global-cover.png" {
		t.Fatalf("expected cover_image from ProductMaster, got %q", foundMaster.CoverImage)
	}

	// --- Simulate GetProductMasterImages ---
	var masterImages []model.ProductMasterImage
	if err := db.Where("product_master_id = ?", master.ID).Order("sort_order ASC").Find(&masterImages).Error; err != nil {
		t.Fatalf("get product_master_images failed: %v", err)
	}
	if len(masterImages) != 1 {
		t.Fatalf("expected 1 ProductMasterImage, got %d", len(masterImages))
	}
	if masterImages[0].Path != "cd/master-img-1.png" {
		t.Fatalf("expected master image path, got %q", masterImages[0].Path)
	}

	// --- Sanity: GetProductImages is separate ---
	var productImages []model.ProductImage
	if err := db.Where("product_id = ?", product.ID).Find(&productImages).Error; err != nil {
		t.Fatalf("get product_images failed: %v", err)
	}
	if len(productImages) != 1 {
		t.Fatalf("expected 1 ProductImage, got %d", len(productImages))
	}
	if productImages[0].Path != "ef/product-only-img.png" {
		t.Fatalf("expected product image path, got %q", productImages[0].Path)
	}

	// Independent table counts.
	var masterImgCount, productImgCount int64
	db.Model(&model.ProductMasterImage{}).Count(&masterImgCount)
	db.Model(&model.ProductImage{}).Count(&productImgCount)
	if masterImgCount != 1 {
		t.Fatalf("ProductMasterImage count should be 1, got %d", masterImgCount)
	}
	if productImgCount != 1 {
		t.Fatalf("ProductImage count should be 1, got %d", productImgCount)
	}
}

// ---------------------------------------------------------------------------
// Scenario 7: restore failure rollback — InitDB failure leaves defaultDB usable
// ---------------------------------------------------------------------------

func TestRestoreFailureRollbackKeepsDefaultDB(t *testing.T) {
	// Cannot run in parallel — manipulates global defaultDB singleton.
	tmpDir := t.TempDir()
	goodPath := filepath.Join(tmpDir, "good.db")

	// Create a file to block directory creation, causing InitDB to fail.
	blockerPath := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blockerPath, []byte("block"), 0644); err != nil {
		t.Fatalf("create blocker file failed: %v", err)
	}
	badPath := filepath.Join(blockerPath, "sub", "bad.db") // blocker is a file, not a dir

	// Step 1: Create a valid database and set it as defaultDB.
	db1, err := database.InitDB(goodPath)
	if err != nil {
		t.Fatalf("InitDB good path failed: %v", err)
	}
	database.SetDefaultDB(db1)
	sqlDB1, _ := db1.DB()
	t.Cleanup(func() {
		sqlDB1.Close()
		database.SetDefaultDB(nil)
	})

	// Verify defaultDB is set and healthy.
	currentDB := database.GetDB()
	if currentDB == nil {
		t.Fatal("GetDB() returned nil after SetDefaultDB")
	}
	if err := currentDB.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("defaultDB ping failed before restore simulation: %v", err)
	}

	// Step 2: Simulate InitDB failure (path blocked by a file).
	_, initErr := database.InitDB(badPath)
	if initErr == nil {
		t.Fatal("expected InitDB to fail when directory creation is blocked, but it succeeded")
	}

	// Step 3: Verify defaultDB is STILL non-nil and usable.
	currentDB = database.GetDB()
	if currentDB == nil {
		t.Fatal("GetDB() returned nil after failed InitDB — defaultDB was lost")
	}
	if err := currentDB.Exec("SELECT 1").Error; err != nil {
		t.Fatalf("defaultDB ping failed after restore simulation: %v", err)
	}

	// Step 4: Verify we can still query the original database.
	var tableName string
	if err := currentDB.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'members'").Scan(&tableName).Error; err != nil {
		t.Fatalf("query original DB failed: %v", err)
	}
	if tableName != "members" {
		t.Fatalf("expected 'members' table, got %q", tableName)
	}

	// Step 5: Close and re-open the original DB to confirm it is not corrupted.
	sqlDB1.Close()
	db2, err := database.InitDB(goodPath)
	if err != nil {
		t.Fatalf("re-open original DB after failed restore failed: %v", err)
	}
	sqlDB2, _ := db2.DB()
	sqlDB2.Close()
}

// ---------------------------------------------------------------------------
// Integration: full wave lifecycle — create, import, delete, master survives
// ---------------------------------------------------------------------------

func TestFullWaveLifecyclePreservesProductMaster(t *testing.T) {
	db := newServiceTestDB(t)

	master := model.ProductMaster{
		Platform:   "douyin",
		Factory:    "f-F",
		FactorySKU: "sku-Lifecycle",
		Name:       "Lifecycle Product",
		ExtraData:  "{}",
	}
	if err := db.Create(&master).Error; err != nil {
		t.Fatalf("create master failed: %v", err)
	}

	wave := model.Wave{WaveNo: "TASK-20260101-007", Name: "LifecycleTest", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}

	product := model.Product{
		Platform:        "douyin",
		Factory:         "f-F",
		FactorySKU:      "sku-Lifecycle",
		Name:            "Lifecycle Product",
		WaveID:          &wave.ID,
		ProductMasterID: &master.ID,
		ExtraData:       "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product snapshot failed: %v", err)
	}

	// Verify product exists in wave.
	var prodInWave int64
	db.Model(&model.Product{}).Where("wave_id = ? AND id = ?", wave.ID, product.ID).Count(&prodInWave)
	if prodInWave != 1 {
		t.Fatalf("product should exist in wave, count=%d", prodInWave)
	}

	// Delete the wave.
	db.Where("wave_id = ?", wave.ID).Delete(&model.DispatchRecord{})
	db.Where("wave_id = ?", wave.ID).Delete(&model.WaveMember{})
	db.Where("wave_id = ?", wave.ID).Delete(&model.Product{})
	db.Delete(&model.Wave{}, wave.ID)

	// Product snapshot is gone.
	var prodAfterDelete int64
	db.Model(&model.Product{}).Where("id = ?", product.ID).Count(&prodAfterDelete)
	if prodAfterDelete != 0 {
		t.Fatalf("product snapshot should be deleted, count=%d", prodAfterDelete)
	}

	// ProductMaster survives.
	var masterAfterDelete model.ProductMaster
	if err := db.First(&masterAfterDelete, master.ID).Error; err != nil {
		t.Fatalf("ProductMaster should survive wave delete, but got: %v", err)
	}
	if masterAfterDelete.Platform != "douyin" || masterAfterDelete.FactorySKU != "sku-Lifecycle" {
		t.Fatalf("ProductMaster data corrupted: %+v", masterAfterDelete)
	}
}

// ---------------------------------------------------------------------------
// Edge case: idempotent InitDB — running migration twice on same DB is safe
// ---------------------------------------------------------------------------

func TestInitDBIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// First InitDB.
	db1, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("first InitDB failed: %v", err)
	}
	master := model.ProductMaster{Platform: "taobao", Factory: "f-X", FactorySKU: "sku-OK", Name: "OK", ExtraData: "{}"}
	db1.Create(&master)
	sqlDB1, _ := db1.DB()
	sqlDB1.Close()

	// Second InitDB on the same path.
	db2, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("second InitDB (idempotent) failed: %v", err)
	}
	sqlDB2, _ := db2.DB()
	t.Cleanup(func() { sqlDB2.Close() })

	// Existing data intact.
	var count int64
	db2.Model(&model.ProductMaster{}).Where("id = ?", master.ID).Count(&count)
	if count != 1 {
		t.Fatalf("data lost after idempotent InitDB: expected 1 master, got %d", count)
	}

	// Unique indexes still enforce constraints.
	dupMaster := model.ProductMaster{Platform: "taobao", Factory: "f-X", FactorySKU: "sku-OK", Name: "Dup", ExtraData: "{}"}
	err = db2.Create(&dupMaster).Error
	if err == nil {
		t.Fatal("expected unique constraint violation on duplicate ProductMaster, but insert succeeded")
	}
}

func TestInitDBUpgradesDirtyProductMasterData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "dirty-upgrade.db")

	db1, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("first InitDB failed: %v", err)
	}
	sqlDB1, err := db1.DB()
	if err != nil {
		t.Fatalf("get sql.DB failed: %v", err)
	}

	if err := db1.Exec("DROP INDEX IF EXISTS idx_product_master_platform_sku").Error; err != nil {
		t.Fatalf("drop idx_product_master_platform_sku failed: %v", err)
	}
	if err := db1.Exec("DROP INDEX IF EXISTS idx_product_wave_platform_sku").Error; err != nil {
		t.Fatalf("drop idx_product_wave_platform_sku failed: %v", err)
	}

	wave := model.Wave{WaveNo: "TASK-20990101-001", Name: "DirtyUpgradeWave", Status: "draft"}
	if err := db1.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}

	if err := db1.Exec(`
		INSERT INTO product_masters (platform, factory, factory_sku, name, cover_image, extra_data, created_at, updated_at)
		VALUES
			('taobao', 'f1', 'sku-dup', 'Master A', '', '{}', datetime('now'), datetime('now')),
			('taobao', 'f1', 'sku-dup', 'Master B', '', '{}', datetime('now'), datetime('now'))
	`).Error; err != nil {
		t.Fatalf("insert duplicate product_masters failed: %v", err)
	}

	if err := db1.Exec(`
		INSERT INTO products (platform, factory, factory_sku, name, wave_id, extra_data, created_at, updated_at)
		VALUES
			('taobao', 'f1', 'sku-wave-dup', 'Wave Product A', ?, '{}', datetime('now'), datetime('now')),
			('taobao', 'f1', 'sku-wave-dup', 'Wave Product B', ?, '{}', datetime('now'), datetime('now'))
	`, wave.ID, wave.ID).Error; err != nil {
		t.Fatalf("insert duplicate products failed: %v", err)
	}

	if err := sqlDB1.Close(); err != nil {
		t.Fatalf("close first sql.DB failed: %v", err)
	}

	db2, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("dirty upgrade InitDB failed: %v", err)
	}
	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatalf("get reopened sql.DB failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB2.Close() })

	var masterCount int64
	if err := db2.Model(&model.ProductMaster{}).
		Where("platform = ? AND factory_sku = ?", "taobao", "sku-dup").
		Count(&masterCount).Error; err != nil {
		t.Fatalf("count deduped masters failed: %v", err)
	}
	if masterCount != 1 {
		t.Fatalf("expected 1 deduped product_master, got %d", masterCount)
	}

	var productCount int64
	if err := db2.Model(&model.Product{}).
		Where("wave_id = ? AND platform = ? AND factory_sku = ?", wave.ID, "taobao", "sku-wave-dup").
		Count(&productCount).Error; err != nil {
		t.Fatalf("count deduped products failed: %v", err)
	}
	if productCount != 1 {
		t.Fatalf("expected 1 deduped product snapshot, got %d", productCount)
	}

	dupMaster := model.ProductMaster{Platform: "taobao", Factory: "f1", FactorySKU: "sku-dup", Name: "Dup", ExtraData: "{}"}
	if err := db2.Create(&dupMaster).Error; err == nil {
		t.Fatal("expected duplicate product_master insert to fail after migration")
	}
}
