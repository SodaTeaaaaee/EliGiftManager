package domain

import (
	"context"
	"time"
)

const (
	MergeRecordStatusExecuting = "executing"
	MergeRecordStatusCompleted = "completed"
	MergeRecordStatusUndone    = "undone"

	MergeAuditLevelExact  = "exact"
	MergeAuditLevelLegacy = "legacy"

	MergeEntityIdentity        = "identity"
	MergeEntityAddress         = "address"
	MergeEntityDemandDocument  = "demand_document"
	MergeEntityNameObservation = "name_observation"
	MergeEntityNameEvent       = "name_event"
	MergeEntityOrigin          = "origin"
	MergeEntityProfile         = "profile"

	MergeMutationReassign          = "reassign"
	MergeMutationProfileState      = "profile_state"
	MergeMutationDisplayProjection = "display_projection"

	MergeRestoreOwnerOnly = "owner_only"
	MergeRestoreExact     = "exact"
)

// MergeEntityState is the canonical, deliberately narrow snapshot used by the
// immutable moved-row ledger. It contains only merge-owned fields so unrelated
// post-merge edits (for example an address label) remain intact on undo.
type MergeEntityState struct {
	ProfileID                *uint  `json:"profileId,omitempty"`
	IsPrimary                *bool  `json:"isPrimary,omitempty"`
	IsDefault                *bool  `json:"isDefault,omitempty"`
	Status                   string `json:"status,omitempty"`
	MergedIntoProfileID      *uint  `json:"mergedIntoProfileId,omitempty"`
	RowVersion               uint64 `json:"rowVersion,omitempty"`
	DisplayName              string `json:"displayName,omitempty"`
	DisplayNameMode          string `json:"displayNameMode,omitempty"`
	DisplayNameObservationID *uint  `json:"displayNameObservationId,omitempty"`
	SoftDeleted              *bool  `json:"softDeleted,omitempty"`
}

type CustomerMergeOperationEvent struct {
	ID            uint
	MergeRecordID *uint
	EventKey      string
	OperationKey  string
	EventType     string
	Status        string
	ActorRef      string
	ReasonCode    string
	Payload       string
	CreatedAt     time.Time
}

type MergeHistoryFilter struct {
	ProfileID       uint
	CandidateID     uint
	Status          string
	BeforeCreatedAt *time.Time
	BeforeID        uint
	Limit           int
}

// MergeExecutionStore is additive and transaction-agnostic. Controllers bind
// the concrete implementation either to the main DB for reads or to a GORM
// transaction for Execute/Undo; the application layer never imports GORM.
type MergeExecutionStore interface {
	FindProfileForMerge(ctx context.Context, id uint, includeDeleted bool) (*CustomerProfile, error)
	ListIdentitiesForMerge(ctx context.Context, profileID uint) ([]CustomerIdentity, error)
	ListAddressesForMerge(ctx context.Context, profileID uint) ([]CustomerAddress, error)
	ListDemandForMerge(ctx context.Context, profileID uint) ([]DemandDocument, error)
	ListNameObservationsForMerge(ctx context.Context, profileID uint) ([]CustomerNameObservation, error)
	ListNameEventsForMerge(ctx context.Context, profileID uint) ([]CustomerNameEvent, error)
	ListOriginsForMerge(ctx context.Context, profileID uint) ([]CustomerProfileOrigin, error)

	FindCandidateExecutionContext(ctx context.Context, id uint) (*MergeCandidate, []MergeEvidence, *MergePolicyRevision, error)
	FindMergeRecord(ctx context.Context, id uint) (*CustomerMergeRecord, error)
	FindMergeRecordByOperationKey(ctx context.Context, operationKey string) (*CustomerMergeRecord, error)
	FindMergeRecordByUndoOperationKey(ctx context.Context, operationKey string) (*CustomerMergeRecord, error)
	ListActiveMergeRecords(ctx context.Context, profileIDs []uint) ([]CustomerMergeRecord, error)
	ListMergeRecords(ctx context.Context, filter MergeHistoryFilter) ([]CustomerMergeRecord, error)

	CreateMergeRecord(ctx context.Context, record *CustomerMergeRecord) error
	CompleteMergeRecord(ctx context.Context, record *CustomerMergeRecord) (bool, error)
	CreateMovedEntities(ctx context.Context, moved []MergeMovedEntity) error
	ListMovedEntities(ctx context.Context, mergeRecordID uint) ([]MergeMovedEntity, error)
	CurrentEntityState(ctx context.Context, moved MergeMovedEntity) (MergeEntityState, error)
	ApplyEntityState(ctx context.Context, moved MergeMovedEntity, state MergeEntityState) error
	MarkMovedEntitiesReverted(ctx context.Context, mergeRecordID uint, operationKey string, revertedAt time.Time) error

	MarkCandidateExecuted(ctx context.Context, candidateID uint, expectedRowVersion uint64, evidenceHash string, policyVersion, policyRevisionID, mergeRecordID uint, at time.Time) (bool, error)
	InvalidateCandidatesAfterMerge(ctx context.Context, sourceProfileID, targetProfileID, executedCandidateID uint) error
	MarkCandidateStaleAfterUndo(ctx context.Context, candidateID uint, mergeRecordID uint) error
	MarkPolicyNeedsScan(ctx context.Context, policyRevisionID *uint) error

	IsDemandDocumentAssigned(ctx context.Context, documentID uint) (bool, error)
	MarkMergeUndone(ctx context.Context, recordID uint, expectedRowVersion uint64, operationKey, undoPlanHash, actorRef, reason string, sourceVersion, targetVersion uint64, at time.Time) (bool, error)
	CreateMergeOperationEvent(ctx context.Context, event *CustomerMergeOperationEvent) error
	ListMergeOperationEvents(ctx context.Context, mergeRecordID uint) ([]CustomerMergeOperationEvent, error)
}
