import { describe, expect, test } from 'vitest'
import { isMergePolicyRevisionConflict, migrateLegacyMergeSettings, setCandidateDetectionEnabled } from './mergePolicy'

describe('legacy merge policy migration', () => {
  for (const master of [false, true]) {
    for (const email of [false, true]) {
      for (const phone of [false, true]) {
        test(`${Number(master)}${Number(email)}${Number(phone)} preserves detection and dormant rules`, () => {
          const result = migrateLegacyMergeSettings({
            autoMergeCrossPlatform: master,
            autoMergeByEmail: email,
            autoMergeByPhone: phone,
          })

          expect(result.candidateDetectionEnabled).toBe(master)
          expect(result.email.enabled).toBe(email)
          expect(result.phone.enabled).toBe(phone)
          expect(result.email.dormant).toBe(email && !master)
          expect(result.phone.dormant).toBe(phone && !master)
          expect(result.email.mode).toBe(email ? 'legacy_raw_exact' : 'off')
          expect(result.phone.mode).toBe(phone ? 'legacy_raw_exact' : 'off')
          expect(result.executionMode).toBe('suggest_only')
        })
      }
    }
  }

  test('turning detection off retains child choices as dormant', () => {
    const enabled = migrateLegacyMergeSettings({
      autoMergeCrossPlatform: true,
      autoMergeByEmail: true,
      autoMergeByPhone: true,
    })

    const disabled = setCandidateDetectionEnabled(enabled, false)
    expect(disabled.email.enabled).toBe(true)
    expect(disabled.phone.enabled).toBe(true)
    expect(disabled.email.dormant).toBe(true)
    expect(disabled.phone.dormant).toBe(true)
  })

  test('classifies only the backend CAS error as a revision conflict', () => {
    expect(isMergePolicyRevisionConflict(new Error('update policy: merge policy revision conflict'))).toBe(true)
    expect(isMergePolicyRevisionConflict(new Error('network unavailable'))).toBe(false)
  })
})
