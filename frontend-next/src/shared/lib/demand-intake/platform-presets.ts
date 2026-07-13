/**
 * Platform presets for the demand-intake wizard (P4). Frontend-only constants —
 * there is no backend "preset" concept; these seed the wizard's first two steps
 * (platform pick + column-mapping starting point) with plausible defaults that
 * the operator confirms/edits before the profile + template are actually created
 * via `createProfile` / `createDocumentTemplate` (bridge.ts).
 *
 * `defaultColumns` keys are the 12 canonical destFields consumed by the backend's
 * `TemplateMappingRules.Columns` (see `internal/app/template_mapping_service.go`
 * `setDemandLineField` — snake_case, NOT the camelCase entity/DTO field names).
 * Values are best-guess CSV header names for that platform's typical export —
 * starter guesses only, always user-editable in the wizard's mapping step.
 */

/** One of the 12 destFields `TemplateMappingRules.Columns`/`Defaults` accepts. */
export type IntakeDestField =
  | 'line_type'
  | 'obligation_trigger_kind'
  | 'entitlement_authority'
  | 'recipient_input_state'
  | 'routing_disposition'
  | 'routing_reason_code'
  | 'eligibility_context_ref'
  | 'entitlement_code'
  | 'gift_level_snapshot'
  | 'recipient_input_payload'
  | 'external_title'
  | 'requested_quantity'

/** The 6 boolean capability flags on `IntegrationProfileDTO` (see `createProfile` in bridge.ts). */
export interface IntakeProfileCapabilities {
  supportsPartialShipment: boolean
  supportsApiImport: boolean
  supportsApiExport: boolean
  requiresCarrierMapping: boolean
  requiresExternalOrderNo: boolean
  allowsManualClosure: boolean
}

export interface PlatformPreset {
  /** Stable identifier, also the `intakeWizard.presets.<key>.*` i18n leaf. */
  key: string
  /** i18n key for the preset's short label. */
  labelKey: string
  /** i18n key for the preset's one-sentence description. */
  descKey: string
  /** Seeds `CreateProfileInput.sourceChannel`. */
  sourceChannel: string
  /** Seeds `CreateProfileInput.sourceSurface`. */
  sourceSurface: string
  /** Seeds `CreateProfileInput.demandKind` — MUST be one of the backend's two whitelisted values. */
  demandKind: 'membership_entitlement' | 'retail_order'
  /** destField -> likely CSV header name, seeding the wizard's column-mapping step. */
  defaultColumns: Partial<Record<IntakeDestField, string>>
  /** Seeds the 6 boolean capability flags on the new integration profile. */
  defaultCapabilities: Partial<IntakeProfileCapabilities>
}

export const PLATFORM_PRESETS: readonly PlatformPreset[] = [
  {
    key: 'patreon',
    labelKey: 'intakeWizard.presets.patreon.label',
    descKey: 'intakeWizard.presets.patreon.description',
    sourceChannel: 'patreon',
    sourceSurface: 'membership',
    demandKind: 'membership_entitlement',
    defaultColumns: {
      external_title: 'Reward',
      requested_quantity: 'Quantity',
      entitlement_code: 'Tier',
      recipient_input_payload: 'Address',
    },
    defaultCapabilities: {
      supportsPartialShipment: true,
      supportsApiImport: false,
      supportsApiExport: false,
      requiresCarrierMapping: false,
      requiresExternalOrderNo: false,
      allowsManualClosure: true,
    },
  },
  {
    key: 'bilibili',
    labelKey: 'intakeWizard.presets.bilibili.label',
    descKey: 'intakeWizard.presets.bilibili.description',
    sourceChannel: 'bilibili',
    sourceSurface: 'membership',
    demandKind: 'membership_entitlement',
    defaultColumns: {
      external_title: '礼物名称',
      requested_quantity: '数量',
      entitlement_code: '大航海等级',
      recipient_input_payload: '收货地址',
    },
    defaultCapabilities: {
      supportsPartialShipment: true,
      supportsApiImport: false,
      supportsApiExport: false,
      requiresCarrierMapping: true,
      requiresExternalOrderNo: false,
      allowsManualClosure: true,
    },
  },
  {
    key: 'gumroad',
    labelKey: 'intakeWizard.presets.gumroad.label',
    descKey: 'intakeWizard.presets.gumroad.description',
    sourceChannel: 'gumroad',
    sourceSurface: 'retail',
    demandKind: 'retail_order',
    defaultColumns: {
      external_title: 'Product Name',
      requested_quantity: 'Quantity',
      recipient_input_payload: 'Shipping Address',
    },
    defaultCapabilities: {
      supportsPartialShipment: false,
      supportsApiImport: true,
      supportsApiExport: true,
      requiresCarrierMapping: true,
      requiresExternalOrderNo: true,
      allowsManualClosure: false,
    },
  },
  {
    key: 'youtube',
    labelKey: 'intakeWizard.presets.youtube.label',
    descKey: 'intakeWizard.presets.youtube.description',
    sourceChannel: 'youtube',
    sourceSurface: 'membership',
    demandKind: 'membership_entitlement',
    defaultColumns: {
      external_title: 'Reward Name',
      requested_quantity: 'Quantity',
      entitlement_code: 'Membership Level',
      recipient_input_payload: 'Shipping Address',
    },
    defaultCapabilities: {
      supportsPartialShipment: true,
      supportsApiImport: false,
      supportsApiExport: false,
      requiresCarrierMapping: false,
      requiresExternalOrderNo: false,
      allowsManualClosure: true,
    },
  },
  {
    key: 'custom',
    labelKey: 'intakeWizard.presets.custom.label',
    descKey: 'intakeWizard.presets.custom.description',
    sourceChannel: '',
    sourceSurface: '',
    demandKind: 'membership_entitlement',
    defaultColumns: {},
    defaultCapabilities: {},
  },
] as const
