// DO NOT EDIT. Generated from internal/domain/enums.go.
// Regenerate with: deno task gen:enums

export const identityTypeValues = [
  'platform_uid',
  'email',
  'username',
  'external_buyer_id',
] as const

export type IdentityType = (typeof identityTypeValues)[number]

export const platformKindValues = [
  'source',
  'factory',
] as const

export type PlatformKind = (typeof platformKindValues)[number]

export const inputFactKindValues = [
  'membership',
  'retail_order',
  'operator_grant',
] as const

export type InputFactKind = (typeof inputFactKindValues)[number]

export const templateDirectionValues = [
  'input',
  'output',
] as const

export type TemplateDirection = (typeof templateDirectionValues)[number]

export const waveCloseResultValues = [
  'open',
  'clean',
  'residual',
] as const

export type WaveCloseResult = (typeof waveCloseResultValues)[number]

export const entitlementSelectorTypeValues = [
  'platform_level',
  'wave_all',
  'instance',
] as const

export type EntitlementSelectorType = (typeof entitlementSelectorTypeValues)[number]

export const fulfillmentSourceKindValues = [
  'entitlement_instance',
  'retail_line',
  'operator_grant',
] as const

export type FulfillmentSourceKind = (typeof fulfillmentSourceKindValues)[number]

export const blockReasonValues = [
  'unaligned_product',
  'unusable_address',
  'identity_unattached',
  'quantity_split_not_summing',
] as const

export type BlockReason = (typeof blockReasonValues)[number]

export const supplierOrderStatusValues = [
  'draft',
  'generated',
  'exported',
  'voided',
] as const

export type SupplierOrderStatus = (typeof supplierOrderStatusValues)[number]

export const duplicateVerdictValues = [
  'record_only',
  'ask_operator',
  'new_responsibility',
] as const

export type DuplicateVerdict = (typeof duplicateVerdictValues)[number]

export const writebackStatusValues = [
  'pending',
  'sent',
  'failed',
] as const

export type WritebackStatus = (typeof writebackStatusValues)[number]

export const workStateValues = [
  'blocked',
  'ready',
  'in_factory',
  'shipped',
  'writeback_failed',
] as const

export type WorkState = (typeof workStateValues)[number]
