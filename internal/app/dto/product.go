package dto

import "time"

// ---- ProductMaster ----

type ProductMasterDTO struct {
	ID                 uint      `json:"id"`
	SupplierPlatform   string    `json:"supplierPlatform"`
	FactorySKU         string    `json:"factorySku"`
	SupplierProductRef string    `json:"supplierProductRef"`
	Name               string    `json:"name"`
	ProductKind        string    `json:"productKind"`
	Archived           bool      `json:"archived"`
	CoverImagePath     string    `json:"coverImagePath"`
	DetailImagePaths   string    `json:"detailImagePaths"` // JSON []string
	ExtraData          string    `json:"extraData"`
	CreatedAt          time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt          time.Time `json:"updatedAt" ts_type:"string"`
}

type CreateProductMasterInput struct {
	SupplierPlatform   string `json:"supplierPlatform"`
	FactorySKU         string `json:"factorySku"`
	SupplierProductRef string `json:"supplierProductRef"`
	Name               string `json:"name"`
	ProductKind        string `json:"productKind"`
	CoverImagePath     string `json:"coverImagePath"`
	DetailImagePaths   string `json:"detailImagePaths"` // JSON []string
	ExtraData          string `json:"extraData"`
}

type UpdateProductMasterInput struct {
	ID                 uint   `json:"id"`
	SupplierPlatform   string `json:"supplierPlatform"`
	FactorySKU         string `json:"factorySku"`
	SupplierProductRef string `json:"supplierProductRef"`
	Name               string `json:"name"`
	ProductKind        string `json:"productKind"`
	Archived           bool   `json:"archived"`
	CoverImagePath     string `json:"coverImagePath"`
	DetailImagePaths   string `json:"detailImagePaths"` // JSON []string
	ExtraData          string `json:"extraData"`
}

// ---- Product (wave-scoped snapshot) ----

type ProductDTO struct {
	ID               uint      `json:"id"`
	WaveID           uint      `json:"waveId"`
	ProductMasterID  *uint     `json:"productMasterId"`
	SupplierPlatform string    `json:"supplierPlatform"`
	FactorySKU       string    `json:"factorySku"`
	Name             string    `json:"name"`
	ExtraData        string    `json:"extraData"`
	CreatedAt        time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt        time.Time `json:"updatedAt" ts_type:"string"`
}

type SnapshotProductsInput struct {
	WaveID    uint   `json:"waveId"`
	MasterIDs []uint `json:"masterIds"`
}

// ImportProductCatalogInput drives template-mapped ProductMaster upsert.
// Prefer FilePath (tabular); Rows is accepted for unit tests / pre-parsed callers.
type ImportProductCatalogInput struct {
	IntegrationProfileID uint                `json:"integrationProfileId"`
	ImportMode           string              `json:"importMode"` // "reject_all" | "skip_invalid"
	FilePath             string              `json:"filePath"`
	Rows                 []map[string]string `json:"rows"`
}

// ImportProductCatalogResult is the dual-mode outcome of a catalog import.
type ImportProductCatalogResult struct {
	CreatedCount   int                         `json:"createdCount"`
	UpdatedCount   int                         `json:"updatedCount"`
	TotalProcessed int                         `json:"totalProcessed"`
	SuccessCount   int                         `json:"successCount"`
	ErrorCount     int                         `json:"errorCount"`
	Errors         []ImportProductCatalogError `json:"errors"`
	Masters        []ProductMasterDTO          `json:"masters"`
	// Warnings are non-blocking, row-level mapping warnings (e.g. mapping
	// dests outside the global legal vocabulary) — values are still kept and
	// imported, but flagged so the operator can review them.
	Warnings []string `json:"warnings"`
}

// ImportProductCatalogError records one failed catalog row.
type ImportProductCatalogError struct {
	RowIndex int    `json:"rowIndex"`
	Reason   string `json:"reason"`
}
