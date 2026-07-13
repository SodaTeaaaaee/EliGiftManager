<script setup lang="ts">
/**
 * ProductsPage — the product-master list (plan §3.7 first half): keyword
 * search + productKind filter + an archive/active view toggle, multi-select
 * -> batch-stock-to-wave, and a create/edit form. The list is filtered,
 * sorted, and paginated by the backend.
 */
import { computed, h, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSwitch, NTag } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { FilterBar } from '@/shared/ui/filter-bar'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { useProductsPage } from './useProductsPage'
import ProductEditDrawer from './ProductEditDrawer.vue'
import BatchStockToWaveDialog from './BatchStockToWaveDialog.vue'
import type { ProductMaster } from '@/entities/product'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const page = useProductsPage()

// ── Selection + batch stock ──
const selectedKeys = ref<Array<string | number>>([])
const selectedMasterCache = reactive(new Map<number, ProductMaster>())
const showBatchStock = ref(false)

const selectedMasters = computed<ProductMaster[]>(() =>
  selectedKeys.value
    .map((key) => selectedMasterCache.get(Number(key)))
    .filter((master): master is ProductMaster => master != null),
)

function openBatchStock(): void {
  if (selectedMasters.value.length === 0) return
  showBatchStock.value = true
}

function handleSelectedKeysChange(keys: Array<string | number>): void {
  for (const master of page.masters.value) {
    if (keys.includes(master.id)) selectedMasterCache.set(master.id, master)
  }
  for (const key of selectedKeys.value) {
    if (!keys.includes(key)) selectedMasterCache.delete(Number(key))
  }
  selectedKeys.value = keys
}

watch(
  () => page.masters.value,
  (masters) => {
    for (const master of masters) {
      if (selectedKeys.value.includes(master.id)) selectedMasterCache.set(master.id, master)
    }
  },
)

function onBatchStockSuccess(): void {
  showBatchStock.value = false
  selectedKeys.value = []
  selectedMasterCache.clear()
}

// ── Create/edit ──
const showEditor = ref(false)
const editingMaster = ref<ProductMaster | null>(null)

function openCreate(): void {
  editingMaster.value = null
  showEditor.value = true
}

function openEdit(master: ProductMaster): void {
  editingMaster.value = master
  showEditor.value = true
}

function onSaved(): void {
  feedback.success(t(editingMaster.value ? 'products.editAction' : 'products.createAction'))
  void page.load()
}

async function handleToggleArchived(master: ProductMaster): Promise<void> {
  try {
    await page.toggleArchived(master)
    feedback.success(t(master.archived ? 'products.unarchiveAction' : 'products.archiveAction'))
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}

// ── Grid columns ──
const columns = computed(() => {
  const specs: DataGridColumnSpec<ProductMaster>[] = [
    { type: 'text', key: 'name', title: t('products.columns.name'), minWidth: 180 },
    { type: 'text', key: 'supplierPlatform', title: t('products.columns.supplierPlatform'), width: 130 },
    { type: 'text', key: 'factorySku', title: t('products.columns.factorySku'), width: 140 },
    { type: 'text', key: 'supplierProductRef', title: t('products.columns.supplierProductRef'), width: 160 },
    {
      type: 'status',
      key: 'productKind',
      title: t('products.columns.productKind'),
      dimension: 'productKind',
      width: 120,
    },
    {
      type: 'text',
      key: 'archived',
      title: t('products.columns.archived'),
      width: 110,
      sortable: true,
      getValue: (row) => (row.archived ? t('products.archivedBadge') : t('products.activeBadge')),
      render: (row) =>
        h(
          NTag,
          { size: 'small', type: row.archived ? 'default' : 'success', round: true },
          { default: () => (row.archived ? t('products.archivedBadge') : t('products.activeBadge')) },
        ),
    },
    {
      type: 'actions',
      key: 'actions',
      title: t('products.columns.actions'),
      width: 180,
      render: (row) =>
        h('div', { class: 'products-page__row-actions' }, [
          h(
            NButton,
            {
              size: 'tiny',
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation()
                openEdit(row)
              },
            },
            { default: () => t('products.editAction') },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation()
                void handleToggleArchived(row)
              },
            },
            { default: () => t(row.archived ? 'products.unarchiveAction' : 'products.archiveAction') },
          ),
        ]),
    },
  ]
  return createColumns<ProductMaster>(specs)
})
</script>

<template>
  <div class="products-page">
    <PageHeader :title="t('products.title')" :description="t('products.subtitle')">
      <template #actions>
        <NButton type="primary" @click="openCreate">{{ t('products.createAction') }}</NButton>
      </template>
    </PageHeader>

    <div class="products-page__filters">
      <FilterBar :filters="page.filters" class="products-page__filter-bar" />
      <label class="products-page__archive-toggle">
        <NSwitch v-model:value="page.archivedOnly.value" size="small" />
        <span>{{ t('products.filter.archiveViewLabel') }}</span>
      </label>
    </div>

    <DataGrid
      :columns="columns"
      :rows="page.masters.value"
      row-key="id"
      :loading="page.loading.value"
      selectable
      :selected-keys="selectedKeys"
      :pagination="{
        server: {
          total: page.totalCount.value,
          page: page.page.value,
          pageSize: page.pageSize.value,
          onChange: page.onPageChange,
          onSort: page.onSort,
        },
      }"
      :empty="{ title: t('products.empty.title'), description: t('products.empty.description') }"
      @update:selected-keys="handleSelectedKeysChange"
      @row-click="openEdit"
    >
      <template #selection-toolbar="{ selectedKeys: keys, clearSelection }">
        <span class="products-page__selection-count">
          {{ t('uiKit.dataGrid.selectionToolbar.countLabel', { n: keys.length }) }}
        </span>
        <NButton size="tiny" type="primary" @click="openBatchStock">{{ t('products.batchStock.action') }}</NButton>
        <button type="button" class="products-page__selection-clear" @click="clearSelection">
          {{ t('uiKit.dataGrid.selectionToolbar.clear') }}
        </button>
      </template>
    </DataGrid>

    <ProductEditDrawer v-model:show="showEditor" :master="editingMaster" @saved="onSaved" />
    <BatchStockToWaveDialog
      v-model:show="showBatchStock"
      :selected-masters="selectedMasters"
      @success="onBatchStockSuccess"
    />
  </div>
</template>

<style scoped>
.products-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.products-page__filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.products-page__filter-bar {
  flex: 1 1 auto;
}

.products-page__archive-toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
}

.products-page__selection-count {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.products-page__selection-clear {
  margin-left: auto;
  border: none;
  background: transparent;
  color: var(--color-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  padding: 0;
}

.products-page__selection-clear:hover {
  color: var(--color-accent-hover);
  text-decoration: underline;
}
</style>

<style>
/* Unscoped: `createColumns`' `actions` render() runs outside this SFC's
   scoped subtree (same reasoning as WavesPage.vue's row-actions class). */
.products-page__row-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}
</style>
