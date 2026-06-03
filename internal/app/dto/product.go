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
}

type UpdateProductMasterInput struct {
	ID                 uint   `json:"id"`
	SupplierPlatform   string `json:"supplierPlatform"`
	FactorySKU         string `json:"factorySku"`
	SupplierProductRef string `json:"supplierProductRef"`
	Name               string `json:"name"`
	ProductKind        string `json:"productKind"`
	Archived           bool   `json:"archived"`
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
