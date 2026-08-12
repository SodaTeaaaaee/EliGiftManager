<script setup lang="ts">
/**
 * InboxPage — the global demand-inbox page (plan P4 §3.5). Master-detail:
 * `PageHeader` (import CSV / manual entry actions) + the bespoke 3-way
 * assignment toggle (per foundations decision #5, NOT a FilterBar
 * dimension) + the business-surface segmented control (all / membership /
 * retail — folds into the `demandKind` filter via `businessSurface.ts`) +
 * `FilterBar`/`SavedViews` (the `demandKind` + `routingDisposition`
 * dimensions) + `DataGrid` (server-paginated via `useInboxGrid`) + row-click ->
 * `RowDetailPanel` + `#selection-toolbar` -> `BatchActionBar`.
 *
 * Wires together `useInboxGrid` + `inbox-grid/{filter-schema,columns}` with
 * `BatchActionBar` and `RowDetailPanel`, reacting to their `'done'`/
 * `'changed'` emits by calling `mutationDone()` (refetches the current
 * page/filter state in place — no route remount).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NRadioButton, NRadioGroup } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { FilterBar, SavedViews } from '@/shared/ui/filter-bar'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { useInboxGrid } from './inbox-grid/useInboxGrid'
import { buildInboxColumns } from './inbox-grid/columns'
import { kindsFromSurface, surfaceFromKinds, type BusinessSurface } from './inbox-grid/businessSurface'
import BatchActionBar from './inbox-grid/BatchActionBar.vue'
import RowDetailPanel from './inbox-grid/RowDetailPanel.vue'
import ImportFileModal from './ImportFileModal.vue'
import ManualEntryModal from './ManualEntryModal.vue'
import type { DemandInboxRow } from '@/entities/demand'

const { t } = useI18n({ useScope: 'global' })

const { assignment, filters, rows, loading, selectedKeys, selectedRows, page, pageSize, totalCount, onPageChange, onSort, mutationDone } =
  useInboxGrid()

const columns = computed(() => createColumns(buildInboxColumns(t)))

const businessSurface = computed<BusinessSurface>(() => surfaceFromKinds(filters.state.demandKind))

function handleSurfaceChange(surface: BusinessSurface): void {
  filters.setEnumValues('demandKind', kindsFromSurface(surface))
}

function handleSelectedKeysChange(keys: Array<string | number>): void {
  selectedKeys.value = keys as number[]
}

// ── Row detail panel ──

const detailRow = ref<DemandInboxRow | null>(null)
const showDetail = ref(false)

function handleRowClick(row: DemandInboxRow): void {
  detailRow.value = row
  showDetail.value = true
}

function handleDetailVisibility(visible: boolean): void {
  showDetail.value = visible
}

// ── Import / manual-entry modals ──

const showImportModal = ref(false)
const showManualEntryModal = ref(false)
</script>

<template>
  <div class="inbox-page">
    <PageHeader :title="t('inbox.title')" :description="t('inbox.subtitle')">
      <template #actions>
        <NButton secondary @click="showManualEntryModal = true">{{ t('inbox.manualEntryButton') }}</NButton>
        <NButton type="primary" @click="showImportModal = true">{{ t('inbox.importFileButton') }}</NButton>
      </template>
    </PageHeader>

    <div class="inbox-page__assignment">
      <span class="inbox-page__assignment-label">{{ t('inbox.filters.assignment') }}</span>
      <NRadioGroup v-model:value="assignment">
        <NRadioButton value="all">{{ t('inbox.assignment.all') }}</NRadioButton>
        <NRadioButton value="assigned">{{ t('inbox.assignment.assigned') }}</NRadioButton>
        <NRadioButton value="unassigned">{{ t('inbox.assignment.unassigned') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <div class="inbox-page__surface">
      <span class="inbox-page__assignment-label">{{ t('inbox.filters.businessSurface') }}</span>
      <NRadioGroup :value="businessSurface" @update:value="handleSurfaceChange">
        <NRadioButton value="all">{{ t('inbox.surface.all') }}</NRadioButton>
        <NRadioButton value="membership_entitlement">{{ t('inbox.surface.membership') }}</NRadioButton>
        <NRadioButton value="retail_order">{{ t('inbox.surface.retail') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <SavedViews :filters="filters" scope-id="inbox-grid" />
    <FilterBar :filters="filters" />

    <DataGrid
      :columns="columns"
      :rows="rows"
      row-key="demandDocumentId"
      selectable
      :selected-keys="selectedKeys"
      :loading="loading"
      :pagination="{ server: { total: totalCount, page: page, pageSize: pageSize, onChange: onPageChange, onSort } }"
      :empty="{ title: t('inbox.empty.noRows') }"
      @update:selected-keys="handleSelectedKeysChange"
      @row-click="handleRowClick"
    >
      <template #selection-toolbar>
        <BatchActionBar :selected-rows="selectedRows" @done="mutationDone" />
      </template>
    </DataGrid>

    <RowDetailPanel :row="detailRow" :show="showDetail" @update:show="handleDetailVisibility" @changed="mutationDone" />

    <ImportFileModal v-model:show="showImportModal" @imported="mutationDone" />
    <ManualEntryModal v-model:show="showManualEntryModal" @created="mutationDone" />
  </div>
</template>

<style scoped>
.inbox-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.inbox-page__assignment,
.inbox-page__surface {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.inbox-page__assignment-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}
</style>
