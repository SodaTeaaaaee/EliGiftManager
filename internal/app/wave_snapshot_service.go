package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// ErrRestoreBlockedByNonVoidedShipments is returned by RestoreSnapshot when the
// wave still has live (non-voided) shipments. WaveSnapshot does not include
// shipments, so DeleteByWave of fulfillment lines would leave shipment_lines
// pointing at deleted FulfillmentLineIDs.
var ErrRestoreBlockedByNonVoidedShipments = errors.New("snapshot: cannot restore wave while non-voided shipments exist")

const snapshotSchemaVersion = "3"

// WaveSnapshot captures the mutable local state of a wave at a point in time,
// including participants and fulfillment lines so that GenerateParticipants /
// ApplyAllocationRules can be properly undone.
//
// Schema versions:
//
//	"1" — original format: rules + adjustments + assignments only (no participants/lines)
//	"2" — adds Participants + FulfillmentLines; IDs preserved on restore
//	"3" — adds ChannelClosureDecisionRecords to the mutable local wave snapshot
type WaveSnapshot struct {
	WaveID           uint                                  `json:"wave_id"`
	Rules            []domain.AllocationPolicyRule         `json:"rules"`
	Adjustments      []domain.FulfillmentAdjustment        `json:"adjustments"`
	Assignments      []domain.WaveDemandAssignment         `json:"assignments"`
	Participants     []domain.WaveParticipantSnapshot      `json:"participants"`
	FulfillmentLines []domain.FulfillmentLine              `json:"fulfillment_lines"`
	ClosureDecisions []domain.ChannelClosureDecisionRecord `json:"closure_decisions,omitempty"`
	SchemaVersion    string                                `json:"schema_version"`
}

// WaveSnapshotService captures and restores wave mutable state for checkpoint-based undo.
type WaveSnapshotService struct {
	db             *gorm.DB
	ruleRepo       domain.AllocationPolicyRuleRepository
	adjRepo        domain.FulfillmentAdjustmentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	waveRepo       domain.WaveRepository
	fulfillRepo    domain.FulfillmentLineRepository
	closureRepo    domain.ChannelClosureDecisionRepository
}

func NewWaveSnapshotService(
	db *gorm.DB,
	ruleRepo domain.AllocationPolicyRuleRepository,
	adjRepo domain.FulfillmentAdjustmentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
	waveRepo domain.WaveRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	closureRepo ...domain.ChannelClosureDecisionRepository,
) *WaveSnapshotService {
	var cr domain.ChannelClosureDecisionRepository
	if len(closureRepo) > 0 {
		cr = closureRepo[0]
	}
	return &WaveSnapshotService{
		db:             db,
		ruleRepo:       ruleRepo,
		adjRepo:        adjRepo,
		assignmentRepo: assignmentRepo,
		waveRepo:       waveRepo,
		fulfillRepo:    fulfillRepo,
		closureRepo:    cr,
	}
}

// CaptureSnapshot serializes the wave's current mutable local state to JSON.
// Includes participants and fulfillment lines so undo of GenerateParticipants /
// ApplyAllocationRules fully restores prior state.
func (s *WaveSnapshotService) CaptureSnapshot(ctx context.Context, waveID uint) (string, error) {
	rules, err := s.ruleRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list rules for wave %d: %w", waveID, err)
	}

	adjs, err := s.adjRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list adjustments for wave %d: %w", waveID, err)
	}

	assignments, err := s.assignmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list assignments for wave %d: %w", waveID, err)
	}

	participants, err := s.waveRepo.ListParticipantsByWave(ctx, waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list participants for wave %d: %w", waveID, err)
	}

	lines, err := s.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list fulfillment lines for wave %d: %w", waveID, err)
	}

	var decisions []domain.ChannelClosureDecisionRecord
	if s.closureRepo != nil {
		decisions, err = s.closureRepo.ListByWave(ctx, waveID)
		if err != nil {
			return "", fmt.Errorf("snapshot: list closure decisions for wave %d: %w", waveID, err)
		}
	}

	snap := WaveSnapshot{
		WaveID:           waveID,
		Rules:            rules,
		Adjustments:      adjs,
		Assignments:      assignments,
		Participants:     participants,
		FulfillmentLines: lines,
		ClosureDecisions: decisions,
		SchemaVersion:    snapshotSchemaVersion,
	}

	b, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("snapshot: marshal wave %d: %w", waveID, err)
	}
	return string(b), nil
}

// CaptureSnapshotForSupplierOrder resolves the wave from a supplier order and
// captures the corresponding wave-local snapshot.
func (s *WaveSnapshotService) CaptureSnapshotForSupplierOrder(ctx context.Context, supplierOrderID uint) (string, error) {
	var order persistence.SupplierOrder
	if err := s.db.First(&order, supplierOrderID).Error; err != nil {
		return "", fmt.Errorf("snapshot: find supplier order %d: %w", supplierOrderID, err)
	}
	return s.CaptureSnapshot(ctx, order.WaveID)
}

// RestoreSnapshot parses a WaveSnapshot JSON and replaces the wave's mutable
// local state with the snapshot contents.
//
// ID preservation: original row IDs from the snapshot are kept on re-insert so
// that downstream references (history node patches, basis pins, FKs from
// FulfillmentAdjustment.FulfillmentLineID) remain valid after undo.
//
// Hard deletes are used throughout so that the original IDs are fully freed
// before re-insertion; GORM's Create with a non-zero primary key then uses the
// specified value directly in SQLite.
//
// Restore is refused when the wave has non-voided shipments: snapshots do not
// include shipments, and deleting fulfillment lines would orphan shipment_lines.
func (s *WaveSnapshotService) RestoreSnapshot(ctx context.Context, payload string) error {
	var snap WaveSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		return fmt.Errorf("snapshot: unmarshal payload: %w", err)
	}

	waveID := snap.WaveID

	// SQLite :memory: pools treat each connection as a separate empty database.
	// Pin to one connection so the shipment guard and DeleteByWave see the
	// same migrated schema. Production InitDB already uses MaxOpenConns(1).
	if s.db != nil {
		if sqlDB, err := s.db.DB(); err == nil && sqlDB != nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}

	if err := s.refuseRestoreIfNonVoidedShipments(ctx, waveID); err != nil {
		return err
	}

	// Hard-delete all mutable wave state.  Order matters: fulfillment lines
	// reference participants via WaveParticipantSnapshotID; delete lines first
	// so the participant rows are free to drop.
	if err := s.fulfillRepo.DeleteByWave(ctx, waveID); err != nil {
		return fmt.Errorf("snapshot: delete fulfillment lines for wave %d: %w", waveID, err)
	}
	if err := s.waveRepo.DeleteParticipantsByWave(ctx, waveID); err != nil {
		return fmt.Errorf("snapshot: delete participants for wave %d: %w", waveID, err)
	}
	if err := s.ruleRepo.DeleteByWave(ctx, waveID); err != nil {
		return fmt.Errorf("snapshot: delete rules for wave %d: %w", waveID, err)
	}
	if err := s.adjRepo.DeleteByWave(ctx, waveID); err != nil {
		return fmt.Errorf("snapshot: delete adjustments for wave %d: %w", waveID, err)
	}
	if err := s.assignmentRepo.DeleteByWave(ctx, waveID); err != nil {
		return fmt.Errorf("snapshot: delete assignments for wave %d: %w", waveID, err)
	}
	if s.closureRepo != nil {
		records, err := s.closureRepo.ListByWave(ctx, waveID)
		if err != nil {
			return fmt.Errorf("snapshot: list closure decisions for delete in wave %d: %w", waveID, err)
		}
		for i := range records {
			if err := s.db.Unscoped().Delete(&persistence.ChannelClosureDecisionRecord{}, records[i].ID).Error; err != nil {
				return fmt.Errorf("snapshot: delete closure decision %d (wave %d): %w", records[i].ID, waveID, err)
			}
		}
	}

	// Re-insert with original IDs preserved.  Because the rows were hard-deleted
	// above there are no ID conflicts; GORM/SQLite honours the non-zero ID in the
	// INSERT statement.

	for i := range snap.Rules {
		r := snap.Rules[i]
		// Build the persistence model directly so we can carry the original ID
		// through — ToPersistenceAllocationPolicyRule strips the ID field.
		selectorJSON, _ := json.Marshal(r.SelectorPayload)
		p := &persistence.AllocationPolicyRule{
			WaveID:               r.WaveID,
			ProductID:            r.ProductID,
			SelectorPayload:      string(selectorJSON),
			ProductTargetRef:     r.ProductTargetRef,
			ContributionQuantity: r.ContributionQuantity,
			RuleKind:             r.RuleKind,
			Priority:             r.Priority,
			Active:               r.Active,
		}
		p.ID = r.ID // preserve original ID; see doc comment above
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore rule %d (wave %d): %w", r.ID, waveID, err)
		}
	}

	for i := range snap.Adjustments {
		a := snap.Adjustments[i]
		p := persistence.FulfillmentAdjustmentFromDomain(&a) // already carries ID via gorm.Model{ID: a.ID}
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore adjustment %d (wave %d): %w", a.ID, waveID, err)
		}
	}

	for i := range snap.Assignments {
		a := snap.Assignments[i]
		p := &persistence.WaveDemandAssignment{
			WaveID:           a.WaveID,
			DemandDocumentID: a.DemandDocumentID,
			AcceptedBy:       a.AcceptedBy,
			ExtraData:        a.ExtraData,
		}
		p.ID = a.ID // preserve original ID
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore assignment %d (wave %d): %w", a.ID, waveID, err)
		}
	}

	for i := range snap.Participants {
		pt := snap.Participants[i]
		p := persistence.WaveParticipantSnapshotFromDomain(&pt)
		// WaveParticipantSnapshotFromDomain already copies d.ID into the struct.
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore participant %d (wave %d): %w", pt.ID, waveID, err)
		}
	}

	// Re-insert fulfillment lines last; they may reference participant IDs.
	for i := range snap.FulfillmentLines {
		fl := snap.FulfillmentLines[i]
		p := &persistence.FulfillmentLine{
			WaveID:                    fl.WaveID,
			CustomerProfileID:         fl.CustomerProfileID,
			WaveParticipantSnapshotID: fl.WaveParticipantSnapshotID,
			ProductID:                 fl.ProductID,
			DemandDocumentID:          fl.DemandDocumentID,
			DemandLineID:              fl.DemandLineID,
			CustomerAddressID:         fl.CustomerAddressID,
			Quantity:                  fl.Quantity,
			AllocationState:           fl.AllocationState,
			AddressState:              fl.AddressState,
			SupplierState:             fl.SupplierState,
			ChannelSyncState:          fl.ChannelSyncState,
			LineReason:                persistence.FulfillmentLineReason(fl.LineReason),
			GeneratedBy:               fl.GeneratedBy,
			ExtraData:                 fl.ExtraData,
		}
		p.ID = fl.ID // preserve original ID
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore fulfillment line %d (wave %d): %w", fl.ID, waveID, err)
		}
	}

	for i := range snap.ClosureDecisions {
		cd := snap.ClosureDecisions[i]
		p := persistence.ChannelClosureDecisionRecordFromDomain(&cd)
		p.ID = cd.ID
		if err := s.db.Create(p).Error; err != nil {
			return fmt.Errorf("snapshot: restore closure decision %d (wave %d): %w", cd.ID, waveID, err)
		}
	}

	return nil
}

// refuseRestoreIfNonVoidedShipments blocks undo when live shipments exist.
// WaveSnapshot cannot be extended to shipments here without also restoring
// shipment_lines (and related FKs) atomically; until that exists, restore
// would DeleteByWave fulfillment lines and leave dangling FulfillmentLineIDs.
func (s *WaveSnapshotService) refuseRestoreIfNonVoidedShipments(ctx context.Context, waveID uint) error {
	if s.db == nil {
		return nil
	}

	var count int64
	err := s.db.WithContext(ctx).
		Model(&persistence.Shipment{}).
		Joins("JOIN supplier_orders ON supplier_orders.id = shipments.supplier_order_id").
		Where("supplier_orders.wave_id = ?", waveID).
		Where("shipments.status <> ?", string(domain.ShipmentStatusVoided)).
		Count(&count).Error
	if err != nil {
		// History fixtures (and schema v1/v2 DBs) may not have shipment tables.
		// Do not use Migrator().HasTable: it can open a second SQLite :memory:
		// connection and make the original schema invisible to later queries.
		if isNoSuchTable(err) {
			return nil
		}
		return fmt.Errorf("snapshot: count non-voided shipments for wave %d: %w", waveID, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: wave %d has %d non-voided shipment(s); snapshots do not include shipments",
			ErrRestoreBlockedByNonVoidedShipments, waveID, count)
	}
	return nil
}

func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table")
}
