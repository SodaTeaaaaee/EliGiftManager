package dto

import (
	"encoding/json"
	"time"
)

// ---- AllocationPolicyRule CRUD ----

type AllocationPolicyRuleDTO struct {
	ID                   uint            `json:"id"`
	WaveID               uint            `json:"waveId"`
	ProductID            uint            `json:"productId"`
	SelectorPayload      json.RawMessage `json:"selectorPayload"`
	ProductTargetRef     string          `json:"productTargetRef"`
	ContributionQuantity int             `json:"contributionQuantity"`
	RuleKind             string          `json:"ruleKind"`
	Priority             int             `json:"priority"`
	Active               bool            `json:"active"`
	CreatedAt            time.Time       `json:"createdAt" ts_type:"string"`
	UpdatedAt            time.Time       `json:"updatedAt" ts_type:"string"`
}

type CreateAllocationPolicyRuleInput struct {
	WaveID               uint            `json:"waveId"`
	ProductID            uint            `json:"productId"`
	SelectorPayload      json.RawMessage `json:"selectorPayload"`
	ProductTargetRef     string          `json:"productTargetRef"`
	ContributionQuantity int             `json:"contributionQuantity"`
	RuleKind             string          `json:"ruleKind"`
	Priority             int             `json:"priority"`
	Active               bool            `json:"active"`
}

type UpdateAllocationPolicyRuleInput struct {
	ID                   uint             `json:"id"`
	ProductID            *uint            `json:"productId,omitempty"`
	SelectorPayload      *json.RawMessage `json:"selectorPayload,omitempty"`
	ProductTargetRef     *string          `json:"productTargetRef,omitempty"`
	ContributionQuantity *int             `json:"contributionQuantity,omitempty"`
	RuleKind             *string          `json:"ruleKind,omitempty"`
	Priority             *int             `json:"priority,omitempty"`
	Active               *bool            `json:"active,omitempty"`
}

// ---- Reconcile result ----

type ReconcileResultDTO struct {
	Created       int                `json:"created"`
	Deleted       int                `json:"deleted"`
	ReplayedCount int                `json:"replayedCount"`
	Failures      []ReplayFailureDTO `json:"failures"`
}

type ReplayFailureDTO struct {
	AdjustmentID uint   `json:"adjustmentId"`
	Reason       string `json:"reason"`
}
