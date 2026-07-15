// DO NOT EDIT. Generated from internal/domain/enums.go.
// Regenerate with: deno task gen:enums

export const profileTypeValues = [
  'member',
  'buyer',
  'mixed',
  'manual',
] as const

export type ProfileType = (typeof profileTypeValues)[number]

export const identityTypeValues = [
  'platform_uid',
  'email',
  'username',
  'external_buyer_id',
] as const

export type IdentityType = (typeof identityTypeValues)[number]

export const demandKindValues = [
  'membership_entitlement',
  'retail_order',
] as const

export type DemandKind = (typeof demandKindValues)[number]

export const captureModeValues = [
  'document_import',
  'api_ingest',
  'manual_entry',
] as const

export type CaptureMode = (typeof captureModeValues)[number]

export const demandLineTypeValues = [
  'entitlement_rule',
  'sku_order',
  'manual_entry',
] as const

export type DemandLineType = (typeof demandLineTypeValues)[number]

export const obligationTriggerKindValues = [
  'periodic_membership',
  'loyalty_membership',
  'supporter_only_purchase',
  'member_only_discount_purchase',
  'campaign_reward',
  'manual_compensation',
] as const

export type ObligationTriggerKind = (typeof obligationTriggerKindValues)[number]

export const entitlementAuthorityValues = [
  'local_policy',
  'upstream_platform',
  'manual_grant',
] as const

export type EntitlementAuthority = (typeof entitlementAuthorityValues)[number]

export const recipientInputStateValues = [
  'not_required',
  'waiting_for_input',
  'partially_collected',
  'ready',
  'waived',
  'expired',
] as const

export type RecipientInputState = (typeof recipientInputStateValues)[number]

export const routingDispositionValues = [
  'pending_intake',
  'accepted',
  'deferred',
  'excluded_manual',
  'excluded_duplicate',
  'excluded_revoked',
] as const

export type RoutingDisposition = (typeof routingDispositionValues)[number]

export const waveTypeValues = [
  'membership',
  'retail',
  'mixed',
] as const

export type WaveType = (typeof waveTypeValues)[number]

export const snapshotTypeValues = [
  'member',
  'buyer',
  'mixed',
] as const

export type SnapshotType = (typeof snapshotTypeValues)[number]

export const fulfillmentLineReasonValues = [
  'entitlement',
  'retail_order',
  'wave_adjustment',
] as const

export type FulfillmentLineReason = (typeof fulfillmentLineReasonValues)[number]

export const submissionModeValues = [
  'csv',
  'manual',
  'api',
] as const

export type SubmissionMode = (typeof submissionModeValues)[number]

export const supplierOrderStatusValues = [
  'draft',
  'submitted',
  'accepted',
  'partially_shipped',
  'shipped',
  'canceled',
] as const

export type SupplierOrderStatus = (typeof supplierOrderStatusValues)[number]

export const shipmentStatusValues = [
  'pending',
  'shipped',
  'in_transit',
  'delivered',
  'exception',
  'returned',
  'voided',
] as const

export type ShipmentStatus = (typeof shipmentStatusValues)[number]

export const adjustmentKindValues = [
  'add',
  'reduce',
  'compensation',
  'remove',
  'replace',
  'reissue',
] as const

export type AdjustmentKind = (typeof adjustmentKindValues)[number]

export const allocationStateValues = [
  'draft',
  'ready',
] as const

export type AllocationState = (typeof allocationStateValues)[number]

export const addressStateValues = [
  'missing',
  'ready',
  'invalid',
] as const

export type AddressState = (typeof addressStateValues)[number]

export const addressValidationStatusValues = [
  'unvalidated',
  'valid',
  'invalid',
] as const

export type AddressValidationStatus = (typeof addressValidationStatusValues)[number]

export const supplierStateValues = [
  'not_submitted',
  'submitted',
  'accepted',
  'producing',
  'partially_shipped',
  'shipped',
  'canceled',
] as const

export type SupplierState = (typeof supplierStateValues)[number]

export const channelSyncStateValues = [
  'not_required',
  'unsupported',
  'pending',
  'synced',
  'manual_confirmed',
  'skipped',
  'failed',
] as const

export type ChannelSyncState = (typeof channelSyncStateValues)[number]

export const lifecycleStageValues = [
  'intake',
  'allocation',
  'review',
  'execution',
  'syncing_back',
  'awaiting_manual_closure',
  'closed',
] as const

export type LifecycleStage = (typeof lifecycleStageValues)[number]

export const productKindValues = [
  'badge',
  'standee',
  'charm',
  'postcard',
  'print',
  'bundle',
  'other',
] as const

export type ProductKind = (typeof productKindValues)[number]

export const businessSurfaceValues = [
  'membership',
  'retail',
  'factory',
] as const

export type BusinessSurface = (typeof businessSurfaceValues)[number]

export const sourceSurfaceValues = [
  'membership',
  'retail',
  'factory',
] as const

export type SourceSurface = (typeof sourceSurfaceValues)[number]
