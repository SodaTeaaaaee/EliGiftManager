package main

import (
	"path/filepath"
	"testing"

	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

func newIdentityTagControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "identity-controller.db")
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
	return db
}

func TestUpsertIdentityTagUsesPartialUniqueIndex(t *testing.T) {
	db := newIdentityTagControllerTestDB(t)

	wave := model.Wave{WaveNo: "TASK-IDX-IDENTITY-001", Name: "wave", Status: "draft"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}
	product := model.Product{
		Platform:   "BILIBILI",
		Factory:    "f",
		FactorySKU: "sku-identity",
		Name:       "gift",
		WaveID:     &wave.ID,
		ExtraData:  "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	var pc ProductController
	if err := pc.UpsertIdentityTag(product.ID, "BILIBILI", "提督", "gift_level", 2); err != nil {
		t.Fatalf("first UpsertIdentityTag failed: %v", err)
	}
	if err := pc.UpsertIdentityTag(product.ID, "BILIBILI", "提督", "gift_level", 5); err != nil {
		t.Fatalf("second UpsertIdentityTag failed: %v", err)
	}

	var tags []model.ProductTag
	if err := db.Where("product_id = ? AND tag_type = 'identity'", product.ID).Find(&tags).Error; err != nil {
		t.Fatalf("query identity tags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected a single identity row after upsert, got %d", len(tags))
	}
	if tags[0].Quantity != 5 {
		t.Fatalf("expected identity quantity to be updated to 5, got %d", tags[0].Quantity)
	}
}

func TestUpsertUserTagUsesPartialUniqueIndex(t *testing.T) {
	db := newIdentityTagControllerTestDB(t)

	wave := model.Wave{WaveNo: "TASK-IDX-USER-001", Name: "wave", Status: "draft"}
	member := model.Member{Platform: "BILIBILI", PlatformUID: "uid-user-1", ExtraData: "{}"}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave failed: %v", err)
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member failed: %v", err)
	}
	product := model.Product{
		Platform:   "BILIBILI",
		Factory:    "f",
		FactorySKU: "sku-user",
		Name:       "gift",
		WaveID:     &wave.ID,
		ExtraData:  "{}",
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}
	wm := model.WaveMember{
		WaveID:         wave.ID,
		MemberID:       member.ID,
		Platform:       member.Platform,
		PlatformUID:    member.PlatformUID,
		GiftLevel:      "提督",
		LatestNickname: "nick",
	}
	if err := db.Create(&wm).Error; err != nil {
		t.Fatalf("create wave member failed: %v", err)
	}

	var pc ProductController
	if err := pc.UpsertUserTag(product.ID, wm.ID, 1); err != nil {
		t.Fatalf("first UpsertUserTag failed: %v", err)
	}
	if err := pc.UpsertUserTag(product.ID, wm.ID, 4); err != nil {
		t.Fatalf("second UpsertUserTag failed: %v", err)
	}

	var tags []model.ProductTag
	if err := db.Where("product_id = ? AND tag_type = 'user'", product.ID).Find(&tags).Error; err != nil {
		t.Fatalf("query user tags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected a single user row after upsert, got %d", len(tags))
	}
	if tags[0].Quantity != 4 {
		t.Fatalf("expected user quantity to be updated to 4, got %d", tags[0].Quantity)
	}
	if tags[0].MatchMode != "user_member" {
		t.Fatalf("expected user match_mode to stay user_member, got %q", tags[0].MatchMode)
	}
}
