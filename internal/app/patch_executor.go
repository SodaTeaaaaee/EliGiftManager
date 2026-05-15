package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

var ErrOperationNotUndoable = errors.New("this operation cannot be undone")

type PatchExecutor struct {
	db *gorm.DB
}

func NewPatchExecutor(db *gorm.DB) *PatchExecutor {
	return &PatchExecutor{db: db}
}

type patchOp struct {
	Op               string          `json:"op"`
	RuleID           uint            `json:"rule_id,omitempty"`
	WaveID           uint            `json:"wave_id,omitempty"`
	AdjustmentID     uint            `json:"adjustment_id,omitempty"`
	DemandDocumentID uint            `json:"demand_document_id,omitempty"`
	Data             json.RawMessage `json:"data,omitempty"`
}

func (pe *PatchExecutor) ApplyPatch(payload string) error {
	if payload == "" {
		return ErrOperationNotUndoable
	}
	var op patchOp
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return fmt.Errorf("patch: invalid payload: %w", err)
	}
	return pe.execute(op)
}

func (pe *PatchExecutor) ApplyInversePatch(payload string) error {
	if payload == "" {
		return ErrOperationNotUndoable
	}
	var op patchOp
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return fmt.Errorf("patch: invalid inverse payload: %w", err)
	}
	return pe.execute(op)
}

func (pe *PatchExecutor) execute(op patchOp) error {
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
	case "generate_participants", "clear_participants",
		"apply_allocation_rules", "clear_allocation_lines":
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
	p := persistence.ToPersistenceAllocationPolicyRule(&rule)
	p.ID = 0
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) deleteRule(op patchOp) error {
	if op.RuleID == 0 {
		return fmt.Errorf("patch: delete_rule missing rule_id")
	}
	return pe.db.Delete(&persistence.AllocationPolicyRule{}, op.RuleID).Error
}

func (pe *PatchExecutor) restoreRule(op patchOp) error {
	if op.Data == nil {
		return ErrOperationNotUndoable
	}
	var rule domain.AllocationPolicyRule
	if err := json.Unmarshal(op.Data, &rule); err != nil {
		return fmt.Errorf("patch: unmarshal rule data: %w", err)
	}
	p := persistence.ToPersistenceAllocationPolicyRule(&rule)
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
	p := persistence.ToPersistenceAllocationPolicyRule(&rule)
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
	p := persistence.FulfillmentAdjustmentFromDomain(&adj)
	p.ID = 0
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) deleteAdjustment(op patchOp) error {
	if op.AdjustmentID == 0 {
		return fmt.Errorf("patch: delete_adjustment missing adjustment_id")
	}
	return pe.db.Delete(&persistence.FulfillmentAdjustment{}, op.AdjustmentID).Error
}

func (pe *PatchExecutor) assignDemand(op patchOp) error {
	if op.WaveID == 0 || op.DemandDocumentID == 0 {
		return fmt.Errorf("patch: assign_demand missing wave_id or demand_document_id")
	}
	p := &persistence.WaveDemandAssignment{
		WaveID:           op.WaveID,
		DemandDocumentID: op.DemandDocumentID,
	}
	return pe.db.Create(p).Error
}

func (pe *PatchExecutor) unassignDemand(op patchOp) error {
	if op.WaveID == 0 || op.DemandDocumentID == 0 {
		return fmt.Errorf("patch: unassign_demand missing wave_id or demand_document_id")
	}
	return pe.db.Where("wave_id = ? AND demand_document_id = ?", op.WaveID, op.DemandDocumentID).
		Delete(&persistence.WaveDemandAssignment{}).Error
}
