<script setup lang="ts">
/**
 * CustomersPage — the customer list (plan §3.6 line 253): search / platform
 * filter / missing-address-only toggle over a `DataGrid`, row click routes
 * straight into the unified detail page, and a read-only quick-preview
 * drawer for a fast glance without leaving the list. Also hosts the
 * "merge suggestions" entry point (`SuggestedMergesList`, MERGE-unit owned)
 * which opens `MergePreviewDialog` (also MERGE-unit owned) on 'preview'.
 *
 * FilterBar is NOT used here (contract `uiPrimitives` decision point — its
 * `FilterField` union has no boolean-toggle or dynamic-single-select field
 * type, and the customer list's 3-filter surface doesn't fit it cleanly) —
 * page-local NInput/NSelect/NSwitch controls instead.
 */
import { computed, h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSwitch } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import { createCustomerProfile } from '@/shared/api/bridge'
import type { CustomerProfileDTO } from '@/entities/customer'
import type { MergeProfilesResult, UndoCustomerMergeResult } from '@/entities/merge'
import { useCustomerList } from './useCustomersPage'
import SuggestedMergesList from './customer-detail/SuggestedMergesList.vue'
import MergePreviewDialog from './customer-detail/MergePreviewDialog.vue'

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()

const {
  loading,
  platform,
  missingAddressOnly,
  platformOptions,
  profiles,
  refresh,
  keywordDraft,
  onKeywordInput,
  page,
  pageSize,
  totalCount,
  onPageChange,
  onSort,
} = useCustomerList()

onMounted(refresh)

function openDetail(profile: CustomerProfileDTO): void {
  router.push({ name: 'customer-detail', params: { id: profile.id } })
}

// ── Quick preview (read-only side panel) ──

const previewProfile = ref<CustomerProfileDTO | null>(null)
const showPreview = ref(false)

function openPreview(profile: CustomerProfileDTO): void {
  previewProfile.value = profile
  showPreview.value = true
}

function onPreviewVisibility(visible: boolean): void {
  showPreview.value = visible
  if (!visible) previewProfile.value = null
}

// ── Grid columns ──

const platformSelectOptions = computed<SelectOption[]>(() => [
  { label: t('customerList.filter.platformAll'), value: '' },
  ...platformOptions.value,
])

const columns = computed(() => {
  const specs: DataGridColumnSpec<CustomerProfileDTO>[] = [
    { type: 'text', key: 'displayName', title: t('customerList.columns.displayName'), minWidth: 180 },
    {
      type: 'status',
      key: 'profileType',
      title: t('customerList.columns.profileType'),
      dimension: 'profileType',
      width: 120,
    },
    {
      type: 'number',
      key: 'identities',
      title: t('customerList.columns.identities'),
      width: 90,
      sortable: false,
      getValue: (row) => row.identities?.length ?? 0,
    },
    {
      type: 'number',
      key: 'addresses',
      title: t('customerList.columns.addresses'),
      width: 90,
      sortable: false,
      getValue: (row) => row.activeAddressCount,
    },
    { type: 'date', key: 'createdAt', title: t('customerList.columns.createdAt'), width: 130 },
    {
      type: 'actions',
      key: 'actions',
      title: t('customerList.columns.actions'),
      width: 200,
      render: (row) =>
        h('div', { class: 'customers-page__row-actions' }, [
          h(
            NButton,
            {
              size: 'tiny',
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation()
                openPreview(row)
              },
            },
            { default: () => t('customerList.quickPreview.title') },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation()
                openDetail(row)
              },
            },
            { default: () => t('customerList.quickPreview.viewDetail') },
          ),
        ]),
    },
  ]
  return createColumns<CustomerProfileDTO>(specs)
})

// ── Create dialog ──

const showCreate = ref(false)
const createName = ref('')
const createType = ref('manual')
const creating = ref(false)

const profileTypeOptions = computed<SelectOption[]>(() => [
  { label: t('glossary.profileType.member.label'), value: 'member' },
  { label: t('glossary.profileType.buyer.label'), value: 'buyer' },
  { label: t('glossary.profileType.mixed.label'), value: 'mixed' },
  { label: t('glossary.profileType.manual.label'), value: 'manual' },
])

function openCreate(): void {
  createName.value = ''
  createType.value = 'manual'
  showCreate.value = true
}

const canCreate = computed(() => !creating.value && createName.value.trim().length > 0)

async function handleCreate(): Promise<void> {
  if (!canCreate.value) return
  creating.value = true
  try {
    const created = await createCustomerProfile({
      displayName: createName.value.trim(),
      profileType: createType.value,
      extraData: '',
    })
    showCreate.value = false
    void refresh()
    router.push({ name: 'customer-detail', params: { id: created.id } })
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    creating.value = false
  }
}

// ── Merge preview wiring (from SuggestedMergesList) ──

const mergeSourceId = ref<number | null>(null)
const mergeTargetId = ref<number | null>(null)
const showMergePreview = ref(false)

function onSuggestedPreview(payload: { sourceProfileId: number; targetProfileId: number }): void {
  mergeSourceId.value = payload.sourceProfileId
  mergeTargetId.value = payload.targetProfileId
  showMergePreview.value = true
}

function onMergePreviewVisibility(visible: boolean): void {
  showMergePreview.value = visible
  if (!visible) {
    mergeSourceId.value = null
    mergeTargetId.value = null
  }
}

function onMerged(_result: MergeProfilesResult): void {
  void refresh()
}

async function onMergeUndone(result: UndoCustomerMergeResult): Promise<void> {
  await refresh()
  await router.push({ name: 'customer-detail', params: { id: result.restoredSourceProfileId } })
}
</script>

<template>
  <div class="customers-page">
    <PageHeader :title="t('customerList.title')" :description="t('customerList.subtitle')">
      <template #actions>
        <NButton type="primary" @click="openCreate">{{ t('customerList.createAction') }}</NButton>
      </template>
    </PageHeader>

    <SectionCard flat>
      <div class="customers-page__filters">
        <NInput
          :value="keywordDraft"
          class="customers-page__filter-keyword"
          :placeholder="t('customerList.searchPlaceholder')"
          clearable
          @update:value="onKeywordInput"
        />
        <NSelect
          v-model:value="platform"
          class="customers-page__filter-platform"
          :options="platformSelectOptions"
          :placeholder="t('customerList.filter.platformLabel')"
        />
        <label class="customers-page__filter-switch">
          <NSwitch v-model:value="missingAddressOnly" />
          <span>{{ t('customerList.filter.missingAddressOnlyLabel') }}</span>
        </label>
      </div>
    </SectionCard>

    <SuggestedMergesList @preview="onSuggestedPreview" />

    <DataGrid
      :columns="columns"
      :rows="profiles"
      row-key="id"
      :loading="loading"
      :pagination="{ server: { total: totalCount, page, pageSize, onChange: onPageChange, onSort } }"
      :empty="{ title: t('customerList.empty.title'), description: t('customerList.empty.description') }"
      @row-click="openDetail"
    />

    <DetailDrawer :show="showPreview" :title="t('customerList.quickPreview.title')" size="md" @update:show="onPreviewVisibility">
      <template v-if="previewProfile">
        <div class="customers-page__preview-field">
          <span class="customers-page__preview-label">{{ t('customerDetail.profile.displayNameLabel') }}</span>
          <span class="customers-page__preview-value">{{ previewProfile.displayName }}</span>
        </div>
        <div class="customers-page__preview-field">
          <span class="customers-page__preview-label">{{ t('customerDetail.profile.profileTypeLabel') }}</span>
          <StatusBadge dimension="profileType" :value="previewProfile.profileType" size="sm" />
        </div>
        <div class="customers-page__preview-field">
          <span class="customers-page__preview-label">{{ t('customerList.columns.identities') }}</span>
          <span class="customers-page__preview-value">{{ previewProfile.identities?.length ?? 0 }}</span>
        </div>
        <div class="customers-page__preview-field">
          <span class="customers-page__preview-label">{{ t('customerList.columns.addresses') }}</span>
          <span class="customers-page__preview-value">{{ previewProfile.activeAddressCount }}</span>
        </div>
      </template>
      <template #footer>
        <NButton type="primary" @click="previewProfile && openDetail(previewProfile)">
          {{ t('customerList.quickPreview.viewDetail') }}
        </NButton>
      </template>
    </DetailDrawer>

    <NModal
      :show="showCreate"
      preset="card"
      :title="t('customerList.createAction')"
      :style="{ width: 'min(420px, 92vw)' }"
      :mask-closable="!creating"
      :close-on-esc="!creating"
      @update:show="(v: boolean) => (showCreate = v)"
    >
      <NForm label-placement="top">
        <NFormItem :label="t('customerDetail.profile.displayNameLabel')">
          <NInput v-model:value="createName" :disabled="creating" @keydown.enter.prevent="handleCreate" />
        </NFormItem>
        <NFormItem :label="t('customerDetail.profile.profileTypeLabel')">
          <NSelect v-model:value="createType" :options="profileTypeOptions" :disabled="creating" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="customers-page__create-footer">
          <NButton :disabled="creating" @click="showCreate = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="creating" :disabled="!canCreate" @click="handleCreate">
            {{ t('common.confirm') }}
          </NButton>
        </div>
      </template>
    </NModal>

    <MergePreviewDialog
      :show="showMergePreview"
      :source-profile-id="mergeSourceId"
      :target-profile-id="mergeTargetId"
      @update:show="onMergePreviewVisibility"
      @merged="onMerged"
      @undone="onMergeUndone"
    />
  </div>
</template>

<style scoped>
.customers-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.customers-page__filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
}

.customers-page__filter-keyword {
  flex: 1 1 240px;
  min-width: 200px;
}

.customers-page__filter-platform {
  flex: 0 1 200px;
  min-width: 160px;
}

.customers-page__filter-switch {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
}

.customers-page__preview-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.customers-page__preview-label {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.customers-page__preview-value {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.customers-page__create-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>

<style>
/* Unscoped: `createColumns`' `actions` render() runs outside this SFC's
   scoped subtree (same reasoning as `WavesPage.vue`'s row-actions class). */
.customers-page__row-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}
</style>
