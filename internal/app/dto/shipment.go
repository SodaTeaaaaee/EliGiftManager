package dto

type CreateShipmentInput struct {
	SupplierOrderID      uint                      `json:"supplierOrderId"`
	SupplierPlatform     string                    `json:"supplierPlatform"`
	ShipmentNo           string                    `json:"shipmentNo"`
	ExternalShipmentNo   string                    `json:"externalShipmentNo"`
	CarrierCode          string                    `json:"carrierCode"`
	CarrierName          string                    `json:"carrierName"`
	TrackingNo           string                    `json:"trackingNo"`
	Status               string                    `json:"status"`
	ShippedAt            string                    `json:"shippedAt"`
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
	ShippedAt            string            `json:"shippedAt"`
	BasisHistoryNodeID   string            `json:"basisHistoryNodeId"`
	BasisProjectionHash  string            `json:"basisProjectionHash"`
	BasisPayloadSnapshot string            `json:"basisPayloadSnapshot"`
	ExtraData            string            `json:"extraData"`
	CreatedAt            string            `json:"createdAt"`
	UpdatedAt            string            `json:"updatedAt"`
	Lines                []ShipmentLineDTO `json:"lines"`
}

type ShipmentLineDTO struct {
	ID                  uint   `json:"id"`
	ShipmentID          uint   `json:"shipmentId"`
	SupplierOrderLineID uint   `json:"supplierOrderLineId"`
	FulfillmentLineID   uint   `json:"fulfillmentLineId"`
	Quantity            int    `json:"quantity"`
	CreatedAt           string `json:"createdAt"`
}

// ImportShipmentInput represents a bulk shipment import request.
type ImportShipmentInput struct {
	WaveID               uint                  `json:"waveId"`
	IntegrationProfileID uint                  `json:"integrationProfileId"`
	Entries              []ImportShipmentEntry `json:"entries"`
}

// ImportShipmentEntry represents one shipment row from a factory return file.
type ImportShipmentEntry struct {
	SupplierOrderLineID uint   `json:"supplierOrderLineId"`
	FulfillmentLineID   uint   `json:"fulfillmentLineId"`
	ExternalShipmentNo  string `json:"externalShipmentNo"`
	CarrierCode         string `json:"carrierCode"`
	CarrierName         string `json:"carrierName"`
	TrackingNo          string `json:"trackingNo"`
	Quantity            int    `json:"quantity"`
	ShippedAt           string `json:"shippedAt"`
}

// ImportShipmentResult contains the outcome of a bulk shipment import.
type ImportShipmentResult struct {
	CreatedShipments []ShipmentDTO        `json:"createdShipments"`
	Errors           []ImportShipmentError `json:"errors"`
	TotalProcessed   int                  `json:"totalProcessed"`
	SuccessCount     int                  `json:"successCount"`
	ErrorCount       int                  `json:"errorCount"`
}

// ImportShipmentError records a single entry that failed during import.
type ImportShipmentError struct {
	EntryIndex int    `json:"entryIndex"`
	Reason     string `json:"reason"`
}
