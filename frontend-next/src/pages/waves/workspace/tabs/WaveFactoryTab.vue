<script setup lang="ts">
/**
 * WaveFactoryTab — 工厂订单 tab (plan 3.3.4 first bullet, P5 unit=
 * factory-orders). Route `wave-workspace-factory`. Orchestrates the wave's
 * supplier-order list (`useFactoryOrders`) and lays out one independently-
 * operable `SupplierOrderCard` per order (v-for, plan's "多订单并列卡片，
 * 各自独立操作").
 *
 * The undo-boundary notice (`ctx.undoBoundaryCrossed` /
 * `waveWorkspace.header.undoBoundaryNotice`) is already rendered
 * unconditionally by `WaveWorkspaceHeader.vue` above every tab once any
 * supplier order exists for the wave — this component does not recompute
 * or duplicate it, only relies on it staying visible.
 *
 * Regenerating when orders already exist goes through a confirm step
 * (`regenerateConfirm.*`) — the backend rebuild (`DeleteDraftsByWave`) only
 * ever wipes still-`draft` orders and re-creates drafts for the same
 * fulfillment lines; submitted/accepted orders are untouched but the
 * operator should still be told what's about to happen (old tree gave zero
 * warning beyond a generic banner — see `WaveExportStep.vue:102-104`).
 *
 * NO advisory gate hint here (deliberately, unlike `WaveTabPlaceholder.vue`'s
 * generic pattern): `guardKeyForRoute('factory')` resolves to `'execution'`,
 * whose actual server check (`GuardExecutionRequiresReview`,
 * workspace_guard_service.go:59-69, wired at controller_wave.go:551-552) is
 * "at least one supplier order already exists for this wave" — i.e. it is
 * ALWAYS reported blocked on this tab's normal, expected first-ever visit
 * (zero orders is the starting state before the operator's first "Generate"
 * click). Wiring that same check into a self-referential "you must generate
 * a factory order before you can generate a factory order" banner here would
 * be permanently wrong, not merely advisory — flagged as a foundations-layer
 * guard-semantics inconsistency (see deviations) rather than worked around
 * in this tab.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useFactoryOrders } from './factory/useFactoryOrders'
import SupplierOrderCard from './factory/SupplierOrderCard.vue'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

// Injected purely so this component fails loudly (per its own contract) if
// ever mounted outside `WaveWorkspaceShell` — mirrors `WaveAllocationTab.vue`'s
// dual-injection pattern alongside `useFactoryOrders()`'s own internal
// `useWaveWorkspaceContext()` call.
useWaveWorkspaceContext()

const factory = useFactoryOrders()

const generating = ref(false)
const regenerateConfirmVisible = ref(false)

function handleGenerateClick(): void {
  if (factory.orders.value.length > 0) {
    regenerateConfirmVisible.value = true
    return
  }
  void runGenerate()
}

async function runGenerate(): Promise<void> {
  generating.value = true
  try {
    await factory.regenerate()
    feedback.success(t('waveWorkspace.factory.generate'))
    regenerateConfirmVisible.value = false
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    generating.value = false
  }
}

async function handleCardChanged(): Promise<void> {
  await factory.loadAll()
}

const hasOrders = computed(() => factory.orders.value.length > 0)
</script>

<template>
  <div class="wave-factory-tab">
    <PageHeader :title="t('waveWorkspace.factory.title')" :description="t('waveWorkspace.factory.subtitle')">
      <template #actions>
        <NButton type="primary" :loading="generating" @click="handleGenerateClick">
          {{ t('waveWorkspace.factory.generate') }}
        </NButton>
      </template>
    </PageHeader>

    <EmptyState
      v-if="factory.ready.value && !hasOrders"
      :title="t('waveWorkspace.factory.empty.title')"
      :description="t('waveWorkspace.factory.empty.description')"
    >
      <NButton type="primary" :loading="generating" @click="handleGenerateClick">
        {{ t('waveWorkspace.factory.generate') }}
      </NButton>
    </EmptyState>

    <div v-else class="wave-factory-tab__cards">
      <SupplierOrderCard
        v-for="order in factory.orders.value"
        :key="order.id"
        :order="order"
        :lines="factory.linesByOrder.value.get(order.id) ?? []"
        @changed="handleCardChanged"
      />
    </div>

    <NModal
      :show="regenerateConfirmVisible"
      preset="dialog"
      type="warning"
      :title="t('waveWorkspace.factory.regenerateConfirm.title')"
      :content="t('waveWorkspace.factory.regenerateConfirm.content')"
      :positive-text="t('waveWorkspace.factory.regenerateConfirm.confirm')"
      :negative-text="t('waveWorkspace.factory.regenerateConfirm.cancel')"
      :loading="generating"
      @positive-click="runGenerate"
      @negative-click="regenerateConfirmVisible = false"
      @update:show="(value: boolean) => (regenerateConfirmVisible = value)"
    />
  </div>
</template>

<style scoped>
.wave-factory-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-factory-tab__cards {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
</style>
