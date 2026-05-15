package app

import (
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

const snapshotSchemaVersion = "1"

// WaveSnapshot captures the mutable local state of a wave at a point in time.
// It intentionally excludes fulfillment lines (derived via ApplyRules) and
// external facts (supplier orders, shipments, channel sync jobs).
type WaveSnapshot struct {
	WaveID        uint                           `json:"wave_id"`
	Rules         []domain.AllocationPolicyRule  `json:"rules"`
	Adjustments   []domain.FulfillmentAdjustment `json:"adjustments"`
	Assignments   []domain.WaveDemandAssignment  `json:"assignments"`
	SchemaVersion string                         `json:"schema_version"`
}

// WaveSnapshotService captures and restores wave mutable state for checkpoint-based undo.
type WaveSnapshotService struct {
	ruleRepo       domain.AllocationPolicyRuleRepository
	adjRepo        domain.FulfillmentAdjustmentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
}

func NewWaveSnapshotService(
	ruleRepo domain.AllocationPolicyRuleRepository,
	adjRepo domain.FulfillmentAdjustmentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
) *WaveSnapshotService {
	return &WaveSnapshotService{
		ruleRepo:       ruleRepo,
		adjRepo:        adjRepo,
		assignmentRepo: assignmentRepo,
	}
}

// CaptureSnapshot serializes the wave's current mutable local state to JSON.
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

	snap := WaveSnapshot{
		WaveID:        waveID,
		Rules:         rules,
		Adjustments:   adjs,
		Assignments:   assignments,
		SchemaVersion: snapshotSchemaVersion,
	}

	b, err := json.Marshal(snap)
	if err != nil {
		return "", fmt.Errorf("snapshot: marshal wave %d: %w", waveID, err)
	}
	return string(b), nil
}

// RestoreSnapshot parses a WaveSnapshot JSON and replaces the wave's mutable
// local state with the snapshot contents. Uses hard deletes to avoid ghost
// records conflicting with re-created rows.
func (s *WaveSnapshotService) RestoreSnapshot(payload string) error {
	var snap WaveSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		return fmt.Errorf("snapshot: unmarshal payload: %w", err)
	}

	waveID := snap.WaveID

	if err := s.ruleRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete rules for wave %d: %w", waveID, err)
	}
	if err := s.adjRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete adjustments for wave %d: %w", waveID, err)
	}
	if err := s.assignmentRepo.DeleteByWave(waveID); err != nil {
		return fmt.Errorf("snapshot: delete assignments for wave %d: %w", waveID, err)
	}

	for i := range snap.Rules {
		r := snap.Rules[i]
		r.ID = 0 // let DB assign a new ID; original ID is not meaningful after hard delete
		if err := s.ruleRepo.Create(&r); err != nil {
			return fmt.Errorf("snapshot: restore rule (wave %d): %w", waveID, err)
		}
	}

	for i := range snap.Adjustments {
		a := snap.Adjustments[i]
		a.ID = 0
		if err := s.adjRepo.Create(&a); err != nil {
			return fmt.Errorf("snapshot: restore adjustment (wave %d): %w", waveID, err)
		}
	}

	for i := range snap.Assignments {
		a := snap.Assignments[i]
		a.ID = 0
		if err := s.assignmentRepo.Create(&a); err != nil {
			return fmt.Errorf("snapshot: restore assignment (wave %d): %w", waveID, err)
		}
	}

	return nil
}
