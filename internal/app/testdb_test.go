package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:ws_%d?mode=memory&cache=shared", time.Now().UnixNano())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := gdb.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if err := db.AutoMigrateAll(gdb); err != nil {
		t.Fatalf("AutoMigrateAll: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return gdb
}

func newTestWorkspace(t *testing.T) *Workspace {
	t.Helper()
	return NewWorkspace(infra.NewGormStore(openTestDB(t)))
}
