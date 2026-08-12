<script setup lang="ts">
/**
 * BatchActionBar (inbox) — mounted in `DataGrid`'s `#selection-toolbar` slot
 * for the demand-inbox grid (plan P4). Offers the single batch action:
 * assign the current selection to a wave (`batchAssignDemandToWave`),
 * surfacing per-item partial-success honestly (never a blanket
 * success/failure).
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NInput, NModal, NSelect } from 'naive-ui'
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
const router = useRouter()

const showPicker = ref(false)
const waveOptions = ref<SelectOption[]>([])
const wavesLoading = ref(false)
const targetWaveId = ref<number | null>(null)
const assigning = ref(false)
const waveKeyword = ref('')
const waveListTruncated = ref(false)

async function loadWaveOptions(): Promise<void> {
  wavesLoading.value = true
  try {
    const page = await listWavesFiltered({
      page: 1,
      pageSize: 200,
      sortBy: 'updatedAt',
      sortDesc: true,
      nameKeyword: waveKeyword.value.trim() || undefined,
    })
    // The backend only exact-matches a single lifecycleStage string (see
    // filterWaves in internal/app/wave_fulfillment_filter_usecase.go) and the
    // domain has no "active" pseudo-value, so closed waves are excluded here.
    waveOptions.value = page.items
      .filter((wave) => wave.lifecycleStage !== 'closed')
      .map((wave) => ({ label: `${wave.name} (${wave.waveNo})`, value: wave.id }))
    waveListTruncated.value = (page.pagination.totalCount ?? 0) > 200
  } catch (err) {
    waveOptions.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    wavesLoading.value = false
  }
}

function openPicker(): void {
  // Reset the keyword while the picker is still closed so the watcher below
  // (which only reloads while open) skips it — openPicker loads explicitly.
  waveKeyword.value = ''
  targetWaveId.value = null
  showPicker.value = true
  void loadWaveOptions()
}

let keywordTimer: ReturnType<typeof setTimeout> | undefined
watch(waveKeyword, () => {
  if (!showPicker.value) return
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
  keywordTimer = setTimeout(() => void loadWaveOptions(), 300)
})

onBeforeUnmount(() => {
  if (keywordTimer !== undefined) clearTimeout(keywordTimer)
})

const canConfirm = computed(() => targetWaveId.value != null && props.selectedRows.length > 0 && !assigning.value)

async function handleConfirm(): Promise<void> {
  if (!canConfirm.value || targetWaveId.value == null) return
  // Capture before the await — the NSelect stays interactive while the batch
  // call is in flight, and the receipt/jump must point at the wave that was
  // actually assigned to, not whatever the user re-picked meanwhile.
  const targetId = targetWaveId.value
  assigning.value = true
  try {
    const result = await batchAssignDemandToWave({
      waveId: targetId,
      docIds: props.selectedRows.map((row) => row.demandDocumentId),
    })
    if (result.failureCount > 0) {
      feedback.error(t('fulfillmentGrid.batch.someFailed', { count: result.failureCount }))
    } else {
      feedback.success(t('feedback.success'))
    }
    feedback.receipt({ kind: 'action', summary: t('inbox.batch.assignToWaveDone', { n: result.successCount }) })
    showPicker.value = false
    emit('done')
    // ReceiptTray entries are read-only (no action callback), so the one-click
    // path to the target wave's intake is a direct navigation after success.
    void router.push({ name: 'wave-workspace-intake', params: { id: targetId } })
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
      <div class="inbox-batch-action-bar__picker">
        <NInput v-model:value="waveKeyword" clearable :placeholder="t('inbox.batch.waveKeywordPlaceholder')" />
        <NAlert v-if="waveListTruncated" type="info" :bordered="false">
          {{ t('inbox.batch.waveListTruncated') }}
        </NAlert>
        <NSelect
          v-model:value="targetWaveId"
          :options="waveOptions"
          :loading="wavesLoading"
          filterable
          :placeholder="t('inbox.batch.chooseWave')"
        />
      </div>
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

.inbox-batch-action-bar__picker {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.inbox-batch-action-bar__modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
