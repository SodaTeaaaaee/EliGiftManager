/**
 * Shipment entity definitions — aligned to Go dto.ShipmentDTO / dto.ShipmentLineDTO.
 * JSON tags in the Go DTO are the authoritative field names.
 * Go `uint` / `int` → TS `number`; Go `string` → TS `string`.
 */

/** Shipment lifecycle status. */
export type ShipmentStatus =
  | 'pending'
  | 'shipped'
  | 'in_transit'
  | 'delivered'
  | 'exception'
  | 'returned'

/**
 * Shipment (aligned to Go dto.ShipmentDTO).
 */
export interface Shipment {
  id: number
  supplierOrderId: number
  supplierPlatform: string
  shipmentNo: string
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  status: string
  shippedAt: string
  basisHistoryNodeId: string
  basisProjectionHash: string
  basisPayloadSnapshot: string
  extraData: string
  createdAt: string
  updatedAt: string
  lines: ShipmentLine[]
}

/** Import mode for bulk shipment import. */
export type ImportMode = 'reject_all' | 'skip_invalid'

/**
 * Input for bulk shipment import (aligned to Go dto.ImportShipmentInput).
 */
export interface ImportShipmentInput {
  waveId: number
  integrationProfileId: number
  importMode?: ImportMode
  entries: ImportShipmentEntry[]
}

/**
 * One row in a bulk shipment import (aligned to Go dto.ImportShipmentEntry).
 */
export interface ImportShipmentEntry {
  supplierOrderLineId: number
  fulfillmentLineId: number
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  quantity: number
  shippedAt: string
}

/**
 * ShipmentLine (aligned to Go dto.ShipmentLineDTO).
 */
export interface ShipmentLine {
  id: number
  shipmentId: number
  supplierOrderLineId: number
  fulfillmentLineId: number
  quantity: number
  createdAt: string
}
