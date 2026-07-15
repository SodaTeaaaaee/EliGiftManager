<script setup lang="ts">
/**
 * WaveAllocationTab — the wave-workspace "需求分配" tab (P4 plan §3.3.3,
 * route `wave-workspace-allocation`). Two-in-one: (A) allocation-policy
 * rules CRUD + reconcile (membership waves) and (B) the demand→fulfillment-
 * line mapping run (retail waves). Both sections read/write through
 * `useAllocationTab()` — this component only composes it with the grid/form
 * pieces and reacts to their emits.
 *
 * Gated by whether ANY demand has been assigned to the wave yet
 * (`listAssignedDemandsByWave`): with zero assigned demand there is nothing
 * to allocate or map, so the whole rules+mapping body collapses to a single
 * `EmptyState` pointing back at the intake tab (acceptance gate #9's
 * "导入 → 规则分配" scenario starts from intake).
 */
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { EmptyState } from '@/shared/ui/empty-state'
import { CalloutBar } from '@/shared/ui/guidance'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { routeNameForStep } from '@/shared/lib/wave-workspace/step-keys'
import { useAllocationTab } from './allocation/useAllocationTab'
import { buildAllocationRuleColumns } from './allocation/rule-columns'
import RuleEditor from './allocation/RuleEditor.vue'
import MappingResultPanel from './allocation/MappingResultPanel.vue'
import BatchStockToWaveDialog from '@/pages/products/BatchStockToWaveDialog.vue'
import type { AllocationPolicyRule } from '@/entities/allocation-policy'

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()

// Injected purely so this component fails loudly (per its own contract) if
// ever mounted outside `WaveWorkspaceShell` — mirrors `WaveLinesTab.vue`'s
// same dual-injection pattern alongside `useAllocationTab()`'s own internal
// `useWaveWorkspaceContext()` call.
const ctx = useWaveWorkspaceContext()
const allocation = useAllocationTab()

const hasAssignedDemand = computed(() => allocation.assignedDemands.value.length > 0)

const columns = computed(() =>
  createColumns(
    buildAllocationRuleColumns(t, {
      productNameById: allocation.productNameById.value,
      onToggleActive: (rule, active) => void handleToggleActive(rule, active),
      onEdit: openEditRule,
      onDelete: (rule) => void handleDeleteRule(rule),
    }),
  ),
)

// ── Rule editor modal ──
const showRuleEditor = ref(false)
const editingRule = ref<AllocationPolicyRule | null>(null)

function openCreateRule(): void {
  editingRule.value = null
  showRuleEditor.value = true
}

function openEditRule(rule: AllocationPolicyRule): void {
  editingRule.value = rule
  showRuleEditor.value = true
}

async function handleRuleSaved(): Promise<void> {
  feedback.success(t(editingRule.value ? 'allocation.rules.editRule' : 'allocation.rules.addRule'))
  // RuleEditor calls the create/update bridge functions directly (matching
  // BatchAdjustDialog.vue's convention) — refresh this composable's own
  // state afterward so the rules table reflects the change.
  await allocation.loadAll()
}

async function handleToggleActive(rule: AllocationPolicyRule, active: boolean): Promise<void> {
  try {
    await allocation.setRuleActive(rule, active)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

async function handleDeleteRule(rule: AllocationPolicyRule): Promise<void> {
  try {
    await allocation.removeRule(rule.id)
    feedback.success(t('allocation.rules.deleteRule'))
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

// ── Generate participants ──
async function handleGenerateParticipants(): Promise<void> {
  try {
    const created = await allocation.runGenerateParticipants()
    feedback.success(t('allocation.participants.generateDone', { count: created }))
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

// ── Reconcile ──
const reconcileSummaryMessage = computed(() => {
  const result = allocation.lastReconcileResult.value
  if (!result) return null
  return t('allocation.rules.reconcileSummary', {
    created: result.created,
    deleted: result.deleted,
    replayedCount: result.replayedCount,
  })
})

async function handleReconcile(): Promise<void> {
  try {
    const result = await allocation.runReconcile()
    if (result.failures.length > 0) {
      feedback.error(t('allocation.rules.reconcileFailures', { count: result.failures.length }))
    } else {
      feedback.success(t('allocation.rules.reconcileDone'))
    }
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

// ── Mapping ──
async function handleRunMapping(): Promise<void> {
  try {
    await allocation.runMapping()
    feedback.success(t('allocation.mapping.runDone'))
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

// ── Summary strip ──
const rulesCount = computed(() => allocation.rules.value.length)
// A general "how many fulfillment lines currently exist in this wave"
// proxy (the workspace overview's own count) — the mapping panel below
// shows the precise created/blocked counts for the LAST run specifically.
const mappedLinesCount = computed(() => ctx.snapshot.value?.overview.fulfillmentCount ?? 0)
// blockedLines can be JSON null when Go returns a nil slice — chain past it.
const blockedLinesCount = computed(() => allocation.lastMappingResult.value?.blockedLines?.length ?? 0)

function goToIntake(): void {
  void router.push({ name: routeNameForStep('intake'), params: { id: allocation.waveId.value } })
}

// ── Reverse entry: "从主档挑选商品" (batch-stock, pre-scoped to this wave) ──
const showPickFromMaster = ref(false)

function openPickFromMaster(): void {
  showPickFromMaster.value = true
}

async function handlePickFromMasterSuccess(): Promise<void> {
  showPickFromMaster.value = false
  await allocation.loadAll()
}
</script>

<template>
  <div class="wave-allocation-tab">
    <SectionCard :title="t('allocation.summary.title')">
      <div class="wave-allocation-tab__summary">
        <StatCard :label="t('allocation.summary.rulesCount')" :value="String(rulesCount)" />
        <StatCard :label="t('allocation.summary.mappedLines')" :value="String(mappedLinesCount)" tone="success" />
        <StatCard
          :label="t('allocation.summary.blockedLines')"
          :value="String(blockedLinesCount)"
          :tone="blockedLinesCount > 0 ? 'warning' : 'neutral'"
        />
      </div>
    </SectionCard>

    <EmptyState v-if="allocation.ready.value && !hasAssignedDemand" :title="t('allocation.mapping.noAssignedDemand')">
      <NButton type="primary" @click="goToIntake">{{ t('waveWorkspace.steps.intake') }}</NButton>
    </EmptyState>

    <template v-else>
      <SectionCard :title="t('allocation.participants.title')">
        <template #actions>
          <NButton
            size="small"
            type="primary"
            :loading="allocation.generatingParticipants.value"
            @click="handleGenerateParticipants"
          >
            {{ t('allocation.participants.action') }}
          </NButton>
        </template>

        <CalloutBar
          :tone="allocation.hasParticipants.value ? 'success' : 'warning'"
          :message="
            allocation.hasParticipants.value
              ? t('allocation.participants.currentCount', { count: allocation.participantCount.value })
              : t('allocation.participants.emptyHint')
          "
        />
      </SectionCard>

      <SectionCard :title="t('allocation.tabs.rules')">
        <template #actions>
          <NButton size="small" @click="openPickFromMaster">{{ t('products.pickFromMasterAction') }}</NButton>
          <NButton size="small" @click="openCreateRule">{{ t('allocation.rules.addRule') }}</NButton>
          <NButton
            size="small"
            :loading="allocation.reconciling.value"
            :disabled="!allocation.hasParticipants.value"
            @click="handleReconcile"
          >
            {{ t('allocation.rules.reconcile') }}
          </NButton>
        </template>

        <CalloutBar v-if="reconcileSummaryMessage" tone="info" :message="reconcileSummaryMessage" />
        <CalloutBar
          v-if="!allocation.hasParticipants.value"
          tone="warning"
          :message="t('allocation.rules.needsParticipantsHint')"
        />

        <DataGrid
          :columns="columns"
          :rows="allocation.rules.value"
          row-key="id"
          :loading="allocation.loadingRules.value"
          pagination="client"
          :empty="{ title: t('allocation.rules.empty') }"
        />
      </SectionCard>

      <SectionCard :title="t('allocation.tabs.mapping')">
        <template #actions>
          <NButton
            size="small"
            type="primary"
            :loading="allocation.mappingRunning.value"
            :disabled="!allocation.hasParticipants.value"
            @click="handleRunMapping"
          >
            {{ t('allocation.mapping.run') }}
          </NButton>
        </template>

        <CalloutBar
          v-if="!allocation.hasParticipants.value"
          tone="warning"
          :message="t('allocation.mapping.needsParticipantsHint')"
        />

        <p class="wave-allocation-tab__assigned-count">
          {{ t('allocation.mapping.assignedDemandCount', { count: allocation.assignedDemands.value.length }) }}
        </p>

        <ul v-if="allocation.assignedDemands.value.length > 0" class="wave-allocation-tab__assigned-list">
          <li v-for="doc in allocation.assignedDemands.value" :key="doc.id" class="wave-allocation-tab__assigned-item">
            <span class="wave-allocation-tab__assigned-doc-no">{{ doc.sourceDocumentNo }}</span>
            <StatusBadge dimension="demandKind" :value="doc.kind" size="sm" />
          </li>
        </ul>

        <MappingResultPanel :result="allocation.lastMappingResult.value" />
      </SectionCard>
    </template>

    <RuleEditor
      v-model:show="showRuleEditor"
      :wave-id="allocation.waveId.value"
      :products="allocation.products.value"
      :rule="editingRule"
      @saved="handleRuleSaved"
    />

    <BatchStockToWaveDialog
      v-model:show="showPickFromMaster"
      :preselected-wave-id="allocation.waveId.value"
      @success="handlePickFromMasterSuccess"
    />
  </div>
</template>

<style scoped>
.wave-allocation-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-allocation-tab__summary {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--space-3);
}

.wave-allocation-tab__assigned-count {
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.wave-allocation-tab__assigned-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  max-height: 220px;
  margin: 0 0 var(--space-4);
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.wave-allocation-tab__assigned-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-1) var(--space-2);
  border-bottom: 1px solid var(--color-border);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}
</style>
