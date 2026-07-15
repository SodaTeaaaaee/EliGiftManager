<script setup lang="ts">
/**
 * RecordAcceptanceDialog — `SupplierOrderCard.vue`'s "record acceptance"
 * form (P5 factory-orders sub-area). Only ever opened for a `submitted`
 * order (the card gates the trigger button).
 *
 * `recordSupplierOrderAcceptance` (supplier_order_lifecycle_usecase.go:
 * 79-132) has NO partial/incremental acceptance — the whole order flips
 * `submitted -> accepted` in one call, and a second call always fails
 * because `order.Status` is no longer `submitted`. So this dialog lists
 * EVERY line of the order (not a selectable subset) and pre-populates each
 * row's input with that line's `submittedQuantity` — the operator adjusts
 * down/up per line, never omits one.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NInputNumber, NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { recordSupplierOrderAcceptance } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  show: boolean
  order: dto.SupplierOrderDTO
  lines: dto.SupplierOrderLineDTO[]
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  done: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const acceptedQuantityByLineId = ref<Map<number, number | null>>(new Map())
const submitting = ref(false)

function resetForm(): void {
  const next = new Map<number, number | null>()
  for (const line of props.lines) next.set(line.id, line.submittedQuantity)
  acceptedQuantityByLineId.value = next
}

// This dialog stays mounted across opens for the same card — reset on every open.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

function setLineQuantity(lineId: number, value: number | null): void {
  const next = new Map(acceptedQuantityByLineId.value)
  next.set(lineId, value)
  acceptedQuantityByLineId.value = next
}

const canSubmit = computed(
  () =>
    !submitting.value &&
    props.lines.length > 0 &&
    props.lines.every((line) => {
      const value = acceptedQuantityByLineId.value.get(line.id)
      return value != null && value >= 0
    }),
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    const entries = props.lines.map((line) => ({
      lineId: line.id,
      acceptedQuantity: acceptedQuantityByLineId.value.get(line.id) ?? 0,
    }))
    await recordSupplierOrderAcceptance({ orderId: props.order.id, lines: entries })
    feedback.success(t('waveWorkspace.factory.recordAcceptance.success'))
    close()
    emit('done')
  } catch (err) {
    feedback.error(t('waveWorkspace.factory.recordAcceptance.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waveWorkspace.factory.recordAcceptance.dialogTitle')"
    :style="{ width: 'min(520px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <p class="record-acceptance-dialog__hint">{{ t('waveWorkspace.factory.recordAcceptance.hint') }}</p>

    <ul class="record-acceptance-dialog__lines">
      <li v-for="line in lines" :key="line.id" class="record-acceptance-dialog__line">
        <span class="record-acceptance-dialog__line-label">
          {{ t('waveWorkspace.factory.recordAcceptance.lineLabel', { lineNo: line.supplierLineNo ?? '—', sku: line.supplierSku }) }}
        </span>
        <NInputNumber
          :value="acceptedQuantityByLineId.get(line.id) ?? null"
          :min="0"
          :precision="0"
          :disabled="submitting"
          :placeholder="t('waveWorkspace.factory.recordAcceptance.acceptedQuantity')"
          style="width: 140px"
          @update:value="(value: number | null) => setLineQuantity(line.id, value)"
        />
      </li>
    </ul>

    <template #footer>
      <div class="record-acceptance-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('waveWorkspace.factory.recordAcceptance.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('waveWorkspace.factory.recordAcceptance.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.record-acceptance-dialog__hint {
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.record-acceptance-dialog__lines {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 360px;
  margin: 0 0 var(--space-2);
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.record-acceptance-dialog__line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-1) 0;
}

.record-acceptance-dialog__line-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.record-acceptance-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
