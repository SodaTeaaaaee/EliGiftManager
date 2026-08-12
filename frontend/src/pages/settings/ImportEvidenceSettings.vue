<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCard, NModal, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import type {
  ImportEvidenceRetentionDTO,
  ImportRunDetailDTO,
  ImportRunSummaryDTO,
} from '@/entities/customer-resolution'
import {
  getImportEvidenceRetention,
  getImportRunDetail,
  listImportRunsPage,
  pruneExpiredImportEvidence,
  setImportEvidenceRetention,
} from '@/shared/api/bridge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'
import { ErrorBanner } from '@/shared/ui/feedback'
import {
  beginDetailLoad,
  completeDetailLoad,
  createDetailLoadFlow,
  failDetailLoad,
  invalidateDetailLoad,
} from '@/shared/lib/customer-resolution'

const { t } = useI18n({ useScope: 'global' })
const featurePolicy = useCustomerResolutionFeaturePolicy()

const retention = ref<ImportEvidenceRetentionDTO | null>(null)
const selectedRetentionDays = ref(90)
const runs = ref<ImportRunSummaryDTO[]>([])
const nextCursor = ref('')
const hasMore = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const savingRetention = ref(false)
const pruning = ref(false)
const error = ref<string | null>(null)
const writeError = ref<string | null>(null)
const pruneSummary = ref<{ runsDeleted: number; recordsDeleted: number } | null>(null)

const detailOpen = ref(false)
const detailFlow = ref(createDetailLoadFlow<ImportRunDetailDTO>())
const detailError = computed(() => detailFlow.value.error)
const detail = computed(() => detailFlow.value.detail)
const detailRunId = computed(() => detailFlow.value.requestedId)
const detailLoading = computed(() => detailFlow.value.loading)

const retentionOptions = computed<SelectOption[]>(() => [
  { label: t('settings.importEvidence.retention.none'), value: 0 },
  { label: t('settings.importEvidence.retention.days30'), value: 30 },
  { label: t('settings.importEvidence.retention.days90'), value: 90 },
  { label: t('settings.importEvidence.retention.permanent'), value: -1 },
])

const evidenceWritesEnabled = computed(() => featurePolicy.isEnabled('importEvidenceEnabled'))
const disabledMessage = computed(() => evidenceWritesEnabled.value ? '' : t('settings.importEvidence.disabledReason'))

function importKindCopy(run: ImportRunSummaryDTO): string {
  return run.importKind
}

function recordResultCopy(record: ImportRunDetailDTO['records'][number]): string {
  return `${record.outcome} · ${record.resultType} · ${record.resultId ?? '—'}`
}

function formatDate(value?: string): string {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}

async function loadInitial(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const [retentionResult, page] = await Promise.all([
      getImportEvidenceRetention(),
      listImportRunsPage({ limit: 25 }),
      featurePolicy.load(),
    ])
    retention.value = retentionResult
    selectedRetentionDays.value = retentionResult.retentionDays
    runs.value = page.items ?? []
    nextCursor.value = page.nextCursor
    hasMore.value = page.hasMore
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function loadMore(): Promise<void> {
  if (!hasMore.value || loadingMore.value) return
  loadingMore.value = true
  error.value = null
  try {
    const page = await listImportRunsPage({ limit: 25, cursor: nextCursor.value })
    runs.value.push(...(page.items ?? []))
    nextCursor.value = page.nextCursor
    hasMore.value = page.hasMore
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loadingMore.value = false
  }
}

async function saveRetention(): Promise<void> {
  if (!evidenceWritesEnabled.value) return
  savingRetention.value = true
  writeError.value = null
  try {
    retention.value = await setImportEvidenceRetention(selectedRetentionDays.value)
    selectedRetentionDays.value = retention.value.retentionDays
  } catch (err) {
    writeError.value = err instanceof Error ? err.message : String(err)
    selectedRetentionDays.value = retention.value?.retentionDays ?? 90
  } finally {
    savingRetention.value = false
  }
}

async function prune(): Promise<void> {
  if (!evidenceWritesEnabled.value) return
  pruning.value = true
  writeError.value = null
  pruneSummary.value = null
  try {
    const result = await pruneExpiredImportEvidence()
    pruneSummary.value = {
      runsDeleted: result.runsDeleted ?? 0,
      recordsDeleted: result.recordsDeleted ?? 0,
    }
    await loadInitial()
  } catch (err) {
    writeError.value = err instanceof Error ? err.message : String(err)
  } finally {
    pruning.value = false
  }
}

async function openSensitiveDetail(runId: number): Promise<void> {
  const attempt = beginDetailLoad(detailFlow.value, runId)
  detailFlow.value = attempt.flow
  detailOpen.value = true
  try {
    const result = await getImportRunDetail(runId)
    detailFlow.value = completeDetailLoad(detailFlow.value, attempt.request, result, result.run.id)
  } catch (err) {
    detailFlow.value = failDetailLoad(
      detailFlow.value,
      attempt.request,
      err instanceof Error ? err.message : String(err),
    )
  }
}

function closeDetail(): void {
  detailOpen.value = false
  detailFlow.value = invalidateDetailLoad(detailFlow.value)
}

onMounted(() => {
  void loadInitial()
})
</script>

<template>
  <div class="import-evidence">
    <ErrorBanner
      v-if="error || writeError"
      :message="writeError ? t('settings.importEvidence.writeFailed') : t('settings.importEvidence.loadFailed')"
      :detail="writeError ?? error ?? undefined"
      @retry="loadInitial"
    />

    <div class="import-evidence__retention">
      <div class="import-evidence__copy">
        <span class="import-evidence__label">{{ t('settings.importEvidence.retentionLabel') }}</span>
        <span class="import-evidence__hint">{{ t('settings.importEvidence.retentionHint') }}</span>
        <span v-if="disabledMessage" class="import-evidence__disabled">{{ disabledMessage }}</span>
      </div>
      <div class="import-evidence__retention-actions">
        <NSelect
          v-model:value="selectedRetentionDays"
          class="import-evidence__retention-select"
          :options="retentionOptions"
          :disabled="loading || savingRetention || !evidenceWritesEnabled"
        />
        <NButton
          :loading="savingRetention"
          :disabled="loading || savingRetention || selectedRetentionDays === retention?.retentionDays || !evidenceWritesEnabled"
          @click="saveRetention"
        >
          {{ t('common.save') }}
        </NButton>
        <NButton :loading="pruning" :disabled="loading || pruning || !evidenceWritesEnabled" @click="prune">
          {{ t('settings.importEvidence.pruneAction') }}
        </NButton>
      </div>
    </div>

    <p v-if="pruneSummary" class="import-evidence__hint">
      {{ t('settings.importEvidence.pruneSummary', pruneSummary) }}
    </p>

    <div class="import-evidence__heading">
      <div class="import-evidence__copy">
        <span class="import-evidence__label">{{ t('settings.importEvidence.runsTitle') }}</span>
        <span class="import-evidence__hint">{{ t('settings.importEvidence.safeListHint') }}</span>
      </div>
      <NButton size="small" :loading="loading" @click="loadInitial">{{ t('common.refresh') }}</NButton>
    </div>

    <p v-if="loading" class="import-evidence__hint">{{ t('common.loading') }}</p>
    <p v-else-if="runs.length === 0" class="import-evidence__hint">{{ t('settings.importEvidence.empty') }}</p>
    <div v-else class="import-evidence__runs">
      <article v-for="run in runs" :key="run.id" class="import-evidence__run">
        <div class="import-evidence__run-main">
          <strong>{{ importKindCopy(run) }}</strong>
          <span>#{{ run.id }} · {{ run.status }} · {{ run.sourceFormat }}</span>
          <span>{{ run.sourceFileName }} · {{ formatDate(run.createdAt) }}</span>
          <span>
            {{ t('settings.importEvidence.counts', {
              records: run.recordCount,
              success: run.successCount,
              failure: run.failureCount,
              quarantine: run.quarantinedCount,
            }) }}
          </span>
        </div>
        <NButton size="small" @click="openSensitiveDetail(run.id)">
          {{ t('settings.importEvidence.viewSensitiveDetail') }}
        </NButton>
      </article>
    </div>
    <NButton v-if="hasMore" :loading="loadingMore" @click="loadMore">
      {{ t('common.loadMore') }}
    </NButton>

    <NModal :show="detailOpen" @update:show="$event || closeDetail()">
      <NCard class="import-evidence__detail-card" :title="t('settings.importEvidence.detailTitle')" closable @close="closeDetail">
        <p class="import-evidence__sensitive-warning">{{ t('settings.importEvidence.sensitiveWarning') }}</p>
        <ErrorBanner
          v-if="detailError"
          :message="t('settings.importEvidence.detailFailed')"
          :detail="detailError"
          @retry="detailRunId != null && openSensitiveDetail(detailRunId)"
        />
        <p v-if="detailLoading">{{ t('common.loading') }}</p>
        <div v-else-if="detail" class="import-evidence__records">
          <article v-for="record in detail.records" :key="record.id" class="import-evidence__record">
            <strong>{{ t('settings.importEvidence.recordTitle', { row: record.rowIndex }) }}</strong>
            <dl>
              <dt>{{ t('settings.importEvidence.rawLogicalRow') }}</dt><dd><pre>{{ record.rawLogicalRow }}</pre></dd>
              <dt>{{ t('settings.importEvidence.unmappedSource') }}</dt><dd><pre>{{ record.unmappedSource }}</pre></dd>
              <dt>{{ t('settings.importEvidence.parserMetadata') }}</dt><dd><pre>{{ record.parserMetadata }}</pre></dd>
              <dt>{{ t('settings.importEvidence.warningCodes') }}</dt><dd><pre>{{ record.warningCodes }}</pre></dd>
              <dt>{{ t('settings.importEvidence.result') }}</dt><dd>{{ recordResultCopy(record) }}</dd>
              <dt>{{ t('settings.importEvidence.error') }}</dt><dd>{{ record.errorCode }} · {{ record.errorMessage }}</dd>
            </dl>
          </article>
        </div>
      </NCard>
    </NModal>
  </div>
</template>

<style scoped>
.import-evidence,
.import-evidence__copy,
.import-evidence__runs,
.import-evidence__run-main,
.import-evidence__records {
  display: flex;
  flex-direction: column;
}

.import-evidence,
.import-evidence__runs,
.import-evidence__records {
  gap: var(--space-3);
}

.import-evidence__retention,
.import-evidence__heading,
.import-evidence__run,
.import-evidence__retention-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.import-evidence__copy,
.import-evidence__run-main {
  gap: var(--space-1);
}

.import-evidence__label {
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.import-evidence__hint,
.import-evidence__disabled,
.import-evidence__run-main {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.import-evidence__disabled,
.import-evidence__sensitive-warning {
  color: var(--status-warning-fg);
}

.import-evidence__retention-select {
  width: 180px;
}

.import-evidence__run,
.import-evidence__record {
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.import-evidence__detail-card {
  width: min(920px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  overflow: auto;
}

.import-evidence__record dl {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  gap: var(--space-2);
  margin: var(--space-2) 0 0;
}

.import-evidence__record dt {
  color: var(--color-text-secondary);
}

.import-evidence__record dd {
  min-width: 0;
  margin: 0;
}

.import-evidence__record pre {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
