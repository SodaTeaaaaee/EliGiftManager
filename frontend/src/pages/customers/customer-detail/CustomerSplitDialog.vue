<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NCheckbox, NInput, NModal, NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import type { CustomerNameObservationDTO, CustomerProfileDTO } from '@/entities/customer'
import type { CustomerProfileOriginDTO, CustomerSplitPreviewResult, ExecuteCustomerSplitResult } from '@/entities/customer-resolution'
import type { DemandDocument } from '@/entities/demand'
import {
  executeCustomerSplit,
  listCustomerProfileOrigins,
  listDemandDocuments,
  previewCustomerSplit,
} from '@/shared/api/bridge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'
import {
  buildCustomerSplitPlan,
  canPreviewCustomerSplit,
  completeSplitParentRefresh,
  completeSplitSupplementalRefresh,
  createSplitExecutionRefreshFlow,
  finishSplitExecutionRefresh,
  identityResolutionGroupKey,
  requireSplitExecutionRefresh,
} from '@/shared/lib/customer-resolution'
import { ErrorBanner } from '@/shared/ui/feedback'
import { StatusBadge } from '@/shared/ui/status'

const props = defineProps<{
  show: boolean
  profile: CustomerProfileDTO
  nameObservations: CustomerNameObservationDTO[]
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  executed: [ExecuteCustomerSplitResult]
  'refresh-required': [request: {
    profileId: number
    resolve: () => void
    reject: (error: unknown) => void
  }]
}>()

const { t } = useI18n({ useScope: 'global' })
const featurePolicy = useCustomerResolutionFeaturePolicy()
const origins = ref<CustomerProfileOriginDTO[]>([])
const demandDocuments = ref<DemandDocument[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)
const previewing = ref(false)
const previewError = ref<string | null>(null)
const preview = ref<CustomerSplitPreviewResult | null>(null)
const executing = ref(false)
const receipt = ref<ExecuteCustomerSplitResult | null>(null)
const refreshFlow = ref(createSplitExecutionRefreshFlow())
let baseGeneration = 0
let baseLoadSequence = 0

const newProfileDisplayName = ref('')
const newProfileType = ref('manual')
const sourceDisplayNameResolution = ref('auto_remaining')
const identityIds = ref<number[]>([])
const addressIds = ref<number[]>([])
const demandDocumentIds = ref<number[]>([])
const nameObservationIds = ref<number[]>([])
const originIds = ref<number[]>([])
const targetPrimaryByGroup = ref<Record<string, number>>({})
const targetDefaultAddressId = ref<number | null>(null)
const targetDisplayNameObservationId = ref<number | null>(null)

const profileTypeOptions = computed<SelectOption[]>(() => [
  { label: t('glossary.profileType.member.label'), value: 'member' },
  { label: t('glossary.profileType.buyer.label'), value: 'buyer' },
  { label: t('glossary.profileType.mixed.label'), value: 'mixed' },
  { label: t('glossary.profileType.manual.label'), value: 'manual' },
])

const sourceDisplayOptions = computed<SelectOption[]>(() => [
  { label: t('customerDetail.split.sourceDisplayAuto'), value: 'auto_remaining' },
  { label: t('customerDetail.split.sourceDisplayKeep'), value: 'keep_current' },
])

const selectedIdentityGroups = computed(() => {
  const selected = props.profile.identities.filter((item) => identityIds.value.includes(item.id))
  const groups = new Map<string, typeof selected>()
  for (const identity of selected) {
    const key = identityResolutionGroupKey(identity.identityPlatform, identity.identityType)
    groups.set(key, [...(groups.get(key) ?? []), identity])
  }
  return [...groups.entries()].map(([key, options]) => ({ key, options }))
})

const selectedAddressOptions = computed<SelectOption[]>(() => props.profile.addresses
  .filter((item) => addressIds.value.includes(item.id))
  .map((item) => ({ value: item.id, label: `${item.label || item.recipientName} · ${item.addressLine1}` })))

const selectedObservationOptions = computed<SelectOption[]>(() => props.nameObservations
  .filter((item) => nameObservationIds.value.includes(item.id))
  .map((item) => ({ value: item.id, label: `${item.kind} · ${item.value}` })))

const splitWritesEnabled = computed(() => featurePolicy.isEnabled('splitExecutionEnabled'))
const canRunPreview = computed(() => canPreviewCustomerSplit(
  refreshFlow.value,
  loading.value || previewing.value || executing.value,
))

function originSummary(item: CustomerProfileOriginDTO): string {
  return t('customerDetail.split.originRow', {
    kind: item.originKind,
    externalRef: item.externalRef || '—',
    profile: item.sourceIntegrationProfileId ?? '—',
    document: item.sourceDocumentId ?? '—',
    lastSeen: item.lastSeenAt ?? '—',
  })
}

const plan = computed(() => buildCustomerSplitPlan({
  sourceProfileId: props.profile.id,
  newProfileDisplayName: newProfileDisplayName.value,
  newProfileType: newProfileType.value,
  targetPrimaryIdentityIds: Object.values(targetPrimaryByGroup.value),
  targetDefaultAddressId: targetDefaultAddressId.value,
  targetDisplayNameObservationId: targetDisplayNameObservationId.value,
  sourceDisplayNameResolution: sourceDisplayNameResolution.value,
  identityIds: identityIds.value,
  addressIds: addressIds.value,
  demandDocumentIds: demandDocumentIds.value,
  nameObservationIds: nameObservationIds.value,
  originIds: originIds.value,
}))

function reset(clearPreviewError = true): void {
  newProfileDisplayName.value = `${props.profile.displayName} · split`
  newProfileType.value = props.profile.profileType
  sourceDisplayNameResolution.value = 'auto_remaining'
  identityIds.value = []
  addressIds.value = []
  demandDocumentIds.value = []
  nameObservationIds.value = []
  originIds.value = []
  targetPrimaryByGroup.value = {}
  targetDefaultAddressId.value = null
  targetDisplayNameObservationId.value = null
  preview.value = null
  receipt.value = null
  if (clearPreviewError) previewError.value = null
}

async function loadSupplemental(resetDraft = true): Promise<boolean> {
  const profileId = props.profile.id
  const generation = baseGeneration
  const loadSequence = ++baseLoadSequence
  loading.value = true
  loadError.value = null
  if (resetDraft) reset()
  try {
    const [originRows, documents] = await Promise.all([
      listCustomerProfileOrigins(profileId),
      listDemandDocuments(),
      featurePolicy.load(),
    ])
    if (
      !props.show
      || props.profile.id !== profileId
      || baseGeneration !== generation
      || baseLoadSequence !== loadSequence
    ) return false
    if (originRows.some((item) => item.customerProfileId !== profileId)) {
      throw new Error('split_origin_profile_mismatch')
    }
    origins.value = originRows
    demandDocuments.value = documents.filter((item) => item.customerProfileId === profileId)
    return true
  } catch (err) {
    if (baseGeneration !== generation || baseLoadSequence !== loadSequence) return false
    loadError.value = err instanceof Error ? err.message : String(err)
    return false
  } finally {
    if (baseGeneration === generation && baseLoadSequence === loadSequence) loading.value = false
  }
}

async function load(): Promise<void> {
  refreshFlow.value = createSplitExecutionRefreshFlow()
  await loadSupplemental(true)
}

function requestParentRefresh(profileId: number): Promise<void> {
  return new Promise((resolve, reject) => {
    emit('refresh-required', { profileId, resolve, reject })
  })
}

async function refreshAfterExecutionFailure(profileId: number): Promise<void> {
  try {
    await requestParentRefresh(profileId)
    await nextTick()
    if (!props.show || props.profile.id !== profileId) return
    refreshFlow.value = completeSplitParentRefresh(refreshFlow.value)
    if (!await loadSupplemental(false)) return
    refreshFlow.value = completeSplitSupplementalRefresh(refreshFlow.value)
    refreshFlow.value = finishSplitExecutionRefresh(refreshFlow.value)
    reset(false)
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : String(err)
  }
}

function toggle(list: number[], id: number, checked: boolean): number[] {
  return checked ? [...new Set([...list, id])] : list.filter((value) => value !== id)
}

async function runPreview(): Promise<void> {
  if (!canRunPreview.value) return
  previewing.value = true
  previewError.value = null
  try {
    preview.value = await previewCustomerSplit(plan.value)
  } catch (err) {
    previewError.value = err instanceof Error ? err.message : String(err)
  } finally {
    previewing.value = false
  }
}

async function execute(): Promise<void> {
  if (!preview.value?.canExecute || !splitWritesEnabled.value) return
  const sourceProfileId = props.profile.id
  const acceptedPreview = preview.value
  const acceptedPlan = plan.value
  executing.value = true
  previewError.value = null
  try {
    const result = await executeCustomerSplit({
      operationKey: `split-ui-${crypto.randomUUID()}`,
      planToken: acceptedPreview.planToken,
      expectedSourceRowVersion: acceptedPreview.sourceRowVersion,
      expectedTargetRowVersion: acceptedPreview.targetRowVersion,
      actorRef: 'local_user',
      decisionReason: 'operator reviewed customer split preview',
      plan: acceptedPlan,
    })
    if (!props.show || props.profile.id !== sourceProfileId) return
    receipt.value = result
    emit('executed', result)
  } catch (err) {
    if (!props.show || props.profile.id !== sourceProfileId) return
    previewError.value = err instanceof Error ? err.message : String(err)
    preview.value = null
    refreshFlow.value = requireSplitExecutionRefresh()
    await refreshAfterExecutionFailure(sourceProfileId)
  } finally {
    if (props.profile.id === sourceProfileId) executing.value = false
  }
}

watch(
  () => [props.show, props.profile.id] as const,
  ([visible]) => {
    baseGeneration += 1
    baseLoadSequence += 1
    loading.value = false
    previewing.value = false
    executing.value = false
    origins.value = []
    demandDocuments.value = []
    refreshFlow.value = createSplitExecutionRefreshFlow()
    reset()
    if (visible) void load()
  },
  { flush: 'sync' },
)

async function retry(): Promise<void> {
  if (refreshFlow.value.refreshRequired) {
    await refreshAfterExecutionFailure(props.profile.id)
  } else if (loadError.value) {
    await load()
  } else {
    await runPreview()
  }
}

watch(plan, () => {
  if (preview.value && !executing.value) preview.value = null
}, { deep: true })

watch(identityIds, (selected) => {
  targetPrimaryByGroup.value = Object.fromEntries(
    Object.entries(targetPrimaryByGroup.value).filter(([, id]) => selected.includes(id)),
  )
}, { deep: true })
watch(addressIds, (selected) => {
  if (targetDefaultAddressId.value != null && !selected.includes(targetDefaultAddressId.value)) targetDefaultAddressId.value = null
}, { deep: true })
watch(nameObservationIds, (selected) => {
  if (targetDisplayNameObservationId.value != null && !selected.includes(targetDisplayNameObservationId.value)) targetDisplayNameObservationId.value = null
}, { deep: true })
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('customerDetail.split.title')"
    :style="{ width: 'min(1040px, 96vw)' }"
    :mask-closable="!executing"
    @update:show="(visible: boolean) => emit('update:show', visible)"
  >
    <NSpin :show="loading || previewing">
      <ErrorBanner v-if="loadError || previewError" :message="t('customerDetail.split.failed')" :detail="loadError ?? previewError ?? undefined" @retry="retry" />
      <div v-if="receipt" class="customer-split__stack">
        <strong>{{ t('customerDetail.split.completed', { id: receipt.splitId, target: receipt.targetProfileId }) }}</strong>
        <p>{{ t('customerDetail.split.noDirectUndo') }}</p>
        <NButton type="primary" @click="emit('update:show', false)">{{ t('common.close') }}</NButton>
      </div>
      <div v-else class="customer-split__stack">
        <p class="customer-split__notice">{{ t('customerDetail.split.createNewOnly') }}</p>
        <p class="customer-split__notice">{{ t('customerDetail.split.restoreGuidance') }}</p>

        <div class="customer-split__form-grid">
          <label><span>{{ t('customerDetail.split.newName') }}</span><NInput v-model:value="newProfileDisplayName" /></label>
          <label><span>{{ t('customerDetail.split.newType') }}</span><NSelect v-model:value="newProfileType" :options="profileTypeOptions" /></label>
          <label><span>{{ t('customerDetail.split.sourceDisplay') }}</span><NSelect v-model:value="sourceDisplayNameResolution" :options="sourceDisplayOptions" /></label>
        </div>

        <section class="customer-split__entity-section">
          <strong>{{ t('customerDetail.split.identities') }}</strong>
          <label v-for="item in profile.identities" :key="item.id" class="customer-split__choice">
            <NCheckbox :checked="identityIds.includes(item.id)" @update:checked="identityIds = toggle(identityIds, item.id, $event)" />
            <span>{{ item.identityPlatform }} · <StatusBadge dimension="identityType" :value="item.identityType" size="sm" /> · {{ item.identityValue }}</span>
          </label>
          <label v-for="group in selectedIdentityGroups" :key="group.key">
            <span>{{ t('customerDetail.split.primaryIdentity') }} · {{ group.options[0].identityPlatform }} / <StatusBadge dimension="identityType" :value="group.options[0].identityType" size="sm" /></span>
            <NSelect
              :value="targetPrimaryByGroup[group.key] ?? null"
              :options="group.options.map((item) => ({ value: item.id, label: item.identityValue }))"
              @update:value="(value: number) => targetPrimaryByGroup = { ...targetPrimaryByGroup, [group.key]: value }"
            />
          </label>
        </section>

        <section class="customer-split__entity-section">
          <strong>{{ t('customerDetail.split.addresses') }}</strong>
          <label v-for="item in profile.addresses" :key="item.id" class="customer-split__choice">
            <NCheckbox :checked="addressIds.includes(item.id)" @update:checked="addressIds = toggle(addressIds, item.id, $event)" />
            <span>{{ item.label || item.recipientName }} · {{ item.addressLine1 }}</span>
          </label>
          <label v-if="selectedAddressOptions.length"><span>{{ t('customerDetail.split.defaultAddress') }}</span><NSelect v-model:value="targetDefaultAddressId" :options="selectedAddressOptions" clearable /></label>
        </section>

        <section class="customer-split__entity-section">
          <strong>{{ t('customerDetail.split.nameObservations') }}</strong>
          <label v-for="item in nameObservations" :key="item.id" class="customer-split__choice">
            <NCheckbox :checked="nameObservationIds.includes(item.id)" @update:checked="nameObservationIds = toggle(nameObservationIds, item.id, $event)" />
            <span>{{ item.kind }} · {{ item.value }} · {{ item.source }}</span>
          </label>
          <label v-if="selectedObservationOptions.length"><span>{{ t('customerDetail.split.displayObservation') }}</span><NSelect v-model:value="targetDisplayNameObservationId" :options="selectedObservationOptions" clearable /></label>
        </section>

        <section class="customer-split__entity-section">
          <strong>{{ t('customerDetail.split.origins') }}</strong>
          <label v-for="item in origins" :key="item.id" class="customer-split__choice">
            <NCheckbox :checked="originIds.includes(item.id)" @update:checked="originIds = toggle(originIds, item.id, $event)" />
            <span>{{ originSummary(item) }}</span>
          </label>
        </section>

        <section class="customer-split__entity-section">
          <strong>{{ t('customerDetail.split.demandDocuments') }}</strong>
          <p class="customer-split__notice">{{ t('customerDetail.split.demandHint') }}</p>
          <label v-for="item in demandDocuments" :key="item.id" class="customer-split__choice">
            <NCheckbox :checked="demandDocumentIds.includes(item.id)" @update:checked="demandDocumentIds = toggle(demandDocumentIds, item.id, $event)" />
            <span>#{{ item.id }} · {{ item.kind }} · {{ item.sourceDocumentNo }}</span>
          </label>
        </section>

        <div v-if="preview" class="customer-split__preview">
          <strong>{{ t('customerDetail.split.previewTitle') }}</strong>
          <code>{{ preview.planToken }}</code>
          <p>{{ t('customerDetail.split.versions', { source: preview.sourceRowVersion, target: preview.targetRowVersion }) }}</p>
          <p>{{ t('customerDetail.split.namesAfter', { source: preview.sourceDisplayNameAfter, target: preview.targetDisplayNameAfter }) }}</p>
          <p>{{ t('customerDetail.split.counts', { identities: preview.counts.identities, addresses: preview.counts.addresses, demand: preview.counts.demandDocuments, names: preview.counts.nameObservations, origins: preview.counts.origins }) }}</p>
          <p>{{ t('customerDetail.split.immutableHistory', { waves: preview.immutableHistory.waveParticipantSnapshotIds.length, lines: preview.immutableHistory.fulfillmentLineIds.length, rewrite: preview.immutableHistory.willRewrite ? t('common.yes') : t('common.no') }) }}</p>
          <ul v-if="preview.blockers.length"><li v-for="blocker in preview.blockers" :key="`${blocker.code}-${blocker.entityId}`">{{ blocker.code }} · {{ blocker.detail }}</li></ul>
          <p v-if="!preview.directUndoSupported">{{ t('customerDetail.split.noDirectUndo') }}</p>
        </div>
      </div>
    </NSpin>

    <template #footer>
      <div v-if="!receipt" class="customer-split__footer">
        <span v-if="!splitWritesEnabled" class="customer-split__disabled">{{ t('customerDetail.split.disabledReason') }}</span>
        <NButton :disabled="executing" @click="emit('update:show', false)">{{ t('common.cancel') }}</NButton>
        <NButton :loading="previewing" :disabled="!canRunPreview" @click="runPreview">{{ t('customerDetail.split.previewAction') }}</NButton>
        <NButton type="primary" :loading="executing" :disabled="!preview?.canExecute || !splitWritesEnabled" @click="execute">{{ t('customerDetail.split.executeAction') }}</NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.customer-split__stack,
.customer-split__entity-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.customer-split__form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
}

.customer-split__form-grid label,
.customer-split__entity-section > label:not(.customer-split__choice) {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.customer-split__entity-section,
.customer-split__preview {
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.customer-split__choice {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.customer-split__notice,
.customer-split__disabled {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.customer-split__disabled {
  color: var(--status-warning-fg);
}

.customer-split__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  flex-wrap: wrap;
}
</style>
