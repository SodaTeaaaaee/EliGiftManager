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
  exportFactoryOrder,
  generateFactoryOrder,
  generateWritebacks,
  importShipment,
  listAddresses,
  listCustomers,
  listPlatforms,
  listProducts,
  listResultViews,
  listSupplierOrderLines,
  listSupplierOrders,
  setResultAddress,
  voidFactoryOrder,
} from '@/shared/api/bridge'
import type {
  CustomerProfile,
  Platform,
  ProductItem,
  RecipientAddress,
  ResultView,
  SupplierOrder,
  SupplierOrderLine,
  Wave,
} from '@/entities/models'

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

const showAddressModal = ref(false)
const targetResultId = ref<number | null>(null)
const availableAddresses = ref<RecipientAddress[]>([])
const selectedAddressId = ref<number | null>(null)

const showFactoryOrderModal = ref(false)
const selectedFactoryId = ref<number | null>(null)

const showOrdersDrawer = ref(false)
const selectedOrderLines = ref<Record<number, SupplierOrderLine[]>>({})

const showShipmentModal = ref(false)
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
    await exportFactoryOrder(orderId)
    await loadData()
  } catch (err) {
    console.error('Failed to export order:', err)
  } finally {
    actionLoading.value = false
  }
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
          <NButton size="small" type="primary" @click="openGenerateFactoryOrder">
            {{ t('waveWorkspace.generateFactoryOrder') }}
          </NButton>
          <NButton size="small" @click="openOrdersDrawer">
            {{ t('waveWorkspace.supplierOrders') }} ({{ supplierOrders.length }})
          </NButton>
          <NButton size="small" @click="showShipmentModal = true">
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
            :columns="columns"
            :data="filteredResults"
            :row-key="(row: ResultView) => row.Result.ID"
            size="small"
          />
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

    <!-- Import Shipment Modal -->
    <NModal
      v-model:show="showShipmentModal"
      preset="card"
      :title="t('waveWorkspace.importShipment')"
      style="width: 480px"
    >
      <NForm label-placement="left" label-width="110">
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
            type="primary"
            :loading="actionLoading"
            :disabled="!shipmentForm.trackingId || !shipmentForm.trackingNo"
            @click="handleImportShipment"
          >
            {{ t('common.confirm') }}
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
</style>
