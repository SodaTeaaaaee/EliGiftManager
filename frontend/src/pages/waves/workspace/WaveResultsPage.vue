<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NDrawer,
  NDrawerContent,
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import {
  exportFactoryOrderFile,
  generateFactoryOrder,
  generateWritebacks,
  importShipment,
  importShipmentFile,
  listAddresses,
  listCustomers,
  listPlatforms,
  listProducts,
  listResultViews,
  listSupplierOrderLines,
  listSupplierOrders,
  pickFile,
  revealInFolder,
  setResultAddress,
  voidFactoryOrder,
} from '@/shared/api/bridge'
import type {
  CustomerProfile,
  ExportFileResult,
  ImportShipmentFileResult,
  Platform,
  ProductItem,
  RecipientAddress,
  ResultView,
  SupplierOrder,
  SupplierOrderLine,
  Wave,
} from '@/entities/models'
import { fulfillmentSourceKindValues, workStateValues } from '@/shared/api/generated/enums'

const props = defineProps<{
  waveId: number
  wave?: Wave | null
}>()

const { t } = useI18n()
const route = useRoute()

const loading = ref(false)
const actionLoading = ref(false)
const results = ref<ResultView[]>([])
const products = ref<ProductItem[]>([])
const customers = ref<CustomerProfile[]>([])
const platforms = ref<Platform[]>([])
const supplierOrders = ref<SupplierOrder[]>([])

const filterProduct = ref<string>('all')

// ── Grouping (view-only; grouping never changes the leaf facts) ──

type GroupDimension = 'workState' | 'none' | 'customer' | 'product' | 'source' | 'factory'

const groupBy = ref<GroupDimension>('workState')

const groupOptions = computed(() => [
  { label: t('waveWorkspace.groupWorkState'), value: 'workState' },
  { label: t('waveWorkspace.groupNone'), value: 'none' },
  { label: t('waveWorkspace.groupCustomer'), value: 'customer' },
  { label: t('waveWorkspace.groupProduct'), value: 'product' },
  { label: t('waveWorkspace.groupSource'), value: 'source' },
  { label: t('waveWorkspace.groupFactory'), value: 'factory' },
])

interface ResultGroup {
  key: string
  title: string
  /** Optional glossary badge rendered instead of the plain title. */
  badge?: { dimension: 'workState' | 'fulfillmentSourceKind'; value: string }
  rows: ResultView[]
  qty: number
  order: number
}

function productLabel(productId: number | null | undefined): string {
  if (productId == null) return t('waveWorkspace.groupNoProduct')
  const prod = products.value.find((p) => p.ID === productId)
  return prod ? `${prod.Name} (${prod.FactorySKU})` : `#${productId}`
}

/** Factory dimension: committed results group under their product's factory
 * platform; everything not yet in a factory order falls into one bucket. */
function factoryGroupKey(row: ResultView): { key: string; title: string; order: number } {
  if (row.InFactory && row.Result.ProductItemID != null) {
    const prod = products.value.find((p) => p.ID === row.Result.ProductItemID)
    const factory = prod ? platforms.value.find((p) => p.ID === prod.FactoryPlatformID) : undefined
    if (factory) {
      return { key: `factory-${factory.ID}`, title: factory.Name, order: 1 }
    }
  }
  return { key: 'factory-none', title: t('waveWorkspace.notInFactory'), order: 0 }
}

const groupedResults = computed<ResultGroup[]>(() => {
  if (groupBy.value === 'none') return []
  const groups = new Map<string, ResultGroup>()
  for (const row of filteredResults.value) {
    let key = ''
    let title = ''
    let badge: ResultGroup['badge'] = undefined
    let order = 0
    switch (groupBy.value) {
      case 'workState': {
        const state = row.WorkState || 'ready'
        key = `state-${state}`
        badge = { dimension: 'workState', value: state }
        order = workStateValues.indexOf(state as (typeof workStateValues)[number])
        break
      }
      case 'customer': {
        const custId = row.Result.CustomerProfileID
        key = custId == null ? 'customer-none' : `customer-${custId}`
        title =
          custId == null
            ? t('waveWorkspace.groupNoCustomer')
            : customers.value.find((c) => c.ID === custId)?.DisplayName ?? `#${custId}`
        order = 1
        break
      }
      case 'product': {
        const pid = row.Result.ProductItemID
        key = pid == null ? 'product-none' : `product-${pid}`
        title = productLabel(pid)
        order = 1
        break
      }
      case 'source': {
        const kind = row.Result.SourceKind || 'entitlement_instance'
        key = `source-${kind}`
        badge = { dimension: 'fulfillmentSourceKind', value: kind }
        order = fulfillmentSourceKindValues.indexOf(
          kind as (typeof fulfillmentSourceKindValues)[number],
        )
        break
      }
      case 'factory': {
        const g = factoryGroupKey(row)
        key = g.key
        title = g.title
        order = g.order
        break
      }
    }
    let group = groups.get(key)
    if (!group) {
      group = { key, title, badge, rows: [], qty: 0, order }
      groups.set(key, group)
    }
    group.rows.push(row)
    group.qty += row.Result.Quantity
  }
  const list = [...groups.values()].filter((g) => g.rows.length > 0)
  list.sort((a, b) => {
    if (a.order !== b.order) return a.order - b.order
    return a.title.localeCompare(b.title, undefined, { numeric: true })
  })
  return list
})

// ── Row selection (shared across grouped tables) ──

const selectedResultKeys = ref<number[]>([])

function groupCheckedKeys(group: ResultGroup): number[] {
  const ids = new Set(group.rows.map((r) => r.Result.ID))
  return selectedResultKeys.value.filter((id) => ids.has(id))
}

function handleGroupChecked(group: ResultGroup, keys: Array<number | string>): void {
  const ids = new Set(group.rows.map((r) => r.Result.ID))
  const kept = selectedResultKeys.value.filter((id) => !ids.has(id))
  const incoming = keys.filter((k): k is number => typeof k === 'number')
  selectedResultKeys.value = [...kept, ...incoming]
}

const showAddressModal = ref(false)
const targetResultId = ref<number | null>(null)
const availableAddresses = ref<RecipientAddress[]>([])
const selectedAddressId = ref<number | null>(null)

const showFactoryOrderModal = ref(false)
const selectedFactoryId = ref<number | null>(null)

const showOrdersDrawer = ref(false)
const selectedOrderLines = ref<Record<number, SupplierOrderLine[]>>({})

// ── File-based export (ExportFactoryOrderFile) ──

const showExportReceiptModal = ref(false)
const exportResult = ref<ExportFileResult | null>(null)

// ── Shipment return import (ImportShipmentFile + manual fallback) ──

const shipmentFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]
const showShipmentModal = ref(false)
const shipmentPlatformId = ref<number | null>(null)
const shipmentFilePath = ref('')
const shipmentResult = ref<ImportShipmentFileResult | null>(null)
const shipmentError = ref('')
const showManualShipment = ref(false)
const shipmentForm = ref({
  trackingId: '',
  trackingNo: '',
  carrierCode: 'SF',
  carrierName: '顺丰速运',
  quantity: 1,
})

const showWritebackModal = ref(false)
const writebackFactId = ref<number | null>(null)

async function loadData() {
  if (!props.waveId) return
  loading.value = true
  try {
    const [resViews, prods, custs, plats, orders] = await Promise.all([
      listResultViews(props.waveId),
      listProducts(),
      listCustomers(),
      listPlatforms(),
      listSupplierOrders(props.waveId),
    ])
    results.value = resViews
    products.value = prods
    customers.value = custs
    platforms.value = plats
    supplierOrders.value = orders
  } catch (err) {
    console.error('Failed to load wave results:', err)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.waveId,
  () => {
    void loadData()
  },
)

onMounted(() => {
  if (route.query.product) {
    filterProduct.value = String(route.query.product)
  }
  void loadData()
})

const factoryPlatformOptions = computed(() =>
  platforms.value
    .filter((p) => p.Kind === 'factory')
    .map((p) => ({
      label: `${p.Name} (${p.Key})`,
      value: p.ID,
    })),
)

const filteredResults = computed(() => {
  let list = results.value
  if (filterProduct.value !== 'all') {
    const pid = Number(filterProduct.value)
    list = list.filter((r) => r.Result.ProductItemID === pid)
  }
  return list
})

async function openAddressModal(result: ResultView) {
  if (result.Result.Frozen || !result.Result.CustomerProfileID) return
  targetResultId.value = result.Result.ID
  actionLoading.value = true
  try {
    availableAddresses.value = await listAddresses(result.Result.CustomerProfileID)
    selectedAddressId.value = availableAddresses.value[0]?.ID ?? null
    showAddressModal.value = true
  } catch (err) {
    console.error('Failed to list addresses:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleSaveAddress() {
  if (!targetResultId.value || !selectedAddressId.value) return
  actionLoading.value = true
  try {
    await setResultAddress(targetResultId.value, selectedAddressId.value)
    showAddressModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to set address:', err)
  } finally {
    actionLoading.value = false
  }
}

function openGenerateFactoryOrder() {
  selectedFactoryId.value = factoryPlatformOptions.value[0]?.value ?? null
  showFactoryOrderModal.value = true
}

async function handleGenerateFactoryOrder() {
  if (!selectedFactoryId.value) return
  actionLoading.value = true
  try {
    await generateFactoryOrder(props.waveId, selectedFactoryId.value)
    showFactoryOrderModal.value = false
    selectedResultKeys.value = []
    await loadData()
  } catch (err) {
    console.error('Failed to generate factory order:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleExportOrder(orderId: number) {
  actionLoading.value = true
  try {
    exportResult.value = await exportFactoryOrderFile(orderId)
    showExportReceiptModal.value = true
    await loadData()
  } catch (err) {
    console.error('Failed to export order:', err)
  } finally {
    actionLoading.value = false
  }
}

function handleRevealExportFile() {
  if (exportResult.value?.Path) void revealInFolder(exportResult.value.Path)
}

async function handleVoidOrder(orderId: number) {
  actionLoading.value = true
  try {
    await voidFactoryOrder(orderId)
    await loadData()
  } catch (err) {
    console.error('Failed to void order:', err)
  } finally {
    actionLoading.value = false
  }
}

async function openOrdersDrawer() {
  showOrdersDrawer.value = true
  for (const order of supplierOrders.value) {
    if (!selectedOrderLines.value[order.ID]) {
      try {
        selectedOrderLines.value[order.ID] = await listSupplierOrderLines(order.ID)
      } catch (err) {
        console.error('Failed to load order lines:', err)
      }
    }
  }
}

function openShipmentModal() {
  shipmentPlatformId.value = factoryPlatformOptions.value[0]?.value ?? null
  shipmentFilePath.value = ''
  shipmentResult.value = null
  shipmentError.value = ''
  showManualShipment.value = false
  shipmentForm.value = {
    trackingId: '',
    trackingNo: '',
    carrierCode: 'SF',
    carrierName: '顺丰速运',
    quantity: 1,
  }
  showShipmentModal.value = true
}

async function handlePickShipmentFile() {
  const path = await pickFile(shipmentFileFilters)
  if (path) shipmentFilePath.value = path
}

async function handleImportShipmentFile() {
  if (!shipmentPlatformId.value || !shipmentFilePath.value) return
  actionLoading.value = true
  shipmentError.value = ''
  try {
    shipmentResult.value = await importShipmentFile(
      shipmentPlatformId.value,
      shipmentFilePath.value,
    )
    await loadData()
  } catch (err) {
    shipmentResult.value = null
    shipmentError.value = err instanceof Error ? err.message : String(err)
  } finally {
    actionLoading.value = false
  }
}

async function handleImportShipment() {
  if (!shipmentForm.value.trackingId || !shipmentForm.value.trackingNo) return
  actionLoading.value = true
  try {
    await importShipment(
      shipmentForm.value.trackingId.trim(),
      shipmentForm.value.trackingNo.trim(),
      shipmentForm.value.carrierCode.trim(),
      shipmentForm.value.carrierName.trim(),
      shipmentForm.value.quantity,
    )
    showShipmentModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to import shipment:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleGenerateWritebacks() {
  if (!writebackFactId.value) return
  actionLoading.value = true
  try {
    await generateWritebacks(writebackFactId.value)
    showWritebackModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to generate writebacks:', err)
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    type: 'selection' as const,
    // Frozen results are already committed to a factory order and can never
    // re-enter submission, so partial selection skips them.
    disabled(row: ResultView) {
      return row.Result.Frozen
    },
  },
  {
    title: '#',
    key: 'ID',
    width: 70,
    render(row: ResultView) {
      return `#${row.Result.ID}`
    },
  },
  {
    title: t('inbox.kind'),
    key: 'SourceKind',
    width: 120,
    render(row: ResultView) {
      return h(StatusBadge, {
        dimension: 'fulfillmentSourceKind',
        value: row.Result.SourceKind || 'entitlement_instance',
      })
    },
  },
  {
    title: t('library.displayName'),
    key: 'Customer',
    render(row: ResultView) {
      const cust = customers.value.find((c) => c.ID === row.Result.CustomerProfileID)
      return cust ? cust.DisplayName : '—'
    },
  },
  {
    title: t('library.productName'),
    key: 'Product',
    render(row: ResultView) {
      const prod = products.value.find((p) => p.ID === row.Result.ProductItemID)
      return prod ? `${prod.Name} (${prod.FactorySKU})` : '—'
    },
  },
  {
    title: t('inbox.quantity'),
    key: 'Quantity',
    width: 80,
    render(row: ResultView) {
      return row.Result.Quantity
    },
  },
  {
    title: t('library.fullAddress'),
    key: 'Address',
    render(row: ResultView) {
      const addr = row.Result.Address
      const hasAddr = addr && addr.recipient_name && addr.address_line1
      if (hasAddr) {
        return `${addr.recipient_name} / ${addr.phone || ''} / ${addr.address_line1}`
      }
      if (!row.Result.Frozen && row.Result.CustomerProfileID) {
        return h(
          NButton,
          {
            size: 'tiny',
            type: 'warning',
            secondary: true,
            onClick: () => openAddressModal(row),
          },
          { default: () => t('waveWorkspace.selectAddress') },
        )
      }
      return '—'
    },
  },
  {
    title: t('waves.status'),
    key: 'WorkState',
    width: 120,
    render(row: ResultView) {
      return h(StatusBadge, {
        dimension: 'workState',
        value: row.WorkState || 'ready',
      })
    },
  },
  {
    title: t('waveWorkspace.blocks'),
    key: 'Blocks',
    render(row: ResultView) {
      if (!row.Blocks || !row.Blocks.length) return '—'
      return h(
        NSpace,
        { size: 'small' },
        () => row.Blocks.map((b) => h(StatusBadge, { dimension: 'blockReason', value: b })),
      )
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 100,
    render(row: ResultView) {
      if (row.Result.InputFactID) {
        return h(
          NButton,
          {
            size: 'tiny',
            quaternary: true,
            type: 'info',
            onClick: () => {
              writebackFactId.value = row.Result.InputFactID!
              showWritebackModal.value = true
            },
          },
          { default: () => t('waveWorkspace.generateWritebacks') },
        )
      }
      return null
    },
  },
]
</script>

<template>
  <div class="wave-results-page">
    <SectionCard :title="t('waveWorkspace.resultsTable')">
      <template #actions>
        <NSpace align="center">
          <NSelect
            v-model:value="groupBy"
            :options="groupOptions"
            size="small"
            class="wave-results-page__group-select"
          />
          <NButton size="small" type="primary" @click="openGenerateFactoryOrder">
            {{ t('waveWorkspace.generateFactoryOrder') }}
          </NButton>
          <NButton size="small" @click="openOrdersDrawer">
            {{ t('waveWorkspace.supplierOrders') }} ({{ supplierOrders.length }})
          </NButton>
          <NButton size="small" @click="openShipmentModal">
            {{ t('waveWorkspace.importShipment') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!filteredResults.length" class="wave-results-page__empty">
          <EmptyState :title="t('waveWorkspace.emptyResults')" size="sm" />
        </div>
        <div v-else class="wave-results-page__table">
          <NDataTable
            v-if="groupBy === 'none'"
            v-model:checked-row-keys="selectedResultKeys"
            :columns="columns"
            :data="filteredResults"
            :row-key="(row: ResultView) => row.Result.ID"
            size="small"
          />
          <template v-else>
            <div
              v-for="group in groupedResults"
              :key="group.key"
              class="wave-results-page__group"
            >
              <div class="wave-results-page__group-header">
                <StatusBadge
                  v-if="group.badge"
                  :dimension="group.badge.dimension"
                  :value="group.badge.value"
                />
                <span v-else class="wave-results-page__group-title">{{ group.title }}</span>
                <span class="wave-results-page__group-meta">
                  {{ t('waveWorkspace.groupRowCount', { n: group.rows.length }) }}
                  ·
                  {{ t('waveWorkspace.groupQtySum', { n: group.qty }) }}
                </span>
              </div>
              <NDataTable
                :columns="columns"
                :data="group.rows"
                :row-key="(row: ResultView) => row.Result.ID"
                :checked-row-keys="groupCheckedKeys(group)"
                size="small"
                @update:checked-row-keys="
                  (keys: Array<number | string>) => handleGroupChecked(group, keys)
                "
              />
            </div>
          </template>
        </div>
      </NSpin>
    </SectionCard>

    <!-- Address Modal -->
    <NModal
      v-model:show="showAddressModal"
      preset="card"
      :title="t('waveWorkspace.selectAddress')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('library.addresses')">
          <NSelect
            v-model:value="selectedAddressId"
            :options="
              availableAddresses.map((a) => ({
                label: `${a.RecipientName} - ${a.Phone || ''} (${a.AddressLine1})`,
                value: a.ID,
              }))
            "
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAddressModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!selectedAddressId"
            @click="handleSaveAddress"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Generate Factory Order Modal -->
    <NModal
      v-model:show="showFactoryOrderModal"
      preset="card"
      :title="t('waveWorkspace.generateFactoryOrder')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('waveWorkspace.factory')">
          <NSelect
            v-model:value="selectedFactoryId"
            :options="factoryPlatformOptions"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showFactoryOrderModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!selectedFactoryId"
            @click="handleGenerateFactoryOrder"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Supplier Orders Drawer -->
    <NDrawer v-model:show="showOrdersDrawer" width="600" placement="right">
      <NDrawerContent :title="t('waveWorkspace.supplierOrders')">
        <div v-if="!supplierOrders.length" class="wave-results-page__empty">
          <EmptyState :title="t('waveWorkspace.emptyResults')" size="sm" />
        </div>
        <div v-else class="wave-results-page__orders-list">
          <div
            v-for="order in supplierOrders"
            :key="order.ID"
            class="wave-results-page__order-item"
          >
            <div class="wave-results-page__order-header">
              <span class="wave-results-page__order-title">
                {{ t('waveWorkspace.orderID') }} #{{ order.ID }}
              </span>
              <StatusBadge dimension="supplierOrderStatus" :value="order.Status || 'generated'" />
            </div>

            <div v-if="selectedOrderLines[order.ID]" class="wave-results-page__order-lines">
              <div
                v-for="line in selectedOrderLines[order.ID]"
                :key="line.ID"
                class="wave-results-page__order-line"
              >
                <span>{{ `${line.FactorySKU} × ${line.Quantity}` }}</span>
                <span class="wave-results-page__tracking-tag">
                  {{ t('waveWorkspace.trackingID') }}: {{ line.TrackingID }}
                </span>
              </div>
            </div>

            <div class="wave-results-page__order-actions">
              <NSpace>
                <NButton
                  size="tiny"
                  type="primary"
                  :loading="actionLoading"
                  @click="handleExportOrder(order.ID)"
                >
                  {{ t('waveWorkspace.exportOrder') }}
                </NButton>
                <NButton
                  v-if="order.Status !== 'exported'"
                  size="tiny"
                  type="error"
                  quaternary
                  :loading="actionLoading"
                  @click="handleVoidOrder(order.ID)"
                >
                  {{ t('waveWorkspace.voidOrder') }}
                </NButton>
              </NSpace>
            </div>
          </div>
        </div>
      </NDrawerContent>
    </NDrawer>

    <!-- Export File Receipt Modal -->
    <NModal
      v-model:show="showExportReceiptModal"
      preset="card"
      :title="t('waveWorkspace.exportFileSuccess')"
      style="width: 520px"
    >
      <div v-if="exportResult" class="wave-results-page__export-receipt">
        <div class="wave-results-page__export-stats">
          <span>{{ t('waveWorkspace.orderID') }}: #{{ exportResult.Order.ID }}</span>
          <span>{{ t('waveWorkspace.exportRows', { n: exportResult.Rows.length }) }}</span>
        </div>
        <div class="wave-results-page__export-path-label">{{ t('waveWorkspace.exportFilePath') }}</div>
        <div class="wave-results-page__export-path">{{ exportResult.Path }}</div>
        <NButton size="small" @click="handleRevealExportFile">
          {{ t('waveWorkspace.revealInFolder') }}
        </NButton>
      </div>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showExportReceiptModal = false">{{ t('common.close') }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Import Shipment Modal (file-based, manual as advanced fallback) -->
    <NModal
      v-model:show="showShipmentModal"
      preset="card"
      :title="t('waveWorkspace.importShipment')"
      style="width: 560px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('waveWorkspace.factory')">
          <NSelect v-model:value="shipmentPlatformId" :options="factoryPlatformOptions" />
        </NFormItem>
        <NFormItem :label="t('inbox.filePath')">
          <div class="wave-results-page__file-row">
            <NButton size="small" @click="handlePickShipmentFile">
              {{ shipmentFilePath ? t('inbox.changeFile') : t('inbox.pickFile') }}
            </NButton>
            <span v-if="shipmentFilePath" class="wave-results-page__file-path">{{ shipmentFilePath }}</span>
            <span v-else class="wave-results-page__file-hint">{{ t('inbox.noFileSelected') }}</span>
          </div>
        </NFormItem>
      </NForm>

      <p v-if="shipmentError" class="wave-results-page__import-error">{{ shipmentError }}</p>

      <div v-if="shipmentResult" class="wave-results-page__shipment-receipt">
        <div class="wave-results-page__export-stats">
          <span>{{ t('waveWorkspace.shipmentImported', { n: shipmentResult.Imported }) }}</span>
          <span>{{ t('waveWorkspace.shipmentSkipped', { n: shipmentResult.Skipped.length }) }}</span>
        </div>
        <div v-if="shipmentResult.Skipped.length" class="wave-results-page__receipt-block">
          <div class="wave-results-page__receipt-subtitle">{{ t('waveWorkspace.skippedRows') }}</div>
          <ul class="wave-results-page__issue-list">
            <li
              v-for="skip in shipmentResult.Skipped"
              :key="`${skip.LineNo}-${skip.TrackingID}`"
              class="wave-results-page__issue-row"
            >
              <span class="wave-results-page__issue-line">#{{ skip.LineNo }}</span>
              <span class="wave-results-page__issue-key">{{ skip.TrackingID }}</span>
              <span>{{ skip.Reason }}</span>
            </li>
          </ul>
        </div>
        <div v-if="shipmentResult.Issues.length" class="wave-results-page__receipt-block">
          <div class="wave-results-page__receipt-subtitle">{{ t('inbox.issues') }}</div>
          <ul class="wave-results-page__issue-list">
            <li
              v-for="(issue, index) in shipmentResult.Issues"
              :key="index"
              class="wave-results-page__issue-row"
            >
              <span class="wave-results-page__issue-line">#{{ issue.LineNo }}</span>
              <span class="wave-results-page__issue-key">{{ issue.Key }}</span>
              <span>{{ issue.Message }}</span>
            </li>
          </ul>
        </div>
      </div>

      <NButton
        size="small"
        quaternary
        type="primary"
        class="wave-results-page__manual-toggle"
        @click="showManualShipment = !showManualShipment"
      >
        {{ t('waveWorkspace.manualShipment') }}
      </NButton>

      <NForm v-if="showManualShipment" label-placement="left" label-width="110">
        <NFormItem :label="t('waveWorkspace.trackingID')">
          <NInput v-model:value="shipmentForm.trackingId" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.trackingNo')">
          <NInput v-model:value="shipmentForm.trackingNo" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.carrier')">
          <NInput v-model:value="shipmentForm.carrierName" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.quantity')">
          <NInputNumber v-model:value="shipmentForm.quantity" :min="1" />
        </NFormItem>
      </NForm>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showShipmentModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            v-if="showManualShipment"
            :loading="actionLoading"
            :disabled="!shipmentForm.trackingId || !shipmentForm.trackingNo"
            @click="handleImportShipment"
          >
            {{ t('waveWorkspace.manualShipmentImport') }}
          </NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!shipmentPlatformId || !shipmentFilePath"
            @click="handleImportShipmentFile"
          >
            {{ t('common.import') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Generate Writebacks Modal -->
    <NModal
      v-model:show="showWritebackModal"
      preset="card"
      :title="t('waveWorkspace.generateWritebacks')"
      style="width: 440px"
    >
      <NForm>
        <NFormItem :label="t('waveWorkspace.factID')">
          <NInputNumber v-model:value="writebackFactId" :min="1" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showWritebackModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!writebackFactId"
            @click="handleGenerateWritebacks"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.wave-results-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-results-page__empty {
  padding: var(--space-4) 0;
}

.wave-results-page__table {
  width: 100%;
}

.wave-results-page__group-select {
  width: 160px;
}

.wave-results-page__group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-results-page__group + .wave-results-page__group {
  margin-top: var(--space-4);
}

.wave-results-page__group-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.wave-results-page__group-title {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.wave-results-page__group-meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-results-page__orders-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.wave-results-page__order-item {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-results-page__order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.wave-results-page__order-title {
  font-weight: var(--font-weight-semibold);
}

.wave-results-page__order-lines {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.wave-results-page__tracking-tag {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-results-page__order-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-2);
}

.wave-results-page__export-receipt {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
}

.wave-results-page__export-stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.wave-results-page__export-path-label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.wave-results-page__export-path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-primary);
  word-break: break-all;
}

.wave-results-page__file-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-results-page__file-path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  word-break: break-all;
}

.wave-results-page__file-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-results-page__import-error {
  margin: 0;
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.wave-results-page__shipment-receipt {
  margin-top: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-results-page__receipt-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.wave-results-page__receipt-subtitle {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.wave-results-page__issue-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.wave-results-page__issue-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-results-page__issue-line {
  font-family: var(--font-mono);
  color: var(--color-text-muted);
  min-width: 40px;
}

.wave-results-page__issue-key {
  color: var(--color-text-secondary);
}

.wave-results-page__manual-toggle {
  margin-top: var(--space-3);
}
</style>
