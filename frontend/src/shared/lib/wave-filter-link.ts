/**
 * `buildWaveFilterLink` — the SINGLE source for the task-center-bucket ->
 * wave-workspace deep-link query mapping. Both the task center (P1) and the
 * fulfillment grid (P3) import this instead of hand-inlining the mapping, so
 * the singular ActionCenterBucketFilterDTO fields and the singular
 * URL query keys the grid's `useUrlFilters` schema reads can never drift
 * apart.
 *
 * Targets `wave-workspace-lines` (the fulfillment grid tab reads those
 * singular keys via its `FULFILLMENT_GRID_FILTER_SCHEMA`); a filter whose
 * `stepKey` is `'intake'` targets `wave-workspace-intake` instead and drops
 * the grid pre-filters entirely.
 *
 * Encoding matches `shared/ui/filter-bar/useUrlFilters.ts`'s
 * `serializeEnumMultiQuery`: a single filter value becomes a one-element
 * array, comma-joined. Empty/absent fields are omitted entirely (never an
 * empty-string query param).
 */
import type { RouteLocationRaw } from 'vue-router'
import type { dto } from '../../../wailsjs/go/models'
import { serializeEnumMultiQuery } from '@/shared/ui/filter-bar/useUrlFilters'

/** Singular DTO field -> singular URL query key (matches the grid's useUrlFilters schema keys). */
const FIELD_TO_QUERY_KEY = {
  allocationState: 'allocationState',
  addressState: 'addressState',
  supplierState: 'supplierState',
  channelSyncState: 'channelSyncState',
  reviewRequirement: 'reviewRequirement',
  drift: 'driftStatus',
} as const

export function buildWaveFilterLink(waveId: number, filter: dto.ActionCenterBucketFilterDTO): RouteLocationRaw {
  const query: Record<string, string> = {}
  const isIntake = filter.stepKey === 'intake'

  if (!isIntake) {
    for (const [field, queryKey] of Object.entries(FIELD_TO_QUERY_KEY) as Array<
      [keyof typeof FIELD_TO_QUERY_KEY, string]
    >) {
      const value = filter[field]
      if (!value) continue
      const serialized = serializeEnumMultiQuery([value])
      if (serialized !== undefined) query[queryKey] = serialized
    }
  }

  return {
    name: isIntake ? 'wave-workspace-intake' : 'wave-workspace-lines',
    params: { id: waveId },
    query,
  }
}
