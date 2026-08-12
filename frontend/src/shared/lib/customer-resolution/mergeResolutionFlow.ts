import {
  buildMergeResolutionRequest,
  type MergeIdentityResolutionGroup,
  type MergeResolutionState,
} from './mergeResolution'

export interface MergeResolutionRequest {
  primaryIdentitySelections: Array<{ namespace: string; identityType: string; identityId: number }>
  defaultAddressId?: number
  displayNameResolution: string
}

export interface MergeResolutionPreviewFlow {
  acceptedPreviewToken: string | null
  acceptedRequest: MergeResolutionRequest | null
  dirty: boolean
}

export interface MergeResolutionPreviewAttempt {
  flow: MergeResolutionPreviewFlow
  request: MergeResolutionRequest
}

export type MergeExecutionErrorKind = 'stale' | 'feature_disabled' | 'other'

export interface MergeOperationAccess {
  canPreview: true
  canReadHistory: true
  canDryRunUndo: true
  canExecuteMerge: boolean
  canExecuteUndo: boolean
}

export function mergeOperationAccess(writesEnabled: boolean): MergeOperationAccess {
  return {
    canPreview: true,
    canReadHistory: true,
    canDryRunUndo: true,
    canExecuteMerge: writesEnabled,
    canExecuteUndo: writesEnabled,
  }
}

export function classifyMergeExecutionError(error: unknown): MergeExecutionErrorKind {
  const message = (error instanceof Error ? error.message : String(error)).toLowerCase()
  if (
    message.includes('merge_execution_disabled')
    || message.includes('customer_resolution_writes_disabled')
    || message.includes('customer_resolution_feature_policy_unavailable')
  ) return 'feature_disabled'
  if (
    message.includes('stale merge preview:')
    || message.includes('merge_preview_stale')
    || message.includes('merge_preview_token_mismatch')
    || message.includes('merge preview token mismatch')
  ) return 'stale'
  return 'other'
}

export function createMergeResolutionPreviewFlow(): MergeResolutionPreviewFlow {
  return {
    acceptedPreviewToken: null,
    acceptedRequest: null,
    dirty: false,
  }
}

export function invalidateMergeResolutionPreview(
  flow: MergeResolutionPreviewFlow,
): MergeResolutionPreviewFlow {
  return {
    ...flow,
    acceptedPreviewToken: null,
    acceptedRequest: null,
    dirty: true,
  }
}

export function beginMergeResolutionPreview(
  flow: MergeResolutionPreviewFlow,
  state: MergeResolutionState,
  groups: MergeIdentityResolutionGroup[],
): MergeResolutionPreviewAttempt {
  return {
    flow: invalidateMergeResolutionPreview(flow),
    request: buildMergeResolutionRequest(state, groups),
  }
}

export function acceptMergeResolutionPreview(
  flow: MergeResolutionPreviewFlow,
  previewToken: string,
  request: MergeResolutionRequest,
): MergeResolutionPreviewFlow {
  return {
    ...flow,
    acceptedPreviewToken: previewToken,
    acceptedRequest: {
      ...request,
      primaryIdentitySelections: request.primaryIdentitySelections.map((selection) => ({ ...selection })),
    },
    dirty: false,
  }
}

export function mergeResolutionExecuteRequest(
  flow: MergeResolutionPreviewFlow,
  previewToken: string,
): MergeResolutionRequest | null {
  if (flow.dirty || flow.acceptedPreviewToken !== previewToken || !flow.acceptedRequest) return null
  return {
    ...flow.acceptedRequest,
    primaryIdentitySelections: flow.acceptedRequest.primaryIdentitySelections.map((selection) => ({ ...selection })),
  }
}
