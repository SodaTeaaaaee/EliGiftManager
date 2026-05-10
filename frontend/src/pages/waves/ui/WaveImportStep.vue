<script setup lang="ts">
import { CloudUploadOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useAdaptiveTable } from '@/shared/composables/useAdaptiveTable'
import { useTableMode } from '@/shared/model/settings'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NDataTable,
  NIcon,
  NPagination,
  NPopconfirm,
  NSelect,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import AdaptivePaginationIndicator from '@/shared/ui/table/AdaptivePaginationIndicator.vue'
import AdaptiveTableMeasureLayer from '@/shared/ui/table/AdaptiveTableMeasureLayer.vue'
import {
  useTableSort,
  nextSortOrderAscFirst,
  type SortDescriptor,
} from '@/shared/composables/useTableSort'
import {
  importDispatchWave,
  importToWave,
  isWailsRuntimeAvailable,
  listProductsWithTags,
  listTemplates,
  listWaveMembers,
  pickCSVFile,
  pickZIPFile,
  removeMemberFromWave,
  removeProductFromWave,
  WAILS_PREVIEW_MESSAGE,
  type MemberItem,
  type TemplateItem,
} from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const templates = ref<TemplateItem[]>([])
const importTemplateId = ref<number | null>(null)
const productTemplateId = ref<number | null>(null)
const waveMembers = ref<MemberItem[]>([])
const isMembersLoading = ref(false)
const errorMessage = ref('')

const productTemplates = computed(() =>
  templates.value.filter((t) => t.type === 'import_product').map(toOption),
)
const dispatchTemplates = computed(() =>
  templates.value.filter((t) => t.type === 'import_dispatch_record').map(toOption),
)

function toOption(template: TemplateItem) {
  return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id }
}

function templateFormat(templateId: number | null): string {
  if (!templateId) return 'csv'
  const t = templates.value.find((t) => t.id === templateId)
  if (!t) return 'csv'
  try {
    const rules = JSON.parse(t.mappingRules)
    return rules.format || 'csv'
  } catch {
    return 'csv'
  }
}

const productFileExt = computed(() =>
  templateFormat(productTemplateId.value) === 'zip' ? 'ZIP' : 'CSV',
)

// ── line-clamped cell renderer ──
const MAX_LINES = 4
const LINE_HEIGHT = 21

function clampedText(text: string, lines = MAX_LINES) {
  return h(
    'div',
    {
      style: {
        display: '-webkit-box',
        '-webkit-line-clamp': String(lines),
        '-webkit-box-orient': 'vertical',
        overflow: 'hidden',
        wordBreak: 'break-all',
        lineHeight: String(LINE_HEIGHT) + 'px',
      },
    },
    String(text ?? ''),
  )
}

// ══════════════════════════════════════════════
// adaptive paging — product + member tables
// ══════════════════════════════════════════════

// ── data refs ──
const allProducts = ref<{ id: number; name: string; factorySku: string }[]>([])
const productIndexMap = computed(() => {
  const map = new Map<number, number>()
  sortedProducts.value.forEach((p, i) => map.set(p.id, i + 1))
  return map
})
const memberIndexMap = computed(() => {
  const map = new Map<number, number>()
  sortedMembers.value.forEach((m, i) => map.set(m.id, i + 1))
  return map
})
const isProductLoading = ref(false)

const productSortDescriptors: SortDescriptor<any>[] = [
  { key: 'name', getValue: (p: any) => p.name },
  { key: 'factorySku', getValue: (p: any) => p.factorySku || '' },
]

const {
  sortedItems: sortedProducts,
  sortState: productSortState,
  applyNaiveSorterEvent: applyProductSorter,
} = useTableSort(allProducts, productSortDescriptors)

const showSkuColumn = computed(
  () => productViewportWidth.value === 0 || productViewportWidth.value >= 400,
)
const showMemberExtraColumns = computed(
  () => memberViewportWidth.value === 0 || memberViewportWidth.value >= 450,
)

const tableMode = useTableMode()

// ── product table composable ──
const productLayoutRef = ref<HTMLElement | null>(null)
const productTableRef = ref<HTMLElement | null>(null)
const productFooterRef = ref<HTMLElement | null>(null)
const productMeasureLayer = ref<InstanceType<typeof AdaptiveTableMeasureLayer> | null>(null)
const memberMeasureLayer = ref<InstanceType<typeof AdaptiveTableMeasureLayer> | null>(null)

const {
  currentPage: productCurrentPage,
  pageCount: productPageCount,
  renderItems: renderProducts,
  tableBodyMaxHeight: productTableBodyMaxHeight,
  viewportWidth: productViewportWidth,
  handlePageChange: handleProductPageChange,
  refreshLayout: refreshProductLayout,
  teardown: teardownProduct,
  init: initProduct,
  applyMeasuredRows: applyProductMeasuredRows,
  pageRanges: productPageRanges,
  schedulePostPaintRefresh: scheduleProductPostPaint,
  measurementInvalidationVersion: productMeasurementVersion,
  measurementRequestId: productMeasurementRequestId,
  requestRemeasure: requestProductRemeasure,
} = useAdaptiveTable(sortedProducts, tableMode, {
  layoutRef: productLayoutRef,
  tableRef: productTableRef,
  paginationRef: productFooterRef,
  rowHeightHint: (w: number) => (w < 550 ? 78 : 68),
  contentSignature: () => sortedProducts.value.map((p) => p.id).join(','),
})

// ── member table composable ──
const memberLayoutRef = ref<HTMLElement | null>(null)
const memberTableRef = ref<HTMLElement | null>(null)
const memberFooterRef = ref<HTMLElement | null>(null)

const memberSortDescriptors: SortDescriptor<MemberItem>[] = [
  { key: 'latestNickname', getValue: (m: MemberItem) => m.latestNickname || '' },
  { key: 'platform', getValue: (m: MemberItem) => m.platform },
  { key: 'platformUid', getValue: (m: MemberItem) => m.platformUid },
  { key: 'giftLevel', getValue: (m: MemberItem) => m.giftLevel || '' },
  { key: 'activeAddressCount', getValue: (m: MemberItem) => m.activeAddressCount },
]

const {
  sortedItems: sortedMembers,
  sortState: memberSortState,
  applyNaiveSorterEvent: applyMemberSorter,
} = useTableSort(waveMembers, memberSortDescriptors)

const {
  currentPage: memberCurrentPage,
  pageCount: memberPageCount,
  renderItems: renderMembers,
  tableBodyMaxHeight: memberTableBodyMaxHeight,
  viewportWidth: memberViewportWidth,
  handlePageChange: handleMemberPageChange,
  refreshLayout: refreshMemberLayout,
  teardown: teardownMember,
  init: initMember,
  applyMeasuredRows: applyMemberMeasuredRows,
  pageRanges: memberPageRanges,
  schedulePostPaintRefresh: scheduleMemberPostPaint,
  measurementInvalidationVersion: memberMeasurementVersion,
  measurementRequestId: memberMeasurementRequestId,
  requestRemeasure: requestMemberRemeasure,
} = useAdaptiveTable(sortedMembers, tableMode, {
  layoutRef: memberLayoutRef,
  tableRef: memberTableRef,
  paginationRef: memberFooterRef,
  rowHeightHint: (w: number) => (w < 550 ? 78 : 68),
  contentSignature: () => sortedMembers.value.map((m) => m.id).join(','),
})

// ── column definitions ──

const memberColumns = computed<DataTableColumns<MemberItem>>(() => {
  const cols: DataTableColumns<MemberItem> = [
    {
      title: '#',
      key: '__index',
      width: 40,
      render: (row: any) =>
        h('span', { style: { color: '#999' } }, String(memberIndexMap.value.get(row.id) ?? '')),
    },
    {
      title: '昵称',
      key: 'latestNickname',
      width: 90,
      sorter: 'default' as const,
      customNextSortOrder: nextSortOrderAscFirst,
      sortOrder:
        memberSortState.value.columnKey === 'latestNickname' ? memberSortState.value.order : false,
      render: (row) => clampedText(row.latestNickname || row.platformUid),
    },
    {
      title: '平台',
      key: 'platform',
      width: 71,
      sorter: 'default' as const,
      customNextSortOrder: nextSortOrderAscFirst,
      sortOrder:
        memberSortState.value.columnKey === 'platform' ? memberSortState.value.order : false,
      render: (row) => clampedText(row.platform),
    },
    {
      title: 'UID',
      key: 'platformUid',
      width: 90,
      sorter: 'default' as const,
      customNextSortOrder: nextSortOrderAscFirst,
      sortOrder:
        memberSortState.value.columnKey === 'platformUid' ? memberSortState.value.order : false,
      render: (row) => clampedText(row.platformUid),
    },
  ]
  if (showMemberExtraColumns.value) {
    cols.push(
      {
        title: '等级',
        key: 'giftLevel',
        width: 50,
        sorter: 'default' as const,
        customNextSortOrder: nextSortOrderAscFirst,
        sortOrder:
          memberSortState.value.columnKey === 'giftLevel' ? memberSortState.value.order : false,
        render: (row) => clampedText(row.giftLevel || '-'),
      },
      {
        title: '地址数',
        key: 'activeAddressCount',
        width: 60,
        sorter: 'default' as const,
        customNextSortOrder: nextSortOrderAscFirst,
        sortOrder:
          memberSortState.value.columnKey === 'activeAddressCount'
            ? memberSortState.value.order
            : false,
      },
    )
  }
  cols.push({
    title: '操作',
    key: '__actions',
    width: 80,
    render(row: any) {
      return h(
        NPopconfirm,
        {
          onPositiveClick: () => handleDeleteMember(row.id),
          negativeText: '取消',
          positiveText: '确认',
        },
        {
          trigger: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
          default: () => '确认从任务中移除此会员？',
        },
      )
    },
  })
  return cols
})

const productDataColumns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    {
      title: '#',
      key: '__index',
      width: 40,
      render: (row: any) =>
        h('span', { style: { color: '#999' } }, String(productIndexMap.value.get(row.id) ?? '')),
    },
    {
      title: '商品名',
      key: 'name',
      width: 140,
      sorter: 'default' as const,
      customNextSortOrder: nextSortOrderAscFirst,
      sortOrder: productSortState.value.columnKey === 'name' ? productSortState.value.order : false,
      render: (row: any) => clampedText(row.name),
    },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'SKU',
      key: 'factorySku',
      width: 160,
      sorter: 'default' as const,
      customNextSortOrder: nextSortOrderAscFirst,
      sortOrder:
        productSortState.value.columnKey === 'factorySku' ? productSortState.value.order : false,
      render: (row: any) =>
        h('span', { style: { whiteSpace: 'nowrap' } }, String(row.factorySku ?? '')),
    })
  }
  cols.push({
    title: '操作',
    key: '__actions',
    width: 80,
    render(row: any) {
      return h(
        NPopconfirm,
        {
          onPositiveClick: () => handleDeleteProduct(row.id),
          negativeText: '取消',
          positiveText: '确认',
        },
        {
          trigger: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
          default: () => '确认从任务中移除此商品？',
        },
      )
    },
  })
  return cols
})

// ── measurement column definitions (for off-screen row packing) ──

const productMeasureColumns = computed(() => {
  const cols: any[] = [
    { title: '#', key: '__index', width: 40 },
    { title: '商品名', key: 'name', width: 140, render: (row: any) => clampedText(row.name) },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'SKU',
      key: 'factorySku',
      width: 160,
      render: (row: any) =>
        h('span', { style: { whiteSpace: 'nowrap' } }, String(row.factorySku ?? '')),
    })
  }
  cols.push({
    title: '操作',
    key: '__actions',
    width: 80,
    render: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
  })
  return cols
})

const memberMeasureColumns = computed(() => {
  const cols: any[] = [
    { title: '#', key: '__index', width: 40 },
    {
      title: '昵称',
      key: 'latestNickname',
      width: 90,
      render: (row: any) => clampedText(row.latestNickname || row.platformUid),
    },
    { title: '平台', key: 'platform', width: 71, render: (row: any) => clampedText(row.platform) },
    {
      title: 'UID',
      key: 'platformUid',
      width: 90,
      render: (row: any) => clampedText(row.platformUid),
    },
  ]
  if (showMemberExtraColumns.value) {
    cols.push({
      title: '等级',
      key: 'giftLevel',
      width: 50,
      render: (row: any) => clampedText(row.giftLevel || '-'),
    })
    cols.push({ title: '地址数', key: 'activeAddressCount', width: 60 })
  }
  cols.push({
    title: '操作',
    key: '__actions',
    width: 80,
    render: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
  })
  return cols
})

// ── single-flight remeasure runners ──

let productMeasureRunning = false
let productMeasurePending = false

async function runProductRemeasure() {
  if (productMeasureRunning) {
    productMeasurePending = true
    return
  }
  productMeasureRunning = true
  const requestId = productMeasurementRequestId.value
  try {
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    refreshProductLayout()
    await nextTick()
    const result = productMeasureLayer.value?.measure()
    if (!result) return
    productMeasureLayer.value?.setWidth(productViewportWidth.value)
    applyProductMeasuredRows(result.rowHeights, result.headerHeight, requestId)
  } finally {
    productMeasureRunning = false
    if (productMeasurePending) {
      productMeasurePending = false
      await runProductRemeasure()
    }
  }
}

let memberMeasureRunning = false
let memberMeasurePending = false

async function runMemberRemeasure() {
  if (memberMeasureRunning) {
    memberMeasurePending = true
    return
  }
  memberMeasureRunning = true
  const requestId = memberMeasurementRequestId.value
  try {
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    refreshMemberLayout()
    await nextTick()
    const result = memberMeasureLayer.value?.measure()
    if (!result) return
    memberMeasureLayer.value?.setWidth(memberViewportWidth.value)
    applyMemberMeasuredRows(result.rowHeights, result.headerHeight, requestId)
  } finally {
    memberMeasureRunning = false
    if (memberMeasurePending) {
      memberMeasurePending = false
      await runMemberRemeasure()
    }
  }
}

// ── data loading ──
async function loadAllProducts() {
  if (!waveId.value) return
  isProductLoading.value = true
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 10000)
    allProducts.value = result.items.map((item) => ({
      id: item.id,
      name: item.name,
      factorySku: item.factorySku || '',
    }))
    await nextTick()
    requestProductRemeasure()
  } catch (e) {
    console.error('加载任务商品失败', e)
  } finally {
    isProductLoading.value = false
  }
}

async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return false
  }
  return true
}

async function loadTemplates() {
  if (!(await guardRuntime())) return
  try {
    templates.value = await listTemplates()
  } catch (e) {
    console.error('加载模板失败', e)
  }
}

async function loadWaveMembers() {
  if (!waveId.value) return
  isMembersLoading.value = true
  try {
    waveMembers.value = await listWaveMembers(waveId.value)
    await nextTick()
    requestMemberRemeasure()
  } catch (e) {
    console.error('加载任务会员失败', e)
  } finally {
    isMembersLoading.value = false
  }
}

async function handleImportProduct() {
  if (!productTemplateId.value) return message.warning('请先选择商品导入模板')
  const fmt = templateFormat(productTemplateId.value)
  const filePath = fmt === 'zip' ? await pickZIPFile() : await pickCSVFile()
  if (!filePath) return
  try {
    await importToWave(waveId.value, filePath, productTemplateId.value)
    message.success('商品导入完成')
    await loadAllProducts()
    await loadWaveMembers()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleDeleteProduct(productId: number) {
  try {
    await removeProductFromWave(waveId.value, productId)
    message.success('已从任务中移除')
    await loadAllProducts()
    await loadWaveMembers()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleDeleteMember(memberId: number) {
  try {
    await removeMemberFromWave(waveId.value, memberId)
    message.success('已从任务中移除')
    await loadWaveMembers()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleImportDispatch() {
  if (!importTemplateId.value) return message.warning('请先选择会员导入模板')
  const filePath = await pickCSVFile()
  if (!filePath) return
  try {
    await importDispatchWave(waveId.value, filePath, importTemplateId.value)
    message.success('会员数据导入完成')
    await loadWaveMembers()
  } catch (e) {
    message.error(String(e))
  }
}

function goNext() {
  router.push({ name: 'waves-step-tags', params: { waveId: String(waveId.value) } })
}

watch(tableMode, () => {
  scheduleProductPostPaint()
  scheduleMemberPostPaint()
})

watch(
  [() => tableMode.value, () => productMeasurementVersion.value],
  async () => {
    if (tableMode.value !== 'paginated') return
    await runProductRemeasure()
  },
  { flush: 'post' },
)

watch(
  [() => tableMode.value, () => memberMeasurementVersion.value],
  async () => {
    if (tableMode.value !== 'paginated') return
    await runMemberRemeasure()
  },
  { flush: 'post' },
)

// ── lifecycle ──
onMounted(async () => {
  await loadTemplates()
  await loadWaveMembers()
  await loadAllProducts()
  await initProduct()
  await initMember()
  await nextTick()
  requestProductRemeasure()
  requestMemberRemeasure()
})

onUnmounted(() => {
  teardownProduct()
  teardownMember()
})
</script>
<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center gap-2 shrink-0 px-1 py-2">
      <NIcon size="18">
        <CloudUploadOutline />
      </NIcon>
      <span class="font-semibold text-sm">步骤一：导入数据</span>
    </div>

    <div class="flex-1 min-h-0 grid gap-4 md:grid-cols-2">
      <!-- 商品导入面板 -->
      <div class="border border-gray-100 dark:border-gray-700 rounded-lg flex flex-col min-h-0">
        <div class="p-3 pb-0 shrink-0">
          <span class="text-xs text-gray-500 block mb-3 font-medium"
            >商品导入（工厂平台 {{ productFileExt }}）</span
          >
          <NSelect
            v-model:value="productTemplateId"
            :options="productTemplates"
            placeholder="选择商品导入模板"
            class="mb-2"
          />
          <NButton block secondary @click="handleImportProduct">导入商品</NButton>
        </div>
        <div v-if="allProducts.length" class="flex-1 min-h-0 flex flex-col overflow-hidden px-3">
          <!-- TABLE VIEWPORT -->
          <div ref="productLayoutRef" class="flex-1 min-h-0 flex flex-col overflow-hidden">
            <!-- scroll mode -->
            <div
              v-if="tableMode === 'scroll'"
              ref="productTableRef"
              class="flex-1 min-h-0 overflow-hidden"
            >
              <NDataTable
                :columns="productDataColumns"
                :data="renderProducts"
                :loading="isProductLoading"
                :remote="true"
                :max-height="productTableBodyMaxHeight"
                :table-layout="'auto'"
                :bordered="false"
                :pagination="false"
                size="small"
                @update:sorter="(s: any) => applyProductSorter(s)"
              />
            </div>
            <!-- paginated mode -->
            <template v-if="tableMode === 'paginated'">
              <div ref="productTableRef" class="shrink-0">
                <NDataTable
                  :columns="productDataColumns"
                  :data="renderProducts"
                  :loading="isProductLoading"
                  :remote="true"
                  :table-layout="'auto'"
                  :bordered="false"
                  :pagination="false"
                  size="small"
                  @update:sorter="(s: any) => applyProductSorter(s)"
                />
              </div>
              <AdaptivePaginationIndicator
                :page="productCurrentPage"
                :page-count="productPageCount"
              />
            </template>
          </div>
          <!-- FOOTER (sibling, outside viewport) -->
          <div
            v-if="tableMode === 'paginated'"
            ref="productFooterRef"
            class="flex justify-center shrink-0"
            style="padding: 8px 0 12px 0"
          >
            <div style="transform: scale(1.3); transform-origin: top center; display: inline-flex">
              <NPagination
                :page="productCurrentPage"
                :page-count="productPageCount"
                size="small"
                @update:page="handleProductPageChange"
              />
            </div>
          </div>
        </div>
        <div v-else class="flex-1" />
      </div>

      <!-- 会员导入面板 -->
      <div class="border border-gray-100 dark:border-gray-700 rounded-lg flex flex-col min-h-0">
        <div class="p-3 pb-0 shrink-0">
          <span class="text-xs text-gray-500 block mb-3 font-medium"
            >会员数据导入（会员来源 CSV）</span
          >
          <NSelect
            v-model:value="importTemplateId"
            :options="dispatchTemplates"
            placeholder="选择会员导入模板"
            class="mb-2"
          />
          <NButton block type="primary" @click="handleImportDispatch">导入会员数据</NButton>
        </div>
        <div v-if="waveMembers.length" class="flex-1 min-h-0 flex flex-col overflow-hidden px-3">
          <!-- TABLE VIEWPORT -->
          <div ref="memberLayoutRef" class="flex-1 min-h-0 flex flex-col overflow-hidden">
            <!-- scroll mode -->
            <div
              v-if="tableMode === 'scroll'"
              ref="memberTableRef"
              class="flex-1 min-h-0 overflow-hidden"
            >
              <NDataTable
                :columns="memberColumns"
                :data="renderMembers"
                :loading="isMembersLoading"
                :remote="true"
                :max-height="memberTableBodyMaxHeight"
                :table-layout="'auto'"
                :bordered="false"
                :pagination="false"
                size="small"
                @update:sorter="(s: any) => applyMemberSorter(s)"
              />
            </div>
            <!-- paginated mode -->
            <template v-if="tableMode === 'paginated'">
              <div ref="memberTableRef" class="shrink-0">
                <NDataTable
                  :columns="memberColumns"
                  :data="renderMembers"
                  :loading="isMembersLoading"
                  :remote="true"
                  :table-layout="'auto'"
                  :bordered="false"
                  :pagination="false"
                  size="small"
                  @update:sorter="(s: any) => applyMemberSorter(s)"
                />
              </div>
              <AdaptivePaginationIndicator
                :page="memberCurrentPage"
                :page-count="memberPageCount"
              />
            </template>
          </div>
          <!-- FOOTER (sibling, outside viewport) -->
          <div
            v-if="tableMode === 'paginated'"
            ref="memberFooterRef"
            class="flex justify-center shrink-0"
            style="padding: 8px 0 12px 0"
          >
            <div style="transform: scale(1.3); transform-origin: top center; display: inline-flex">
              <NPagination
                :page="memberCurrentPage"
                :page-count="memberPageCount"
                size="small"
                @update:page="handleMemberPageChange"
              />
            </div>
          </div>
        </div>
        <div v-else class="flex-1" />
      </div>
    </div>

    <div class="flex justify-end shrink-0 pt-3 pb-1">
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>

    <!-- Measurement layers (off-screen, paginated mode only) -->
    <AdaptiveTableMeasureLayer
      v-if="tableMode === 'paginated' && allProducts.length"
      ref="productMeasureLayer"
      :data="sortedProducts"
      :columns="productMeasureColumns"
      :width="productViewportWidth"
      size="small"
    />
    <AdaptiveTableMeasureLayer
      v-if="tableMode === 'paginated' && waveMembers.length"
      ref="memberMeasureLayer"
      :data="sortedMembers"
      :columns="memberMeasureColumns"
      :width="memberViewportWidth"
      size="small"
    />
  </div>
</template>
