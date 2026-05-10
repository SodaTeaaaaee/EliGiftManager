<script setup lang="ts">
import { DownloadOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NDataTable,
  NIcon,
  NPagination,
  NSelect,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { useTableMode } from '@/shared/model/settings'
import { useAdaptiveTable } from '@/shared/composables/useAdaptiveTable'
import AdaptivePaginationIndicator from '@/shared/ui/table/AdaptivePaginationIndicator.vue'
import AdaptiveTableMeasureLayer from '@/shared/ui/table/AdaptiveTableMeasureLayer.vue'
import {
  useTableSort,
  nextSortOrderAscFirst,
  type SortDescriptor,
} from '@/shared/composables/useTableSort'
import {
  bindDefaultAddresses,
  exportOrderCSV,
  isWailsRuntimeAvailable,
  listDispatchRecords,
  listTemplates,
  listWaves,
  previewExport,
  WAILS_PREVIEW_MESSAGE,
  type DispatchRecordItem,
  type TemplateItem,
  type WaveItem,
} from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const wave = ref<WaveItem | null>(null)
const templates = ref<TemplateItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const isBindingAddresses = ref(false)
const errorMessage = ref('')

const tableMode = useTableMode()

// Sort descriptors for export records
const exportSortDescriptors: SortDescriptor<DispatchRecordItem>[] = [
  { key: 'memberNickname', getValue: (r) => r.memberNickname || r.platformUid },
  { key: 'memberPlatform', getValue: (r) => r.memberPlatform },
  { key: 'productName', getValue: (r) => r.productName },
  { key: 'quantity', getValue: (r) => r.quantity },
  { key: 'hasAddress', compare: (a, b) => Number(a.hasAddress) - Number(b.hasAddress) },
  { key: 'address', getValue: (r) => r.address || '' },
]

const {
  sortedItems: sortedRecords,
  sortState,
  applySorter,
} = useTableSort(records, exportSortDescriptors)

const pendingAddressCount = computed(() => wave.value?.pendingAddressRecords ?? 0)

const platformTemplateSelections = ref<Record<string, number | null>>({})

const exportPlatforms = computed(() => {
  const platforms = [...new Set(records.value.map((r) => r.productPlatform))]
  for (const platform of platforms) {
    if (!(platform in platformTemplateSelections.value)) {
      const candidates = templates.value.filter(
        (t) => t.type === 'export_order' && t.platform === platform,
      )
      platformTemplateSelections.value[platform] = candidates.length === 1 ? candidates[0].id : null
    }
  }
  return platforms.map((platform) => {
    const candidates = templates.value.filter(
      (t) => t.type === 'export_order' && t.platform === platform,
    )
    return {
      platform,
      templateId: platformTemplateSelections.value[platform] ?? null,
      options: candidates.map((t) => ({ label: t.name, value: t.id })),
    }
  })
})

const recordColumns = computed<DataTableColumns<DispatchRecordItem>>(() => [
  {
    title: '会员',
    key: 'memberNickname',
    minWidth: 120,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'memberNickname' ? sortState.value.order : false,
    render: (row) => row.memberNickname || row.platformUid,
  },
  {
    title: '平台',
    key: 'memberPlatform',
    width: 100,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'memberPlatform' ? sortState.value.order : false,
  },
  {
    title: '礼物',
    key: 'productName',
    minWidth: 140,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'productName' ? sortState.value.order : false,
  },
  {
    title: '数量',
    key: 'quantity',
    width: 80,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'quantity' ? sortState.value.order : false,
  },
  {
    title: '地址',
    key: 'hasAddress',
    width: 110,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'hasAddress' ? sortState.value.order : false,
    render: (row) =>
      h(
        NTag,
        { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true },
        { default: () => (row.hasAddress ? '已绑定' : '待补全') },
      ),
  },
  {
    title: '收件信息',
    key: 'address',
    minWidth: 180,
    ellipsis: { tooltip: true },
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: sortState.value.columnKey === 'address' ? sortState.value.order : false,
    render: (row) => row.address || '-',
  },
])

// Adaptive table refs
const exportLayoutRef = ref<HTMLElement | null>(null)
const exportTableRef = ref<HTMLElement | null>(null)
const exportFooterRef = ref<HTMLElement | null>(null)
const exportMeasureLayer = ref<InstanceType<typeof AdaptiveTableMeasureLayer> | null>(null)

const {
  currentPage,
  pageCount: totalPages,
  renderItems: renderRecords,
  tableBodyMaxHeight,
  viewportWidth,
  measurementInvalidationVersion: exportMeasurementVersion,
  measurementRequestId: exportMeasurementRequestId,
  requestRemeasure: requestExportRemeasure,
  applyMeasuredRows: applyExportMeasuredRows,
  handlePageChange,
  refreshLayout,
  teardown,
  init,
} = useAdaptiveTable(sortedRecords, tableMode, {
  layoutRef: exportLayoutRef,
  tableRef: exportTableRef,
  paginationRef: exportFooterRef,
  rowHeightHint: 56,
  contentSignature: () => sortedRecords.value.map((r) => r.id ?? r.memberId).join(','),
})

// Measure columns (includes render fallback for accurate row-height measurement)
const exportMeasureColumns = computed(() => [
  {
    title: '会员',
    key: 'memberNickname',
    minWidth: 120,
    render: (row: any) => row.memberNickname || row.platformUid,
  },
  { title: '平台', key: 'memberPlatform', width: 100 },
  { title: '礼物', key: 'productName', minWidth: 140 },
  { title: '数量', key: 'quantity', width: 80 },
  {
    title: '地址',
    key: 'hasAddress',
    width: 110,
    render: (row: any) =>
      h(
        NTag,
        { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true },
        { default: () => (row.hasAddress ? '已绑定' : '待补全') },
      ),
  },
  { title: '收件信息', key: 'address', minWidth: 180, render: (row: any) => row.address || '-' },
])

let exportMeasureRunning = false
let exportMeasurePending = false

async function runExportRemeasure() {
  if (exportMeasureRunning) {
    exportMeasurePending = true
    return
  }
  exportMeasureRunning = true
  const requestId = exportMeasurementRequestId.value
  try {
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    refreshLayout()
    await nextTick()
    const result = exportMeasureLayer.value?.measure()
    if (!result) return
    exportMeasureLayer.value?.setWidth(viewportWidth.value)
    applyExportMeasuredRows(result.rowHeights, result.headerHeight, requestId)
  } finally {
    exportMeasureRunning = false
    if (exportMeasurePending) {
      exportMeasurePending = false
      await runExportRemeasure()
    }
  }
}

watch(
  [() => tableMode.value, () => exportMeasurementVersion.value],
  async () => {
    if (tableMode.value !== 'paginated') return
    await runExportRemeasure()
  },
  { flush: 'post' },
)

async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return false
  }
  return true
}

async function loadWave() {
  if (!(await guardRuntime())) return
  try {
    const waves = await listWaves()
    wave.value = waves.find((w) => w.id === waveId.value) ?? null
  } catch (e) {
    console.error('加载波次失败', e)
  }
}

async function loadTemplates() {
  if (!(await guardRuntime())) return
  try {
    templates.value = await listTemplates()
  } catch (e) {
    console.error('加载模板失败', e)
  }
}

async function loadRecords() {
  if (!waveId.value) return
  try {
    records.value = await listDispatchRecords(waveId.value)
  } catch (e) {
    console.error('加载发货记录失败', e)
  }
}

async function syncAvailableAddresses(options?: { silent?: boolean }) {
  if (!waveId.value) return
  isBindingAddresses.value = true
  try {
    const result = await bindDefaultAddresses(waveId.value)
    await loadWave()
    await loadRecords()
    if (!options?.silent) {
      message.success(
        '已同步 ' + result.updated + ' 条已有地址，' + result.skipped + ' 条仍缺地址跳过',
      )
    }
  } catch (e) {
    if (!options?.silent) message.error(String(e))
  } finally {
    isBindingAddresses.value = false
  }
}

async function handleBindAddresses() {
  if (!waveId.value) return message.warning('请先选择发货任务')
  await syncAvailableAddresses()
}

async function handleExport() {
  if (!waveId.value) return message.warning('请选择发货任务')
  try {
    const preview = await previewExport(waveId.value)
    if (preview.missingAddressCount > 0) {
      message.warning('仍有 ' + preview.missingAddressCount + ' 条记录缺失地址，请先补全后再导出')
      return
    }
    const platforms = exportPlatforms.value
    if (!platforms.length) {
      message.warning('无导出数据')
      return
    }
    const missingTemplate = platforms.find((ep) => !ep.templateId)
    if (missingTemplate) {
      message.warning(`平台 ${missingTemplate.platform} 未选择导出模板`)
      return
    }
    for (const ep of platforms) {
      if (!ep.templateId) continue
      const path = await exportOrderCSV(waveId.value, ep.templateId)
      message.success(`${ep.platform} 清单已导出：${path}`)
    }
    await loadWave()
  } catch (e) {
    message.error(String(e))
  }
}

function goPrev() {
  router.push({ name: 'waves-step-preview', params: { waveId: String(waveId.value) } })
}

onMounted(async () => {
  await loadWave()
  await loadTemplates()
  await loadRecords()
  if (waveId.value) await syncAvailableAddresses({ silent: true })
  await init()
  await nextTick()
  requestExportRemeasure()
})

onUnmounted(() => {
  teardown()
})
</script>
<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center gap-2 shrink-0 px-1 py-2">
      <NIcon size="18"><DownloadOutline /></NIcon>
      <span class="font-semibold text-sm">步骤四：异常检查与导出</span>
      <NTag v-if="pendingAddressCount > 0" type="warning" size="small" round>
        {{ pendingAddressCount }} 条待补全
      </NTag>
      <NButton
        v-if="wave && pendingAddressCount > 0"
        size="small"
        type="warning"
        :loading="isBindingAddresses"
        @click="handleBindAddresses"
        class="ml-auto"
        >一键同步已有地址</NButton
      >
    </div>

    <NAlert v-if="errorMessage" type="warning" :show-icon="false" class="mx-1 mb-3">
      {{ errorMessage }}
    </NAlert>

    <!-- Export controls (moved above table) -->
    <div class="shrink-0 px-1 pb-2">
      <div class="p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
        <span class="text-xs text-gray-500 block mb-2 font-medium">导出发货清单</span>
        <div v-if="exportPlatforms.length" class="space-y-2 mb-3">
          <div class="text-xs text-gray-400">导出模板（已自动匹配）</div>
          <div v-for="ep in exportPlatforms" :key="ep.platform" class="flex items-center gap-2">
            <NTag size="small" round>{{ ep.platform }}</NTag>
            <NSelect
              :value="ep.templateId"
              :options="ep.options"
              size="small"
              style="width: 220px"
              placeholder="选择导出模板"
              @update:value="
                (v: number) => {
                  platformTemplateSelections[ep.platform] = v
                }
              "
            />
          </div>
        </div>
        <NButton block type="success" @click="handleExport">
          <template #icon
            ><NIcon><DownloadOutline /></NIcon
          ></template>
          生成发货清单
        </NButton>
      </div>
    </div>

    <!-- Table viewport -->
    <div ref="exportLayoutRef" class="flex-1 min-h-0 flex flex-col overflow-hidden px-1">
      <div
        v-if="tableMode === 'scroll'"
        ref="exportTableRef"
        class="flex-1 min-h-0 overflow-hidden"
      >
        <NDataTable
          :columns="recordColumns"
          :data="renderRecords"
          :bordered="false"
          :remote="true"
          :pagination="false"
          :max-height="tableBodyMaxHeight"
          size="small"
          @update:sorter="
            (s: any) => applySorter({ columnKey: s?.columnKey ?? null, order: s?.order ?? false })
          "
        />
      </div>
      <template v-if="tableMode === 'paginated'">
        <div ref="exportTableRef" class="shrink-0">
          <NDataTable
            :columns="recordColumns"
            :data="renderRecords"
            :bordered="false"
            :remote="true"
            :pagination="false"
            size="small"
            @update:sorter="
              (s: any) => applySorter({ columnKey: s?.columnKey ?? null, order: s?.order ?? false })
            "
          />
        </div>
        <AdaptivePaginationIndicator :page="currentPage" :page-count="totalPages" />
      </template>
    </div>

    <!-- Footer (sibling of viewport) -->
    <div
      v-if="tableMode === 'paginated'"
      ref="exportFooterRef"
      class="flex justify-center shrink-0"
      style="padding: 8px 0 12px 0"
    >
      <div style="transform: scale(1.3); transform-origin: top center; display: inline-flex">
        <NPagination
          :page="currentPage"
          :page-count="totalPages"
          size="small"
          @update:page="handlePageChange"
        />
      </div>
    </div>

    <!-- Bottom nav -->
    <div
      class="flex justify-between shrink-0 pt-3 pb-1 px-1 border-t border-gray-100 dark:border-gray-700"
    >
      <NButton @click="goPrev">上一步</NButton>
      <div></div>
    </div>

    <!-- Measure layer -->
    <AdaptiveTableMeasureLayer
      v-if="tableMode === 'paginated' && records.length"
      ref="exportMeasureLayer"
      :data="sortedRecords"
      :columns="exportMeasureColumns"
      :width="viewportWidth"
      size="small"
    />
  </div>
</template>
