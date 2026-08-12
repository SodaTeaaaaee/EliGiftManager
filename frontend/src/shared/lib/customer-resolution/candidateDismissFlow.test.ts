import { describe, expect, it } from 'vitest'
import { classifyCandidateDismissError } from './candidateDismissFlow'

describe('candidate dismiss error classification', () => {
  it('keeps stable feature-policy errors out of the stale-candidate path', () => {
    expect(classifyCandidateDismissError(new Error('candidate_scan_disabled: disabled by operator'))).toBe('feature_disabled')
    expect(classifyCandidateDismissError(new Error('customer_resolution_writes_disabled: disabled by operator'))).toBe('feature_disabled')
  })

  it('uses changed only for the backend candidate-CAS failure', () => {
    expect(classifyCandidateDismissError(new Error('merge candidate evidence or policy version changed'))).toBe('changed')
    expect(classifyCandidateDismissError(new Error('database temporarily unavailable'))).toBe('other')
    expect(classifyCandidateDismissError(new Error('candidate id, evidenceHash, and policyVersion are required'))).toBe('other')
  })
})
