<script setup lang="ts">
/**
 * RowDetailDrawer — the fulfillment grid's row inspector (plan 3.3.2/3.3.3,
 * P3 unit C). Hosts, for one `FulfillmentGridRow`:
 * - a compact "state timeline" of the row's own dimension values,
 * - the provenance chain (demand kind / source summary / generatedBy /
 *   review reason),
 * - this line's shipment records (`listShipmentsByWave`, client-filtered by
 *   `fulfillmentLineId`) and adjustment history (`listAdjustmentsByWave`,
 *   same filter),
 * - an embedded single-line adjustment form (mirrors `BatchActionBar`'s
 *   `targetKind` branching: add/reduce/remove/replace target the
 *   fulfillment line itself; compensation/reissue target the participant
 *   snapshot and are blocked with an inline warning when the row has none),
 * - the inline address editor (only for address-abnormal rows), and
 * - an on-demand cross-wave customer history drill-in
 *   (`getCustomerFulfillmentHistory`, soft-fail, only offered when
 *   `customerProfileId` is present).
 *
 * Emits `'changed'` after any mutation performed from inside the drawer
 * (an adjustment recorded, or an address bound via `InlineAddressEditor`) —
 * the caller (`WaveLinesTab.vue`) wires this straight to
 * `useFulfillmentGrid().mutationDone()`.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NAutoComplete } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback, useRelativeTime } from '@/shared/ui/feedback'
import { useGlossary } from '@/shared/i18n/glossary'
import { getCustomerFulfillmentHistory, listAdjustmentsByWave, listShipmentsByWave, recordAdjustment } from '@/shared/api/bridge'
import { useOperatorRosterStore } from '@/shared/model/operator-roster'
import type { AdjustmentKind } from '@/entities/fulfillment'
import type { dto } from '@/../wailsjs/go/models'
import type { FulfillmentGridRow } from './useFulfillmentGrid'
import InlineAddressEditor from './InlineAddressEditor.vue'

const props = defineProps<{
  row: FulfillmentGridRow | null
  show: boolean
  waveId: number
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'changed'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const relativeTime = useRelativeTime()
const { label: glossaryLabel } = useGlossary()
const operatorRoster = useOperatorRosterStore()

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}

const isAddressAbnormal = computed(
  () => props.row != null && (props.row.addressState === 'missing' || props.row.addressState === 'invalid'),
)

// ── Shipments + adjustment history (client-filtered by fulfillmentLineId) ──

const shipments = ref<dto.ShipmentDTO[]>([])
const shipmentsLoading = ref(false)
const adjustments = ref<dto.FulfillmentAdjustmentDTO[]>([])
const adjustmentsLoading = ref(false)

async function loadLineData(): Promise<void> {
  const currentRow = props.row
  if (!currentRow) {
    shipments.value = []
    adjustments.value = []
    return
  }
  shipmentsLoading.value = true
  try {
    const list = await listShipmentsByWave(props.waveId)
    shipments.value = list.filter((shipment) =>
      (shipment.lines ?? []).some((line) => line.fulfillmentLineId === currentRow.fulfillmentLineId),
    )
  } catch (err) {
    // `listShipmentsByWave` is soft-fail only for the "no Wails runtime"
    // case (returns `[]`) — a real backend RPC error still rejects, so this
    // must be caught explicitly rather than left as an unhandled rejection.
    shipments.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    shipmentsLoading.value = false
  }

  adjustmentsLoading.value = true
  try {
    const list = await listAdjustmentsByWave(props.waveId)
    adjustments.value = list.filter((adjustment) => adjustment.fulfillmentLineId === currentRow.fulfillmentLineId)
  } catch (err) {
    adjustments.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    adjustmentsLoading.value = false
  }
}

// ── Customer cross-wave history — lazy, fetched only on drill-in click ──

const customerHistory = ref<dto.CustomerFulfillmentHistoryRowDTO[]>([])
const customerHistoryLoading = ref(false)
const customerHistoryOpened = ref(false)

const canViewCustomerHistory = computed(() => props.row?.customerProfileId != null)

async function handleOpenCustomerHistory(): Promise<void> {
  const profileId = props.row?.customerProfileId
  if (!profileId) return
  customerHistoryOpened.value = true
  customerHistoryLoading.value = true
  try {
    customerHistory.value = await getCustomerFulfillmentHistory(profileId)
  } catch (err) {
    customerHistory.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    customerHistoryLoading.value = false
  }
}

// ── Embedded single-line adjustment form (mirrors BatchActionBar's kind → targetKind branching) ──

const ADJUSTMENT_KINDS: AdjustmentKind[] = ['add', 'reduce', 'compensation', 'remove', 'replace', 'reissue']
const PARTICIPANT_TARGET_KINDS = new Set<AdjustmentKind>(['compensation', 'reissue'])
const REASON_CODE_PRESETS = [
  'out_of_stock_reissue',
  'address_error',
  'customer_request',
  'quantity_correction',
  'product_replacement',
  'quality_issue',
  'other',
] as const

const adjustKind = ref<AdjustmentKind>('add')
const quantityDelta = ref<number | null>(0)
const fromProductId = ref<number | null>(null)
const toProductId = ref<number | null>(null)
const reasonCode = ref<string | null>(null)
const operatorId = ref('')
const note = ref('')
const evidenceRef = ref('')
const submittingAdjust = ref(false)

function resetAdjustForm(): void {
  adjustKind.value = 'add'
  quantityDelta.value = 0
  fromProductId.value = null
  toProductId.value = null
  reasonCode.value = null
  note.value = ''
  evidenceRef.value = ''
  // operatorId is intentionally NOT reset — the operator running a batch of
  // manual adjustments across several rows in one sitting is almost always
  // the same person.
}

const requiresParticipantTarget = computed(() => PARTICIPANT_TARGET_KINDS.has(adjustKind.value))
const participantMissing = computed(() => requiresParticipantTarget.value && !props.row?.waveParticipantSnapshotId)
const requiresReplaceProducts = computed(() => adjustKind.value === 'replace')

const adjustmentKindOptions = computed<SelectOption[]>(() =>
  ADJUSTMENT_KINDS.map((kind) => ({ label: glossaryLabel('adjustmentKind', kind), value: kind })),
)

const reasonCodeOptions = computed<SelectOption[]>(() =>
  REASON_CODE_PRESETS.map((code) => ({ label: t(`fulfillmentGrid.reasonCode.presets.${code}`), value: code })),
)

const recentOperatorIds = computed(() => operatorRoster.list())

const canSubmitAdjust = computed(() => {
  if (!props.row || submittingAdjust.value) return false
  if (participantMissing.value) return false
  if (requiresReplaceProducts.value && (!fromProductId.value || !toProductId.value)) return false
  if (!reasonCode.value || reasonCode.value.trim().length === 0) return false
  if (operatorId.value.trim().length === 0) return false
  return true
})

async function handleSubmitAdjust(): Promise<void> {
  const currentRow = props.row
  if (!currentRow || !canSubmitAdjust.value) return
  submittingAdjust.value = true
  try {
    const targetsParticipant = requiresParticipantTarget.value
    await recordAdjustment({
      waveId: props.waveId,
      targetKind: targetsParticipant ? 'participant' : 'fulfillment_line',
      fulfillmentLineId: targetsParticipant ? null : currentRow.fulfillmentLineId,
      waveParticipantSnapshotId: targetsParticipant ? currentRow.waveParticipantSnapshotId ?? null : null,
      adjustmentKind: adjustKind.value,
      quantityDelta: quantityDelta.value ?? 0,
      fromProductId: requiresReplaceProducts.value ? fromProductId.value : null,
      toProductId: requiresReplaceProducts.value ? toProductId.value : null,
      reasonCode: reasonCode.value!.trim(),
      operatorId: operatorId.value.trim(),
      note: note.value,
      evidenceRef: evidenceRef.value,
    })
    operatorRoster.add(operatorId.value.trim())
    feedback.success(t('feedback.success'))
    resetAdjustForm()
    emit('changed')
    void loadLineData()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submittingAdjust.value = false
  }
}

function handleAddressBound(): void {
  emit('changed')
}

// Refetch this line's data whenever the drawer opens, or a different row is
// clicked while it is already open (Assembly may swap `row` without an
// intervening close). Also resets the adjustment form and the lazily-loaded
// customer history so a stale row's data never leaks into a new one.
watch(
  () => [props.show, props.row?.fulfillmentLineId] as const,
  ([show]) => {
    if (!show) return
    customerHistoryOpened.value = false
    customerHistory.value = []
    resetAdjustForm()
    void loadLineData()
  },
  { immediate: true },
)
</script>

<template>
  <DetailDrawer :show="show" size="lg" :title="t('fulfillmentGrid.detail.title')" @update:show="handleUpdateShow">
    <template v-if="row">
      <p class="row-detail-drawer__subtitle">{{ row.participantDisplay }} · {{ row.productDisplay }} × {{ row.quantity }}</p>

      <SectionCard flat :title="t('fulfillmentGrid.detail.timeline')">
        <div class="row-detail-drawer__badges">
          <StatusBadge dimension="lineReason" :value="row.lineReason" size="sm" />
          <StatusBadge dimension="allocationState" :value="row.allocationState" size="sm" />
          <StatusBadge dimension="addressState" :value="row.addressState" size="sm" />
          <StatusBadge dimension="supplierState" :value="row.supplierState" size="sm" />
          <StatusBadge dimension="channelSyncState" :value="row.channelSyncState" size="sm" />
          <StatusBadge dimension="reviewRequirement" :value="row.reviewRequirement" size="sm" />
          <StatusBadge dimension="basisDriftStatus" :value="row.basisDriftStatus" size="sm" />
        </div>
      </SectionCard>

      <SectionCard flat :title="t('fulfillmentGrid.detail.provenance')">
        <div class="row-detail-drawer__provenance">
          <StatusBadge dimension="demandKind" :value="row.demandKind" size="sm" />
          <p v-if="row.demandSourceSummary" class="row-detail-drawer__provenance-text">{{ row.demandSourceSummary }}</p>
          <p v-if="row.generatedBy" class="row-detail-drawer__provenance-text">{{ row.generatedBy }}</p>
          <p v-if="row.reviewReasonSummary" class="row-detail-drawer__provenance-text">{{ row.reviewReasonSummary }}</p>
        </div>
      </SectionCard>

      <SectionCard flat :title="t('fulfillmentGrid.detail.shipments')">
        <EmptyState
          v-if="!shipmentsLoading && shipments.length === 0"
          size="sm"
          :title="t('fulfillmentGrid.detail.noShipments')"
        />
        <ul v-else class="row-detail-drawer__list">
          <li v-for="shipment in shipments" :key="shipment.id" class="row-detail-drawer__list-item">
            <span class="row-detail-drawer__list-primary">{{ shipment.trackingNo || '—' }}</span>
            <StatusBadge dimension="shipmentStatus" :value="shipment.status" size="sm" />
            <span class="row-detail-drawer__list-meta">{{ relativeTime.format(new Date(shipment.createdAt).getTime()) }}</span>
          </li>
        </ul>
      </SectionCard>

      <SectionCard flat :title="t('fulfillmentGrid.detail.adjustments')">
        <EmptyState
          v-if="!adjustmentsLoading && adjustments.length === 0"
          size="sm"
          :title="t('fulfillmentGrid.detail.noAdjustments')"
        />
        <ul v-else class="row-detail-drawer__list">
          <li v-for="adjustment in adjustments" :key="adjustment.id" class="row-detail-drawer__list-item">
            <StatusBadge dimension="adjustmentKind" :value="adjustment.adjustmentKind" size="sm" />
            <span class="row-detail-drawer__list-primary">{{ adjustment.quantityDelta }}</span>
            <span class="row-detail-drawer__list-meta">{{ relativeTime.format(new Date(adjustment.createdAt).getTime()) }}</span>
          </li>
        </ul>

        <NForm label-placement="top" class="row-detail-drawer__adjust-form">
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.kind')">
            <NSelect v-model:value="adjustKind" :options="adjustmentKindOptions" :disabled="submittingAdjust" />
          </NFormItem>
          <p v-if="participantMissing" class="row-detail-drawer__warning">
            {{ t('fulfillmentGrid.adjustDialog.reissueNeedsParticipant') }}
          </p>
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.quantityDelta')">
            <NInputNumber v-model:value="quantityDelta" style="width: 100%" :disabled="submittingAdjust" />
          </NFormItem>
          <template v-if="requiresReplaceProducts">
            <NFormItem :label="t('fulfillmentGrid.adjustDialog.fromProduct')">
              <NInputNumber v-model:value="fromProductId" style="width: 100%" :disabled="submittingAdjust" />
            </NFormItem>
            <NFormItem :label="t('fulfillmentGrid.adjustDialog.toProduct')">
              <NInputNumber v-model:value="toProductId" style="width: 100%" :disabled="submittingAdjust" />
            </NFormItem>
          </template>
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.reasonCode')">
            <NSelect
              v-model:value="reasonCode"
              :options="reasonCodeOptions"
              filterable
              tag
              clearable
              :disabled="submittingAdjust"
            />
          </NFormItem>
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.operatorId')">
            <NAutoComplete v-model:value="operatorId" :options="recentOperatorIds" :disabled="submittingAdjust" />
          </NFormItem>
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.note')">
            <NInput v-model:value="note" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :disabled="submittingAdjust" />
          </NFormItem>
          <NFormItem :label="t('fulfillmentGrid.adjustDialog.evidenceRef')">
            <NInput v-model:value="evidenceRef" :disabled="submittingAdjust" />
          </NFormItem>
          <div class="row-detail-drawer__adjust-actions">
            <NButton type="primary" :loading="submittingAdjust" :disabled="!canSubmitAdjust" @click="handleSubmitAdjust">
              {{ t('fulfillmentGrid.adjustDialog.submit') }}
            </NButton>
          </div>
        </NForm>
      </SectionCard>

      <SectionCard v-if="isAddressAbnormal" flat :title="t('fulfillmentGrid.address.sectionTitle')">
        <InlineAddressEditor :row="row!" @bound="handleAddressBound" />
      </SectionCard>

      <SectionCard flat :title="t('fulfillmentGrid.detail.customerHistory')">
        <NButton v-if="!customerHistoryOpened" :disabled="!canViewCustomerHistory" @click="handleOpenCustomerHistory">
          {{ t('fulfillmentGrid.detail.openCustomerHistory') }}
        </NButton>
        <ul v-else-if="customerHistory.length > 0" class="row-detail-drawer__list">
          <li v-for="historyRow in customerHistory" :key="historyRow.fulfillmentLineId" class="row-detail-drawer__list-item">
            <span class="row-detail-drawer__list-primary">
              {{ historyRow.waveNo }} · {{ historyRow.productName }} × {{ historyRow.quantity }}
            </span>
            <StatusBadge dimension="supplierState" :value="historyRow.supplierState" size="sm" />
            <StatusBadge v-if="historyRow.shipmentStatus" dimension="shipmentStatus" :value="historyRow.shipmentStatus" size="sm" />
            <span class="row-detail-drawer__list-meta">
              {{ relativeTime.format(new Date(historyRow.createdAt).getTime()) }}
            </span>
          </li>
        </ul>
      </SectionCard>
    </template>
  </DetailDrawer>
</template>

<style scoped>
.row-detail-drawer__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.row-detail-drawer__badges {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.row-detail-drawer__provenance {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
}

.row-detail-drawer__provenance-text {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.row-detail-drawer__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0 0 var(--space-3);
  padding: 0;
  list-style: none;
}

.row-detail-drawer__list-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--card-bg);
}

.row-detail-drawer__list-primary {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.row-detail-drawer__list-meta {
  margin-left: auto;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.row-detail-drawer__adjust-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.row-detail-drawer__warning {
  margin: calc(var(--space-2) * -1) 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.row-detail-drawer__adjust-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
