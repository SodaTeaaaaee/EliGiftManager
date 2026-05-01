<script setup lang="ts">
import { CloudUploadOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDataTable, NIcon, NPagination, NSelect, NFlex, useMessage, type DataTableColumns } from 'naive-ui'
import { importDispatchWave, importToWave, isWailsRuntimeAvailable, listProductsWithTags, listTemplates, listWaveMembers, pickCSVFile, pickZIPFile, WAILS_PREVIEW_MESSAGE, type MemberItem, type TemplateItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const templates = ref<TemplateItem[]>([])
const csvPath = ref('')
const productCsvPath = ref('')
const importTemplateId = ref<number | null>(null)
const productTemplateId = ref<number | null>(null)
const waveMembers = ref<MemberItem[]>([])
const isMembersLoading = ref(false)
const errorMessage = ref('')

const importTemplates = computed(() => templates.value.filter(t => t.type.startsWith('import_')).map(toOption))
const productTemplates = computed(() => templates.value.filter(t => t.type === 'import_product').map(toOption))
const dispatchTemplates = computed(() => templates.value.filter(t => t.type === 'import_dispatch_record').map(toOption))

function toOption(template: TemplateItem) {
  return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id }
}

function templateFormat(templateId: number | null): string {
  if (!templateId) return 'csv'
  const t = templates.value.find(t => t.id === templateId)
  if (!t) return 'csv'
  try { const rules = JSON.parse(t.mappingRules); return rules.format || 'csv' }
  catch { return 'csv' }
}

const productFileExt = computed(() => templateFormat(productTemplateId.value) === 'zip' ? 'ZIP' : 'CSV')

// ── line-clamped cell renderer ──
const MAX_LINES = 4
const LINE_HEIGHT = 21
const CELL_PAD_V = 16

function clampedText(text: string, lines = MAX_LINES) {
  return h('div', {
    style: {
      display: '-webkit-box',
      '-webkit-line-clamp': String(lines),
      '-webkit-box-orient': 'vertical',
      overflow: 'hidden',
      wordBreak: 'break-all',
      lineHeight: String(LINE_HEIGHT) + 'px',
    },
  }, String(text ?? ''))
}

// ── per-row height estimation (column widths measured from DOM) ──
function estimateCellLines(text: string, colWidth: number): number {
  const avgCharW = 12
  const charsPerLine = Math.max(1, Math.floor(colWidth / avgCharW))
  return Math.min(MAX_LINES, Math.max(1, Math.ceil(text.length / charsPerLine)))
}

function estimateRowHeight(row: Record<string, any>, keys: string[], widths: number[]): number {
  let maxLines = 1
  for (let i = 0; i < keys.length; i++) {
    const colW = widths[i] ?? 100
    const text = String(row[keys[i]] ?? '')
    const lines = estimateCellLines(text, colW)
    if (lines > maxLines) maxLines = lines
  }
  return maxLines * LINE_HEIGHT + CELL_PAD_V
}

// Column keys for estimation & DOM measurement (must match column `key` in definitions)
const MEMBER_EST_KEYS = ['latestNickname', 'platform', 'platformUid', 'extraData']
const PRODUCT_EST_KEYS = ['name'] // only name column wraps; SKU is nowrap → always 1 line

// Measured column widths — updated from DOM after render & on resize
const memberColWidths = ref<number[]>(MEMBER_EST_KEYS.map(() => 100))
const productColWidths = ref<number[]>(PRODUCT_EST_KEYS.map(() => 100))

function measureColWidths(wrapper: HTMLElement | null, keys: string[], fallback: number[]): number[] {
  if (!wrapper) return fallback
  const ths = wrapper.querySelectorAll('.n-data-table-th')
  if (ths.length === 0) return fallback
  const result: number[] = []
  for (let i = 0; i < keys.length; i++) {
    const el = wrapper.querySelector(`[data-col-key="${keys[i]}"]`)
    result.push(el instanceof HTMLElement ? el.offsetWidth : (fallback[i] ?? 100))
  }
  return result
}

function memberEstimateRow(m: MemberItem): Record<string, any> {
  let giftLevel = '-'
  try { giftLevel = JSON.parse(m.extraData).giftLevel || '-' } catch { /* ignore */ }
  return {
    latestNickname: m.latestNickname || m.platformUid,
    platform: m.platform,
    platformUid: m.platformUid,
    extraData: giftLevel,
  }
}

function productEstimateRow(p: { name: string; factorySku: string }): Record<string, any> {
  return { name: p.name }
}

// ── measured header & pagination heights (DOM, updated on resize) ──
const productHeaderH = ref(38)
const memberHeaderH = ref(38)
const productPaginationH = ref(32)
const memberPaginationH = ref(32)

function measureHeaderHeight(wrapper: HTMLElement | null): number {
  if (!wrapper) return 38
  const thead = wrapper.querySelector('.n-data-table-thead')
  return thead instanceof HTMLElement ? thead.offsetHeight : 40
}

function measurePaginationHeight(el: HTMLElement | null): number {
  return el ? el.offsetHeight : 32
}

// ── page packing: accumulate row heights, break at overflow (no clipping tolerated) ──

function packRows<T>(
  rows: T[],
  extract: (r: T) => Record<string, any>,
  keys: string[],
  widths: number[],
  availableH: number,
  headerH: number,
): Array<{ start: number; end: number }> {
  const pages: Array<{ start: number; end: number }> = []
  if (rows.length === 0) return pages
  const bodyH = availableH - headerH
  if (bodyH <= 0) {
    for (let i = 0; i < rows.length; i++) pages.push({ start: i, end: i })
    return pages
  }
  let pageStart = 0
  let used = 0
  for (let i = 0; i < rows.length; i++) {
    const h = estimateRowHeight(extract(rows[i]), keys, widths)
    if (used + h > bodyH && i > pageStart) {
      pages.push({ start: pageStart, end: i - 1 })
      pageStart = i
      used = h
    } else {
      used += h
    }
  }
  pages.push({ start: pageStart, end: rows.length - 1 })
  return pages
}

// ── member table: all data client-side ──
const memberTableWrapper = ref<HTMLElement | null>(null)
const memberPaginationRef = ref<HTMLElement | null>(null)
const memberAvailableH = ref(400)
const memberCurrentPage = ref(1)

const memberPages = computed(() =>
  packRows(waveMembers.value, memberEstimateRow, MEMBER_EST_KEYS, memberColWidths.value,
    memberAvailableH.value - memberPaginationH.value * 2, memberHeaderH.value),
)

const memberTotalPages = computed(() => memberPages.value.length || 1)

const visibleMembers = computed(() => {
  const page = memberPages.value[memberCurrentPage.value - 1]
  if (!page) return waveMembers.value
  return waveMembers.value.slice(page.start, page.end + 1)
})

function handleMemberPageChange(p: number) { memberCurrentPage.value = p }

// ── product table: fetch all at once, paginate client-side ──
const allProducts = ref<{ id: number; name: string; factorySku: string }[]>([])
const productTableWrapper = ref<HTMLElement | null>(null)
const productPaginationRef = ref<HTMLElement | null>(null)
const productAvailableH = ref(400)
const productCurrentPage = ref(1)
const isProductLoading = ref(false)

const productPages = computed(() =>
  packRows(allProducts.value, productEstimateRow, PRODUCT_EST_KEYS, productColWidths.value,
    productAvailableH.value - productPaginationH.value * 2, productHeaderH.value),
)

const productTotalPages = computed(() => productPages.value.length || 1)

const visibleProducts = computed(() => {
  const page = productPages.value[productCurrentPage.value - 1]
  if (!page) return allProducts.value
  return allProducts.value.slice(page.start, page.end + 1)
})

function handleProductPageChange(p: number) { productCurrentPage.value = p }

// ── ResizeObserver: track wrapper height & width → re-measure column widths → repack ──
let resizeObserver: ResizeObserver | null = null
const lastProductW = ref(0)
const lastMemberW = ref(0)

function syncAllColWidths() {
  productColWidths.value = measureColWidths(productTableWrapper.value, PRODUCT_EST_KEYS, productColWidths.value)
  memberColWidths.value = measureColWidths(memberTableWrapper.value, MEMBER_EST_KEYS, memberColWidths.value)
  productHeaderH.value = measureHeaderHeight(productTableWrapper.value)
  memberHeaderH.value = measureHeaderHeight(memberTableWrapper.value)
  productPaginationH.value = measurePaginationHeight(productPaginationRef.value)
  memberPaginationH.value = measurePaginationHeight(memberPaginationRef.value)
}

function setupResizeObserver() {
  resizeObserver = new ResizeObserver(entries => {
    for (const entry of entries) {
      if (entry.target === productTableWrapper.value) {
        const w = entry.contentRect.width
        const h = entry.contentRect.height
        if (h <= 0) continue
        const wChanged = w !== lastProductW.value
        const hChanged = h !== productAvailableH.value
        if (!wChanged && !hChanged) continue
        if (wChanged) {
          lastProductW.value = w
          productColWidths.value = measureColWidths(productTableWrapper.value, PRODUCT_EST_KEYS, productColWidths.value)
        }
        if (hChanged) productAvailableH.value = h
        productCurrentPage.value = 1
      } else if (entry.target === memberTableWrapper.value) {
        const w = entry.contentRect.width
        const h = entry.contentRect.height
        if (h <= 0) continue
        const wChanged = w !== lastMemberW.value
        const hChanged = h !== memberAvailableH.value
        if (!wChanged && !hChanged) continue
        if (wChanged) {
          lastMemberW.value = w
          memberColWidths.value = measureColWidths(memberTableWrapper.value, MEMBER_EST_KEYS, memberColWidths.value)
        }
        if (hChanged) memberAvailableH.value = h
        memberCurrentPage.value = 1
      }
    }
  })
  if (productTableWrapper.value) resizeObserver.observe(productTableWrapper.value)
  if (memberTableWrapper.value) resizeObserver.observe(memberTableWrapper.value)
}

// ── column definitions ──
const memberColumns: DataTableColumns<MemberItem> = [
  { title: '昵称', key: 'latestNickname', minWidth: 100, render: (row) => clampedText(row.latestNickname || row.platformUid) },
  { title: '平台', key: 'platform', width: 100, render: (row) => clampedText(row.platform) },
  { title: 'UID', key: 'platformUid', minWidth: 100, render: (row) => clampedText(row.platformUid) },
  {
    title: '等级', key: 'extraData', width: 100, render: (row) => clampedText((() => {
      try { const ed = JSON.parse(row.extraData); return ed.giftLevel || '-' }
      catch { return '-' }
    })())
  },
  { title: '地址数', key: 'activeAddressCount', width: 70 },
]

// Hide SKU column when wrapper too narrow for both columns at their min-widths
const showSkuColumn = computed(() => lastProductW.value === 0 || lastProductW.value >= 260)

const productDataColumns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    { title: '商品名', key: 'name', minWidth: 140, render: (row: any) => clampedText(row.name) },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'SKU', key: 'factorySku', minWidth: 120,
      render: (row: any) => h('span', { style: { whiteSpace: 'nowrap' } }, String(row.factorySku ?? '')),
    })
  }
  return cols
})

// ── data loading ──
async function loadAllProducts() {
  if (!waveId.value) return
  isProductLoading.value = true
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 10000)
    allProducts.value = result.items.map(item => ({ id: item.id, name: item.name, factorySku: item.factorySku || '' }))
  } catch (e) { console.error('加载波次商品失败', e) }
  finally { isProductLoading.value = false }
}

async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false }
  return true
}

async function loadTemplates() {
  if (!(await guardRuntime())) return
  try { templates.value = await listTemplates() }
  catch (e) { console.error('加载模板失败', e) }
}

async function loadWaveMembers() {
  if (!waveId.value) return
  isMembersLoading.value = true
  try { waveMembers.value = await listWaveMembers(waveId.value) }
  catch (e) { console.error('加载波次会员失败', e) }
  finally { isMembersLoading.value = false }
}

async function handlePickCSV() {
  const p = await pickCSVFile()
  if (p) csvPath.value = p
}

async function handlePickProductFile() {
  const fmt = templateFormat(productTemplateId.value)
  const p = fmt === 'zip' ? await pickZIPFile() : await pickCSVFile()
  if (p) productCsvPath.value = p
}

async function handleImportProduct() {
  if (!waveId.value || !productCsvPath.value || !productTemplateId.value) return message.warning('请选择商品 CSV 文件和导入模板')
  try { await importToWave(waveId.value, productCsvPath.value, productTemplateId.value); message.success('商品导入完成'); await loadAllProducts(); await loadWaveMembers() }
  catch (e) { message.error(String(e)) }
}

async function handleImportDispatch() {
  if (!waveId.value || !csvPath.value || !importTemplateId.value) return message.warning('请选择发货任务、填写 CSV 路径、并选择导入模板')
  try { await importDispatchWave(waveId.value, csvPath.value, importTemplateId.value); message.success('发货数据导入完成'); await loadWaveMembers() }
  catch (e) { message.error(String(e)) }
}

function goNext() {
  router.push({ name: 'waves-step-tags', params: { waveId: String(waveId.value) } })
}

// ── lifecycle ──
onMounted(async () => {
  await loadTemplates()
  await loadWaveMembers()
  await loadAllProducts()
  await nextTick()
  syncAllColWidths()
  if (productTableWrapper.value) {
    const h = productTableWrapper.value.clientHeight
    if (h > 0) productAvailableH.value = h
    lastProductW.value = productTableWrapper.value.clientWidth
  }
  if (memberTableWrapper.value) {
    const h = memberTableWrapper.value.clientHeight
    if (h > 0) memberAvailableH.value = h
    lastMemberW.value = memberTableWrapper.value.clientWidth
  }
  setupResizeObserver()
})

watch([() => allProducts.value.length, () => waveMembers.value.length], async () => {
  await nextTick()
  syncAllColWidths()
  if (productTableWrapper.value) {
    resizeObserver?.observe(productTableWrapper.value)
    const h = productTableWrapper.value.clientHeight
    if (h > 0) productAvailableH.value = h
  }
  if (memberTableWrapper.value) {
    resizeObserver?.observe(memberTableWrapper.value)
    const h = memberTableWrapper.value.clientHeight
    if (h > 0) memberAvailableH.value = h
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
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
          <span class="text-xs text-gray-500 block mb-3 font-medium">商品导入（工厂平台 {{ productFileExt }}）</span>
          <NFlex :wrap="false" class="mb-2">
            <NButton v-if="productTemplateId" size="small" secondary @click="handlePickProductFile">选择 {{ productFileExt
            }}</NButton>
            <span class="text-xs text-gray-400 self-center truncate max-w-[200px]">{{ productCsvPath || '未选择文件'
            }}</span>
          </NFlex>
          <NSelect v-model:value="productTemplateId" :options="productTemplates" placeholder="选择商品导入模板" class="mb-2" />
          <NButton block secondary @click="handleImportProduct">导入商品</NButton>
        </div>
        <div v-if="allProducts.length" class="flex-1 min-h-0 flex flex-col overflow-hidden px-3 pb-3">
          <div ref="productTableWrapper" class="flex-1 min-h-0 overflow-hidden mt-2">
            <NDataTable :columns="productDataColumns" :data="visibleProducts" :loading="isProductLoading"
              :bordered="false" :pagination="false" size="small" />
          </div>
          <div ref="productPaginationRef" class="flex justify-center mt-2 shrink-0">
            <NPagination :page="productCurrentPage" :page-count="productTotalPages" size="small"
              @update:page="handleProductPageChange" />
          </div>
        </div>
        <div v-else class="flex-1" />
      </div>

      <!-- 会员导入面板 -->
      <div class="border border-gray-100 dark:border-gray-700 rounded-lg flex flex-col min-h-0">
        <div class="p-3 pb-0 shrink-0">
          <span class="text-xs text-gray-500 block mb-3 font-medium">发货数据导入（会员来源 CSV）</span>
          <NFlex :wrap="false" class="mb-2">
            <NButton v-if="importTemplateId" size="small" secondary @click="handlePickCSV">选择 CSV</NButton>
            <span class="text-xs text-gray-400 self-center truncate max-w-[200px]">{{ csvPath || '未选择文件' }}</span>
          </NFlex>
          <NSelect v-model:value="importTemplateId" :options="dispatchTemplates" placeholder="选择发货导入模板" class="mb-2" />
          <NButton block type="primary" @click="handleImportDispatch">导入发货数据</NButton>
        </div>
        <div v-if="waveMembers.length" class="flex-1 min-h-0 flex flex-col overflow-hidden px-3 pb-3">
          <div ref="memberTableWrapper" class="flex-1 min-h-0 overflow-hidden mt-2">
            <NDataTable :columns="memberColumns" :data="visibleMembers" :loading="isMembersLoading" :bordered="false"
              :pagination="false" size="small" />
          </div>
          <div ref="memberPaginationRef" class="flex justify-center mt-2 shrink-0">
            <NPagination :page="memberCurrentPage" :page-count="memberTotalPages" size="small"
              @update:page="handleMemberPageChange" />
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
