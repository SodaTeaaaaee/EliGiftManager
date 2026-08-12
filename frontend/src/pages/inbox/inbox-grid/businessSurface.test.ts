import { describe, expect, it } from 'vitest'
import { kindsFromSurface, surfaceFromKinds } from './businessSurface'

describe('businessSurface', () => {
  it('derives all from 0 or 2 selected kinds', () => {
    expect(surfaceFromKinds([])).toBe('all')
    expect(surfaceFromKinds(['membership_entitlement', 'retail_order'])).toBe('all')
  })
  it('derives the single selected kind', () => {
    expect(surfaceFromKinds(['retail_order'])).toBe('retail_order')
    expect(surfaceFromKinds(['membership_entitlement'])).toBe('membership_entitlement')
  })
  it('maps surface back to kinds', () => {
    expect(kindsFromSurface('all')).toEqual([])
    expect(kindsFromSurface('retail_order')).toEqual(['retail_order'])
  })
})
