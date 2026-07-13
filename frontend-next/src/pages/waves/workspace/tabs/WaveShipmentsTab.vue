<script setup lang="ts">
/**
 * WaveShipmentsTab — 发货回传 tab (P5 plan §3.3.4 second bullet, route
 * `wave-workspace-shipments`). Three sub-views, switched by a plain NTabs
 * (no house wrapper exists for tabs — mirrors the "Naive UI directly" carve-
 * out already used elsewhere in this sub-area, e.g. `NModal`/`NSteps` in
 * `ImportWizard.vue`):
 *
 * 1. CSV import (`shipments/ImportWizard.vue`) — reconciliation-key column
 *    mapping, resolved client-side via `shipments/useReconciliationIndex.ts`
 *    against the wave's own supplier orders/lines, never asking the
 *    operator to supply our internal DB ids. Its result view
 *    (`shipments/ImportResultView.vue`) is a persistent, non-auto-reset
 *    state — the direct fix for the old tree's confirmed bug (see that
 *    file's doc comment).
 * 2. Manual entry (`shipments/ManualShipmentForm.vue`) — any supplier order
 *    in the wave (not `order[0]`), with shipped/remaining quantities shown
 *    per line.
 * 3. History (`shipments/ShipmentHistory.vue`) — `listShipmentsByWave` +
 *    correct/void, both honestly presented as outside the undo/redo
 *    history.
 *
 * `refreshSignal` is bumped whenever the import wizard or the manual form
 * successfully records a shipment, so the History tab reflects new rows
 * without a route/tab remount even while it stays mounted-but-hidden behind
 * naive-ui's default tab-pane display mode.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTabs, NTabPane } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { CalloutBar } from '@/shared/ui/guidance'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import ImportWizard from './shipments/ImportWizard.vue'
import ManualShipmentForm from './shipments/ManualShipmentForm.vue'
import ShipmentHistory from './shipments/ShipmentHistory.vue'

const { t } = useI18n({ useScope: 'global' })

// Injected purely so this component fails loudly (per its own contract) if
// ever mounted outside `WaveWorkspaceShell` — mirrors `WaveFactoryTab.vue`'s
// same dual-injection pattern alongside each child's own
// `useWaveWorkspaceContext()` call.
const ctx = useWaveWorkspaceContext()

const activeTab = ref<'import' | 'manual' | 'history'>('import')
const refreshSignal = ref(0)

function bumpRefreshSignal(): void {
  refreshSignal.value += 1
}
</script>

<template>
  <div class="wave-shipments-tab">
    <PageHeader :title="t('waveWorkspace.shipments.title')" :description="t('waveWorkspace.shipments.subtitle')" />

    <CalloutBar
      v-if="ctx.undoBoundaryCrossed.value"
      tone="info"
      :message="t('waveWorkspace.header.undoBoundaryNotice')"
    />

    <NTabs v-model:value="activeTab" type="line" animated>
      <NTabPane name="import" :tab="t('waveWorkspace.shipments.tabs.import')">
        <ImportWizard @imported="bumpRefreshSignal" />
      </NTabPane>
      <NTabPane name="manual" :tab="t('waveWorkspace.shipments.tabs.manual')">
        <ManualShipmentForm @created="bumpRefreshSignal" />
      </NTabPane>
      <NTabPane name="history" :tab="t('waveWorkspace.shipments.tabs.history')">
        <ShipmentHistory :refresh-signal="refreshSignal" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<style scoped>
.wave-shipments-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
</style>
