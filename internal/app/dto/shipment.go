package dto

import "time"

type CreateShipmentInput struct {
	SupplierOrderID      uint                      `json:"supplierOrderId"`
	SupplierPlatform     string                    `json:"supplierPlatform"`
	ShipmentNo           string                    `json:"shipmentNo"`
	ExternalShipmentNo   string                    `json:"externalShipmentNo"`
	CarrierCode          string                    `json:"carrierCode"`
	CarrierName          string                    `json:"carrierName"`
	TrackingNo           string                    `json:"trackingNo"`
	Status               string                    `json:"status"`
	ShippedAt            *time.Time                `json:"shippedAt" ts_type:"string"`
	BasisPayloadSnapshot string                    `json:"basisPayloadSnapshot"`
	Lines                []CreateShipmentLineInput `json:"lines"`
}

type CreateShipmentLineInput struct {
	SupplierOrderLineID uint `json:"supplierOrderLineId"`
	FulfillmentLineID   uint `json:"fulfillmentLineId"`
	Quantity            int  `json:"quantity"`
}

type ShipmentDTO struct {
	ID                   uint              `json:"id"`
	SupplierOrderID      uint              `json:"supplierOrderId"`
	SupplierPlatform     string            `json:"supplierPlatform"`
	ShipmentNo           string            `json:"shipmentNo"`
	ExternalShipmentNo   string            `json:"externalShipmentNo"`
	CarrierCode          string            `json:"carrierCode"`
	CarrierName          string            `json:"carrierName"`
	TrackingNo           string            `json:"trackingNo"`
	Status               string            `json:"status"`
	ShippedAt            *time.Time        `json:"shippedAt" ts_type:"string"`
	BasisHistoryNodeID   string            `json:"basisHistoryNodeId"`
	BasisProjectionHash  string            `json:"basisProjectionHash"`
	BasisPayloadSnapshot string            `json:"basisPayloadSnapshot"`
	ExtraData            string            `json:"extraData"`
	CreatedAt            time.Time         `json:"createdAt" ts_type:"string"`
	UpdatedAt            time.Time         `json:"updatedAt" ts_type:"string"`
	Lines                []ShipmentLineDTO `json:"lines"`
}

type ShipmentLineDTO struct {
	ID                  uint      `json:"id"`
	ShipmentID          uint      `json:"shipmentId"`
	SupplierOrderLineID uint      `json:"supplierOrderLineId"`
	FulfillmentLineID   uint      `json:"fulfillmentLineId"`
	Quantity            int       `json:"quantity"`
	CreatedAt           time.Time `json:"createdAt" ts_type:"string"`
}

// ImportShipmentInput represents a bulk shipment import request.
type ImportShipmentInput struct {
	WaveID               uint                  `json:"waveId"`
	IntegrationProfileID uint                  `json:"integrationProfileId"`
	ImportMode           string                `json:"importMode"` // "reject_all" | "skip_invalid" (default: "skip_invalid")
	Entries              []ImportShipmentEntry `json:"entries"`
}

// ImportShipmentEntry represents one shipment row from a factory return file.
type ImportShipmentEntry struct {
	SupplierOrderLineID uint       `json:"supplierOrderLineId"`
	FulfillmentLineID   uint       `json:"fulfillmentLineId"`
	ExternalShipmentNo  string     `json:"externalShipmentNo"`
	CarrierCode         string     `json:"carrierCode"`
	CarrierName         string     `json:"carrierName"`
	TrackingNo          string     `json:"trackingNo"`
	Quantity            int        `json:"quantity"`
	ShippedAt           *time.Time `json:"shippedAt" ts_type:"string"`
}

// ImportShipmentResult contains the outcome of a bulk shipment import.
type ImportShipmentResult struct {
	ImportRunID      uint                  `json:"importRunId"`
	EvidenceDisabled bool                  `json:"evidenceDisabled"`
	CreatedShipments []ShipmentDTO         `json:"createdShipments"`
	Errors           []ImportShipmentError `json:"errors"`
	TotalProcessed   int                   `json:"totalProcessed"`
	SuccessCount     int                   `json:"successCount"`
	ErrorCount       int                   `json:"errorCount"`
	// Warnings are non-blocking, row-level mapping warnings (e.g. mapping
	// dests outside the global legal vocabulary) surfaced by the
	// MapAndReconcileShipments template-mapping pass — values are still kept
	// and imported, but flagged so the operator can review them.
	Warnings []string `json:"warnings"`
}

// ImportShipmentError records a single entry that failed during import.
type ImportShipmentError struct {
	EntryIndex int    `json:"entryIndex"`
	Reason     string `json:"reason"`
}

// MapAndReconcileShipmentsInput maps an external factory-return sheet onto internal IDs
// then runs ImportShipments. FilePath is preferred; Rows is for pre-parsed callers/tests.
type MapAndReconcileShipmentsInput struct {
	WaveID               uint                `json:"waveId"`
	IntegrationProfileID uint                `json:"integrationProfileId"`
	ImportMode           string              `json:"importMode"` // "reject_all" | "skip_invalid"
	FilePath             string              `json:"filePath"`
	Rows                 []map[string]string `json:"rows"`
}
