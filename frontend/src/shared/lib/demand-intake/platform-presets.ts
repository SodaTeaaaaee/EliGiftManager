/**
 * Platform presets for the demand-intake wizard (P4). Frontend-only constants —
 * there is no backend "preset" concept; these seed the wizard's first two steps
 * (platform pick + column-mapping starting point) with plausible defaults that
 * the operator confirms/edits before the profile + template are actually created
 * via `createProfile` / `createDocumentTemplate` (bridge.ts).
 *
 * Mapping seeds may be either:
 * - `defaultColumns` (v1 header-mode dest → CSV header), or
 * - `defaultMapping` (full v2 FieldMappingValue shape: mode/positions/hasHeader).
 *
 * Dest keys use v2 namespaced form (`line.*` / `document.*` / `recipient.*`).
 * Bare line keys are still accepted and normalised by `mappingFromPreset` /
 * `serializeMappingRules`.
 *
 * Bilibili is one source-platform card. Entitlement (positional three-column,
 * no header) and retail (header-mapped workshop export) are file-kind mapping
 * seeds, not a second preset.
 */

import {
  ensureNamespacedDestKey,
  type FieldMappingValue,
} from '@/shared/ui/field-mapping'

/** One of the 12 destFields `TemplateMappingRules.Columns`/`Defaults` accepts (unprefixed). */
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
  /**
   * Leftover hint for single-kind presets. Empty when the platform can bind
   * both entitlement and retail files — the operator checks those later.
   */
  demandKind?: 'membership_entitlement' | 'retail_order' | ''
  /**
   * Which source file kinds to pre-check. When omitted, derived from
   * `demandKind` (retail → sales order only; else entitlement only).
   */
  defaultFileKinds?: { entitlement: boolean; salesOrder: boolean }
  /**
   * Seeds `CreateProfileInput.factorySupplierPlatform` — factory-facing platform
   * label written onto supplier orders / product catalog fallback.
   */
  factorySupplierPlatform?: string
  /**
   * destField -> likely CSV header name, seeding the wizard's column-mapping
   * step (header mode). Optional when `defaultMapping` is present.
   */
  defaultColumns?: Partial<Record<IntakeDestField | string, string>>
  /**
   * Optional full v2 mapping seed. When present, takes precedence over
   * `defaultColumns` for the wizard's initial FieldMappingValue.
   */
  defaultMapping?: Partial<FieldMappingValue>
  /**
   * Per-file-kind mapping seeds. Used when one source platform has both
   * entitlement and retail samples (e.g. Bilibili).
   */
  mappingsByDocumentType?: Partial<
    Record<'import_entitlement' | 'import_sales_order', Partial<FieldMappingValue>>
  >
  /** Seeds the 6 boolean capability flags on the new integration profile. */
  defaultCapabilities: Partial<IntakeProfileCapabilities>
}

function mapRecordKeys<T>(record: Record<string, T> | undefined): Record<string, T> {
  const out: Record<string, T> = {}
  for (const [k, v] of Object.entries(record ?? {})) {
    out[ensureNamespacedDestKey(k)] = v
  }
  return out
}

/** File-kind toggles implied by a preset (explicit flags, else leftover demandKind). */
export function fileFlagsFromPreset(preset: PlatformPreset): { entitlement: boolean; salesOrder: boolean } {
  if (preset.defaultFileKinds) return { ...preset.defaultFileKinds }
  if (preset.demandKind === 'retail_order') return { entitlement: false, salesOrder: true }
  return { entitlement: true, salesOrder: false }
}

/** Mapping seed for one document type; leftover demandKind only when it explicitly matches. */
export function mappingFromPresetForDocumentType(
  preset: PlatformPreset,
  docType: string,
): FieldMappingValue | null {
  const byType = preset.mappingsByDocumentType?.[docType as 'import_entitlement' | 'import_sales_order']
  if (byType) {
    return mappingFromPreset({ ...preset, defaultMapping: byType, defaultColumns: undefined })
  }
  if (preset.demandKind === 'retail_order' && docType === 'import_sales_order') {
    return mappingFromPreset(preset)
  }
  if (preset.demandKind === 'membership_entitlement' && docType === 'import_entitlement') {
    return mappingFromPreset(preset)
  }
  return null
}

/** Build a v2 FieldMappingValue from a preset (columns or full defaultMapping). */
export function mappingFromPreset(preset: PlatformPreset): FieldMappingValue {
  if (preset.defaultMapping) {
    return {
      version: 2,
      mode: preset.defaultMapping.mode ?? 'header',
      hasHeader: preset.defaultMapping.hasHeader ?? true,
      columns: mapRecordKeys(preset.defaultMapping.columns ?? {}),
      positions: mapRecordKeys(preset.defaultMapping.positions ?? {}),
      defaults: mapRecordKeys(preset.defaultMapping.defaults ?? {}),
      transforms: preset.defaultMapping.transforms
        ? mapRecordKeys(preset.defaultMapping.transforms)
        : undefined,
      columnOrder: preset.defaultMapping.columnOrder
        ? preset.defaultMapping.columnOrder.map(ensureNamespacedDestKey)
        : [],
      required: preset.defaultMapping.required
        ? preset.defaultMapping.required.map(ensureNamespacedDestKey)
        : undefined,
    }
  }
  return {
    version: 2,
    mode: 'header',
    hasHeader: true,
    columns: mapRecordKeys((preset.defaultColumns ?? {}) as Record<string, string>),
    positions: {},
    defaults: {},
    columnOrder: [],
  }
}

export const PLATFORM_PRESETS: readonly PlatformPreset[] = [
  {
    key: 'patreon',
    labelKey: 'intakeWizard.presets.patreon.label',
    descKey: 'intakeWizard.presets.patreon.description',
    sourceChannel: 'patreon',
    sourceSurface: 'membership',
    demandKind: 'membership_entitlement',
    factorySupplierPlatform: 'patreon',
    defaultColumns: {
      'line.external_title': 'Reward',
      'line.requested_quantity': 'Quantity',
      'line.entitlement_code': 'Tier',
      'line.recipient_input_payload': 'Address',
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
    demandKind: '',
    defaultFileKinds: { entitlement: true, salesOrder: true },
    factorySupplierPlatform: 'bilibili',
    mappingsByDocumentType: {
      // Config-level positional three-column example (SampleData 会员列表, no header):
      // col0 = gift level, col1 = platform UID, col2 = display name.
      import_entitlement: {
        version: 2,
        mode: 'positional',
        hasHeader: false,
        columns: {},
        positions: {
          'line.gift_level_snapshot': 0,
          'document.source_customer_ref': 1,
          'document.display_name': 2,
        },
        defaults: {
          'line.line_type': 'entitlement_rule',
          'line.requested_quantity': '1',
          'line.obligation_trigger_kind': 'periodic_membership',
          'line.entitlement_authority': 'upstream_platform',
          'line.recipient_input_state': 'not_required',
          'line.routing_disposition': 'accepted',
        },
        columnOrder: [
          'line.gift_level_snapshot',
          'document.source_customer_ref',
          'document.display_name',
        ],
      },
      // SampleData 单个订单数据 — Chinese header columns locked to export headers.
      import_sales_order: {
        version: 2,
        mode: 'header',
        hasHeader: true,
        columns: {
          'line.external_title': '商品名称',
          'line.requested_quantity': '数量',
          'recipient.name': '收货人姓名',
          'recipient.phone': '联系电话',
          'recipient.address_line1': '收货地址',
          'document.source_document_no': '订单号',
          'document.source_customer_ref': '买家昵称',
          'document.display_name': '买家昵称',
        },
        positions: {},
        defaults: {
          'line.line_type': 'sku_order',
          'line.entitlement_authority': 'upstream_platform',
          'line.recipient_input_state': 'ready',
          'line.routing_disposition': 'accepted',
        },
        columnOrder: [
          'line.external_title',
          'line.requested_quantity',
          'recipient.name',
          'recipient.phone',
          'recipient.address_line1',
          'document.source_document_no',
          'document.source_customer_ref',
          'document.display_name',
        ],
      },
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
    factorySupplierPlatform: 'gumroad',
    defaultColumns: {
      'line.external_title': 'Product Name',
      'line.requested_quantity': 'Quantity',
      'line.recipient_input_payload': 'Shipping Address',
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
    factorySupplierPlatform: 'youtube',
    defaultColumns: {
      'line.external_title': 'Reward Name',
      'line.requested_quantity': 'Quantity',
      'line.entitlement_code': 'Membership Level',
      'line.recipient_input_payload': 'Shipping Address',
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
    factorySupplierPlatform: '',
    defaultColumns: {},
    defaultCapabilities: {},
  },
] as const
