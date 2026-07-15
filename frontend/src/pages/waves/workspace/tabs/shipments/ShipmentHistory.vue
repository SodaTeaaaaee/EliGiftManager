<script setup lang="ts">
/**
 * ShipmentHistory — 发货记录 list (P5 shipment-backfill sub-area, plan
 * §3.3.4 second bullet + §5.2's correction/void pairing). Fetches
 * `listShipmentsByWavePage` and wires row-level correct/void actions to the two
 * bridge wrappers `updateShipment`/`voidShipment` — both compensating
 * writes OUTSIDE the undo/redo command history by design
 * (`shipment_lifecycle_usecase.go:13-18`), surfaced honestly via
 * `history.outsideUndoNotice` rather than a misleading "undo" affordance.
 *
 * `refreshSignal` is a plain number bumped by the parent tab after an
 * import/manual submission elsewhere in the tab — watched here (not
 * `:key`-remounted) so this view's own state (naive-ui's internal table
 * paging state) isn't discarded on every cross-tab mutation.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { SectionCard } from '@/shared/ui/cards'
import { CalloutBar } from '@/shared/ui/guidance'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { listShipmentsByWavePage } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { buildShipmentHistoryColumns } from './history-columns'
import CorrectShipmentDialog from './CorrectShipmentDialog.vue'
import VoidShipmentDialog from './VoidShipmentDialog.vue'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{ refreshSignal: number }>()

const { t } = useI18n({ useScope: 'global' })
const ctx = useWaveWorkspaceContext()

const shipments = ref<dto.ShipmentDTO[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(50)
const totalCount = ref(0)
const sortBy = ref<string | null>(null)
const sortDir = ref<'asc' | 'desc' | null>(null)

async function loadShipments(): Promise<void> {
  loading.value = true
  try {
    const result = await listShipmentsByWavePage({
      waveId: ctx.waveId.value,
      sortBy: sortBy.value ?? undefined,
      sortDir: sortDir.value ?? undefined,
      limit: pageSize.value,
      offset: (page.value - 1) * pageSize.value,
    })
    totalCount.value = result.totalCount
    if ((page.value - 1) * pageSize.value >= result.totalCount && page.value > 1) {
      page.value = Math.max(1, Math.ceil(result.totalCount / pageSize.value))
      await loadShipments()
      return
    }
    shipments.value = result.items
  } finally {
    loading.value = false
  }
}

watch(ctx.waveId, () => {
  page.value = 1
  void loadShipments()
}, { immediate: true })
watch(() => props.refreshSignal, () => void loadShipments())

function onPageChange(nextPage: number, nextPageSize: number): void {
  page.value = nextPageSize === pageSize.value ? nextPage : 1
  pageSize.value = nextPageSize
  void loadShipments()
}

function onSort(nextSortBy: string | null, nextSortDir: 'asc' | 'desc' | null): void {
  sortBy.value = nextSortBy
  sortDir.value = nextSortDir
  page.value = 1
  void loadShipments()
}

// ── Correct / void dialogs ──

const correctingShipment = ref<dto.ShipmentDTO | null>(null)
const showCorrectDialog = ref(false)
const voidingShipment = ref<dto.ShipmentDTO | null>(null)
const showVoidDialog = ref(false)

function openCorrect(row: dto.ShipmentDTO): void {
  correctingShipment.value = row
  showCorrectDialog.value = true
}

function openVoid(row: dto.ShipmentDTO): void {
  voidingShipment.value = row
  showVoidDialog.value = true
}

async function handleMutated(): Promise<void> {
  await Promise.all([loadShipments(), ctx.refresh()])
}

const columns = computed(() =>
  createColumns(buildShipmentHistoryColumns(t, { onCorrect: openCorrect, onVoid: openVoid })),
)
</script>

<template>
  <SectionCard :title="t('waveWorkspace.shipments.history.title')">
    <div class="shipment-history">
      <CalloutBar tone="info" :message="t('waveWorkspace.shipments.history.outsideUndoNotice')" />

      <DataGrid
        :columns="columns"
        :rows="shipments"
        row-key="id"
        :loading="loading"
        :pagination="{ server: { total: totalCount, page, pageSize, onChange: onPageChange, onSort } }"
        :empty="{ title: t('waveWorkspace.shipments.history.empty') }"
      />
    </div>

    <CorrectShipmentDialog
      v-model:show="showCorrectDialog"
      :shipment="correctingShipment"
      @done="handleMutated"
    />
    <VoidShipmentDialog
      v-model:show="showVoidDialog"
      :shipment="voidingShipment"
      @done="handleMutated"
    />
  </SectionCard>
</template>

<style scoped>
.shipment-history {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
