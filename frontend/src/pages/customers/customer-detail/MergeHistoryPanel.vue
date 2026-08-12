<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSpin } from 'naive-ui'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { MergeHistoryList, UndoDryRunPanel } from '@/shared/ui/customer-resolution'
import {
  buildMergeHistory,
  buildUndoDryRun,
  mergeBlockerTranslationKey,
  mergeEventStatusTranslationKey,
  mergeEventTypeTranslationKey,
  mergeOperationAccess,
} from '@/shared/lib/customer-resolution'
import {
  dryRunCustomerMergeUndo,
  executeCustomerMergeUndo,
  getCustomerMergeHistory,
  listCustomerMergeHistory,
} from '@/shared/api/bridge'
import type {
  CustomerMergeHistoryDetail,
  CustomerMergeHistoryItem,
  CustomerMergeUndoDryRunResult,
  ExecuteCustomerMergeUndoResult,
} from '@/entities/merge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'

const props = defineProps<{ customerProfileId: number }>()
const emit = defineEmits<{ undone: [ExecuteCustomerMergeUndoResult] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()
const items = ref<CustomerMergeHistoryItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const loadError = ref<string | null>(null)
const nextCreatedAt = ref<string | null>(null)
const nextId = ref<number | null>(null)
const detail = ref<CustomerMergeHistoryDetail | null>(null)
const inspecting = ref(false)
const dryRun = ref<CustomerMergeUndoDryRunResult | null>(null)
const dryRunLoading = ref(false)
const undoing = ref(false)
const mergeWritesEnabled = computed(() => featurePolicy.isEnabled('mergeExecutionEnabled'))
const mergeAccess = computed(() => mergeOperationAccess(mergeWritesEnabled.value))

const historyView = computed(() => buildMergeHistory(items.value.map((item) => ({
  id: item.mergeId,
  sourceProfileId: item.sourceProfileId,
  targetProfileId: item.targetProfileId,
  status: item.status === 'completed' || item.status === 'undone' || item.status === 'blocked' ? item.status : 'failed',
  triggerType: item.mergeMode === 'policy' || item.mergeMode === 'migration' ? item.mergeMode : 'manual',
  createdAt: item.createdAt,
  undoneAt: item.undoneAt,
}))))

const undoView = computed(() => dryRun.value ? buildUndoDryRun({
  mergeId: dryRun.value.mergeId,
  eligible: dryRun.value.eligible,
  blockers: (dryRun.value.blockers ?? []).map((blocker) => ({ code: blocker.code, entityType: blocker.entityType, entityId: blocker.entityId })),
  restoreCounts: {
    identities: dryRun.value.restoreCounts.identities,
    addresses: dryRun.value.restoreCounts.addresses,
    nameObservations: dryRun.value.restoreCounts.nameObservations,
    demandDocuments: dryRun.value.restoreCounts.demandDocuments,
  },
}) : null)

function blockerLabel(code: string): string {
  const key = mergeBlockerTranslationKey(code)
  return key ? t(key) : t('merge.blocker.unknown', { code })
}

function eventTypeLabel(eventType: string): string {
  const key = mergeEventTypeTranslationKey(eventType)
  return key ? t(key) : t('merge.history.eventType.unknown', { value: eventType })
}

function eventStatusLabel(status: string): string {
  const key = mergeEventStatusTranslationKey(status)
  return key ? t(key) : t('merge.history.eventStatus.unknown', { value: status })
}

const undoBlockerLabels = computed<Record<string, string>>(() => Object.fromEntries(
  (dryRun.value?.blockers ?? []).map((blocker) => [blocker.code, blockerLabel(blocker.code)]),
))

async function load(reset = true): Promise<void> {
  if (reset) loading.value = true
  else loadingMore.value = true
  loadError.value = null
  try {
    const page = await listCustomerMergeHistory({
      profileId: props.customerProfileId,
      limit: 50,
      beforeCreatedAt: reset ? undefined : nextCreatedAt.value ?? undefined,
      beforeId: reset ? undefined : nextId.value ?? undefined,
    })
    items.value = reset
      ? page.items
      : [...items.value, ...page.items.filter((item) => !items.value.some((existing) => existing.mergeId === item.mergeId))]
    nextCreatedAt.value = page.nextCreatedAt ?? null
    nextId.value = page.nextId || null
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (reset) loading.value = false
    else loadingMore.value = false
  }
}

async function inspect(mergeId: number): Promise<void> {
  if (inspecting.value) return
  inspecting.value = true
  try {
    detail.value = await getCustomerMergeHistory(mergeId)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    inspecting.value = false
  }
}

async function requestUndoDryRun(mergeId: number): Promise<void> {
  if (dryRunLoading.value || !mergeAccess.value.canDryRunUndo) return
  dryRunLoading.value = true
  try {
    dryRun.value = await dryRunCustomerMergeUndo(mergeId)
  } catch (err) {
    feedback.error(t('merge.undoError'), err instanceof Error ? err.message : String(err))
  } finally {
    dryRunLoading.value = false
  }
}

async function executeUndo(): Promise<void> {
  if (!dryRun.value || !undoView.value?.canConfirm || undoing.value || !mergeAccess.value.canExecuteUndo) return
  undoing.value = true
  try {
    const result = await executeCustomerMergeUndo({
      mergeId: dryRun.value.mergeId,
      undoOperationKey: `merge-history-undo-${crypto.randomUUID()}`,
      eligibilityToken: dryRun.value.eligibilityToken,
      expectedSourceRowVersion: dryRun.value.sourceRowVersion,
      expectedTargetRowVersion: dryRun.value.targetRowVersion,
      actorRef: 'local_user',
      reason: 'operator confirmed history undo dry-run',
    })
    dryRun.value = null
    detail.value = null
    await load()
    emit('undone', result)
  } catch (err) {
    feedback.error(t('merge.undoError'), err instanceof Error ? err.message : String(err))
    dryRun.value = null
    await load()
  } finally {
    undoing.value = false
  }
}

watch(() => props.customerProfileId, () => void load(true))
onMounted(() => {
  void load(true)
  void featurePolicy.load()
})
</script>

<template>
  <div class="merge-history-panel">
    <ErrorBanner
      v-if="loadError"
      :message="t('merge.history.loadFailed')"
      :detail="loadError"
      @retry="load(true)"
    />
    <p v-if="!mergeWritesEnabled" class="merge-history-panel__disabled">{{ t('merge.disabledReason') }}</p>
    <div v-if="loading" class="merge-history-panel__loading"><NSpin size="small" /></div>
    <MergeHistoryList
      v-else
      :operations="historyView"
      :status-labels="{
        completed: t('merge.history.status.completed'),
        undone: t('merge.history.status.undone'),
        blocked: t('merge.history.status.blocked'),
        failed: t('merge.history.status.failed'),
      }"
      :inspect-label="t('merge.history.inspectAction')"
      :undo-dry-run-label="t('merge.undoDryRunAction')"
      :empty-label="t('merge.history.empty')"
      @inspect="inspect"
      @undo-dry-run="requestUndoDryRun"
    />
    <NButton
      v-if="nextCreatedAt && nextId"
      size="small"
      :loading="loadingMore"
      @click="load(false)"
    >
      {{ t('merge.history.loadMore') }}
    </NButton>

    <NModal
      :show="detail != null"
      preset="card"
      :title="t('merge.history.detailTitle', { id: detail?.mergeId ?? 0 })"
      :style="{ width: 'min(760px, 94vw)' }"
      @update:show="(visible: boolean) => { if (!visible) detail = null }"
    >
      <template v-if="detail">
        <dl class="merge-history-panel__detail">
          <div><dt>{{ t('merge.history.source') }}</dt><dd>{{ detail.sourceDisplayName }} (#{{ detail.sourceProfileId }})</dd></div>
          <div><dt>{{ t('merge.history.target') }}</dt><dd>{{ detail.targetDisplayName }} (#{{ detail.targetProfileId }})</dd></div>
          <div><dt>{{ t('merge.history.actor') }}</dt><dd>{{ detail.actorRef }}</dd></div>
          <div><dt>{{ t('merge.history.reason') }}</dt><dd>{{ t('merge.history.reasonValue', { value: detail.decisionReason }) }}</dd></div>
          <div><dt>{{ t('merge.history.auditLevel') }}</dt><dd>{{ detail.auditLevel }}</dd></div>
        </dl>
        <dl class="merge-history-panel__counts">
          <div><dt>{{ t('merge.receipt.migratedIdentityCount') }}</dt><dd>{{ detail.counts.identities }}</dd></div>
          <div><dt>{{ t('merge.receipt.migratedAddressCount') }}</dt><dd>{{ detail.counts.addresses }}</dd></div>
          <div><dt>{{ t('merge.receipt.updatedDemandDocs') }}</dt><dd>{{ detail.counts.demandDocuments }}</dd></div>
          <div><dt>{{ t('merge.receipt.nameObservations') }}</dt><dd>{{ detail.counts.nameObservations }}</dd></div>
          <div><dt>{{ t('merge.receipt.nameEvents') }}</dt><dd>{{ detail.counts.nameEvents }}</dd></div>
          <div><dt>{{ t('merge.receipt.origins') }}</dt><dd>{{ detail.counts.origins }}</dd></div>
          <div><dt>{{ t('merge.receipt.profileMutations') }}</dt><dd>{{ detail.counts.profileMutations }}</dd></div>
        </dl>
        <ol class="merge-history-panel__events">
          <li v-for="event in detail.events" :key="`${event.eventType}-${event.createdAt}`">
            {{ event.createdAt }} · {{ eventTypeLabel(event.eventType) }} · {{ eventStatusLabel(event.status) }}
          </li>
        </ol>
        <NButton
          v-if="detail.canRequestUndoDryRun"
          type="warning"
          @click="detail && requestUndoDryRun(detail.mergeId)"
        >
          {{ t('merge.undoDryRunAction') }}
        </NButton>
      </template>
    </NModal>

    <NModal
      :show="dryRun != null"
      preset="card"
      :title="t('merge.undoDryRunTitle')"
      :style="{ width: 'min(640px, 94vw)' }"
      @update:show="(visible: boolean) => { if (!visible && !undoing) dryRun = null }"
    >
      <section v-if="dryRun" class="merge-history-panel__undo-meta">
        <p><strong>{{ t('merge.history.auditLevel') }}:</strong> {{ dryRun.auditLevel }}</p>
        <dl class="merge-history-panel__counts">
          <div><dt>{{ t('merge.receipt.nameEvents') }}</dt><dd>{{ dryRun.restoreCounts.nameEvents }}</dd></div>
          <div><dt>{{ t('merge.receipt.origins') }}</dt><dd>{{ dryRun.restoreCounts.origins }}</dd></div>
          <div><dt>{{ t('merge.receipt.profileMutations') }}</dt><dd>{{ dryRun.restoreCounts.profileMutations }}</dd></div>
        </dl>
        <template v-if="(dryRun.dependentMergeIds ?? []).length">
          <strong>{{ t('merge.history.dependencies') }}</strong>
          <ul><li v-for="id in (dryRun.dependentMergeIds ?? [])" :key="id">#{{ id }}</li></ul>
        </template>
        <template v-if="(dryRun.warnings ?? []).length">
          <strong>{{ t('merge.history.warnings') }}</strong>
          <ul><li v-for="(warning, index) in (dryRun.warnings ?? [])" :key="index">{{ warning }}</li></ul>
        </template>
      </section>
      <UndoDryRunPanel
        v-if="undoView"
        :result="undoView"
        :blocker-labels="undoBlockerLabels"
        :labels="{
          identities: t('merge.receipt.migratedIdentityCount'),
          addresses: t('merge.receipt.migratedAddressCount'),
          nameObservations: t('merge.receipt.nameObservations'),
          demandDocuments: t('merge.receipt.updatedDemandDocs'),
          confirm: t('merge.undoAction'),
          blocked: t('merge.blockersTitle'),
        }"
        :disabled="!mergeAccess.canExecuteUndo"
        :disabled-message="!mergeAccess.canExecuteUndo ? t('merge.disabledReason') : ''"
        @confirm="executeUndo"
      />
    </NModal>
  </div>
</template>

<style scoped>
.merge-history-panel__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-3);
}

.merge-history-panel__disabled {
  margin: 0 0 var(--space-2);
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.merge-history-panel__detail {
  display: grid;
  gap: var(--space-2);
  margin: 0 0 var(--space-3);
}

.merge-history-panel__counts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: var(--space-2);
  margin: 0 0 var(--space-3);
}

.merge-history-panel__counts > div,
.merge-history-panel__undo-meta {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.merge-history-panel__counts dd {
  margin: var(--space-1) 0 0;
  font-weight: var(--font-weight-semibold);
}

.merge-history-panel__undo-meta {
  margin-top: var(--space-3);
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.merge-history-panel__detail > div {
  display: grid;
  grid-template-columns: 100px minmax(0, 1fr);
  gap: var(--space-2);
}

.merge-history-panel__detail dt {
  color: var(--color-text-muted);
}

.merge-history-panel__detail dd {
  margin: 0;
}

.merge-history-panel__events {
  margin: 0 0 var(--space-3);
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}
</style>
