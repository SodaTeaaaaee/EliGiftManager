package domain

import (
	"context"
	"time"
)

const (
	CustomerProfileStatusActive = "active"
	CustomerProfileStatusMerged = "merged"

	DisplayNameModeAuto   = "auto"
	DisplayNameModePinned = "pinned"

	MergePolicyActionSuggestOnly = "suggest_only"
	MergePolicyKeyDefault        = "customer_merge_default"

	MergeEvidenceModeOff                = "off"
	MergeEvidenceModeLegacyRawExact     = "legacy_raw_exact"
	MergeEvidenceModeNormalizedVerified = "normalized_verified"
	MergeEvidenceModeNormalized         = "normalized"

	MergeCandidateStatusPending    = "pending"
	MergeCandidateStatusBlocked    = "blocked"
	MergeCandidateStatusDismissed  = "dismissed"
	MergeCandidateStatusStale      = "stale"
	MergeCandidateStatusSuperseded = "superseded"
	MergeCandidateStatusExpired    = "expired"
	MergeCandidateStatusFailed     = "failed"
	MergeCandidateStatusExecuted   = "executed"

	MergeScanStatusRunning   = "running"
	MergeScanStatusCompleted = "completed"
	MergeScanStatusFailed    = "failed"

	CustomerNameKindStableIdentityNickname = "stable_identity_nickname"
	CustomerNameKindTrustedNickname        = "trusted_nickname"
	CustomerNameKindRecipient              = "recipient"
	CustomerNameKindManual                 = "manual"
	CustomerOriginKindRetailOrder          = "retail_order"
)

// CustomerNameObservation preserves every observed nickname or recipient name.
// Resolution code can later apply the documented priority without discarding history.
type CustomerNameObservation struct {
	ID                uint
	CustomerProfileID uint
	// OriginProfileID is a read projection. For observations moved by a merge
	// it remains the earliest profile recorded in the moved-entity ledger.
	OriginProfileID            uint
	Name                       string
	NormalizedName             string
	SourceEventKey             string
	EpisodeKey                 string
	ObservationCount           uint
	NameKind                   string
	Authority                  string
	TrustScore                 float64
	SourceIntegrationProfileID *uint
	SourceDocumentID           *uint
	SourceIdentityID           *uint
	ObservedAt                 *time.Time
	FirstSeenAt                *time.Time
	LastSeenAt                 *time.Time
	IsPinned                   bool
	IsActive                   bool
	ExtraData                  string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type CustomerNameEvent struct {
	ID                uint
	EventKey          string
	CustomerProfileID uint
	ObservationID     *uint
	EventKind         string
	PreviousName      string
	NewName           string
	ReasonCode        string
	ActorRef          string
	Payload           string
	CreatedAt         time.Time
}

// CustomerNameEventPayload is the shared wire/storage contract for observed
// name events. Both live imports and legacy migration must emit this shape so
// chronological episode rebuilding remains deterministic.
type CustomerNameEventPayload struct {
	NameKind                   string  `json:"nameKind"`
	Authority                  string  `json:"authority"`
	TrustScore                 float64 `json:"trustScore"`
	SourceIntegrationProfileID *uint   `json:"sourceIntegrationProfileId,omitempty"`
	SourceDocumentID           *uint   `json:"sourceDocumentId,omitempty"`
	SourceIdentityID           *uint   `json:"sourceIdentityId,omitempty"`
	ExtraData                  string  `json:"extraData,omitempty"`
}

type CustomerProfileOrigin struct {
	ID                         uint
	CustomerProfileID          uint
	OriginKind                 string
	SourceIntegrationProfileID *uint
	SourceDocumentID           *uint
	ExternalRef                string
	IsProvisional              bool
	FirstSeenAt                *time.Time
	LastSeenAt                 *time.Time
	ExtraData                  string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type MergeCandidate struct {
	ID                    uint
	SourceProfileID       uint
	TargetProfileID       uint
	Status                string
	Score                 float64
	MergePolicyRevisionID *uint
	Reason                string
	CanonicalPairKey      string
	EvidenceHash          string
	PolicyVersion         uint
	ExplanationCode       string
	Confidence            float64
	Blockers              string
	LastEvaluatedAt       *time.Time
	ExpiresAt             *time.Time
	ScanRunID             *uint
	RowVersion            uint64
	ExecutedMergeRecordID *uint
	ExecutedAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MergeEvidence struct {
	ID               uint
	MergeCandidateID uint
	EvidenceKind     string
	SourceRef        string
	Weight           float64
	Confidence       float64
	Payload          string
	EvidenceKey      string
	Polarity         string
	ExplanationCode  string
	ValueHash        string
	MaskedValue      string
	SourceEntityType string
	SourceEntityID   uint
	ObservedAt       *time.Time
	CreatedAt        time.Time
}

type MergePolicy struct {
	ID                uint
	PolicyKey         string
	Name              string
	IsActive          bool
	DefaultAction     string
	CurrentRevisionID *uint
	ExtraData         string
	RowVersion        uint64
	NeedsScan         bool
	LastScanAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MergePolicyRevision struct {
	ID            uint
	MergePolicyID uint
	Revision      uint
	Action        string
	Rules         string
	Checksum      string
	CreatedBy     string
	SchemaVersion uint
	CreatedAt     time.Time
}

type MergePolicyRules struct {
	SchemaVersion             uint   `json:"schemaVersion"`
	CandidateDetectionEnabled bool   `json:"candidateDetectionEnabled"`
	EmailEvidenceMode         string `json:"emailEvidenceMode"`
	PhoneEvidenceMode         string `json:"phoneEvidenceMode"`
	ExecutionMode             string `json:"executionMode"`
}

type MergeScanRun struct {
	ID                uint
	MergePolicyID     uint
	PolicyRevisionID  uint
	PolicyVersion     uint
	Status            string
	StartedAt         time.Time
	CompletedAt       *time.Time
	ProfilesScanned   uint
	PairsEvaluated    uint
	CandidatesCreated uint
	CandidatesUpdated uint
	CandidatesBlocked uint
	ErrorMessage      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type MergeMovedEntity struct {
	ID                   uint
	MergeRecordID        uint
	EntityType           string
	EntityID             uint
	FromProfileID        uint
	ToProfileID          uint
	MoveOrder            uint
	BeforeSnapshot       string
	MutationKind         string
	RestoreMode          string
	SnapshotVersion      uint
	AfterSnapshot        string
	AfterStateHash       string
	EntityUpdatedAtAfter *time.Time
	UndoState            string
	UndoBlockerCode      string
	RevertOperationKey   string
	RevertedAt           *time.Time
	CreatedAt            time.Time
}

// These ports are deliberately additive. Stage 1 persists facts and audit state;
// production customer resolution and merge orchestration continue to use existing ports.
type CustomerNameObservationRepository interface {
	Create(ctx context.Context, observation *CustomerNameObservation) error
	Update(ctx context.Context, observation *CustomerNameObservation) error
	FindByID(ctx context.Context, id uint) (*CustomerNameObservation, error)
	FindByEpisodeKey(ctx context.Context, profileID uint, episodeKey string) (*CustomerNameObservation, error)
	FindBySourceEventKey(ctx context.Context, sourceEventKey string) (*CustomerNameObservation, error)
	ListByProfile(ctx context.Context, profileID uint) ([]CustomerNameObservation, error)
	ListByIDs(ctx context.Context, ids []uint) ([]CustomerNameObservation, error)
	DeactivateByProfile(ctx context.Context, profileID uint) error
	BulkUpdateProfileIDByIDs(ctx context.Context, ids []uint, profileID uint) error
}

type CustomerNameEventRepository interface {
	Create(ctx context.Context, event *CustomerNameEvent) error
	CreateIfAbsent(ctx context.Context, event *CustomerNameEvent) (bool, error)
	FindByEventKey(ctx context.Context, eventKey string) (*CustomerNameEvent, error)
	UpdateObservationID(ctx context.Context, eventID uint, observationID uint) error
	ListByProfile(ctx context.Context, profileID uint) ([]CustomerNameEvent, error)
}

type CustomerProfileOriginRepository interface {
	Create(ctx context.Context, origin *CustomerProfileOrigin) error
	CreateIfAbsent(ctx context.Context, origin *CustomerProfileOrigin) (bool, error)
	FindByExternalRef(ctx context.Context, originKind string, integrationProfileID uint, externalRef string) (*CustomerProfileOrigin, error)
	FindByID(ctx context.Context, id uint) (*CustomerProfileOrigin, error)
	Update(ctx context.Context, origin *CustomerProfileOrigin) error
	ListByProfile(ctx context.Context, profileID uint) ([]CustomerProfileOrigin, error)
	ListByIDs(ctx context.Context, ids []uint) ([]CustomerProfileOrigin, error)
	BulkUpdateProfileIDByIDs(ctx context.Context, ids []uint, profileID uint) error
}

// CustomerProfileOriginReadRepository is an additive audit projection used by
// read-only profile detail APIs. For a merged profile it includes origins that
// were moved by the still-active merge ledger without changing current owner
// fields or widening split selection.
type CustomerProfileOriginReadRepository interface {
	ListForProfileRead(ctx context.Context, profileID uint) ([]CustomerProfileOrigin, error)
}

// CustomerDisplayNameRepository is an additive optimistic-locking port used by
// name projection. Keeping it separate avoids widening legacy profile mocks.
type CustomerDisplayNameRepository interface {
	UpdateDisplayNameProjection(ctx context.Context, profileID uint, expectedVersion uint64, displayName, mode string, observationID *uint) (bool, error)
}

// CustomerProfileNativeRepository adds lifecycle-safe reads and metadata CAS
// without widening the legacy CustomerProfileRepository test contract.
type CustomerProfileNativeRepository interface {
	FindByIDIncludingDeleted(ctx context.Context, id uint) (*CustomerProfile, error)
	UpdateProfileMetadataCAS(ctx context.Context, profileID uint, expectedVersion uint64, profileType, extraData string) (bool, error)
}

// StrongIdentityRepository resolves canonical identity keys without silently
// selecting the first row when legacy duplicate data exists.
type StrongIdentityRepository interface {
	ListByResolutionKey(ctx context.Context, namespace, identityType, normalizedValue string) ([]CustomerIdentity, error)
	UpdateResolutionMetadata(ctx context.Context, identity *CustomerIdentity) error
}

type MergeCandidateRepository interface {
	Create(ctx context.Context, candidate *MergeCandidate) error
	FindByID(ctx context.Context, id uint) (*MergeCandidate, error)
	ListPending(ctx context.Context) ([]MergeCandidate, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
}

type MergeEvidenceRepository interface {
	Create(ctx context.Context, evidence *MergeEvidence) error
	ListByCandidate(ctx context.Context, candidateID uint) ([]MergeEvidence, error)
}

type MergePolicyRepository interface {
	Create(ctx context.Context, policy *MergePolicy) error
	CreateRevision(ctx context.Context, revision *MergePolicyRevision) error
	ListActive(ctx context.Context) ([]MergePolicy, error)
	ListRevisions(ctx context.Context, policyID uint) ([]MergePolicyRevision, error)
}

// MergeGovernanceRepository owns the transactional policy revision, candidate
// evidence, and explicit scan-run writes. Read methods never perform detection.
type MergeGovernanceRepository interface {
	EnsurePolicy(ctx context.Context, policy *MergePolicy, revision *MergePolicyRevision) (bool, error)
	FindPolicyByKey(ctx context.Context, policyKey string) (*MergePolicy, *MergePolicyRevision, error)
	UpdatePolicyCAS(ctx context.Context, policyKey string, expectedRevision uint, revision *MergePolicyRevision) (*MergePolicy, bool, error)
	CompletePolicyScan(ctx context.Context, policyID, policyRevisionID uint, completedAt time.Time) error

	CreateScanRun(ctx context.Context, run *MergeScanRun) error
	UpdateScanRun(ctx context.Context, run *MergeScanRun) error
	FindScanRun(ctx context.Context, id uint) (*MergeScanRun, error)

	UpsertCandidateEvaluation(ctx context.Context, candidate *MergeCandidate, evidence []MergeEvidence) (bool, error)
	MarkUnseenCandidatesStale(ctx context.Context, policyVersion, scanRunID uint) error
	FindCandidateWithEvidence(ctx context.Context, id uint) (*MergeCandidate, []MergeEvidence, error)
	ListCandidates(ctx context.Context, status string) ([]MergeCandidate, error)
	DismissCandidate(ctx context.Context, id uint, evidenceHash string, policyVersion uint) (bool, error)
}

type MergeMovedEntityRepository interface {
	Create(ctx context.Context, moved *MergeMovedEntity) error
	ListByMergeRecord(ctx context.Context, mergeRecordID uint) ([]MergeMovedEntity, error)
}
