/**
 * useClosureTab — data/action composable for `WaveClosureTab.vue` (P5,
 * plan §3.3.4 third bullet "回填收尾 + 关闭波次", route
 * `wave-workspace-closure`). Owns:
 *
 * - The integration-profile picker + `planChannelClosure` run (decides
 *   whether the profile auto-creates a job, needs a manual closure form, or
 *   is unsupported).
 * - The channel-sync job list (`listChannelSyncJobsByWave`) + execute/retry
 *   actions, with auto-poll while any job is `pending`/`running` (plan's
 *   "运行中任务自动轮询 + 手动刷新" requirement) plus a manual refresh.
 * - `parseOutputFilePath` — the client-side JSON.parse of
 *   `ChannelSyncJobDTO.responsePayload`/`ExecuteSyncResult.responsePayload`
 *   to recover the `output_file` key. This key is NOT a typed DTO field —
 *   both `internal/app/document_export_executor.go:118-123` and
 *   `internal/app/csv_export_executor.go:91-97` embed it as plain JSON
 *   inside the payload string, using the identical key name today. This is
 *   an implicit executor-internal contract, not a typed one — a future
 *   executor author must keep emitting `output_file` for this UI's output-
 *   path column (and the "open containing folder" action) to keep working.
 *
 * Manual closure decisions themselves are NOT owned here — see
 * `ManualClosureForm.vue`, which is self-contained (it only needs the
 * current plan's `items`/`profile`/`waveId`, and calls
 * `recordChannelClosureDecision` directly, mirroring
 * `allocation/RuleEditor.vue`'s "child component owns its own submit"
 * convention).
 */
import { computed, onBeforeUnmount, onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import {
  listIntegrationProfiles,
  listChannelSyncJobsByWave,
  planChannelClosure,
  executeChannelSyncJob,
  retryChannelSyncJob,
} from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type { dto } from '@/../wailsjs/go/models'

/** In-flight statuses that keep the auto-poll timer alive. */
const POLLABLE_STATUSES = new Set(['pending', 'running'])

/**
 * Auto-poll cadence while any job is pending/running (plan: "每 N 秒自动刷新").
 * Exported so `WaveClosureTab.vue` can interpolate the real cadence into
 * `jobs.autoRefreshHint`'s `{seconds}` placeholder instead of a duplicated
 * magic number.
 */
export const POLL_INTERVAL_MS = 4000

export interface UseClosureTabApi {
  waveId: ComputedRef<number>
  wave: ComputedRef<dto.WaveDTO | null>
  overview: ComputedRef<dto.WaveOverviewDTO | null>

  profiles: Ref<dto.IntegrationProfileSummaryDTO[]>
  loadingProfiles: Ref<boolean>
  selectedProfileId: Ref<number | null>
  selectedProfile: ComputedRef<dto.IntegrationProfileSummaryDTO | null>

  planning: Ref<boolean>
  planResult: Ref<dto.PlanChannelClosureResult | null>
  runPlan(): Promise<void>

  jobs: Ref<dto.ChannelSyncJobDTO[]>
  loadingJobs: Ref<boolean>
  hasInFlightJobs: ComputedRef<boolean>
  loadJobs(): Promise<void>

  /** Job ids currently mid-`executeChannelSyncJob` call (drives per-row button spinners). */
  executingJobIds: Ref<Set<number>>
  /** Job ids currently mid-`retryChannelSyncJob` call. */
  retryingJobIds: Ref<Set<number>>
  runJob(jobId: number): Promise<void>
  retryJob(jobId: number): Promise<void>

  loadAll(): Promise<void>
}

/** `JSON.parse(payload).output_file` with a defensive fallback — never throws. */
export function parseOutputFilePath(responsePayload: string | null | undefined): string | null {
  if (!responsePayload) return null
  try {
    const parsed: unknown = JSON.parse(responsePayload)
    if (parsed && typeof parsed === 'object' && 'output_file' in parsed) {
      const value = (parsed as Record<string, unknown>).output_file
      return typeof value === 'string' && value.length > 0 ? value : null
    }
    return null
  } catch {
    return null
  }
}

export function useClosureTab(): UseClosureTabApi {
  const ctx = useWaveWorkspaceContext()

  const wave = computed<dto.WaveDTO | null>(() => ctx.snapshot.value?.wave ?? null)
  const overview = computed<dto.WaveOverviewDTO | null>(() => ctx.snapshot.value?.overview ?? null)

  const profiles = ref<dto.IntegrationProfileSummaryDTO[]>([]) as Ref<dto.IntegrationProfileSummaryDTO[]>
  const loadingProfiles = ref(false)
  const selectedProfileId = ref<number | null>(null)

  const selectedProfile = computed<dto.IntegrationProfileSummaryDTO | null>(
    () => profiles.value.find((profile) => profile.id === selectedProfileId.value) ?? null,
  )

  const planning = ref(false)
  const planResult = ref<dto.PlanChannelClosureResult | null>(null) as Ref<dto.PlanChannelClosureResult | null>

  const jobs = ref<dto.ChannelSyncJobDTO[]>([]) as Ref<dto.ChannelSyncJobDTO[]>
  const loadingJobs = ref(false)
  const executingJobIds = ref(new Set<number>()) as Ref<Set<number>>
  const retryingJobIds = ref(new Set<number>()) as Ref<Set<number>>

  const hasInFlightJobs = computed(() => jobs.value.some((job) => POLLABLE_STATUSES.has(job.status)))

  async function loadProfiles(): Promise<void> {
    loadingProfiles.value = true
    try {
      profiles.value = await listIntegrationProfiles()
      if (selectedProfileId.value == null && profiles.value.length > 0) {
        selectedProfileId.value = profiles.value[0]!.id
      }
    } finally {
      loadingProfiles.value = false
    }
  }

  async function loadJobs(): Promise<void> {
    loadingJobs.value = true
    try {
      jobs.value = await listChannelSyncJobsByWave(ctx.waveId.value)
    } finally {
      loadingJobs.value = false
    }
  }

  async function loadAll(): Promise<void> {
    await Promise.all([loadProfiles(), loadJobs()])
  }

  async function runPlan(): Promise<void> {
    if (selectedProfileId.value == null) return
    planning.value = true
    try {
      const result = await planChannelClosure({
        waveId: ctx.waveId.value,
        integrationProfileId: selectedProfileId.value,
      })
      planResult.value = result
      // The `create_job` branch already persisted a job — refresh the jobs
      // table (+ auto-poll picks up from here) so it shows up immediately.
      await loadJobs()
    } finally {
      planning.value = false
      await ctx.refresh()
    }
  }

  async function runJob(jobId: number): Promise<void> {
    executingJobIds.value = new Set(executingJobIds.value).add(jobId)
    try {
      await executeChannelSyncJob(jobId)
      await loadJobs()
    } finally {
      const next = new Set(executingJobIds.value)
      next.delete(jobId)
      executingJobIds.value = next
      await ctx.refresh()
    }
  }

  async function retryJob(jobId: number): Promise<void> {
    retryingJobIds.value = new Set(retryingJobIds.value).add(jobId)
    try {
      await retryChannelSyncJob(jobId)
      await loadJobs()
    } finally {
      const next = new Set(retryingJobIds.value)
      next.delete(jobId)
      retryingJobIds.value = next
      await ctx.refresh()
    }
  }

  // ── Auto-poll while any job is pending/running ──
  let pollTimer: ReturnType<typeof setInterval> | null = null

  function ensurePolling(): void {
    if (hasInFlightJobs.value) {
      if (pollTimer == null) {
        pollTimer = setInterval(() => void loadJobs(), POLL_INTERVAL_MS)
      }
    } else if (pollTimer != null) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  watch(jobs, () => ensurePolling())

  onBeforeUnmount(() => {
    if (pollTimer != null) clearInterval(pollTimer)
  })

  // Cross-wave deep link (workspace shell stays mounted, only `route.params.id` changes).
  watch(ctx.waveId, () => {
    planResult.value = null
    void loadAll()
  })

  onMounted(() => {
    void loadAll()
  })

  return {
    waveId: ctx.waveId,
    wave,
    overview,
    profiles,
    loadingProfiles,
    selectedProfileId,
    selectedProfile,
    planning,
    planResult,
    runPlan,
    jobs,
    loadingJobs,
    hasInFlightJobs,
    loadJobs,
    executingJobIds,
    retryingJobIds,
    runJob,
    retryJob,
    loadAll,
  }
}
