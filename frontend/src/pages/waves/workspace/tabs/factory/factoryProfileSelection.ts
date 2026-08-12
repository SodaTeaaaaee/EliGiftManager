export interface FactoryProfileOptionSource {
  id: number
  profileKey: string
  sourceSurface: string
  supportsExportSupplierOrder: boolean
  factorySupplierPlatform: string
}

export interface FactoryOrderProfileSource {
  supplierPlatform: string
  factoryIntegrationProfileId?: number | null
}

export type FactoryGenerationDecision =
  | { kind: 'new_platform' }
  | { kind: 'rebuild_profile' }
  | { kind: 'profile_conflict'; existingProfileId: number | null }

export function eligibleFactoryProfiles<T extends FactoryProfileOptionSource>(
  profiles: T[],
): T[] {
  return profiles
    .filter((profile) =>
      profile.sourceSurface === 'factory' &&
      profile.supportsExportSupplierOrder &&
      profile.factorySupplierPlatform.trim().length > 0,
    )
    .sort((left, right) => left.profileKey.localeCompare(right.profileKey) || left.id - right.id)
}

/**
 * Classifies a profile choice before export. A different platform is an
 * independent factory slice. Reusing the same profile is an explicit rebuild.
 * A different profile for an already-present platform is blocked by the UI:
 * the backend rebuild is profile-scoped, so running it would duplicate the
 * platform's fulfillment lines instead of replacing the existing drafts.
 */
export function factoryGenerationDecision(
  profile: FactoryProfileOptionSource,
  orders: FactoryOrderProfileSource[],
): FactoryGenerationDecision {
  const platform = profile.factorySupplierPlatform.trim()
  const samePlatform = orders.filter((order) => order.supplierPlatform.trim() === platform)
  if (samePlatform.length === 0) return { kind: 'new_platform' }
  if (samePlatform.some((order) => order.factoryIntegrationProfileId === profile.id)) {
    return { kind: 'rebuild_profile' }
  }
  return {
    kind: 'profile_conflict',
    existingProfileId: samePlatform.find((order) => order.factoryIntegrationProfileId != null)
      ?.factoryIntegrationProfileId ?? null,
  }
}
