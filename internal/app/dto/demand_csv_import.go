package dto

// DemandCSVImportError records a single row that failed template-driven mapping.
type DemandCSVImportError struct {
	RowIndex int    `json:"rowIndex"`
	Reason   string `json:"reason"`
}

// ImportDemandCSVInput represents a dual-mode (reject_all / skip_invalid) demand CSV import
// request. Rows are header-keyed maps, typically produced by DemandController.ParseCSVFile.
// When FilePath is set, the backend re-reads the tabular file (hasHeader from mapping rules)
// and ignores Rows — preferred for positional mapping and multi-format sheets.
type ImportDemandCSVInput struct {
	IntegrationProfileID uint                `json:"integrationProfileId"`
	DocumentType         string              `json:"documentType"`
	SourceDocumentNo     string              `json:"sourceDocumentNo"`
	SourceCustomerRef    string              `json:"sourceCustomerRef"`
	ImportMode           string              `json:"importMode"` // "reject_all" | "skip_invalid"; default "skip_invalid" when empty
	Rows                 []map[string]string `json:"rows"`
	FilePath             string              `json:"filePath"`
}

// ImportDemandCSVResult contains the outcome of a dual-mode demand CSV import.
type ImportDemandCSVResult struct {
	Document       *DemandDocumentDTO     `json:"document"` // nil when nothing was persisted
	Errors         []DemandCSVImportError `json:"errors"`
	TotalProcessed int                    `json:"totalProcessed"`
	SuccessCount   int                    `json:"successCount"`
	ErrorCount     int                    `json:"errorCount"`
	// Warnings are non-blocking, row-level mapping warnings (e.g. mapping
	// dests outside the global legal vocabulary) — values are still kept and
	// imported, but flagged so the operator can review them.
	Warnings []string `json:"warnings"`
}
