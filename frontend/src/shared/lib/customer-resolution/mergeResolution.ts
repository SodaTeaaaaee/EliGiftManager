export interface MergePrimaryIdentityOption {
  namespace: string
  identityType: string
  identityId: number
  customerProfileId: number
  displayValue: string
  currentPrimary: boolean
}

export interface MergeDefaultAddressOption {
  addressId: number
  customerProfileId: number
  displayValue: string
  currentDefault: boolean
}

export interface MergeDisplayNameOption {
  resolution: string
  displayName: string
  profileId: number
}

export interface MergeIdentityResolutionGroup {
  key: string
  namespace: string
  identityType: string
  options: MergePrimaryIdentityOption[]
}

export interface MergeResolutionOptionsViewModel {
  primaryIdentityGroups: MergeIdentityResolutionGroup[]
  defaultAddresses: MergeDefaultAddressOption[]
  displayNames: MergeDisplayNameOption[]
}

export interface MergeResolutionState {
  primaryIdentitySelections: Record<string, number>
  defaultAddressId: number | null
  displayNameResolution: string
}

export function identityResolutionGroupKey(namespace: string, identityType: string): string {
  return `${namespace.trim().toLowerCase()}\u0000${identityType.trim().toLowerCase()}`
}

export function buildMergeResolutionOptions(input: {
  primaryIdentityOptions?: MergePrimaryIdentityOption[]
  defaultAddressOptions?: MergeDefaultAddressOption[]
  displayNameOptions?: MergeDisplayNameOption[]
}): MergeResolutionOptionsViewModel {
  const grouped = new Map<string, MergePrimaryIdentityOption[]>()
  for (const identity of input.primaryIdentityOptions ?? []) {
    const key = identityResolutionGroupKey(identity.namespace, identity.identityType)
    grouped.set(key, [...(grouped.get(key) ?? []), identity])
  }
  return {
    primaryIdentityGroups: [...grouped.entries()].map(([key, options]) => ({
      key,
      namespace: options[0].namespace,
      identityType: options[0].identityType,
      options,
    })),
    defaultAddresses: input.defaultAddressOptions ?? [],
    displayNames: input.displayNameOptions ?? [],
  }
}

export function recommendedMergeResolutionState(input: {
  recommendedPrimaryIdentitySelections?: Array<{ namespace: string; identityType: string; identityId: number }>
  recommendedDefaultAddressId?: number
  recommendedDisplayNameResolution?: string
}): MergeResolutionState {
  return {
    primaryIdentitySelections: Object.fromEntries(
      (input.recommendedPrimaryIdentitySelections ?? []).map((selection) => [
        identityResolutionGroupKey(selection.namespace, selection.identityType),
        selection.identityId,
      ]),
    ),
    defaultAddressId: input.recommendedDefaultAddressId ?? null,
    displayNameResolution: input.recommendedDisplayNameResolution ?? '',
  }
}

export function buildMergeResolutionRequest(
  state: MergeResolutionState,
  groups: MergeIdentityResolutionGroup[],
): {
  primaryIdentitySelections: Array<{ namespace: string; identityType: string; identityId: number }>
  defaultAddressId?: number
  displayNameResolution: string
} {
  return {
    primaryIdentitySelections: groups.flatMap((group) => {
      const identityId = state.primaryIdentitySelections[group.key]
      return identityId == null ? [] : [{ namespace: group.namespace, identityType: group.identityType, identityId }]
    }),
    ...(state.defaultAddressId == null ? {} : { defaultAddressId: state.defaultAddressId }),
    displayNameResolution: state.displayNameResolution,
  }
}
