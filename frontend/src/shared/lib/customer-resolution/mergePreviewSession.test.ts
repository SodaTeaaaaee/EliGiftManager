import { describe, expect, it } from 'vitest'
import {
  acceptMergePreviewSession,
  acceptedMergePreviewIsCurrent,
  captureMergePreviewSession,
  mergePreviewResponseIsCurrent,
} from './mergePreviewSession'

describe('merge preview request session', () => {
  it('rejects an A response after the dialog switches to B', () => {
    const request = { ...captureMergePreviewSession({ sourceProfileId: 1, targetProfileId: 2, candidateId: 3, generation: 1 })!, requestSequence: 1 }
    expect(mergePreviewResponseIsCurrent(
      request,
      { sourceProfileId: 4, targetProfileId: 5, candidateId: 6, generation: 2, requestSequence: 2 },
      { sourceProfileId: 1, targetProfileId: 2, candidateId: 3 },
      1,
      2,
    )).toBe(false)
  })

  it('rejects mismatched source, target, candidate, and profile entities', () => {
    const request = { ...captureMergePreviewSession({ sourceProfileId: 1, targetProfileId: 2, candidateId: 3, generation: 1 })!, requestSequence: 1 }
    const current = { sourceProfileId: 1, targetProfileId: 2, candidateId: 3, generation: 1, requestSequence: 1 }

    expect(mergePreviewResponseIsCurrent(request, current, { sourceProfileId: 9, targetProfileId: 2, candidateId: 3 }, 1, 2)).toBe(false)
    expect(mergePreviewResponseIsCurrent(request, current, { sourceProfileId: 1, targetProfileId: 2, candidateId: 8 }, 1, 2)).toBe(false)
    expect(mergePreviewResponseIsCurrent(request, current, { sourceProfileId: 1, targetProfileId: 2, candidateId: 3 }, 9, 2)).toBe(false)
  })

  it('executes only the token accepted for the current session', () => {
    const request = { ...captureMergePreviewSession({ sourceProfileId: 1, targetProfileId: 2, generation: 1 })!, requestSequence: 1 }
    const accepted = acceptMergePreviewSession(request, 'token-a')

    expect(acceptedMergePreviewIsCurrent(
      accepted,
      { sourceProfileId: 1, targetProfileId: 2, generation: 1 },
      { sourceProfileId: 1, targetProfileId: 2, previewToken: 'token-a' },
    )).toBe(true)
    expect(acceptedMergePreviewIsCurrent(
      accepted,
      { sourceProfileId: 1, targetProfileId: 2, generation: 2 },
      { sourceProfileId: 1, targetProfileId: 2, previewToken: 'token-a' },
    )).toBe(false)
  })
})
