/**
 * Pure derivation helpers for fields `CreateProfileInput` requires that the
 * intake wizard's named steps have no dedicated UI for: the 7 profile
 * "strategy" enums and `documentType` (which the backend derives 1:1 from
 * `demandKind` for demand faces; factory uses catalog/shipment/export types).
 *
 * Rather than adding a 6th wizard step, these are deterministically derived
 * from answers the operator already gave (business surface + capability
 * toggles) — every value produced here is still shown to the operator on
 * the confirm step via `StatusBadge`, never silently applied.
 */
import type { IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'

export type DemandKind = 'membership_entitlement' | 'retail_order' | ''

/** Business-surface choice shown on the wizard's second step. */
export type BusinessSurfaceChoice = 'membership' | 'retail' | 'factory'

/** Factory capability flags on IntegrationProfile (Create/Update DTO). */
export interface FactoryProfileCapabilities {
  supportsExportSupplierOrder: boolean
  supportsImportProductCatalog: boolean
  supportsImportSupplierShipment: boolean
}

/** `documentType` for demand faces — must match `validDocumentTypes` backend. */
export function documentTypeForDemandKind(demandKind: DemandKind): string {
  if (demandKind === 'retail_order') return 'import_sales_order'
  if (demandKind === 'membership_entitlement') return 'import_entitlement'
  return 'import_entitlement'
}

/**
 * Primary document type for a factory surface profile, by capability priority:
 * product catalog → supplier shipment → export supplier order.
 */
export function documentTypeForFactoryCaps(caps: FactoryProfileCapabilities): string {
  if (caps.supportsImportProductCatalog) return 'import_product_catalog'
  if (caps.supportsImportSupplierShipment) return 'import_supplier_shipment'
  if (caps.supportsExportSupplierOrder) return 'export_supplier_order'
  return 'import_product_catalog'
}

/** All document types implied by enabled factory caps (for multi-template seed). */
export function documentTypesForFactoryCaps(caps: FactoryProfileCapabilities): string[] {
  const types: string[] = []
  if (caps.supportsImportProductCatalog) types.push('import_product_catalog')
  if (caps.supportsImportSupplierShipment) types.push('import_supplier_shipment')
  if (caps.supportsExportSupplierOrder) types.push('export_supplier_order')
  return types
}

export interface DerivedProfileStrategy {
  initialAllocationStrategy: string
  identityStrategy: string
  entitlementAuthorityMode: string
  recipientInputMode: string
  referenceStrategy: string
  trackingSyncMode: string
  closurePolicy: string
}

export interface DeriveProfileStrategyOptions {
  /** Whether the operator picked a `connectorKey` in the capabilities step. */
  hasConnectorKey: boolean
  /**
   * Explicit `trackingSyncMode` choice, ONLY meaningful when `hasConnectorKey` is true.
   * See prior doc: never silently guess `api_push` / `document_export`.
   */
  trackingSyncModeOverride?: string
  /** When true, strategy fields collapse to factory-safe defaults. */
  isFactorySurface?: boolean
}

/**
 * Derives the 7 profile-strategy enum fields from `demandKind` + capability
 * toggles. Factory surfaces skip demand-side strategy enums (backend exempts
 * them) and only keep a safe trackingSyncMode / closurePolicy pair.
 */
export function deriveProfileStrategyDefaults(
  demandKind: DemandKind,
  capabilities: IntakeProfileCapabilities,
  options: DeriveProfileStrategyOptions = { hasConnectorKey: false },
): DerivedProfileStrategy {
  if (options.isFactorySurface) {
    const trackingSyncMode = options.hasConnectorKey
      ? (options.trackingSyncModeOverride ?? 'document_export')
      : 'unsupported'
    return {
      initialAllocationStrategy: '',
      identityStrategy: '',
      entitlementAuthorityMode: '',
      recipientInputMode: '',
      referenceStrategy: '',
      trackingSyncMode,
      closurePolicy: trackingSyncMode === 'unsupported' ? '' : 'close_after_sync',
    }
  }

  const isMembership = demandKind === 'membership_entitlement'

  const trackingSyncMode = options.hasConnectorKey
    ? (options.trackingSyncModeOverride ?? 'document_export')
    : capabilities.allowsManualClosure
      ? 'manual_confirmation'
      : 'unsupported'

  const closurePolicy =
    trackingSyncMode === 'api_push' || trackingSyncMode === 'document_export'
      ? 'close_after_sync'
      : capabilities.allowsManualClosure
        ? 'close_after_manual_confirmation'
        : 'close_after_shipment'

  return {
    initialAllocationStrategy: 'policy_driven',
    identityStrategy: 'platform_uid',
    entitlementAuthorityMode: 'local_policy',
    recipientInputMode: isMembership ? 'platform_claim' : 'none',
    referenceStrategy: isMembership ? 'member_level' : 'order_level',
    trackingSyncMode,
    closurePolicy,
  }
}
