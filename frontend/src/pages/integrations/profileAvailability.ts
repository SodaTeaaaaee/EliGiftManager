export interface ProfileAvailabilityShape {
  sourceSurface: string
  demandKind: string
  supportsImportProductCatalog?: boolean
  supportsImportSupplierShipment?: boolean
}

export function canImportDemand(profile: ProfileAvailabilityShape): boolean {
  return profile.sourceSurface !== 'factory' &&
    (profile.demandKind === 'membership_entitlement' || profile.demandKind === 'retail_order')
}

export function canCreateRetailDemand(profile: ProfileAvailabilityShape): boolean {
  return profile.sourceSurface !== 'factory' && profile.demandKind === 'retail_order'
}

export function canImportProductCatalog(profile: ProfileAvailabilityShape): boolean {
  return profile.sourceSurface === 'factory' && profile.supportsImportProductCatalog === true
}

export function canImportSupplierShipment(profile: ProfileAvailabilityShape): boolean {
  return profile.sourceSurface === 'factory' && profile.supportsImportSupplierShipment === true
}
