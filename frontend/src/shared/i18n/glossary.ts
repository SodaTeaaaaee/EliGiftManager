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
  IdentityType,
  PlatformKind,
  InputFactKind,
  TemplateDirection,
  WaveCloseResult,
  EntitlementSelectorType,
  FulfillmentSourceKind,
  BlockReason,
  SupplierOrderStatus,
  DuplicateVerdict,
  WritebackStatus,
  WorkState,
} from '@/shared/api/generated/enums'
import { i18n } from './index'

/** The 6 status token families the theme system defines (`--status-*-fg/bg/border`). */
export type StatusTone = 'success' | 'warning' | 'error' | 'info' | 'progress' | 'neutral'

export type IdentityTypeValue = IdentityType
export type PlatformKindValue = PlatformKind
export type InputFactKindValue = InputFactKind
export type TemplateDirectionValue = TemplateDirection
export type WaveCloseResultValue = WaveCloseResult
export type EntitlementSelectorTypeValue = EntitlementSelectorType
export type FulfillmentSourceKindValue = FulfillmentSourceKind
export type BlockReasonValue = BlockReason
export type SupplierOrderStatusValue = SupplierOrderStatus
export type DuplicateVerdictValue = DuplicateVerdict
export type WritebackStatusValue = WritebackStatus
export type WorkStateValue = WorkState

/** One map entry per glossary dimension -> the value union it accepts. */
export interface GlossaryDimensionValueMap {
  identityType: IdentityTypeValue
  platformKind: PlatformKindValue
  inputFactKind: InputFactKindValue
  templateDirection: TemplateDirectionValue
  waveCloseResult: WaveCloseResultValue
  entitlementSelectorType: EntitlementSelectorTypeValue
  fulfillmentSourceKind: FulfillmentSourceKindValue
  blockReason: BlockReasonValue
  supplierOrderStatus: SupplierOrderStatusValue
  duplicateVerdict: DuplicateVerdictValue
  writebackStatus: WritebackStatusValue
  workState: WorkStateValue
}

export type GlossaryDimension = keyof GlossaryDimensionValueMap

export interface GlossaryEntry {
  readonly label: string
  readonly description: string
  readonly tone: StatusTone
  readonly dimension: GlossaryDimension
  readonly value: string
}

type GlossaryTable<D extends GlossaryDimension> = Record<GlossaryDimensionValueMap[D], GlossaryEntry>

function entry<D extends GlossaryDimension>(dimension: D, value: GlossaryDimensionValueMap[D], tone: StatusTone): GlossaryEntry {
  return {
    get label(): string {
      return i18n.global.t(`glossary.${dimension}.${value}.label`)
    },
    get description(): string {
      return i18n.global.t(`glossary.${dimension}.${value}.description`)
    },
    tone,
    dimension,
    value,
  }
}

export const identityTypeGlossary: GlossaryTable<'identityType'> = {
  platform_uid: entry('identityType', 'platform_uid', 'neutral'),
  email: entry('identityType', 'email', 'neutral'),
  username: entry('identityType', 'username', 'neutral'),
  external_buyer_id: entry('identityType', 'external_buyer_id', 'neutral'),
}

export const platformKindGlossary: GlossaryTable<'platformKind'> = {
  source: entry('platformKind', 'source', 'info'),
  factory: entry('platformKind', 'factory', 'progress'),
}

export const inputFactKindGlossary: GlossaryTable<'inputFactKind'> = {
  membership: entry('inputFactKind', 'membership', 'info'),
  retail_order: entry('inputFactKind', 'retail_order', 'neutral'),
  operator_grant: entry('inputFactKind', 'operator_grant', 'warning'),
}

export const templateDirectionGlossary: GlossaryTable<'templateDirection'> = {
  input: entry('templateDirection', 'input', 'info'),
  output: entry('templateDirection', 'output', 'progress'),
}

export const waveCloseResultGlossary: GlossaryTable<'waveCloseResult'> = {
  open: entry('waveCloseResult', 'open', 'progress'),
  clean: entry('waveCloseResult', 'clean', 'success'),
  residual: entry('waveCloseResult', 'residual', 'warning'),
}

export const entitlementSelectorTypeGlossary: GlossaryTable<'entitlementSelectorType'> = {
  platform_level: entry('entitlementSelectorType', 'platform_level', 'info'),
  wave_all: entry('entitlementSelectorType', 'wave_all', 'neutral'),
  instance: entry('entitlementSelectorType', 'instance', 'warning'),
}

export const fulfillmentSourceKindGlossary: GlossaryTable<'fulfillmentSourceKind'> = {
  entitlement_instance: entry('fulfillmentSourceKind', 'entitlement_instance', 'info'),
  retail_line: entry('fulfillmentSourceKind', 'retail_line', 'neutral'),
  operator_grant: entry('fulfillmentSourceKind', 'operator_grant', 'warning'),
}

export const blockReasonGlossary: GlossaryTable<'blockReason'> = {
  unaligned_product: entry('blockReason', 'unaligned_product', 'error'),
  unusable_address: entry('blockReason', 'unusable_address', 'error'),
  identity_unattached: entry('blockReason', 'identity_unattached', 'error'),
  quantity_split_not_summing: entry('blockReason', 'quantity_split_not_summing', 'error'),
}

export const supplierOrderStatusGlossary: GlossaryTable<'supplierOrderStatus'> = {
  draft: entry('supplierOrderStatus', 'draft', 'neutral'),
  generated: entry('supplierOrderStatus', 'generated', 'info'),
  exported: entry('supplierOrderStatus', 'exported', 'progress'),
  voided: entry('supplierOrderStatus', 'voided', 'error'),
}

export const duplicateVerdictGlossary: GlossaryTable<'duplicateVerdict'> = {
  record_only: entry('duplicateVerdict', 'record_only', 'neutral'),
  ask_operator: entry('duplicateVerdict', 'ask_operator', 'warning'),
  new_responsibility: entry('duplicateVerdict', 'new_responsibility', 'info'),
}

export const writebackStatusGlossary: GlossaryTable<'writebackStatus'> = {
  pending: entry('writebackStatus', 'pending', 'neutral'),
  sent: entry('writebackStatus', 'sent', 'success'),
  failed: entry('writebackStatus', 'failed', 'error'),
}

export const workStateGlossary: GlossaryTable<'workState'> = {
  blocked: entry('workState', 'blocked', 'error'),
  ready: entry('workState', 'ready', 'neutral'),
  in_factory: entry('workState', 'in_factory', 'progress'),
  shipped: entry('workState', 'shipped', 'success'),
  writeback_failed: entry('workState', 'writeback_failed', 'warning'),
}

export const glossaryTables: { [D in GlossaryDimension]: GlossaryTable<D> } = {
  identityType: identityTypeGlossary,
  platformKind: platformKindGlossary,
  inputFactKind: inputFactKindGlossary,
  templateDirection: templateDirectionGlossary,
  waveCloseResult: waveCloseResultGlossary,
  entitlementSelectorType: entitlementSelectorTypeGlossary,
  fulfillmentSourceKind: fulfillmentSourceKindGlossary,
  blockReason: blockReasonGlossary,
  supplierOrderStatus: supplierOrderStatusGlossary,
  duplicateVerdict: duplicateVerdictGlossary,
  writebackStatus: writebackStatusGlossary,
  workState: workStateGlossary,
}

function lookup<D extends GlossaryDimension>(dimension: D, value: string): GlossaryEntry | undefined {
  const table = glossaryTables[dimension]
  return (table as Record<string, GlossaryEntry>)[value]
}

/**
 * The glossary composable. Resolves through the global vue-i18n composer
 * instance so labels and descriptions update automatically across locale
 * switches without requiring component re-renders.
 */
export function useGlossary() {
  return {
    entry: lookup,
    label<D extends GlossaryDimension>(dimension: D, value: string): string {
      return lookup(dimension, value)?.label ?? value
    },
    desc<D extends GlossaryDimension>(dimension: D, value: string): string {
      return lookup(dimension, value)?.description ?? ''
    },
    description<D extends GlossaryDimension>(dimension: D, value: string): string {
      return lookup(dimension, value)?.description ?? ''
    },
    tone<D extends GlossaryDimension>(dimension: D, value: string): StatusTone {
      return lookup(dimension, value)?.tone ?? 'neutral'
    },
    dimensionTable<D extends GlossaryDimension>(dimension: D): GlossaryTable<D> {
      return glossaryTables[dimension]
    },
  }
}
