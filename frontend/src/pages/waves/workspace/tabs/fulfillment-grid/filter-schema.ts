/**
 * fulfillment-grid/filter-schema — the `FilterSchema` for the fulfillment
 * lines grid (plan 3.3.2, P3 grid core) plus the 4 built-in saved-view
 * presets (阻塞项/可提交工厂/回填失败/已调整).
 *
 * Schema field keys deliberately mirror `dto.WaveFulfillmentFilterInput`'s
 * dimension names singularized (`allocationState` <-> `allocationStates`,
 * etc.) — `useFulfillmentGrid.ts` pluralizes them back when building the
 * bridge call input. `driftStatus` maps to the `basisDriftStatus` glossary
 * dimension (backend field `driftStatuses` / row field `basisDriftStatus`);
 * the key is singular/generic ("driftStatus") because the FilterBar UI and
 * i18n copy (`fulfillmentGrid.filters.driftStatus`) refer to it generically,
 * while the `dimension` property pins the exact glossary table used for
 * option labels/tones.
 *
 * CAVEAT surfaced via `fulfillmentGrid.filters.driftCaveat` (rendered by the
 * FilterBar/Assembly layer, not this module): `driftStatus` (and
 * `reviewRequirement`) are stamped identically on every row for a given wave
 * (wave_overview_query_usecase.go:475-483,737-742) — filtering by either is
 * effectively all-or-nothing for the whole wave, not a per-line predicate.
 */
import type { FilterSchema, FilterSnapshot, FilterViewPreset } from '@/shared/ui/filter-bar'

/** The 4 built-in, non-deletable saved views (`SavedViews` `:presets`). */
export type FulfillmentGridPresetId = 'blocked' | 'submittable' | 'backfillFailed' | 'adjusted'

/** `useUrlFilters(FULFILLMENT_GRID_FILTER_SCHEMA)` drives the grid's filter state + URL sync. */
export const FULFILLMENT_GRID_FILTER_SCHEMA = [
  { key: 'allocationState', type: 'enum-multi', dimension: 'allocationState' },
  { key: 'addressState', type: 'enum-multi', dimension: 'addressState' },
  { key: 'supplierState', type: 'enum-multi', dimension: 'supplierState' },
  { key: 'channelSyncState', type: 'enum-multi', dimension: 'channelSyncState' },
  { key: 'reviewRequirement', type: 'enum-multi', dimension: 'reviewRequirement' },
  { key: 'driftStatus', type: 'enum-multi', dimension: 'basisDriftStatus' },
  { key: 'keyword', type: 'keyword' },
] as const satisfies FilterSchema

/**
 * Raw preset snapshots — the source of truth `useFulfillmentGrid.ts` reads
 * from directly (no i18n dependency) to apply `options.initialPreset` on
 * mount. `buildFulfillmentGridPresets()` below wraps these with resolved
 * `label`s for `<SavedViews :presets>`.
 *
 * Honest-mapping notes (backend `dto.WaveFulfillmentFilterInput` only
 * supports these 6 enum dimensions + keyword — there is no per-line
 * "was this line touched by a wave adjustment" filter field server-side):
 * - `blocked`: address issues are the single biggest per-line blocker
 *   (`addressState` missing/invalid) — chosen over a cross-dimension OR
 *   (not expressible: dimensions AND together server-side) as the most
 *   actionable single-dimension reading of "blocked".
 * - `submittable`: address resolved AND supplier order not yet submitted —
 *   exactly the brief's stated mapping.
 * - `backfillFailed`: channel sync failed — exact 1:1 mapping.
 * - `adjusted`: NO backend field identifies "line touched by a manual
 *   adjustment" (`lineReason`/`generatedBy` are not in the filter DTO). Per
 *   the brief's explicit fallback instruction, this maps to
 *   `reviewRequirement: [recommended, required]` (the closest available
 *   proxy: adjustments frequently drive a wave into needing review) —
 *   flagged in the handoff `deviations` as an approximation, not a precise
 *   "was adjusted" predicate. A future Go change (add `lineReasons` to
 *   `WaveFulfillmentFilterInput`) would make this exact.
 */
export const FULFILLMENT_GRID_PRESET_SNAPSHOTS: Record<FulfillmentGridPresetId, FilterSnapshot> = {
  blocked: { addressState: ['missing', 'invalid'] },
  submittable: { addressState: ['ready'], supplierState: ['not_submitted'] },
  backfillFailed: { channelSyncState: ['failed'] },
  adjusted: { reviewRequirement: ['recommended', 'required'] },
}

const PRESET_IDS: FulfillmentGridPresetId[] = ['blocked', 'submittable', 'backfillFailed', 'adjusted']

/**
 * Builds the `FilterViewPreset[]` for `<SavedViews :presets>`. Takes an
 * already-resolved translate function (call `useI18n()` in the consuming
 * component/composable and pass its `t` straight through — mirrors
 * `WavesPage.vue`'s column-building convention; this module itself is not a
 * component/composable and must not call `useI18n()` on its own).
 */
export function buildFulfillmentGridPresets(t: (key: string) => string): FilterViewPreset[] {
  return PRESET_IDS.map((id) => ({
    id,
    label: t(`fulfillmentGrid.savedViews.${id}`),
    snapshot: FULFILLMENT_GRID_PRESET_SNAPSHOTS[id],
  }))
}
