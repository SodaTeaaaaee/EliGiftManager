import type { CustomerSplitPlanRequest } from '@/shared/api/bridge'

export interface CustomerSplitPlanState {
  sourceProfileId: number
  newProfileDisplayName: string
  newProfileType: string
  targetPrimaryIdentityIds: number[]
  targetDefaultAddressId: number | null
  targetDisplayNameObservationId: number | null
  sourceDisplayNameResolution: string
  identityIds: number[]
  addressIds: number[]
  demandDocumentIds: number[]
  nameObservationIds: number[]
  originIds: number[]
}

function stableIds(values: number[]): number[] {
  return [...new Set(values)].sort((a, b) => a - b)
}

export function buildCustomerSplitPlan(state: CustomerSplitPlanState): CustomerSplitPlanRequest {
  return {
    sourceProfileId: state.sourceProfileId,
    targetStrategy: 'create_new',
    newProfileDisplayName: state.newProfileDisplayName.trim(),
    newProfileType: state.newProfileType,
    targetPrimaryIdentityIds: stableIds(state.targetPrimaryIdentityIds),
    ...(state.targetDefaultAddressId == null ? {} : { targetDefaultAddressId: state.targetDefaultAddressId }),
    ...(state.targetDisplayNameObservationId == null ? {} : { targetDisplayNameObservationId: state.targetDisplayNameObservationId }),
    sourceDisplayNameResolution: state.sourceDisplayNameResolution,
    selection: {
      identityIds: stableIds(state.identityIds),
      addressIds: stableIds(state.addressIds),
      demandDocumentIds: stableIds(state.demandDocumentIds),
      nameObservationIds: stableIds(state.nameObservationIds),
      originIds: stableIds(state.originIds),
    },
  }
}
