/**
 * The domain-terminology layer (plan section 2.2). This is the ONLY place
 * that maps a raw backend enum value to a display label / tooltip / status
 * tone. `<StatusBadge>` / `<StatusDot>` / `<StatusLegend>` must resolve every
 * status they render through `useGlossary()` — never print a raw enum value.
 *
 * Enum value sets are kept in lockstep with the backend domain enums and the
 * corresponding entity string unions in this frontend.
 */
import type {
  AddressState,
  AdjustmentKind,
  AllocationState,
  CaptureMode,
  ChannelSyncState,
  DemandKind,
  FulfillmentLineReason,
  IdentityType,
  LifecycleStage,
  ProductKind,
  ProfileType,
  RecipientInputState,
  RoutingDisposition,
  ShipmentStatus,
  SubmissionMode,
  SupplierOrderStatus,
  SupplierState,
} from '@/shared/api/generated/enums'
import { i18n } from './index'

/** The 6 status token families the theme system defines (`--status-*-fg/bg/border`). */
export type StatusTone = 'success' | 'warning' | 'error' | 'info' | 'progress' | 'neutral'

export type LifecycleStageValue = LifecycleStage

export type RoutingDispositionValue = RoutingDisposition

export type RecipientInputStateValue = RecipientInputState

export type AddressStateValue = AddressState

export type SupplierStateValue = SupplierState

export type ChannelSyncStateValue = ChannelSyncState

export type AllocationStateValue = AllocationState

export type ShipmentStatusValue = ShipmentStatus

/** `domain.SubmissionMode` (`internal/domain/enums.go:122`) — how a supplier order was submitted. */
export type SubmissionModeValue = SubmissionMode

export type DemandKindValue = DemandKind

/** `domain.CaptureMode` (`internal/domain/enums.go:32-38`) — how a DemandDocument entered the inbox. */
export type CaptureModeValue = CaptureMode

export type AdjustmentKindValue = AdjustmentKind

export type ProductKindValue = ProductKind

/** `domain.ProfileType` (`internal/domain/enums.go:7-10`) — a customer profile's classification. */
export type ProfileTypeValue = ProfileType

/** `domain.IdentityType` (`internal/domain/enums.go:17-20`) — how a `CustomerIdentity` was resolved. */
export type IdentityTypeValue = IdentityType

/** Projected single-value summary of the two-axis basis-drift model (plan 3.3.1). */
export type DriftSummaryValue = 'in_sync' | 'drifted_none' | 'drifted_recommended' | 'drifted_required'

/** Whether a fulfillment line's review needs human attention. Computed once per wave and
 * stamped identically on every `WaveFulfillmentRowDTO` row (see fulfillment-grid CANON) —
 * distinct data source from `driftSummary` above (overview-level, 4-value projection). */
export type ReviewRequirementValue = 'none' | 'recommended' | 'required'

/** Why a fulfillment line exists (`WaveFulfillmentRowDTO.lineReason`). */
export type LineReasonValue = FulfillmentLineReason

/** Wave-level basis-drift flag stamped on every fulfillment row (`WaveFulfillmentRowDTO.basisDriftStatus`).
 * Distinct dimension from `driftSummary` (4-value overview projection) and from
 * `waveWorkspace.drift.reviewRequirement` copy — different value set, different data source. */
export type BasisDriftStatusValue = 'drifted' | 'in_sync'

/** `SelectorPayload.type` (`internal/domain/models.go`) — which participants an allocation
 * policy rule applies to. Fixed 4-value union (not free text), so it must render through
 * `StatusBadge`, not a plain text cell (P4 allocation tab, `WaveAllocationTab.vue`). */
export type AllocationSelectorTypeValue = 'wave_all' | 'platform_all' | 'identity_level' | 'explicit_override'

/** `DemandMappingBlockedLine.Reason` (`internal/app/use_cases.go` demand→fulfillment mapping) —
 * fixed 2-value set, both branches hardcoded server-side (not free text). P4 allocation tab's
 * mapping-result panel. */
export type DemandMappingBlockedReasonValue = 'wave_product_missing' | 'address_unavailable'

/** `CreateProfileInput.initialAllocationStrategy` (`internal/app/profile_usecase.go` enum
 * whitelist) — P4 integrations page (profile detail + intake wizard confirm step). */
export type InitialAllocationStrategyValue = 'policy_driven' | 'demand_driven'

/** `CreateProfileInput.identityStrategy` — how a customer identity is resolved for this
 * integration profile's demand imports. */
export type IdentityStrategyValue = 'platform_uid' | 'email' | 'external_buyer_id'

/** `CreateProfileInput.entitlementAuthorityMode` — who asserts an entitlement is valid for
 * this profile's demand. */
export type EntitlementAuthorityModeValue = 'local_policy' | 'upstream_platform' | 'manual_grant_only'

/** `CreateProfileInput.recipientInputMode` — how recipient information is collected for this
 * profile's demand lines. */
export type RecipientInputModeValue = 'none' | 'platform_claim' | 'external_form' | 'manual_collection'

/** `CreateProfileInput.referenceStrategy` — the granularity at which this profile's demand
 * references its source (member/order/order-line level). */
export type ReferenceStrategyValue = 'member_level' | 'order_level' | 'order_line_level'

/** `CreateProfileInput.trackingSyncMode` — how shipment/tracking info is synced back to the
 * source platform for this profile. */
export type TrackingSyncModeValue = 'api_push' | 'document_export' | 'manual_confirmation' | 'unsupported'

/** `CreateProfileInput.closurePolicy` — when a fulfillment line under this profile is
 * considered closed. */
export type ClosurePolicyValue = 'close_after_sync' | 'close_after_manual_confirmation' | 'close_after_shipment'

/** `DocumentTemplate.DocumentType` / `validDocumentTypes` (`internal/app/template_usecase.go`)
 * — fixed 6-value whitelist. P4 integrations detail drawer's template/binding lists. */
export type DocumentTypeValue =
  | 'import_entitlement'
  | 'import_sales_order'
  | 'import_product_catalog'
  | 'export_supplier_order'
  | 'import_supplier_shipment'
  | 'export_source_tracking_update'

/** `SupplierOrderDTO.status` (`internal/domain/enums.go:131-140`) — the supplier ORDER's own
 * lifecycle status. Distinct from `supplierState` above (the fulfillment-LINE's *projected*
 * supplier state — different value set: `not_submitted`/`producing` instead of `draft`).
 * P5 factory-orders tab (`WaveFactoryTab.vue`). */
export type SupplierOrderStatusValue = SupplierOrderStatus

/** `ChannelSyncJob.Status` (`internal/domain/models.go:307-323`, raw string field, no domain
 * enum consts exist). Distinct from `channelSyncState` above (`FulfillmentLine.ChannelSyncState`
 * — a different enum on a different entity). P5 closure tab's jobs table (`WaveClosureTab.vue`). */
export type ChannelSyncJobStatusValue = 'pending' | 'running' | 'success' | 'failed' | 'partial_success'

/** `ChannelSyncItem.Status` (`internal/domain/models.go:336`, raw string field). Distinct from
 * `channelSyncJobStatus` above (the JOB's aggregate status) — this is the per-ITEM outcome,
 * a 2-value set hardcoded by every executor (`ChannelSyncItemResult.Status`,
 * `internal/app/executor.go:138-141`, `csv_export_executor.go:88`). P5 closure tab's
 * `JobItemsDrawer.vue` item-detail table. */
export type ChannelSyncItemStatusValue = 'success' | 'failed'

/** `RecordClosureDecisionEntry.DecisionKind` (`internal/app/closure_action_usecase.go:73-77`)
 * — fixed 3-value whitelist validated server-side. P5 closure tab's manual decision form.
 * NOTE: `RecordClosureDecisionEntry.ReasonCode` is confirmed free text
 * (`internal/domain/enums.go:156`) — it is NOT a glossary dimension, render as plain text. */
export type ClosureDecisionKindValue =
  | 'mark_sync_unsupported'
  | 'mark_sync_skipped'
  | 'mark_sync_completed_manually'

/** One map entry per glossary dimension -> the value union it accepts. */
export interface GlossaryDimensionValueMap {
  lifecycleStage: LifecycleStageValue
  routingDisposition: RoutingDispositionValue
  recipientInputState: RecipientInputStateValue
  addressState: AddressStateValue
  supplierState: SupplierStateValue
  channelSyncState: ChannelSyncStateValue
  allocationState: AllocationStateValue
  shipmentStatus: ShipmentStatusValue
  submissionMode: SubmissionModeValue
  demandKind: DemandKindValue
  adjustmentKind: AdjustmentKindValue
  productKind: ProductKindValue
  profileType: ProfileTypeValue
  identityType: IdentityTypeValue
  driftSummary: DriftSummaryValue
  reviewRequirement: ReviewRequirementValue
  lineReason: LineReasonValue
  basisDriftStatus: BasisDriftStatusValue
  allocationSelectorType: AllocationSelectorTypeValue
  demandMappingBlockedReason: DemandMappingBlockedReasonValue
  initialAllocationStrategy: InitialAllocationStrategyValue
  identityStrategy: IdentityStrategyValue
  entitlementAuthorityMode: EntitlementAuthorityModeValue
  recipientInputMode: RecipientInputModeValue
  referenceStrategy: ReferenceStrategyValue
  trackingSyncMode: TrackingSyncModeValue
  closurePolicy: ClosurePolicyValue
  documentType: DocumentTypeValue
  supplierOrderStatus: SupplierOrderStatusValue
  channelSyncJobStatus: ChannelSyncJobStatusValue
  channelSyncItemStatus: ChannelSyncItemStatusValue
  closureDecisionKind: ClosureDecisionKindValue
  captureMode: CaptureModeValue
}

export type GlossaryDimension = keyof GlossaryDimensionValueMap

export interface GlossaryEntry {
  /** i18n message key resolving to the display label, e.g. `glossary.lifecycleStage.intake.label`. */
  labelKey: string
  /** i18n message key resolving to the one-sentence tooltip explanation. */
  descKey: string
  /** Which of the 6 status token families this value renders with. */
  tone: StatusTone
}

type GlossaryTable<D extends GlossaryDimension> = Record<GlossaryDimensionValueMap[D], GlossaryEntry>

function entry<D extends GlossaryDimension>(dimension: D, value: GlossaryDimensionValueMap[D], tone: StatusTone): GlossaryEntry {
  return {
    labelKey: `glossary.${dimension}.${value}.label`,
    descKey: `glossary.${dimension}.${value}.desc`,
    tone,
  }
}

export const lifecycleStageGlossary: GlossaryTable<'lifecycleStage'> = {
  intake: entry('lifecycleStage', 'intake', 'info'),
  allocation: entry('lifecycleStage', 'allocation', 'progress'),
  review: entry('lifecycleStage', 'review', 'warning'),
  execution: entry('lifecycleStage', 'execution', 'progress'),
  syncing_back: entry('lifecycleStage', 'syncing_back', 'progress'),
  awaiting_manual_closure: entry('lifecycleStage', 'awaiting_manual_closure', 'warning'),
  closed: entry('lifecycleStage', 'closed', 'neutral'),
}

export const routingDispositionGlossary: GlossaryTable<'routingDisposition'> = {
  pending_intake: entry('routingDisposition', 'pending_intake', 'info'),
  accepted: entry('routingDisposition', 'accepted', 'success'),
  deferred: entry('routingDisposition', 'deferred', 'warning'),
  excluded_manual: entry('routingDisposition', 'excluded_manual', 'neutral'),
  excluded_duplicate: entry('routingDisposition', 'excluded_duplicate', 'neutral'),
  excluded_revoked: entry('routingDisposition', 'excluded_revoked', 'error'),
}

export const recipientInputStateGlossary: GlossaryTable<'recipientInputState'> = {
  not_required: entry('recipientInputState', 'not_required', 'neutral'),
  waiting_for_input: entry('recipientInputState', 'waiting_for_input', 'warning'),
  partially_collected: entry('recipientInputState', 'partially_collected', 'info'),
  ready: entry('recipientInputState', 'ready', 'success'),
  waived: entry('recipientInputState', 'waived', 'neutral'),
  expired: entry('recipientInputState', 'expired', 'error'),
}

export const addressStateGlossary: GlossaryTable<'addressState'> = {
  missing: entry('addressState', 'missing', 'warning'),
  ready: entry('addressState', 'ready', 'success'),
  invalid: entry('addressState', 'invalid', 'error'),
}

export const supplierStateGlossary: GlossaryTable<'supplierState'> = {
  not_submitted: entry('supplierState', 'not_submitted', 'neutral'),
  submitted: entry('supplierState', 'submitted', 'info'),
  accepted: entry('supplierState', 'accepted', 'success'),
  producing: entry('supplierState', 'producing', 'progress'),
  partially_shipped: entry('supplierState', 'partially_shipped', 'progress'),
  shipped: entry('supplierState', 'shipped', 'success'),
  canceled: entry('supplierState', 'canceled', 'neutral'),
}

export const channelSyncStateGlossary: GlossaryTable<'channelSyncState'> = {
  not_required: entry('channelSyncState', 'not_required', 'neutral'),
  unsupported: entry('channelSyncState', 'unsupported', 'neutral'),
  pending: entry('channelSyncState', 'pending', 'info'),
  synced: entry('channelSyncState', 'synced', 'success'),
  manual_confirmed: entry('channelSyncState', 'manual_confirmed', 'success'),
  skipped: entry('channelSyncState', 'skipped', 'neutral'),
  failed: entry('channelSyncState', 'failed', 'error'),
}

export const allocationStateGlossary: GlossaryTable<'allocationState'> = {
  draft: entry('allocationState', 'draft', 'neutral'),
  ready: entry('allocationState', 'ready', 'success'),
}

export const shipmentStatusGlossary: GlossaryTable<'shipmentStatus'> = {
  pending: entry('shipmentStatus', 'pending', 'neutral'),
  shipped: entry('shipmentStatus', 'shipped', 'progress'),
  in_transit: entry('shipmentStatus', 'in_transit', 'progress'),
  delivered: entry('shipmentStatus', 'delivered', 'success'),
  exception: entry('shipmentStatus', 'exception', 'error'),
  returned: entry('shipmentStatus', 'returned', 'warning'),
  voided: entry('shipmentStatus', 'voided', 'neutral'),
}

export const submissionModeGlossary: GlossaryTable<'submissionMode'> = {
  csv: entry('submissionMode', 'csv', 'neutral'),
  manual: entry('submissionMode', 'manual', 'neutral'),
  api: entry('submissionMode', 'api', 'neutral'),
}

export const demandKindGlossary: GlossaryTable<'demandKind'> = {
  membership_entitlement: entry('demandKind', 'membership_entitlement', 'neutral'),
  retail_order: entry('demandKind', 'retail_order', 'neutral'),
}

export const adjustmentKindGlossary: GlossaryTable<'adjustmentKind'> = {
  add: entry('adjustmentKind', 'add', 'neutral'),
  reduce: entry('adjustmentKind', 'reduce', 'neutral'),
  compensation: entry('adjustmentKind', 'compensation', 'neutral'),
  remove: entry('adjustmentKind', 'remove', 'neutral'),
  replace: entry('adjustmentKind', 'replace', 'neutral'),
  reissue: entry('adjustmentKind', 'reissue', 'neutral'),
}

export const productKindGlossary: GlossaryTable<'productKind'> = {
  badge: entry('productKind', 'badge', 'neutral'),
  standee: entry('productKind', 'standee', 'neutral'),
  charm: entry('productKind', 'charm', 'neutral'),
  postcard: entry('productKind', 'postcard', 'neutral'),
  print: entry('productKind', 'print', 'neutral'),
  bundle: entry('productKind', 'bundle', 'neutral'),
  other: entry('productKind', 'other', 'neutral'),
}

export const profileTypeGlossary: GlossaryTable<'profileType'> = {
  member: entry('profileType', 'member', 'info'),
  buyer: entry('profileType', 'buyer', 'neutral'),
  mixed: entry('profileType', 'mixed', 'progress'),
  manual: entry('profileType', 'manual', 'neutral'),
}

export const identityTypeGlossary: GlossaryTable<'identityType'> = {
  platform_uid: entry('identityType', 'platform_uid', 'neutral'),
  email: entry('identityType', 'email', 'neutral'),
  username: entry('identityType', 'username', 'neutral'),
  external_buyer_id: entry('identityType', 'external_buyer_id', 'neutral'),
}

export const driftSummaryGlossary: GlossaryTable<'driftSummary'> = {
  in_sync: entry('driftSummary', 'in_sync', 'success'),
  drifted_none: entry('driftSummary', 'drifted_none', 'neutral'),
  drifted_recommended: entry('driftSummary', 'drifted_recommended', 'warning'),
  drifted_required: entry('driftSummary', 'drifted_required', 'error'),
}

export const reviewRequirementGlossary: GlossaryTable<'reviewRequirement'> = {
  none: entry('reviewRequirement', 'none', 'neutral'),
  recommended: entry('reviewRequirement', 'recommended', 'warning'),
  required: entry('reviewRequirement', 'required', 'error'),
}

export const lineReasonGlossary: GlossaryTable<'lineReason'> = {
  entitlement: entry('lineReason', 'entitlement', 'neutral'),
  retail_order: entry('lineReason', 'retail_order', 'neutral'),
  wave_adjustment: entry('lineReason', 'wave_adjustment', 'info'),
}

export const basisDriftStatusGlossary: GlossaryTable<'basisDriftStatus'> = {
  drifted: entry('basisDriftStatus', 'drifted', 'warning'),
  in_sync: entry('basisDriftStatus', 'in_sync', 'success'),
}

export const allocationSelectorTypeGlossary: GlossaryTable<'allocationSelectorType'> = {
  wave_all: entry('allocationSelectorType', 'wave_all', 'neutral'),
  platform_all: entry('allocationSelectorType', 'platform_all', 'info'),
  identity_level: entry('allocationSelectorType', 'identity_level', 'info'),
  explicit_override: entry('allocationSelectorType', 'explicit_override', 'warning'),
}

export const demandMappingBlockedReasonGlossary: GlossaryTable<'demandMappingBlockedReason'> = {
  wave_product_missing: entry('demandMappingBlockedReason', 'wave_product_missing', 'warning'),
  address_unavailable: entry('demandMappingBlockedReason', 'address_unavailable', 'warning'),
}

export const initialAllocationStrategyGlossary: GlossaryTable<'initialAllocationStrategy'> = {
  policy_driven: entry('initialAllocationStrategy', 'policy_driven', 'neutral'),
  demand_driven: entry('initialAllocationStrategy', 'demand_driven', 'neutral'),
}

export const identityStrategyGlossary: GlossaryTable<'identityStrategy'> = {
  platform_uid: entry('identityStrategy', 'platform_uid', 'neutral'),
  email: entry('identityStrategy', 'email', 'neutral'),
  external_buyer_id: entry('identityStrategy', 'external_buyer_id', 'neutral'),
}

export const entitlementAuthorityModeGlossary: GlossaryTable<'entitlementAuthorityMode'> = {
  local_policy: entry('entitlementAuthorityMode', 'local_policy', 'neutral'),
  upstream_platform: entry('entitlementAuthorityMode', 'upstream_platform', 'info'),
  manual_grant_only: entry('entitlementAuthorityMode', 'manual_grant_only', 'warning'),
}

export const recipientInputModeGlossary: GlossaryTable<'recipientInputMode'> = {
  none: entry('recipientInputMode', 'none', 'neutral'),
  platform_claim: entry('recipientInputMode', 'platform_claim', 'neutral'),
  external_form: entry('recipientInputMode', 'external_form', 'info'),
  manual_collection: entry('recipientInputMode', 'manual_collection', 'warning'),
}

export const referenceStrategyGlossary: GlossaryTable<'referenceStrategy'> = {
  member_level: entry('referenceStrategy', 'member_level', 'neutral'),
  order_level: entry('referenceStrategy', 'order_level', 'neutral'),
  order_line_level: entry('referenceStrategy', 'order_line_level', 'neutral'),
}

export const trackingSyncModeGlossary: GlossaryTable<'trackingSyncMode'> = {
  api_push: entry('trackingSyncMode', 'api_push', 'success'),
  document_export: entry('trackingSyncMode', 'document_export', 'info'),
  manual_confirmation: entry('trackingSyncMode', 'manual_confirmation', 'warning'),
  unsupported: entry('trackingSyncMode', 'unsupported', 'neutral'),
}

export const closurePolicyGlossary: GlossaryTable<'closurePolicy'> = {
  close_after_sync: entry('closurePolicy', 'close_after_sync', 'success'),
  close_after_manual_confirmation: entry('closurePolicy', 'close_after_manual_confirmation', 'warning'),
  close_after_shipment: entry('closurePolicy', 'close_after_shipment', 'neutral'),
}

export const documentTypeGlossary: GlossaryTable<'documentType'> = {
  import_entitlement: entry('documentType', 'import_entitlement', 'neutral'),
  import_sales_order: entry('documentType', 'import_sales_order', 'neutral'),
  import_product_catalog: entry('documentType', 'import_product_catalog', 'neutral'),
  export_supplier_order: entry('documentType', 'export_supplier_order', 'info'),
  import_supplier_shipment: entry('documentType', 'import_supplier_shipment', 'info'),
  export_source_tracking_update: entry('documentType', 'export_source_tracking_update', 'info'),
}

export const supplierOrderStatusGlossary: GlossaryTable<'supplierOrderStatus'> = {
  draft: entry('supplierOrderStatus', 'draft', 'neutral'),
  submitted: entry('supplierOrderStatus', 'submitted', 'info'),
  accepted: entry('supplierOrderStatus', 'accepted', 'success'),
  partially_shipped: entry('supplierOrderStatus', 'partially_shipped', 'progress'),
  shipped: entry('supplierOrderStatus', 'shipped', 'success'),
  canceled: entry('supplierOrderStatus', 'canceled', 'neutral'),
}

export const channelSyncJobStatusGlossary: GlossaryTable<'channelSyncJobStatus'> = {
  pending: entry('channelSyncJobStatus', 'pending', 'neutral'),
  running: entry('channelSyncJobStatus', 'running', 'progress'),
  success: entry('channelSyncJobStatus', 'success', 'success'),
  failed: entry('channelSyncJobStatus', 'failed', 'error'),
  partial_success: entry('channelSyncJobStatus', 'partial_success', 'warning'),
}

export const channelSyncItemStatusGlossary: GlossaryTable<'channelSyncItemStatus'> = {
  success: entry('channelSyncItemStatus', 'success', 'success'),
  failed: entry('channelSyncItemStatus', 'failed', 'error'),
}

export const closureDecisionKindGlossary: GlossaryTable<'closureDecisionKind'> = {
  mark_sync_unsupported: entry('closureDecisionKind', 'mark_sync_unsupported', 'neutral'),
  mark_sync_skipped: entry('closureDecisionKind', 'mark_sync_skipped', 'neutral'),
  mark_sync_completed_manually: entry('closureDecisionKind', 'mark_sync_completed_manually', 'warning'),
}

export const captureModeGlossary: GlossaryTable<'captureMode'> = {
  document_import: entry('captureMode', 'document_import', 'neutral'),
  api_ingest: entry('captureMode', 'api_ingest', 'neutral'),
  manual_entry: entry('captureMode', 'manual_entry', 'neutral'),
}

export const glossaryTables: { [D in GlossaryDimension]: GlossaryTable<D> } = {
  lifecycleStage: lifecycleStageGlossary,
  routingDisposition: routingDispositionGlossary,
  recipientInputState: recipientInputStateGlossary,
  addressState: addressStateGlossary,
  supplierState: supplierStateGlossary,
  channelSyncState: channelSyncStateGlossary,
  allocationState: allocationStateGlossary,
  shipmentStatus: shipmentStatusGlossary,
  submissionMode: submissionModeGlossary,
  demandKind: demandKindGlossary,
  adjustmentKind: adjustmentKindGlossary,
  productKind: productKindGlossary,
  profileType: profileTypeGlossary,
  identityType: identityTypeGlossary,
  driftSummary: driftSummaryGlossary,
  reviewRequirement: reviewRequirementGlossary,
  lineReason: lineReasonGlossary,
  basisDriftStatus: basisDriftStatusGlossary,
  allocationSelectorType: allocationSelectorTypeGlossary,
  demandMappingBlockedReason: demandMappingBlockedReasonGlossary,
  initialAllocationStrategy: initialAllocationStrategyGlossary,
  identityStrategy: identityStrategyGlossary,
  entitlementAuthorityMode: entitlementAuthorityModeGlossary,
  recipientInputMode: recipientInputModeGlossary,
  referenceStrategy: referenceStrategyGlossary,
  trackingSyncMode: trackingSyncModeGlossary,
  closurePolicy: closurePolicyGlossary,
  documentType: documentTypeGlossary,
  supplierOrderStatus: supplierOrderStatusGlossary,
  channelSyncJobStatus: channelSyncJobStatusGlossary,
  channelSyncItemStatus: channelSyncItemStatusGlossary,
  closureDecisionKind: closureDecisionKindGlossary,
  captureMode: captureModeGlossary,
}

function lookup<D extends GlossaryDimension>(dimension: D, value: string): GlossaryEntry | undefined {
  const table = glossaryTables[dimension] as Record<string, GlossaryEntry>
  return table[value]
}

/**
 * The glossary composable. Resolves through the global vue-i18n composer
 * directly (not the component-scoped `useI18n()` hook), so it is safe to
 * call from anywhere — components, composables, or plain utility code —
 * without requiring an active component instance.
 *
 * Unknown `(dimension, value)` pairs never throw: `label` and `desc` fall
 * back to the raw value string, and `tone` falls back to `'neutral'`.
 */
export function useGlossary() {
  function label<D extends GlossaryDimension>(dimension: D, value: GlossaryDimensionValueMap[D] | (string & {})): string {
    const found = lookup(dimension, value)
    return found ? i18n.global.t(found.labelKey) : value
  }

  function desc<D extends GlossaryDimension>(dimension: D, value: GlossaryDimensionValueMap[D] | (string & {})): string {
    const found = lookup(dimension, value)
    return found ? i18n.global.t(found.descKey) : value
  }

  function tone<D extends GlossaryDimension>(dimension: D, value: GlossaryDimensionValueMap[D] | (string & {})): StatusTone {
    const found = lookup(dimension, value)
    return found ? found.tone : 'neutral'
  }

  return { label, desc, tone }
}
