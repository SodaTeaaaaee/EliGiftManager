<script setup lang="ts">
/**
 * WaveIntakeTab — the wave-scoped demand-intake tab (plan P4, Task 7 turns
 * it into the self-sufficient 波内导入页面). Reuses `useInboxGrid` scoped to
 * the current wave via `useWaveWorkspaceContext()`: shows this wave's
 * already-assigned demand documents with the full inbox capability set
 * (FilterBar + SavedViews + server pagination + selection), and offers
 * three toolbar actions:
 *
 * - 拉取需求 (primary) — `PullDemandsDialog`: browse the unassigned pool
 *   (business-surface segmented + FilterBar + server paging), batch-pull
 *   into the wave via `batchAssignDemandToWave`.
 * - 导入文件入波 — `ImportFileModal` with `:target-wave-id` so a freshly
 *   imported document is assigned into the wave automatically.
 * - 从收件箱分派更多 (secondary) — deep link to `/inbox?assignment=unassigned`.
 *
 * Selection is wired to a batch-unassign action in `DataGrid`'s
 * `#selection-toolbar` (`batchUnassignDemandFromWave`); single-row unassign
 * and the row-detail panel are kept as before.
 */
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton } from 'naive-ui'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { FilterBar, SavedViews } from '@/shared/ui/filter-bar'
import { useFeedback } from '@/shared/ui/feedback'
import { batchUnassignDemandFromWave, unassignDemandFromWave } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useInboxGrid } from '@/pages/inbox/inbox-grid/useInboxGrid'
import { buildInboxColumns } from '@/pages/inbox/inbox-grid/columns'
import RowDetailPanel from '@/pages/inbox/inbox-grid/RowDetailPanel.vue'
import PullDemandsDialog from './intake/PullDemandsDialog.vue'
import ImportFileModal from '@/pages/inbox/ImportFileModal.vue'
import type { DemandInboxRow } from '@/entities/demand'

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()

const { filters, rows, loading, selectedKeys, page, pageSize, totalCount, onPageChange, onSort, mutationDone } =
  useInboxGrid({ waveId: ctx.waveId })

const unassigningId = ref<number | null>(null)

async function handleUnassign(row: DemandInboxRow): Promise<void> {
  unassigningId.value = row.demandDocumentId
  try {
    await unassignDemandFromWave({ waveId: ctx.waveId.value, demandDocumentId: row.demandDocumentId })
    feedback.success(t('feedback.success'))
    await Promise.all([mutationDone(), ctx.refresh()])
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    unassigningId.value = null
  }
}

// ── 批量退单（selection toolbar）──

const unassigningBatch = ref(false)

async function handleBatchUnassign(): Promise<void> {
  const docIds = selectedKeys.value
  if (docIds.length === 0) return
  unassigningBatch.value = true
  try {
    const result = await batchUnassignDemandFromWave({ waveId: ctx.waveId.value, docIds })
    if (result.failureCount > 0) {
      feedback.error(t('waveWorkspace.intake.unassignSomeFailed', { count: result.failureCount }))
    } else {
      feedback.success(t('feedback.success'))
    }
    feedback.receipt({ kind: 'action', summary: t('waveWorkspace.intake.unassignSelected') })
    selectedKeys.value = []
    await Promise.all([mutationDone(), ctx.refresh()])
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    unassigningBatch.value = false
  }
}

function handleSelectedKeysChange(keys: Array<string | number>): void {
  selectedKeys.value = keys as number[]
}

const columns = computed<DataGridColumnSpec<DemandInboxRow>[]>(() => [
  ...buildInboxColumns(t),
  {
    type: 'actions',
    key: 'unassign',
    title: '',
    width: 110,
    render: (row) =>
      h(
        NButton,
        {
          size: 'tiny',
          quaternary: true,
          loading: unassigningId.value === row.demandDocumentId,
          onClick: (event: MouseEvent) => {
            event.stopPropagation()
            void handleUnassign(row)
          },
        },
        { default: () => t('inbox.unassign') },
      ),
  },
])

const gridColumns = computed(() => createColumns(columns.value))

function handleAssignMore(): void {
  void router.push({ name: 'inbox', query: { assignment: 'unassigned' } })
}

// ── 拉取需求 / 波内文件导入 ──

const showPullDialog = ref(false)
const showImportModal = ref(false)

async function handlePulled(_count: number): Promise<void> {
  await Promise.all([mutationDone(), ctx.refresh()])
}

async function handleAssignedToWave(_docIds: number[]): Promise<void> {
  await Promise.all([mutationDone(), ctx.refresh()])
}

// ── Row detail panel ──

const detailRow = ref<DemandInboxRow | null>(null)
const showDetail = ref(false)

function handleRowClick(row: DemandInboxRow): void {
  detailRow.value = row
  showDetail.value = true
}

function handleDetailVisibility(visible: boolean): void {
  showDetail.value = visible
}

async function handleDetailChanged(): Promise<void> {
  await Promise.all([mutationDone(), ctx.refresh()])
}
</script>

<template>
  <div class="wave-intake-tab">
    <div class="wave-intake-tab__toolbar">
      <NButton secondary @click="handleAssignMore">{{ t('inbox.assignMoreFromInbox') }}</NButton>
      <NButton @click="showImportModal = true">{{ t('waveWorkspace.intake.importIntoWave') }}</NButton>
      <NButton type="primary" @click="showPullDialog = true">{{ t('waveWorkspace.intake.pullDemands') }}</NButton>
    </div>

    <SavedViews :filters="filters" scope-id="wave-intake" />
    <FilterBar :filters="filters" />

    <DataGrid
      :columns="gridColumns"
      :rows="rows"
      row-key="demandDocumentId"
      selectable
      :selected-keys="selectedKeys"
      :loading="loading"
      :pagination="{
        server: {
          total: totalCount,
          page: page,
          pageSize: pageSize,
          onChange: onPageChange,
          onSort: onSort,
        },
      }"
      :empty="{ title: t('inbox.empty.noneAssignedToWave') }"
      @update:selected-keys="handleSelectedKeysChange"
      @row-click="handleRowClick"
    >
      <template #selection-toolbar>
        <span class="wave-intake-tab__selection-count">
          {{ t('inbox.batch.selected', { n: selectedKeys.length }) }}
        </span>
        <NButton
          size="small"
          type="error"
          :loading="unassigningBatch"
          class="wave-intake-tab__selection-action"
          @click="handleBatchUnassign"
        >
          {{ t('waveWorkspace.intake.unassignSelected') }}
        </NButton>
      </template>
    </DataGrid>

    <RowDetailPanel :row="detailRow" :show="showDetail" @update:show="handleDetailVisibility" @changed="handleDetailChanged" />

    <PullDemandsDialog v-model:show="showPullDialog" :wave-id="ctx.waveId.value" @pulled="handlePulled" />
    <ImportFileModal
      v-model:show="showImportModal"
      :target-wave-id="ctx.waveId.value"
      @assigned-to-wave="handleAssignedToWave"
    />
  </div>
</template>

<style scoped>
.wave-intake-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-intake-tab__toolbar {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.wave-intake-tab__selection-count {
  font-weight: var(--font-weight-medium);
}

.wave-intake-tab__selection-action {
  margin-left: auto;
}
</style>
