import { describe, expect, test } from 'vitest'
import {
  buildMergeResolutionOptions,
  buildMergeResolutionRequest,
  identityResolutionGroupKey,
  recommendedMergeResolutionState,
} from './mergeResolution'

describe('merge resolution payload', () => {
  const options = buildMergeResolutionOptions({
    primaryIdentityOptions: [
      { namespace: 'Shop', identityType: 'email', identityId: 1, customerProfileId: 10, displayValue: 'first', currentPrimary: true },
      { namespace: 'shop', identityType: 'email', identityId: 2, customerProfileId: 20, displayValue: 'second', currentPrimary: true },
    ],
    defaultAddressOptions: [
      { addressId: 11, customerProfileId: 10, displayValue: 'first address', currentDefault: true },
      { addressId: 22, customerProfileId: 20, displayValue: 'second address', currentDefault: true },
    ],
    displayNameOptions: [
      { resolution: 'keep_source', displayName: 'First', profileId: 10 },
      { resolution: 'keep_target', displayName: 'Second', profileId: 20 },
    ],
  })

  test('uses and groups only backend-provided resolution options', () => {
    expect(options.primaryIdentityGroups).toHaveLength(1)
    expect(options.primaryIdentityGroups[0].options.map((item) => item.identityId)).toEqual([1, 2])
    expect(options.displayNames.map((item) => item.resolution)).toEqual(['keep_source', 'keep_target'])
  })

  test('seeds the official backend recommendations', () => {
    expect(recommendedMergeResolutionState({
      recommendedPrimaryIdentitySelections: [{ namespace: 'Shop', identityType: 'email', identityId: 2 }],
      recommendedDefaultAddressId: 22,
      recommendedDisplayNameResolution: 'keep_target',
    })).toEqual({
      primaryIdentitySelections: { [identityResolutionGroupKey('shop', 'email')]: 2 },
      defaultAddressId: 22,
      displayNameResolution: 'keep_target',
    })
  })

  test('builds the exact preview and execute resolution payload', () => {
    const state = recommendedMergeResolutionState({
      recommendedPrimaryIdentitySelections: [{ namespace: 'Shop', identityType: 'email', identityId: 2 }],
      recommendedDefaultAddressId: 22,
      recommendedDisplayNameResolution: 'keep_target',
    })
    expect(buildMergeResolutionRequest(state, options.primaryIdentityGroups)).toEqual({
      primaryIdentitySelections: [{ namespace: 'Shop', identityType: 'email', identityId: 2 }],
      defaultAddressId: 22,
      displayNameResolution: 'keep_target',
    })
  })
})
