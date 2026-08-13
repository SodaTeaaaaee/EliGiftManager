/**
 * Pure derivation helpers for fields `CreateProfileInput` requires that the
 * intake wizard's named steps have no dedicated UI for: the 7 profile
 * "strategy" enums and `documentType` (demand faces require an explicit
 * documentType; leftover DemandKind is not inferred. Factory uses
 * catalog/shipment/export types).
 *
 * Rather than adding a 6th wizard step, these are deterministically derived
 * from answers the operator already gave (business surface + capability
 * toggles) — every value produced here is still shown to the operator on
 * the confirm step via `StatusBadge`, never silently applied.
 */
import type { IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'

export type DemandKind = 'membership_entitlement' | 'retail_order' | ''

/** Platform kind shown on the wizard's second step — source vs factory, not demand kind. */
export type BusinessSurfaceChoice = 'source' | 'factory'

/** Factory capability flags on IntegrationProfile (Create/Update DTO). */
export interface FactoryProfileCapabilities {
  supportsExportSupplierOrder: boolean
  supportsImportProductCatalog: boolean
  supportsImportSupplierShipment: boolean
}

/** `documentType` for demand faces — empty leftover DemandKind is not inferred. */
export function documentTypeForDemandKind(demandKind: DemandKind): string {
  if (demandKind === 'retail_order') return 'import_sales_order'
  if (demandKind === 'membership_entitlement') return 'import_entitlement'
  return ''
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

/** Document types implied by enabled factory caps (listing only — never fan-out one sample). */
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

/** Fields for `createProfile` after the custom-create first screen. */
export interface CustomCreateDraft {
  displayName: string
  profileKey: string
  surface: BusinessSurfaceChoice
  factorySupplierPlatform: string
}

export type CustomCreateProfileInput = {
  profileKey: string
  sourceChannel: string
  sourceSurface: string
  demandKind: string
  initialAllocationStrategy: string
  identityStrategy: string
  entitlementAuthorityMode: string
  recipientInputMode: string
  referenceStrategy: string
  trackingSyncMode: string
  closurePolicy: string
  supportsPartialShipment: boolean
  supportsApiImport: boolean
  supportsApiExport: boolean
  requiresCarrierMapping: boolean
  requiresExternalOrderNo: boolean
  allowsManualClosure: boolean
  supportsExportSupplierOrder: boolean
  supportsImportProductCatalog: boolean
  supportsImportSupplierShipment: boolean
  connectorKey: string
  factorySupplierPlatform: string
  supportedLocales: string
  defaultLocale: string
  extraData: string
}

/** ASCII slug for profileKey; empty when the display name has no latin/digit chars. */
export function suggestProfileKey(displayName: string): string {
  return displayName
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
}

/**
 * Custom create defaults: source gets empty demandKind (both file kinds can
 * bind) plus an executable local-export pair; factory turns catalog / shipment
 * / export caps on.
 */
export function buildCustomCreateProfileInput(draft: CustomCreateDraft): CustomCreateProfileInput {
  const profileKey = draft.profileKey.trim()
  const displayName = draft.displayName.trim()
  const sourceChannel = displayName || profileKey
  if (draft.surface === 'factory') {
    const factoryLabel = draft.factorySupplierPlatform.trim() || sourceChannel
    return {
      profileKey,
      sourceChannel,
      sourceSurface: 'factory',
      demandKind: '',
      initialAllocationStrategy: '',
      identityStrategy: '',
      entitlementAuthorityMode: '',
      recipientInputMode: '',
      referenceStrategy: '',
      trackingSyncMode: 'unsupported',
      closurePolicy: '',
      supportsPartialShipment: false,
      supportsApiImport: false,
      supportsApiExport: false,
      requiresCarrierMapping: false,
      requiresExternalOrderNo: false,
      allowsManualClosure: false,
      supportsExportSupplierOrder: true,
      supportsImportProductCatalog: true,
      supportsImportSupplierShipment: true,
      connectorKey: '',
      factorySupplierPlatform: factoryLabel,
      supportedLocales: '',
      defaultLocale: '',
      extraData: '',
    }
  }
  return {
    profileKey,
    sourceChannel,
    sourceSurface: 'membership',
    demandKind: '',
    initialAllocationStrategy: '',
    identityStrategy: '',
    entitlementAuthorityMode: '',
    recipientInputMode: '',
    referenceStrategy: '',
    trackingSyncMode: 'document_export',
    closurePolicy: 'close_after_sync',
    supportsPartialShipment: false,
    supportsApiImport: false,
    supportsApiExport: false,
    requiresCarrierMapping: false,
    requiresExternalOrderNo: false,
    allowsManualClosure: false,
    supportsExportSupplierOrder: false,
    supportsImportProductCatalog: false,
    supportsImportSupplierShipment: false,
    connectorKey: 'eli.local_export',
    factorySupplierPlatform: '',
    supportedLocales: '',
    defaultLocale: '',
    extraData: '',
  }
}
