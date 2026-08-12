import { describe, expect, it } from 'vitest'
import { resolveImportEvidenceReference } from './importEvidenceReference'

describe('import evidence reference', () => {
  it('returns a retained run id', () => {
    expect(resolveImportEvidenceReference({ importRunId: 42, evidenceDisabled: false })).toEqual({ kind: 'run', importRunId: 42 })
  })

  it('does not invent a run when evidence is disabled', () => {
    expect(resolveImportEvidenceReference({ importRunId: 0, evidenceDisabled: true })).toEqual({ kind: 'disabled' })
  })
})
