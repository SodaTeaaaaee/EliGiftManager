<script setup lang="ts">
import { CloudUploadOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDataTable, NIcon, NPagination, NPopconfirm, NSelect, useMessage, type DataTableColumns } from 'naive-ui'
import { importDispatchWave, importToWave, isWailsRuntimeAvailable, listProductsWithTags, listTemplates, listWaveMembers, pickCSVFile, pickZIPFile, removeMemberFromWave, removeProductFromWave, WAILS_PREVIEW_MESSAGE, type MemberItem, type TemplateItem } from '@/shared/lib/wails/app'

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

// ── page packing: accumulate DOM-measured row heights, break at overflow ──

function packByHeights(heights: number[], availableH: number, headerH: number): Array<{ start: number; end: number }> {
  const pages: Array<{ start: number; end: number }> = []
  if (heights.length === 0) return pages
  const bodyH = availableH - headerH
  if (bodyH <= 0) {
    for (let i = 0; i < heights.length; i++) pages.push({ start: i, end: i })
    return pages
  }
  let pageStart = 0
  let used = 0
  for (let i = 0; i < heights.length; i++) {
    if (used + heights[i] > bodyH && i > pageStart) {
      pages.push({ start: pageStart, end: i - 1 })
      pageStart = i
      used = heights[i]
    } else {
      used += heights[i]
    }
  }
  pages.push({ start: pageStart, end: heights.length - 1 })
  return pages
}

// ── member table: all data client-side ──
const memberTableWrapper = ref<HTMLElement | null>(null)
const memberPaginationRef = ref<HTMLElement | null>(null)
const memberAvailableH = ref(400)
const memberCurrentPage = ref(1)

const memberNeedsMeasure = ref(true)
const memberMeasuredHeights = ref<number[]>([])

const memberPages = computed(() =>
  packByHeights(memberMeasuredHeights.value,
    memberAvailableH.value - memberPaginationH.value * 2, memberHeaderH.value),
)

const memberTotalPages = computed(() => memberPages.value.length || 1)

const visibleMembers = computed(() => {
  if (memberNeedsMeasure.value) return waveMembers.value
  const page = memberPages.value[memberCurrentPage.value - 1]
  if (!page) return waveMembers.value
  return waveMembers.value.slice(page.start, page.end + 1)
})

function handleMemberPageChange(p: number) { memberCurrentPage.value = p }

async function remeasureMembers() {
  memberNeedsMeasure.value = true
  await nextTick()
  const trs = memberTableWrapper.value?.querySelectorAll('tbody tr')
  if (trs && trs.length > 0) {
    memberMeasuredHeights.value = Array.from(trs).map(tr => (tr as HTMLElement).offsetHeight)
  }
  memberNeedsMeasure.value = false
  if (memberCurrentPage.value > memberPages.value.length) memberCurrentPage.value = 1
}

// ── product table: fetch all at once, paginate client-side ──
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
const productTableWrapper = ref<HTMLElement | null>(null)
const productPaginationRef = ref<HTMLElement | null>(null)
const productAvailableH = ref(400)
const productCurrentPage = ref(1)
const isProductLoading = ref(false)

const productNeedsMeasure = ref(true)
const productMeasuredHeights = ref<number[]>([])

const productPages = computed(() =>
  packByHeights(productMeasuredHeights.value,
    productAvailableH.value - productPaginationH.value * 2, productHeaderH.value),
)

const productTotalPages = computed(() => productPages.value.length || 1)

const visibleProducts = computed(() => {
  if (productNeedsMeasure.value) return allProducts.value
  const page = productPages.value[productCurrentPage.value - 1]
  if (!page) return allProducts.value
  return allProducts.value.slice(page.start, page.end + 1)
})

function handleProductPageChange(p: number) { productCurrentPage.value = p }

async function remeasureProducts() {
  productNeedsMeasure.value = true
  await nextTick()
  const trs = productTableWrapper.value?.querySelectorAll('tbody tr')
  if (trs && trs.length > 0) {
    productMeasuredHeights.value = Array.from(trs).map(tr => (tr as HTMLElement).offsetHeight)
  }
  productNeedsMeasure.value = false
  if (productCurrentPage.value > productPages.value.length) productCurrentPage.value = 1
}

// ── ResizeObserver: track wrapper height & width → re-measure row heights → repack ──
let resizeObserver: ResizeObserver | null = null
const lastProductW = ref(0)
const lastMemberW = ref(0)

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
          remeasureProducts()
        }
        if (hChanged) {
          productAvailableH.value = h
          productCurrentPage.value = 1
        }
      } else if (entry.target === memberTableWrapper.value) {
        const w = entry.contentRect.width
        const h = entry.contentRect.height
        if (h <= 0) continue
        const wChanged = w !== lastMemberW.value
        const hChanged = h !== memberAvailableH.value
        if (!wChanged && !hChanged) continue
        if (wChanged) {
          lastMemberW.value = w
          remeasureMembers()
        }
        if (hChanged) {
          memberAvailableH.value = h
          memberCurrentPage.value = 1
        }
      }
    }
  })
  if (productTableWrapper.value) resizeObserver.observe(productTableWrapper.value)
  if (memberTableWrapper.value) resizeObserver.observe(memberTableWrapper.value)
}

// ── column definitions ──
const showSkuColumn = computed(() => lastProductW.value === 0 || lastProductW.value >= 400)
const showMemberExtraColumns = computed(() => lastMemberW.value === 0 || lastMemberW.value >= 450)

const memberColumns = computed<DataTableColumns<MemberItem>>(() => {
  const cols: DataTableColumns<MemberItem> = [
    {
      title: '#', key: '__index', width: 40,
      render: (row: any) => h('span', { style: { color: '#999' } }, String(memberIndexMap.value.get(row.id) ?? ''))
    },
    { title: '昵称', key: 'latestNickname', minWidth: 90, render: (row) => clampedText(row.latestNickname || row.platformUid) },
    { title: '平台', key: 'platform', width: 70, render: (row) => clampedText(row.platform) },
    { title: 'UID', key: 'platformUid', minWidth: 90, render: (row) => clampedText(row.platformUid) },
  ]
  if (showMemberExtraColumns.value) {
    cols.push({
      title: '等级', key: 'extraData', width: 50, render: (row) => clampedText((() => {
        try { const ed = JSON.parse(row.extraData); return ed.giftLevel || '-' }
        catch { return '-' }
      })())
    })
    cols.push({ title: '地址数', key: 'activeAddressCount', width: 60 })
  }
  cols.push({
    title: '操作', key: '__actions', width: 80,
    render(row: any) {
      return h(NPopconfirm, {
        onPositiveClick: () => handleDeleteMember(row.id),
        negativeText: '取消', positiveText: '确认',
      }, {
        trigger: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
        default: () => '确认从任务中移除此会员？',
      })
    },
  })
  return cols
})

const productDataColumns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    {
      title: '#', key: '__index', width: 40,
      render: (row: any) => h('span', { style: { color: '#999' } }, String(productIndexMap.value.get(row.id) ?? ''))
    },
    { title: '商品名', key: 'name', minWidth: 140, render: (row: any) => clampedText(row.name) },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'SKU', key: 'factorySku', minWidth: 120,
      render: (row: any) => h('span', { style: { whiteSpace: 'nowrap' } }, String(row.factorySku ?? '')),
    })
  }
  cols.push({
    title: '操作', key: '__actions', width: 80,
    render(row: any) {
      return h(NPopconfirm, {
        onPositiveClick: () => handleDeleteProduct(row.id),
        negativeText: '取消', positiveText: '确认',
      }, {
        trigger: () => h(NButton, { size: 'tiny', type: 'error' }, { default: () => '删除' }),
        default: () => '确认从任务中移除此商品？',
      })
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
    allProducts.value = result.items.map(item => ({ id: item.id, name: item.name, factorySku: item.factorySku || '' }))
  } catch (e) { console.error('加载任务商品失败', e) }
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
  catch (e) { console.error('加载任务会员失败', e) }
  finally { isMembersLoading.value = false }
}

async function handleImportProduct() {
  if (!productTemplateId.value) return message.warning('请先选择商品导入模板')
  const fmt = templateFormat(productTemplateId.value)
  const filePath = fmt === 'zip' ? await pickZIPFile() : await pickCSVFile()
  if (!filePath) return
  try { await importToWave(waveId.value, filePath, productTemplateId.value); message.success('商品导入完成'); await loadAllProducts(); await loadWaveMembers() }
  catch (e) { message.error(String(e)) }
}

async function handleDeleteProduct(productId: number) {
  try { await removeProductFromWave(waveId.value, productId); message.success('已从任务中移除'); await loadAllProducts(); await loadWaveMembers() }
  catch (e) { message.error(String(e)) }
}

async function handleDeleteMember(memberId: number) {
  try { await removeMemberFromWave(waveId.value, memberId); message.success('已从任务中移除'); await loadWaveMembers() }
  catch (e) { message.error(String(e)) }
}

async function handleImportDispatch() {
  if (!importTemplateId.value) return message.warning('请先选择会员导入模板')
  const filePath = await pickCSVFile()
  if (!filePath) return
  try { await importDispatchWave(waveId.value, filePath, importTemplateId.value); message.success('会员数据导入完成'); await loadWaveMembers() }
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
  // 测量 header/pagination
  productHeaderH.value = measureHeaderHeight(productTableWrapper.value)
  memberHeaderH.value = measureHeaderHeight(memberTableWrapper.value)
  productPaginationH.value = measurePaginationHeight(productPaginationRef.value)
  memberPaginationH.value = measurePaginationHeight(memberPaginationRef.value)
  // 初始 DOM 高度
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
  // DOM 实测行高 + 打包
  await remeasureProducts()
  await remeasureMembers()
  setupResizeObserver()
})

watch([() => allProducts.value.length, () => waveMembers.value.length], async () => {
  await nextTick()
  productHeaderH.value = measureHeaderHeight(productTableWrapper.value)
  memberHeaderH.value = measureHeaderHeight(memberTableWrapper.value)
  productPaginationH.value = measurePaginationHeight(productPaginationRef.value)
  memberPaginationH.value = measurePaginationHeight(memberPaginationRef.value)
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
  await remeasureProducts()
  await remeasureMembers()
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
          <span class="text-xs text-gray-500 block mb-3 font-medium">会员数据导入（会员来源 CSV）</span>
          <NSelect v-model:value="importTemplateId" :options="dispatchTemplates" placeholder="选择会员导入模板" class="mb-2" />
          <NButton block type="primary" @click="handleImportDispatch">导入会员数据</NButton>
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
