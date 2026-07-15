/**
 * Shipment entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'
import type { ShipmentStatus as DomainShipmentStatus } from '@/shared/api/generated/enums'

/** Shipment lifecycle status. */
export type ShipmentStatus = DomainShipmentStatus

/** Import mode for bulk shipment import. */
export type ImportMode = 'reject_all' | 'skip_invalid'

/** Shipment DTO — re-exported from generated model. */
export type Shipment = dto.ShipmentDTO

/** ShipmentLine DTO — re-exported from generated model. */
export type ShipmentLine = dto.ShipmentLineDTO

/** ImportShipmentInput — re-exported from generated model. */
export type ImportShipmentInput = dto.ImportShipmentInput

/** ImportShipmentEntry — re-exported from generated model. */
export type ImportShipmentEntry = dto.ImportShipmentEntry
