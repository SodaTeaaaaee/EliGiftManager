/**
 * Fulfillment entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'
import type {
  AddressState as DomainAddressState,
  AdjustmentKind as DomainAdjustmentKind,
  AllocationState as DomainAllocationState,
  ChannelSyncState as DomainChannelSyncState,
  FulfillmentLineReason,
  SubmissionMode as DomainSubmissionMode,
  SupplierOrderStatus as DomainSupplierOrderStatus,
  SupplierState as DomainSupplierState,
} from '@/shared/api/generated/enums'

/** The allocation state of a fulfillment line. */
export type AllocationState = DomainAllocationState

/** The address resolution state for this fulfillment line. */
export type AddressState = DomainAddressState

/** Supplier-side execution state. */
export type SupplierState = DomainSupplierState

/** Channel sync state. */
export type ChannelSyncState = DomainChannelSyncState

/** Why this fulfillment line exists. */
export type LineReason = FulfillmentLineReason

/** Kinds of manual adjustments to fulfillment lines within a wave. */
export type AdjustmentKind = DomainAdjustmentKind

/** Whether a fulfillment line's review needs human attention (wave-level, stamped per row). */
export type ReviewRequirement = 'none' | 'recommended' | 'required'

/** Whether a wave's fulfillment basis has drifted from its upstream source (wave-level, stamped per row). */
export type BasisDriftStatus = 'drifted' | 'in_sync'

/** How a supplier order was submitted. */
export type SubmissionMode = DomainSubmissionMode

/** Supplier order lifecycle status. */
export type SupplierOrderStatus = DomainSupplierOrderStatus

/** FulfillmentLine DTO — re-exported from generated model. */
export type FulfillmentLine = dto.FulfillmentLineDTO

/** FulfillmentAdjustment DTO — re-exported from generated model. */
export type FulfillmentAdjustment = dto.FulfillmentAdjustmentDTO

/** SupplierOrder DTO — re-exported from generated model. */
export type SupplierOrder = dto.SupplierOrderDTO

/** SupplierOrderLine DTO — re-exported from generated model. */
export type SupplierOrderLine = dto.SupplierOrderLineDTO

/** WaveFulfillmentRow DTO — re-exported from generated model (grid row shape). */
export type WaveFulfillmentRow = dto.WaveFulfillmentRowDTO

/** WaveFulfillmentRowsPage DTO — re-exported from generated model (server-paginated grid page). */
export type WaveFulfillmentRowsPage = dto.WaveFulfillmentRowsPage
