<script setup lang="ts">
/**
 * WaveIntakeTab — the wave-scoped demand-intake tab (plan P4, replaces the
 * `WaveTabPlaceholder` at `wave-workspace-intake`). A THIN reuse of
 * `useInboxGrid`/`RowDetailPanel` (see `pages/inbox/inbox-grid/useInboxGrid.ts`)
 * scoped to the current wave via `useWaveWorkspaceContext()`: shows this
 * wave's already-assigned demand documents, offers an "assign more from
 * inbox" deep link to the global `/inbox` page (pre-filtered to
 * unassigned), and a per-row "unassign" action
 * (`unassignDemandFromWave`).
 *
 * `options.waveId` forces `assignment: 'assigned'` and sends the wave ID to
 * the server-paginated inbox endpoint.
 */
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { NButton } from 'naive-ui'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { unassignDemandFromWave } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useInboxGrid } from '@/pages/inbox/inbox-grid/useInboxGrid'
import { buildInboxColumns } from '@/pages/inbox/inbox-grid/columns'
import RowDetailPanel from '@/pages/inbox/inbox-grid/RowDetailPanel.vue'
import type { DemandInboxRow } from '@/entities/demand'

const { t } = useI18n({ useScope: 'global' })
const router = useRouter()
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()

const grid = useInboxGrid({ waveId: ctx.waveId })

const unassigningId = ref<number | null>(null)

async function handleUnassign(row: DemandInboxRow): Promise<void> {
  unassigningId.value = row.demandDocumentId
  try {
    await unassignDemandFromWave({ waveId: ctx.waveId.value, demandDocumentId: row.demandDocumentId })
    feedback.success(t('feedback.success'))
    await Promise.all([grid.mutationDone(), ctx.refresh()])
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    unassigningId.value = null
  }
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
  await Promise.all([grid.mutationDone(), ctx.refresh()])
}
</script>

<template>
  <div class="wave-intake-tab">
    <div class="wave-intake-tab__toolbar">
      <NButton type="primary" @click="handleAssignMore">{{ t('inbox.assignMoreFromInbox') }}</NButton>
    </div>

    <DataGrid
      :columns="gridColumns"
      :rows="grid.rows.value"
      row-key="demandDocumentId"
      :loading="grid.loading.value"
      :pagination="{
        server: {
          total: grid.totalCount.value,
          page: grid.page.value,
          pageSize: grid.pageSize.value,
          onChange: grid.onPageChange,
          onSort: grid.onSort,
        },
      }"
      :empty="{ title: t('inbox.empty.noneAssignedToWave') }"
      @row-click="handleRowClick"
    />

    <RowDetailPanel :row="detailRow" :show="showDetail" @update:show="handleDetailVisibility" @changed="handleDetailChanged" />
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
}
</style>
