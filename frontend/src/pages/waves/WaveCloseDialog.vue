<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NRadio,
  NRadioGroup,
  NSpace,
} from 'naive-ui'
import { closeWave } from '@/shared/api/bridge'
import { useFeedback } from '@/shared/ui/feedback'
import type { Wave } from '@/entities/models'

/**
 * The single wave-close dialog shared by the wave list and the workspace
 * shell: close-result choice (clean / residual) plus note, submitted through
 * CloseWave on the bridge. Parents reload their own data on `closed`.
 */
const props = defineProps<{
  show: boolean
  wave: Wave | null
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  closed: []
}>()

const { t } = useI18n()
const feedback = useFeedback()

const actionLoading = ref(false)
const closeForm = ref({ result: 'clean', note: '' })

watch(
  () => props.show,
  (show) => {
    if (show) closeForm.value = { result: 'clean', note: '' }
  },
)

async function handleClose() {
  if (!props.wave) return
  actionLoading.value = true
  try {
    await closeWave(props.wave.ID, closeForm.value.result, closeForm.value.note)
    feedback.success(t('waves.closeSuccess'))
    emit('update:show', false)
    emit('closed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    actionLoading.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('waves.close')"
    style="width: 480px"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm>
      <NFormItem :label="t('waves.status')">
        <NRadioGroup v-model:value="closeForm.result">
          <NSpace>
            <NRadio value="clean">{{ t('waves.cleanClose') }}</NRadio>
            <NRadio value="residual">{{ t('waves.residualClose') }}</NRadio>
          </NSpace>
        </NRadioGroup>
      </NFormItem>
      <NFormItem :label="t('waves.closeNote')">
        <NInput
          v-model:value="closeForm.note"
          type="textarea"
          :placeholder="t('waves.closeNote')"
        />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="emit('update:show', false)">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="actionLoading" @click="handleClose">
          {{ t('common.confirm') }}
        </NButton>
      </NSpace>
    </template>
  </NModal>
</template>
