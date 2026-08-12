import { describe, expect, it } from 'vitest'
import {
  acceptMergeResolutionPreview,
  beginMergeResolutionPreview,
  classifyMergeExecutionError,
  createMergeResolutionPreviewFlow,
  invalidateMergeResolutionPreview,
  mergeOperationAccess,
  mergeResolutionExecuteRequest,
} from './mergeResolutionFlow'
import { buildMergeResolutionOptions, identityResolutionGroupKey } from './mergeResolution'

const groups = buildMergeResolutionOptions({
  primaryIdentityOptions: [
    { namespace: 'Shop', identityType: 'email', identityId: 1, customerProfileId: 10, displayValue: 'first', currentPrimary: true },
    { namespace: 'shop', identityType: 'email', identityId: 2, customerProfileId: 20, displayValue: 'second', currentPrimary: true },
  ],
}).primaryIdentityGroups

const draft = {
  primaryIdentitySelections: { [identityResolutionGroupKey('shop', 'email')]: 2 },
  defaultAddressId: 22,
  displayNameResolution: 'keep_target',
}

describe('merge resolution preview flow', () => {
  it('captures primary, default-address, and display-name choices before invalidating the old preview', () => {
    const accepted = acceptMergeResolutionPreview(createMergeResolutionPreviewFlow(), 'old-token', {
      primaryIdentitySelections: [],
      displayNameResolution: '',
    })

    const attempt = beginMergeResolutionPreview(accepted, draft, groups)

    expect(attempt.request).toEqual({
      primaryIdentitySelections: [{ namespace: 'Shop', identityType: 'email', identityId: 2 }],
      defaultAddressId: 22,
      displayNameResolution: 'keep_target',
    })
    expect(attempt.flow.acceptedPreviewToken).toBeNull()
    expect(attempt.flow.acceptedRequest).toBeNull()
  })

  it('invalidates the accepted token as soon as a choice becomes dirty', () => {
    const accepted = acceptMergeResolutionPreview(createMergeResolutionPreviewFlow(), 'token-a', {
      primaryIdentitySelections: [],
      displayNameResolution: '',
    })

    const invalidated = invalidateMergeResolutionPreview(accepted)

    expect(mergeResolutionExecuteRequest(invalidated, 'token-a')).toBeNull()
  })

  it('retries a stale preview with the preserved draft and executes only against the replacement token', () => {
    const firstAttempt = beginMergeResolutionPreview(createMergeResolutionPreviewFlow(), draft, groups)
    const firstAccepted = acceptMergeResolutionPreview(firstAttempt.flow, 'token-a', firstAttempt.request)
    const staleRetry = beginMergeResolutionPreview(firstAccepted, draft, groups)
    const retried = acceptMergeResolutionPreview(staleRetry.flow, 'token-b', staleRetry.request)

    expect(mergeResolutionExecuteRequest(retried, 'token-a')).toBeNull()
    expect(mergeResolutionExecuteRequest(retried, 'token-b')).toEqual(staleRetry.request)
    expect(staleRetry.request).toEqual(firstAttempt.request)
  })

  it('keeps preview, history, and undo dry-run readable while gating only writes', () => {
    expect(mergeOperationAccess(false)).toEqual({
      canPreview: true,
      canReadHistory: true,
      canDryRunUndo: true,
      canExecuteMerge: false,
      canExecuteUndo: false,
    })
  })

  it('re-previews only explicit stale or token-mismatch execution errors', () => {
    expect(classifyMergeExecutionError(new Error('stale merge preview: profile row version changed'))).toBe('stale')
    expect(classifyMergeExecutionError(new Error('merge_preview_token_mismatch'))).toBe('stale')
    expect(classifyMergeExecutionError(new Error('merge_execution_disabled: feature is disabled'))).toBe('feature_disabled')
    expect(classifyMergeExecutionError(new Error('database unavailable'))).toBe('other')
  })
})
