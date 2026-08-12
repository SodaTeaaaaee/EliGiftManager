package dto

import "time"

type CustomerMergePreviewInput struct {
	SourceProfileID           uint                       `json:"sourceProfileId"`
	TargetProfileID           uint                       `json:"targetProfileId"`
	CandidateID               *uint                      `json:"candidateId"`
	PrimaryIdentitySelections []PrimaryIdentitySelection `json:"primaryIdentitySelections"`
	DefaultAddressID          *uint                      `json:"defaultAddressId"`
	DisplayNameResolution     string                     `json:"displayNameResolution"`
}

type MergeEntityCounts struct {
	Identities       int `json:"identities"`
	Addresses        int `json:"addresses"`
	DemandDocuments  int `json:"demandDocuments"`
	NameObservations int `json:"nameObservations"`
	NameEvents       int `json:"nameEvents"`
	Origins          int `json:"origins"`
	ProfileMutations int `json:"profileMutations"`
}

type MergePlannedEntity struct {
	EntityType    string `json:"entityType"`
	EntityID      uint   `json:"entityId"`
	MutationKind  string `json:"mutationKind"`
	FromProfileID uint   `json:"fromProfileId"`
	ToProfileID   uint   `json:"toProfileId"`
}

type MergeBlocker struct {
	Code       string `json:"code"`
	EntityType string `json:"entityType"`
	EntityID   uint   `json:"entityId"`
	Detail     string `json:"detail"`
}

type PrimaryIdentityOption struct {
	Namespace         string `json:"namespace"`
	IdentityType      string `json:"identityType"`
	IdentityID        uint   `json:"identityId"`
	CustomerProfileID uint   `json:"customerProfileId"`
	DisplayValue      string `json:"displayValue"`
	CurrentPrimary    bool   `json:"currentPrimary"`
}

type DefaultAddressOption struct {
	AddressID         uint   `json:"addressId"`
	CustomerProfileID uint   `json:"customerProfileId"`
	DisplayValue      string `json:"displayValue"`
	CurrentDefault    bool   `json:"currentDefault"`
}

type DisplayNameOption struct {
	Resolution  string `json:"resolution"`
	DisplayName string `json:"displayName"`
	ProfileID   uint   `json:"profileId"`
}

type CustomerMergePreviewResult struct {
	PreviewToken                         string                     `json:"previewToken"`
	PreviewHash                          string                     `json:"previewHash"`
	GeneratedAt                          time.Time                  `json:"generatedAt" ts_type:"string"`
	SourceProfileID                      uint                       `json:"sourceProfileId"`
	TargetProfileID                      uint                       `json:"targetProfileId"`
	SourceStatus                         string                     `json:"sourceStatus"`
	TargetStatus                         string                     `json:"targetStatus"`
	SourceRowVersion                     uint64                     `json:"sourceRowVersion"`
	TargetRowVersion                     uint64                     `json:"targetRowVersion"`
	CandidateID                          *uint                      `json:"candidateId"`
	CandidateRowVersion                  uint64                     `json:"candidateRowVersion"`
	EvidenceHash                         string                     `json:"evidenceHash"`
	PolicyVersion                        uint                       `json:"policyVersion"`
	PolicyRevisionID                     *uint                      `json:"policyRevisionId"`
	DependsOnMergeRecordID               *uint                      `json:"dependsOnMergeRecordId"`
	PlannedEntities                      []MergePlannedEntity       `json:"plannedEntities"`
	FrozenDemandDocumentIDs              []uint                     `json:"frozenDemandDocumentIds"`
	Counts                               MergeEntityCounts          `json:"counts"`
	Blockers                             []MergeBlocker             `json:"blockers"`
	CanExecute                           bool                       `json:"canExecute"`
	PrimaryIdentityOptions               []PrimaryIdentityOption    `json:"primaryIdentityOptions"`
	RecommendedPrimaryIdentitySelections []PrimaryIdentitySelection `json:"recommendedPrimaryIdentitySelections"`
	DefaultAddressOptions                []DefaultAddressOption     `json:"defaultAddressOptions"`
	RecommendedDefaultAddressID          *uint                      `json:"recommendedDefaultAddressId"`
	DisplayNameOptions                   []DisplayNameOption        `json:"displayNameOptions"`
	RecommendedDisplayNameResolution     string                     `json:"recommendedDisplayNameResolution"`
}

type PrimaryIdentitySelection struct {
	Namespace    string `json:"namespace"`
	IdentityType string `json:"identityType"`
	IdentityID   uint   `json:"identityId"`
}

type ExecuteCustomerMergeInput struct {
	OperationKey                string                     `json:"operationKey"`
	PreviewToken                string                     `json:"previewToken"`
	SourceProfileID             uint                       `json:"sourceProfileId"`
	TargetProfileID             uint                       `json:"targetProfileId"`
	ExpectedSourceRowVersion    uint64                     `json:"expectedSourceRowVersion"`
	ExpectedTargetRowVersion    uint64                     `json:"expectedTargetRowVersion"`
	CandidateID                 *uint                      `json:"candidateId"`
	ExpectedCandidateRowVersion uint64                     `json:"expectedCandidateRowVersion"`
	ExpectedEvidenceHash        string                     `json:"expectedEvidenceHash"`
	ExpectedPolicyVersion       uint                       `json:"expectedPolicyVersion"`
	ExpectedPolicyRevisionID    *uint                      `json:"expectedPolicyRevisionId"`
	PrimaryIdentitySelections   []PrimaryIdentitySelection `json:"primaryIdentitySelections"`
	DefaultAddressID            *uint                      `json:"defaultAddressId"`
	DisplayNameResolution       string                     `json:"displayNameResolution"`
	ActorRef                    string                     `json:"actorRef"`
	DecisionReason              string                     `json:"decisionReason"`
}

type ExecuteCustomerMergeResult struct {
	MergeID            uint              `json:"mergeId"`
	OperationKey       string            `json:"operationKey"`
	Status             string            `json:"status"`
	Counts             MergeEntityCounts `json:"counts"`
	SourceRowVersion   uint64            `json:"sourceRowVersion"`
	TargetRowVersion   uint64            `json:"targetRowVersion"`
	CandidateStatus    string            `json:"candidateStatus"`
	UndoDryRunRequired bool              `json:"undoDryRunRequired"`
	IdempotentReplay   bool              `json:"idempotentReplay"`
}

type CustomerMergeUndoDryRunInput struct {
	MergeID uint `json:"mergeId"`
}

type CustomerMergeUndoDryRunResult struct {
	MergeID           uint              `json:"mergeId"`
	Eligible          bool              `json:"eligible"`
	EligibilityToken  string            `json:"eligibilityToken"`
	GeneratedAt       time.Time         `json:"generatedAt" ts_type:"string"`
	SourceRowVersion  uint64            `json:"sourceRowVersion"`
	TargetRowVersion  uint64            `json:"targetRowVersion"`
	RestoreCounts     MergeEntityCounts `json:"restoreCounts"`
	Blockers          []MergeBlocker    `json:"blockers"`
	Warnings          []string          `json:"warnings"`
	AuditLevel        string            `json:"auditLevel"`
	DependentMergeIDs []uint            `json:"dependentMergeIds"`
}

type ExecuteCustomerMergeUndoInput struct {
	MergeID                  uint   `json:"mergeId"`
	UndoOperationKey         string `json:"undoOperationKey"`
	EligibilityToken         string `json:"eligibilityToken"`
	ExpectedSourceRowVersion uint64 `json:"expectedSourceRowVersion"`
	ExpectedTargetRowVersion uint64 `json:"expectedTargetRowVersion"`
	ActorRef                 string `json:"actorRef"`
	Reason                   string `json:"reason"`
}

type ExecuteCustomerMergeUndoResult struct {
	MergeID                 uint              `json:"mergeId"`
	Status                  string            `json:"status"`
	RestoredSourceProfileID uint              `json:"restoredSourceProfileId"`
	TargetProfileID         uint              `json:"targetProfileId"`
	RestoreCounts           MergeEntityCounts `json:"restoreCounts"`
	IdempotentReplay        bool              `json:"idempotentReplay"`
}

type CustomerMergeHistoryQuery struct {
	ProfileID       uint       `json:"profileId"`
	CandidateID     uint       `json:"candidateId"`
	Status          string     `json:"status"`
	BeforeCreatedAt *time.Time `json:"beforeCreatedAt" ts_type:"string"`
	BeforeID        uint       `json:"beforeId"`
	Limit           int        `json:"limit"`
}

type CustomerMergeHistoryItem struct {
	MergeID              uint              `json:"mergeId"`
	SourceProfileID      uint              `json:"sourceProfileId"`
	TargetProfileID      uint              `json:"targetProfileId"`
	SourceDisplayName    string            `json:"sourceDisplayName"`
	TargetDisplayName    string            `json:"targetDisplayName"`
	Status               string            `json:"status"`
	MergeMode            string            `json:"mergeMode"`
	ActorRef             string            `json:"actorRef"`
	DecisionReason       string            `json:"decisionReason"`
	CandidateID          *uint             `json:"candidateId"`
	PolicyRevisionID     *uint             `json:"policyRevisionId"`
	Counts               MergeEntityCounts `json:"counts"`
	AuditLevel           string            `json:"auditLevel"`
	CreatedAt            time.Time         `json:"createdAt" ts_type:"string"`
	UndoneAt             *time.Time        `json:"undoneAt" ts_type:"string"`
	CanRequestUndoDryRun bool              `json:"canRequestUndoDryRun"`
}

type CustomerMergeHistoryPage struct {
	Items         []CustomerMergeHistoryItem `json:"items"`
	NextCreatedAt *time.Time                 `json:"nextCreatedAt" ts_type:"string"`
	NextID        uint                       `json:"nextId"`
}

type CustomerMergeHistoryDetail struct {
	CustomerMergeHistoryItem
	PlannedEntities  []MergePlannedEntity     `json:"plannedEntities"`
	Events           []MergeOperationEventDTO `json:"events"`
	EvidenceSnapshot string                   `json:"evidenceSnapshot"`
}

type MergeOperationEventDTO struct {
	EventType  string    `json:"eventType"`
	Status     string    `json:"status"`
	ActorRef   string    `json:"actorRef"`
	ReasonCode string    `json:"reasonCode"`
	CreatedAt  time.Time `json:"createdAt" ts_type:"string"`
}
