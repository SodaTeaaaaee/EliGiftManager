package dto

import "time"

// UpdateShipmentInput carries corrections to an existing, non-voided
// shipment (carrier/tracking/shipment numbers, shipped-at timestamp).
// See plan 5.2 "UpdateShipment / VoidShipment".
type UpdateShipmentInput struct {
	ID                 uint       `json:"id"`
	SupplierPlatform   string     `json:"supplierPlatform"`
	ShipmentNo         string     `json:"shipmentNo"`
	ExternalShipmentNo string     `json:"externalShipmentNo"`
	CarrierCode        string     `json:"carrierCode"`
	CarrierName        string     `json:"carrierName"`
	TrackingNo         string     `json:"trackingNo"`
	ShippedAt          *time.Time `json:"shippedAt" ts_type:"string"`
}

// VoidShipmentInput requests a shipment be marked void — a terminal,
// compensating state, not a deletion and not part of the undo/redo command
// history (plan 5.2's "配合撤销边界的如实呈现": the undo boundary is
// deliberately not extended to this operation rather than faked).
//
// FOLLOWUP: the Shipment persistence model has no dedicated
// voidReason/voidedAt/voidedBy column set. Note/OperatorID are carried in the
// shipment's existing ExtraData JSON blob (keys "voidNote"/"voidedBy"/
// "voidedAt") instead of a proper audit column — flagged for the Integrate
// phase / a future migration unit.
type VoidShipmentInput struct {
	ID         uint   `json:"id"`
	Note       string `json:"note"`
	OperatorID string `json:"operatorId"`
}
