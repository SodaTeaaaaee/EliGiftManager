package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLitePartialIndexConflictTargetWithParameters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sqlite-probe.db")

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	stmts := []string{
		`CREATE TABLE product_tags (
			id INTEGER PRIMARY KEY,
			product_id INTEGER NOT NULL,
			platform TEXT NOT NULL,
			tag_name TEXT NOT NULL,
			match_mode TEXT NOT NULL,
			tag_type TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			wave_member_id INTEGER
		)`,
		`CREATE UNIQUE INDEX idx_prod_identity_tag ON product_tags(product_id, platform, tag_name, match_mode) WHERE tag_type = 'identity'`,
		`CREATE UNIQUE INDEX idx_prod_user_tag ON product_tags(product_id, wave_member_id) WHERE tag_type = 'user'`,
	}
	for _, stmt := range stmts {
		if _, err := sqlDB.Exec(stmt); err != nil {
			t.Fatalf("setup exec failed: %v\nsql=%s", err, stmt)
		}
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO product_tags(product_id, platform, tag_name, match_mode, tag_type, quantity)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, platform, tag_name, match_mode)
		WHERE tag_type = 'identity'
		DO UPDATE SET quantity = excluded.quantity
	`, 1, "BILIBILI", "提督", "gift_level", "identity", 2); err != nil {
		t.Fatalf("literal WHERE conflict target should succeed, got %v", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO product_tags(product_id, platform, tag_name, match_mode, tag_type, quantity)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, platform, tag_name, match_mode)
		WHERE tag_type = ?
		DO UPDATE SET quantity = excluded.quantity
	`, 1, "BILIBILI", "提督", "gift_level", "identity", 5, "identity"); err == nil {
		t.Fatalf("parameterized WHERE conflict target unexpectedly succeeded")
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO product_tags(product_id, platform, tag_name, match_mode, tag_type, quantity, wave_member_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, wave_member_id)
		WHERE tag_type = 'user'
		DO UPDATE SET quantity = excluded.quantity
	`, 1, "BILIBILI", "uid-1", "user_member", "user", 2, 11); err != nil {
		t.Fatalf("literal user WHERE conflict target should succeed, got %v", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO product_tags(product_id, platform, tag_name, match_mode, tag_type, quantity, wave_member_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_id, wave_member_id)
		WHERE tag_type = ?
		DO UPDATE SET quantity = excluded.quantity
	`, 1, "BILIBILI", "uid-1", "user_member", "user", 4, 11, "user"); err == nil {
		t.Fatalf("parameterized user WHERE conflict target unexpectedly succeeded")
	}
}
