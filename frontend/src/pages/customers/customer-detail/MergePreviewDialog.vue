<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NRadioButton, NRadioGroup, NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { UndoDryRunPanel } from '@/shared/ui/customer-resolution'
import {
  buildMergeResolutionOptions,
  buildMergeResolutionRequest,
  recommendedMergeResolutionState,
  acceptMergeResolutionPreview,
  acceptMergePreviewSession,
  acceptedMergePreviewIsCurrent,
  beginMergeResolutionPreview,
  captureMergePreviewSession,
  classifyMergeExecutionError,
  createMergeResolutionPreviewFlow,
  invalidateMergeResolutionPreview,
  mergeOperationAccess,
  mergePreviewResponseIsCurrent,
  mergePreviewSessionIsCurrent,
  mergeResolutionExecuteRequest,
  buildUndoDryRun,
  mergeBlockerTranslationKey,
  type MergeResolutionState,
  type AcceptedMergePreviewSession,
  type MergePreviewRequestSession,
} from '@/shared/lib/customer-resolution'
import {
  dryRunCustomerMergeUndo,
  executeCustomerMerge,
  executeCustomerMergeUndo,
  getCustomerProfile,
  previewCustomerMerge,
} from '@/shared/api/bridge'
import type { CustomerProfileDTO } from '@/entities/customer'
import type {
  CustomerMergePreviewResult,
  CustomerMergeUndoDryRunResult,
  ExecuteCustomerMergeResult,
  ExecuteCustomerMergeUndoResult,
} from '@/entities/merge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'

const props = defineProps<{
  show: boolean
  sourceProfileId: number | null
  targetProfileId: number | null
  candidateId?: number | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  merged: [ExecuteCustomerMergeResult]
  undone: [ExecuteCustomerMergeUndoResult]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()

const loading = ref(false)
const loadError = ref<string | null>(null)
const staleNotice = ref(false)
const preview = ref<CustomerMergePreviewResult | null>(null)
const source = ref<CustomerProfileDTO | null>(null)
const target = ref<CustomerProfileDTO | null>(null)
const executing = ref(false)
const receipt = ref<ExecuteCustomerMergeResult | null>(null)
const undoDryRun = ref<CustomerMergeUndoDryRunResult | null>(null)
const undoing = ref(false)
const resolution = ref<MergeResolutionState>({
  primaryIdentitySelections: {},
  defaultAddressId: null,
  displayNameResolution: '',
})
const resolutionFlow = ref(createMergeResolutionPreviewFlow())
const resolutionDirty = computed(() => resolutionFlow.value.dirty)
const mergeWritesEnabled = computed(() => featurePolicy.isEnabled('mergeExecutionEnabled'))
const mergeAccess = computed(() => mergeOperationAccess(mergeWritesEnabled.value))
const acceptedPreviewSession = ref<AcceptedMergePreviewSession | null>(null)
let previewGeneration = 0
let previewRequestSequence = 0

function currentPreviewContext() {
  return {
    sourceProfileId: props.sourceProfileId,
    targetProfileId: props.targetProfileId,
    candidateId: props.candidateId,
    generation: previewGeneration,
  }
}

function previewRequestIsCurrent(
  request: MergePreviewRequestSession,
  result: CustomerMergePreviewResult,
  sourceEntityId: number,
  targetEntityId: number,
): boolean {
  return props.show && mergePreviewResponseIsCurrent(
    request,
    { ...currentPreviewContext(), requestSequence: previewRequestSequence },
    result,
    sourceEntityId,
    targetEntityId,
  )
}

function operationKey(prefix: string): string {
  return `${prefix}-${crypto.randomUUID()}`
}

async function loadPreview(
  sourceId: number,
  targetId: number,
  mode: 'initialize' | 'preserve' = 'initialize',
): Promise<void> {
  const session = captureMergePreviewSession({
    sourceProfileId: sourceId,
    targetProfileId: targetId,
    candidateId: props.candidateId,
    generation: previewGeneration,
  })
  if (!props.show || !session || !mergePreviewSessionIsCurrent(session, currentPreviewContext())) return
  const requestSession: MergePreviewRequestSession = {
    ...session,
    requestSequence: ++previewRequestSequence,
  }
  const currentGroups = buildMergeResolutionOptions(preview.value ?? {}).primaryIdentityGroups
  const attempt = beginMergeResolutionPreview(resolutionFlow.value, resolution.value, currentGroups)
  resolutionFlow.value = attempt.flow
  acceptedPreviewSession.value = null
  loading.value = true
  loadError.value = null
  try {
    let acceptedRequest = attempt.request
    let [nextPreview, nextSource, nextTarget] = await Promise.all([
      previewCustomerMerge({
        sourceProfileId: sourceId,
        targetProfileId: targetId,
        candidateId: props.candidateId ?? undefined,
        ...acceptedRequest,
      }),
      getCustomerProfile(sourceId),
      getCustomerProfile(targetId),
    ])
    if (!previewRequestIsCurrent(requestSession, nextPreview, nextSource.id, nextTarget.id)) return
    if (mode === 'initialize') {
      const recommended = recommendedMergeResolutionState(nextPreview)
      const officialOptions = buildMergeResolutionOptions(nextPreview)
      const recommendedRequest = buildMergeResolutionRequest(recommended, officialOptions.primaryIdentityGroups)
      resolution.value = recommended
      acceptedRequest = recommendedRequest
      if (
        recommendedRequest.primaryIdentitySelections.length > 0
        || recommendedRequest.defaultAddressId != null
        || recommendedRequest.displayNameResolution !== ''
      ) {
        nextPreview = await previewCustomerMerge({
          sourceProfileId: sourceId,
          targetProfileId: targetId,
          candidateId: props.candidateId ?? undefined,
          ...recommendedRequest,
        })
        if (!previewRequestIsCurrent(requestSession, nextPreview, nextSource.id, nextTarget.id)) return
      }
    }
    preview.value = nextPreview
    source.value = nextSource
    target.value = nextTarget
    resolutionFlow.value = acceptMergeResolutionPreview(
      resolutionFlow.value,
      nextPreview.previewToken,
      acceptedRequest,
    )
    acceptedPreviewSession.value = acceptMergePreviewSession(requestSession, nextPreview.previewToken)
    void featurePolicy.load()
  } catch (err) {
    if (!mergePreviewSessionIsCurrent(requestSession, currentPreviewContext())
      || requestSession.requestSequence !== previewRequestSequence) return
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (mergePreviewSessionIsCurrent(requestSession, currentPreviewContext())
      && requestSession.requestSequence === previewRequestSequence) loading.value = false
  }
}

watch(
  () => [props.show, props.sourceProfileId, props.targetProfileId, props.candidateId] as const,
  ([visible, sourceId, targetId]) => {
    previewGeneration += 1
    previewRequestSequence += 1
    receipt.value = null
    undoDryRun.value = null
    staleNotice.value = false
    preview.value = null
    source.value = null
    target.value = null
    loadError.value = null
    loading.value = false
    executing.value = false
    undoing.value = false
    resolution.value = { primaryIdentitySelections: {}, defaultAddressId: null, displayNameResolution: '' }
    resolutionFlow.value = createMergeResolutionPreviewFlow()
    acceptedPreviewSession.value = null
    if (!visible || sourceId == null || targetId == null) return
    void loadPreview(sourceId, targetId)
  },
  { flush: 'sync', immediate: true },
)

const undoPresentation = computed(() => {
  if (!undoDryRun.value) return null
  return buildUndoDryRun({
    mergeId: undoDryRun.value.mergeId,
    eligible: undoDryRun.value.eligible,
    blockers: (undoDryRun.value.blockers ?? []).map((blocker) => ({
      code: blocker.code,
      entityType: blocker.entityType,
      entityId: blocker.entityId,
    })),
    restoreCounts: {
      identities: undoDryRun.value.restoreCounts.identities,
      addresses: undoDryRun.value.restoreCounts.addresses,
      nameObservations: undoDryRun.value.restoreCounts.nameObservations,
      demandDocuments: undoDryRun.value.restoreCounts.demandDocuments,
    },
  })
})

const resolutionOptions = computed(() => buildMergeResolutionOptions(preview.value ?? {}))

const identitySelectOptions = computed<Record<string, SelectOption[]>>(() => Object.fromEntries(
  resolutionOptions.value.primaryIdentityGroups.map((group) => [group.key, group.options.map((identity) => ({
    value: identity.identityId,
    label: `${identity.customerProfileId === preview.value?.sourceProfileId ? t('merge.sourceSide') : t('merge.targetSide')} · ${identity.displayValue}`,
  }))]),
))

const defaultAddressOptions = computed<SelectOption[]>(() => resolutionOptions.value.defaultAddresses.map((address) => ({
  value: address.addressId,
  label: `${address.customerProfileId === preview.value?.sourceProfileId ? t('merge.sourceSide') : t('merge.targetSide')} · ${address.displayValue}`,
})))

function choosePrimary(groupKey: string, identityId: number): void {
  resolution.value.primaryIdentitySelections = {
    ...resolution.value.primaryIdentitySelections,
    [groupKey]: identityId,
  }
  resolutionFlow.value = invalidateMergeResolutionPreview(resolutionFlow.value)
}

function chooseDefaultAddress(addressId: number | null): void {
  resolution.value.defaultAddressId = addressId
  resolutionFlow.value = invalidateMergeResolutionPreview(resolutionFlow.value)
}

function chooseDisplayName(value: string): void {
  resolution.value.displayNameResolution = value === 'keep_source' ? 'keep_source' : 'keep_target'
  resolutionFlow.value = invalidateMergeResolutionPreview(resolutionFlow.value)
}

async function applyResolution(): Promise<void> {
  if (props.sourceProfileId == null || props.targetProfileId == null || loading.value) return
  await loadPreview(props.sourceProfileId, props.targetProfileId, 'preserve')
}

async function retryPreview(): Promise<void> {
  if (props.sourceProfileId == null || props.targetProfileId == null) return
  await loadPreview(
    props.sourceProfileId,
    props.targetProfileId,
    preview.value ? 'preserve' : 'initialize',
  )
}

function blockerLabel(code: string): string {
  const key = mergeBlockerTranslationKey(code)
  return key ? t(key) : t('merge.blocker.unknown', { code })
}

const undoBlockerLabels = computed<Record<string, string>>(() => Object.fromEntries(
  (undoDryRun.value?.blockers ?? []).map((blocker) => [blocker.code, blockerLabel(blocker.code)]),
))

function close(): void {
  if (executing.value || undoing.value) return
  const merged = receipt.value
  emit('update:show', false)
  if (merged) emit('merged', merged)
}

async function executeMerge(): Promise<void> {
  if (!preview.value || !preview.value.canExecute || executing.value || !mergeAccess.value.canExecuteMerge) return
  const session = acceptedPreviewSession.value
  if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), preview.value)) return
  const request = mergeResolutionExecuteRequest(resolutionFlow.value, preview.value.previewToken)
  if (!request) return
  const acceptedPreview = preview.value
  executing.value = true
  try {
    const result = await executeCustomerMerge({
      operationKey: operationKey('merge-ui'),
      previewToken: acceptedPreview.previewToken,
      sourceProfileId: acceptedPreview.sourceProfileId,
      targetProfileId: acceptedPreview.targetProfileId,
      expectedSourceRowVersion: acceptedPreview.sourceRowVersion,
      expectedTargetRowVersion: acceptedPreview.targetRowVersion,
      candidateId: acceptedPreview.candidateId ?? undefined,
      expectedCandidateRowVersion: acceptedPreview.candidateRowVersion,
      expectedEvidenceHash: acceptedPreview.evidenceHash,
      expectedPolicyVersion: acceptedPreview.policyVersion,
      expectedPolicyRevisionId: acceptedPreview.policyRevisionId ?? undefined,
      ...request,
      actorRef: 'local_user',
      decisionReason: acceptedPreview.candidateId ? 'reviewed merge candidate' : 'manual merge preview',
    })
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
    receipt.value = result
  } catch (err) {
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
    const errorKind = classifyMergeExecutionError(err)
    const detail = err instanceof Error ? err.message : String(err)
    if (errorKind === 'stale') {
      staleNotice.value = true
      feedback.error(t('merge.staleError'), detail)
      if (props.sourceProfileId != null && props.targetProfileId != null) {
        await loadPreview(props.sourceProfileId, props.targetProfileId, 'preserve')
      }
    } else if (errorKind === 'feature_disabled') {
      feedback.error(t('merge.disabledReason'), detail)
      await featurePolicy.load(true)
    } else {
      feedback.error(t('merge.executeError'), detail)
    }
  } finally {
    if (mergePreviewSessionIsCurrent(session, currentPreviewContext())) executing.value = false
  }
}

async function requestUndoDryRun(): Promise<void> {
  if (!receipt.value || !mergeAccess.value.canDryRunUndo) return
  const session = acceptedPreviewSession.value
  const acceptedPreview = preview.value
  const mergeId = receipt.value.mergeId
  if (!acceptedPreview || !acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
  try {
    const result = await dryRunCustomerMergeUndo(mergeId)
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)
      || receipt.value?.mergeId !== mergeId
      || result.mergeId !== mergeId) return
    undoDryRun.value = result
  } catch (err) {
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
    feedback.error(t('merge.undoError'), err instanceof Error ? err.message : String(err))
  }
}

async function executeUndo(): Promise<void> {
  if (!undoDryRun.value || !undoPresentation.value?.canConfirm || undoing.value || !mergeAccess.value.canExecuteUndo) return
  const session = acceptedPreviewSession.value
  const acceptedPreview = preview.value
  const acceptedDryRun = undoDryRun.value
  if (!acceptedPreview || !acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
  undoing.value = true
  try {
    const result = await executeCustomerMergeUndo({
      mergeId: acceptedDryRun.mergeId,
      undoOperationKey: operationKey('merge-undo-ui'),
      eligibilityToken: acceptedDryRun.eligibilityToken,
      expectedSourceRowVersion: acceptedDryRun.sourceRowVersion,
      expectedTargetRowVersion: acceptedDryRun.targetRowVersion,
      actorRef: 'local_user',
      reason: 'operator confirmed undo dry-run',
    })
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
    undoDryRun.value = null
    receipt.value = null
    emit('update:show', false)
    emit('undone', result)
  } catch (err) {
    if (!acceptedMergePreviewIsCurrent(session, currentPreviewContext(), acceptedPreview)) return
    feedback.error(t('merge.undoError'), err instanceof Error ? err.message : String(err))
    undoDryRun.value = null
  } finally {
    if (mergePreviewSessionIsCurrent(session, currentPreviewContext())) undoing.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="receipt ? t('merge.receiptTitle') : t('merge.previewTitle')"
    :style="{ width: 'min(960px, 96vw)' }"
    :mask-closable="!executing && !undoing"
    @update:show="(visible: boolean) => { if (!visible) close() }"
  >
    <div v-if="receipt" class="merge-v4__receipt">
      <dl class="merge-v4__counts">
        <div><dt>{{ t('merge.receipt.migratedIdentityCount') }}</dt><dd>{{ receipt.counts.identities }}</dd></div>
        <div><dt>{{ t('merge.receipt.migratedAddressCount') }}</dt><dd>{{ receipt.counts.addresses }}</dd></div>
        <div><dt>{{ t('merge.receipt.updatedDemandDocs') }}</dt><dd>{{ receipt.counts.demandDocuments }}</dd></div>
        <div><dt>{{ t('merge.receipt.nameObservations') }}</dt><dd>{{ receipt.counts.nameObservations }}</dd></div>
        <div><dt>{{ t('merge.receipt.nameEvents') }}</dt><dd>{{ receipt.counts.nameEvents }}</dd></div>
        <div><dt>{{ t('merge.receipt.origins') }}</dt><dd>{{ receipt.counts.origins }}</dd></div>
        <div><dt>{{ t('merge.receipt.profileMutations') }}</dt><dd>{{ receipt.counts.profileMutations }}</dd></div>
      </dl>
      <p>{{ t('merge.operationRecorded', { id: receipt.mergeId }) }}</p>
    </div>

    <div v-else-if="loading" class="merge-v4__center"><NSpin size="large" /></div>
    <div v-else-if="loadError" class="merge-v4__center">
      <p>{{ loadError }}</p>
      <NButton @click="retryPreview">
        {{ t('common.retry') }}
      </NButton>
    </div>

    <div v-else-if="preview" class="merge-v4">
      <p v-if="staleNotice" class="merge-v4__warning">{{ t('merge.staleReloaded') }}</p>
      <div class="merge-v4__profiles">
        <section><small>{{ t('merge.sourceSide') }}</small><strong>{{ source?.displayName }}</strong><span>{{ t('merge.profileVersion', { id: preview.sourceProfileId, version: preview.sourceRowVersion }) }}</span></section>
        <span aria-hidden="true">→</span>
        <section><small>{{ t('merge.targetSide') }}</small><strong>{{ target?.displayName }}</strong><span>{{ t('merge.profileVersion', { id: preview.targetProfileId, version: preview.targetRowVersion }) }}</span></section>
      </div>
      <dl class="merge-v4__counts">
        <div><dt>{{ t('merge.receipt.migratedIdentityCount') }}</dt><dd>{{ preview.counts.identities }}</dd></div>
        <div><dt>{{ t('merge.receipt.migratedAddressCount') }}</dt><dd>{{ preview.counts.addresses }}</dd></div>
        <div><dt>{{ t('merge.receipt.updatedDemandDocs') }}</dt><dd>{{ preview.counts.demandDocuments }}</dd></div>
        <div><dt>{{ t('merge.receipt.nameObservations') }}</dt><dd>{{ preview.counts.nameObservations }}</dd></div>
        <div><dt>{{ t('merge.receipt.nameEvents') }}</dt><dd>{{ preview.counts.nameEvents }}</dd></div>
        <div><dt>{{ t('merge.receipt.origins') }}</dt><dd>{{ preview.counts.origins }}</dd></div>
        <div><dt>{{ t('merge.receipt.profileMutations') }}</dt><dd>{{ preview.counts.profileMutations }}</dd></div>
      </dl>
      <section
        v-if="resolutionOptions.primaryIdentityGroups.length || defaultAddressOptions.length > 1 || resolutionOptions.displayNames.length"
        class="merge-v4__resolution"
      >
        <strong>{{ t('merge.resolution.title') }}</strong>
        <label v-for="group in resolutionOptions.primaryIdentityGroups" :key="group.key">
          <span>{{ t('merge.resolution.primaryIdentity', { namespace: group.namespace, type: group.identityType }) }}</span>
          <NSelect
            :value="resolution.primaryIdentitySelections[group.key] ?? null"
            :options="identitySelectOptions[group.key] ?? []"
            :placeholder="t('merge.resolution.selectPrimary')"
            @update:value="(value: number) => choosePrimary(group.key, value)"
          />
        </label>
        <label v-if="defaultAddressOptions.length > 1">
          <span>{{ t('merge.resolution.defaultAddress') }}</span>
          <NSelect
            :value="resolution.defaultAddressId"
            :options="defaultAddressOptions"
            clearable
            :placeholder="t('merge.resolution.selectDefaultAddress')"
            @update:value="chooseDefaultAddress"
          />
        </label>
        <div v-if="resolutionOptions.displayNames.length" class="merge-v4__display-resolution">
          <span>{{ t('merge.resolution.displayName') }}</span>
          <NRadioGroup :value="resolution.displayNameResolution" @update:value="chooseDisplayName">
            <NRadioButton v-for="option in resolutionOptions.displayNames" :key="option.resolution" :value="option.resolution">
              {{ option.displayName }}
            </NRadioButton>
          </NRadioGroup>
        </div>
        <NButton type="primary" secondary :disabled="!resolutionDirty" :loading="loading" @click="applyResolution">
          {{ t('merge.resolution.apply') }}
        </NButton>
      </section>
      <section v-if="(preview.frozenDemandDocumentIds ?? []).length > 0" class="merge-v4__notice">
        {{ t('merge.frozenDocuments', { count: (preview.frozenDemandDocumentIds ?? []).length }) }}
      </section>
      <section v-if="!mergeWritesEnabled" class="merge-v4__notice">{{ t('merge.disabledReason') }}</section>
      <section v-if="(preview.blockers ?? []).length > 0" class="merge-v4__blockers">
        <strong>{{ t('merge.blockersTitle') }}</strong>
        <ul><li v-for="blocker in (preview.blockers ?? [])" :key="`${blocker.code}-${blocker.entityId}`">{{ blockerLabel(blocker.code) }}</li></ul>
      </section>
    </div>

    <template #footer>
      <div class="merge-v4__footer">
        <template v-if="receipt">
          <NButton type="warning" @click="requestUndoDryRun">{{ t('merge.undoDryRunAction') }}</NButton>
          <NButton type="primary" @click="close">{{ t('common.close') }}</NButton>
        </template>
        <template v-else>
          <NButton :disabled="executing" @click="close">{{ t('merge.cancelAction') }}</NButton>
          <NButton type="primary" :loading="executing" :disabled="!preview?.canExecute || resolutionDirty || !mergeAccess.canExecuteMerge" @click="executeMerge">
            {{ t('merge.confirmAction') }}
          </NButton>
        </template>
      </div>
    </template>
  </NModal>

  <NModal
    :show="undoDryRun != null"
    preset="card"
    :title="t('merge.undoDryRunTitle')"
    :style="{ width: 'min(640px, 94vw)' }"
    @update:show="(visible: boolean) => { if (!visible && !undoing) undoDryRun = null }"
  >
    <section v-if="undoDryRun" class="merge-v4__undo-meta">
      <p><strong>{{ t('merge.history.auditLevel') }}:</strong> {{ undoDryRun.auditLevel }}</p>
      <dl class="merge-v4__counts">
        <div><dt>{{ t('merge.receipt.nameEvents') }}</dt><dd>{{ undoDryRun.restoreCounts.nameEvents }}</dd></div>
        <div><dt>{{ t('merge.receipt.origins') }}</dt><dd>{{ undoDryRun.restoreCounts.origins }}</dd></div>
        <div><dt>{{ t('merge.receipt.profileMutations') }}</dt><dd>{{ undoDryRun.restoreCounts.profileMutations }}</dd></div>
      </dl>
      <template v-if="(undoDryRun.dependentMergeIds ?? []).length">
        <strong>{{ t('merge.history.dependencies') }}</strong>
        <ul><li v-for="id in (undoDryRun.dependentMergeIds ?? [])" :key="id">#{{ id }}</li></ul>
      </template>
      <template v-if="(undoDryRun.warnings ?? []).length">
        <strong>{{ t('merge.history.warnings') }}</strong>
        <ul><li v-for="(warning, index) in (undoDryRun.warnings ?? [])" :key="index">{{ warning }}</li></ul>
      </template>
    </section>
    <UndoDryRunPanel
      v-if="undoPresentation"
      :result="undoPresentation"
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
</template>

<style scoped>
.merge-v4,
.merge-v4__receipt {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.merge-v4__center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-8);
}

.merge-v4__profiles {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  gap: var(--space-3);
}

.merge-v4__profiles section {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.merge-v4__profiles small,
.merge-v4__profiles span {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.merge-v4__counts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: var(--space-2);
  margin: 0;
}

.merge-v4__counts > div {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.merge-v4__counts dt {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.merge-v4__counts dd {
  margin: var(--space-1) 0 0;
  font-weight: var(--font-weight-semibold);
}

.merge-v4__notice,
.merge-v4__warning,
.merge-v4__blockers,
.merge-v4__resolution {
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.merge-v4__resolution {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--color-text-primary);
  background: var(--color-surface-raised);
  border: 1px solid var(--color-border);
}

.merge-v4__resolution label,
.merge-v4__display-resolution {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.merge-v4__notice,
.merge-v4__warning {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border: 1px solid var(--status-warning-border);
}

.merge-v4__blockers {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border: 1px solid var(--status-error-border);
}

.merge-v4__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.merge-v4__undo-meta {
  margin-bottom: var(--space-3);
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}
</style>
