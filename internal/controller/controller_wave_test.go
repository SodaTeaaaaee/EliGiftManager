package controller

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
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

func TestPersistLifecycleNilServiceIsNoop(t *testing.T) {
	t.Parallel()
	c := &WaveController{}
	if err := c.persistLifecycle(1); err != nil {
		t.Fatalf("nil lifecycleSvc should be a no-op, got %v", err)
	}
}

func TestPersistLifecycleDoesNotReopenClosedWave(t *testing.T) {
	t.Parallel()

	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	if err := gdb.AutoMigrate(
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.WaveDemandAssignment{},
		&persistence.FulfillmentLine{},
		&persistence.SupplierOrder{},
		&persistence.Shipment{},
		&persistence.ChannelSyncJob{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	waveRepo := infra.NewWaveRepository(gdb)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	shipmentRepo := infra.NewShipmentRepository(gdb)
	channelSyncRepo := infra.NewChannelSyncRepository(gdb)

	ctx := context.Background()
	wave := &domain.Wave{
		WaveNo:         "WAVE-CLOSED-1",
		Name:           "closed",
		LifecycleStage: string(domain.LifecycleStageClosed),
	}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("Create wave: %v", err)
	}
	if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{
		WaveID:           wave.ID,
		DemandDocumentID: 1,
	}); err != nil {
		t.Fatalf("Create assignment: %v", err)
	}

	c := &WaveController{
		lifecycleSvc: app.NewLifecycleProjectionService(
			waveRepo, fulfillRepo, supplierRepo, shipmentRepo, assignmentRepo, channelSyncRepo,
		),
	}
	if err := c.persistLifecycle(wave.ID); err != nil {
		t.Fatalf("persistLifecycle: %v", err)
	}

	got, err := waveRepo.FindByID(ctx, wave.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.LifecycleStage != string(domain.LifecycleStageClosed) {
		t.Errorf("LifecycleStage = %q, want %q (counts would otherwise be allocation)", got.LifecycleStage, domain.LifecycleStageClosed)
	}
}
