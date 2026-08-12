package dto

import "time"

type CustomerSplitSelection struct {
	IdentityIDs        []uint `json:"identityIds"`
	AddressIDs         []uint `json:"addressIds"`
	DemandDocumentIDs  []uint `json:"demandDocumentIds"`
	NameObservationIDs []uint `json:"nameObservationIds"`
	OriginIDs          []uint `json:"originIds"`
}

type CustomerSplitPreviewInput struct {
	SourceProfileID                uint                   `json:"sourceProfileId"`
	TargetStrategy                 string                 `json:"targetStrategy"`
	NewProfileDisplayName          string                 `json:"newProfileDisplayName"`
	NewProfileType                 string                 `json:"newProfileType"`
	TargetPrimaryIdentityIDs       []uint                 `json:"targetPrimaryIdentityIds"`
	TargetDefaultAddressID         *uint                  `json:"targetDefaultAddressId"`
	TargetDisplayNameObservationID *uint                  `json:"targetDisplayNameObservationId"`
	SourceDisplayNameResolution    string                 `json:"sourceDisplayNameResolution"`
	Selection                      CustomerSplitSelection `json:"selection"`
}

type SplitImmutableHistoryDTO struct {
	WaveParticipantSnapshotIDs []uint `json:"waveParticipantSnapshotIds"`
	FulfillmentLineIDs         []uint `json:"fulfillmentLineIds"`
	WillRewrite                bool   `json:"willRewrite"`
}

type CustomerSplitPreviewResult struct {
	PlanToken                     string                   `json:"planToken"`
	PlanHash                      string                   `json:"planHash"`
	GeneratedAt                   time.Time                `json:"generatedAt" ts_type:"string"`
	SourceProfileID               uint                     `json:"sourceProfileId"`
	TargetProfileID               uint                     `json:"targetProfileId"`
	TargetStrategy                string                   `json:"targetStrategy"`
	SourceRowVersion              uint64                   `json:"sourceRowVersion"`
	TargetRowVersion              uint64                   `json:"targetRowVersion"`
	SourceDisplayNameAfter        string                   `json:"sourceDisplayNameAfter"`
	TargetDisplayNameAfter        string                   `json:"targetDisplayNameAfter"`
	PlannedEntities               []MergePlannedEntity     `json:"plannedEntities"`
	Counts                        MergeEntityCounts        `json:"counts"`
	Blockers                      []MergeBlocker           `json:"blockers"`
	ImmutableHistory              SplitImmutableHistoryDTO `json:"immutableHistory"`
	CanExecute                    bool                     `json:"canExecute"`
	DirectUndoSupported           bool                     `json:"directUndoSupported"`
	ReverseOperationKind          string                   `json:"reverseOperationKind"`
	UnsupportedTargetStrategyHint string                   `json:"unsupportedTargetStrategyHint"`
}

type ExecuteCustomerSplitInput struct {
	OperationKey             string                    `json:"operationKey"`
	PlanToken                string                    `json:"planToken"`
	ExpectedSourceRowVersion uint64                    `json:"expectedSourceRowVersion"`
	ExpectedTargetRowVersion uint64                    `json:"expectedTargetRowVersion"`
	ActorRef                 string                    `json:"actorRef"`
	DecisionReason           string                    `json:"decisionReason"`
	Plan                     CustomerSplitPreviewInput `json:"plan"`
}

type ExecuteCustomerSplitResult struct {
	SplitID              uint              `json:"splitId"`
	OperationKey         string            `json:"operationKey"`
	Status               string            `json:"status"`
	SourceProfileID      uint              `json:"sourceProfileId"`
	TargetProfileID      uint              `json:"targetProfileId"`
	Counts               MergeEntityCounts `json:"counts"`
	SourceRowVersion     uint64            `json:"sourceRowVersion"`
	TargetRowVersion     uint64            `json:"targetRowVersion"`
	IdempotentReplay     bool              `json:"idempotentReplay"`
	DirectUndoSupported  bool              `json:"directUndoSupported"`
	ReverseOperationKind string            `json:"reverseOperationKind"`
}

type CustomerSplitHistoryQuery struct {
	ProfileID       uint       `json:"profileId"`
	Status          string     `json:"status"`
	BeforeCreatedAt *time.Time `json:"beforeCreatedAt" ts_type:"string"`
	BeforeID        uint       `json:"beforeId"`
	Limit           int        `json:"limit"`
}

type CustomerSplitHistoryItem struct {
	SplitID              uint              `json:"splitId"`
	OperationType        string            `json:"operationType"`
	SourceProfileID      uint              `json:"sourceProfileId"`
	TargetProfileID      uint              `json:"targetProfileId"`
	TargetStrategy       string            `json:"targetStrategy"`
	Status               string            `json:"status"`
	ActorRef             string            `json:"actorRef"`
	DecisionReason       string            `json:"decisionReason"`
	Counts               MergeEntityCounts `json:"counts"`
	CreatedAt            time.Time         `json:"createdAt" ts_type:"string"`
	CompletedAt          *time.Time        `json:"completedAt" ts_type:"string"`
	DirectUndoSupported  bool              `json:"directUndoSupported"`
	ReverseOperationKind string            `json:"reverseOperationKind"`
}

type CustomerSplitHistoryPage struct {
	Items      []CustomerSplitHistoryItem `json:"items"`
	HasMore    bool                       `json:"hasMore"`
	NextBefore *time.Time                 `json:"nextBefore" ts_type:"string"`
	NextID     uint                       `json:"nextId"`
}

type CustomerSplitMovedEntityDTO struct {
	EntityType     string `json:"entityType"`
	EntityID       uint   `json:"entityId"`
	FromProfileID  uint   `json:"fromProfileId"`
	ToProfileID    uint   `json:"toProfileId"`
	MutationKind   string `json:"mutationKind"`
	BeforeSnapshot string `json:"beforeSnapshot"`
	AfterSnapshot  string `json:"afterSnapshot"`
}

type CustomerSplitOperationEventDTO struct {
	EventType  string    `json:"eventType"`
	Status     string    `json:"status"`
	ActorRef   string    `json:"actorRef"`
	ReasonCode string    `json:"reasonCode"`
	CreatedAt  time.Time `json:"createdAt" ts_type:"string"`
}

type CustomerSplitHistoryDetail struct {
	CustomerSplitHistoryItem
	MovedEntities   []CustomerSplitMovedEntityDTO    `json:"movedEntities"`
	Events          []CustomerSplitOperationEventDTO `json:"events"`
	PlanHash        string                           `json:"planHash"`
	ReverseGuidance string                           `json:"reverseGuidance"`
}
