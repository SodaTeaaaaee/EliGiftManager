export interface ProfileAvailabilityShape {
  sourceSurface: string
  demandKind: string
  supportsImportProductCatalog?: boolean
  supportsImportSupplierShipment?: boolean
  supportsExportSupplierOrder?: boolean
}

/** Seeded Bilibili source platform. Not an installable catalog of fake presets. */
export const BILIBILI_BUILTIN_PROFILE_KEY = 'bilibili_membership_demo'
/** Seeded 柔造 factory platform. */
export const ROUZAO_BUILTIN_PROFILE_KEY = 'factory_rouzao_demo'
/** Generic leftover default — never shown as an installable builtin card. */
export const MEMBERSHIP_DEFAULT_PROFILE_KEY = 'membership_default'

export const SOURCE_FILE_KINDS = [
  'import_entitlement',
  'import_sales_order',
  'import_carrier_mapping',
  'export_source_tracking_update',
] as const

export const FACTORY_FILE_KINDS = [
  'import_product_catalog',
  'import_supplier_shipment',
  'export_supplier_order',
] as const

export type BuiltinInstallKey = 'bilibili' | 'rouzao'

export interface BuiltinPlatformDef {
  installKey: BuiltinInstallKey
  profileKey: string
  i18nKey: 'bilibili' | 'rouzao'
  surface: 'source' | 'factory'
}

/**
 * Add-time shortcuts only. After install, these are ordinary profiles —
 * cards, drawer, remap, and import must not branch on this list.
 */
export const BUILTIN_PLATFORMS: readonly BuiltinPlatformDef[] = [
  {
    installKey: 'bilibili',
    profileKey: BILIBILI_BUILTIN_PROFILE_KEY,
    i18nKey: 'bilibili',
    surface: 'source',
  },
  {
    installKey: 'rouzao',
    profileKey: ROUZAO_BUILTIN_PROFILE_KEY,
    i18nKey: 'rouzao',
    surface: 'factory',
  },
]

/** True only to hide an already-added shortcut from the installable strip. */
export function isBuiltinProfileKey(profileKey: string): boolean {
  return BUILTIN_PLATFORMS.some((item) => item.profileKey === profileKey)
}

/** Builtins not yet in the added list. Installed ones are opened as normal cards. */
export function installableBuiltins<T extends { profileKey: string }>(
  profiles: T[],
): BuiltinPlatformDef[] {
  const installed = new Set(profiles.map((profile) => profile.profileKey))
  return BUILTIN_PLATFORMS.filter((item) => !installed.has(item.profileKey))
}

/** Factory-side profiles (catalog / supplier shipment / supplier-order export). */
export function isFactoryProfile(profile: ProfileAvailabilityShape): boolean {
  return profile.sourceSurface === 'factory'
}

/**
 * Source-platform profiles (档案=平台). Leftover or empty `demandKind` is not a
 * grouping key — dual-kind, empty, and membership leftover all stay here.
 */
export function isSourcePlatformProfile(profile: ProfileAvailabilityShape): boolean {
  return !isFactoryProfile(profile)
}

export function partitionProfilesForList<T extends ProfileAvailabilityShape>(
  profiles: T[],
): { source: T[]; factory: T[] } {
  const source: T[] = []
  const factory: T[] = []
  for (const profile of profiles) {
    if (isFactoryProfile(profile)) factory.push(profile)
    else source.push(profile)
  }
  return { source, factory }
}

export function canImportDemand(profile: ProfileAvailabilityShape): boolean {
  // Leftover DemandKind is not authoritative: a source platform may bind both
  // demand import types (or carry an empty hint after the intake wizard).
  return isSourcePlatformProfile(profile)
}

export function canCreateRetailDemand(profile: ProfileAvailabilityShape): boolean {
  // Same as import: leftover membership / empty demandKind still names a
  // source platform, not a second profile that should vanish from selectors.
  return canImportDemand(profile)
}

export function canImportProductCatalog(profile: ProfileAvailabilityShape): boolean {
  return isFactoryProfile(profile) && profile.supportsImportProductCatalog === true
}

export function canImportSupplierShipment(profile: ProfileAvailabilityShape): boolean {
  return isFactoryProfile(profile) && profile.supportsImportSupplierShipment === true
}

/** File kinds this platform is expected to configure (已配 / 未配 on the detail list). */
export function expectedFileKinds(profile: ProfileAvailabilityShape): string[] {
  if (isFactoryProfile(profile)) {
    const types: string[] = []
    if (profile.supportsImportProductCatalog) types.push('import_product_catalog')
    if (profile.supportsImportSupplierShipment) types.push('import_supplier_shipment')
    if (profile.supportsExportSupplierOrder) types.push('export_supplier_order')
    return types.length > 0 ? types : [...FACTORY_FILE_KINDS]
  }
  return [...SOURCE_FILE_KINDS]
}

export function hasDefaultFileKindBinding(
  bindings: Array<{ documentType: string; isDefault?: boolean }>,
  documentType: string,
): boolean {
  return bindings.some((binding) => binding.documentType === documentType && binding.isDefault === true)
}
