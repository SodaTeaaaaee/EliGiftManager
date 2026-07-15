<script setup lang="ts">
/**
 * BatchActionBar — mounted in `DataGrid`'s `#selection-toolbar` slot for the
 * fulfillment lines grid (P3). Offers the 3 batch actions: adjust (opens
 * `BatchAdjustDialog`), bind default address (targeted to the current
 * selection), and CSV export (pure client-side, no mutation).
 *
 * Address-bind design choice: rather than the wave-wide
 * `bindDefaultAddressesForWave` (which would silently touch every
 * address-missing line in the wave, not just what's selected), this uses
 * `batchBindAddressToLines` scoped to the selected rows — each selected
 * row's `customerProfileId` is looked up (cached per profile within one
 * click) for its `isDefault` address, and only rows that resolve to one are
 * included. This keeps the action faithful to "batch action bar acts on the
 * current selection" rather than reaching outside it.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary } from '@/shared/i18n/glossary'
import { batchBindAddressToLines, listAddressesByProfile } from '@/shared/api/bridge'
import { exportRowsToCsv, type CsvColumnSpec } from '@/shared/lib/csv/exportRowsToCsv'
import type { FulfillmentGridRow } from './useFulfillmentGrid'
import BatchAdjustDialog from './BatchAdjustDialog.vue'

const props = defineProps<{
  selectedRows: FulfillmentGridRow[]
  waveId: number
}>()

const emit = defineEmits<{
  done: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()

const showAdjustDialog = ref(false)
const bindingAddress = ref(false)

function reportOutcome(summaryKey: string, successCount: number, failureCount: number): void {
  if (failureCount > 0) {
    feedback.error(t('fulfillmentGrid.batch.someFailed', { count: failureCount }))
  } else {
    feedback.success(t('fulfillmentGrid.batch.done'))
  }
  feedback.receipt({ kind: 'action', summary: t(summaryKey) })
}

function handleAdjustSuccess(payload: { successCount: number; failureCount: number }): void {
  reportOutcome('fulfillmentGrid.batch.adjust', payload.successCount, payload.failureCount)
  emit('done')
}

async function handleBindDefaultAddress(): Promise<void> {
  const rowsWithProfile = props.selectedRows.filter((row) => row.customerProfileId != null)
  const noProfileCount = props.selectedRows.length - rowsWithProfile.length
  if (rowsWithProfile.length === 0) {
    feedback.error(t('fulfillmentGrid.batch.someFailed', { count: props.selectedRows.length }))
    return
  }

  bindingAddress.value = true
  try {
    const defaultAddressByProfile = new Map<number, number | null>()
    const entries: Array<{ fulfillmentLineId: number; customerAddressId: number }> = []
    let unresolvedCount = noProfileCount

    for (const row of rowsWithProfile) {
      const profileId = row.customerProfileId as number
      if (!defaultAddressByProfile.has(profileId)) {
        const addresses = await listAddressesByProfile(profileId)
        const defaultAddress = addresses.find((address) => address.isDefault)
        defaultAddressByProfile.set(profileId, defaultAddress ? defaultAddress.id : null)
      }
      const addressId = defaultAddressByProfile.get(profileId) ?? null
      if (addressId == null) {
        unresolvedCount += 1
        continue
      }
      entries.push({ fulfillmentLineId: row.fulfillmentLineId, customerAddressId: addressId })
    }

    if (entries.length === 0) {
      feedback.error(t('fulfillmentGrid.batch.someFailed', { count: props.selectedRows.length }))
      return
    }

    const results = await batchBindAddressToLines(entries)
    const successCount = results.filter((result) => result.success).length
    const failureCount = results.length - successCount + unresolvedCount
    reportOutcome('fulfillmentGrid.batch.bindDefaultAddress', successCount, failureCount)
    emit('done')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    bindingAddress.value = false
  }
}

function handleExportCsv(): void {
  const columns: CsvColumnSpec<FulfillmentGridRow>[] = [
    { key: 'participantDisplay', header: t('fulfillmentGrid.columns.participant') },
    { key: 'productDisplay', header: t('fulfillmentGrid.columns.product') },
    { key: 'quantity', header: t('fulfillmentGrid.columns.quantity') },
    {
      key: 'lineReason',
      header: t('fulfillmentGrid.columns.source'),
      getValue: (row) => glossaryLabel('lineReason', row.lineReason),
    },
    {
      key: 'allocationState',
      header: t('fulfillmentGrid.columns.allocationState'),
      getValue: (row) => glossaryLabel('allocationState', row.allocationState),
    },
    {
      key: 'addressState',
      header: t('fulfillmentGrid.columns.addressState'),
      getValue: (row) => glossaryLabel('addressState', row.addressState),
    },
    {
      key: 'supplierState',
      header: t('fulfillmentGrid.columns.supplierState'),
      getValue: (row) => glossaryLabel('supplierState', row.supplierState),
    },
    {
      key: 'channelSyncState',
      header: t('fulfillmentGrid.columns.channelSyncState'),
      getValue: (row) => glossaryLabel('channelSyncState', row.channelSyncState),
    },
    {
      key: 'reviewRequirement',
      header: t('fulfillmentGrid.columns.reviewRequirement'),
      getValue: (row) => glossaryLabel('reviewRequirement', row.reviewRequirement),
    },
    {
      key: 'basisDriftStatus',
      header: t('fulfillmentGrid.filters.driftStatus'),
      getValue: (row) => glossaryLabel('basisDriftStatus', row.basisDriftStatus),
    },
    {
      key: 'trackingNo',
      header: t('fulfillmentGrid.columns.trackingNo'),
      getValue: (row) => row.trackingNo ?? '—',
    },
  ]
  // Pure client-side serialization — no mutation, so no `done` emit.
  exportRowsToCsv(props.selectedRows, columns, `fulfillment-lines-wave-${props.waveId}.csv`)
}
</script>

<template>
  <div class="batch-action-bar">
    <span class="batch-action-bar__count">
      {{ t('fulfillmentGrid.batch.selectedCount', { n: selectedRows.length }) }}
    </span>
    <div class="batch-action-bar__actions">
      <NButton size="small" :disabled="selectedRows.length === 0" @click="showAdjustDialog = true">
        {{ t('fulfillmentGrid.batch.adjust') }}
      </NButton>
      <NButton
        size="small"
        :loading="bindingAddress"
        :disabled="selectedRows.length === 0"
        @click="handleBindDefaultAddress"
      >
        {{ t('fulfillmentGrid.batch.bindDefaultAddress') }}
      </NButton>
      <NButton size="small" :disabled="selectedRows.length === 0" @click="handleExportCsv">
        {{ t('fulfillmentGrid.batch.exportCsv') }}
      </NButton>
    </div>

    <BatchAdjustDialog
      v-model:show="showAdjustDialog"
      :selected-rows="selectedRows"
      :wave-id="waveId"
      @success="handleAdjustSuccess"
    />
  </div>
</template>

<style scoped>
.batch-action-bar {
  display: contents;
}

.batch-action-bar__count {
  font-weight: var(--font-weight-medium);
}

.batch-action-bar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}
</style>
