package dto

import "time"

// CarrierMappingDTO is returned by carrier mapping queries.
type CarrierMappingDTO struct {
	ID                   uint      `json:"id"`
	IntegrationProfileID uint      `json:"integrationProfileId"`
	InternalCarrierCode  string    `json:"internalCarrierCode"`
	ExternalCarrierCode  string    `json:"externalCarrierCode"`
	ExternalCarrierName  string    `json:"externalCarrierName"`
	Aliases              string    `json:"aliases"` // JSON []string
	IsDefault            bool      `json:"isDefault"`
	CreatedAt            time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt            time.Time `json:"updatedAt" ts_type:"string"`
}

// CreateCarrierMappingInput is the input for creating a carrier mapping.
type CreateCarrierMappingInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	InternalCarrierCode  string `json:"internalCarrierCode"`
	ExternalCarrierCode  string `json:"externalCarrierCode"`
	ExternalCarrierName  string `json:"externalCarrierName"`
	Aliases              string `json:"aliases"` // JSON []string
	IsDefault            bool   `json:"isDefault"`
}

// ImportCarrierMappingsInput drives template-mapped carrier mapping upsert.
type ImportCarrierMappingsInput struct {
	IntegrationProfileID uint                `json:"integrationProfileId"`
	ImportMode           string              `json:"importMode"` // "reject_all" | "skip_invalid"
	FilePath             string              `json:"filePath"`
	Rows                 []map[string]string `json:"rows"`
}

// ImportCarrierMappingsResult is the dual-mode outcome of a carrier mapping import.
type ImportCarrierMappingsResult struct {
	CreatedCount   int                         `json:"createdCount"`
	UpdatedCount   int                         `json:"updatedCount"`
	TotalProcessed int                         `json:"totalProcessed"`
	SuccessCount   int                         `json:"successCount"`
	ErrorCount     int                         `json:"errorCount"`
	Errors         []ImportCarrierMappingError `json:"errors"`
	Mappings       []CarrierMappingDTO         `json:"mappings"`
	// Warnings are non-blocking, row-level mapping warnings (e.g. mapping
	// dests outside the global legal vocabulary) — values are still kept and
	// imported, but flagged so the operator can review them.
	Warnings []string `json:"warnings"`
}

// ImportCarrierMappingError records one failed carrier mapping row.
type ImportCarrierMappingError struct {
	RowIndex int    `json:"rowIndex"`
	Reason   string `json:"reason"`
}
