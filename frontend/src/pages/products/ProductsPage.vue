<script setup lang="ts">
/**
 * ProductsPage — the product-master list (plan §3.7 first half): keyword
 * search + productKind filter + an archive/active view toggle, multi-select
 * -> batch-stock-to-wave, and a create/edit form. The list is filtered,
 * sorted, and paginated by the backend.
 */
import { computed, h, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSelect, NSpin, NSwitch, NTag } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { FilterBar } from '@/shared/ui/filter-bar'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import {
  getDefaultTemplateForProfile,
  importProductCatalog,
  listProfiles,
  pickCatalogImportFile,
} from '@/shared/api/bridge'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import { useProductsPage } from './useProductsPage'
import ProductEditDrawer from './ProductEditDrawer.vue'
import BatchStockToWaveDialog from './BatchStockToWaveDialog.vue'
import { localImageUrl, type ProductMaster } from '@/entities/product'
import { canImportProductCatalog } from '@/pages/integrations/profileAvailability'
import ImportEvidenceReference from '@/shared/ui/customer-resolution/ImportEvidenceReference.vue'

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
  const selected = new Set(keys.map((key) => Number(key)))
  for (const master of page.masters.value) {
    if (selected.has(Number(master.id))) selectedMasterCache.set(Number(master.id), master)
  }
  for (const key of selectedKeys.value) {
    const id = Number(key)
    if (!selected.has(id)) selectedMasterCache.delete(id)
  }
  selectedKeys.value = keys
}

watch(
  () => page.masters.value,
  (masters) => {
    const selected = new Set(selectedKeys.value.map((key) => Number(key)))
    for (const master of masters) {
      if (selected.has(Number(master.id))) selectedMasterCache.set(Number(master.id), master)
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

// ── Catalog import (ImportProductCatalog) ──
// Document type is implied — this entry is catalog-only; no documentType picker.
const CATALOG_DOCUMENT_TYPE = 'import_product_catalog'

const showCatalogImport = ref(false)
const catalogProfileId = ref<number | null>(null)
const catalogProfiles = ref<SelectOption[]>([])
const catalogImporting = ref(false)
const catalogProfilesLoading = ref(false)
const catalogProfilesError = ref<string | null>(null)
const catalogImportEvidence = ref<{ importRunId: number; evidenceDisabled: boolean } | null>(null)
const catalogBindingKey = ref<string | null>(null)
const catalogBindingPresent = ref(false)
const catalogBindingLoading = ref(false)
const catalogBindingError = ref<string | null>(null)
let catalogBindingSeq = 0

const catalogPickDisabled = computed(
  () =>
    catalogProfileId.value == null
    || catalogProfilesLoading.value
    || !!catalogProfilesError.value
    || catalogImportEvidence.value != null
    || catalogBindingLoading.value
    || !catalogBindingPresent.value,
)

function resetCatalogBindingState(): void {
  catalogBindingSeq += 1
  catalogBindingKey.value = null
  catalogBindingPresent.value = false
  catalogBindingLoading.value = false
  catalogBindingError.value = null
}

async function loadCatalogDefaultBinding(profileId: number | null): Promise<void> {
  const seq = ++catalogBindingSeq
  catalogBindingKey.value = null
  catalogBindingPresent.value = false
  catalogBindingError.value = null
  if (profileId == null) {
    catalogBindingLoading.value = false
    return
  }
  catalogBindingLoading.value = true
  try {
    const tmpl = await getDefaultTemplateForProfile(profileId, CATALOG_DOCUMENT_TYPE)
    if (seq !== catalogBindingSeq) return
    if (tmpl) {
      catalogBindingKey.value = tmpl.templateKey
      catalogBindingPresent.value = true
    }
  } catch (err) {
    if (seq !== catalogBindingSeq) return
    catalogBindingError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (seq === catalogBindingSeq) catalogBindingLoading.value = false
  }
}

watch(catalogProfileId, (id) => {
  void loadCatalogDefaultBinding(id)
})

async function openCatalogImport(): Promise<void> {
  showCatalogImport.value = true
  catalogProfileId.value = null
  catalogProfilesLoading.value = true
  catalogProfilesError.value = null
  catalogImportEvidence.value = null
  resetCatalogBindingState()
  try {
    const profiles = await listProfiles()
    catalogProfiles.value = profiles
      .filter(canImportProductCatalog)
      .map((p) => ({
        label: `${p.profileKey} (${p.factorySupplierPlatform || p.sourceChannel})`,
        value: p.id,
      }))
  } catch (err) {
    catalogProfiles.value = []
    catalogProfilesError.value = err instanceof Error ? err.message : String(err)
  } finally {
    catalogProfilesLoading.value = false
  }
}

async function runCatalogImport(): Promise<void> {
  if (catalogProfileId.value == null || !catalogBindingPresent.value) return
  catalogImporting.value = true
  try {
    const path = await pickCatalogImportFile()
    if (!path) return
    const result = await importProductCatalog({
      integrationProfileId: catalogProfileId.value,
      importMode: 'skip_invalid',
      filePath: path,
    })
    catalogImportEvidence.value = {
      importRunId: result.importRunId,
      evidenceDisabled: result.evidenceDisabled,
    }
    if (result.errorCount > 0) {
      feedback.error(
        t('feedback.error'),
        t('products.catalogImport.partial', {
          success: result.successCount,
          errors: result.errorCount,
        }),
      )
    } else {
      feedback.success(t('products.catalogImport.success', { count: result.successCount }))
    }
    if (result.warnings && result.warnings.length > 0) {
      feedback.info(
        t('products.catalogImport.warnings', {
          count: result.warnings.length,
          items: t('products.catalogImport.warningDetailsWithheld'),
        }),
      )
    }
    void page.load()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    catalogImporting.value = false
  }
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
    {
      type: 'text',
      key: 'coverImagePath',
      title: t('products.columns.cover'),
      width: 72,
      sortable: false,
      render: (row) => {
        const src = localImageUrl(row.coverImagePath)
        if (src) {
          return h('img', {
            class: 'products-page__cover',
            src,
            alt: row.name || '',
            loading: 'lazy',
          })
        }
        return h('div', {
          class: 'products-page__cover products-page__cover--placeholder',
          title: t('products.coverPlaceholder'),
        })
      },
    },
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
        <NButton @click="openCatalogImport">{{ t('products.catalogImport.action') }}</NButton>
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

    <ErrorBanner
      v-if="page.error.value"
      :message="t('feedback.error')"
      :detail="page.error.value"
      @retry="page.load"
    />

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
      :empty="page.error.value
        ? { title: t('feedback.error') }
        : { title: t('products.empty.title'), description: t('products.empty.description') }"
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

    <NModal
      v-model:show="showCatalogImport"
      preset="card"
      :title="t('products.catalogImport.title')"
      style="width: 480px"
    >
      <p class="products-page__catalog-hint">{{ t('products.catalogImport.hint') }}</p>
      <p class="products-page__catalog-doc-type">
        <span>{{
          t('products.catalogImport.documentTypeNote', {
            type: t('glossary.documentType.import_product_catalog.label'),
          })
        }}</span>
        <StatusBadge dimension="documentType" :value="CATALOG_DOCUMENT_TYPE" size="sm" />
      </p>
      <ImportEvidenceReference
        v-if="catalogImportEvidence"
        :import-run-id="catalogImportEvidence.importRunId"
        :evidence-disabled="catalogImportEvidence.evidenceDisabled"
      />
      <NSpin v-if="catalogProfilesLoading" size="small" />
      <ErrorBanner
        v-else-if="catalogProfilesError"
        :message="t('products.catalogImport.profileLoadFailed')"
        :detail="catalogProfilesError"
        @retry="openCatalogImport"
      />
      <NSelect
        v-else
        v-model:value="catalogProfileId"
        :options="catalogProfiles"
        filterable
        :placeholder="t('products.catalogImport.profilePlaceholder')"
      />
      <NSpin v-if="catalogBindingLoading" size="small" class="products-page__catalog-binding" />
      <CalloutBar
        v-else-if="catalogProfileId != null && catalogBindingPresent"
        class="products-page__catalog-binding"
        tone="info"
        :message="t('products.catalogImport.defaultBindingLoaded', { key: catalogBindingKey ?? '' })"
      />
      <CalloutBar
        v-else-if="catalogProfileId != null"
        class="products-page__catalog-binding"
        :tone="catalogBindingError ? 'error' : 'warning'"
        :message="catalogBindingError || t('products.catalogImport.defaultBindingMissing')"
      />
      <div class="products-page__catalog-actions">
        <NButton @click="showCatalogImport = false">{{ catalogImportEvidence ? t('common.close') : t('common.cancel') }}</NButton>
        <NButton
          type="primary"
          :disabled="catalogPickDisabled"
          :loading="catalogImporting"
          @click="runCatalogImport"
        >
          {{ t('products.catalogImport.pickAndImport') }}
        </NButton>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.products-page__catalog-hint {
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.products-page__catalog-doc-type {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.products-page__catalog-binding {
  margin-top: var(--space-3);
}

.products-page__catalog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-4);
}

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

.products-page__cover {
  display: block;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm, 4px);
  object-fit: cover;
  background: var(--color-surface-muted, #f0f0f0);
}

.products-page__cover--placeholder {
  border: 1px dashed var(--color-border, #d0d0d0);
  background: var(--color-surface-muted, #f5f5f5);
}
</style>
