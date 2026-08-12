<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSpin } from 'naive-ui'
import type { CustomerSplitHistoryDetail, CustomerSplitHistoryItem } from '@/entities/customer-resolution'
import { getCustomerSplitHistory, listCustomerSplitHistory } from '@/shared/api/bridge'
import { ErrorBanner } from '@/shared/ui/feedback'
import {
  beginDetailLoad,
  completeDetailLoad,
  createDetailLoadFlow,
  failDetailLoad,
  invalidateDetailLoad,
} from '@/shared/lib/customer-resolution'

const props = defineProps<{ customerProfileId: number; refreshSignal?: number }>()
const { t } = useI18n({ useScope: 'global' })
const items = ref<CustomerSplitHistoryItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const error = ref<string | null>(null)
const hasMore = ref(false)
const nextBefore = ref<string | null>(null)
const nextId = ref<number | null>(null)
const detailFlow = ref(createDetailLoadFlow<CustomerSplitHistoryDetail>())
const detail = computed(() => detailFlow.value.detail)
const detailError = computed(() => detailFlow.value.error)
const detailSplitId = computed(() => detailFlow.value.requestedId)
const detailLoading = computed(() => detailFlow.value.loading)

function decisionCopy(row: CustomerSplitHistoryDetail): string {
  return `${row.actorRef} · ${row.decisionReason}`
}

function movedEntityCopy(entity: CustomerSplitHistoryDetail['movedEntities'][number]): string {
  return `${entity.entityType} #${entity.entityId} · ${entity.fromProfileId} → ${entity.toProfileId} · ${entity.mutationKind}`
}

function eventCopy(event: CustomerSplitHistoryDetail['events'][number]): string {
  return `${event.createdAt} · ${event.eventType} · ${event.status} · ${event.reasonCode}`
}

async function load(reset = true): Promise<void> {
  if (reset) loading.value = true
  else loadingMore.value = true
  error.value = null
  try {
    const page = await listCustomerSplitHistory({
      profileId: props.customerProfileId,
      limit: 30,
      beforeCreatedAt: reset ? undefined : nextBefore.value ?? undefined,
      beforeId: reset ? undefined : nextId.value ?? undefined,
    })
    items.value = reset ? page.items : [
      ...items.value,
      ...page.items.filter((item) => !items.value.some((existing) => existing.splitId === item.splitId)),
    ]
    hasMore.value = page.hasMore
    nextBefore.value = page.nextBefore ?? null
    nextId.value = page.nextId || null
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

async function inspect(splitId: number): Promise<void> {
  const attempt = beginDetailLoad(detailFlow.value, splitId)
  detailFlow.value = attempt.flow
  try {
    const result = await getCustomerSplitHistory(splitId)
    const belongsToProfile = result.sourceProfileId === props.customerProfileId
      || result.targetProfileId === props.customerProfileId
    detailFlow.value = completeDetailLoad(
      detailFlow.value,
      attempt.request,
      result,
      belongsToProfile ? result.splitId : -1,
    )
  } catch (err) {
    detailFlow.value = failDetailLoad(
      detailFlow.value,
      attempt.request,
      err instanceof Error ? err.message : String(err),
    )
  }
}

function closeDetail(): void {
  detailFlow.value = invalidateDetailLoad(detailFlow.value)
}

watch(
  () => [props.customerProfileId, props.refreshSignal] as const,
  ([profileId], previous) => {
    if (previous && profileId !== previous[0]) closeDetail()
    void load(true)
  },
  { immediate: true },
)
</script>

<template>
  <div class="split-history">
    <ErrorBanner v-if="error" :message="t('customerDetail.split.historyLoadFailed')" :detail="error" @retry="load(true)" />
    <NSpin v-if="loading" size="small" />
    <p v-else-if="items.length === 0" class="split-history__hint">{{ t('customerDetail.split.historyEmpty') }}</p>
    <div v-else class="split-history__list">
      <article v-for="item in items" :key="item.splitId" class="split-history__item">
        <div class="split-history__summary">
          <strong>#{{ item.splitId }} · {{ item.status }}</strong>
          <span>{{ item.sourceProfileId }} → {{ item.targetProfileId }} · {{ item.createdAt }}</span>
          <span>{{ t('customerDetail.split.counts', { identities: item.counts.identities, addresses: item.counts.addresses, demand: item.counts.demandDocuments, names: item.counts.nameObservations, origins: item.counts.origins }) }}</span>
          <span v-if="!item.directUndoSupported">{{ t('customerDetail.split.noDirectUndo') }}</span>
        </div>
        <NButton size="small" :loading="detailLoading" @click="inspect(item.splitId)">{{ t('common.details') }}</NButton>
      </article>
    </div>
    <NButton v-if="hasMore" size="small" :loading="loadingMore" @click="load(false)">{{ t('common.loadMore') }}</NButton>

    <NModal
      :show="detailSplitId != null"
      preset="card"
      :title="t('customerDetail.split.historyDetail', { id: detail?.splitId ?? detailSplitId ?? 0 })"
      :style="{ width: 'min(840px, 94vw)' }"
      @update:show="(visible: boolean) => { if (!visible) closeDetail() }"
    >
      <ErrorBanner
        v-if="detailError"
        :message="t('customerDetail.split.historyLoadFailed')"
        :detail="detailError"
        @retry="detailSplitId != null && inspect(detailSplitId)"
      />
      <NSpin v-if="detailLoading" size="small" />
      <div v-if="detail" class="split-history__detail">
        <p>{{ decisionCopy(detail) }}</p>
        <code>{{ detail.planHash }}</code>
        <p>{{ detail.reverseGuidance || t('customerDetail.split.noDirectUndo') }}</p>
        <h4>{{ t('customerDetail.split.movedEntities') }}</h4>
        <ul>
          <li v-for="entity in detail.movedEntities" :key="`${entity.entityType}-${entity.entityId}`">
            {{ movedEntityCopy(entity) }}
          </li>
        </ul>
        <h4>{{ t('customerDetail.split.events') }}</h4>
        <ol>
          <li v-for="(event, index) in detail.events" :key="`${event.eventType}-${index}`">
            {{ eventCopy(event) }}
          </li>
        </ol>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.split-history,
.split-history__list,
.split-history__summary,
.split-history__detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.split-history__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.split-history__summary,
.split-history__hint,
.split-history__detail {
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.split-history__hint {
  margin: 0;
}
</style>
