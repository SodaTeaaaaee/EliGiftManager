import { describe, expect, it } from 'vitest'
import { buildWaveFilterLink } from './wave-filter-link'

const filter = (overrides: Record<string, unknown> = {}) =>
  ({ allocationState: 'missing', stepKey: '', ...overrides }) as never

describe('buildWaveFilterLink', () => {
  it('targets the lines tab with singular query keys', () => {
    const link = buildWaveFilterLink(3, filter())
    expect(link).toMatchObject({ name: 'wave-workspace-lines', params: { id: 3 }, query: { allocationState: 'missing' } })
  })
  it('targets intake tab when stepKey is intake and drops grid filters', () => {
    const link = buildWaveFilterLink(3, filter({ stepKey: 'intake' }))
    expect(link).toMatchObject({ name: 'wave-workspace-intake', params: { id: 3 }, query: {} })
  })
})
