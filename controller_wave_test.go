package main

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupWaveTestDB spins up an in-memory sqlite DB migrated with only the Wave table,
// which is all controller_wave.go's CreateWave touches.
func setupWaveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(&persistence.Wave{}); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return gdb
}

// newWaveTestController builds a WaveController wired to the given in-memory DB,
// bypassing NewWaveController's db.GetDB() singleton lookup. CreateWave only reads
// c.gdb, so no other field needs to be populated for these tests.
func newWaveTestController(gdb *gorm.DB) *WaveController {
	return &WaveController{gdb: gdb}
}

// TestCreateWave_WaveTypeAndNotes covers plan 3.2's create-time surface: waveType +
// notes (+ levelTags) must persist when supplied, and an empty waveType must leave
// the field blank so the persistence-layer default ('mixed') still applies —
// preserving backward compatibility with existing name-only callers.
func TestCreateWave_WaveTypeAndNotes(t *testing.T) {
	gdb := setupWaveTestDB(t)
	c := newWaveTestController(gdb)

	created, err := c.CreateWave(dto.CreateWaveInput{
		Name:      "Membership Wave",
		WaveType:  "membership",
		Notes:     "created via dialog",
		LevelTags: `["gold"]`,
	})
	if err != nil {
		t.Fatalf("CreateWave with waveType+notes failed: %v", err)
	}
	if created.WaveType != "membership" {
		t.Fatalf("expected waveType 'membership', got %q", created.WaveType)
	}
	if created.Notes != "created via dialog" {
		t.Fatalf("expected notes to persist, got %q", created.Notes)
	}
	if created.LevelTags != `["gold"]` {
		t.Fatalf("expected levelTags to persist, got %q", created.LevelTags)
	}
	if created.Name != "Membership Wave" {
		t.Fatalf("expected name to persist, got %q", created.Name)
	}
}

func TestCreateWave_NameOnlyDefaultsWaveTypeToMixed(t *testing.T) {
	gdb := setupWaveTestDB(t)
	c := newWaveTestController(gdb)

	created, err := c.CreateWave(dto.CreateWaveInput{Name: "Plain Wave"})
	if err != nil {
		t.Fatalf("CreateWave with name only failed: %v", err)
	}
	if created.WaveType != "mixed" {
		t.Fatalf("expected waveType to default to 'mixed', got %q", created.WaveType)
	}
}

func TestCreateWave_RejectsInvalidWaveType(t *testing.T) {
	gdb := setupWaveTestDB(t)
	c := newWaveTestController(gdb)

	if _, err := c.CreateWave(dto.CreateWaveInput{Name: "Bad Wave", WaveType: "bogus"}); err == nil {
		t.Fatal("expected an error for an invalid waveType, got nil")
	}
}
