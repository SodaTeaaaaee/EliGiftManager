/**
 * useWaveWorkspace — the SINGLE source of truth for one wave workspace
 * (plan section 7 / P2 foundations). `WaveWorkspaceShell.vue` is the only
 * caller of `useWaveWorkspace()` itself; it `provide()`s the resulting
 * context and every descendant (header, nav, overview tab, step
 * placeholders) reads it via `useWaveWorkspaceContext()` — nobody else
 * calls `getWaveWorkspaceSnapshot` directly, so there is exactly one
 * in-flight snapshot per mounted workspace.
 *
 * Undo/redo's "撤销不丢 UI 状态" requirement is implemented by `refresh()`:
 * it re-fetches into the SAME `snapshot` ref (no ref replacement of the
 * whole object identity's containing structure — Vue's reactivity diffs
 * through it), so no caller ever needs a `:key` bump or route remount.
 */
import { computed, inject, provide, ref, watch, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { getActionCenterSummary, getWaveWorkspaceSnapshot } from '@/shared/api/bridge'
import type { DriftSummaryValue } from '@/shared/i18n/glossary'
import type { dto } from '../../../../wailsjs/go/models'

export interface WaveWorkspaceContext {
  /** The wave id this context was built for (reactive — follows the route param). */
  waveId: ComputedRef<number>
  /** The full snapshot. `null` while loading or after a hard failure. */
  snapshot: Ref<dto.WaveWorkspaceSnapshotDTO | null>
  /** True only during the very first load and while `refresh()` is in flight. */
  loading: Ref<boolean>
  /** True after `getWaveWorkspaceSnapshot` throws — `snapshot` is `null` in this state. */
  error: Ref<boolean>
  /**
   * Re-fetch `getWaveWorkspaceSnapshot` (+ `getActionCenterSummary`) into
   * the SAME reactive refs. This is what undo/redo call after a successful
   * command — never remount/re-key the route.
   */
  refresh(): Promise<void>
  /** The 4-branch projected drift summary (plan 3.3.1) — feeds `<StatusBadge dimension="driftSummary">`. */
  driftSummaryValue: ComputedRef<DriftSummaryValue>
  /** This wave's action-center buckets. `[]` when the wave has zero blocked buckets (not an error). */
  sixBuckets: ComputedRef<dto.ActionCenterWaveBucketDTO[]>
  /** `overview.supplierOrderCount > 0` — drives the passive "earlier actions may not be undoable" notice. */
  undoBoundaryCrossed: ComputedRef<boolean>
}

export const WAVE_WORKSPACE_INJECTION_KEY: InjectionKey<WaveWorkspaceContext> = Symbol('eligiftmanager:wave-workspace')

/**
 * Builds the context. Call exactly once, from `WaveWorkspaceShell.vue`'s
 * `setup()`, then `provideWaveWorkspaceContext(ctx)`.
 *
 * `waveId` is a getter (not a plain number) so the composable can `watch`
 * it internally — passing `() => Number(route.params.id)` means changing
 * the route param automatically re-fetches, with no extra watcher needed
 * in the caller.
 */
export function useWaveWorkspace(waveId: () => number): WaveWorkspaceContext {
  const waveIdRef = computed(waveId)

  const snapshot = ref<dto.WaveWorkspaceSnapshotDTO | null>(null) as Ref<dto.WaveWorkspaceSnapshotDTO | null>
  const loading = ref(true)
  const error = ref(false)
  const actionCenterSummary = ref<dto.ActionCenterSummaryDTO | null>(null) as Ref<dto.ActionCenterSummaryDTO | null>

  async function loadSnapshot(id: number): Promise<void> {
    try {
      snapshot.value = await getWaveWorkspaceSnapshot(id)
      error.value = false
    } catch {
      // Hard-fail bridge call (per bridge.ts contract) — mirrors
      // WaveWorkspacePlaceholderPage.vue's loadFailed pattern: never throw
      // into the template, surface via `error` instead.
      snapshot.value = null
      error.value = true
    }
  }

  async function loadActionCenterSummary(): Promise<void> {
    try {
      actionCenterSummary.value = await getActionCenterSummary()
    } catch {
      // Soft-fail by contract — an empty/missing summary just means the
      // six-bucket row renders empty, not an error state for the page.
      actionCenterSummary.value = null
    }
  }

  async function refresh(): Promise<void> {
    loading.value = true
    await Promise.all([loadSnapshot(waveIdRef.value), loadActionCenterSummary()])
    loading.value = false
  }

  // Re-fetch whenever the wave id changes (covers both the initial mount
  // and later route-param navigation, e.g. jumping from one wave to
  // another via a deep link while the shell stays mounted).
  watch(waveIdRef, () => void refresh(), { immediate: true })

  const driftSummaryValue = computed<DriftSummaryValue>(() => {
    const overview = snapshot.value?.overview
    if (!overview) return 'in_sync'
    if (overview.hasRequiredReviewBasis) return 'drifted_required'
    if (overview.basisDriftSignals?.some((signal) => signal.reviewRequirement === 'recommended')) {
      return 'drifted_recommended'
    }
    if (overview.hasDriftedBasis) return 'drifted_none'
    return 'in_sync'
  })

  const sixBuckets = computed<dto.ActionCenterWaveBucketDTO[]>(() => {
    const wave = actionCenterSummary.value?.waves.find((entry) => entry.waveId === waveIdRef.value)
    return wave?.buckets ?? []
  })

  const undoBoundaryCrossed = computed<boolean>(() => (snapshot.value?.overview.supplierOrderCount ?? 0) > 0)

  return {
    waveId: waveIdRef,
    snapshot,
    loading,
    error,
    refresh,
    driftSummaryValue,
    sixBuckets,
    undoBoundaryCrossed,
  }
}

/** Provides a `useWaveWorkspace()` result to the component subtree. Call once, from `WaveWorkspaceShell.vue`. */
export function provideWaveWorkspaceContext(ctx: WaveWorkspaceContext): void {
  provide(WAVE_WORKSPACE_INJECTION_KEY, ctx)
}

/**
 * The only way descendants of `WaveWorkspaceShell` read workspace state.
 * Throws outside a provider (same "fail loudly" convention as
 * `useFeedback()`) rather than silently returning a dead context.
 */
export function useWaveWorkspaceContext(): WaveWorkspaceContext {
  const ctx = inject(WAVE_WORKSPACE_INJECTION_KEY, null)
  if (!ctx) {
    throw new Error(
      '[useWaveWorkspaceContext] no wave-workspace context provided. ' +
        'This must be called from a descendant of WaveWorkspaceShell.vue.',
    )
  }
  return ctx
}
