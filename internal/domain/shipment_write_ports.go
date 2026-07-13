package domain

import "context"

// ShipmentWriteRepository provides mutation operations for existing Shipment
// records — corrections (UpdateShipment) and the terminal void transition
// (VoidShipment) — that the read/create-oriented ShipmentRepository (ports.go)
// does not cover. Both operations funnel through Update: VoidShipment is
// implemented at the app layer as a status transition plus an Update call,
// not a distinct repository method, so this repository stays a single-method
// seam. See plan 5.2 "UpdateShipment / VoidShipment".
type ShipmentWriteRepository interface {
	// Update persists all mutable fields of an existing shipment (identified
	// by shipment.ID) and refreshes shipment in place with the stored row
	// (including UpdatedAt).
	Update(ctx context.Context, shipment *Shipment) error
}
