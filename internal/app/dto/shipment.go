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
