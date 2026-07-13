<script setup lang="ts">
/**
 * MarkSubmittedDialog — `SupplierOrderCard.vue`'s "mark submitted" form (P5
 * factory-orders sub-area). Only ever opened for a `draft` order (the card
 * gates the trigger button); `markSupplierOrderSubmitted` itself hard-fails
 * server-side if the order is no longer `draft`
 * (`supplier_order_lifecycle_usecase.go:44-77`) — that raw error surfaces
 * via `useFeedback().error()`, never swallowed.
 *
 * `externalOrderNo` is required (server-validated); `submittedAt` is
 * optional and defaults server-side to `time.Now()` when omitted — the
 * `NDatePicker` is `type="datetime"` and clearable, submitting `undefined`
 * when left empty rather than a synthesized client-side timestamp.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NInput, NDatePicker, NButton } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { markSupplierOrderSubmitted } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  show: boolean
  order: dto.SupplierOrderDTO
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  done: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const externalOrderNo = ref('')
const submittedAtMs = ref<number | null>(null)
const submitting = ref(false)

function resetForm(): void {
  externalOrderNo.value = ''
  submittedAtMs.value = null
}

// This dialog stays mounted across opens for the same card — reset on every open.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const canSubmit = computed(() => !submitting.value && externalOrderNo.value.trim().length > 0)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    await markSupplierOrderSubmitted({
      orderId: props.order.id,
      externalOrderNo: externalOrderNo.value.trim(),
      submittedAt: submittedAtMs.value != null ? new Date(submittedAtMs.value).toISOString() : undefined,
    })
    feedback.success(t('waveWorkspace.factory.markSubmitted.success'))
    close()
    emit('done')
  } catch (err) {
    feedback.error(t('waveWorkspace.factory.markSubmitted.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waveWorkspace.factory.markSubmitted.dialogTitle')"
    :style="{ width: 'min(420px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('waveWorkspace.factory.markSubmitted.externalOrderNo')">
        <NInput
          v-model:value="externalOrderNo"
          :placeholder="t('waveWorkspace.factory.markSubmitted.externalOrderNoPlaceholder')"
          :disabled="submitting"
        />
      </NFormItem>
      <NFormItem :label="t('waveWorkspace.factory.markSubmitted.submittedAt')">
        <NDatePicker
          v-model:value="submittedAtMs"
          type="datetime"
          clearable
          style="width: 100%"
          :disabled="submitting"
        />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="mark-submitted-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('waveWorkspace.factory.markSubmitted.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('waveWorkspace.factory.markSubmitted.submit') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.mark-submitted-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
