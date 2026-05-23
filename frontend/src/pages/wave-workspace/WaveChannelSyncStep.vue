<script setup lang="ts">
import { computed, onMounted, ref, h, inject, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { 
  NAlert, 
  NButton, 
  NCard, 
  NDataTable, 
  NEmpty, 
  NInput, 
  NSelect, 
  NSpace, 
  NTag, 
  NGrid, 
  NGridItem, 
  NProgress, 
  NTooltip,
  useMessage 
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { 
  executeChannelSyncJob, 
  listChannelSyncJobsByWave, 
  listIntegrationProfiles, 
  planChannelClosure, 
  recordChannelClosureDecision, 
  retryChannelSyncJob,
  getWaveWorkspaceSnapshot
} from '@/shared/lib/wails/app'
import { useI18n } from '@/shared/i18n'
import { dto } from '@/../wailsjs/go/models'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()
const waveId = computed(() => Number(route.params.waveId) || 0)

const snapshot = inject('waveWorkspaceSnapshot') as Ref<dto.WaveWorkspaceSnapshotDTO | null> | undefined
const overview = computed(() => snapshot?.value?.overview)

const loading = ref(false)
const profilesLoading = ref(false)
const planning = ref(false)
const submitting = ref(false)
const profiles = ref<dto.IntegrationProfileSummaryDTO[]>([])
const jobs = ref<dto.ChannelSyncJobDTO[]>([])
const selectedProfileId = ref<number | null>(null)
const planResult = ref<dto.PlanChannelClosureResult | null>(null)
const error = ref('')

const manualForms = ref<Record<number, { decisionKind: string; reasonCode: string; note: string; evidenceRef: string; operatorId: string }>>({})

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    label: `${profile.profileKey} (${profile.sourceChannel})`,
    value: profile.id,
  })),
)

const selectedProfileDetail = computed(() => {
  return profiles.value.find((profile) => profile.id === selectedProfileId.value)
})

const decisionKindOptions = computed(() => {
  const base = [
    { label: t('sync.decisionOptions.mark_sync_unsupported'), value: 'mark_sync_unsupported' },
    { label: t('sync.decisionOptions.mark_sync_skipped'), value: 'mark_sync_skipped' },
  ]
  if (selectedProfileDetail.value?.allowsManualClosure) {
    base.push({ label: t('sync.decisionOptions.mark_sync_completed_manually'), value: 'mark_sync_completed_manually' })
  }
  return base
})

// Compute sync progress percentage
const syncProgressPct = computed(() => {
  if (!overview.value) return 0
  const total = overview.value.channelSyncJobCount || 0
  if (total === 0) {
    return overview.value.projectedLifecycleStage === 'closed' ? 100 : 0
  }
  const success = overview.value.channelSyncSuccessCount || 0
  return Math.round((success / total) * 100)
})

const columns: DataTableColumns<dto.ChannelSyncJobDTO> = [
  { title: 'Job ID', key: 'id', width: 80, align: 'center' },
  { 
    title: t('sync.columns.profile'), 
    key: 'integrationProfileId', 
    width: 140,
    render(row) {
      const prof = profiles.value.find(p => p.id === row.integrationProfileId)
      return prof ? `${prof.profileKey} (${prof.sourceChannel})` : `Profile #${row.integrationProfileId}`
    }
  },
  { 
    title: t('sync.columns.direction'), 
    key: 'direction', 
    width: 120,
    render(row) {
      const type = row.direction === 'upload' || row.direction === 'sync_back' ? 'info' : 'default'
      return h(NTag, { type, size: 'small', bordered: false }, { default: () => row.direction })
    }
  },
  {
    title: t('sync.columns.status'),
    key: 'status',
    width: 140,
    render(row) {
      let type: 'default' | 'error' | 'success' | 'warning' | 'info' = 'default'
      if (row.status === 'failed') type = 'error'
      else if (row.status === 'success') type = 'success'
      else if (row.status === 'running') type = 'info'
      else if (row.status === 'pending') type = 'warning'
      else if (row.status === 'partial_success') type = 'warning'
      
      return h(NTag, { type, size: 'small', round: true, bordered: false }, { default: () => t(`sync.statusOptions.${row.status}`) || row.status })
    },
  },
  { 
    title: t('sync.columns.error'), 
    key: 'errorMessage',
    render(row) {
      if (!row.errorMessage) return h('span', { class: 'text-slate-400 dark:text-slate-600 text-xs italic' }, 'None')
      return h(
        NTooltip,
        { trigger: 'hover', placement: 'top-start' },
        {
          trigger: () => h('span', { class: 'text-red-500 text-xs truncate max-w-xs block cursor-help' }, row.errorMessage),
          default: () => row.errorMessage
        }
      )
    }
  },
  {
    title: t('sync.columns.actions'),
    key: 'actions',
    width: 180,
    align: 'center',
    render(row) {
      return h(NSpace, { size: 'small', justify: 'center' }, () => [
        row.status === 'pending'
          ? h(NButton, { size: 'small', type: 'primary', onClick: () => handleExecute(row.id) }, { default: () => t('sync.run') })
          : null,
        row.status === 'failed' || row.status === 'partial_success'
          ? h(NButton, { size: 'small', type: 'warning', onClick: () => handleRetry(row.id) }, { default: () => t('sync.retry') })
          : null,
      ])
    },
  },
]

const itemColumns: DataTableColumns<dto.ChannelSyncItemDTO> = [
  { title: 'Item ID', key: 'id', width: 80, align: 'center' },
  { title: 'Fulfillment Line ID', key: 'fulfillmentLineId', width: 140, align: 'center' },
  { title: 'External Doc', key: 'externalDocumentNo', width: 150 },
  { title: 'External Line', key: 'externalLineNo', width: 110, align: 'center' },
  { title: 'Carrier', key: 'carrierCode', width: 100 },
  { title: 'Tracking No', key: 'trackingNo', width: 160 },
  {
    title: 'Status',
    key: 'status',
    width: 120,
    render(itemRow) {
      const type = itemRow.status === 'success' ? 'success' : itemRow.status === 'failed' ? 'error' : 'default'
      return h(NTag, { type, size: 'small', round: true, bordered: false }, { default: () => itemRow.status })
    }
  },
  { 
    title: 'Error Details', 
    key: 'errorMessage',
    render(itemRow) {
      if (!itemRow.errorMessage) return h('span', { class: 'text-slate-400 dark:text-slate-600 text-xs italic' }, 'None')
      return h('span', { class: 'text-red-500 text-xs' }, itemRow.errorMessage)
    }
  }
]

const renderExpand = (row: dto.ChannelSyncJobDTO) => {
  return h(
    'div',
    { class: 'p-4 bg-slate-500/5 dark:bg-slate-900/40 rounded-lg border border-slate-700/10 dark:border-slate-700/30' },
    [
      h('h4', { class: 'text-xs font-bold uppercase tracking-wider text-slate-400 mb-3 flex items-center gap-2' }, [
        h('span', { class: 'inline-block w-2 h-2 rounded-full bg-primary' }),
        'Synced Items Details'
      ]),
      h(NDataTable, {
        columns: itemColumns,
        data: row.items || [],
        size: 'small',
        pagination: false
      })
    ]
  )
}

async function refreshSnapshot() {
  if (waveId.value && snapshot) {
    try {
      snapshot.value = await getWaveWorkspaceSnapshot(waveId.value)
    } catch (e: unknown) {
      console.error('Failed to refresh wave workspace snapshot:', e)
    }
  }
}

async function loadProfiles() {
  profilesLoading.value = true
  try {
    profiles.value = await listIntegrationProfiles()
  } finally {
    profilesLoading.value = false
  }
}

async function loadJobs() {
  loading.value = true
  try {
    jobs.value = await listChannelSyncJobsByWave(waveId.value)
  } finally {
    loading.value = false
  }
}

async function handlePlan() {
  if (!selectedProfileId.value) return
  planning.value = true
  error.value = ''
  planResult.value = null
  try {
    const result = await planChannelClosure({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
    })
    planResult.value = result
    if ((result.decision === 'manual_closure' || result.decision === 'unsupported') && result.items) {
      const forms: Record<number, { decisionKind: string; reasonCode: string; note: string; evidenceRef: string; operatorId: string }> = {}
      for (const item of result.items) {
        forms[item.fulfillmentLineId] = {
          decisionKind: result.decision === 'unsupported' ? 'mark_sync_unsupported' : '',
          reasonCode: result.decision === 'unsupported' ? 'UNSUPPORTED' : 'MANUAL_SHIPPED',
          note: '',
          evidenceRef: '',
          operatorId: 'Operator',
        }
      }
      manualForms.value = forms
    } else {
      message.success('Sync plan calculated successfully. Auto-job registered.')
    }
    await loadJobs()
    await refreshSnapshot()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    planning.value = false
  }
}

async function handleExecute(jobId: number) {
  try {
    await executeChannelSyncJob(jobId)
    message.success('Sync job started.')
    await loadJobs()
    await refreshSnapshot()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function handleRetry(jobId: number) {
  try {
    await retryChannelSyncJob(jobId)
    message.success('Sync job retried.')
    await loadJobs()
    await refreshSnapshot()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function handleSubmitDecisions() {
  if (!selectedProfileId.value || !planResult.value) return
  submitting.value = true
  try {
    const entries = Object.entries(manualForms.value)
      .filter(([, form]) => form.decisionKind)
      .map(([lineId, form]) => ({
        fulfillmentLineId: Number(lineId),
        decisionKind: form.decisionKind,
        reasonCode: form.reasonCode,
        note: form.note,
        evidenceRef: form.evidenceRef,
        operatorId: form.operatorId,
      }))
    await recordChannelClosureDecision({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
      entries,
    })
    message.success('Manual decisions recorded successfully. Channel sync updated.')
    planResult.value = null
    await loadJobs()
    await refreshSnapshot()
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await loadProfiles()
  await loadJobs()
  await refreshSnapshot()
})
</script>

<template>
  <div class="wave-channel-sync-step">
    <!-- Header -->
    <div class="mb-6">
      <div class="app-kicker">{{ t('wave.sync') }}</div>
      <h2 class="app-title mt-2">{{ t('sync.title') }}</h2>
      <p class="app-copy mt-2">{{ t('sync.subtitle') }}</p>
    </div>

    <!-- Error Alert -->
    <NAlert v-if="error" type="error" class="mb-4" :title="error" closable />

    <!-- Channel Sync Health Dashboard -->
    <NGrid :cols="24" :x-gap="16" :y-gap="16" class="mb-5">
      <!-- Sync Progress Card -->
      <NGridItem :span="16">
        <NCard class="glow-card h-full" style="background: linear-gradient(135deg, var(--surface-strong) 0%, var(--surface-muted) 100%); border-left: 5px solid var(--accent);">
          <div class="flex flex-col justify-between h-full">
            <div>
              <div class="app-kicker">Channel Sync Health</div>
              <h3 class="app-heading-md mt-1">Transmission Progress</h3>
            </div>
            <div class="mt-4">
              <NProgress
                type="line"
                :status="overview?.channelSyncFailedCount ? 'error' : overview?.channelSyncRunningCount ? 'info' : 'success'"
                :percentage="syncProgressPct"
                :show-indicator="true"
                processing
              />
              <div class="text-xs text-slate-400 mt-2 flex justify-between">
                <span>
                  {{ t('sync.jobs') }}: {{ overview?.channelSyncSuccessCount ?? 0 }} / {{ overview?.channelSyncJobCount ?? 0 }} {{ t('sync.statusOptions.success') }}
                </span>
                <span v-if="overview?.channelSyncFailedCount" class="text-red-500 font-bold">
                  {{ overview.channelSyncFailedCount }} {{ t('sync.statusOptions.failed') }}
                </span>
              </div>
            </div>
          </div>
        </NCard>
      </NGridItem>

      <!-- Sync Status Grid Details -->
      <NGridItem :span="8">
        <NCard class="glow-card h-full flex flex-col justify-between">
          <div>
            <div class="app-kicker">Manual Decisions</div>
            <h3 class="app-heading-md mt-1 text-2xl font-bold flex items-center gap-2">
              {{ overview?.manualClosureCandidateCount ?? 0 }}
              <span class="text-xs font-normal text-slate-400">candidates</span>
            </h3>
          </div>
          <div class="border-t border-slate-700/10 dark:border-slate-700/30 pt-3 mt-4">
            <div class="flex justify-between text-xs py-1">
              <span class="text-slate-400">Completed Manually:</span>
              <span class="font-bold text-emerald-500">{{ overview?.manualCompletedCount ?? 0 }}</span>
            </div>
            <div class="flex justify-between text-xs py-1">
              <span class="text-slate-400">Unsupported:</span>
              <span class="font-bold text-amber-500">{{ overview?.manualUnsupportedCount ?? 0 }}</span>
            </div>
            <div class="flex justify-between text-xs py-1">
              <span class="text-slate-400">Skipped:</span>
              <span class="font-bold text-slate-500">{{ overview?.manualSkippedCount ?? 0 }}</span>
            </div>
          </div>
        </NCard>
      </NGridItem>
    </NGrid>

    <!-- Orchestrator (Planning) Card -->
    <NCard :title="t('sync.planning')" class="mb-5 glow-card">
      <div class="flex flex-col gap-4">
        <p class="text-xs text-slate-400">
          Select an integration profile to calculate a closure plan. The system will determine whether to push shipment updates automatically or request manual verification.
        </p>
        <div class="flex items-center gap-4 flex-wrap">
          <NSelect
            v-model:value="selectedProfileId"
            :options="profileOptions"
            :loading="profilesLoading"
            placeholder="Select Integration Profile..."
            style="width: 320px"
          />
          <NButton type="primary" :loading="planning" :disabled="!selectedProfileId" @click="handlePlan">
            Calculate Closure Plan
          </NButton>
        </div>
        <div v-if="selectedProfileDetail" class="mt-2 flex flex-wrap gap-2">
          <NTag size="small" type="info" :bordered="false">
            Mode: {{ selectedProfileDetail.trackingSyncMode }}
          </NTag>
          <NTag size="small" :type="selectedProfileDetail.allowsManualClosure ? 'warning' : 'success'" :bordered="false">
            Manual Closure: {{ selectedProfileDetail.allowsManualClosure ? 'Allowed' : 'Not Allowed' }}
          </NTag>
          <NTag size="small" type="default" :bordered="false">
            Policy: {{ selectedProfileDetail.closurePolicy }}
          </NTag>
        </div>
      </div>
    </NCard>

    <!-- Manual Decisions Card -->
    <NCard v-if="planResult && (planResult.decision === 'manual_closure' || planResult.decision === 'unsupported')" class="mb-5 glow-card" title="Awaiting Manual Closure Decisions">
      <template #header-extra>
        <NTag type="warning" size="small">{{ planResult.items?.length ?? 0 }} Lines Require Audit</NTag>
      </template>
      <div class="mb-4 text-xs text-slate-400">
        The target channel does not support automated tracking sync, or policy requires manual confirmation. Record decisions for the following lines.
      </div>
      <div class="flex flex-col gap-4">
        <div
          v-for="item in planResult.items"
          :key="item.fulfillmentLineId"
          class="rounded-lg border border-slate-700/10 dark:border-slate-700/30 p-4 bg-slate-500/5 dark:bg-slate-900/20"
        >
          <!-- Item Context info header -->
          <div class="flex justify-between items-center border-b border-slate-700/10 dark:border-slate-700/30 pb-3 mb-4 flex-wrap gap-2">
            <span class="text-sm font-bold text-slate-200">
              {{ t('sync.fulfillmentLine') }} #{{ item.fulfillmentLineId }}
            </span>
            <div class="flex gap-3 text-xs text-slate-400">
              <span>Doc: <strong class="text-slate-300">{{ item.externalDocumentNo || '—' }}</strong> (Line {{ item.externalLineNo || '—' }})</span>
              <span>Carrier: <strong class="text-slate-300">{{ item.carrierCode || '—' }}</strong></span>
              <span>Tracking: <strong class="text-slate-300">{{ item.trackingNo || '—' }}</strong></span>
            </div>
          </div>

          <!-- Form inputs grid -->
          <NGrid :cols="24" :x-gap="16" :y-gap="16">
            <NGridItem :span="8">
              <div class="text-xs text-slate-400 mb-1">Decision Action</div>
              <NSelect
                v-model:value="manualForms[item.fulfillmentLineId].decisionKind"
                :options="decisionKindOptions"
                placeholder="Choose action..."
              />
            </NGridItem>
            <NGridItem :span="8">
              <div class="text-xs text-slate-400 mb-1">Reason Code</div>
              <NInput 
                v-model:value="manualForms[item.fulfillmentLineId].reasonCode" 
                :placeholder="t('sync.reasonCode')" 
              />
            </NGridItem>
            <NGridItem :span="8">
              <div class="text-xs text-slate-400 mb-1">Evidence Reference</div>
              <NInput 
                v-model:value="manualForms[item.fulfillmentLineId].evidenceRef" 
                :placeholder="t('sync.evidenceRef')" 
              />
            </NGridItem>
            <NGridItem :span="8">
              <div class="text-xs text-slate-400 mb-1">Operator ID</div>
              <NInput 
                v-model:value="manualForms[item.fulfillmentLineId].operatorId" 
                :placeholder="t('sync.operatorId')" 
              />
            </NGridItem>
            <NGridItem :span="16">
              <div class="text-xs text-slate-400 mb-1">Audit Notes</div>
              <NInput 
                v-model:value="manualForms[item.fulfillmentLineId].note" 
                type="textarea" 
                :rows="1" 
                :placeholder="t('sync.note')" 
              />
            </NGridItem>
          </NGrid>
        </div>

        <div class="flex justify-end mt-2">
          <NButton type="primary" :loading="submitting" @click="handleSubmitDecisions">
            Submit Decisions & Complete Sync
          </NButton>
        </div>
      </div>
    </NCard>

    <!-- Jobs Table -->
    <NCard :title="t('sync.jobs')" class="glow-card">
      <NEmpty v-if="!loading && jobs.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="jobs"
        :loading="loading"
        :pagination="{ pageSize: 5 }"
        size="small"
        :row-key="(row: dto.ChannelSyncJobDTO) => row.id"
        :render-expand="renderExpand"
      />
    </NCard>

    <!-- Step Navigation Footer -->
    <div class="mt-6 flex justify-between border-t border-slate-700/10 dark:border-slate-700/30 pt-4">
      <NButton @click="router.push(`/waves/${waveId}/shipment`)">
        {{ t('wave.prevStep') }}
      </NButton>
      <NButton secondary @click="router.push(`/waves`)">
        {{ t('wave.returnToQueue') }}
      </NButton>
    </div>
  </div>
</template>

<style scoped>
.wave-channel-sync-step {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.glow-card {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

:root[data-theme='dark'] .glow-card {
  border: 1px solid rgba(255, 255, 255, 0.05);
  background: #111827;
}

.glow-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.04);
}
</style>
