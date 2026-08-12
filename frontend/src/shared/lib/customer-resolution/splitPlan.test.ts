import { describe, expect, it } from 'vitest'
import { buildCustomerSplitPlan } from './splitPlan'

describe('customer split plan mapping', () => {
  it('uses create_new and maps every selected entity and projection field', () => {
    expect(buildCustomerSplitPlan({
      sourceProfileId: 7,
      newProfileDisplayName: '  New owner  ',
      newProfileType: 'buyer',
      targetPrimaryIdentityIds: [4, 4],
      targetDefaultAddressId: 9,
      targetDisplayNameObservationId: 12,
      sourceDisplayNameResolution: 'auto_remaining',
      identityIds: [4, 3],
      addressIds: [9],
      demandDocumentIds: [22],
      nameObservationIds: [12],
      originIds: [31],
    })).toEqual({
      sourceProfileId: 7,
      targetStrategy: 'create_new',
      newProfileDisplayName: 'New owner',
      newProfileType: 'buyer',
      targetPrimaryIdentityIds: [4],
      targetDefaultAddressId: 9,
      targetDisplayNameObservationId: 12,
      sourceDisplayNameResolution: 'auto_remaining',
      selection: {
        identityIds: [3, 4],
        addressIds: [9],
        demandDocumentIds: [22],
        nameObservationIds: [12],
        originIds: [31],
      },
    })
  })
})
