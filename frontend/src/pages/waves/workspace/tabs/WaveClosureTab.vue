<script setup lang="ts">
/**
 * WaveClosureTab — the wave-workspace "回填收尾 + 关闭波次" tab (P5 plan
 * §3.3.4 third bullet, route `wave-workspace-closure`). Three-part flow:
 *
 * 1. Health summary — `WaveOverviewDTO` channel-sync/manual-closure counts.
 * 2. Generate backfill file — profile picker + `planChannelClosure`, which
 *    branches into an auto-created job (`create_job`), a manual decision
 *    form (`manual_closure`/`unsupported`, see `ManualClosureForm.vue`), or
 *    (implicitly) nothing further to do.
 * 3. Jobs table — `listChannelSyncJobsByWave`, auto-polling while any job is
 *    pending/running, run/retry actions, output-file path + "open
 *    containing folder" (sensei-approved reuse of `revealInFolder` here for
 *    parity with the factory tab).
 * 4. Close Wave — the terminal card. Re-mounts the EXISTING
 *    `CloseWaveDialog.vue` (force+note flow) rather than forking it; this
 *    tab only adds an imprecise pre-warn using already-fetched
 *    `WaveOverviewDTO` counts (see `useClosureTab.ts`'s doc comment on
 *    `overview` — `CloseWave`'s own residual computation is the
 *    authoritative source, this is only an advisory).
 *
 * All backend calls go through `useClosureTab()` (data/actions) — this file
 * only composes it with the shared UI kit and reacts to its state.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSelect, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { revealInFolder } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useClosureTab, parseOutputFilePath, POLL_INTERVAL_MS } from './closure/useClosureTab'
import { buildClosureJobColumns, type ClosureJobRow } from './closure/job-columns'
import ManualClosureForm from './closure/ManualClosureForm.vue'
import JobItemsDrawer from './closure/JobItemsDrawer.vue'
import CloseWaveDialog from '../../components/CloseWaveDialog.vue'
import type { dto } from '@/../wailsjs/go/models'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

// Injected purely so this component fails loudly (per its own contract) if
// ever mounted outside `WaveWorkspaceShell` — mirrors `WaveAllocationTab.vue`'s
// same dual-injection pattern alongside `useClosureTab()`'s own internal
// `useWaveWorkspaceContext()` call.
const ctx = useWaveWorkspaceContext()
const closure = useClosureTab()

// ── Health summary ──
const healthCards = computed(() => {
  const overview = closure.overview.value
  if (!overview) return []
  return [
    { key: 'jobCount', label: t('waveWorkspace.closure.health.jobCount'), value: overview.channelSyncJobCount, tone: 'neutral' as const },
    { key: 'successCount', label: t('waveWorkspace.closure.health.successCount'), value: overview.channelSyncSuccessCount, tone: 'success' as const },
    { key: 'runningCount', label: t('waveWorkspace.closure.health.runningCount'), value: overview.channelSyncRunningCount, tone: 'progress' as const },
    { key: 'failedCount', label: t('waveWorkspace.closure.health.failedCount'), value: overview.channelSyncFailedCount, tone: (overview.channelSyncFailedCount > 0 ? 'error' : 'neutral') as const },
    { key: 'manualCandidateCount', label: t('waveWorkspace.closure.health.manualCandidateCount'), value: overview.manualClosureCandidateCount, tone: (overview.manualClosureCandidateCount > 0 ? 'warning' : 'neutral') as const },
    { key: 'manualCompletedCount', label: t('waveWorkspace.closure.health.manualCompletedCount'), value: overview.manualCompletedCount, tone: 'success' as const },
    { key: 'manualUnsupportedCount', label: t('waveWorkspace.closure.health.manualUnsupportedCount'), value: overview.manualUnsupportedCount, tone: 'neutral' as const },
    { key: 'manualSkippedCount', label: t('waveWorkspace.closure.health.manualSkippedCount'), value: overview.manualSkippedCount, tone: 'neutral' as const },
  ]
})

// ── Plan (generate backfill file) ──
const profileOptions = computed<SelectOption[]>(() =>
  closure.profiles.value.map((profile) => ({ label: profile.profileKey, value: profile.id })),
)

const planDecisionMessage = computed<string | null>(() => {
  const result = closure.planResult.value
  if (!result) return null
  switch (result.decision) {
    case 'create_job':
      return t('waveWorkspace.closure.plan.decisionAutoJob')
    case 'manual_closure':
      return t('waveWorkspace.closure.plan.decisionManual')
    case 'unsupported':
      return t('waveWorkspace.closure.plan.decisionUnsupported')
    default:
      return null
  }
})

const showManualForm = computed(
  () => closure.planResult.value != null && (closure.planResult.value.decision === 'manual_closure' || closure.planResult.value.decision === 'unsupported'),
)

async function handleGenerate(): Promise<void> {
  try {
    await closure.runPlan()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

async function handleManualSubmitted(): Promise<void> {
  // The manual decision just recorded projects into `manualCompleted/Unsupported/SkippedCount`
  // on the next overview fetch — refresh both the workspace snapshot and
  // the jobs table so all three surfaces (health cards, plan callout, jobs
  // table) stay consistent.
  await Promise.all([ctx.refresh(), closure.loadJobs()])
}

// ── Jobs table ──
const jobRows = computed<ClosureJobRow[]>(() => {
  const profileKeyById = new Map(closure.profiles.value.map((profile) => [profile.id, profile.profileKey]))
  return closure.jobs.value.map((job) => ({
    ...job,
    profileLabel: profileKeyById.get(job.integrationProfileId) ?? `#${job.integrationProfileId}`,
    outputFilePath: parseOutputFilePath(job.responsePayload),
  }))
})

const viewingJob = ref<dto.ChannelSyncJobDTO | null>(null)
const showItemsDrawer = ref(false)

const jobColumns = computed(() =>
  createColumns(
    buildClosureJobColumns(t, {
      isRunning: (jobId) => closure.executingJobIds.value.has(jobId),
      isRetrying: (jobId) => closure.retryingJobIds.value.has(jobId),
      onRun: (row) => void handleRunJob(row.id),
      onRetry: (row) => void handleRetryJob(row.id),
      onViewItems: (row) => {
        viewingJob.value = row
        showItemsDrawer.value = true
      },
      onOpenFolder: (path) => void handleOpenFolder(path),
    }),
  ),
)

async function handleRunJob(jobId: number): Promise<void> {
  try {
    await closure.runJob(jobId)
    feedback.success(t('waveWorkspace.closure.jobs.actions.runSuccess'))
  } catch (err) {
    feedback.error(t('waveWorkspace.closure.jobs.actions.runError'), err instanceof Error ? err.message : String(err))
  }
}

async function handleRetryJob(jobId: number): Promise<void> {
  try {
    await closure.retryJob(jobId)
    feedback.success(t('waveWorkspace.closure.jobs.actions.retrySuccess'))
  } catch (err) {
    feedback.error(t('waveWorkspace.closure.jobs.actions.retryError'), err instanceof Error ? err.message : String(err))
  }
}

async function handleOpenFolder(path: string): Promise<void> {
  try {
    await revealInFolder(path)
  } catch (err) {
    feedback.error(t('waveWorkspace.factory.generateFile.openFolderError'), err instanceof Error ? err.message : String(err))
  }
}

async function handleManualRefresh(): Promise<void> {
  await closure.loadJobs()
}

// ── Close wave ──
// Known `WaveOverviewDTO.blockingIssues` codes (`buildBlockingIssues` in
// `internal/app/wave_overview_query_usecase.go:717`) — an imprecise but
// server-computed proxy for "this wave probably isn't ready to close
// without force" (see `useClosureTab.ts`'s module doc / GAP 4). The
// authoritative check is `CloseWave`'s own residual-count computation,
// surfaced by `CloseWaveDialog` itself on an unforced attempt.
const KNOWN_BLOCKING_ISSUE_CODES = new Set([
  'address_missing',
  'basis_drifted',
  'review_required',
  'mapping_blocked',
  'replay_failures_detected',
])

const blockingIssueMessages = computed<string[]>(() => {
  const issues = closure.overview.value?.blockingIssues ?? []
  return issues.map((code) =>
    t(`waveWorkspace.overview.blockingIssues.${KNOWN_BLOCKING_ISSUE_CODES.has(code) ? code : 'unknown'}`),
  )
})

const showCloseDialog = ref(false)

function handleClosed(_result: dto.CloseWaveResult): void {
  void ctx.refresh()
}
</script>

<template>
  <div class="wave-closure-tab">
    <SectionCard :title="t('waveWorkspace.closure.title')" :description="t('waveWorkspace.closure.subtitle')">
      <div class="wave-closure-tab__health-grid">
        <StatCard v-for="card in healthCards" :key="card.key" :label="card.label" :value="String(card.value)" :tone="card.tone" />
      </div>
    </SectionCard>

    <CalloutBar
      v-if="ctx.undoBoundaryCrossed.value"
      tone="info"
      :message="t('waveWorkspace.header.undoBoundaryNotice')"
    />

    <SectionCard :title="t('waveWorkspace.closure.plan.title')">
      <div class="wave-closure-tab__plan">
        <div class="wave-closure-tab__plan-row">
          <NSelect
            v-model:value="closure.selectedProfileId.value"
            :options="profileOptions"
            :loading="closure.loadingProfiles.value"
            :placeholder="t('waveWorkspace.closure.plan.profilePlaceholder')"
            style="max-width: 360px"
          />
          <NButton type="primary" :loading="closure.planning.value" :disabled="closure.selectedProfileId.value == null" @click="handleGenerate">
            {{ t('waveWorkspace.closure.plan.action') }}
          </NButton>
        </div>

        <div v-if="closure.selectedProfile.value" class="wave-closure-tab__profile-tags">
          <span class="wave-closure-tab__profile-tag-label">{{ t('waveWorkspace.closure.plan.trackingSyncMode') }}</span>
          <StatusBadge dimension="trackingSyncMode" :value="closure.selectedProfile.value.trackingSyncMode" size="sm" />
          <span class="wave-closure-tab__profile-tag-label">{{ t('waveWorkspace.closure.plan.closurePolicy') }}</span>
          <StatusBadge dimension="closurePolicy" :value="closure.selectedProfile.value.closurePolicy" size="sm" />
          <span class="wave-closure-tab__profile-tag-label">{{ t('waveWorkspace.closure.plan.allowsManualClosure') }}</span>
          <span class="wave-closure-tab__profile-tag-bool">
            {{ closure.selectedProfile.value.allowsManualClosure ? t('common.yes') : t('common.no') }}
          </span>
        </div>

        <CalloutBar v-if="planDecisionMessage" tone="info" :message="planDecisionMessage" />
      </div>
    </SectionCard>

    <ManualClosureForm
      v-if="showManualForm"
      :wave-id="closure.waveId.value"
      :profile="closure.selectedProfile.value"
      :items="closure.planResult.value?.items ?? []"
      @submitted="handleManualSubmitted"
    />

    <SectionCard :title="t('waveWorkspace.closure.jobs.title')">
      <template #actions>
        <span v-if="closure.hasInFlightJobs.value" class="wave-closure-tab__poll-hint">
          {{ t('waveWorkspace.closure.jobs.autoRefreshHint', { seconds: POLL_INTERVAL_MS / 1000 }) }}
        </span>
        <NButton size="small" :loading="closure.loadingJobs.value" @click="handleManualRefresh">
          {{ t('waveWorkspace.closure.jobs.refresh') }}
        </NButton>
      </template>

      <DataGrid
        :columns="jobColumns"
        :rows="jobRows"
        row-key="id"
        :loading="closure.loadingJobs.value"
        pagination="client"
        :empty="{ title: t('waveWorkspace.closure.jobs.empty') }"
      />
    </SectionCard>

    <SectionCard :title="t('waveWorkspace.closure.closeWave.title')" :description="t('waveWorkspace.closure.closeWave.description')">
      <div class="wave-closure-tab__close">
        <CalloutBar
          v-for="(message, index) in blockingIssueMessages"
          :key="index"
          tone="warning"
          :message="message"
        />
        <NButton type="error" :disabled="!ctx.snapshot.value" @click="showCloseDialog = true">
          {{ t('waveWorkspace.closure.closeWave.action') }}
        </NButton>
      </div>
    </SectionCard>

    <CloseWaveDialog
      v-if="ctx.snapshot.value"
      v-model:show="showCloseDialog"
      :wave="ctx.snapshot.value.wave"
      @closed="handleClosed"
    />

    <JobItemsDrawer v-model:show="showItemsDrawer" :job="viewingJob" />
  </div>
</template>

<style scoped>
.wave-closure-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-closure-tab__health-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: var(--space-3);
}

.wave-closure-tab__plan {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.wave-closure-tab__plan-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.wave-closure-tab__profile-tags {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.wave-closure-tab__profile-tag-label {
  color: var(--color-text-muted);
}

.wave-closure-tab__profile-tag-bool {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.wave-closure-tab__poll-hint {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-closure-tab__close {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  align-items: flex-start;
}
</style>
