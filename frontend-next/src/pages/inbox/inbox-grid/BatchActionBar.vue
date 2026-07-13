<script setup lang="ts">
/**
 * BatchActionBar (inbox) — mounted in `DataGrid`'s `#selection-toolbar` slot
 * for the demand-inbox grid (plan P4). Offers the single batch action:
 * assign the current selection to a wave (`batchAssignDemandToWave`),
 * surfacing per-item partial-success honestly (never a blanket
 * success/failure).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { batchAssignDemandToWave, listWavesFiltered } from '@/shared/api/bridge'
import type { DemandInboxRow } from '@/entities/demand'

const props = defineProps<{
  selectedRows: DemandInboxRow[]
}>()

const emit = defineEmits<{
  done: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const showPicker = ref(false)
const waveOptions = ref<SelectOption[]>([])
const wavesLoading = ref(false)
const targetWaveId = ref<number | null>(null)
const assigning = ref(false)

async function openPicker(): Promise<void> {
  showPicker.value = true
  targetWaveId.value = null
  wavesLoading.value = true
  try {
    const page = await listWavesFiltered({ page: 1, pageSize: 200, sortBy: 'updatedAt', sortDesc: true })
    waveOptions.value = page.items.map((wave) => ({ label: `${wave.name} (${wave.waveNo})`, value: wave.id }))
  } catch (err) {
    waveOptions.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    wavesLoading.value = false
  }
}

const canConfirm = computed(() => targetWaveId.value != null && props.selectedRows.length > 0 && !assigning.value)

async function handleConfirm(): Promise<void> {
  if (!canConfirm.value || targetWaveId.value == null) return
  assigning.value = true
  try {
    const result = await batchAssignDemandToWave({
      waveId: targetWaveId.value,
      docIds: props.selectedRows.map((row) => row.demandDocumentId),
    })
    if (result.failureCount > 0) {
      feedback.error(t('fulfillmentGrid.batch.someFailed', { count: result.failureCount }))
    } else {
      feedback.success(t('feedback.success'))
    }
    feedback.receipt({ kind: 'action', summary: t('inbox.batch.assignToWave') })
    showPicker.value = false
    emit('done')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    assigning.value = false
  }
}
</script>

<template>
  <div class="inbox-batch-action-bar">
    <span class="inbox-batch-action-bar__count">
      {{ t('inbox.batch.selected', { n: selectedRows.length }) }}
    </span>
    <div class="inbox-batch-action-bar__actions">
      <NButton size="small" type="primary" :disabled="selectedRows.length === 0" @click="openPicker">
        {{ t('inbox.batch.assignToWave') }}
      </NButton>
    </div>

    <NModal v-model:show="showPicker" preset="card" :title="t('inbox.batch.chooseWave')" style="width: 420px">
      <NSelect
        v-model:value="targetWaveId"
        :options="waveOptions"
        :loading="wavesLoading"
        filterable
        :placeholder="t('inbox.batch.chooseWave')"
      />
      <template #footer>
        <div class="inbox-batch-action-bar__modal-footer">
          <NButton @click="showPicker = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="assigning" :disabled="!canConfirm" @click="handleConfirm">
            {{ t('inbox.batch.confirm') }}
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.inbox-batch-action-bar {
  display: contents;
}

.inbox-batch-action-bar__count {
  font-weight: var(--font-weight-medium);
}

.inbox-batch-action-bar__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}

.inbox-batch-action-bar__modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
