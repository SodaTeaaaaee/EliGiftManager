package domain

import (
	"context"
	"time"
)

const (
	ImportRetentionImmediate = 0
	ImportRetention30Days    = 30
	ImportRetention90Days    = 90
	ImportRetentionPermanent = -1
)

// ImportRun is the safe, listable import-level audit record. Raw row content is
// held only in ImportRawRecord and is exposed through an explicit detail API.
type ImportRun struct {
	ID                     uint
	RunKey                 string
	ImportKind             string
	IntegrationProfileID   *uint
	SourceFormat           string
	SourceFileName         string
	ImportMode             string
	Status                 string
	RetentionDays          int
	RetentionPolicyVersion uint
	ExpiresAt              *time.Time
	RecordCount            int
	SuccessCount           int
	FailureCount           int
	QuarantinedCount       int
	ParserMetadata         string
	CreatedAt              time.Time
	CompletedAt            *time.Time
}

type ImportRawRecord struct {
	ID             uint
	ImportRunID    uint
	RowIndex       int
	RawLogicalRow  string
	UnmappedSource string
	ParserMetadata string
	WarningCodes   string
	AssetMembers   string
	Outcome        string
	ErrorCode      string
	ErrorMessage   string
	ResultType     string
	ResultID       *uint
	RetentionDays  int
	ExpiresAt      *time.Time
	CreatedAt      time.Time
}

type ImportEvidenceSetting struct {
	ID            uint
	RetentionDays int
	Revision      uint
	UpdatedAt     time.Time
}

type ImportRunListQuery struct {
	Limit           int
	Status          string
	ProfileID       *uint
	DocumentType    string
	BeforeCreatedAt *time.Time
	BeforeID        uint
}

type ImportEvidenceRepository interface {
	GetSetting(ctx context.Context) (*ImportEvidenceSetting, error)
	SaveSetting(ctx context.Context, setting *ImportEvidenceSetting) error
	CreateRun(ctx context.Context, run *ImportRun) error
	UpdateRun(ctx context.Context, run *ImportRun) error
	CreateRecord(ctx context.Context, record *ImportRawRecord) error
	UpdateRecord(ctx context.Context, record *ImportRawRecord) error
	CreateRunWithRecords(ctx context.Context, run *ImportRun, records []ImportRawRecord) error
	FinalizeRunWithRecords(ctx context.Context, run *ImportRun, records []ImportRawRecord) error
	ListRuns(ctx context.Context, limit int) ([]ImportRun, error)
	ListRunsPage(ctx context.Context, query ImportRunListQuery) ([]ImportRun, error)
	FindRunByID(ctx context.Context, id uint) (*ImportRun, error)
	ListRecordsByRun(ctx context.Context, runID uint) ([]ImportRawRecord, error)
	PruneExpired(ctx context.Context, now time.Time) (runsDeleted int64, recordsDeleted int64, err error)
}

type ExternalCarrier struct {
	ID                   uint
	IntegrationProfileID uint
	CanonicalKey         string
	ExternalCarrierCode  string
	ExternalCarrierName  string
	NameKeyStrategy      string
	InternalCarrierCode  *string
	Status               string
	ConflictReason       string
	SourceImportRunID    *uint
	SourceRawRecordID    *uint
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ExternalCarrierConflict struct {
	ID                     uint
	IntegrationProfileID   uint
	CanonicalKey           string
	ConflictKind           string
	ExternalCarrierCode    string
	ExternalCarrierName    string
	InternalCarrierCode    string
	SourceImportRunID      *uint
	SourceRawRecordID      *uint
	LegacyCarrierMappingID *uint
	Payload                string
	CreatedAt              time.Time
}

type ExternalCarrierRepository interface {
	Create(ctx context.Context, carrier *ExternalCarrier) error
	Update(ctx context.Context, carrier *ExternalCarrier) error
	FindByID(ctx context.Context, id uint) (*ExternalCarrier, error)
	FindByCanonicalKey(ctx context.Context, profileID uint, canonicalKey string) (*ExternalCarrier, error)
	ListByProfile(ctx context.Context, profileID uint) ([]ExternalCarrier, error)
	CreateConflict(ctx context.Context, conflict *ExternalCarrierConflict) error
	CreateConflicts(ctx context.Context, conflicts []ExternalCarrierConflict) error
}
