<script setup lang="ts">
/**
 * WaveWorkspaceShell — the P2 parent route component for `/waves/:id`
 * (plan section 7). Owns the ONE `useWaveWorkspace()` instance for this
 * wave and `provide()`s it to every descendant (header, nav, tabs) via
 * `provideWaveWorkspaceContext` — no descendant calls
 * `getWaveWorkspaceSnapshot` on its own.
 *
 * `<RouterView/>` below carries NO `:key` — switching tabs, and undo/redo's
 * `refresh()`, both update the SAME provided context in place, so the
 * active tab component never unmounts/remounts on either action (the
 * "撤销不丢 UI 状态" acceptance criterion).
 */
import { computed, onBeforeMount, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { EmptyState } from '@/shared/ui/empty-state'
import { WorkspaceNav, type WorkspaceNavGroupSpec, type WorkspaceNavItemSpec } from '@/shared/ui/shell'
import { useWaveWorkspace, provideWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { registerRefreshTarget } from '@/shared/lib/view-hotkeys'
import { NAV_GROUPS, STEP_LABEL_KEY, routeForStep, routeNameForStep, toneForStepStatus, type RouteStepKey } from '@/shared/lib/wave-workspace/step-keys'
import WaveWorkspaceHeader from './WaveWorkspaceHeader.vue'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()

const waveId = computed(() => Number(Array.isArray(route.params.id) ? route.params.id[0] : route.params.id))

const ctx = useWaveWorkspace(() => waveId.value)
provideWaveWorkspaceContext(ctx)

let unregisterRefresh: (() => void) | undefined
onBeforeMount(() => {
  unregisterRefresh = registerRefreshTarget(ctx.refresh)
})
onBeforeUnmount(() => unregisterRefresh?.())

// ── WorkspaceNav wiring ──
//
// Baseline tone comes from `WaveStepStateDTO.status` (a static per-step
// label, see `toneForStepStatus`'s doc comment); it is UPGRADED to
// warning/error when `snapshot.guidance` carries a signal targeting that
// route (a real, dynamic blocking/warning condition) — guidance always
// wins because it is the actually-meaningful signal.
function guidanceToneFor(step: RouteStepKey) {
  let worst: 'warning' | 'error' | undefined
  for (const g of ctx.snapshot.value?.guidance ?? []) {
    if (routeForStep(g.targetStepKey) !== step) continue
    if (g.severity === 'error') return 'error' as const
    if (g.severity === 'warning') worst = 'warning'
  }
  return worst
}

function stepStateFor(step: RouteStepKey) {
  // stepStates can be JSON null if the Go side ever emits a nil slice.
  return ctx.snapshot.value?.stepStates?.find((s) => routeForStep(s.stepKey) === step)
}

function buildNavItem(step: RouteStepKey): WorkspaceNavItemSpec {
  const state = stepStateFor(step)
  const tone = guidanceToneFor(step) ?? (state ? toneForStepStatus(state.status) : undefined)
  return {
    key: step || 'overview',
    labelKey: STEP_LABEL_KEY[step],
    to: { name: routeNameForStep(step), params: { id: waveId.value } },
    tone,
    count: state && state.primaryCount > 0 ? state.primaryCount : undefined,
  }
}

// The overview tab is its own ungrouped cluster (no `labelKey` -> no
// visible group heading), rendered above 准备/审查/执行 — WorkspaceNav's
// `WorkspaceNavGroupSpec.labelKey` being optional is exactly this hook.
const navGroups = computed<WorkspaceNavGroupSpec[]>(() => [
  { key: 'overview', items: [buildNavItem('')] },
  ...NAV_GROUPS.map((group) => ({
    key: group.key,
    labelKey: group.labelKey,
    items: group.steps.map(buildNavItem),
  })),
])
</script>

<template>
  <div class="wave-workspace-shell">
    <EmptyState v-if="ctx.loading.value && !ctx.snapshot.value" :title="t('waveWorkspace.shell.loading')" />
    <EmptyState v-else-if="ctx.error.value || !ctx.snapshot.value" :title="t('waveWorkspace.shell.notFound')" />
    <template v-else>
      <WaveWorkspaceHeader />
      <div class="wave-workspace-shell__body">
        <WorkspaceNav :groups="navGroups" />
        <div class="wave-workspace-shell__content">
          <RouterView />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.wave-workspace-shell {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  height: 100%;
  min-height: 0;
}

.wave-workspace-shell__body {
  display: flex;
  align-items: stretch;
  flex: 1 1 auto;
  min-height: 0;
  gap: var(--space-4);
}

.wave-workspace-shell__content {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
}
</style>
