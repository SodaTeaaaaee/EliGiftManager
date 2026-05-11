package db

import (
	"path/filepath"
	"testing"
)

func TestInitDBCreatesPartialUniqueIndexesForProductTags(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "partial-indexes.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	var indexes []struct {
		Name string
		SQL  string
	}
	if err := db.Raw(`
		SELECT name, sql
		FROM sqlite_master
		WHERE type = 'index' AND tbl_name = 'product_tags'
		ORDER BY name
	`).Scan(&indexes).Error; err != nil {
		t.Fatalf("load indexes failed: %v", err)
	}

	var identitySQL, userSQL string
	for _, idx := range indexes {
		switch idx.Name {
		case "idx_prod_identity_tag":
			identitySQL = idx.SQL
		case "idx_prod_user_tag":
			userSQL = idx.SQL
		}
	}

	if identitySQL == "" {
		t.Fatalf("idx_prod_identity_tag not found, indexes=%#v", indexes)
	}
	if userSQL == "" {
		t.Fatalf("idx_prod_user_tag not found, indexes=%#v", indexes)
	}
	if identitySQL != "CREATE UNIQUE INDEX idx_prod_identity_tag ON product_tags(product_id, platform, tag_name, match_mode) WHERE tag_type = 'identity'" {
		t.Fatalf("unexpected idx_prod_identity_tag SQL: %s", identitySQL)
	}
	if userSQL != "CREATE UNIQUE INDEX idx_prod_user_tag ON product_tags(product_id, wave_member_id) WHERE tag_type = 'user'" {
		t.Fatalf("unexpected idx_prod_user_tag SQL: %s", userSQL)
	}
}

