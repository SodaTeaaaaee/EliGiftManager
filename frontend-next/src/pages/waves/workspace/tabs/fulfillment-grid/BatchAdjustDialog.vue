<script setup lang="ts">
/**
 * BatchAdjustDialog — the batch adjustment form for `BatchActionBar`'s
 * "batch adjust" action (P3 fulfillment grid). Builds one
 * `RecordAdjustmentInput` per eligible selected row and submits them all via
 * `batchRecordAdjustments` (partial-success — every entry gets its own
 * savepoint server-side).
 *
 * targetKind branching (CANON — adjustment_usecase.go:50-110):
 * - add / reduce / remove / replace -> targetKind='fulfillment_line',
 *   keyed by each row's `fulfillmentLineId`. `replace` additionally requires
 *   a non-zero `fromProductId` (defaults to the row's own `productId`,
 *   overridable via a single shared input for the "same source product
 *   across the whole batch" case) and `toProductId` (operator-entered,
 *   shared across all entries).
 * - compensation / reissue -> targetKind='participant', keyed by each row's
 *   `waveParticipantSnapshotId`. Rows with no snapshot id (or, for replace,
 *   no resolvable source product) are silently excluded from the submitted
 *   entries and surfaced via a hint + folded into the reported failure
 *   count — never crash the batch over one bad row.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NSelect, NInputNumber, NInput, NAutoComplete, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary } from '@/shared/i18n/glossary'
import { batchRecordAdjustments } from '@/shared/api/bridge'
import { useOperatorRosterStore } from '@/shared/model/operator-roster'
import type { AdjustmentKind } from '@/entities/fulfillment'
import type { FulfillmentGridRow } from './useFulfillmentGrid'

const props = defineProps<{
  show: boolean
  selectedRows: FulfillmentGridRow[]
  waveId: number
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  /** Fires after a (partially) successful `batchRecordAdjustments` call — `failureCount` folds in both server-reported failures and client-side skipped rows. */
  success: [{ successCount: number; failureCount: number }]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()
const operatorRoster = useOperatorRosterStore()

const REASON_CODE_PRESETS = [
  'out_of_stock_reissue',
  'address_error',
  'customer_request',
  'quantity_correction',
  'product_replacement',
  'quality_issue',
  'other',
] as const

const ADJUSTMENT_KINDS: AdjustmentKind[] = ['add', 'reduce', 'compensation', 'remove', 'replace', 'reissue']

const adjustmentKind = ref<AdjustmentKind>('add')
const quantityDelta = ref<number | null>(0)
const fromProductIdOverride = ref<number | null>(null)
const toProductId = ref<number | null>(null)
const reasonCode = ref<string | null>(null)
const operatorId = ref('')
const note = ref('')
const evidenceRef = ref('')
const submitting = ref(false)

function resetForm(): void {
  adjustmentKind.value = 'add'
  quantityDelta.value = 0
  fromProductIdOverride.value = null
  toProductId.value = null
  reasonCode.value = null
  operatorId.value = ''
  note.value = ''
  evidenceRef.value = ''
}

// This dialog stays mounted (no `v-if` at the call site) — reset on every open.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const adjustmentKindOptions = computed<SelectOption[]>(() =>
  ADJUSTMENT_KINDS.map((kind) => ({ label: glossaryLabel('adjustmentKind', kind), value: kind })),
)

const reasonCodeOptions = computed<SelectOption[]>(() =>
  REASON_CODE_PRESETS.map((code) => ({ label: t(`fulfillmentGrid.reasonCode.presets.${code}`), value: code })),
)

const isParticipantTarget = computed(() => adjustmentKind.value === 'compensation' || adjustmentKind.value === 'reissue')
const isReplace = computed(() => adjustmentKind.value === 'replace')

/** The row's effective `fromProductId` for a `replace` — the shared override wins, else the row's own current product. */
function resolvedFromProductId(row: FulfillmentGridRow): number | null {
  return fromProductIdOverride.value ?? row.productId ?? null
}

/** Rows that can actually receive this adjustment kind — server-required identifiers must be resolvable. */
const eligibleRows = computed<FulfillmentGridRow[]>(() => {
  if (isParticipantTarget.value) {
    return props.selectedRows.filter((row) => row.waveParticipantSnapshotId != null)
  }
  if (isReplace.value) {
    return props.selectedRows.filter((row) => resolvedFromProductId(row) != null)
  }
  return props.selectedRows
})

const skippedCount = computed(() => props.selectedRows.length - eligibleRows.value.length)

const canSubmit = computed(
  () =>
    !submitting.value &&
    eligibleRows.value.length > 0 &&
    quantityDelta.value != null &&
    (reasonCode.value ?? '').trim().length > 0 &&
    operatorId.value.trim().length > 0 &&
    (!isReplace.value || (toProductId.value != null && toProductId.value > 0)),
)

const operatorIdOptions = computed(() => operatorRoster.list())

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    const kind = adjustmentKind.value
    const targetKind = isParticipantTarget.value ? 'participant' : 'fulfillment_line'
    const trimmedReasonCode = (reasonCode.value ?? '').trim()
    const trimmedOperatorId = operatorId.value.trim()
    const entries = eligibleRows.value.map((row) => ({
      waveId: props.waveId,
      targetKind,
      fulfillmentLineId: isParticipantTarget.value ? null : row.fulfillmentLineId,
      waveParticipantSnapshotId: isParticipantTarget.value ? (row.waveParticipantSnapshotId ?? null) : null,
      adjustmentKind: kind,
      quantityDelta: quantityDelta.value ?? 0,
      reasonCode: trimmedReasonCode,
      operatorId: trimmedOperatorId,
      note: note.value,
      evidenceRef: evidenceRef.value,
      fromProductId: isReplace.value ? resolvedFromProductId(row) : null,
      toProductId: isReplace.value ? toProductId.value : null,
    }))
    const result = await batchRecordAdjustments({ entries })
    operatorRoster.add(trimmedOperatorId)
    emit('success', { successCount: result.successCount, failureCount: result.failureCount + skippedCount.value })
    close()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('fulfillmentGrid.adjustDialog.title')"
    :style="{ width: 'min(560px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('fulfillmentGrid.adjustDialog.kind')">
        <NSelect v-model:value="adjustmentKind" :options="adjustmentKindOptions" :disabled="submitting" />
      </NFormItem>

      <p v-if="isParticipantTarget && skippedCount > 0" class="batch-adjust-dialog__hint">
        {{ t('fulfillmentGrid.adjustDialog.reissueNeedsParticipant') }}
      </p>
      <p v-else-if="isReplace && skippedCount > 0" class="batch-adjust-dialog__hint">
        {{ t('fulfillmentGrid.adjustDialog.replaceNeedsSourceProduct') }}
      </p>

      <NFormItem :label="t('fulfillmentGrid.adjustDialog.quantityDelta')">
        <NInputNumber v-model:value="quantityDelta" :precision="0" :disabled="submitting" style="width: 100%" />
      </NFormItem>

      <template v-if="isReplace">
        <NFormItem :label="t('fulfillmentGrid.adjustDialog.fromProduct')">
          <NInputNumber
            v-model:value="fromProductIdOverride"
            :min="1"
            :precision="0"
            :disabled="submitting"
            :placeholder="t('fulfillmentGrid.adjustDialog.fromProduct')"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem :label="t('fulfillmentGrid.adjustDialog.toProduct')">
          <NInputNumber v-model:value="toProductId" :min="1" :precision="0" :disabled="submitting" style="width: 100%" />
        </NFormItem>
      </template>

      <NFormItem :label="t('fulfillmentGrid.adjustDialog.reasonCode')">
        <NSelect v-model:value="reasonCode" filterable tag :options="reasonCodeOptions" :disabled="submitting" />
      </NFormItem>

      <NFormItem :label="t('fulfillmentGrid.adjustDialog.operatorId')">
        <NAutoComplete v-model:value="operatorId" :options="operatorIdOptions" :disabled="submitting" />
      </NFormItem>

      <NFormItem :label="t('fulfillmentGrid.adjustDialog.note')">
        <NInput v-model:value="note" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" :disabled="submitting" />
      </NFormItem>

      <NFormItem :label="t('fulfillmentGrid.adjustDialog.evidenceRef')">
        <NInput v-model:value="evidenceRef" :disabled="submitting" />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="batch-adjust-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('fulfillmentGrid.adjustDialog.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('fulfillmentGrid.adjustDialog.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.batch-adjust-dialog__hint {
  margin: calc(var(--space-2) * -1) 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.batch-adjust-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
