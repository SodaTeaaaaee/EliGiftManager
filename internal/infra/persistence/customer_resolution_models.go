package persistence

import (
	"time"

	"gorm.io/gorm"
)

type CustomerNameObservation struct {
	gorm.Model
	CustomerProfileID          uint       `gorm:"not null;index;uniqueIndex:idx_name_episode_profile,priority:1"`
	Name                       string     `gorm:"type:text;not null"`
	NormalizedName             string     `gorm:"type:text;not null;default:'';index"`
	SourceEventKey             string     `gorm:"type:text;not null;default:'';index"`
	EpisodeKey                 string     `gorm:"type:text;not null;default:'';uniqueIndex:idx_name_episode_profile,priority:2"`
	ObservationCount           uint       `gorm:"not null;default:1"`
	NameKind                   string     `gorm:"type:text;not null;default:'';index"`
	Authority                  string     `gorm:"type:text;not null;default:'';index"`
	TrustScore                 float64    `gorm:"not null;default:0"`
	SourceIntegrationProfileID *uint      `gorm:"index"`
	SourceDocumentID           *uint      `gorm:"index"`
	SourceIdentityID           *uint      `gorm:"index"`
	ObservedAt                 *time.Time `gorm:"index"`
	FirstSeenAt                *time.Time
	LastSeenAt                 *time.Time
	IsPinned                   bool   `gorm:"not null;default:false;index"`
	IsActive                   bool   `gorm:"not null;default:true;index"`
	ExtraData                  string `gorm:"type:text;not null;default:''"`
}

func (CustomerNameObservation) TableName() string { return "customer_name_observations" }

type CustomerNameEvent struct {
	ID                uint      `gorm:"primaryKey;autoIncrement"`
	EventKey          string    `gorm:"type:text;not null;default:'';uniqueIndex"`
	CustomerProfileID uint      `gorm:"not null;index"`
	ObservationID     *uint     `gorm:"index"`
	EventKind         string    `gorm:"type:text;not null;index"`
	PreviousName      string    `gorm:"type:text;not null;default:''"`
	NewName           string    `gorm:"type:text;not null;default:''"`
	ReasonCode        string    `gorm:"type:text;not null;default:'';index"`
	ActorRef          string    `gorm:"type:text;not null;default:''"`
	Payload           string    `gorm:"type:text;not null;default:''"`
	CreatedAt         time.Time `gorm:"not null;index"`
}

func (CustomerNameEvent) TableName() string { return "customer_name_events" }

type CustomerProfileOrigin struct {
	gorm.Model
	CustomerProfileID          uint       `gorm:"not null;index"`
	OriginKind                 string     `gorm:"type:text;not null;default:'';index;uniqueIndex:idx_origin_external_key,priority:1"`
	SourceIntegrationProfileID *uint      `gorm:"index;uniqueIndex:idx_origin_external_key,priority:2"`
	SourceDocumentID           *uint      `gorm:"index"`
	ExternalRef                string     `gorm:"type:text;not null;default:'';index;uniqueIndex:idx_origin_external_key,priority:3"`
	IsProvisional              bool       `gorm:"not null;default:false;index"`
	FirstSeenAt                *time.Time `gorm:"index"`
	LastSeenAt                 *time.Time
	ExtraData                  string `gorm:"type:text;not null;default:''"`
}

func (CustomerProfileOrigin) TableName() string { return "customer_profile_origins" }

type MergeCandidate struct {
	gorm.Model
	SourceProfileID       uint       `gorm:"not null;index"`
	TargetProfileID       uint       `gorm:"not null;index"`
	Status                string     `gorm:"type:text;not null;default:'pending';index"`
	Score                 float64    `gorm:"not null;default:0;index"`
	MergePolicyRevisionID *uint      `gorm:"index"`
	Reason                string     `gorm:"type:text;not null;default:''"`
	CanonicalPairKey      string     `gorm:"type:text;not null;default:'';index"`
	EvidenceHash          string     `gorm:"type:text;not null;default:'';index"`
	PolicyVersion         uint       `gorm:"not null;default:0;index"`
	ExplanationCode       string     `gorm:"type:text;not null;default:'';index"`
	Confidence            float64    `gorm:"not null;default:0"`
	Blockers              string     `gorm:"type:text;not null;default:'[]'"`
	LastEvaluatedAt       *time.Time `gorm:"index"`
	ExpiresAt             *time.Time `gorm:"index"`
	ScanRunID             *uint      `gorm:"index"`
	RowVersion            uint64     `gorm:"not null;default:1"`
	ExecutedMergeRecordID *uint      `gorm:"index"`
	ExecutedAt            *time.Time `gorm:"index"`
}

func (MergeCandidate) TableName() string { return "merge_candidates" }

type MergeEvidence struct {
	ID               uint       `gorm:"primaryKey;autoIncrement"`
	MergeCandidateID uint       `gorm:"not null;index"`
	EvidenceKind     string     `gorm:"type:text;not null;index"`
	SourceRef        string     `gorm:"type:text;not null;default:''"`
	Weight           float64    `gorm:"not null;default:0"`
	Confidence       float64    `gorm:"not null;default:0"`
	Payload          string     `gorm:"type:text;not null;default:''"`
	EvidenceKey      string     `gorm:"type:text;not null;default:'';index"`
	Polarity         string     `gorm:"type:text;not null;default:'positive';index"`
	ExplanationCode  string     `gorm:"type:text;not null;default:'';index"`
	ValueHash        string     `gorm:"type:text;not null;default:'';index"`
	MaskedValue      string     `gorm:"type:text;not null;default:''"`
	SourceEntityType string     `gorm:"type:text;not null;default:''"`
	SourceEntityID   uint       `gorm:"not null;default:0"`
	ObservedAt       *time.Time `gorm:"index"`
	CreatedAt        time.Time  `gorm:"not null;index"`
}

func (MergeEvidence) TableName() string { return "merge_evidence" }

type MergePolicy struct {
	gorm.Model
	PolicyKey         string     `gorm:"type:text;not null;index"`
	Name              string     `gorm:"type:text;not null"`
	IsActive          bool       `gorm:"not null;default:true;index"`
	DefaultAction     string     `gorm:"type:text;not null;default:'suggest_only';index"`
	CurrentRevisionID *uint      `gorm:"index"`
	ExtraData         string     `gorm:"type:text;not null;default:''"`
	RowVersion        uint64     `gorm:"not null;default:1"`
	NeedsScan         bool       `gorm:"not null;default:true;index"`
	LastScanAt        *time.Time `gorm:"index"`
}

func (MergePolicy) TableName() string { return "merge_policies" }

type MergePolicyRevision struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	MergePolicyID uint      `gorm:"not null;index"`
	Revision      uint      `gorm:"not null;default:1;index"`
	Action        string    `gorm:"type:text;not null;default:'suggest_only';index"`
	Rules         string    `gorm:"type:text;not null;default:''"`
	Checksum      string    `gorm:"type:text;not null;default:'';index"`
	CreatedBy     string    `gorm:"type:text;not null;default:''"`
	SchemaVersion uint      `gorm:"not null;default:1"`
	CreatedAt     time.Time `gorm:"not null;index"`
}

type MergeScanRun struct {
	ID                uint       `gorm:"primaryKey;autoIncrement"`
	MergePolicyID     uint       `gorm:"not null;index"`
	PolicyRevisionID  uint       `gorm:"not null;index"`
	PolicyVersion     uint       `gorm:"not null;index"`
	Status            string     `gorm:"type:text;not null;index"`
	StartedAt         time.Time  `gorm:"not null;index"`
	CompletedAt       *time.Time `gorm:"index"`
	ProfilesScanned   uint       `gorm:"not null;default:0"`
	PairsEvaluated    uint       `gorm:"not null;default:0"`
	CandidatesCreated uint       `gorm:"not null;default:0"`
	CandidatesUpdated uint       `gorm:"not null;default:0"`
	CandidatesBlocked uint       `gorm:"not null;default:0"`
	ErrorMessage      string     `gorm:"type:text;not null;default:''"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (MergeScanRun) TableName() string { return "merge_scan_runs" }

func (MergePolicyRevision) TableName() string { return "merge_policy_revisions" }

type MergeMovedEntity struct {
	ID                   uint   `gorm:"primaryKey;autoIncrement"`
	MergeRecordID        uint   `gorm:"not null;index"`
	EntityType           string `gorm:"type:text;not null;index"`
	EntityID             uint   `gorm:"not null;index"`
	FromProfileID        uint   `gorm:"not null;index"`
	ToProfileID          uint   `gorm:"not null;index"`
	MoveOrder            uint   `gorm:"not null;default:0;index"`
	BeforeSnapshot       string `gorm:"type:text;not null;default:''"`
	MutationKind         string `gorm:"type:text;not null;default:''"`
	RestoreMode          string `gorm:"type:text;not null;default:''"`
	SnapshotVersion      uint   `gorm:"not null;default:0"`
	AfterSnapshot        string `gorm:"type:text;not null;default:''"`
	AfterStateHash       string `gorm:"type:text;not null;default:''"`
	EntityUpdatedAtAfter *time.Time
	UndoState            string     `gorm:"type:text;not null;default:''"`
	UndoBlockerCode      string     `gorm:"type:text;not null;default:''"`
	RevertOperationKey   string     `gorm:"type:text;not null;default:''"`
	RevertedAt           *time.Time `gorm:"index"`
	CreatedAt            time.Time  `gorm:"not null;index"`
}

func (MergeMovedEntity) TableName() string { return "merge_moved_entities" }

type CustomerMergeOperationEvent struct {
	ID            uint      `gorm:"primaryKey;autoIncrement"`
	MergeRecordID *uint     `gorm:"index"`
	EventKey      string    `gorm:"type:text;not null;default:''"`
	OperationKey  string    `gorm:"type:text;not null;default:'';index"`
	EventType     string    `gorm:"type:text;not null;index"`
	Status        string    `gorm:"type:text;not null;index"`
	ActorRef      string    `gorm:"type:text;not null;default:''"`
	ReasonCode    string    `gorm:"type:text;not null;default:''"`
	Payload       string    `gorm:"type:text;not null;default:''"`
	CreatedAt     time.Time `gorm:"not null;index"`
}

func (CustomerMergeOperationEvent) TableName() string { return "customer_merge_operation_events" }
