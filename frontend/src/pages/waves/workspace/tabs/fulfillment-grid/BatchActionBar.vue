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
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary } from '@/shared/i18n/glossary'
import { batchBindAddressToLines, listAddressesByProfile } from '@/shared/api/bridge'
import { exportRowsToCsv, type CsvColumnSpec } from '@/shared/lib/csv/exportRowsToCsv'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'
import { customerResolutionWriteAccess } from '@/shared/lib/customer-resolution'
import type { FulfillmentGridRow } from './useFulfillmentGrid'
import BatchAdjustDialog from './BatchAdjustDialog.vue'
import { bindDefaultAddressesForRows } from './addressWriteFlow'

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
const featurePolicy = useCustomerResolutionFeaturePolicy()
const addressWritesEnabled = computed(
  () => customerResolutionWriteAccess(featurePolicy.policy.value).canManageAddresses,
)
void featurePolicy.load()

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
  if (!addressWritesEnabled.value || bindingAddress.value || props.selectedRows.length === 0) return

  bindingAddress.value = true
  try {
    const outcome = await bindDefaultAddressesForRows(
      addressWritesEnabled.value,
      props.selectedRows,
      { listAddressesByProfile, batchBindAddressToLines },
    )
    if (!outcome) return
    if (!outcome.attempted) {
      feedback.error(t('fulfillmentGrid.batch.someFailed', { count: outcome.failureCount }))
      return
    }
    reportOutcome('fulfillmentGrid.batch.bindDefaultAddress', outcome.successCount, outcome.failureCount)
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
    <span v-if="!addressWritesEnabled" class="batch-action-bar__disabled-reason">
      {{ t('fulfillmentGrid.address.writesDisabledReason') }}
    </span>
    <div class="batch-action-bar__actions">
      <NButton size="small" :disabled="selectedRows.length === 0" @click="showAdjustDialog = true">
        {{ t('fulfillmentGrid.batch.adjust') }}
      </NButton>
      <NButton
        data-testid="batch-bind-default-address"
        size="small"
        :loading="bindingAddress"
        :disabled="selectedRows.length === 0 || !addressWritesEnabled"
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

.batch-action-bar__disabled-reason {
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.batch-action-bar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}
</style>
