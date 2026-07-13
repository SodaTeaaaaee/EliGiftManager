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
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import {
  updateCustomerProfile,
  deleteCustomerProfile,
  deleteAddress,
  listCustomerProfiles,
} from '@/shared/api/bridge'
import type { CustomerAddressDTO } from '@/entities/address'
import type { MergeProfilesResult, UndoCustomerMergeResult } from '@/entities/merge'
import { useCustomerDetail } from './useCustomersPage'
import FulfillmentHistoryPanel from './customer-detail/FulfillmentHistoryPanel.vue'
import IdentityList from './customer-detail/IdentityList.vue'
import AddressFormDialog from './customer-detail/AddressFormDialog.vue'
import MergePreviewDialog from './customer-detail/MergePreviewDialog.vue'

const props = defineProps<{
  id: string
}>()

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()

const numericId = computed(() => {
  const parsed = Number(props.id)
  return Number.isFinite(parsed) ? parsed : null
})

const { profile, loading, notFound, refresh } = useCustomerDetail(numericId)

function backToList(): void {
  router.push({ name: 'customers' })
}

// ── Profile edit ──

const isEditingProfile = ref(false)
const editDisplayName = ref('')
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
  if (!profile.value) return
  editDisplayName.value = profile.value.displayName
  editProfileType.value = profile.value.profileType
  editExtraData.value = profile.value.extraData
  isEditingProfile.value = true
}

function cancelEditProfile(): void {
  isEditingProfile.value = false
}

const canSaveProfile = computed(() => !savingProfile.value && editDisplayName.value.trim().length > 0)

async function saveProfile(): Promise<void> {
  if (!profile.value || !canSaveProfile.value) return
  savingProfile.value = true
  try {
    await updateCustomerProfile({
      id: profile.value.id,
      displayName: editDisplayName.value.trim(),
      profileType: editProfileType.value,
      extraData: editExtraData.value,
    })
    isEditingProfile.value = false
    feedback.success(t('customerDetail.feedback.saved'))
    await refresh()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    savingProfile.value = false
  }
}

// ── Profile delete ──

const showDeleteConfirm = ref(false)
const deletingProfile = ref(false)

async function confirmDeleteProfile(): Promise<void> {
  if (!profile.value) return
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
  editingAddress.value = null
  showAddressDialog.value = true
}

function openEditAddress(address: CustomerAddressDTO): void {
  editingAddress.value = address
  showAddressDialog.value = true
}

async function onAddressSaved(): Promise<void> {
  await refresh()
}

const pendingDeleteAddressId = ref<number | null>(null)
const deletingAddress = ref(false)

function requestDeleteAddress(id: number): void {
  pendingDeleteAddressId.value = id
}

function cancelDeleteAddress(): void {
  pendingDeleteAddressId.value = null
}

async function confirmDeleteAddress(): Promise<void> {
  if (pendingDeleteAddressId.value == null) return
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

async function openTargetPicker(): Promise<void> {
  if (!profile.value) return
  selectedTargetId.value = null
  showTargetPicker.value = true
  loadingTargets.value = true
  try {
    const all = await listCustomerProfiles()
    targetOptions.value = all
      .filter((p) => p.id !== profile.value?.id)
      .map((p) => ({ label: `${p.displayName} (#${p.id})`, value: p.id }))
  } finally {
    loadingTargets.value = false
  }
}

const showMergePreview = ref(false)

function confirmTargetPicker(): void {
  if (selectedTargetId.value == null) return
  showTargetPicker.value = false
  showMergePreview.value = true
}

function onMergePreviewVisibility(visible: boolean): void {
  showMergePreview.value = visible
}

async function onMerged(_result: MergeProfilesResult): Promise<void> {
  // This profile was the merge source and no longer exists — the current
  // detail page's id is now dangling, so route back to the list.
  backToList()
}

async function onMergeUndone(result: UndoCustomerMergeResult): Promise<void> {
  await refresh()
  await router.push({ name: 'customer-detail', params: { id: result.restoredSourceProfileId } })
}

function goToSuggestedMerges(): void {
  router.push({ name: 'customers' })
}

onMounted(refresh)
watch(numericId, () => {
  isEditingProfile.value = false
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
          <NButton @click="goToSuggestedMerges">{{ t('customerDetail.merge.suggestedAction') }}</NButton>
          <NButton @click="openTargetPicker">{{ t('customerDetail.merge.manualAction') }}</NButton>
        </template>
      </PageHeader>

      <FulfillmentHistoryPanel :customer-profile-id="profile.id" />

      <SectionCard :title="t('customerDetail.sections.profile')">
        <template #actions>
          <template v-if="!isEditingProfile">
            <NButton size="small" @click="startEditProfile">{{ t('customerDetail.profile.editAction') }}</NButton>
            <NButton size="small" type="error" quaternary @click="showDeleteConfirm = true">
              {{ t('customerDetail.profile.deleteAction') }}
            </NButton>
          </template>
          <template v-else>
            <NButton size="small" :disabled="savingProfile" @click="cancelEditProfile">
              {{ t('customerDetail.profile.cancelAction') }}
            </NButton>
            <NButton size="small" type="primary" :loading="savingProfile" :disabled="!canSaveProfile" @click="saveProfile">
              {{ t('customerDetail.profile.saveAction') }}
            </NButton>
          </template>
        </template>

        <NForm v-if="isEditingProfile" label-placement="top">
          <NFormItem :label="t('customerDetail.profile.displayNameLabel')">
            <NInput v-model:value="editDisplayName" :disabled="savingProfile" />
          </NFormItem>
          <NFormItem :label="t('customerDetail.profile.profileTypeLabel')">
            <NSelect v-model:value="editProfileType" :options="profileTypeOptions" :disabled="savingProfile" />
          </NFormItem>
          <NFormItem :label="t('customerDetail.profile.extraDataLabel')">
            <NInput v-model:value="editExtraData" type="textarea" :autosize="{ minRows: 2, maxRows: 5 }" :disabled="savingProfile" />
          </NFormItem>
        </NForm>
        <dl v-else class="customer-detail-page__profile-grid">
          <div class="customer-detail-page__profile-row">
            <dt>{{ t('customerDetail.profile.displayNameLabel') }}</dt>
            <dd>{{ profile.displayName }}</dd>
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

      <SectionCard :title="t('customerDetail.sections.identities')">
        <IdentityList :customer-profile-id="profile.id" :identities="profile.identities" @changed="refresh" />
      </SectionCard>

      <SectionCard :title="t('customerDetail.sections.addresses')">
        <template #actions>
          <NButton size="small" @click="openCreateAddress">{{ t('customerDetail.addresses.addAction') }}</NButton>
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
            <div class="customer-detail-page__address-actions">
              <NButton size="tiny" quaternary @click="openEditAddress(address)">
                {{ t('customerDetail.addresses.editAction') }}
              </NButton>
              <NButton size="tiny" quaternary @click="requestDeleteAddress(address.id)">
                {{ t('customerDetail.addresses.deleteAction') }}
              </NButton>
            </div>
          </li>
        </ul>
      </SectionCard>
    </template>

    <AddressFormDialog
      v-if="profile"
      :show="showAddressDialog"
      :customer-profile-id="profile.id"
      :address="editingAddress"
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
