<script setup lang="ts">
import { CloudUploadOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useAdaptiveTable } from '@/shared/composables/useAdaptiveTable'
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
  allProducts.value.forEach((p, i) => map.set(p.id, i + 1))
  return map
})
const memberIndexMap = computed(() => {
  const map = new Map<number, number>()
  waveMembers.value.forEach((m, i) => map.set(m.id, i + 1))
  return map
})
const isProductLoading = ref(false)

// ── product table composable ──
const productTableParent = ref<HTMLElement | null>(null)
const productTableWrapper = ref<HTMLElement | null>(null)
const productPaginationRef = ref<HTMLElement | null>(null)
const productIndicatorRef = ref<HTMLElement | null>(null)

const {
  currentPage: productCurrentPage,
  totalPages: productTotalPages,
  visibleItems: visibleProducts,
  scrollMode,
  lastW: lastProductW,
  indicatorFontSize: productIndicatorFontSize,
  indicatorLeft: productIndicatorLeft,
  indicatorRight: productIndicatorRight,
  handlePageChange: handleProductPageChange,
  remeasure: remeasureProducts,
  setupIndicatorObserver: setupProductIndicatorObserver,
  teardown: teardownProduct,
  init: initProduct,
} = useAdaptiveTable(allProducts, {
  tableParentRef: productTableParent,
  tableWrapperRef: productTableWrapper,
  paginationRef: productPaginationRef,
  indicatorRef: productIndicatorRef,
})

// ── member table composable ──
const memberTableParent = ref<HTMLElement | null>(null)
const memberTableWrapper = ref<HTMLElement | null>(null)
const memberPaginationRef = ref<HTMLElement | null>(null)
const memberIndicatorRef = ref<HTMLElement | null>(null)

const {
  currentPage: memberCurrentPage,
  totalPages: memberTotalPages,
  visibleItems: visibleMembers,
  lastW: lastMemberW,
  indicatorFontSize: memberIndicatorFontSize,
  indicatorLeft: memberIndicatorLeft,
  indicatorRight: memberIndicatorRight,
  handlePageChange: handleMemberPageChange,
  remeasure: remeasureMembers,
  setupIndicatorObserver: setupMemberIndicatorObserver,
  teardown: teardownMember,
  init: initMember,
} = useAdaptiveTable(waveMembers, {
  tableParentRef: memberTableParent,
  tableWrapperRef: memberTableWrapper,
  paginationRef: memberPaginationRef,
  indicatorRef: memberIndicatorRef,
})

// ── column definitions ──
const showSkuColumn = computed(() => lastProductW.value === 0 || lastProductW.value >= 400)
const showMemberExtraColumns = computed(() => lastMemberW.value === 0 || lastMemberW.value >= 450)

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
      minWidth: 90,
      render: (row) => clampedText(row.latestNickname || row.platformUid),
    },
    { title: '平台', key: 'platform', width: 70, render: (row) => clampedText(row.platform) },
    {
      title: 'UID',
      key: 'platformUid',
      minWidth: 90,
      render: (row) => clampedText(row.platformUid),
    },
  ]
  if (showMemberExtraColumns.value) {
    cols.push({
      title: '等级',
      key: 'giftLevel',
      width: 50,
      render: (row) => clampedText(row.giftLevel || '-'),
    })
    cols.push({ title: '地址数', key: 'activeAddressCount', width: 60 })
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
    { title: '商品名', key: 'name', minWidth: 140, render: (row: any) => clampedText(row.name) },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'SKU',
      key: 'factorySku',
      minWidth: 120,
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

// ── lifecycle ──
onMounted(async () => {
  await loadTemplates()
  await loadWaveMembers()
  await loadAllProducts()
  await initProduct()
  await initMember()
})

watch([() => allProducts.value.length, () => waveMembers.value.length], async () => {
  await nextTick()
  await remeasureProducts()
  await remeasureMembers()
})

watch(scrollMode, async (v) => {
  if (!v) {
    await nextTick()
    setupProductIndicatorObserver()
    setupMemberIndicatorObserver()
  }
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
        <div
          v-if="allProducts.length"
          ref="productTableParent"
          class="flex-1 min-h-0 flex flex-col overflow-hidden px-3 pb-3"
        >
          <div
            ref="productTableWrapper"
            :class="scrollMode ? 'overflow-y-auto flex-1 min-h-0 mt-2' : 'overflow-hidden mt-2'"
          >
            <NDataTable
              :columns="productDataColumns"
              :data="visibleProducts"
              :loading="isProductLoading"
              :bordered="false"
              :pagination="false"
              size="small"
            />
          </div>
          <div
            v-if="!scrollMode"
            ref="productIndicatorRef"
            class="flex-1 flex justify-center items-center select-none"
            :style="{
              fontSize: productIndicatorFontSize + 'px',
              lineHeight: 1,
              fontFamily: 'monospace',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              marginBottom: '12px',
            }"
          >
            <span style="color: rgba(96, 165, 250, 0.1)">{{ productIndicatorLeft }}</span
            ><span style="color: rgba(251, 191, 36, 0.1)">{{ productIndicatorRight }}</span>
          </div>
          <div
            v-if="!scrollMode"
            ref="productPaginationRef"
            class="flex justify-center mt-0 mb-3 shrink-0"
            style="transform: scale(1.3); transform-origin: top center"
          >
            <NPagination
              :page="productCurrentPage"
              :page-count="productTotalPages"
              size="small"
              @update:page="handleProductPageChange"
            />
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
        <div
          v-if="waveMembers.length"
          ref="memberTableParent"
          class="flex-1 min-h-0 flex flex-col overflow-hidden px-3 pb-3"
        >
          <div
            ref="memberTableWrapper"
            :class="scrollMode ? 'overflow-y-auto flex-1 min-h-0 mt-2' : 'overflow-hidden mt-2'"
          >
            <NDataTable
              :columns="memberColumns"
              :data="visibleMembers"
              :loading="isMembersLoading"
              :bordered="false"
              :pagination="false"
              size="small"
            />
          </div>
          <div
            v-if="!scrollMode"
            ref="memberIndicatorRef"
            class="flex-1 flex justify-center items-center select-none"
            :style="{
              fontSize: memberIndicatorFontSize + 'px',
              lineHeight: 1,
              fontFamily: 'monospace',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              marginBottom: '12px',
            }"
          >
            <span style="color: rgba(96, 165, 250, 0.1)">{{ memberIndicatorLeft }}</span
            ><span style="color: rgba(251, 191, 36, 0.1)">{{ memberIndicatorRight }}</span>
          </div>
          <div
            v-if="!scrollMode"
            ref="memberPaginationRef"
            class="flex justify-center mt-0 mb-3 shrink-0"
            style="transform: scale(1.3); transform-origin: top center"
          >
            <NPagination
              :page="memberCurrentPage"
              :page-count="memberTotalPages"
              size="small"
              @update:page="handleMemberPageChange"
            />
          </div>
        </div>
        <div v-else class="flex-1" />
      </div>
    </div>

    <div class="flex justify-end shrink-0 pt-3 pb-1">
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>
  </div>
</template>
