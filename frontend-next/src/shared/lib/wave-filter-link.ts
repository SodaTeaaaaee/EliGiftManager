/**
 * `buildWaveFilterLink` — the SINGLE source for the task-center-bucket ->
 * wave-workspace deep-link query mapping. Both the task center (P1) and the
 * fulfillment grid (P3) import this instead of hand-inlining the mapping, so
 * the singular ActionCenterBucketFilterDTO fields and the plural
 * WaveFulfillmentFilterInput query keys the grid's `useUrlFilters` schema
 * will use can never drift apart.
 *
 * Encoding matches `shared/ui/filter-bar/useUrlFilters.ts`'s
 * `serializeEnumMultiQuery`: a single filter value becomes a one-element
 * array, comma-joined. Empty/absent fields are omitted entirely (never an
 * empty-string query param).
 */
import type { RouteLocationRaw } from 'vue-router'
import type { dto } from '../../../wailsjs/go/models'
import { serializeEnumMultiQuery } from '@/shared/ui/filter-bar/useUrlFilters'

/** Singular DTO field -> plural query-param name (matches the future WaveFulfillmentFilterInput field names). */
const FIELD_TO_QUERY_KEY = {
  allocationState: 'allocationStates',
  addressState: 'addressStates',
  supplierState: 'supplierStates',
  channelSyncState: 'channelSyncStates',
  reviewRequirement: 'reviewRequirements',
  drift: 'driftStatuses',
} as const

/**
 * Builds a `RouteLocationRaw` targeting the wave workspace, carrying the
 * bucket's pre-filter as URL query params. Only non-empty filter fields are
 * included; `stepKey` (not an enum-multi dimension) is passed through as a
 * plain string query value.
 */
export function buildWaveFilterLink(
  waveId: number,
  filter: dto.ActionCenterBucketFilterDTO,
): RouteLocationRaw {
  const query: Record<string, string> = {}

  for (const [field, queryKey] of Object.entries(FIELD_TO_QUERY_KEY) as Array<
    [keyof typeof FIELD_TO_QUERY_KEY, string]
  >) {
    const value = filter[field]
    if (!value) continue
    const serialized = serializeEnumMultiQuery([value])
    if (serialized !== undefined) query[queryKey] = serialized
  }

  if (filter.stepKey) {
    query.stepKey = filter.stepKey
  }

  return {
    name: 'wave-workspace',
    params: { id: waveId },
    query,
  }
}
