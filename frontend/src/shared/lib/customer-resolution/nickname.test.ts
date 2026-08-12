import { describe, expect, test } from 'vitest'
import {
  buildNicknameTimeline,
  canSaveDisplayName,
  createDisplayNameEditState,
  reduceDisplayNameEditState,
} from './nickname'
import type { CustomerNameObservationInput } from './types'

function observation(id: number, value: string, day: number): CustomerNameObservationInput {
  const date = `2026-01-${String(day).padStart(2, '0')}T00:00:00Z`
  return {
    id,
    kind: 'platform_nickname',
    displayValue: value,
    normalizedValue: value,
    sourceNamespace: 'bilibili:demo',
    firstSeenAt: date,
    lastSeenAt: date,
    observationCount: 1,
  }
}

describe('nickname timeline', () => {
  test('coalesces A → A → B into two newest-first episodes', () => {
    const result = buildNicknameTimeline([observation(1, 'A', 1), observation(2, 'A', 2), observation(3, 'B', 3)])
    expect(result.map((item) => item.displayValue)).toEqual(['B', 'A'])
    expect(result[1].observationCount).toBe(2)
    expect(result[1].lastSeenAt).toContain('02')
  })

  test('preserves A → B → A as three episodes and current marker through coalescing', () => {
    const result = buildNicknameTimeline(
      [observation(1, 'A', 1), observation(2, 'B', 2), observation(3, 'A', 3)],
      1,
    )
    expect(result.map((item) => item.displayValue)).toEqual(['A', 'B', 'A'])
    expect(result[2].isCurrentDisplayName).toBe(true)
  })

  test('uses id as a stable tie-break for equal timestamps', () => {
    const first = observation(1, 'A', 1)
    const second = observation(2, 'B', 1)
    expect(buildNicknameTimeline([second, first]).map((item) => item.id)).toEqual([2, 1])
  })

  test('extends the latest episode per source stream across observations from another source', () => {
    const first = observation(1, 'A', 1)
    const other = { ...observation(2, 'X', 2), sourceNamespace: 'manual' }
    const repeated = observation(3, 'A', 3)
    const result = buildNicknameTimeline([first, other, repeated])
    expect(result).toHaveLength(2)
    expect(result[0].displayValue).toBe('A')
    expect(result[0].observationCount).toBe(2)
  })
})

describe('display name reducer', () => {
  test('manual editing pins the name and becomes saveable', () => {
    const initial = createDisplayNameEditState({ name: 'Old', mode: 'auto', autoName: 'Old', rowVersion: 4 })
    const edited = reduceDisplayNameEditState(initial, { type: 'edit_name', value: 'Manual' })
    expect(edited.mode).toBe('pinned')
    expect(canSaveDisplayName(edited)).toBe(true)
  })

  test('switching to auto previews the auto-selected name', () => {
    const initial = createDisplayNameEditState({ name: 'Pinned', mode: 'pinned', autoName: 'Latest', rowVersion: 4 })
    const automatic = reduceDisplayNameEditState(initial, { type: 'select_mode', mode: 'auto' })
    expect(automatic.draftName).toBe('Latest')
    expect(automatic.dirty).toBe(true)
  })

  test('empty pinned names cannot be saved', () => {
    const initial = createDisplayNameEditState({ name: 'Old', mode: 'pinned', autoName: 'Auto', rowVersion: 4 })
    const edited = reduceDisplayNameEditState(initial, { type: 'edit_name', value: '  ' })
    expect(canSaveDisplayName(edited)).toBe(false)
  })

  test('reset restores the persisted mode and name', () => {
    const initial = createDisplayNameEditState({ name: 'Pinned', mode: 'pinned', autoName: 'Auto', rowVersion: 4 })
    const automatic = reduceDisplayNameEditState(initial, { type: 'select_mode', mode: 'auto' })
    const reset = reduceDisplayNameEditState(automatic, { type: 'reset' })
    expect(reset.mode).toBe('pinned')
    expect(reset.draftName).toBe('Pinned')
    expect(reset.dirty).toBe(false)
  })
})
