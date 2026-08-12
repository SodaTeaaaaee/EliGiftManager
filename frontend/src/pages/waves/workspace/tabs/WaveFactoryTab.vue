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
 * (`regenerateConfirm.*`) — the backend profile-scoped rebuild only replaces
 * still-`draft` orders for the selected profile and re-creates drafts for the same
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
import { NButton, NModal, NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { EmptyState } from '@/shared/ui/empty-state'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useFactoryOrders } from './factory/useFactoryOrders'
import { factoryGenerationDecision } from './factory/factoryProfileSelection'
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
const profileConflictVisible = ref(false)
const conflictingExistingProfileId = ref<number | null>(null)
const profilePickerVisible = ref(false)
const selectedFactoryProfileId = ref<number | null>(null)

const factoryProfileOptions = computed<SelectOption[]>(() => factory.factoryProfiles.value.map((profile) => ({
  label: `${profile.profileKey} · ${profile.factorySupplierPlatform}`,
  value: profile.id,
})))

function handleGenerateClick(): void {
  selectedFactoryProfileId.value = factory.factoryProfiles.value.length === 1
    ? factory.factoryProfiles.value[0].id
    : null
  profilePickerVisible.value = true
}

function confirmFactoryProfile(): void {
  if (selectedFactoryProfileId.value == null) return
  const selected = factory.factoryProfiles.value.find((profile) => profile.id === selectedFactoryProfileId.value)
  if (!selected) return
  profilePickerVisible.value = false
  const decision = factoryGenerationDecision(selected, factory.orders.value)
  if (decision.kind === 'profile_conflict') {
    conflictingExistingProfileId.value = decision.existingProfileId
    profileConflictVisible.value = true
    return
  }
  if (decision.kind === 'rebuild_profile') {
    regenerateConfirmVisible.value = true
    return
  }
  void runGenerate()
}

async function runGenerate(): Promise<void> {
  generating.value = true
  try {
    if (selectedFactoryProfileId.value == null) return
    await factory.regenerate(selectedFactoryProfileId.value)
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

function useExistingFactoryProfile(): void {
  if (conflictingExistingProfileId.value == null) return
  selectedFactoryProfileId.value = conflictingExistingProfileId.value
  profileConflictVisible.value = false
  regenerateConfirmVisible.value = true
}
</script>

<template>
  <div class="wave-factory-tab">
    <PageHeader :title="t('waveWorkspace.factory.title')" :description="t('waveWorkspace.factory.subtitle')">
      <template #actions>
        <NButton type="primary" :loading="generating" :disabled="factory.loading.value || !!factory.error.value" @click="handleGenerateClick">
          {{ t('waveWorkspace.factory.generate') }}
        </NButton>
      </template>
    </PageHeader>

    <ErrorBanner
      v-if="factory.error.value"
      :message="t('waveWorkspace.factory.loadFailed')"
      :detail="factory.error.value"
      @retry="factory.loadAll"
    />

    <div v-if="factory.loading.value && !factory.ready.value" class="wave-factory-tab__loading">
      <NSpin size="large" />
    </div>

    <EmptyState
      v-else-if="factory.ready.value && !factory.error.value && !hasOrders"
      :title="t('waveWorkspace.factory.empty.title')"
      :description="t('waveWorkspace.factory.empty.description')"
    >
      <NButton type="primary" :loading="generating" :disabled="factory.loading.value || !!factory.error.value" @click="handleGenerateClick">
        {{ t('waveWorkspace.factory.generate') }}
      </NButton>
    </EmptyState>

    <div v-else-if="hasOrders" class="wave-factory-tab__cards">
      <SupplierOrderCard
        v-for="order in factory.orders.value"
        :key="order.id"
        :order="order"
        :lines="factory.linesByOrder.value.get(order.id) ?? []"
        @changed="handleCardChanged"
      />
    </div>

    <NModal
      :show="profilePickerVisible"
      preset="card"
      :title="t('waveWorkspace.factory.profilePicker.title')"
      :style="{ width: 'min(520px, 94vw)' }"
      @update:show="(value: boolean) => (profilePickerVisible = value)"
    >
      <p class="wave-factory-tab__profile-hint">{{ t('waveWorkspace.factory.profilePicker.description') }}</p>
      <NSelect
        v-model:value="selectedFactoryProfileId"
        :options="factoryProfileOptions"
        :placeholder="t('waveWorkspace.factory.profilePicker.placeholder')"
      />
      <p v-if="factoryProfileOptions.length === 0" class="wave-factory-tab__profile-error">
        {{ t('waveWorkspace.factory.profilePicker.empty') }}
      </p>
      <template #footer>
        <div class="wave-factory-tab__modal-actions">
          <NButton @click="profilePickerVisible = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :disabled="selectedFactoryProfileId == null" @click="confirmFactoryProfile">
            {{ t('common.confirm') }}
          </NButton>
        </div>
      </template>
    </NModal>

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

    <NModal
      :show="profileConflictVisible"
      preset="dialog"
      type="warning"
      :title="t('waveWorkspace.factory.profileConflict.title')"
      :content="t('waveWorkspace.factory.profileConflict.content')"
      :positive-text="conflictingExistingProfileId == null ? t('common.close') : t('waveWorkspace.factory.profileConflict.rebuildExisting')"
      :negative-text="t('common.cancel')"
      @positive-click="conflictingExistingProfileId == null ? (profileConflictVisible = false) : useExistingFactoryProfile()"
      @negative-click="profileConflictVisible = false"
      @update:show="(value: boolean) => (profileConflictVisible = value)"
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

.wave-factory-tab__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-8);
}

.wave-factory-tab__profile-hint {
  margin: 0 0 var(--space-3);
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

.wave-factory-tab__profile-error {
  color: var(--status-error-fg);
  font-size: var(--font-size-xs);
}

.wave-factory-tab__modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
