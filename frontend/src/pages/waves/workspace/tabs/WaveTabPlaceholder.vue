<script setup lang="ts">
/**
 * WaveTabPlaceholder — generic "this step ships in P3-P5" placeholder for
 * every non-overview workspace tab (plan section 7). Also implements the
 * ADVISORY GATE HINT (plan 3.3.3 "门禁提示准确"): on route enter it maps
 * its own route step to a `ValidateStepAccess` guard key, calls it, and —
 * if blocked — renders a non-blocking `CalloutBar` explaining why and
 * offering a CTA to the step that actually needs attention. The step is
 * always reachable (no hard lock); this is guidance, not enforcement.
 *
 * `reasonCode` is derived from the injected snapshot's overview COUNTS via
 * `reasonCodeForGuard` (never from `ValidateStepAccess`'s raw English error
 * string, which is not i18n-safe) — the guard key deterministically
 * identifies which single count is at zero, matching
 * `internal/app/workspace_guard_service.go`'s guard methods 1:1.
 */
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { CalloutBar } from '@/shared/ui/guidance'
import { EmptyState } from '@/shared/ui/empty-state'
import { validateStepAccess } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { guardKeyForRoute, routeNameForStep, type GuardKey, type RouteStepKey } from '@/shared/lib/wave-workspace/step-keys'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()
const router = useRouter()
const ctx = useWaveWorkspaceContext()

/** This component is only ever mounted at `wave-workspace-<step>` routes (never the bare overview route). */
const step = computed<RouteStepKey>(() => {
  const name = String(route.name ?? '')
  const prefix = 'wave-workspace-'
  return (name.startsWith(prefix) ? name.slice(prefix.length) : '') as RouteStepKey
})

const guardKey = computed<GuardKey | null>(() => guardKeyForRoute(step.value))

/**
 * Guard key -> the overview-count-derived guidance code. Each branch
 * mirrors exactly the count `WorkspaceGuardService`'s matching guard
 * method checks server-side (allocation: demand assignments, review:
 * fulfillment lines, execution/shipment: supplier orders, sync: shipments)
 * — never a re-parse of the guard's raw English error text.
 */
function reasonCodeForGuard(guard: GuardKey): string {
  switch (guard) {
    case 'allocation':
      return 'need_demand'
    case 'review':
      return 'need_fulfillment'
    case 'execution':
    case 'shipment':
      return 'need_supplier_order'
    case 'sync':
      return 'need_shipment'
  }
}

/** reasonCode -> the step that actually needs attention to clear the gate. */
const FIX_TARGET: Record<string, RouteStepKey> = {
  need_demand: 'intake',
  need_fulfillment: 'allocation',
  need_supplier_order: 'readiness',
  need_shipment: 'shipments',
}

const gateBlocked = ref(false)
const reasonCode = ref<string | null>(null)

async function checkGate(): Promise<void> {
  const guard = guardKey.value
  if (!guard) {
    gateBlocked.value = false
    reasonCode.value = null
    return
  }
  try {
    await validateStepAccess(ctx.waveId.value, guard)
    gateBlocked.value = false
    reasonCode.value = null
  } catch {
    gateBlocked.value = true
    reasonCode.value = reasonCodeForGuard(guard)
  }
}

// Sibling placeholder routes (intake/allocation/lines/...) all resolve to
// THIS SAME component instance — Vue Router reuses it across siblings, so
// `onMounted` alone would only fire once. Watch the route name (+ wave id,
// for cross-wave deep links) instead.
watch([() => route.name, () => ctx.waveId.value], () => void checkGate(), { immediate: true })

function goFix(): void {
  const target = reasonCode.value ? FIX_TARGET[reasonCode.value] : undefined
  if (!target) return
  void router.push({ name: routeNameForStep(target), params: { id: ctx.waveId.value } })
}
</script>

<template>
  <div class="wave-tab-placeholder">
    <CalloutBar
      v-if="gateBlocked && reasonCode"
      tone="warning"
      :message="t('waveWorkspace.gateHint.' + reasonCode)"
      :action-label="t('waveWorkspace.gateHint.goFix')"
      @action="goFix"
    />
    <EmptyState :title="t('waveWorkspace.steps.' + (step || 'overview'))" :description="t('waveWorkspace.placeholder.description')" />
  </div>
</template>

<style scoped>
.wave-tab-placeholder {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
</style>
