package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

var ErrOperationNotUndoable = errors.New("this operation cannot be undone")

type PatchExecutor struct {
	db          *gorm.DB
	snapshotSvc *WaveSnapshotService
}

func NewPatchExecutor(db *gorm.DB, snapshotSvc ...*WaveSnapshotService) *PatchExecutor {
	pe := &PatchExecutor{db: db}
	if len(snapshotSvc) > 0 && snapshotSvc[0] != nil {
		pe.snapshotSvc = snapshotSvc[0]
	}
	return pe
}

type patchOp struct {
	Op                string          `json:"op"`
	RuleID            uint            `json:"rule_id,omitempty"`
	WaveID            uint            `json:"wave_id,omitempty"`
	AdjustmentID      uint            `json:"adjustment_id,omitempty"`
	DemandDocumentID  uint            `json:"demand_document_id,omitempty"`
	DemandDocumentIDs []uint          `json:"demand_document_ids"`
	Data              json.RawMessage `json:"data,omitempty"`
}

func (pe *PatchExecutor) ApplyPatch(ctx context.Context, payload string) error {
	if payload == "" {
		return ErrOperationNotUndoable
	}
	var op patchOp
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return fmt.Errorf("patch: invalid payload: %w", err)
	}
	return pe.execute(ctx, op)
}

func (pe *PatchExecutor) ApplyInversePatch(ctx context.Context, payload string) error {
	if payload == "" {
		return ErrOperationNotUndoable
	}
	var op patchOp
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return fmt.Errorf("patch: invalid inverse payload: %w", err)
	}
	return pe.execute(ctx, op)
}

func (pe *PatchExecutor) execute(ctx context.Context, op patchOp) error {
	switch op.Op {
	case "create_rule":
		return pe.createRule(op)
	case "delete_rule":
		return pe.deleteRule(op)
	case "restore_rule":
		return pe.restoreRule(op)
	case "update_rule":
		return pe.updateRule(op)
	case "record_adjustment":
		return pe.createAdjustment(op)
	case "delete_adjustment":
		return pe.deleteAdjustment(op)
	case "assign_demand":
		return pe.assignDemand(op)
	case "unassign_demand":
		return pe.unassignDemand(op)
	case "batch_assign_demand":
		return pe.batchAssignDemand(op)
	case "batch_unassign_demand":
		return pe.batchUnassignDemand(op)
	case "restore_checkpoint":
		return pe.restoreCheckpoint(ctx, op)
	case "generate_participants", "clear_participants",
		"map_demand_lines", "clear_allocation_lines":
		return ErrOperationNotUndoable
	default:
		return fmt.Errorf("patch: unknown op %q", op.Op)
	}
}

func (pe *PatchExecutor) createRule(op patchOp) error {
	if op.Data == nil {
		return ErrOperationNotUndoable
	}
	var rule domain.AllocationPolicyRule
	if err := json.Unmarshal(op.Data, &rule); err != nil {
		return fmt.Errorf("patch: unmarshal rule data: %w", err)
	}
	// Preserve the original ID from patch data so that downstream references
	// (history node patches, basis pins) stored at record time remain valid.
	// The row was hard-deleted by the inverse op so there is no ID conflict.
	selectorJSON, _ := json.Marshal(rule.SelectorPayload)
	p := &persistence.AllocationPolicyRule{
		WaveID:               rule.WaveID,
		ProductID:            rule.ProductID,
		SelectorPayload:      string(selectorJSON),
		ProductTargetRef:     rule.ProductTargetRef,
		ContributionQuantity: rule.ContributionQuantity,
		RuleKind:             rule.RuleKind,
		Priority:             rule.Priority,
		Active:               rule.Active,
	}
	p.ID = rule.ID
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) deleteRule(op patchOp) error {
	if op.RuleID == 0 {
		return fmt.Errorf("patch: delete_rule missing rule_id")
	}
	// Use Unscoped so that the row is hard-deleted; the paired restore_rule /
	// create_rule op re-inserts with the original ID, which requires the row to
	// be fully gone (not merely soft-deleted).
	return pe.db.Unscoped().Delete(&persistence.AllocationPolicyRule{}, op.RuleID).Error
}

func (pe *PatchExecutor) restoreRule(op patchOp) error {
	if op.Data == nil {
		return ErrOperationNotUndoable
	}
	var rule domain.AllocationPolicyRule
	if err := json.Unmarshal(op.Data, &rule); err != nil {
		return fmt.Errorf("patch: unmarshal rule data: %w", err)
	}
	// Preserve the original ID (undo-of-delete path); the row was hard-deleted
	// by deleteRule so there is no conflict.
	selectorJSON, _ := json.Marshal(rule.SelectorPayload)
	p := &persistence.AllocationPolicyRule{
		WaveID:               rule.WaveID,
		ProductID:            rule.ProductID,
		SelectorPayload:      string(selectorJSON),
		ProductTargetRef:     rule.ProductTargetRef,
		ContributionQuantity: rule.ContributionQuantity,
		RuleKind:             rule.RuleKind,
		Priority:             rule.Priority,
		Active:               rule.Active,
	}
	p.ID = rule.ID
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) updateRule(op patchOp) error {
	if op.Data == nil || op.RuleID == 0 {
		return ErrOperationNotUndoable
	}
	var rule domain.AllocationPolicyRule
	if err := json.Unmarshal(op.Data, &rule); err != nil {
		return fmt.Errorf("patch: unmarshal rule data: %w", err)
	}
	p, err := persistence.AllocationPolicyRuleFromDomain(&rule)
	if err != nil {
		return fmt.Errorf("patch: convert rule: %w", err)
	}
	p.ID = op.RuleID
	return pe.db.Save(p).Error
}

func (pe *PatchExecutor) createAdjustment(op patchOp) error {
	if op.Data == nil {
		return ErrOperationNotUndoable
	}
	var adj domain.FulfillmentAdjustment
	if err := json.Unmarshal(op.Data, &adj); err != nil {
		return fmt.Errorf("patch: unmarshal adjustment data: %w", err)
	}
	// Preserve the original ID so that FulfillmentLineID back-references remain
	// stable; FulfillmentAdjustmentFromDomain already embeds it via gorm.Model{ID}.
	p := persistence.FulfillmentAdjustmentFromDomain(&adj)
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) deleteAdjustment(op patchOp) error {
	if op.AdjustmentID == 0 {
		return fmt.Errorf("patch: delete_adjustment missing adjustment_id")
	}
	// Hard delete so the original ID is free for re-insertion by the paired
	// record_adjustment / create_adjustment op.
	return pe.db.Unscoped().Delete(&persistence.FulfillmentAdjustment{}, op.AdjustmentID).Error
}

func (pe *PatchExecutor) assignDemand(op patchOp) error {
	if op.WaveID == 0 || op.DemandDocumentID == 0 {
		return fmt.Errorf("patch: assign_demand missing wave_id or demand_document_id")
	}
	return pe.assignDemandTo(pe.db, op.WaveID, op.DemandDocumentID)
}

// assignDemandTo creates the single wave-demand linkage row against db — the
// shared executor DB for single-item ops, or a transaction handle for batch ops.
//
// The use-case side deletes unassigned rows via a soft delete, which still
// occupies the (wave_id, demand_document_id) unique index slot (idx_wave_demand
// carries no deleted_at column). Purge that residue with a hard delete first so
// the re-created row does not collide — same treatment as batchAssignDemand.
func (pe *PatchExecutor) assignDemandTo(db *gorm.DB, waveID, demandDocumentID uint) error {
	if err := db.Unscoped().
		Where("wave_id = ? AND demand_document_id = ?", waveID, demandDocumentID).
		Delete(&persistence.WaveDemandAssignment{}).Error; err != nil {
		return err
	}
	p := &persistence.WaveDemandAssignment{
		WaveID:           waveID,
		DemandDocumentID: demandDocumentID,
	}
	return db.Create(p).Error
}

func (pe *PatchExecutor) unassignDemand(op patchOp) error {
	if op.WaveID == 0 || op.DemandDocumentID == 0 {
		return fmt.Errorf("patch: unassign_demand missing wave_id or demand_document_id")
	}
	return pe.unassignDemandFrom(pe.db, op.WaveID, op.DemandDocumentID)
}

// unassignDemandFrom deletes the single wave-demand linkage row against db —
// the shared executor DB for single-item ops, or a transaction handle for batch ops.
func (pe *PatchExecutor) unassignDemandFrom(db *gorm.DB, waveID, demandDocumentID uint) error {
	return db.Where("wave_id = ? AND demand_document_id = ?", waveID, demandDocumentID).
		Delete(&persistence.WaveDemandAssignment{}).Error
}

// batchAssignDemand replays the inverse of a batch unassign: every listed
// document is re-linked to the wave inside ONE transaction so a mid-batch
// failure cannot leave the undo half-applied.
//
// The use-case side deletes unassigned rows via a soft delete, which still
// occupies the (wave_id, demand_document_id) unique index slot. Purge that
// residue with a hard delete first so the re-created row does not collide.
func (pe *PatchExecutor) batchAssignDemand(op patchOp) error {
	if op.WaveID == 0 || len(op.DemandDocumentIDs) == 0 {
		return fmt.Errorf("patch: batch_assign_demand missing wave_id or demand_document_ids")
	}
	return pe.db.Transaction(func(tx *gorm.DB) error {
		for _, docID := range op.DemandDocumentIDs {
			if docID == 0 {
				return fmt.Errorf("patch: batch_assign_demand contains demand_document_id 0")
			}
			if err := tx.Unscoped().
				Where("wave_id = ? AND demand_document_id = ?", op.WaveID, docID).
				Delete(&persistence.WaveDemandAssignment{}).Error; err != nil {
				return err
			}
			if err := pe.assignDemandTo(tx, op.WaveID, docID); err != nil {
				return err
			}
		}
		return nil
	})
}

// batchUnassignDemand replays the forward of a batch unassign: every listed
// document is unlinked from the wave inside ONE transaction. Rows are
// hard-deleted (matching the delete_rule / delete_adjustment pattern) so the
// unique index slot is free for a later undo's re-insertion.
func (pe *PatchExecutor) batchUnassignDemand(op patchOp) error {
	if op.WaveID == 0 || len(op.DemandDocumentIDs) == 0 {
		return fmt.Errorf("patch: batch_unassign_demand missing wave_id or demand_document_ids")
	}
	return pe.db.Transaction(func(tx *gorm.DB) error {
		for _, docID := range op.DemandDocumentIDs {
			if docID == 0 {
				return fmt.Errorf("patch: batch_unassign_demand contains demand_document_id 0")
			}
			if err := tx.Unscoped().
				Where("wave_id = ? AND demand_document_id = ?", op.WaveID, docID).
				Delete(&persistence.WaveDemandAssignment{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (pe *PatchExecutor) restoreCheckpoint(ctx context.Context, op patchOp) error {
	if pe.snapshotSvc == nil {
		return fmt.Errorf("patch: restore_checkpoint requires snapshot service")
	}
	if op.Data == nil {
		return fmt.Errorf("patch: restore_checkpoint missing data")
	}
	// op.Data contains the raw snapshot JSON string (quoted)
	var payload string
	if err := json.Unmarshal(op.Data, &payload); err != nil {
		// op.Data may already be the unquoted JSON object
		payload = string(op.Data)
	}
	return pe.snapshotSvc.RestoreSnapshot(ctx, payload)
}
