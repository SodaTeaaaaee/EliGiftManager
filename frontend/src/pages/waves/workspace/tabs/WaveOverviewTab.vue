<script setup lang="ts">
/**
 * WaveOverviewTab — the wave-workspace's default (empty-path, route name
 * `wave-workspace`) tab: the doc-mandated "explainable funnel" (plan 3.3.1),
 * the single "suggested next step" guidance card, the six action-center
 * buckets scoped to this wave, the projected basis-drift summary, and the
 * blocking-issues list. Reads everything from the injected
 * `useWaveWorkspaceContext()` — never fetches on its own (P2 foundations
 * contract: `WaveWorkspaceShell` is the sole owner of the fetch lifecycle).
 */
import { computed, ref } from 'vue'
import { useRouter, type RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { FunnelBar, type FunnelStage } from '@/shared/ui/funnel'
import { GuidanceCard, CalloutBar } from '@/shared/ui/guidance'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { EmptyState } from '@/shared/ui/empty-state'
import { serializeEnumMultiQuery } from '@/shared/ui/filter-bar/useUrlFilters'
import { buildWaveFilterLink } from '@/shared/lib/wave-filter-link'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import {
  routeForStep,
  routeNameForStep,
  STEP_LABEL_KEY,
  type RouteStepKey,
} from '@/shared/lib/wave-workspace/step-keys'
import type { StatusTone } from '@/shared/i18n/glossary'
import type { dto } from '@/../wailsjs/go/models'
import WaveDriftDrawer from '../WaveDriftDrawer.vue'
import { BUCKET_LABEL_KEYS, BUCKET_TONES } from './wave-overview-buckets'

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const ctx = useWaveWorkspaceContext()

const overview = computed<dto.WaveOverviewDTO | undefined>(() => ctx.snapshot.value?.overview)

// ── Funnel (plan 3.3.1 CANON) ──
const funnelStages = computed<FunnelStage[]>(() => {
  const o = overview.value
  if (!o) return []
  return [
    { key: 'totalLines', labelKey: 'overview.funnel.totalLines', count: o.fulfillmentCount },
    { key: 'addressReady', labelKey: 'overview.funnel.addressReady', count: o.addressReadyCount, tone: 'success' },
    {
      key: 'submittedToFactory',
      labelKey: 'overview.funnel.submittedToFactory',
      count: o.supplierSubmittedCount,
      tone: 'progress',
    },
    { key: 'tracked', labelKey: 'overview.funnel.tracked', count: o.trackedFulfillmentCount, tone: 'progress' },
    {
      key: 'backfilled',
      labelKey: 'overview.funnel.backfilled',
      count: o.channelSyncSuccessCount + o.channelSyncPartialSuccessCount,
      tone: 'success',
    },
    {
      key: 'manualClosureOrFailed',
      labelKey: 'overview.funnel.manualClosureOrFailed',
      count: o.manualClosureCandidateCount + o.channelSyncFailedCount,
      tone: 'warning',
    },
  ]
})

// Only `addressReady` and `submittedToFactory` map onto a fulfillment-line-
// level enum the future grid can filter by (`AddressState`/`SupplierState`,
// matching wave_overview_query_usecase.go's per-line switch exactly).
// `tracked` (shipment-tracking derived), `backfilled` and
// `manualClosureOrFailed` (channel-sync JOB status + manual-closure
// candidate counts — a different level than the per-line filter dimensions)
// have no clean 1:1 filter mapping, so their segments stay non-navigating
// rather than pushing a wrong/misleading query.
const FUNNEL_STAGE_FILTERS: Partial<Record<string, { queryKey: string; values: string[] }>> = {
  addressReady: { queryKey: 'addressStates', values: ['ready'] },
  submittedToFactory: {
    queryKey: 'supplierStates',
    values: ['submitted', 'accepted', 'producing', 'partially_shipped'],
  },
}

function handleStageClick(key: string): void {
  const mapping = FUNNEL_STAGE_FILTERS[key]
  if (!mapping) return
  const serialized = serializeEnumMultiQuery(mapping.values)
  if (serialized === undefined) return
  void router.push({
    name: 'wave-workspace',
    params: { id: ctx.waveId.value },
    query: { [mapping.queryKey]: serialized },
  })
}

// ── Guidance card ──
const showGuidance = computed(() => !!overview.value && overview.value.suggestedNextStep !== 'wave_overview')

const guidanceReason = computed(() => (overview.value ? t('overview.guidance.' + overview.value.nextStepReason) : ''))

const guidanceTargetStep = computed<RouteStepKey | undefined>(() => {
  const step = overview.value?.suggestedNextStep
  return step ? routeForStep(step) : undefined
})

const guidanceCtaLabel = computed(() => {
  const target = guidanceTargetStep.value
  return target !== undefined ? t(STEP_LABEL_KEY[target]) : ''
})

const guidanceCtaTarget = computed<RouteLocationRaw>(() => ({
  name: routeNameForStep(guidanceTargetStep.value ?? ''),
  params: { id: ctx.waveId.value },
}))

// ── Six action-center buckets ──
const sixBuckets = computed<dto.ActionCenterWaveBucketDTO[]>(() => ctx.sixBuckets.value)

function bucketLabel(bucket: dto.ActionCenterWaveBucketDTO): string {
  const suffix = BUCKET_LABEL_KEYS[bucket.bucketKind]
  return suffix ? t('taskCenter.buckets.' + suffix) : bucket.bucketKind
}

function bucketTone(bucket: dto.ActionCenterWaveBucketDTO): StatusTone {
  return BUCKET_TONES[bucket.bucketKind] ?? 'neutral'
}

function handleBucketClick(bucket: dto.ActionCenterWaveBucketDTO): void {
  void router.push(buildWaveFilterLink(ctx.waveId.value, bucket.filter))
}

// ── Drift summary + drawer ──
const showDriftDrawer = ref(false)
// Wails may deliver Go nil slices as JSON null (no omitempty on the DTO);
// optional-chain through the array itself so a null list never TypeErrors.
const driftSignalCount = computed(() => overview.value?.basisDriftSignals?.length ?? 0)

// ── Blocking issues ──
const KNOWN_BLOCKING_ISSUES = new Set([
  'address_missing',
  'basis_drifted',
  'review_required',
  'mapping_blocked',
  'replay_failures_detected',
])

function blockingIssueMessage(code: string): string {
  const key = KNOWN_BLOCKING_ISSUES.has(code) ? code : 'unknown'
  return t('overview.blockingIssues.' + key)
}
</script>

<template>
  <div class="wave-overview-tab">
    <GuidanceCard v-if="showGuidance" :title="t('overview.suggestedNext.title')" :reason="guidanceReason">
      <template #primary>
        <RouterLink :to="guidanceCtaTarget" class="wave-overview-tab__guidance-cta">
          {{ guidanceCtaLabel }}
        </RouterLink>
      </template>
    </GuidanceCard>

    <div v-if="overview && (overview.blockingIssues?.length ?? 0) > 0" class="wave-overview-tab__blocking-issues">
      <CalloutBar v-for="code in overview.blockingIssues ?? []" :key="code" tone="warning" :message="blockingIssueMessage(code)" />
    </div>

    <SectionCard :title="t('overview.sections.funnel')">
      <FunnelBar :stages="funnelStages" @stage-click="handleStageClick" />
    </SectionCard>

    <SectionCard :title="t('overview.sections.buckets')">
      <EmptyState v-if="sixBuckets.length === 0" size="sm" :title="t('taskCenter.actionStream.empty')" />
      <div v-else class="wave-overview-tab__buckets">
        <StatCard
          v-for="bucket in sixBuckets"
          :key="bucket.bucketKind"
          :label="bucketLabel(bucket)"
          :value="String(bucket.count)"
          :tone="bucketTone(bucket)"
          clickable
          @click="handleBucketClick(bucket)"
        />
      </div>
    </SectionCard>

    <SectionCard :title="t('overview.sections.drift')">
      <button type="button" class="wave-overview-tab__drift-trigger" @click="showDriftDrawer = true">
        <StatusBadge dimension="driftSummary" :value="ctx.driftSummaryValue.value" show-dot />
        <span class="wave-overview-tab__drift-message">
          {{ t('overview.driftCallout.message', { count: driftSignalCount }) }}
        </span>
        <span class="wave-overview-tab__drift-action">{{ t('overview.driftCallout.action') }}</span>
      </button>
    </SectionCard>

    <WaveDriftDrawer v-model:show="showDriftDrawer" />
  </div>
</template>

<style scoped>
.wave-overview-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-overview-tab__blocking-issues {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-overview-tab__guidance-cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--control-height);
  padding: 0 var(--space-4);
  border-radius: var(--control-radius);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  text-decoration: none;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.wave-overview-tab__guidance-cta:hover {
  background: var(--color-accent-hover);
}

.wave-overview-tab__guidance-cta:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.wave-overview-tab__buckets {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-3);
}

.wave-overview-tab__drift-trigger {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2) 0;
  border: none;
  background: transparent;
  text-align: left;
  cursor: pointer;
  font-family: var(--font-body);
}

.wave-overview-tab__drift-trigger:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.wave-overview-tab__drift-message {
  flex: 1 1 auto;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.wave-overview-tab__drift-action {
  flex-shrink: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-accent);
  text-decoration: underline;
  text-underline-offset: 2px;
}
</style>
