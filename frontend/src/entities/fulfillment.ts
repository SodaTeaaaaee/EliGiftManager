/** The allocation state of a fulfillment line. */
export type AllocationState =
  | "unallocated"
  | "allocated"
  | "adjusted"
  | "final";

/** The address resolution state for this fulfillment line. */
export type AddressState =
  | "pending"
  | "resolved"
  | "overridden"
  | "missing"
  | "waived";

/** Supplier-side execution state. */
export type SupplierState =
  | "pending"
  | "queued"
  | "submitted"
  | "accepted"
  | "rejected"
  | "shipped";

/** Channel sync state (placeholder for future channel integration). */
export type ChannelSyncState =
  | "not_required"
  | "pending"
  | "synced"
  | "failed";

/** Why this fulfillment line exists. */
export type LineReason =
  | "entitlement"
  | "retail_order"
  | "wave_adjustment";

/**
 * FulfillmentLine (aligned to Go dto.FulfillmentLineDTO).
 * Go `*uint` fields → TS `number | null`.
 * Go `string` fields → TS `string`.
 */
export interface FulfillmentLine {
  id: number;
  waveId: number;
  customerProfileId: number;
  waveParticipantSnapshotId: number;
  productId: number | null;
  demandDocumentId: number | null;
  demandLineId: number | null;
  customerAddressId: number | null;
  quantity: number;
  allocationState: string;
  addressState: string;
  supplierState: string;
  channelSyncState: string;
  lineReason: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/** Kinds of manual adjustments to fulfillment lines within a wave. */
export type AdjustmentKind =
  | "add"
  | "reduce"
  | "compensation"
  | "remove";

/**
 * An explicit adjustment made against a fulfillment line during wave review.
 */
export interface FulfillmentAdjustment {
  id: number;
  waveId: number;
  fulfillmentLineId: number | null;
  waveParticipantSnapshotId: number | null;
  adjustmentKind: AdjustmentKind;
  quantityDelta: number;
  reasonCode: string | null;
  note: string;
  operatorId: string;
  evidenceRef: string | null;
  createdAt: string;
  updatedAt: string;
}

/** How a supplier order was submitted. */
export type SubmissionMode = "csv" | "manual" | "api";

/** Supplier order lifecycle status. */
export type SupplierOrderStatus =
  | "draft"
  | "submitted"
  | "accepted"
  | "partially_shipped"
  | "shipped"
  | "canceled";

/**
 * SupplierOrder (aligned to Go dto.SupplierOrderDTO).
 * Go `string` fields → TS `string` (templateId is `string`, not `number | null`).
 */
export interface SupplierOrder {
  id: number;
  waveId: number;
  supplierPlatform: string;
  templateId: string;
  batchNo: string;
  externalOrderNo: string;
  submissionMode: string;
  submittedAt: string;
  status: string;
  requestPayload: string;
  responsePayload: string;
  basisHistoryNodeId: string;
  basisProjectionHash: string;
  basisPayloadSnapshot: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * SupplierOrderLine (aligned to Go dto.SupplierOrderLineDTO).
 * Go `*int` fields → TS `number | null`.
 */
export interface SupplierOrderLine {
  id: number;
  supplierOrderId: number;
  fulfillmentLineId: number;
  supplierLineNo: number | null;
  supplierSku: string;
  submittedQuantity: number;
  acceptedQuantity: number | null;
  status: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}
