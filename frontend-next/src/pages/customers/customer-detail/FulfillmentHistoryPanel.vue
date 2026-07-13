<script setup lang="ts">
/**
 * FulfillmentHistoryPanel — the customer detail page's core section (plan
 * §3.6 line 254 / acceptance D-1: "这个人到底发了没" answerable in 3
 * seconds). Cross-wave timeline: wave · product · quantity · the four
 * fulfillment-line state dimensions (StatusBadge, never raw enum strings) ·
 * tracking number. Self-contained — fetches on mount and whenever
 * `customerProfileId` changes.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { SectionCard } from '@/shared/ui/cards'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { getCustomerFulfillmentHistory } from '@/shared/api/bridge'
import type { CustomerFulfillmentHistoryRowDTO } from '@/entities/customer'

const props = defineProps<{
  customerProfileId: number
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const rows = ref<CustomerFulfillmentHistoryRowDTO[]>([])
const loading = ref(true)

async function refresh(): Promise<void> {
  loading.value = true
  try {
    rows.value = await getCustomerFulfillmentHistory(props.customerProfileId)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    loading.value = false
  }
}

watch(() => props.customerProfileId, () => void refresh(), { immediate: true })

defineExpose({ refresh })

const columns = computed(() => {
  const specs: DataGridColumnSpec<CustomerFulfillmentHistoryRowDTO>[] = [
    {
      type: 'text',
      key: 'waveNo',
      title: t('customerDetail.fulfillmentHistory.columns.wave'),
      minWidth: 160,
      getValue: (row) => `${row.waveNo} · ${row.waveName}`,
    },
    {
      type: 'text',
      key: 'productName',
      title: t('customerDetail.fulfillmentHistory.columns.product'),
      minWidth: 180,
      getValue: (row) => (row.productSku ? `${row.productName} (${row.productSku})` : row.productName),
    },
    {
      type: 'number',
      key: 'quantity',
      title: t('customerDetail.fulfillmentHistory.columns.quantity'),
      width: 90,
    },
    {
      type: 'status',
      key: 'allocationState',
      title: t('customerDetail.fulfillmentHistory.columns.allocationState'),
      dimension: 'allocationState',
      width: 130,
      showDot: true,
    },
    {
      type: 'status',
      key: 'addressState',
      title: t('customerDetail.fulfillmentHistory.columns.addressState'),
      dimension: 'addressState',
      width: 120,
      showDot: true,
    },
    {
      type: 'status',
      key: 'supplierState',
      title: t('customerDetail.fulfillmentHistory.columns.supplierState'),
      dimension: 'supplierState',
      width: 140,
      showDot: true,
    },
    {
      type: 'status',
      key: 'channelSyncState',
      title: t('customerDetail.fulfillmentHistory.columns.channelSyncState'),
      dimension: 'channelSyncState',
      width: 140,
      showDot: true,
    },
    {
      type: 'text',
      key: 'trackingNo',
      title: t('customerDetail.fulfillmentHistory.columns.tracking'),
      minWidth: 160,
      sortable: false,
      getValue: (row) =>
        row.trackingNo
          ? `${row.trackingNo}${row.carrierName ? ` (${row.carrierName})` : ''}`
          : t('customerDetail.fulfillmentHistory.noTracking'),
    },
    {
      type: 'date',
      key: 'createdAt',
      title: t('customerDetail.fulfillmentHistory.columns.createdAt'),
      width: 130,
    },
  ]
  return createColumns<CustomerFulfillmentHistoryRowDTO>(specs)
})
</script>

<template>
  <SectionCard :title="t('customerDetail.fulfillmentHistory.title')" :description="t('customerDetail.fulfillmentHistory.subtitle')">
    <EmptyState v-if="!loading && rows.length === 0" :title="t('customerDetail.fulfillmentHistory.empty')" size="sm" />
    <DataGrid
      v-else
      :columns="columns"
      :rows="rows"
      row-key="fulfillmentLineId"
      :loading="loading"
      pagination="client"
    />
  </SectionCard>
</template>
