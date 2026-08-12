<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSpin } from 'naive-ui'
import { EmptyState } from '@/shared/ui/empty-state'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { CandidateStatusBadge, MergeEvidenceList } from '@/shared/ui/customer-resolution'
import {
  buildCandidateViewModel,
  classifyCandidateDismissError,
  type MergeCandidateStatus,
  type MergeEvidenceInput,
  type MergeCandidateViewModel,
} from '@/shared/lib/customer-resolution'
import {
  dismissMergeCandidate,
  getCustomerProfile,
  getMergeCandidate,
  listMergeCandidates,
  scanMergeCandidates,
} from '@/shared/api/bridge'
import type { MergeCandidateDTO, MergeScanRunDTO } from '@/entities/merge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'

interface CandidateRow {
  candidate: MergeCandidateDTO
  sourceName: string
  targetName: string
  presentation: MergeCandidateViewModel
  loadWarnings: string[]
}

const emit = defineEmits<{
  preview: [{ candidateId: number; sourceProfileId: number; targetProfileId: number }]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()
const candidateWriteRejected = ref(false)
const candidateWritesEnabled = computed(
  () => !candidateWriteRejected.value && featurePolicy.isEnabled('candidateScanEnabled'),
)

const rows = ref<CandidateRow[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)
const scanning = ref(false)
const lastScan = ref<MergeScanRunDTO | null>(null)
const pendingDismiss = ref<CandidateRow | null>(null)
const dismissing = ref(false)

function normalizeStatus(status: string): MergeCandidateStatus {
  if (status === 'executed') return 'merged'
  const supported: MergeCandidateStatus[] = [
    'pending', 'reviewing', 'blocked', 'stale', 'dismissed', 'superseded', 'expired', 'executing', 'merged', 'failed',
  ]
  return supported.includes(status as MergeCandidateStatus) ? status as MergeCandidateStatus : 'failed'
}

function evidenceInput(candidate: MergeCandidateDTO): MergeEvidenceInput[] {
  const evidence = (candidate.evidence ?? []).map((item) => ({
    id: String(item.id),
    type: item.evidenceKind,
    polarity: item.polarity === 'blocker' || item.polarity === 'negative' ? item.polarity : 'positive',
    strength: item.polarity === 'blocker' ? 'hard' : item.confidence >= 0.8 ? 'strong' : item.confidence >= 0.5 ? 'medium' : 'weak',
    explanationCode: item.explanationCode,
    maskedValue: item.maskedValue,
    observedAt: item.observedAt,
  } satisfies MergeEvidenceInput))
  const evidenceCodes = new Set(evidence.map((item) => item.explanationCode))
  for (const code of candidate.blockerCodes ?? []) {
    if (!evidenceCodes.has(code)) {
      evidence.push({ id: `blocker-${code}`, type: 'blocker', polarity: 'blocker', strength: 'hard', explanationCode: code })
    }
  }
  return evidence
}

async function refetch(): Promise<void> {
  loading.value = true
  loadError.value = null
  try {
    const listed = (await listMergeCandidates()).filter((candidate) => candidate.status !== 'dismissed')
    const detailed = await Promise.all(listed.map(async (candidate) => {
      try {
        return { candidate: await getMergeCandidate(candidate.id), detailFailed: false }
      } catch {
        return { candidate, detailFailed: true }
      }
    }))
    const profileIDs = [...new Set(detailed.flatMap(({ candidate }) => [candidate.sourceProfileId, candidate.targetProfileId]))]
    const profiles = await Promise.all(profileIDs.map(async (id) => {
      try {
        return { id, profile: await getCustomerProfile(id), failed: false }
      } catch {
        return { id, profile: null, failed: true }
      }
    }))
    const names = new Map(profiles.filter((item) => item.profile != null).map((item) => [item.id, item.profile!.displayName]))
    const failedProfiles = new Set(profiles.filter((item) => item.failed).map((item) => item.id))
    rows.value = detailed.map(({ candidate, detailFailed }) => ({
      candidate,
      sourceName: names.get(candidate.sourceProfileId) ?? `#${candidate.sourceProfileId}`,
      targetName: names.get(candidate.targetProfileId) ?? `#${candidate.targetProfileId}`,
      presentation: buildCandidateViewModel(candidate.id, normalizeStatus(candidate.status), evidenceInput(candidate)),
      loadWarnings: [
        ...(detailFailed ? [t('suggestedMerges.detailLoadFailed')] : []),
        ...(failedProfiles.has(candidate.sourceProfileId) || failedProfiles.has(candidate.targetProfileId)
          ? [t('suggestedMerges.profileLoadFailed')]
          : []),
      ],
    }))
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function runScan(): Promise<void> {
  if (!candidateWritesEnabled.value) return
  scanning.value = true
  try {
    lastScan.value = await scanMergeCandidates()
    await refetch()
  } catch (err) {
    feedback.error(t('suggestedMerges.scanFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    scanning.value = false
  }
}

function requestPreview(row: CandidateRow): void {
  if (!row.presentation.actions.canPreview) return
  emit('preview', {
    candidateId: row.candidate.id,
    sourceProfileId: row.candidate.sourceProfileId,
    targetProfileId: row.candidate.targetProfileId,
  })
}

async function confirmDismiss(): Promise<void> {
  if (!pendingDismiss.value || !candidateWritesEnabled.value) return
  dismissing.value = true
  try {
    await dismissMergeCandidate({
      id: pendingDismiss.value.candidate.id,
      evidenceHash: pendingDismiss.value.candidate.evidenceHash,
      policyVersion: pendingDismiss.value.candidate.policyVersion,
    })
    pendingDismiss.value = null
    await refetch()
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err)
    const kind = classifyCandidateDismissError(err)
    pendingDismiss.value = null
    if (kind === 'feature_disabled') {
      candidateWriteRejected.value = true
      feedback.error(t('suggestedMerges.dismissDisabled'), detail)
      const refreshed = await featurePolicy.load(true)
      if (refreshed) candidateWriteRejected.value = !featurePolicy.isEnabled('candidateScanEnabled')
      await refetch()
    } else if (kind === 'changed') {
      feedback.error(t('suggestedMerges.dismissChanged'), detail)
      await refetch()
    } else {
      feedback.error(t('suggestedMerges.dismissFailed'), detail)
    }
  } finally {
    dismissing.value = false
  }
}

const statusLabels = computed<Record<MergeCandidateStatus, string>>(() => ({
  pending: t('suggestedMerges.status.pending'), reviewing: t('suggestedMerges.status.reviewing'),
  blocked: t('suggestedMerges.status.blocked'), stale: t('suggestedMerges.status.stale'),
  dismissed: t('suggestedMerges.status.dismissed'), superseded: t('suggestedMerges.status.superseded'),
  expired: t('suggestedMerges.status.expired'), executing: t('suggestedMerges.status.executing'),
  merged: t('suggestedMerges.status.merged'), failed: t('suggestedMerges.status.failed'),
}))

const evidenceLabels = computed<Record<string, string>>(() => ({
  stable_identity_match: t('suggestedMerges.evidence.stableIdentityMatch'),
  stable_identity_conflict: t('suggestedMerges.evidence.stableIdentityConflict'),
  verified_email_match: t('suggestedMerges.evidence.verifiedEmailMatch'),
  legacy_email_exact: t('suggestedMerges.evidence.legacyEmailExact'),
  normalized_phone_match: t('suggestedMerges.evidence.normalizedPhoneMatch'),
  legacy_phone_exact: t('suggestedMerges.evidence.legacyPhoneExact'),
  address_fingerprint_match: t('suggestedMerges.evidence.addressFingerprintMatch'),
}))

onMounted(() => {
  void featurePolicy.load()
  void refetch()
})
defineExpose({ refetch })
</script>

<template>
  <div class="suggested-merges">
    <header class="suggested-merges__header">
      <div>
        <h3 class="suggested-merges__title">{{ t('suggestedMerges.title') }}</h3>
        <p class="suggested-merges__subtitle">{{ t('suggestedMerges.subtitle') }}</p>
      </div>
      <NButton size="small" :loading="scanning" :disabled="!candidateWritesEnabled" @click="runScan">{{ t('suggestedMerges.scanAction') }}</NButton>
    </header>

    <p v-if="lastScan" class="suggested-merges__scan-result">
      {{ t('suggestedMerges.scanResult', { created: lastScan.candidatesCreated, updated: lastScan.candidatesUpdated, blocked: lastScan.candidatesBlocked }) }}
    </p>
    <p v-if="!candidateWritesEnabled" class="suggested-merges__write-disabled">
      {{ t('suggestedMerges.dismissDisabled') }}
    </p>
    <ErrorBanner
      v-if="loadError"
      :message="t('suggestedMerges.loadFailed')"
      :detail="loadError"
      @retry="refetch"
    />
    <div v-if="loading" class="suggested-merges__loading"><NSpin size="small" /></div>
    <EmptyState v-else-if="rows.length === 0" :title="t('suggestedMerges.empty')" size="sm" />

    <ul v-else class="suggested-merges__list">
      <li v-for="row in rows" :key="row.candidate.id" class="suggested-merges__item">
        <div class="suggested-merges__profiles">
          <span>{{ row.sourceName }}</span><span aria-hidden="true">→</span><span>{{ row.targetName }}</span>
          <CandidateStatusBadge
            :status="row.presentation.status"
            :label="statusLabels[row.presentation.status]"
          />
          <span class="suggested-merges__confidence">{{ Math.round(row.candidate.confidence * 100) }}%</span>
        </div>
        <MergeEvidenceList
          :evidence="row.presentation.evidence"
          :explanation-labels="evidenceLabels"
          :polarity-labels="{
            positive: t('suggestedMerges.polarity.positive'),
            negative: t('suggestedMerges.polarity.negative'),
            blocker: t('suggestedMerges.polarity.blocker'),
          }"
          :empty-label="t('suggestedMerges.evidenceEmpty')"
        />
        <p v-for="warning in row.loadWarnings" :key="warning" class="suggested-merges__load-warning">
          {{ warning }}
        </p>
        <div class="suggested-merges__actions">
          <NButton
            size="small"
            type="primary"
            :disabled="!row.presentation.actions.canPreview"
            @click="requestPreview(row)"
          >
            {{ row.presentation.isStale ? t('suggestedMerges.staleAction') : t('suggestedMerges.previewAction') }}
          </NButton>
          <NButton
            v-if="row.presentation.actions.canDismiss"
            size="small"
            quaternary
            :disabled="!candidateWritesEnabled"
            @click="pendingDismiss = row"
          >
            {{ t('suggestedMerges.dismissAction') }}
          </NButton>
        </div>
      </li>
    </ul>

    <NModal
      :show="pendingDismiss != null"
      preset="dialog"
      type="warning"
      :title="t('suggestedMerges.dismissAction')"
      :content="t('suggestedMerges.dismissConfirmExact')"
      :positive-text="t('common.confirm')"
      :negative-text="t('common.cancel')"
      :loading="dismissing"
      @positive-click="confirmDismiss"
      @negative-click="pendingDismiss = null"
      @update:show="(value: boolean) => { if (!value) pendingDismiss = null }"
    />
  </div>
</template>

<style scoped>
.suggested-merges,
.suggested-merges__list,
.suggested-merges__item {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.suggested-merges__header,
.suggested-merges__profiles,
.suggested-merges__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.suggested-merges__header {
  justify-content: space-between;
}

.suggested-merges__title,
.suggested-merges__subtitle,
.suggested-merges__scan-result {
  margin: 0;
}

.suggested-merges__title {
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
}

.suggested-merges__subtitle,
.suggested-merges__scan-result,
.suggested-merges__confidence,
.suggested-merges__load-warning {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.suggested-merges__write-disabled {
  margin: 0;
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.suggested-merges__load-warning {
  margin: 0;
  color: var(--status-warning-fg);
}

.suggested-merges__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.suggested-merges__item {
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.suggested-merges__profiles {
  flex-wrap: wrap;
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.suggested-merges__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}
</style>
