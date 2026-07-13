/**
 * inbox-grid/filter-schema — the `FilterSchema` for the demand-inbox grid
 * (plan P4, `pages/inbox/InboxPage.vue` + the wave-scoped
 * `pages/waves/workspace/tabs/WaveIntakeTab.vue`).
 *
 * Per the P4 foundations contract's decision #5, the `assignment`
 * (all/assigned/unassigned) dimension is deliberately NOT part of this
 * schema — it has no status-tone semantics (a boolean-ish "is this doc
 * linked to a wave yet", not a lifecycle/health state) and does not fit
 * FilterBar's enum-multi machinery (which requires a `GlossaryDimension` +
 * `StatusTone` per value). It is driven by a bespoke 3-way toggle owned by
 * `useInboxGrid.ts` directly, wired straight to the bridge's `assignment`
 * filter param — never through `useGlossary()` / `glossaryTables` /
 * `FilterBar`.
 *
 * `demandKind` DOES reuse the existing `demandKind` glossary dimension (2
 * values: `membership_entitlement` / `retail_order`) through FilterBar's
 * enum-multi field type. The backend's `DemandInboxFilterInput.demandKind`
 * is a single string, not a list — `useInboxGrid.ts`'s
 * `resolveDemandKindParam()` folds the `string[]` selection down to that
 * single param: 0 or 2 selected values both mean "no filter" (all/nothing
 * selected are equivalent when there are only 2 possible values), and
 * exactly 1 selected value is passed straight through.
 */
import type { FilterSchema } from '@/shared/ui/filter-bar'

export const INBOX_GRID_FILTER_SCHEMA = [{ key: 'demandKind', type: 'enum-multi', dimension: 'demandKind' }] as const satisfies FilterSchema
