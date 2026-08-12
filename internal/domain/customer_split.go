package domain

import (
	"context"
	"time"
)

const (
	SplitRecordStatusExecuting = "executing"
	SplitRecordStatusCompleted = "completed"

	SplitTargetStrategyCreateNew     = "create_new"
	SplitTargetStrategyRestoreMerged = "restore_merged"

	SplitReverseOperationManualMerge = "manual_merge_required"

	SplitMutationReassign          = "split_reassign"
	SplitMutationPrimaryProjection = "split_primary_projection"
	SplitMutationDefaultProjection = "split_default_projection"
	SplitMutationSourceProjection  = "split_source_projection"
	SplitMutationTargetCreated     = "split_target_created"
)

// SplitEntityState captures every split-owned field used for optimistic
// concurrency checks and the immutable before/after ledger. Unrelated fields
// remain outside this snapshot and are never overwritten by split execution.
type SplitEntityState struct {
	Exists                   bool   `json:"exists"`
	ProfileID                *uint  `json:"profileId,omitempty"`
	IsPrimary                *bool  `json:"isPrimary,omitempty"`
	IsDefault                *bool  `json:"isDefault,omitempty"`
	Status                   string `json:"status,omitempty"`
	MergedIntoProfileID      *uint  `json:"mergedIntoProfileId,omitempty"`
	RowVersion               uint64 `json:"rowVersion,omitempty"`
	DisplayName              string `json:"displayName,omitempty"`
	DisplayNameMode          string `json:"displayNameMode,omitempty"`
	DisplayNameObservationID *uint  `json:"displayNameObservationId,omitempty"`
}

type CustomerSplitRecord struct {
	ID                    uint
	OperationKey          string
	CommandHash           string
	PreviewHash           string
	MovePlanHash          string
	Status                string
	SourceProfileID       uint
	TargetProfileID       uint
	TargetStrategy        string
	ActorRef              string
	DecisionReason        string
	SourceRowVersion      uint64
	TargetRowVersion      uint64
	SourceRowVersionAfter uint64
	TargetRowVersionAfter uint64
	SourceProfileSnapshot string
	TargetProfileSnapshot string
	Payload               string
	RowVersion            uint64
	ReverseOperationKind  string
	CreatedAt             time.Time
	CompletedAt           *time.Time
}

type SplitMovedEntity struct {
	ID              uint
	SplitRecordID   uint
	EntityType      string
	EntityID        uint
	FromProfileID   uint
	ToProfileID     uint
	MoveOrder       uint
	BeforeSnapshot  string
	AfterSnapshot   string
	AfterStateHash  string
	MutationKind    string
	SnapshotVersion uint
	CreatedAt       time.Time
}

type CustomerSplitOperationEvent struct {
	ID            uint
	SplitRecordID uint
	EventKey      string
	OperationKey  string
	EventType     string
	Status        string
	ActorRef      string
	ReasonCode    string
	Payload       string
	CreatedAt     time.Time
}

type SplitImmutableHistoryRefs struct {
	WaveParticipantSnapshotIDs []uint
	FulfillmentLineIDs         []uint
}

type SplitHistoryFilter struct {
	ProfileID       uint
	Status          string
	BeforeCreatedAt *time.Time
	BeforeID        uint
	Limit           int
}

type SplitIdentityKey struct {
	Namespace       string
	IdentityType    string
	NormalizedValue string
}

// SplitExecutionStore is transaction-agnostic. Preview/history bind it to the
// main DB; Execute binds it to the controller-owned transaction.
type SplitExecutionStore interface {
	FindProfileForSplit(ctx context.Context, id uint) (*CustomerProfile, error)
	CreateSplitTargetProfile(ctx context.Context, profile *CustomerProfile) error
	ListIdentitiesForSplit(ctx context.Context, profileID uint) ([]CustomerIdentity, error)
	ListAddressesForSplit(ctx context.Context, profileID uint) ([]CustomerAddress, error)
	ListDemandForSplit(ctx context.Context, profileID uint) ([]DemandDocument, error)
	ListNameObservationsForSplit(ctx context.Context, profileID uint) ([]CustomerNameObservation, error)
	ListNameEventsForSplit(ctx context.Context, profileID uint) ([]CustomerNameEvent, error)
	ListOriginsForSplit(ctx context.Context, profileID uint) ([]CustomerProfileOrigin, error)
	ListActiveMergeRecordsForSplit(ctx context.Context, profileID uint) ([]CustomerMergeRecord, error)
	ListImmutableHistoryRefsForSplit(ctx context.Context, profileID uint) (SplitImmutableHistoryRefs, error)
	ListStrongIdentityOwnerIDs(ctx context.Context, key SplitIdentityKey, excludingProfileID uint) ([]uint, error)
	IsDemandDocumentFrozenForSplit(ctx context.Context, documentID uint) (bool, error)

	CreateSplitRecord(ctx context.Context, record *CustomerSplitRecord) error
	FindSplitRecord(ctx context.Context, id uint) (*CustomerSplitRecord, error)
	FindSplitRecordByOperationKey(ctx context.Context, operationKey string) (*CustomerSplitRecord, error)
	CompleteSplitRecord(ctx context.Context, record *CustomerSplitRecord) (bool, error)
	ListSplitRecords(ctx context.Context, filter SplitHistoryFilter) ([]CustomerSplitRecord, error)

	CreateSplitMovedEntities(ctx context.Context, moved []SplitMovedEntity) error
	ListSplitMovedEntities(ctx context.Context, splitRecordID uint) ([]SplitMovedEntity, error)
	CurrentSplitEntityState(ctx context.Context, moved SplitMovedEntity) (SplitEntityState, error)
	ApplySplitEntityState(ctx context.Context, moved SplitMovedEntity, before, after SplitEntityState) error

	CreateSplitOperationEvent(ctx context.Context, event *CustomerSplitOperationEvent) error
	ListSplitOperationEvents(ctx context.Context, splitRecordID uint) ([]CustomerSplitOperationEvent, error)
	InvalidateCandidatesAfterSplit(ctx context.Context, sourceProfileID, targetProfileID uint) error
}
