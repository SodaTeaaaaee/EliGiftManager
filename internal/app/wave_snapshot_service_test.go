package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type waveSnapshotServiceFixture struct {
	db          *gorm.DB
	svc         *WaveSnapshotService
	fulfillRepo *snapshotHarnessFulfillRepo
}

func newWaveSnapshotServiceTestDB(t *testing.T, withShipmentTables bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil && sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}

	models := []interface{}{
		&persistence.Wave{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.AllocationPolicyRule{},
		&persistence.FulfillmentAdjustment{},
		&persistence.FulfillmentLine{},
		&persistence.WaveDemandAssignment{},
	}
	if withShipmentTables {
		models = append(models,
			&persistence.SupplierOrder{},
			&persistence.SupplierOrderLine{},
			&persistence.Shipment{},
			&persistence.ShipmentLine{},
		)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func newWaveSnapshotServiceFixture(t *testing.T, withShipmentTables bool) *waveSnapshotServiceFixture {
	t.Helper()
	db := newWaveSnapshotServiceTestDB(t, withShipmentTables)
	fulfillRepo := &snapshotHarnessFulfillRepo{db: db}
	svc := NewWaveSnapshotService(
		db,
		&snapshotHarnessRuleRepo{},
		&snapshotHarnessAdjRepo{},
		&snapshotHarnessAssignmentRepo{},
		&snapshotHarnessWaveRepo{},
		fulfillRepo,
	)
	return &waveSnapshotServiceFixture{db: db, svc: svc, fulfillRepo: fulfillRepo}
}

func mustCreateSnapshotTestLine(t *testing.T, f *waveSnapshotServiceFixture, waveID uint, qty int) *domain.FulfillmentLine {
	t.Helper()
	line := &domain.FulfillmentLine{
		WaveID:          waveID,
		Quantity:        qty,
		AllocationState: "allocated",
		LineReason:      string(domain.LineReasonRetailOrder),
		GeneratedBy:     "test",
	}
	if err := f.fulfillRepo.Create(context.Background(), line); err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}
	return line
}

func mustCreateSnapshotTestShipment(t *testing.T, f *waveSnapshotServiceFixture, waveID, fulfillmentLineID uint, status persistence.ShipmentStatus) {
	t.Helper()
	order := &persistence.SupplierOrder{
		WaveID:           waveID,
		SupplierPlatform: "test.factory",
		BatchNo:          "SNAP-BATCH",
		SubmissionMode:   persistence.SubmissionModeCSV,
		Status:           persistence.SupplierOrderStatusSubmitted,
	}
	if err := f.db.Create(order).Error; err != nil {
		t.Fatalf("create supplier order: %v", err)
	}
	shipment := &persistence.Shipment{
		SupplierOrderID:  order.ID,
		SupplierPlatform: "test.factory",
		ShipmentNo:       "SHIP-SNAP-1",
		Status:           status,
	}
	if err := f.db.Create(shipment).Error; err != nil {
		t.Fatalf("create shipment: %v", err)
	}
	shipLine := &persistence.ShipmentLine{
		ShipmentID:        shipment.ID,
		FulfillmentLineID: fulfillmentLineID,
		Quantity:          1,
	}
	if err := f.db.Create(shipLine).Error; err != nil {
		t.Fatalf("create shipment line: %v", err)
	}
}

func TestRestoreSnapshotSucceedsWithoutShipments(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, true)
	ctx := context.Background()
	waveID := uint(1)
	line := mustCreateSnapshotTestLine(t, f, waveID, 5)
	origID := line.ID

	payload, err := f.svc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}

	line.Quantity = 99
	if err := f.fulfillRepo.Update(ctx, line); err != nil {
		t.Fatalf("update line: %v", err)
	}

	if err := f.svc.RestoreSnapshot(ctx, payload); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	restored, err := f.fulfillRepo.FindByID(ctx, origID)
	if err != nil {
		t.Fatalf("FindByID after restore: %v", err)
	}
	if restored.Quantity != 5 {
		t.Errorf("quantity after restore = %d, want 5", restored.Quantity)
	}
}

func TestRestoreSnapshotSucceedsWhenShipmentTablesMissing(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, false)
	ctx := context.Background()
	waveID := uint(2)
	mustCreateSnapshotTestLine(t, f, waveID, 2)

	payload, err := f.svc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}
	if err := f.svc.RestoreSnapshot(ctx, payload); err != nil {
		t.Fatalf("RestoreSnapshot without shipment tables: %v", err)
	}
}

func TestCaptureSnapshotOmitsShipments(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, true)
	ctx := context.Background()
	waveID := uint(3)
	line := mustCreateSnapshotTestLine(t, f, waveID, 1)
	mustCreateSnapshotTestShipment(t, f, waveID, line.ID, persistence.ShipmentStatusShipped)

	payload, err := f.svc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if _, ok := raw["shipments"]; ok {
		t.Fatal("snapshot unexpectedly includes shipments")
	}
	var snap WaveSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		t.Fatalf("unmarshal WaveSnapshot: %v", err)
	}
	if snap.SchemaVersion != snapshotSchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", snap.SchemaVersion, snapshotSchemaVersion)
	}
}

func TestRestoreSnapshotRefusesNonVoidedShipments(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, true)
	ctx := context.Background()
	waveID := uint(4)
	line := mustCreateSnapshotTestLine(t, f, waveID, 3)
	origID := line.ID
	mustCreateSnapshotTestShipment(t, f, waveID, origID, persistence.ShipmentStatusShipped)

	payload, err := f.svc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}

	err = f.svc.RestoreSnapshot(ctx, payload)
	if err == nil {
		t.Fatal("expected RestoreSnapshot to refuse when non-voided shipments exist")
	}
	if !errors.Is(err, ErrRestoreBlockedByNonVoidedShipments) {
		t.Fatalf("error = %v, want %v", err, ErrRestoreBlockedByNonVoidedShipments)
	}
	if !strings.Contains(err.Error(), "non-voided shipment") {
		t.Errorf("error %q missing 'non-voided shipment'", err)
	}

	still, findErr := f.fulfillRepo.FindByID(ctx, origID)
	if findErr != nil {
		t.Fatalf("fulfillment line was deleted despite refused restore: %v", findErr)
	}
	if still.ID != origID {
		t.Errorf("fulfillment line ID = %d, want %d", still.ID, origID)
	}

	var shipLine persistence.ShipmentLine
	if dbErr := f.db.Where("fulfillment_line_id = ?", origID).First(&shipLine).Error; dbErr != nil {
		t.Fatalf("shipment line missing after refused restore: %v", dbErr)
	}
	if shipLine.FulfillmentLineID != origID {
		t.Errorf("shipment line FulfillmentLineID = %d, want %d", shipLine.FulfillmentLineID, origID)
	}
}

func TestRestoreSnapshotAllowsWhenOnlyVoidedShipments(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, true)
	ctx := context.Background()
	waveID := uint(5)
	line := mustCreateSnapshotTestLine(t, f, waveID, 4)
	origID := line.ID
	mustCreateSnapshotTestShipment(t, f, waveID, origID, persistence.ShipmentStatusVoided)

	payload, err := f.svc.CaptureSnapshot(ctx, waveID)
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}

	line.Quantity = 40
	if err := f.fulfillRepo.Update(ctx, line); err != nil {
		t.Fatalf("update line: %v", err)
	}

	if err := f.svc.RestoreSnapshot(ctx, payload); err != nil {
		t.Fatalf("RestoreSnapshot with only voided shipments: %v", err)
	}

	restored, err := f.fulfillRepo.FindByID(ctx, origID)
	if err != nil {
		t.Fatalf("FindByID after restore: %v", err)
	}
	if restored.Quantity != 4 {
		t.Errorf("quantity after restore = %d, want 4", restored.Quantity)
	}
}

func TestRestoreSnapshotInvalidJSON(t *testing.T) {
	t.Parallel()
	f := newWaveSnapshotServiceFixture(t, false)
	err := f.svc.RestoreSnapshot(context.Background(), "{not-json")
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "snapshot: unmarshal payload") {
		t.Errorf("error %q missing unmarshal prefix", err)
	}
}

// snapshotHarnessFulfillRepo is a test-local GORM adapter so snapshot tests do
// not depend on infra.NewFulfillmentRepository (and the rest of internal/infra).
type snapshotHarnessFulfillRepo struct {
	db *gorm.DB
}

func (r *snapshotHarnessFulfillRepo) Create(ctx context.Context, line *domain.FulfillmentLine) error {
	p := persistence.FulfillmentLineFromDomain(line)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.FulfillmentLineToDomain(p)
	return nil
}

func (r *snapshotHarnessFulfillRepo) FindByID(ctx context.Context, id uint) (*domain.FulfillmentLine, error) {
	var p persistence.FulfillmentLine
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FulfillmentLineToDomain(&p), nil
}

func (r *snapshotHarnessFulfillRepo) ListByWave(ctx context.Context, waveID uint) ([]domain.FulfillmentLine, error) {
	var ps []persistence.FulfillmentLine
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.FulfillmentLine, len(ps))
	for i := range ps {
		result[i] = *persistence.FulfillmentLineToDomain(&ps[i])
	}
	return result, nil
}

func (r *snapshotHarnessFulfillRepo) Update(ctx context.Context, line *domain.FulfillmentLine) error {
	p := persistence.FulfillmentLineFromDomain(line)
	p.ID = line.ID
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *snapshotHarnessFulfillRepo) DeleteByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("wave_id = ?", waveID).Delete(&persistence.FulfillmentLine{}).Error
}

func (r *snapshotHarnessFulfillRepo) DeleteByWaveAndGeneratedBy(context.Context, uint, string) error {
	return fmt.Errorf("not implemented")
}

func (r *snapshotHarnessFulfillRepo) ReplaceByWaveAndGeneratedBy(context.Context, uint, string, []domain.FulfillmentLine) error {
	return fmt.Errorf("not implemented")
}

func (r *snapshotHarnessFulfillRepo) BulkUpdateStates(context.Context, []domain.FulfillmentLineStateUpdate) error {
	return fmt.Errorf("not implemented")
}

func (r *snapshotHarnessFulfillRepo) BulkUpdateCustomerProfileID(context.Context, uint, uint) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

type snapshotHarnessRuleRepo struct{}

func (snapshotHarnessRuleRepo) Create(context.Context, *domain.AllocationPolicyRule) error {
	return nil
}
func (snapshotHarnessRuleRepo) FindByID(context.Context, uint) (*domain.AllocationPolicyRule, error) {
	return nil, fmt.Errorf("not found")
}
func (snapshotHarnessRuleRepo) ListByWave(context.Context, uint) ([]domain.AllocationPolicyRule, error) {
	return nil, nil
}
func (snapshotHarnessRuleRepo) Update(context.Context, *domain.AllocationPolicyRule) error {
	return nil
}
func (snapshotHarnessRuleRepo) Delete(context.Context, uint) error       { return nil }
func (snapshotHarnessRuleRepo) DeleteByWave(context.Context, uint) error { return nil }

type snapshotHarnessAdjRepo struct{}

func (snapshotHarnessAdjRepo) Create(context.Context, *domain.FulfillmentAdjustment) error {
	return nil
}
func (snapshotHarnessAdjRepo) FindByID(context.Context, uint) (*domain.FulfillmentAdjustment, error) {
	return nil, fmt.Errorf("not found")
}
func (snapshotHarnessAdjRepo) Update(context.Context, *domain.FulfillmentAdjustment) error {
	return nil
}
func (snapshotHarnessAdjRepo) Delete(context.Context, uint) error       { return nil }
func (snapshotHarnessAdjRepo) DeleteByWave(context.Context, uint) error { return nil }
func (snapshotHarnessAdjRepo) ListByWave(context.Context, uint) ([]domain.FulfillmentAdjustment, error) {
	return nil, nil
}
func (snapshotHarnessAdjRepo) ListByFulfillmentLine(context.Context, uint) ([]domain.FulfillmentAdjustment, error) {
	return nil, nil
}

type snapshotHarnessAssignmentRepo struct{}

func (snapshotHarnessAssignmentRepo) Create(context.Context, *domain.WaveDemandAssignment) error {
	return nil
}
func (snapshotHarnessAssignmentRepo) ExistsByDocument(context.Context, uint) (bool, error) {
	return false, nil
}
func (snapshotHarnessAssignmentRepo) DeleteByWaveAndDocument(context.Context, uint, uint) error {
	return nil
}
func (snapshotHarnessAssignmentRepo) DeleteByWave(context.Context, uint) error { return nil }
func (snapshotHarnessAssignmentRepo) ListByWave(context.Context, uint) ([]domain.WaveDemandAssignment, error) {
	return nil, nil
}
func (snapshotHarnessAssignmentRepo) ListByDemandDocument(context.Context, uint) ([]domain.WaveDemandAssignment, error) {
	return nil, nil
}
func (snapshotHarnessAssignmentRepo) ListDemandDocumentsByWave(context.Context, uint) ([]domain.DemandDocument, error) {
	return nil, nil
}

type snapshotHarnessWaveRepo struct{}

func (snapshotHarnessWaveRepo) Create(context.Context, *domain.Wave) error { return nil }
func (snapshotHarnessWaveRepo) FindByID(context.Context, uint) (*domain.Wave, error) {
	return nil, fmt.Errorf("not found")
}
func (snapshotHarnessWaveRepo) FindByWaveNo(context.Context, string) (*domain.Wave, error) {
	return nil, fmt.Errorf("not found")
}
func (snapshotHarnessWaveRepo) List(context.Context) ([]domain.Wave, error) { return nil, nil }
func (snapshotHarnessWaveRepo) ListPaginated(context.Context, int, int) ([]domain.Wave, int64, error) {
	return nil, 0, nil
}
func (snapshotHarnessWaveRepo) UpdateLifecycle(context.Context, uint, string, string) error {
	return nil
}
func (snapshotHarnessWaveRepo) AddParticipant(context.Context, *domain.WaveParticipantSnapshot) error {
	return nil
}
func (snapshotHarnessWaveRepo) ListParticipantsByWave(context.Context, uint) ([]domain.WaveParticipantSnapshot, error) {
	return nil, nil
}
func (snapshotHarnessWaveRepo) ListParticipantsByProfile(context.Context, uint) ([]domain.WaveParticipantSnapshot, error) {
	return nil, nil
}
func (snapshotHarnessWaveRepo) UpdateParticipantProfileID(context.Context, uint, uint) (int64, error) {
	return 0, nil
}
func (snapshotHarnessWaveRepo) DeleteParticipantsByWave(context.Context, uint) error { return nil }
func (snapshotHarnessWaveRepo) CountByDatePrefix(context.Context, string) (int, error) {
	return 0, nil
}
