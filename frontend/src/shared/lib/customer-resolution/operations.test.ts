import { describe, expect, test } from 'vitest'
import { buildMergeHistory, buildUndoDryRun, summarizeSplitSelection } from './operations'

describe('merge history and undo dry-run', () => {
  test('sorts history newest-first and only offers dry-run for active completed operations', () => {
    const history = buildMergeHistory([
      { id: 1, sourceProfileId: 1, targetProfileId: 2, status: 'completed', triggerType: 'manual', createdAt: '2026-01-01T00:00:00Z' },
      { id: 2, sourceProfileId: 3, targetProfileId: 2, status: 'undone', triggerType: 'policy', createdAt: '2026-01-02T00:00:00Z', undoneAt: '2026-01-03T00:00:00Z' },
    ])
    expect(history.map((item) => item.id)).toEqual([2, 1])
    expect(history[0].canRequestUndoDryRun).toBe(false)
    expect(history[1].canRequestUndoDryRun).toBe(true)
  })

  test('undo confirmation requires eligibility and no blockers', () => {
    const base = {
      mergeId: 1,
      eligible: true,
      blockers: [],
      restoreCounts: { identities: 1, addresses: 2, nameObservations: 3, demandDocuments: 4 },
    }
    expect(buildUndoDryRun(base).canConfirm).toBe(true)
    expect(buildUndoDryRun(base).totalRestoreCount).toBe(10)
    expect(buildUndoDryRun({ ...base, blockers: [{ code: 'ownership_changed' }] }).canConfirm).toBe(false)
  })
})

describe('split selection validation', () => {
  test('requires a selection', () => {
    const result = summarizeSplitSelection([], { newProfileDisplayName: '' })
    expect(result.validationCodes).toEqual(['selection_required'])
    expect(result.canPreview).toBe(false)
  })

  test('allows an identity-backed split', () => {
    const result = summarizeSplitSelection([
      { entityType: 'identity', entityId: 1, selected: true, isPrimary: true, primaryGroup: 'bilibili:uid' },
      { entityType: 'name_observation', entityId: 2, selected: true },
    ], { newProfileDisplayName: '' })
    expect(result.validationCodes).toEqual([])
    expect(result.canPreview).toBe(true)
  })

  test('rejects frozen documents and conflicting default or primary selections', () => {
    const result = summarizeSplitSelection([
      { entityType: 'demand_document', entityId: 1, selected: true, frozen: true },
      { entityType: 'address', entityId: 2, selected: true, isDefault: true },
      { entityType: 'address', entityId: 3, selected: true, isDefault: true },
      { entityType: 'identity', entityId: 4, selected: true, isPrimary: true, primaryGroup: 'uid' },
      { entityType: 'identity', entityId: 5, selected: true, isPrimary: true, primaryGroup: 'uid' },
    ], { newProfileDisplayName: 'Restored customer' })
    expect(result.validationCodes).toEqual([
      'frozen_document_selected',
      'multiple_default_addresses',
      'multiple_primary_identities',
    ])
    expect(result.canPreview).toBe(false)
  })
})
