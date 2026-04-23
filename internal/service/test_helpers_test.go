package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"gorm.io/gorm"
)

func writeTestCSVFile(t *testing.T, lines []string) string {
	t.Helper()

	tempDir := t.TempDir()
	csvFile := filepath.Join(tempDir, "test.csv")
	content := strings.Join(lines, "\n")

	if err := os.WriteFile(csvFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write test csv file: %v", err)
	}

	return csvFile
}

func newServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "service-test.db")

	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql db: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}
