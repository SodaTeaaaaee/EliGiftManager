package app

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

const snapshotSchemaVersion = "2"

// WaveSnapshot captures the mutable local state of a wave at a point in time,
// including participants and fulfillment lines so that GenerateParticipants /
// ApplyAllocationRules can be properly undone.
//
// Schema versions:
//
//	"1" — original format: rules + adjustments + assignments only (no participants/lines)
//	"2" — current format: adds Participants + FulfillmentLines; IDs preserved on restore
type WaveSnapshot struct {
	WaveID           uint                           `json:"wave_id"`
	Rules            []domain.AllocationPolicyRule  `json:"rules"`
	Adjustments      []domain.FulfillmentAdjustment `json:"adjustments"`
	Assignments      []domain.WaveDemandAssignment  `json:"assignments"`
	Participants     []domain.WaveParticipantSnapshot `json:"participants"`
	FulfillmentLines []domain.FulfillmentLine         `json:"fulfillment_lines"`
	SchemaVersion    string                         `json:"schema_version"`
}

// WaveSnapshotService captures and restores wave mutable state for checkpoint-based undo.
type WaveSnapshotService struct {
	db             *gorm.DB
	ruleRepo       domain.AllocationPolicyRuleRepository
	adjRepo        domain.FulfillmentAdjustmentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	waveRepo       domain.WaveRepository
	fulfillRepo    domain.FulfillmentLineRepository
}

func NewWaveSnapshotService(
	db *gorm.DB,
	ruleRepo domain.AllocationPolicyRuleRepository,
	adjRepo domain.FulfillmentAdjustmentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
	waveRepo domain.WaveRepository,
	fulfillRepo domain.FulfillmentLineRepository,
) *WaveSnapshotService {
	return &WaveSnapshotService{
		db:             db,
		ruleRepo:       ruleRepo,
		adjRepo:        adjRepo,
		assignmentRepo: assignmentRepo,
		waveRepo:       waveRepo,
		fulfillRepo:    fulfillRepo,
	}
}

// CaptureSnapshot serializes the wave's current mutable local state to JSON.
// Includes participants and fulfillment lines so undo of GenerateParticipants /
// ApplyAllocationRules fully restores prior state.
func (s *WaveSnapshotService) CaptureSnapshot(waveID uint) (string, error) {
	rules, err := s.ruleRepo.ListByWave(waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list rules for wave %d: %w", waveID, err)
	}

	adjs, err := s.adjRepo.ListByWave(waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list adjustments for wave %d: %w", waveID, err)
	}

	assignments, err := s.assignmentRepo.ListByWave(waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list assignments for wave %d: %w", waveID, err)
	}

	participants, err := s.waveRepo.ListParticipantsByWave(waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list participants for wave %d: %w", waveID, err)
	}

	lines, err := s.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return "", fmt.Errorf("snapshot: list fulfillment lines for wave %d: %w", waveID, err)
	}

	snap := WaveSnapshot{
		WaveID:           waveID,
		Rules:            rules,
		Adjustments:      adjs,
		Assignments:      assignments,
		Participants:     participants,
		FulfillmentLines: lines,
		SchemaVersion:    snapshotSchemaVersion,
	}

	b, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("snapshot: marshal wave %d: %w", waveID, err)
	}
	return string(b), nil
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
func (s *WaveSnapshotService) RestoreSnapshot(payload string) error {
	var snap WaveSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		return fmt.Errorf("snapshot: unmarshal payload: %w", err)
	}

	waveID := snap.WaveID

	// Hard-delete all mutable wave state.  Order matters: fulfillment lines
	// reference participants via WaveParticipantSnapshotID; delete lines first
	// so the participant rows are free to drop.
	if err := s.fulfillRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete fulfillment lines for wave %d: %w", waveID, err)
	}
	if err := s.waveRepo.DeleteParticipantsByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete participants for wave %d: %w", waveID, err)
	}
	if err := s.ruleRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete rules for wave %d: %w", waveID, err)
	}
	if err := s.adjRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete adjustments for wave %d: %w", waveID, err)
	}
	if err := s.assignmentRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete assignments for wave %d: %w", waveID, err)
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
		p := persistence.ToPersistenceWaveParticipantSnapshot(&pt)
		// ToPersistenceWaveParticipantSnapshot already copies d.ID into the struct.
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

	return nil
}
