<script setup lang="ts">
/**
 * CustomerDetailPage — the unified customer detail page (plan §3.6 line
 * 253): the SOLE edit surface for profile fields, identities, addresses,
 * PLUS the cross-wave fulfillment-history section (acceptance D-1 — renders
 * on first paint, no extra navigation, no placeholder). Absorbs what used
 * to be split across the old tree's `CustomerManagementPage.vue` drawer
 * (CRUD) and `CustomerDetailPage.vue` (read-only) — see contract
 * `oldTreeReference`.
 *
 * `id` arrives as a route prop (router `props: true`, see
 * `app/router/index.ts`) — read via `defineProps`, not `useRoute().params`.
 */
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { DisplayNameModeControl, NicknameTimeline } from '@/shared/ui/customer-resolution'
import {
  buildNicknameTimeline,
  canSaveDisplayName,
  createDisplayNameEditState,
  customerResolutionWriteAccess,
  reduceDisplayNameEditState,
  type DisplayNameEditState,
  type DisplayNameMode,
} from '@/shared/lib/customer-resolution'
import {
  updateCustomerProfile,
  deleteCustomerProfile,
  deleteAddress,
  listCustomerProfiles,
  listCustomerNameObservations,
  pinCustomerDisplayName,
  unpinCustomerDisplayName,
} from '@/shared/api/bridge'
import type { CustomerAddressDTO } from '@/entities/address'
import type { CustomerNameObservationDTO } from '@/entities/customer'
import type { ExecuteCustomerMergeResult, ExecuteCustomerMergeUndoResult } from '@/entities/merge'
import { useCustomerDetail } from './useCustomersPage'
import FulfillmentHistoryPanel from './customer-detail/FulfillmentHistoryPanel.vue'
import IdentityList from './customer-detail/IdentityList.vue'
import AddressFormDialog from './customer-detail/AddressFormDialog.vue'
import MergePreviewDialog from './customer-detail/MergePreviewDialog.vue'
import MergeHistoryPanel from './customer-detail/MergeHistoryPanel.vue'
import CustomerSplitDialog from './customer-detail/CustomerSplitDialog.vue'
import CustomerSplitHistoryPanel from './customer-detail/CustomerSplitHistoryPanel.vue'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'

const props = defineProps<{
  id: string
}>()

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()
const customerWriteAccess = computed(() => customerResolutionWriteAccess(featurePolicy.policy.value))
void featurePolicy.load()

const numericId = computed(() => {
  const parsed = Number(props.id)
  return Number.isFinite(parsed) ? parsed : null
})

const { profile, loading, notFound, refresh } = useCustomerDetail(numericId)
const isMerged = computed(() => profile.value?.status === 'merged' || profile.value?.mergedIntoProfileId != null)

const nameObservations = ref<CustomerNameObservationDTO[]>([])
const splitHistoryRefreshSignal = ref(0)
const nameObservationsLoading = ref(false)
const nameObservationsError = ref<string | null>(null)
const nicknameTimeline = computed(() => buildNicknameTimeline(nameObservations.value.map((item) => ({
  id: item.id,
  kind: item.kind,
  displayValue: item.value,
  sourceLabel: item.source,
  originProfileId: item.originProfileId,
  firstSeenAt: item.firstSeenAt ?? '',
  lastSeenAt: item.lastSeenAt ?? item.firstSeenAt ?? '',
  observationCount: item.count,
})), profile.value?.displayNameObservationId ?? undefined))

async function loadNameObservations(): Promise<void> {
  if (!profile.value) return
  nameObservationsLoading.value = true
  nameObservationsError.value = null
  try {
    nameObservations.value = await listCustomerNameObservations(profile.value.id)
  } catch (err) {
    nameObservations.value = []
    nameObservationsError.value = err instanceof Error ? err.message : String(err)
  } finally {
    nameObservationsLoading.value = false
  }
}

async function refreshAll(): Promise<void> {
  await refresh()
  await loadNameObservations()
}

function backToList(): void {
  router.push({ name: 'customers' })
}

// ── Profile edit ──

const isEditingProfile = ref(false)
const displayNameEdit = ref<DisplayNameEditState | null>(null)
const editProfileType = ref('manual')
const editExtraData = ref('')
const savingProfile = ref(false)

const profileTypeOptions = computed<SelectOption[]>(() => [
  { label: t('glossary.profileType.member.label'), value: 'member' },
  { label: t('glossary.profileType.buyer.label'), value: 'buyer' },
  { label: t('glossary.profileType.mixed.label'), value: 'mixed' },
  { label: t('glossary.profileType.manual.label'), value: 'manual' },
])

function startEditProfile(): void {
  if (!profile.value || !customerWriteAccess.value.canEditProfile) return
  const automatic = nicknameTimeline.value.find((item) => item.kind !== 'manual_display_name')?.displayValue ?? profile.value.displayName
  displayNameEdit.value = createDisplayNameEditState({
    name: profile.value.displayName,
    mode: profile.value.displayNameMode === 'pinned' ? 'pinned' : 'auto',
    autoName: automatic,
    rowVersion: profile.value.rowVersion,
  })
  editProfileType.value = profile.value.profileType
  editExtraData.value = profile.value.extraData
  isEditingProfile.value = true
}

function cancelEditProfile(): void {
  isEditingProfile.value = false
  displayNameEdit.value = null
}

function setDisplayNameMode(mode: DisplayNameMode): void {
  if (displayNameEdit.value && customerWriteAccess.value.canEditProfile) {
    displayNameEdit.value = reduceDisplayNameEditState(displayNameEdit.value, { type: 'select_mode', mode })
  }
}

function setDisplayName(value: string): void {
  if (displayNameEdit.value && customerWriteAccess.value.canEditProfile) {
    displayNameEdit.value = reduceDisplayNameEditState(displayNameEdit.value, { type: 'edit_name', value })
  }
}

const canSaveProfile = computed(() => {
  if (!profile.value || !displayNameEdit.value || savingProfile.value || isMerged.value || !customerWriteAccess.value.canEditProfile) return false
  const metadataChanged = editProfileType.value !== profile.value.profileType || editExtraData.value !== profile.value.extraData
  return metadataChanged || canSaveDisplayName(displayNameEdit.value)
})

async function saveProfile(): Promise<void> {
  if (!profile.value || !canSaveProfile.value) return
  savingProfile.value = true
  try {
    let current = profile.value
    const edit = displayNameEdit.value
    if (!edit) return
    if (edit.mode === 'auto' && current.displayNameMode === 'pinned') {
      current = await unpinCustomerDisplayName({
        profileId: current.id,
        expectedRowVersion: current.rowVersion,
        actorRef: 'local_user',
        idempotencyKey: `display-unpin-${crypto.randomUUID()}`,
      })
    } else if (edit.mode === 'pinned' && (current.displayNameMode !== 'pinned' || edit.draftName.trim() !== current.displayName)) {
      current = await pinCustomerDisplayName({
        profileId: current.id,
        name: edit.draftName.trim(),
        expectedRowVersion: current.rowVersion,
        actorRef: 'local_user',
        idempotencyKey: `display-pin-${crypto.randomUUID()}`,
      })
    }
    if (editProfileType.value !== current.profileType || editExtraData.value !== current.extraData) {
      await updateCustomerProfile({
        id: current.id,
        displayName: current.displayName,
        profileType: editProfileType.value,
        extraData: editExtraData.value,
        expectedRowVersion: current.rowVersion,
        actorRef: 'local_user',
        idempotencyKey: `profile-update-${crypto.randomUUID()}`,
      })
    }
    isEditingProfile.value = false
    feedback.success(t('customerDetail.feedback.saved'))
    await refreshAll()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
    await refreshAll()
  } finally {
    savingProfile.value = false
  }
}

// ── Profile delete ──

const showDeleteConfirm = ref(false)
const deletingProfile = ref(false)

async function confirmDeleteProfile(): Promise<void> {
  if (!profile.value || !customerWriteAccess.value.canDeleteProfile) return
  deletingProfile.value = true
  try {
    await deleteCustomerProfile(profile.value.id)
    showDeleteConfirm.value = false
    backToList()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    deletingProfile.value = false
  }
}

// ── Addresses ──

const showAddressDialog = ref(false)
const editingAddress = ref<CustomerAddressDTO | null>(null)

function openCreateAddress(): void {
  if (!customerWriteAccess.value.canManageAddresses) return
  editingAddress.value = null
  showAddressDialog.value = true
}

function openEditAddress(address: CustomerAddressDTO): void {
  if (!customerWriteAccess.value.canManageAddresses) return
  editingAddress.value = address
  showAddressDialog.value = true
}

async function onAddressSaved(): Promise<void> {
  await refresh()
}

const pendingDeleteAddressId = ref<number | null>(null)
const deletingAddress = ref(false)

function requestDeleteAddress(id: number): void {
  if (!customerWriteAccess.value.canManageAddresses) return
  pendingDeleteAddressId.value = id
}

function cancelDeleteAddress(): void {
  pendingDeleteAddressId.value = null
}

async function confirmDeleteAddress(): Promise<void> {
  if (pendingDeleteAddressId.value == null || !customerWriteAccess.value.canManageAddresses) return
  deletingAddress.value = true
  try {
    await deleteAddress(pendingDeleteAddressId.value)
    pendingDeleteAddressId.value = null
    feedback.success(t('customerDetail.feedback.addressDeleted'))
    await refresh()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    deletingAddress.value = false
  }
}

// ── Manual merge (contract: "Mount <MergePreviewDialog> for a manual
// 'merge into another profile' action" — target picker lives locally, the
// dialog itself is the fixed MERGE-unit-owned component). ──

const showTargetPicker = ref(false)
const targetOptions = ref<SelectOption[]>([])
const selectedTargetId = ref<number | null>(null)
const loadingTargets = ref(false)
const targetLoadError = ref<string | null>(null)

async function openTargetPicker(): Promise<void> {
  if (!profile.value) return
  selectedTargetId.value = null
  showTargetPicker.value = true
  loadingTargets.value = true
  targetLoadError.value = null
  try {
    const all = await listCustomerProfiles()
    targetOptions.value = all
      .filter((p) => p.id !== profile.value?.id && p.status !== 'merged' && p.mergedIntoProfileId == null)
      .map((p) => ({ label: `${p.displayName} (#${p.id})`, value: p.id }))
  } catch (err) {
    targetOptions.value = []
    targetLoadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    loadingTargets.value = false
  }
}

const showMergePreview = ref(false)
const showSplitDialog = ref(false)

function confirmTargetPicker(): void {
  if (selectedTargetId.value == null) return
  showTargetPicker.value = false
  showMergePreview.value = true
}

function onMergePreviewVisibility(visible: boolean): void {
  showMergePreview.value = visible
}

async function onMerged(_result: ExecuteCustomerMergeResult): Promise<void> {
  if (selectedTargetId.value != null) {
    await router.push({ name: 'customer-detail', params: { id: selectedTargetId.value } })
  } else {
    backToList()
  }
}

async function onMergeUndone(result: ExecuteCustomerMergeUndoResult): Promise<void> {
  await refreshAll()
  await router.push({ name: 'customer-detail', params: { id: result.restoredSourceProfileId } })
}

async function onSplitExecuted(): Promise<void> {
  await refreshAll()
  splitHistoryRefreshSignal.value += 1
}

async function onSplitRefreshRequired(request: {
  profileId: number
  resolve: () => void
  reject: (error: unknown) => void
}): Promise<void> {
  if (profile.value?.id !== request.profileId) {
    request.reject(new Error('split_refresh_profile_mismatch'))
    return
  }
  try {
    await refreshAll()
    if (profile.value?.id !== request.profileId) throw new Error('split_refresh_profile_changed')
    request.resolve()
  } catch (err) {
    request.reject(err)
  }
}

function goToMergedTarget(): void {
  if (profile.value?.mergedIntoProfileId != null) {
    void router.push({ name: 'customer-detail', params: { id: profile.value.mergedIntoProfileId } })
  }
}

function goToSuggestedMerges(): void {
  router.push({ name: 'customers' })
}

watch(numericId, () => {
  isEditingProfile.value = false
  displayNameEdit.value = null
})
watch(() => profile.value?.id, (id, previous) => {
  if (id == null) {
    nameObservations.value = []
    nameObservationsError.value = null
    return
  }
  if (id !== previous) void loadNameObservations()
}, { immediate: true })
watch(() => customerWriteAccess.value.canEditProfile, (enabled) => {
  if (enabled) return
  cancelEditProfile()
  showDeleteConfirm.value = false
  showAddressDialog.value = false
  pendingDeleteAddressId.value = null
})
</script>

<template>
  <div class="customer-detail-page">
    <template v-if="loading && !profile">
      <div class="customer-detail-page__loading">
        <NSpin size="large" />
      </div>
    </template>

    <template v-else-if="notFound || !profile">
      <EmptyState :title="t('customerDetail.notFound')" size="md">
        <NButton @click="backToList">{{ t('customerDetail.backToList') }}</NButton>
      </EmptyState>
    </template>

    <template v-else>
      <PageHeader :title="profile.displayName" :description="t('customerDetail.subtitle')">
        <template #actions>
          <NButton quaternary @click="backToList">{{ t('customerDetail.backToList') }}</NButton>
          <NButton v-if="!isMerged" @click="goToSuggestedMerges">{{ t('customerDetail.merge.suggestedAction') }}</NButton>
          <NButton v-if="!isMerged" @click="openTargetPicker">{{ t('customerDetail.merge.manualAction') }}</NButton>
          <NButton v-if="!isMerged" @click="showSplitDialog = true">{{ t('customerDetail.split.action') }}</NButton>
        </template>
      </PageHeader>

      <p v-if="!customerWriteAccess.canEditProfile" class="customer-detail-page__writes-disabled">
        {{ t('customerDetail.writesDisabledReason') }}
      </p>

      <section v-if="isMerged" class="customer-detail-page__merged-notice">
        <span>{{ t('customerDetail.mergedNotice') }}</span>
        <NButton v-if="profile.mergedIntoProfileId" size="small" @click="goToMergedTarget">
          {{ t('customerDetail.viewMergedTarget') }}
        </NButton>
      </section>

      <FulfillmentHistoryPanel :customer-profile-id="profile.id" />

      <SectionCard :title="t('customerDetail.sections.profile')">
        <template #actions>
          <template v-if="!isEditingProfile && !isMerged">
            <NButton size="small" :disabled="!customerWriteAccess.canEditProfile" @click="startEditProfile">{{ t('customerDetail.profile.editAction') }}</NButton>
            <NButton size="small" type="error" quaternary :disabled="!customerWriteAccess.canDeleteProfile" @click="showDeleteConfirm = true">
              {{ t('customerDetail.profile.deleteAction') }}
            </NButton>
          </template>
          <template v-else-if="isEditingProfile && !isMerged">
            <NButton size="small" :disabled="savingProfile" @click="cancelEditProfile">
              {{ t('customerDetail.profile.cancelAction') }}
            </NButton>
            <NButton size="small" type="primary" :loading="savingProfile" :disabled="!canSaveProfile" @click="saveProfile">
              {{ t('customerDetail.profile.saveAction') }}
            </NButton>
          </template>
        </template>

        <NForm v-if="isEditingProfile && displayNameEdit" label-placement="top">
          <NFormItem :label="t('customerDetail.profile.displayNameLabel')">
            <DisplayNameModeControl
              :state="displayNameEdit"
              :disabled="savingProfile || !customerWriteAccess.canEditProfile"
              :labels="{
                auto: t('customerDetail.profile.displayNameAuto'),
                pinned: t('customerDetail.profile.displayNamePinned'),
                autoPreview: t('customerDetail.profile.displayNameAutoPreview'),
                nameInput: t('customerDetail.profile.displayNameLabel'),
              }"
              @update:mode="setDisplayNameMode"
              @update:name="setDisplayName"
            />
          </NFormItem>
          <NFormItem :label="t('customerDetail.profile.profileTypeLabel')">
            <NSelect v-model:value="editProfileType" :options="profileTypeOptions" :disabled="savingProfile || !customerWriteAccess.canEditProfile" />
          </NFormItem>
          <NFormItem :label="t('customerDetail.profile.extraDataLabel')">
            <NInput v-model:value="editExtraData" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" :disabled="savingProfile || !customerWriteAccess.canEditProfile" />
          </NFormItem>
        </NForm>
        <dl v-else class="customer-detail-page__profile-grid">
          <div class="customer-detail-page__profile-row">
            <dt>{{ t('customerDetail.profile.displayNameLabel') }}</dt>
            <dd>
              {{ profile.displayName }}
              <span class="customer-detail-page__mode-tag">
                {{ profile.displayNameMode === 'pinned' ? t('customerDetail.profile.displayNamePinned') : t('customerDetail.profile.displayNameAuto') }}
              </span>
            </dd>
          </div>
          <div class="customer-detail-page__profile-row">
            <dt>{{ t('customerDetail.profile.profileTypeLabel') }}</dt>
            <dd><StatusBadge dimension="profileType" :value="profile.profileType" /></dd>
          </div>
          <div class="customer-detail-page__profile-row">
            <dt>{{ t('customerDetail.profile.createdAtLabel') }}</dt>
            <dd>{{ profile.createdAt }}</dd>
          </div>
          <div class="customer-detail-page__profile-row">
            <dt>{{ t('customerDetail.profile.updatedAtLabel') }}</dt>
            <dd>{{ profile.updatedAt }}</dd>
          </div>
        </dl>
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.nameHistory')">
        <NSpin v-if="nameObservationsLoading" size="small" />
        <ErrorBanner
          v-else-if="nameObservationsError"
          :message="t('customerDetail.nameHistory.loadFailed')"
          :detail="nameObservationsError"
          @retry="loadNameObservations"
        />
        <NicknameTimeline
          v-else
          :episodes="nicknameTimeline"
          :labels="{
            currentDisplayName: t('customerDetail.nameHistory.current'),
            observedCount: (count: number) => t('customerDetail.nameHistory.observedCount', { count }),
            sourceFallback: t('customerDetail.nameHistory.unknownSource'),
            empty: t('customerDetail.nameHistory.empty'),
          }"
        />
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.identities')">
        <IdentityList v-if="!isMerged" :customer-profile-id="profile.id" :identities="profile.identities" @changed="refreshAll" />
        <p v-else class="customer-detail-page__readonly-hint">{{ t('customerDetail.mergedReadonly') }}</p>
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.addresses')">
        <template v-if="!isMerged" #actions>
          <NButton size="small" :disabled="!customerWriteAccess.canManageAddresses" @click="openCreateAddress">{{ t('customerDetail.addresses.addAction') }}</NButton>
        </template>
        <EmptyState v-if="profile.addresses.length === 0" :title="t('customerDetail.addresses.empty')" size="sm" />
        <ul v-else class="customer-detail-page__address-list">
          <li v-for="address in profile.addresses" :key="address.id" class="customer-detail-page__address-item">
            <div class="customer-detail-page__address-main">
              <span class="customer-detail-page__address-recipient">{{ address.recipientName }} · {{ address.phone }}</span>
              <span class="customer-detail-page__address-region">
                {{ address.province }}{{ address.city }}{{ address.district }} {{ address.addressLine1 }}
              </span>
              <span v-if="address.isDefault" class="customer-detail-page__default-tag">
                {{ t('customerDetail.addresses.defaultBadge') }}
              </span>
            </div>
            <div v-if="!isMerged" class="customer-detail-page__address-actions">
              <NButton size="tiny" quaternary :disabled="!customerWriteAccess.canManageAddresses" @click="openEditAddress(address)">
                {{ t('customerDetail.addresses.editAction') }}
              </NButton>
              <NButton size="tiny" quaternary :disabled="!customerWriteAccess.canManageAddresses" @click="requestDeleteAddress(address.id)">
                {{ t('customerDetail.addresses.deleteAction') }}
              </NButton>
            </div>
          </li>
        </ul>
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.mergeHistory')">
        <MergeHistoryPanel :customer-profile-id="profile.id" @undone="onMergeUndone" />
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.splitHistory')">
        <CustomerSplitHistoryPanel
          :customer-profile-id="profile.id"
          :refresh-signal="splitHistoryRefreshSignal"
        />
      </SectionCard>
    </template>

    <AddressFormDialog
      v-if="profile && !isMerged"
      :show="showAddressDialog"
      :customer-profile-id="profile.id"
      :address="editingAddress"
      :writes-enabled="customerWriteAccess.canManageAddresses"
      @update:show="(v: boolean) => (showAddressDialog = v)"
      @saved="onAddressSaved"
    />

    <NModal
      :show="pendingDeleteAddressId != null"
      preset="dialog"
      type="warning"
      :title="t('customerDetail.addresses.deleteAction')"
      :content="t('customerDetail.addresses.deleteConfirm')"
      :positive-text="t('common.confirm')"
      :negative-text="t('common.cancel')"
      :loading="deletingAddress"
      @positive-click="confirmDeleteAddress"
      @negative-click="cancelDeleteAddress"
      @update:show="(v: boolean) => { if (!v) cancelDeleteAddress() }"
    />

    <NModal
      :show="showDeleteConfirm"
      preset="dialog"
      type="warning"
      :title="t('customerDetail.profile.deleteAction')"
      :content="t('customerDetail.profile.deleteConfirm')"
      :positive-text="t('common.confirm')"
      :negative-text="t('common.cancel')"
      :loading="deletingProfile"
      @positive-click="confirmDeleteProfile"
      @negative-click="showDeleteConfirm = false"
      @update:show="(v: boolean) => (showDeleteConfirm = v)"
    />

    <NModal
      :show="showTargetPicker"
      preset="card"
      :title="t('customerDetail.merge.manualAction')"
      :style="{ width: 'min(420px, 92vw)' }"
      @update:show="(v: boolean) => (showTargetPicker = v)"
    >
      <NSpin v-if="loadingTargets" size="small" />
      <ErrorBanner
        v-else-if="targetLoadError"
        :message="t('customerDetail.merge.targetLoadFailed')"
        :detail="targetLoadError"
        @retry="openTargetPicker"
      />
      <NSelect
        v-else
        v-model:value="selectedTargetId"
        :options="targetOptions"
        filterable
      />
      <template #footer>
        <div class="customer-detail-page__target-picker-footer">
          <NButton @click="showTargetPicker = false">{{ t('merge.cancelAction') }}</NButton>
          <NButton type="primary" :disabled="selectedTargetId == null" @click="confirmTargetPicker">
            {{ t('merge.previewTitle') }}
          </NButton>
        </div>
      </template>
    </NModal>

    <MergePreviewDialog
      v-if="profile"
      :show="showMergePreview"
      :source-profile-id="profile.id"
      :target-profile-id="selectedTargetId"
      @update:show="onMergePreviewVisibility"
      @merged="onMerged"
      @undone="onMergeUndone"
    />
    <CustomerSplitDialog
      v-if="profile && !isMerged"
      :show="showSplitDialog"
      :profile="profile"
      :name-observations="nameObservations"
      @update:show="showSplitDialog = $event"
      @executed="onSplitExecuted"
      @refresh-required="onSplitRefreshRequired"
    />
  </div>
</template>

<style scoped>
.customer-detail-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.customer-detail-page__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-8) 0;
}

.customer-detail-page__writes-disabled {
  margin: 0;
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.customer-detail-page__merged-notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--status-warning-border);
  border-radius: var(--radius-md);
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
}

.customer-detail-page__mode-tag {
  display: inline-flex;
  margin-left: var(--space-2);
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  font-size: var(--font-size-xs);
}

.customer-detail-page__readonly-hint {
  margin: 0;
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.customer-detail-page__profile-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
}

.customer-detail-page__profile-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.customer-detail-page__profile-row dt {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.customer-detail-page__profile-row dd {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.customer-detail-page__address-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.customer-detail-page__address-item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
}

.customer-detail-page__address-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  min-width: 0;
}

.customer-detail-page__address-recipient {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.customer-detail-page__address-region {
  color: var(--color-text-secondary);
}

.customer-detail-page__default-tag {
  display: inline-flex;
  align-items: center;
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border: 1px solid var(--status-info-border);
}

.customer-detail-page__address-actions {
  display: flex;
  gap: var(--space-1);
  flex-shrink: 0;
}

.customer-detail-page__target-picker-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
