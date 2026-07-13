<script setup lang="ts">
/**
 * CorrectShipmentDialog — `ShipmentHistory.vue`'s "correct" form (P5
 * shipment-backfill sub-area, plan §5.2's `UpdateShipment` pairing).
 * `UpdateShipment` is a compensating write OUTSIDE the undo/redo command
 * history by design (`shipment_lifecycle_usecase.go:13-18`) — it rejects
 * already-voided shipments server-side (`ShipmentHistory.vue` also disables
 * the trigger button for voided rows, so a rejected submit here can only
 * happen via a stale row).
 */
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NInput, NDatePicker, NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { updateShipment } from '@/shared/api/bridge'
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

const supplierPlatform = ref('')
const shipmentNo = ref('')
const externalShipmentNo = ref('')
const carrierCode = ref('')
const carrierName = ref('')
const trackingNo = ref('')
const shippedAtMs = ref<number | null>(null)
const submitting = ref(false)

function resetForm(): void {
  const shipment = props.shipment
  supplierPlatform.value = shipment?.supplierPlatform ?? ''
  shipmentNo.value = shipment?.shipmentNo ?? ''
  externalShipmentNo.value = shipment?.externalShipmentNo ?? ''
  carrierCode.value = shipment?.carrierCode ?? ''
  carrierName.value = shipment?.carrierName ?? ''
  trackingNo.value = shipment?.trackingNo ?? ''
  shippedAtMs.value = shipment?.shippedAt ? new Date(shipment.shippedAt).getTime() : null
}

watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!props.shipment || submitting.value) return
  submitting.value = true
  try {
    await updateShipment({
      id: props.shipment.id,
      supplierPlatform: supplierPlatform.value.trim(),
      shipmentNo: shipmentNo.value.trim(),
      externalShipmentNo: externalShipmentNo.value.trim(),
      carrierCode: carrierCode.value.trim(),
      carrierName: carrierName.value.trim(),
      trackingNo: trackingNo.value.trim(),
      shippedAt: shippedAtMs.value != null ? new Date(shippedAtMs.value).toISOString() : undefined,
    })
    feedback.success(t('waveWorkspace.shipments.history.correctDialog.success'))
    close()
    emit('done')
  } catch (err) {
    feedback.error(t('waveWorkspace.shipments.history.correctDialog.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waveWorkspace.shipments.history.correctDialog.title')"
    :style="{ width: 'min(480px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('waveWorkspace.shipments.history.columns.supplierPlatform')">
        <NInput v-model:value="supplierPlatform" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.history.columns.shipmentNo')">
        <NInput v-model:value="shipmentNo" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.history.columns.externalShipmentNo')">
        <NInput v-model:value="externalShipmentNo" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.manual.carrierCode')">
        <NInput v-model:value="carrierCode" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.manual.carrierName')">
        <NInput v-model:value="carrierName" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.history.columns.trackingNo')">
        <NInput v-model:value="trackingNo" :disabled="submitting" />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.shipments.history.columns.shippedAt')">
        <NDatePicker v-model:value="shippedAtMs" type="datetime" clearable style="width: 100%" :disabled="submitting" />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="correct-shipment-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ t('waveWorkspace.shipments.history.correctDialog.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.correct-shipment-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
