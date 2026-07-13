package app

import (
	"context"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCustomerFulfillmentHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.Wave{},
		&persistence.Product{},
		&persistence.FulfillmentLine{},
		&persistence.Shipment{},
		&persistence.ShipmentLine{},
		&persistence.SupplierOrder{},
		&persistence.CustomerMergeRecord{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

func TestCustomerFulfillmentHistoryUseCase_SpansMultipleWaves(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	ctx := context.Background()

	const customerProfileID uint = 42

	waveOne := persistence.Wave{WaveNo: "W-0001", Name: "January Wave", WaveType: persistence.WaveType("mixed")}
	if err := db.Create(&waveOne).Error; err != nil {
		t.Fatalf("create wave one: %v", err)
	}
	waveTwo := persistence.Wave{WaveNo: "W-0002", Name: "February Wave", WaveType: persistence.WaveType("mixed")}
	if err := db.Create(&waveTwo).Error; err != nil {
		t.Fatalf("create wave two: %v", err)
	}

	productOne := persistence.Product{WaveID: waveOne.ID, SupplierPlatform: "taobao", FactorySKU: "SKU-1", Name: "Charm"}
	if err := db.Create(&productOne).Error; err != nil {
		t.Fatalf("create product one: %v", err)
	}
	productTwo := persistence.Product{WaveID: waveTwo.ID, SupplierPlatform: "taobao", FactorySKU: "SKU-2", Name: "Postcard"}
	if err := db.Create(&productTwo).Error; err != nil {
		t.Fatalf("create product two: %v", err)
	}

	profileIDCopy1 := customerProfileID
	lineOne := persistence.FulfillmentLine{
		WaveID:            waveOne.ID,
		CustomerProfileID: &profileIDCopy1,
		ProductID:         &productOne.ID,
		Quantity:          1,
		AllocationState:   "allocated",
		AddressState:      "ready",
		SupplierState:     "submitted",
		ChannelSyncState:  "synced",
		LineReason:        persistence.FulfillmentLineReason("entitlement"),
	}
	if err := db.Create(&lineOne).Error; err != nil {
		t.Fatalf("create line one: %v", err)
	}

	profileIDCopy2 := customerProfileID
	lineTwo := persistence.FulfillmentLine{
		WaveID:            waveTwo.ID,
		CustomerProfileID: &profileIDCopy2,
		ProductID:         &productTwo.ID,
		Quantity:          2,
		AllocationState:   "allocated",
		AddressState:      "pending",
		SupplierState:     "draft",
		ChannelSyncState:  "pending",
		LineReason:        persistence.FulfillmentLineReason("entitlement"),
	}
	if err := db.Create(&lineTwo).Error; err != nil {
		t.Fatalf("create line two: %v", err)
	}

	// A fulfillment line belonging to a DIFFERENT customer must not leak in.
	otherProfileID := uint(999)
	otherLine := persistence.FulfillmentLine{
		WaveID:            waveOne.ID,
		CustomerProfileID: &otherProfileID,
		ProductID:         &productOne.ID,
		Quantity:          1,
		AllocationState:   "allocated",
		AddressState:      "ready",
		SupplierState:     "submitted",
		ChannelSyncState:  "synced",
		LineReason:        persistence.FulfillmentLineReason("entitlement"),
	}
	if err := db.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other-customer line: %v", err)
	}

	// Give line one a recorded shipment so tracking fields should surface.
	supplierOrder := persistence.SupplierOrder{WaveID: waveOne.ID, SupplierPlatform: "taobao"}
	if err := db.Create(&supplierOrder).Error; err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	shipment := persistence.Shipment{
		SupplierOrderID: supplierOrder.ID,
		TrackingNo:      "TRACK-123",
		CarrierName:     "SF Express",
		Status:          persistence.ShipmentStatusShipped,
	}
	if err := db.Create(&shipment).Error; err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	shipmentLine := persistence.ShipmentLine{ShipmentID: shipment.ID, FulfillmentLineID: lineOne.ID, Quantity: 1}
	if err := db.Create(&shipmentLine).Error; err != nil {
		t.Fatalf("create shipment line: %v", err)
	}

	uc := NewCustomerFulfillmentHistoryUseCase(db)
	rows, err := uc.GetCustomerFulfillmentHistory(ctx, customerProfileID)
	if err != nil {
		t.Fatalf("GetCustomerFulfillmentHistory: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows spanning 2 waves, got %d", len(rows))
	}

	seenWaves := map[uint]bool{}
	var shippedRow *struct {
		trackingNo  string
		carrierName string
		status      string
	}
	for _, row := range rows {
		seenWaves[row.WaveID] = true
		if row.FulfillmentLineID == lineOne.ID {
			if row.TrackingNo != "TRACK-123" {
				t.Fatalf("expected tracking no TRACK-123 on line one, got %q", row.TrackingNo)
			}
			if row.CarrierName != "SF Express" {
				t.Fatalf("expected carrier SF Express on line one, got %q", row.CarrierName)
			}
			if row.ShipmentID == nil || *row.ShipmentID != shipment.ID {
				t.Fatalf("expected shipment id %d on line one, got %v", shipment.ID, row.ShipmentID)
			}
			if row.WaveNo != "W-0001" || row.WaveName != "January Wave" {
				t.Fatalf("unexpected wave context on line one: %+v", row)
			}
			if row.ProductName != "Charm" || row.ProductSKU != "SKU-1" {
				t.Fatalf("unexpected product context on line one: %+v", row)
			}
			shippedRow = &struct {
				trackingNo  string
				carrierName string
				status      string
			}{row.TrackingNo, row.CarrierName, row.ShipmentStatus}
		}
		if row.FulfillmentLineID == lineTwo.ID && row.TrackingNo != "" {
			t.Fatalf("line two should have no shipment, got tracking %q", row.TrackingNo)
		}
	}
	if shippedRow == nil {
		t.Fatal("expected to find the shipped row (line one) in results")
	}
	if !seenWaves[waveOne.ID] || !seenWaves[waveTwo.ID] {
		t.Fatalf("expected rows to span both waves, got waves: %+v", seenWaves)
	}
}

func TestCustomerFulfillmentHistoryUseCase_NoRows(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	ctx := context.Background()

	uc := NewCustomerFulfillmentHistoryUseCase(db)
	rows, err := uc.GetCustomerFulfillmentHistory(ctx, 12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows, got %d", len(rows))
	}
}

func createCustomerFulfillmentHistoryLines(t *testing.T, db *gorm.DB, profileIDs ...uint) map[uint]uint {
	t.Helper()

	wave := persistence.Wave{WaveNo: "W-MERGE", WaveType: persistence.WaveType("mixed")}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create merge history wave: %v", err)
	}

	lineIDs := make(map[uint]uint, len(profileIDs))
	for _, profileID := range profileIDs {
		id := profileID
		line := persistence.FulfillmentLine{
			WaveID:            wave.ID,
			CustomerProfileID: &id,
			Quantity:          1,
			LineReason:        persistence.FulfillmentLineReason("entitlement"),
		}
		if err := db.Create(&line).Error; err != nil {
			t.Fatalf("create fulfillment line for profile %d: %v", profileID, err)
		}
		lineIDs[profileID] = line.ID
	}
	return lineIDs
}

func createCustomerMergeRecord(t *testing.T, db *gorm.DB, sourceProfileID, targetProfileID uint) persistence.CustomerMergeRecord {
	t.Helper()

	record := persistence.CustomerMergeRecord{
		SourceProfileID: sourceProfileID,
		TargetProfileID: targetProfileID,
		Payload:         `{}`,
		CreatedAt:       time.Now(),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create customer merge record %d -> %d: %v", sourceProfileID, targetProfileID, err)
	}
	return record
}

func assertCustomerFulfillmentHistoryLineIDs(t *testing.T, rows []dto.CustomerFulfillmentHistoryRowDTO, want ...uint) {
	t.Helper()

	if len(rows) != len(want) {
		t.Fatalf("expected %d fulfillment history rows, got %d", len(want), len(rows))
	}
	got := make(map[uint]bool, len(rows))
	for _, row := range rows {
		got[row.FulfillmentLineID] = true
	}
	for _, lineID := range want {
		if !got[lineID] {
			t.Errorf("expected fulfillment history to include line %d; got line IDs %v", lineID, got)
		}
	}
}

func TestCustomerFulfillmentHistoryUseCase_IncludesSourceAndTargetHistoryAfterMerge(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	lineIDs := createCustomerFulfillmentHistoryLines(t, db, 10, 20)
	createCustomerMergeRecord(t, db, 10, 20)

	rows, err := NewCustomerFulfillmentHistoryUseCase(db).GetCustomerFulfillmentHistory(context.Background(), 20)
	if err != nil {
		t.Fatalf("GetCustomerFulfillmentHistory: %v", err)
	}
	assertCustomerFulfillmentHistoryLineIDs(t, rows, lineIDs[10], lineIDs[20])

	var sourceLine persistence.FulfillmentLine
	if err := db.First(&sourceLine, lineIDs[10]).Error; err != nil {
		t.Fatalf("reload source fulfillment line: %v", err)
	}
	if sourceLine.CustomerProfileID == nil || *sourceLine.CustomerProfileID != 10 {
		t.Fatalf("source fulfillment line profile was rewritten: got %v, want 10", sourceLine.CustomerProfileID)
	}
}

func TestCustomerFulfillmentHistoryUseCase_IncludesTransitiveMergeHistory(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	lineIDs := createCustomerFulfillmentHistoryLines(t, db, 10, 20, 30)
	createCustomerMergeRecord(t, db, 10, 20)
	createCustomerMergeRecord(t, db, 20, 30)

	rows, err := NewCustomerFulfillmentHistoryUseCase(db).GetCustomerFulfillmentHistory(context.Background(), 30)
	if err != nil {
		t.Fatalf("GetCustomerFulfillmentHistory: %v", err)
	}
	assertCustomerFulfillmentHistoryLineIDs(t, rows, lineIDs[10], lineIDs[20], lineIDs[30])
}

func TestCustomerFulfillmentHistoryUseCase_ExcludesUndoneMergeHistory(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	lineIDs := createCustomerFulfillmentHistoryLines(t, db, 10, 20)
	record := createCustomerMergeRecord(t, db, 10, 20)

	undoneAt := time.Now()
	if err := db.Model(&persistence.CustomerMergeRecord{}).
		Where("id = ?", record.ID).
		Update("undone_at", undoneAt).Error; err != nil {
		t.Fatalf("mark customer merge record undone: %v", err)
	}

	rows, err := NewCustomerFulfillmentHistoryUseCase(db).GetCustomerFulfillmentHistory(context.Background(), 20)
	if err != nil {
		t.Fatalf("GetCustomerFulfillmentHistory: %v", err)
	}
	assertCustomerFulfillmentHistoryLineIDs(t, rows, lineIDs[20])
}

func TestCustomerFulfillmentHistoryUseCase_CycleTerminates(t *testing.T) {
	db := setupCustomerFulfillmentHistoryTestDB(t)
	lineIDs := createCustomerFulfillmentHistoryLines(t, db, 10, 20)
	createCustomerMergeRecord(t, db, 10, 20)
	createCustomerMergeRecord(t, db, 20, 10)

	type result struct {
		rows []dto.CustomerFulfillmentHistoryRowDTO
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		rows, err := NewCustomerFulfillmentHistoryUseCase(db).GetCustomerFulfillmentHistory(context.Background(), 20)
		resultCh <- result{rows: rows, err: err}
	}()

	select {
	case got := <-resultCh:
		if got.err != nil {
			t.Fatalf("GetCustomerFulfillmentHistory: %v", got.err)
		}
		assertCustomerFulfillmentHistoryLineIDs(t, got.rows, lineIDs[10], lineIDs[20])
	case <-time.After(2 * time.Second):
		t.Fatal("GetCustomerFulfillmentHistory did not terminate for cyclic merge records")
	}
}
