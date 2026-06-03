/**
 * Fulfillment entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'

/** The allocation state of a fulfillment line. */
export type AllocationState = 'draft' | 'ready'

/** The address resolution state for this fulfillment line. */
export type AddressState = 'missing' | 'ready' | 'invalid'

/** Supplier-side execution state. */
export type SupplierState =
  | 'not_submitted'
  | 'submitted'
  | 'accepted'
  | 'producing'
  | 'partially_shipped'
  | 'shipped'
  | 'canceled'

/** Channel sync state. */
export type ChannelSyncState =
  | 'not_required'
  | 'unsupported'
  | 'pending'
  | 'synced'
  | 'manual_confirmed'
  | 'skipped'
  | 'failed'

/** Why this fulfillment line exists. */
export type LineReason = 'entitlement' | 'retail_order' | 'wave_adjustment'

/** Kinds of manual adjustments to fulfillment lines within a wave. */
export type AdjustmentKind = 'add' | 'reduce' | 'compensation' | 'remove'

/** How a supplier order was submitted. */
export type SubmissionMode = 'csv' | 'manual' | 'api'

/** Supplier order lifecycle status. */
export type SupplierOrderStatus =
  | 'draft'
  | 'submitted'
  | 'accepted'
  | 'partially_shipped'
  | 'shipped'
  | 'canceled'

/** FulfillmentLine DTO — re-exported from generated model. */
export type FulfillmentLine = dto.FulfillmentLineDTO

/** FulfillmentAdjustment DTO — re-exported from generated model. */
export type FulfillmentAdjustment = dto.FulfillmentAdjustmentDTO

/** SupplierOrder DTO — re-exported from generated model. */
export type SupplierOrder = dto.SupplierOrderDTO

/** SupplierOrderLine DTO — re-exported from generated model. */
export type SupplierOrderLine = dto.SupplierOrderLineDTO
