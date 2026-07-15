/**
 * Wave-workspace step-key vocabulary (P2 foundations, plan section 7).
 *
 * THREE distinct step-key sets coexist in this codebase and must never be
 * assumed interchangeable:
 *
 * 1. `RouteStepKey` — the P2 route path segments (also the `WorkspaceNav`
 *    item keys). `''` is the overview tab (route name `wave-workspace`).
 * 2. `GuardKey` — the strings `ValidateStepAccess` (controller_wave.go:546-
 *    558) accepts. Only 5 values; anything else throws "unknown step key".
 * 3. `LegacyStepStateKey` — `WaveStepStateDTO.stepKey`'s 8-value set, as
 *    produced by `buildWorkspaceStepStates` (wave_overview_query_usecase.go
 *    :744-755). This is a legacy/back-compat shape from before the P2 route
 *    model existed — it does NOT line up 1:1 with `RouteStepKey`.
 *
 * `STEPSTATE_TO_ROUTE` intentionally collapses `membership_allocation` AND
 * `demand_mapping` onto the single `allocation` route (both precede the
 * "review the resulting lines" step), and `readiness` has NO legacy
 * counterpart (it's a new P2 route with no matching `WaveStepStateDTO`
 * entry) — a nav item for `readiness` simply gets no step-state-derived
 * signal. `wave_overview` maps to `''` (the overview route stays outside
 * `NAV_GROUPS`, rendered as the default tab above the grouped rail).
 */
import type { StatusTone } from '@/shared/i18n/glossary'

/** The 8 P2 route step segments, `''` = overview (route name `wave-workspace`). */
export type RouteStepKey = '' | 'intake' | 'allocation' | 'lines' | 'readiness' | 'factory' | 'shipments' | 'closure'

export const ROUTE_STEP_KEYS: RouteStepKey[] = [
  '',
  'intake',
  'allocation',
  'lines',
  'readiness',
  'factory',
  'shipments',
  'closure',
]

/** The 5 keys `ValidateStepAccess` accepts (controller_wave.go:546-558). */
export type GuardKey = 'allocation' | 'review' | 'execution' | 'shipment' | 'sync'

/**
 * Route -> guard key. `null` means the step has no guard (always
 * accessible): `intake` is the first step, `''` (overview) is never gated.
 */
export const ROUTE_TO_GUARD: Record<RouteStepKey, GuardKey | null> = {
  '': null,
  intake: null,
  allocation: 'allocation',
  lines: 'review',
  readiness: 'review',
  factory: 'execution',
  shipments: 'shipment',
  closure: 'sync',
}

/** `WaveStepStateDTO.stepKey`'s legacy 8-value set (wave_overview_query_usecase.go:744-755). */
export type LegacyStepStateKey =
  | 'demand_intake'
  | 'membership_allocation'
  | 'demand_mapping'
  | 'wave_overview'
  | 'adjustment_review'
  | 'supplier_execution'
  | 'shipment_intake'
  | 'channel_sync'

/** Legacy `WaveStepStateDTO.stepKey` -> canonical P2 route step key. */
export const STEPSTATE_TO_ROUTE: Record<LegacyStepStateKey, RouteStepKey> = {
  demand_intake: 'intake',
  membership_allocation: 'allocation',
  demand_mapping: 'allocation',
  wave_overview: '',
  adjustment_review: 'lines',
  supplier_execution: 'factory',
  shipment_intake: 'shipments',
  channel_sync: 'closure',
}

/** i18n label key for a route step's `WorkspaceNav` item / `GuidanceCard` CTA. */
export const STEP_LABEL_KEY: Record<RouteStepKey, string> = {
  '': 'waveWorkspace.steps.overview',
  intake: 'waveWorkspace.steps.intake',
  allocation: 'waveWorkspace.steps.allocation',
  lines: 'waveWorkspace.steps.lines',
  readiness: 'waveWorkspace.steps.readiness',
  factory: 'waveWorkspace.steps.factory',
  shipments: 'waveWorkspace.steps.shipments',
  closure: 'waveWorkspace.steps.closure',
}

/** One `WorkspaceNav` group definition: which route steps it clusters. */
export interface WaveWorkspaceNavGroupDef {
  key: string
  labelKey: string
  steps: RouteStepKey[]
}

/**
 * 准备(prepare) / 审查(review) / 执行(execute) grouping (plan 3.3's step
 * list). `''` (overview) is deliberately absent — it sits above the groups
 * as the default tab, not inside `WorkspaceNav`'s grouped rail.
 */
export const NAV_GROUPS: WaveWorkspaceNavGroupDef[] = [
  { key: 'prepare', labelKey: 'waveWorkspace.nav.groups.prepare', steps: ['intake', 'allocation'] },
  { key: 'review', labelKey: 'waveWorkspace.nav.groups.review', steps: ['lines', 'readiness'] },
  { key: 'execute', labelKey: 'waveWorkspace.nav.groups.execute', steps: ['factory', 'shipments', 'closure'] },
]

/** Vue Router route `name` for a given route step key (matches app/router/index.ts). */
export function routeNameForStep(step: RouteStepKey): string {
  return step === '' ? 'wave-workspace' : `wave-workspace-${step}`
}

/** Legacy `WaveStepStateDTO.stepKey` -> canonical route step key. `undefined` for an unrecognized legacy key. */
export function routeForStep(legacyStepKey: string): RouteStepKey | undefined {
  return STEPSTATE_TO_ROUTE[legacyStepKey as LegacyStepStateKey]
}

/** Route step key -> `ValidateStepAccess` guard key. `null` = no guard (always accessible). */
export function guardKeyForRoute(routeKey: RouteStepKey): GuardKey | null {
  return ROUTE_TO_GUARD[routeKey] ?? null
}

/**
 * `WaveStepStateDTO.status` -> `WorkspaceNav` dot tone. NOTE: `status` is a
 * static per-step label hardcoded by `buildWorkspaceStepStates` (always
 * `"active"` for demand_intake, `"current"` for wave_overview, `"available"`
 * for every other step) — it carries no dynamic blocked/warning signal by
 * itself. `WaveWorkspaceShell` layers `snapshot.guidance` (real, dynamic
 * severity signals) on top of this baseline tone; see its inline comment.
 */
export function toneForStepStatus(status: string): StatusTone | undefined {
  switch (status) {
    case 'current':
      return 'info'
    case 'active':
      return 'progress'
    default:
      return undefined
  }
}
