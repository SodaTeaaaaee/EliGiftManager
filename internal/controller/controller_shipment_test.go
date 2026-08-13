package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
)

func TestValidateShipmentImportFilePath(t *testing.T) {
	if err := validateShipmentImportFilePath(""); err != nil {
		t.Fatalf("empty path should be allowed: %v", err)
	}

	if err := validateShipmentImportFilePath("..\x00secret.csv"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("expected NUL rejection, got %v", err)
	}

	if err := validateShipmentImportFilePath(filepath.Join("..", "outside.csv")); err == nil || !strings.Contains(err.Error(), "..") {
		t.Fatalf("expected .. rejection, got %v", err)
	}

	if err := validateShipmentImportFilePath("notes.txt"); err == nil || !strings.Contains(err.Error(), "unsupported import file extension") {
		t.Fatalf("expected extension rejection, got %v", err)
	}

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "factory-return.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if err := validateShipmentImportFilePath(csvPath); err != nil {
		t.Fatalf("csv outside data dir should be allowed: %v", err)
	}

	orig := maxShipmentImportFileBytes
	maxShipmentImportFileBytes = 4
	t.Cleanup(func() { maxShipmentImportFileBytes = orig })
	if err := validateShipmentImportFilePath(csvPath); err == nil || !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestMapAndReconcileShipmentsRejectsDotDotFilePath(t *testing.T) {
	t.Parallel()

	c := &ShipmentController{}
	_, err := c.MapAndReconcileShipments(dto.MapAndReconcileShipmentsInput{
		WaveID:   1,
		FilePath: filepath.Join("..", "secret.csv"),
	})
	if err == nil || !strings.Contains(err.Error(), "..") {
		t.Fatalf("expected .. rejection before import work, got %v", err)
	}
}
