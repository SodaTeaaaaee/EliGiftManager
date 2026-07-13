package dto

import "time"

// CustomerFulfillmentHistoryRowDTO is one fulfillment line belonging to a
// customer profile, enriched with cross-wave context (wave/product/tracking)
// so CustomerDetail's "did this person actually get shipped" question can be
// answered in a single call spanning every wave the customer has ever
// appeared in (plan 5.4, filling the CustomerDetail placeholder from
// diagnosis 1.1#3).
type CustomerFulfillmentHistoryRowDTO struct {
	FulfillmentLineID uint      `json:"fulfillmentLineId"`
	WaveID            uint      `json:"waveId"`
	WaveNo            string    `json:"waveNo"`
	WaveName          string    `json:"waveName"`
	ProductID         *uint     `json:"productId"`
	ProductName       string    `json:"productName"`
	ProductSKU        string    `json:"productSku"`
	Quantity          int       `json:"quantity"`
	AllocationState   string    `json:"allocationState"`
	AddressState      string    `json:"addressState"`
	SupplierState     string    `json:"supplierState"`
	ChannelSyncState  string    `json:"channelSyncState"`
	ShipmentID        *uint     `json:"shipmentId"`
	ShipmentStatus    string    `json:"shipmentStatus"`
	TrackingNo        string    `json:"trackingNo"`
	CarrierName       string    `json:"carrierName"`
	CreatedAt         time.Time `json:"createdAt" ts_type:"string"`
}
