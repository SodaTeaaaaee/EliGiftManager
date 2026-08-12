import { describe, expect, it } from 'vitest'
import {
  captureProfileScope,
  isProfileEntityActive,
  isProfileLoadActive,
  isProfileScopeActive,
} from './profileScopedFlow'

describe('integration-detail profile scope', () => {
  it('invalidates every profile-scoped editor when the generation changes', () => {
    const capability = captureProfileScope(10, 1)!
    const expert = captureProfileScope(10, 1)!
    const templateCreator = captureProfileScope(10, 1)!
    const rerunWizard = captureProfileScope(10, 1)!

    for (const scope of [capability, expert, templateCreator, rerunWizard]) {
      expect(isProfileScopeActive(scope, 20, 2)).toBe(false)
    }
  })

  it('rejects carrier and template entities belonging to the previous profile', () => {
    const openedUnderA = captureProfileScope(10, 1)!
    const scope = captureProfileScope(20, 2)!

    expect(isProfileEntityActive(openedUnderA, 20, 1, 10)).toBe(false)
    expect(isProfileEntityActive(scope, 20, 2, 10)).toBe(false)
    expect(isProfileEntityActive(scope, 20, 2, 20)).toBe(true)
  })

  it('rejects an out-of-order A load after B becomes current', () => {
    const loadA = { profileId: 10, generation: 1, loadSequence: 1 }
    const loadB = { profileId: 20, generation: 2, loadSequence: 2 }

    expect(isProfileLoadActive(loadA, 20, 2, 2, 10)).toBe(false)
    expect(isProfileLoadActive(loadB, 20, 2, 2, 20)).toBe(true)
  })

  it('lets only the newest load update one unchanged profile session', () => {
    const older = { profileId: 10, generation: 1, loadSequence: 1 }
    const newer = { profileId: 10, generation: 1, loadSequence: 2 }

    expect(isProfileLoadActive(older, 10, 1, 2, 10)).toBe(false)
    expect(isProfileLoadActive(newer, 10, 1, 2, 10)).toBe(true)
  })
})
