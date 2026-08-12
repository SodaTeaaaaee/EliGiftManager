package dto

import "time"

// ImportRunSummaryDTO intentionally excludes raw rows and parser payloads so
// ordinary lists/logging cannot expose imported PII.
type ImportRunSummaryDTO struct {
	ID                     uint       `json:"id"`
	ImportKind             string     `json:"importKind"`
	IntegrationProfileID   *uint      `json:"integrationProfileId"`
	SourceFormat           string     `json:"sourceFormat"`
	SourceFileName         string     `json:"sourceFileName"`
	ImportMode             string     `json:"importMode"`
	Status                 string     `json:"status"`
	RetentionDays          int        `json:"retentionDays"`
	RetentionPolicyVersion uint       `json:"retentionPolicyVersion"`
	ExpiresAt              *time.Time `json:"expiresAt" ts_type:"string"`
	RecordCount            int        `json:"recordCount"`
	SuccessCount           int        `json:"successCount"`
	FailureCount           int        `json:"failureCount"`
	QuarantinedCount       int        `json:"quarantinedCount"`
	CreatedAt              time.Time  `json:"createdAt" ts_type:"string"`
	CompletedAt            *time.Time `json:"completedAt" ts_type:"string"`
}

type ListImportRunsPageInput struct {
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
	Status       string `json:"status"`
	ProfileID    *uint  `json:"profileId"`
	DocumentType string `json:"documentType"`
}

type ImportRunPageDTO struct {
	Items      []ImportRunSummaryDTO `json:"items"`
	NextCursor string                `json:"nextCursor"`
	HasMore    bool                  `json:"hasMore"`
}

type ImportRawRecordDetailDTO struct {
	ID             uint       `json:"id"`
	RowIndex       int        `json:"rowIndex"`
	RawLogicalRow  string     `json:"rawLogicalRow"`
	UnmappedSource string     `json:"unmappedSource"`
	ParserMetadata string     `json:"parserMetadata"`
	WarningCodes   string     `json:"warningCodes"`
	AssetMembers   string     `json:"assetMembers"`
	Outcome        string     `json:"outcome"`
	ErrorCode      string     `json:"errorCode"`
	ErrorMessage   string     `json:"errorMessage"`
	ResultType     string     `json:"resultType"`
	ResultID       *uint      `json:"resultId"`
	ExpiresAt      *time.Time `json:"expiresAt" ts_type:"string"`
	CreatedAt      time.Time  `json:"createdAt" ts_type:"string"`
}

type ImportRunDetailDTO struct {
	Run     ImportRunSummaryDTO        `json:"run"`
	Records []ImportRawRecordDetailDTO `json:"records"`
}

type ImportEvidenceRetentionDTO struct {
	RetentionDays int       `json:"retentionDays"`
	Revision      uint      `json:"revision"`
	UpdatedAt     time.Time `json:"updatedAt" ts_type:"string"`
}

type SetImportEvidenceRetentionInput struct {
	RetentionDays int `json:"retentionDays"`
}

type ExternalCarrierDTO struct {
	ID                   uint      `json:"id"`
	IntegrationProfileID uint      `json:"integrationProfileId"`
	CanonicalKey         string    `json:"canonicalKey"`
	ExternalCarrierCode  string    `json:"externalCarrierCode"`
	ExternalCarrierName  string    `json:"externalCarrierName"`
	NameKeyStrategy      string    `json:"nameKeyStrategy"`
	InternalCarrierCode  *string   `json:"internalCarrierCode"`
	Status               string    `json:"status"`
	ConflictReason       string    `json:"conflictReason"`
	CreatedAt            time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt            time.Time `json:"updatedAt" ts_type:"string"`
}

type RegisterExternalCarrierInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	ExternalCarrierCode  string `json:"externalCarrierCode"`
	ExternalCarrierName  string `json:"externalCarrierName"`
}

type BindInternalCarrierInput struct {
	ExternalCarrierID   uint   `json:"externalCarrierId"`
	InternalCarrierCode string `json:"internalCarrierCode"`
}
