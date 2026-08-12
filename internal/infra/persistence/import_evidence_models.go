package persistence

import "time"

type ImportRun struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement"`
	RunKey                 string     `gorm:"type:text;not null;uniqueIndex"`
	ImportKind             string     `gorm:"type:text;not null;index"`
	IntegrationProfileID   *uint      `gorm:"index"`
	SourceFormat           string     `gorm:"type:text;not null;default:''"`
	SourceFileName         string     `gorm:"type:text;not null;default:''"`
	ImportMode             string     `gorm:"type:text;not null;default:'skip_invalid'"`
	Status                 string     `gorm:"type:text;not null;default:'running';index"`
	RetentionDays          int        `gorm:"not null;default:90"`
	RetentionPolicyVersion uint       `gorm:"not null;default:1"`
	ExpiresAt              *time.Time `gorm:"index"`
	RecordCount            int        `gorm:"not null;default:0"`
	SuccessCount           int        `gorm:"not null;default:0"`
	FailureCount           int        `gorm:"not null;default:0"`
	QuarantinedCount       int        `gorm:"not null;default:0"`
	ParserMetadata         string     `gorm:"type:text;not null;default:''"`
	CreatedAt              time.Time  `gorm:"not null;index"`
	CompletedAt            *time.Time
}

func (ImportRun) TableName() string { return "import_runs" }

type ImportRawRecord struct {
	ID             uint       `gorm:"primaryKey;autoIncrement"`
	ImportRunID    uint       `gorm:"not null;index"`
	RowIndex       int        `gorm:"not null"`
	RawLogicalRow  string     `gorm:"type:text;not null;default:''"`
	UnmappedSource string     `gorm:"type:text;not null;default:''"`
	ParserMetadata string     `gorm:"type:text;not null;default:''"`
	WarningCodes   string     `gorm:"type:text;not null;default:'[]'"`
	AssetMembers   string     `gorm:"type:text;not null;default:'[]'"`
	Outcome        string     `gorm:"type:text;not null;default:'pending';index"`
	ErrorCode      string     `gorm:"type:text;not null;default:''"`
	ErrorMessage   string     `gorm:"type:text;not null;default:''"`
	ResultType     string     `gorm:"type:text;not null;default:''"`
	ResultID       *uint      `gorm:"index"`
	RetentionDays  int        `gorm:"not null;default:90"`
	ExpiresAt      *time.Time `gorm:"index"`
	CreatedAt      time.Time  `gorm:"not null;index"`
}

func (ImportRawRecord) TableName() string { return "import_raw_records" }

type ImportEvidenceSetting struct {
	ID            uint      `gorm:"primaryKey"`
	RetentionDays int       `gorm:"not null;default:90"`
	Revision      uint      `gorm:"not null;default:1"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (ImportEvidenceSetting) TableName() string { return "import_evidence_settings" }

type ExternalCarrier struct {
	ID                   uint      `gorm:"primaryKey;autoIncrement"`
	IntegrationProfileID uint      `gorm:"not null;index;uniqueIndex:idx_external_carrier_profile_key,priority:1"`
	CanonicalKey         string    `gorm:"type:text;not null;uniqueIndex:idx_external_carrier_profile_key,priority:2"`
	ExternalCarrierCode  string    `gorm:"type:text;not null;default:'';index"`
	ExternalCarrierName  string    `gorm:"type:text;not null;default:'';index"`
	NameKeyStrategy      string    `gorm:"type:text;not null;default:'code_or_normalized_name_v1'"`
	InternalCarrierCode  *string   `gorm:"type:text;index"`
	Status               string    `gorm:"type:text;not null;default:'provisional';index"`
	ConflictReason       string    `gorm:"type:text;not null;default:''"`
	SourceImportRunID    *uint     `gorm:"index"`
	SourceRawRecordID    *uint     `gorm:"index"`
	CreatedAt            time.Time `gorm:"not null"`
	UpdatedAt            time.Time `gorm:"not null"`
}

func (ExternalCarrier) TableName() string { return "external_carriers" }

type ExternalCarrierConflict struct {
	ID                     uint      `gorm:"primaryKey;autoIncrement"`
	IntegrationProfileID   uint      `gorm:"not null;index"`
	CanonicalKey           string    `gorm:"type:text;not null;index"`
	ConflictKind           string    `gorm:"type:text;not null;index"`
	ExternalCarrierCode    string    `gorm:"type:text;not null;default:''"`
	ExternalCarrierName    string    `gorm:"type:text;not null;default:''"`
	InternalCarrierCode    string    `gorm:"type:text;not null;default:''"`
	SourceImportRunID      *uint     `gorm:"index"`
	SourceRawRecordID      *uint     `gorm:"index"`
	LegacyCarrierMappingID *uint     `gorm:"index"`
	Payload                string    `gorm:"type:text;not null;default:''"`
	CreatedAt              time.Time `gorm:"not null;index"`
}

func (ExternalCarrierConflict) TableName() string { return "external_carrier_conflicts" }
