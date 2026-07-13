package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newActionCenterTestDB opens an in-memory sqlite DB and migrates every persistence
// model touched by the wave-overview / basis-drift call graph that
// ActionCenterUseCase depends on (mirrors controller_wave.go's wiring).
func newActionCenterTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.DemandDocument{},
		&persistence.DemandLine{},
		&persistence.WaveDemandAssignment{},
		&persistence.FulfillmentLine{},
		&persistence.SupplierOrder{},
		&persistence.SupplierOrderLine{},
		&persistence.Shipment{},
		&persistence.ShipmentLine{},
		&persistence.ChannelSyncJob{},
		&persistence.ChannelSyncItem{},
		&persistence.IntegrationProfile{},
		&persistence.ChannelClosureDecisionRecord{},
		&persistence.FulfillmentAdjustment{},
		&persistence.HistoryScope{},
		&persistence.HistoryNode{},
		&persistence.ProductMaster{},
		&persistence.Product{},
	); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	return db
}

// buildActionCenterUseCase wires ActionCenterUseCase exactly the way
// controller_action_center.go's NewActionCenterController does, against the given DB.
func buildActionCenterUseCase(db *gorm.DB) ActionCenterUseCase {
	waveRepo := infra.NewWaveRepository(db)
	demandRepo := infra.NewDemandRepository(db)
	fulfillRepo := infra.NewFulfillmentRepository(db)
	supplierRepo := infra.NewSupplierOrderRepository(db)
	assignmentRepo := infra.NewWaveDemandAssignmentRepository(db)
	shipmentRepo := infra.NewShipmentRepository(db)
	productRepo := infra.NewProductRepository(db)
	profileRepo := infra.NewIntegrationProfileRepository(db)
	channelSyncRepo := infra.NewChannelSyncRepository(db)
	closureDecisionRepo := infra.NewClosureDecisionRepository(db)
	historyScopeRepo := infra.NewHistoryScopeRepository(db)
	historyNodeRepo := infra.NewHistoryNodeRepository(db)
	adjustmentRepo := infra.NewFulfillmentAdjustmentRepository(db)

	basisDriftUC := NewBasisDriftDetectionUseCase(supplierRepo, shipmentRepo, channelSyncRepo, fulfillRepo)
	historyHeadUC := NewHistoryHeadQueryUseCase(historyScopeRepo, historyNodeRepo)
	overviewProjUC := NewWaveOverviewProjectionUseCase(channelSyncRepo, closureDecisionRepo, basisDriftUC, historyHeadUC)
	overviewQueryUC := NewWaveOverviewQueryUseCase(
		waveRepo, fulfillRepo, supplierRepo, assignmentRepo, demandRepo,
		shipmentRepo, productRepo, profileRepo, historyScopeRepo, historyNodeRepo,
		overviewProjUC, adjustmentRepo,
	)

	return NewActionCenterUseCase(waveRepo, demandRepo, overviewQueryUC)
}

// TestActionCenterSummary_BucketsAndInboxCount seeds:
//   - a wave with one address-blocked fulfillment line (AddressState "missing")
//   - a non-draft supplier order line pointing at a fulfillment line that does not
//     exist in the wave, which the basis-drift structural-integrity check must flag
//     as a "target_deleted" / ReviewRequirement "required" drift signal
//   - a demand document that is NOT assigned to any wave, with one DemandLine still
//     at RoutingDisposition "pending_intake"
//
// and asserts the resulting bucket counts and inbox pending-intake count.
func TestActionCenterSummary_BucketsAndInboxCount(t *testing.T) {
	db := newActionCenterTestDB(t)
	ctx := context.Background()

	waveRepo := infra.NewWaveRepository(db)
	fulfillRepo := infra.NewFulfillmentRepository(db)
	supplierRepo := infra.NewSupplierOrderRepository(db)
	demandRepo := infra.NewDemandRepository(db)

	wave := &domain.Wave{
		WaveNo:         "WAVE-AC-TEST-1",
		Name:           "action center test wave",
		WaveType:       "mixed",
		LifecycleStage: "intake",
	}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("create wave: %v", err)
	}

	// Address-blocked fulfillment line.
	addressBlockedLine := &domain.FulfillmentLine{
		WaveID:          wave.ID,
		Quantity:        1,
		AllocationState: "ready",
		AddressState:    "missing",
		LineReason:      "entitlement",
	}
	if err := fulfillRepo.Create(ctx, addressBlockedLine); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}

	// Non-draft supplier order whose line targets a fulfillment line ID that does not
	// exist in this wave — triggers the structural "target_deleted" drift signal.
	nonExistentFulfillmentLineID := uint(999999)
	order := &domain.SupplierOrder{
		WaveID: wave.ID,
		Status: "submitted",
	}
	if err := supplierRepo.Create(ctx, order); err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	orderLine := &domain.SupplierOrderLine{
		SupplierOrderID:   order.ID,
		FulfillmentLineID: nonExistentFulfillmentLineID,
		SubmittedQuantity: 1,
	}
	if err := supplierRepo.CreateLine(ctx, orderLine); err != nil {
		t.Fatalf("create supplier order line: %v", err)
	}

	// Unassigned demand document with one pending-intake line.
	doc := &domain.DemandDocument{
		Kind:        "membership_entitlement",
		CaptureMode: "manual",
	}
	if err := demandRepo.Create(ctx, doc); err != nil {
		t.Fatalf("create demand document: %v", err)
	}
	line := &domain.DemandLine{
		DemandDocumentID:   doc.ID,
		LineType:           "gift",
		RoutingDisposition: string(domain.RoutingDispositionPendingIntake),
	}
	if err := demandRepo.CreateLine(ctx, line); err != nil {
		t.Fatalf("create demand line: %v", err)
	}

	uc := buildActionCenterUseCase(db)
	summary, err := uc.GetActionCenterSummary(ctx)
	if err != nil {
		t.Fatalf("GetActionCenterSummary: %v", err)
	}

	if len(summary.Waves) != 1 {
		t.Fatalf("expected 1 wave summary, got %d (%+v)", len(summary.Waves), summary.Waves)
	}
	waveSummary := summary.Waves[0]
	if waveSummary.WaveID != wave.ID {
		t.Errorf("waveSummary.WaveID = %d, want %d", waveSummary.WaveID, wave.ID)
	}

	bucketCounts := make(map[string]int, len(waveSummary.Buckets))
	for _, b := range waveSummary.Buckets {
		bucketCounts[b.BucketKind] = b.Count
		if b.WaveID != wave.ID {
			t.Errorf("bucket %q WaveID = %d, want %d", b.BucketKind, b.WaveID, wave.ID)
		}
	}

	if bucketCounts["missing_address"] != 1 {
		t.Errorf("missing_address count = %d, want 1", bucketCounts["missing_address"])
	}
	if bucketCounts["drift_needs_review"] != 1 {
		t.Errorf("drift_needs_review count = %d, want 1", bucketCounts["drift_needs_review"])
	}
	for _, absentKind := range []string{"waiting_input", "mapping_blocked", "channel_sync_failed", "awaiting_manual_closure"} {
		if count, ok := bucketCounts[absentKind]; ok {
			t.Errorf("expected bucket %q to be absent, got count %d", absentKind, count)
		}
	}

	wantWaveTotal := 2
	if waveSummary.TotalBlockedCount != wantWaveTotal {
		t.Errorf("TotalBlockedCount = %d, want %d", waveSummary.TotalBlockedCount, wantWaveTotal)
	}

	if summary.InboxPendingIntakeCount != 1 {
		t.Errorf("InboxPendingIntakeCount = %d, want 1", summary.InboxPendingIntakeCount)
	}

	navCounts := make(map[string]int, len(summary.NavBadges))
	for _, b := range summary.NavBadges {
		navCounts[b.NavKey] = b.Count
	}
	if navCounts["waves"] != 1 {
		t.Errorf("nav badge waves = %d, want 1", navCounts["waves"])
	}
	if navCounts["inbox"] != 1 {
		t.Errorf("nav badge inbox = %d, want 1", navCounts["inbox"])
	}
	if navCounts["home"] != wantWaveTotal+1 {
		t.Errorf("nav badge home = %d, want %d", navCounts["home"], wantWaveTotal+1)
	}
}
