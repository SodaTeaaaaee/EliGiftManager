<script setup lang="ts">
import { PlayOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDataTable, NDrawer, NDrawerContent, NDivider, NEmpty, NFlex, NIcon, NPagination, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { allocateSingleTag, assignProductTag, getProductImages, isWailsRuntimeAvailable, listProductsWithTags, listWaves, removeProductTag, removeSingleTag, WAILS_PREVIEW_MESSAGE, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const wave = ref<WaveItem | null>(null)
const allTagProducts = ref<{ id: number; name: string; factorySku: string; platform: string; tags: string[]; coverImage: string }[]>([])
const checkedProductIds = ref<number[]>([])
const selectedBatchTag = ref<string | null>(null)
const isTagLoading = ref(false)
const errorMessage = ref('')

const showProductDrawer = ref(false)
const drawerProduct = ref<{ id: number; name: string; factorySku: string; platform: string; tags: string[]; coverImage: string } | null>(null)
const drawerProductImages = ref<{ id: number; path: string; sortOrder: number; sourceDir: string }[]>([])

type LevelTag = { platform: string; tagName: string }
const waveLevelTags = computed<LevelTag[]>(() => {
  if (!wave.value?.levelTags) return []
  try { return JSON.parse(wave.value.levelTags) as LevelTag[] }
  catch { return [] }
})

const batchTagOptions = computed(() =>
  waveLevelTags.value.map(t => ({ label: `${t.platform}·${t.tagName}`, value: `${t.platform}|${t.tagName}` }))
)

function platformTagColor(platform: string) {
  const colors: Record<string, { color: string; textColor: string }> = {
    BILIBILI: { color: '#00A1D633', textColor: '#00A1D6' },
    DOUYIN: { color: '#FE2C5533', textColor: '#FE2C55' },
  }
  return colors[platform] || { color: undefined, textColor: undefined }
}

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
const tagHeaderH = ref(38)
const tagPaginationH = ref(32)

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

// ── tag table: fetch all at once, paginate client-side ──
const tagTableWrapper = ref<HTMLElement | null>(null)
const tagPaginationRef = ref<HTMLElement | null>(null)
const tagAvailableH = ref(400)
const tagCurrentPage = ref(1)

const tagNeedsMeasure = ref(true)
const tagMeasuredHeights = ref<number[]>([])

const tagPages = computed(() =>
  packByHeights(tagMeasuredHeights.value,
    tagAvailableH.value - tagPaginationH.value * 2, tagHeaderH.value),
)

const tagTotalPages = computed(() => tagPages.value.length || 1)

const visibleTagProducts = computed(() => {
  if (tagNeedsMeasure.value) return allTagProducts.value
  const page = tagPages.value[tagCurrentPage.value - 1]
  if (!page) return allTagProducts.value
  return allTagProducts.value.slice(page.start, page.end + 1)
})

async function remeasureTags() {
  tagNeedsMeasure.value = true
  await nextTick()
  const trs = tagTableWrapper.value?.querySelectorAll('tbody tr')
  if (trs && trs.length > 0) {
    tagMeasuredHeights.value = Array.from(trs).map(tr => (tr as HTMLElement).offsetHeight)
  }
  tagNeedsMeasure.value = false
  if (tagCurrentPage.value > tagPages.value.length) tagCurrentPage.value = 1
}

function handleTagPageChange(p: number) { tagCurrentPage.value = p; lastClickedRowIndex.value = -1 }

const lastClickedRowIndex = ref(-1)

const anchorId = computed(() => {
  if (lastClickedRowIndex.value < 0) return null
  return visibleTagProducts.value[lastClickedRowIndex.value]?.id ?? null
})

function handleRowClick(row: any, event: MouseEvent) {
  const el = event.target as HTMLElement

  // 1. 复选框点击 → 设锚点（不弹 drawer）
  if (el.closest('.n-checkbox')) {
    const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
    if (idx >= 0) lastClickedRowIndex.value = idx
    return
  }

  // 2. Ctrl+Shift + Click → 加性范围选中（保留范围外旧选择，不弹 drawer）
  if ((event.ctrlKey || event.metaKey) && event.shiftKey) {
    const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
    if (lastClickedRowIndex.value >= 0 && idx >= 0) {
      const lo = Math.min(lastClickedRowIndex.value, idx)
      const hi = Math.max(lastClickedRowIndex.value, idx)
      const rangeIds = visibleTagProducts.value.slice(lo, hi + 1).map((p: any) => p.id)
      checkedProductIds.value = [...new Set([...checkedProductIds.value, ...rangeIds])]
    }
    return
  }

  // 3. Ctrl/Cmd + Click → toggle 单行选中，设锚点（不弹 drawer）
  if (event.ctrlKey || event.metaKey) {
    const id = row.id
    const idx = visibleTagProducts.value.findIndex((p: any) => p.id === id)
    if (idx >= 0) lastClickedRowIndex.value = idx
    if (checkedProductIds.value.includes(id)) {
      checkedProductIds.value = checkedProductIds.value.filter(x => x !== id)
    } else {
      checkedProductIds.value = [...checkedProductIds.value, id]
    }
    return
  }

  // 4. Shift + Click → 替换式范围选中（清掉范围外旧选择，不弹 drawer）
  if (event.shiftKey) {
    const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
    if (lastClickedRowIndex.value >= 0 && idx >= 0) {
      const lo = Math.min(lastClickedRowIndex.value, idx)
      const hi = Math.max(lastClickedRowIndex.value, idx)
      checkedProductIds.value = visibleTagProducts.value.slice(lo, hi + 1).map((p: any) => p.id)
    }
    return
  }

  // 5. 普通点击 → 开 drawer，设锚点
  const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
  if (idx >= 0) lastClickedRowIndex.value = idx
  openProductDrawer(row)
}

function rowProps(row: any) {
  const selected = checkedProductIds.value.includes(row.id)
  const isAnchor = anchorId.value === row.id
  const cls: string[] = []
  if (selected) cls.push('row-selected')
  if (isAnchor) cls.push('row-anchor')
  return { class: cls.join(' '), style: { cursor: 'pointer' }, onClick: (e: MouseEvent) => handleRowClick(row, e) }
}

// ── selection buttons ──
const allSelected = computed(() => allTagProducts.value.length > 0 && checkedProductIds.value.length === allTagProducts.value.length)
const pageAllSelected = computed(() => {
  if (visibleTagProducts.value.length === 0) return false
  return visibleTagProducts.value.every(p => checkedProductIds.value.includes(p.id))
})

function handleSelectAll() {
  if (allSelected.value) { checkedProductIds.value = [] }
  else { checkedProductIds.value = allTagProducts.value.map(p => p.id) }
}
function handleSelectPage() {
  const pageIds = visibleTagProducts.value.map(p => p.id)
  if (pageAllSelected.value) {
    checkedProductIds.value = checkedProductIds.value.filter(id => !pageIds.includes(id))
  } else {
    checkedProductIds.value = [...new Set([...checkedProductIds.value, ...pageIds])]
  }
}
function handleInvertPage() {
  const pageIds = new Set(visibleTagProducts.value.map(p => p.id))
  const toAdd = visibleTagProducts.value.filter(p => !checkedProductIds.value.includes(p.id)).map(p => p.id)
  checkedProductIds.value = checkedProductIds.value.filter(id => !pageIds.has(id)).concat(toAdd)
}

// ── ResizeObserver: track wrapper height & width → remeasure row heights → repack ──
let resizeObserver: ResizeObserver | null = null
const lastTagW = ref(0)

function setupResizeObserver() {
  resizeObserver = new ResizeObserver(entries => {
    for (const entry of entries) {
      if (entry.target === tagTableWrapper.value) {
        const w = entry.contentRect.width
        const h = entry.contentRect.height
        if (h <= 0) continue
        const wChanged = w !== lastTagW.value
        const hChanged = h !== tagAvailableH.value
        if (!wChanged && !hChanged) continue
        if (wChanged) {
          lastTagW.value = w
          remeasureTags()
        }
        if (hChanged) { tagAvailableH.value = h; tagCurrentPage.value = 1 }
      }
    }
  })
  if (tagTableWrapper.value) resizeObserver.observe(tagTableWrapper.value)
}

// ── column definitions ──
const showSkuColumn = computed(() => lastTagW.value === 0 || lastTagW.value >= 400)

const productIndexMap = computed(() => {
  const map = new Map<number, number>()
  allTagProducts.value.forEach((p, i) => map.set(p.id, i + 1))
  return map
})

const tagColumns = computed<DataTableColumns>(() => {
  const cols: DataTableColumns = [
    { type: 'selection' as const },
    {
      title: '#', key: '__index', width: 50,
      render: (row: any) => h('span', { style: { color: '#999' } }, String(productIndexMap.value.get(row.id) ?? '')),
    },
    {
      title: '', key: 'coverImage', width: 56,
      render: (row: any) =>
        row.coverImage ? h('img', { src: '/local-images/' + row.coverImage, class: 'w-10 h-10 rounded object-cover' }) : h('div', { class: 'w-10 h-10 rounded bg-gray-100' }),
    },
    {
      title: '商品名', key: 'name', minWidth: 120,
      render: (row: any) => clampedText(row.name),
    },
  ]
  if (showSkuColumn.value) {
    cols.push({
      title: 'FactorySKU', key: 'factorySku', minWidth: 100,
      render: (row: any) => h('span', { style: { whiteSpace: 'nowrap' } }, String(row.factorySku ?? '')),
    })
  }
  cols.push({
    title: 'Tags', key: 'tags', minWidth: 140,
    render: (row) => h(NFlex, { size: 'small', wrap: true }, {
      default: () => (row.tags as string[]).map((tag: string) => h(NTag, {
        size: 'small', round: true, closable: true,
        onClose: () => handleRemoveTag(row.id as number, row.platform as string, tag),
      }, { default: () => tag })),
    }),
  })
  return cols
})

// ── data loading ──
async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false }
  return true
}

async function loadWave() {
  if (!(await guardRuntime())) return
  try {
    const waves = await listWaves()
    wave.value = waves.find(w => w.id === waveId.value) ?? null
  } catch (e) { console.error('加载波次失败', e) }
}

async function loadTagProducts() {
  if (!waveId.value) return
  isTagLoading.value = true
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 10000)
    allTagProducts.value = result.items.map(item => ({
      id: item.id, name: item.name, factorySku: item.factorySku,
      platform: item.platform, tags: item.tags, coverImage: (item as any).coverImage || '',
    }))
  } catch (e) { console.error('加载商品标签失败', e) }
  finally { isTagLoading.value = false }
}

async function handleAssignTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await assignProductTag(productId, platform, tagName) }
    const count = await allocateSingleTag(waveId.value, platform, tagName)
    message.success(`已为 ${checkedProductIds.value.length} 件商品打上 ${platform}·${tagName} 标签，分配 ${count} 条记录`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleBatchRemoveTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await removeProductTag(productId, platform, tagName) }
    await removeSingleTag(waveId.value, platform, tagName)
    message.success(`已为 ${checkedProductIds.value.length} 件商品移除 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleRemoveTag(productId: number, platform: string, tagName: string) {
  try {
    await removeProductTag(productId, platform, tagName)
    await removeSingleTag(waveId.value, platform, tagName)
    await loadTagProducts()
  }
  catch (e) { message.error(String(e)) }
}

async function openProductDrawer(product: typeof allTagProducts.value[0]) {
  drawerProduct.value = product
  showProductDrawer.value = true
  try { drawerProductImages.value = await getProductImages(product.id) }
  catch { drawerProductImages.value = [] }
}

function goPrev() {
  router.push({ name: 'waves-step-import', params: { waveId: String(waveId.value) } })
}
function goNext() {
  router.push({ name: 'waves-step-preview', params: { waveId: String(waveId.value) } })
}

// ── lifecycle ──
onMounted(async () => {
  await loadWave()
  await loadTagProducts()
  await nextTick()
  tagHeaderH.value = measureHeaderHeight(tagTableWrapper.value)
  tagPaginationH.value = measurePaginationHeight(tagPaginationRef.value)
  if (tagTableWrapper.value) {
    const h = tagTableWrapper.value.clientHeight
    if (h > 0) tagAvailableH.value = h
    lastTagW.value = tagTableWrapper.value.clientWidth
  }
  await remeasureTags()
  setupResizeObserver()
})

watch([() => allTagProducts.value.length], async () => {
  await nextTick()
  tagHeaderH.value = measureHeaderHeight(tagTableWrapper.value)
  tagPaginationH.value = measurePaginationHeight(tagPaginationRef.value)
  if (tagTableWrapper.value) {
    resizeObserver?.observe(tagTableWrapper.value)
    const h = tagTableWrapper.value.clientHeight
    if (h > 0) tagAvailableH.value = h
  }
  await remeasureTags()
})

onUnmounted(() => {
  resizeObserver?.disconnect()
})
</script>
<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center gap-2 shrink-0 px-1 py-2">
      <NIcon size="18">
        <PlayOutline />
      </NIcon>
      <span class="font-semibold text-sm">步骤二：Tag 管理与分配</span>
    </div>

    <div class="shrink-0 px-1 space-y-3">
      <div v-if="waveLevelTags.length > 0">
        <span class="text-xs text-gray-500 block mb-2">可选 Tag：</span>
        <NFlex :size="'small'" :wrap="true">
          <NTag v-for="tag in waveLevelTags" :key="`${tag.platform}|${tag.tagName}`" size="small" round
            :color="platformTagColor(tag.platform)">{{ tag.platform }}·{{ tag.tagName }}</NTag>
        </NFlex>
      </div>
      <NEmpty v-else description="当前波次无等级 Tag，导入会员数据后将自动提取" size="small" />

      <NFlex :size="'small'" :wrap="true" class="items-center">
        <span class="text-xs shrink-0">选择：</span>
        <NButton size="small" secondary @click="handleSelectAll">{{ allSelected ? '取消全选' : '全选所有' }}</NButton>
        <NButton size="small" secondary @click="handleSelectPage">{{ pageAllSelected ? '取消本页' : '本页全选' }}</NButton>
        <NButton size="small" secondary @click="handleInvertPage">本页反选</NButton>
        <NTag size="small" round :bordered="false">已选 {{ checkedProductIds.length }} / {{ allTagProducts.length }}</NTag>
      </NFlex>
      <NFlex :size="'small'" :wrap="true" class="items-center">
        <span class="text-xs shrink-0">批量操作：</span>
        <NSelect v-model:value="selectedBatchTag" :options="batchTagOptions" placeholder="勾选 tag" size="small"
          style="width: 180px" clearable />
        <NButton size="medium" type="primary" @click="handleAssignTag"
          :disabled="!selectedBatchTag || checkedProductIds.length === 0">打标</NButton>
        <NButton size="medium" type="warning" @click="handleBatchRemoveTag"
          :disabled="!selectedBatchTag || checkedProductIds.length === 0">取消打标</NButton>
      </NFlex>
    </div>

    <div class="flex-1 min-h-0 flex flex-col overflow-hidden px-1">
      <div ref="tagTableWrapper" class="flex-1 min-h-0 overflow-hidden">
        <NDataTable :columns="tagColumns" :data="visibleTagProducts" :loading="isTagLoading" :bordered="false"
          :row-key="(row: any) => row.id" v-model:checked-row-keys="checkedProductIds"
          :pagination="false" size="small"
          :row-props="rowProps" />
      </div>
      <div ref="tagPaginationRef" class="flex justify-center mt-2 shrink-0">
        <NPagination :page="tagCurrentPage" :page-count="tagTotalPages" size="small"
          @update:page="handleTagPageChange" />
      </div>
    </div>

    <div class="flex justify-between shrink-0 pt-3 pb-1 px-1 border-t border-gray-100 dark:border-gray-700">
      <NButton @click="goPrev">上一步</NButton>
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>

    <!-- Product Detail Drawer -->
    <NDrawer :show="showProductDrawer" :width="560" @update:show="(v: boolean) => { if (!v) { showProductDrawer = false; drawerProduct = null; drawerProductImages = [] } }">
      <NDrawerContent title="商品详情" closable>
        <template v-if="drawerProduct">
          <div class="space-y-3">
            <img v-if="drawerProduct.coverImage" :src="'/local-images/' + drawerProduct.coverImage" class="w-full rounded-lg object-contain" style="max-height:50vh" />
            <h2 class="text-xl font-semibold">{{ drawerProduct.name }}</h2>
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm text-gray-500">{{ drawerProduct.factorySku }}</span>
              <NTag size="small" round>{{ drawerProduct.platform }}</NTag>
            </div>
            <NFlex :size="'small'" :wrap="true">
              <NTag v-for="tag in drawerProduct.tags" :key="tag" size="small" round :color="platformTagColor(drawerProduct.platform)">{{ tag }}</NTag>
            </NFlex>
          </div>
          <template v-if="drawerProductImages.length">
            <NDivider>详情图片</NDivider>
            <div class="space-y-3">
              <img v-for="img in drawerProductImages" :key="img.id" :src="'/local-images/' + img.path" class="w-full rounded-lg object-contain" />
            </div>
          </template>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>

<style>
.n-data-table-th--selection .n-checkbox {
  display: none;
}
.n-data-table-td--selection {
  vertical-align: middle;
}
.n-data-table-td--selection .n-checkbox {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: scale(1.5);
  transform-origin: center center;
}
.row-selected td {
  background: rgba(32, 128, 240, 0.12) !important;
}
.row-anchor {
  outline: 2px solid #2080f0;
  outline-offset: -2px;
}
</style>
