import { describe, expect, test } from 'vitest'
import {
  buildCandidateViewModel,
  candidateActions,
  candidateStatusTone,
  maskEvidenceValue,
  sortEvidence,
  toEvidenceViewModel,
} from './candidates'
import type { MergeCandidateStatus } from './types'

describe('candidate presentation', () => {
  test('sorts blockers before negative and positive evidence', () => {
    const sorted = sortEvidence([
      toEvidenceViewModel({ id: 'positive', type: 'phone', polarity: 'positive', strength: 'medium', explanationCode: 'phone' }),
      toEvidenceViewModel({ id: 'negative', type: 'name', polarity: 'negative', strength: 'weak', explanationCode: 'name' }),
      toEvidenceViewModel({ id: 'blocker', type: 'uid', polarity: 'blocker', strength: 'hard', explanationCode: 'uid_conflict' }),
    ])
    expect(sorted.map((item) => item.id)).toEqual(['blocker', 'negative', 'positive'])
  })

  test('masks raw sensitive evidence and honors backend-masked values', () => {
    expect(maskEvidenceValue('email', 'alice@example.com')).toBe('a***@example.com')
    expect(maskEvidenceValue('phone', '138 0013 8000')).toBe('***8000')
    expect(maskEvidenceValue('address', 'Some full address')).toBe('***')
    expect(toEvidenceViewModel({
      id: '1', type: 'phone', polarity: 'positive', strength: 'medium', explanationCode: 'phone', rawValue: '13800138000', maskedValue: 'backend-mask',
    }).displayValue).toBe('backend-mask')
  })

  test('blocker evidence disables preview even while candidate is pending', () => {
    const result = buildCandidateViewModel(1, 'pending', [
      { id: '1', type: 'uid', polarity: 'blocker', strength: 'hard', explanationCode: 'uid_conflict' },
    ])
    expect(result.isBlocked).toBe(true)
    expect(result.actions.canPreview).toBe(false)
  })

  test('maps every status to stable actions and tones', () => {
    const statuses: MergeCandidateStatus[] = [
      'pending', 'reviewing', 'blocked', 'stale', 'dismissed', 'superseded', 'expired', 'executing', 'merged', 'failed',
    ]
    const result = Object.fromEntries(statuses.map((status) => [status, { actions: candidateActions(status), tone: candidateStatusTone(status) }]))

    expect(result.pending.actions.canPreview).toBe(true)
    expect(result.blocked.actions.canPreview).toBe(false)
    expect(result.stale.actions.canRescan).toBe(true)
    expect(result.executing.actions.canPreview).toBe(false)
    expect(result.merged.actions.canViewHistory).toBe(true)
    expect(result.failed.tone).toBe('error')
  })
})
