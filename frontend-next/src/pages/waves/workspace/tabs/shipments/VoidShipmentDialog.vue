<script setup lang="ts">
/**
 * VoidShipmentDialog — `ShipmentHistory.vue`'s "void" confirmation (P5
 * shipment-backfill sub-area, plan §5.2's `VoidShipment` pairing). Collects
 * `note`/`operatorId` (stashed server-side in the shipment's `ExtraData`
 * JSON blob — no dedicated audit column yet, see
 * `dto.VoidShipmentInput`'s own doc comment) — a plain confirm-only dialog
 * isn't enough since those two fields are mandatory wire-contract input.
 * `VoidShipment` is idempotent server-side and outside the undo/redo
 * command history by design, same as `UpdateShipment` — surfaced honestly
 * via `history.voidConfirmContent` / `history.outsideUndoNotice` rather than
 * offering a misleading "undo" affordance.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NInput, NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { voidShipment } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  show: boolean
  shipment: dto.ShipmentDTO | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  done: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const note = ref('')
const operatorId = ref('')
const submitting = ref(false)

function resetForm(): void {
  note.value = ''
  operatorId.value = ''
}

watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const canSubmit = computed(() => !submitting.value && note.value.trim().length > 0 && operatorId.value.trim().length > 0)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!props.shipment || !canSubmit.value) return
  submitting.value = true
  try {
    await voidShipment({
      id: props.shipment.id,
      note: note.value.trim(),
      operatorId: operatorId.value.trim(),
    })
    feedback.success(t('waveWorkspace.shipments.history.actions.voidSuccess'))
    close()
    emit('done')
  } catch (err) {
    feedback.error(t('waveWorkspace.shipments.history.actions.voidError'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waveWorkspace.shipments.history.actions.voidConfirmTitle')"
    :style="{ width: 'min(420px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <p class="void-shipment-dialog__content">{{ t('waveWorkspace.shipments.history.actions.voidConfirmContent') }}</p>

    <NForm label-placement="top">
      <NFormItem :label="t('waveWorkspace.shipments.history.actions.voidReason')">
        <NInput v-model:value="note" type="textarea" :rows="2" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.history.actions.voidOperator')">
        <NInput v-model:value="operatorId" :disabled="submitting" />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="void-shipment-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton type="error" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('waveWorkspace.shipments.history.actions.voidSubmit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.void-shipment-dialog__content {
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.void-shipment-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
