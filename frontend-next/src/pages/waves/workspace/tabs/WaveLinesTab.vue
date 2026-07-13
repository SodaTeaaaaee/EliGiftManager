<script setup lang="ts">
/**
 * WaveLinesTab — the fulfillment-lines grid tab (plan 3.3.2/3.3.3, P3
 * Assembly). Mounted at both `wave-workspace-lines` (the regular "review the
 * lines" step) and `wave-workspace-readiness` (the same grid, pre-filtered to
 * the `'blocked'` saved view, plus a minimal readiness->factory gate footer
 * below the grid) — see `app/router/index.ts`.
 *
 * Wires together the P3 grid core (`useFulfillmentGrid` + `filter-schema` +
 * `columns`) with the P3 batch UI (`BatchActionBar`) and the row detail panel
 * (`RowDetailDrawer`). This component owns none of that logic itself — it
 * only composes the pieces and reacts to their `'done'`/`'changed'` emits by
 * calling `mutationDone()`, which refreshes both this grid's current page and
 * the injected workspace shell (`useWaveWorkspaceContext().refresh()`), all
 * in place (no `:key` bump, no route remount).
 */
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { FilterBar, SavedViews } from '@/shared/ui/filter-bar'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { CalloutBar } from '@/shared/ui/guidance'
import { validateStepAccess } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { guardKeyForRoute, routeNameForStep, type GuardKey } from '@/shared/lib/wave-workspace/step-keys'
import { useFulfillmentGrid, type FulfillmentGridRow } from './fulfillment-grid/useFulfillmentGrid'
import { buildFulfillmentGridPresets } from './fulfillment-grid/filter-schema'
import { buildFulfillmentColumns } from './fulfillment-grid/columns'
import BatchActionBar from './fulfillment-grid/BatchActionBar.vue'
import RowDetailDrawer from './fulfillment-grid/RowDetailDrawer.vue'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()
const router = useRouter()

// `useWaveWorkspaceContext()` is injected purely so this component fails
// loudly (per its own contract) if ever mounted outside `WaveWorkspaceShell`
// — `useFulfillmentGrid` already injects it internally for `waveId`/`refresh`.
useWaveWorkspaceContext()

const isReadinessMode = computed(() => route.name === 'wave-workspace-readiness')

// `initialPreset` is read once at composable-creation time (matches
// `useFulfillmentGrid`'s own "pre-applies on mount" contract) — a later
// in-place route-name change (sibling route reusing this component instance)
// does not retroactively re-apply the preset, which is the desired behavior:
// only the very first mount of the readiness route should default to
// `'blocked'`, never overriding filters the operator has since changed.
const {
  filters,
  waveId,
  rows,
  loading,
  selectedKeys,
  selectedRows,
  page,
  pageSize,
  totalCount,
  onPageChange,
  mutationDone,
} = useFulfillmentGrid({ initialPreset: route.name === 'wave-workspace-readiness' ? 'blocked' : undefined })

const columns = computed(() => createColumns(buildFulfillmentColumns(t)))
const fulfillmentPresets = computed(() => buildFulfillmentGridPresets(t))

function handleSelectedKeysChange(keys: Array<string | number>): void {
  selectedKeys.value = keys as number[]
}

// ── Row detail drawer ──

const detailRow = ref<FulfillmentGridRow | null>(null)
const showDrawer = ref(false)

function handleRowClick(row: FulfillmentGridRow): void {
  detailRow.value = row
  showDrawer.value = true
}

function handleDrawerVisibility(visible: boolean): void {
  showDrawer.value = visible
}

// ── Readiness-mode footer gate (readiness -> factory transition) ──
//
// Mirrors `WaveTabPlaceholder.vue`'s advisory-gate pattern: a bare try/catch
// around `validateStepAccess`, never a hard route lock. `'factory'`'s guard
// key is always `'execution'` (`ROUTE_TO_GUARD.factory`), resolved via
// `guardKeyForRoute` rather than a raw string literal.
const FACTORY_GUARD_KEY: GuardKey | null = guardKeyForRoute('factory')

const readinessGateBlocked = ref(true)

async function checkReadinessGate(): Promise<void> {
  if (!isReadinessMode.value) return
  if (!FACTORY_GUARD_KEY) {
    readinessGateBlocked.value = false
    return
  }
  try {
    await validateStepAccess(waveId.value, FACTORY_GUARD_KEY)
    readinessGateBlocked.value = false
  } catch {
    readinessGateBlocked.value = true
  }
}

watch([isReadinessMode, waveId], () => void checkReadinessGate(), { immediate: true })

function handleProceedToFactory(): void {
  if (readinessGateBlocked.value) return
  void router.push({ name: routeNameForStep('factory'), params: { id: waveId.value } })
}
</script>

<template>
  <div class="wave-lines-tab">
    <SavedViews :filters="filters" scope-id="fulfillment-grid" :presets="fulfillmentPresets" />

    <div class="wave-lines-tab__filters">
      <FilterBar :filters="filters" />
      <p class="wave-lines-tab__drift-caveat">{{ t('fulfillmentGrid.filters.driftCaveat') }}</p>
    </div>

    <DataGrid
      :columns="columns"
      :rows="rows"
      row-key="fulfillmentLineId"
      selectable
      :selected-keys="selectedKeys"
      :loading="loading"
      :pagination="{ server: { total: totalCount, page: page, pageSize: pageSize, onChange: onPageChange } }"
      :empty="{ title: t('fulfillmentGrid.empty.noRows'), description: t('fulfillmentGrid.empty.noRowsHint') }"
      @update:selected-keys="handleSelectedKeysChange"
      @row-click="handleRowClick"
    >
      <template #selection-toolbar>
        <BatchActionBar :selected-rows="selectedRows" :wave-id="waveId" @done="mutationDone" />
      </template>
    </DataGrid>

    <RowDetailDrawer :row="detailRow" :show="showDrawer" :wave-id="waveId" @update:show="handleDrawerVisibility" @changed="mutationDone" />

    <div v-if="isReadinessMode" class="wave-lines-tab__readiness-footer">
      <h3 class="wave-lines-tab__readiness-title">{{ t('fulfillmentGrid.readiness.footerTitle') }}</h3>
      <CalloutBar
        :tone="readinessGateBlocked ? 'warning' : 'success'"
        :message="t(readinessGateBlocked ? 'fulfillmentGrid.readiness.blockedReason' : 'fulfillmentGrid.readiness.ready')"
      />
      <NButton type="primary" :disabled="readinessGateBlocked" @click="handleProceedToFactory">
        {{ t('fulfillmentGrid.readiness.proceedToFactory') }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.wave-lines-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-lines-tab__filters {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-lines-tab__drift-caveat {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-lines-tab__readiness-footer {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--color-surface);
}

.wave-lines-tab__readiness-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}
</style>
