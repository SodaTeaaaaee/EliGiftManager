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
 * `demandKind` and `routingDisposition` DO reuse the existing glossary
 * dimensions through FilterBar's enum-multi field type, and
 * `useInboxGrid.ts` passes the `string[]` selections straight through to
 * the bridge's `demandKinds` / `routingDispositions` array params (empty
 * array means "no filter"). `demandKind` is also the source of truth for
 * the page's business-surface segmented control
 * (`businessSurface.ts` folds it into the all/membership/retail 3-way
 * state).
 */
import type { FilterSchema } from '@/shared/ui/filter-bar'

export const INBOX_GRID_FILTER_SCHEMA = [
  { key: 'demandKind', type: 'enum-multi', dimension: 'demandKind' },
  { key: 'routingDisposition', type: 'enum-multi', dimension: 'routingDisposition' },
] as const satisfies FilterSchema
