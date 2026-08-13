<script setup lang="ts">
/**
 * RowDetailPanel — the demand-inbox grid's row inspector (plan P4, wide
 * side panel per `docs/FRONTEND-REDESIGN-PLAN.md:249`, replacing the old
 * tree's narrow 3/8-column 7-col-table layout — `DetailDrawer size="lg"`,
 * the widest house drawer variant). Hosts, for one `DemandInboxRow`:
 *
 * - the document header (kind/capture mode/source channel/surface/document
 *   no/integration profile),
 * - its demand lines (`listDemandLines`), each with routing-state badges,
 * - per-line routing edit (`updateDemandLineRouting`) and multi-select bulk
 *   routing edit (`batchUpdateDemandLineRouting`) — both require an explicit
 *   `routingDisposition` AND `recipientInputState` selection (the backend
 *   rejects an empty `recipientInputState` as invalid, so this never sends a
 *   blank value the way the old tree's bulk-apply did),
 * - an "open assigned wave" deep link when the document is already routed.
 *
 * Emits `'changed'` after any routing mutation — the caller (`InboxPage.vue`
 * / `WaveIntakeTab.vue`) wires this to `useInboxGrid().mutationDone()`.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton, NCheckbox, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { StatusBadge } from '@/shared/ui/status'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary, glossaryTables } from '@/shared/i18n/glossary'
import { listDemandLines, updateDemandLineRouting, batchUpdateDemandLineRouting } from '@/shared/api/bridge'
import type { DemandInboxRow, DemandLine } from '@/entities/demand'

const props = defineProps<{
  row: DemandInboxRow | null
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'changed'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}

function handleOpenAssignedWave(): void {
  const waveId = props.row?.assignedWaveId
  if (waveId == null) return
  void router.push({ name: 'wave-workspace', params: { id: waveId } })
}

// ── Demand lines ──

const lines = ref<DemandLine[]>([])
const linesLoading = ref(false)

async function loadLines(): Promise<void> {
  const documentId = props.row?.demandDocumentId
  if (documentId == null) {
    lines.value = []
    return
  }
  linesLoading.value = true
  try {
    lines.value = await listDemandLines(documentId)
  } catch (err) {
    lines.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    linesLoading.value = false
  }
}

const routingDispositionOptions = computed<SelectOption[]>(() =>
  Object.keys(glossaryTables.routingDisposition).map((value) => ({ label: glossaryLabel('routingDisposition', value), value })),
)

const recipientInputStateOptions = computed<SelectOption[]>(() =>
  Object.keys(glossaryTables.recipientInputState).map((value) => ({ label: glossaryLabel('recipientInputState', value), value })),
)

// ── Per-line inline edit ──

const editingLineId = ref<number | null>(null)
const editDraft = reactive<{ routingDisposition: string | null; recipientInputState: string | null }>({
  routingDisposition: null,
  recipientInputState: null,
})
const savingLineId = ref<number | null>(null)

function startEdit(line: DemandLine): void {
  editingLineId.value = line.id
  editDraft.routingDisposition = line.routingDisposition
  editDraft.recipientInputState = line.recipientInputState
}

function cancelEdit(): void {
  editingLineId.value = null
}

async function saveEdit(line: DemandLine): Promise<void> {
  if (!editDraft.routingDisposition || !editDraft.recipientInputState) return
  savingLineId.value = line.id
  try {
    await updateDemandLineRouting({
      demandLineId: line.id,
      routingDisposition: editDraft.routingDisposition,
      recipientInputState: editDraft.recipientInputState,
      routingReasonCode: line.routingReasonCode ?? '',
    })
    feedback.success(t('feedback.success'))
    editingLineId.value = null
    await loadLines()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    savingLineId.value = null
  }
}

// ── Multi-select bulk routing edit ──

const selectedLineIds = ref<number[]>([])
const bulkRoutingDisposition = ref<string | null>(null)
const bulkRecipientInputState = ref<string | null>(null)
const applyingBulk = ref(false)

function toggleLineSelection(lineId: number, checked: boolean): void {
  if (checked) {
    if (!selectedLineIds.value.includes(lineId)) selectedLineIds.value = [...selectedLineIds.value, lineId]
  } else {
    selectedLineIds.value = selectedLineIds.value.filter((id) => id !== lineId)
  }
}

const canApplyBulk = computed(
  () => selectedLineIds.value.length > 0 && !!bulkRoutingDisposition.value && !!bulkRecipientInputState.value && !applyingBulk.value,
)

async function applyBulkRouting(): Promise<void> {
  if (!canApplyBulk.value || !bulkRoutingDisposition.value || !bulkRecipientInputState.value) return
  applyingBulk.value = true
  try {
    const result = await batchUpdateDemandLineRouting({
      updates: selectedLineIds.value.map((demandLineId) => {
        const line = lines.value.find((candidate) => candidate.id === demandLineId)
        return {
          demandLineId,
          routingDisposition: bulkRoutingDisposition.value as string,
          recipientInputState: bulkRecipientInputState.value as string,
          routingReasonCode: line?.routingReasonCode ?? '',
        }
      }),
    })
    if (result.errors.length > 0) {
      feedback.error(t('fulfillmentGrid.batch.someFailed', { count: result.errors.length }))
    } else {
      feedback.success(t('feedback.success'))
    }
    selectedLineIds.value = []
    bulkRoutingDisposition.value = null
    bulkRecipientInputState.value = null
    await loadLines()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    applyingBulk.value = false
  }
}

// Refetch this document's lines whenever the panel opens, or a different row
// is clicked while it is already open. Resets all editor/selection state so
// nothing from a previous row leaks into the new one.
watch(
  () => [props.show, props.row?.demandDocumentId] as const,
  ([show]) => {
    if (!show) return
    editingLineId.value = null
    selectedLineIds.value = []
    bulkRoutingDisposition.value = null
    bulkRecipientInputState.value = null
    void loadLines()
  },
  { immediate: true },
)
</script>

<template>
  <DetailDrawer :show="show" size="lg" :title="t('inbox.detail.title')" @update:show="handleUpdateShow">
    <template v-if="row">
      <SectionCard flat>
        <div class="row-detail-panel__header-grid">
          <StatusBadge dimension="demandKind" :value="row.kind" size="sm" />
          <span class="row-detail-panel__header-item">{{ t('inbox.columns.captureMode') }}: <StatusBadge dimension="captureMode" :value="row.captureMode" size="sm" /></span>
          <span class="row-detail-panel__header-item">{{ t('inbox.columns.sourceChannel') }}: {{ row.sourceChannel }}</span>
          <span v-if="row.sourceSurface" class="row-detail-panel__header-item">{{ t('inbox.detail.sourceSurface') }}: {{ row.sourceSurface }}</span>
          <span class="row-detail-panel__header-item">{{ t('inbox.columns.sourceDoc') }}: {{ row.sourceDocumentNo || '—' }}</span>
          <span class="row-detail-panel__header-item">{{ t('inbox.columns.profile') }}: {{ row.integrationProfileLabel || '—' }}</span>
        </div>

        <div class="row-detail-panel__assignment">
          <template v-if="row.assigned">
            <span class="row-detail-panel__header-item">{{ t('inbox.columns.assignedWave') }}: {{ row.assignedWaveLabel || '—' }}</span>
            <NButton v-if="row.assignedWaveId" size="small" quaternary @click="handleOpenAssignedWave">
              {{ t('inbox.openAssignedWave') }}
            </NButton>
          </template>
          <span v-else class="row-detail-panel__header-item">{{ t('inbox.assignment.unassigned') }}</span>
        </div>
      </SectionCard>

      <SectionCard flat :title="t('inbox.detail.routing')">
        <EmptyState v-if="!linesLoading && lines.length === 0" size="sm" :title="t('fulfillmentGrid.detail.noAdjustments')" />

        <template v-else>
          <div v-if="selectedLineIds.length > 0" class="row-detail-panel__bulk-bar">
            <span class="row-detail-panel__header-item">{{ t('inbox.detail.selectedLines', { n: selectedLineIds.length }) }}</span>
            <NSelect
              v-model:value="bulkRoutingDisposition"
              :options="routingDispositionOptions"
              size="small"
              style="width: 180px"
              :placeholder="t('inbox.detail.editRouting')"
            />
            <NSelect
              v-model:value="bulkRecipientInputState"
              :options="recipientInputStateOptions"
              size="small"
              style="width: 180px"
              :placeholder="t('inbox.detail.editRouting')"
            />
            <NButton size="small" type="primary" :loading="applyingBulk" :disabled="!canApplyBulk" @click="applyBulkRouting">
              {{ t('common.apply') }}
            </NButton>
          </div>

          <ul class="row-detail-panel__list">
            <li v-for="line in lines" :key="line.id" class="row-detail-panel__list-item">
              <NCheckbox
                :checked="selectedLineIds.includes(line.id)"
                @update:checked="(checked: boolean) => toggleLineSelection(line.id, checked)"
              />
              <div class="row-detail-panel__list-main">
                <span class="row-detail-panel__list-primary">{{ line.externalTitle }} × {{ line.requestedQuantity }}</span>

                <template v-if="editingLineId === line.id">
                  <NSelect
                    v-model:value="editDraft.routingDisposition"
                    :options="routingDispositionOptions"
                    size="small"
                    style="width: 160px"
                  />
                  <NSelect
                    v-model:value="editDraft.recipientInputState"
                    :options="recipientInputStateOptions"
                    size="small"
                    style="width: 160px"
                  />
                  <NButton size="tiny" type="primary" :loading="savingLineId === line.id" @click="saveEdit(line)">
                    {{ t('common.save') }}
                  </NButton>
                  <NButton size="tiny" quaternary @click="cancelEdit">{{ t('common.cancel') }}</NButton>
                </template>
                <template v-else>
                  <StatusBadge dimension="routingDisposition" :value="line.routingDisposition" size="sm" />
                  <StatusBadge dimension="recipientInputState" :value="line.recipientInputState" size="sm" />
                  <NButton size="tiny" quaternary @click="startEdit(line)">{{ t('common.edit') }}</NButton>
                </template>
              </div>
            </li>
          </ul>
        </template>
      </SectionCard>
    </template>
  </DetailDrawer>
</template>

<style scoped>
.row-detail-panel__header-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
}

.row-detail-panel__header-item {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.row-detail-panel__assignment {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

.row-detail-panel__bulk-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-accent-subtle);
}

.row-detail-panel__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.row-detail-panel__list-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--card-bg);
}

.row-detail-panel__list-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.row-detail-panel__list-primary {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
  flex-basis: 100%;
}
</style>
