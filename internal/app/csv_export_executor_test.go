package app

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestCSVExportExecutor_SanitizesFormulaCells(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	exec := NewCSVExportExecutor(outputDir, nil)

	job := &domain.ChannelSyncJob{ID: 7, IntegrationProfileID: 1}
	items := []domain.ChannelSyncItem{{
		ID:                 1,
		FulfillmentLineID:  9,
		ShipmentID:         3,
		TrackingNo:         `=HYPERLINK("http://evil")`,
		CarrierCode:        "+cmd",
		ExternalDocumentNo: "@SUM(A1)",
		ExternalLineNo:     "-1+2",
	}}
	profile := &domain.IntegrationProfile{
		ID:               1,
		ConnectorKey:     "eli.csv_export",
		SourceSurface:    string(domain.SourceSurfaceRetail),
		TrackingSyncMode: "document_export",
	}

	result, err := exec.Execute(context.Background(), job, items, profile)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.AggregateStatus != "success" {
		t.Fatalf("status=%q", result.AggregateStatus)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 export file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(outputDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%v", rows)
	}
	got := rows[1]
	want := []string{"9", "3", `'=HYPERLINK("http://evil")`, "'+cmd", "'@SUM(A1)", "'-1+2"}
	if len(got) != len(want) {
		t.Fatalf("data=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cell[%d]=%q want %q (row=%v)", i, got[i], want[i], got)
		}
	}
}
