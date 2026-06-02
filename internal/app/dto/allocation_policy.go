package dto

import "encoding/json"

// ---- AllocationPolicyRule CRUD ----

type AllocationPolicyRuleDTO struct {
	ID                   uint            `json:"id"`
	WaveID               uint            `json:"wave_id"`
	ProductID            uint            `json:"product_id"`
	SelectorPayload      json.RawMessage `json:"selector_payload"`
	ProductTargetRef     string          `json:"product_target_ref"`
	ContributionQuantity int             `json:"contribution_quantity"`
	RuleKind             string          `json:"rule_kind"`
	Priority             int             `json:"priority"`
	Active               bool            `json:"active"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
}

type CreateAllocationPolicyRuleInput struct {
	WaveID               uint            `json:"wave_id"`
	ProductID            uint            `json:"product_id"`
	SelectorPayload      json.RawMessage `json:"selector_payload"`
	ProductTargetRef     string          `json:"product_target_ref"`
	ContributionQuantity int             `json:"contribution_quantity"`
	RuleKind             string          `json:"rule_kind"`
	Priority             int             `json:"priority"`
	Active               bool            `json:"active"`
}

type UpdateAllocationPolicyRuleInput struct {
	ID                   uint             `json:"id"`
	ProductID            *uint            `json:"product_id,omitempty"`
	SelectorPayload      *json.RawMessage `json:"selector_payload,omitempty"`
	ProductTargetRef     *string          `json:"product_target_ref,omitempty"`
	ContributionQuantity *int             `json:"contribution_quantity,omitempty"`
	RuleKind             *string          `json:"rule_kind,omitempty"`
	Priority             *int             `json:"priority,omitempty"`
	Active               *bool            `json:"active,omitempty"`
}

// ---- Reconcile result ----

type ReconcileResultDTO struct {
	Created       int                `json:"created"`
	Deleted       int                `json:"deleted"`
	ReplayedCount int                `json:"replayed_count"`
	Failures      []ReplayFailureDTO `json:"failures"`
}

type ReplayFailureDTO struct {
	AdjustmentID uint   `json:"adjustment_id"`
	Reason       string `json:"reason"`
}
