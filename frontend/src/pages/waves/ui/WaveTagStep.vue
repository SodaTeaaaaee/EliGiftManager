<script setup lang="ts">
import { PlayOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDataTable, NDrawer, NDrawerContent, NDivider, NEmpty, NFlex, NIcon, NInputNumber, NPopover, NPagination, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { getProductImages, isWailsRuntimeAvailable, listProductsWithTags, listWaveMembers, listWaves, removeLevelTag, removeUserTag, upsertLevelTag, upsertUserTag, WAILS_PREVIEW_MESSAGE, type MemberItem, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

// ── types ──
type TagInfo = { tagName: string; quantity: number; tagType: string; platform: string; waveMemberId: number }

// ── state ──
const wave = ref<WaveItem | null>(null)
const allTagProducts = ref<{ id: number; name: string; factorySku: string; platform: string; tags: TagInfo[]; coverImage: string }[]>([])
const checkedProductIds = ref<number[]>([])
const selectedLevelTag = ref<string | null>(null)
const levelTagQuantity = ref(1)
const selectedUserTagMember = ref<string | null>(null)
const userTagQuantity = ref(1)
const isTagLoading = ref(false)
const errorMessage = ref('')

const showProductDrawer = ref(false)
const drawerProduct = ref<any>(null)
const drawerProductImages = ref<{ id: number; path: string; sortOrder: number; sourceDir: string }[]>([])

// ── tag edit popover ──
const editTagProduct = ref<any>(null)
const editTagInfo = ref<TagInfo | null>(null)
const editTagNewQty = ref(1)
const showTagPopover = ref(false)

function openTagEdit(row: any, tag: TagInfo) {
  editTagProduct.value = row
  editTagInfo.value = tag
  editTagNewQty.value = tag.quantity
  showTagPopover.value = true
}

async function handleUpdateTagQuantity() {
  if (!editTagProduct.value || !editTagInfo.value) return
  const row = editTagProduct.value
  const tag = editTagInfo.value
  const newQty = editTagNewQty.value

  try {
    if (newQty === 0) {
      await (tag.tagType === 'level' ? removeLevelTag(row.id, tag.platform, tag.tagName) : removeUserTag(row.id, tag.waveMemberId))
    } else {
      await (tag.tagType === 'level' ? upsertLevelTag(row.id, tag.platform, tag.tagName, newQty) : upsertUserTag(row.id, tag.waveMemberId, newQty))
    }
    await loadTagProducts()
    showTagPopover.value = false
    message.success('标签数量已更新')
  } catch (e) { message.error(String(e)) }
}

async function handleDeleteTag() {
  if (!editTagProduct.value || !editTagInfo.value) return
  const row = editTagProduct.value
  const tag = editTagInfo.value
  try {
    if (tag.tagType === 'level') {
      await removeLevelTag(row.id, tag.platform, tag.tagName)
    } else {
      await removeUserTag(row.id, tag.waveMemberId)
    }
    await loadTagProducts()
    showTagPopover.value = false
    message.success('标签已删除')
  } catch (e) { message.error(String(e)) }
}

// ── wave level tags ──
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
  return colors[platform] || { color: '#99999933', textColor: '#999999' }
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

// ── wave member lookup for user tag display ──
const wmNicknameMap = computed(() => {
  const map = new Map<number, string>()
  for (const m of waveMembers.value) {
    map.set(m.id, m.latestNickname || m.platformUid)
  }
  return map
})

// ── tag chip renderer (with NPopover for quantity editing + delete) ──
function renderTagChip(row: any, tag: TagInfo) {
  const displayName = tag.tagType === 'user'
    ? (wmNicknameMap.value.get(tag.waveMemberId) || tag.tagName)
    : tag.tagName
  const accent = platformTagColor(tag.platform).textColor || '#aaa'
  const content = tag.quantity === 1
    ? h('span', { style: { color: accent, fontWeight: 500 } }, displayName)
    : h('span', { style: { display: 'inline-flex', alignItems: 'baseline', gap: '1px' } }, [
        h('span', { style: { color: accent, fontWeight: 500 } }, displayName),
        h('span', { style: { color: '#666', margin: '0 1px' } }, ':'),
        h('span', { style: { color: '#fff', fontWeight: 600 } }, String(tag.quantity)),
      ])
  return h(NPopover, {
    trigger: 'click',
    show: showTagPopover.value && editTagProduct.value?.id === row.id && editTagInfo.value?.tagName === tag.tagName && editTagInfo.value?.tagType === tag.tagType,
    'onUpdate:show': (v: boolean) => { if (!v) showTagPopover.value = false },
    placement: 'bottom',
  }, {
    trigger: () => h(NTag, {
      size: 'medium', round: true,
      color: platformTagColor(tag.platform).color,
      style: { cursor: 'pointer' },
      onClick: (e: MouseEvent) => {
        e.stopPropagation()
        openTagEdit(row, tag)
      },
    }, { default: () => content }),
    default: () => h('div', { style: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px' } }, [
      h(NInputNumber, {
        value: editTagNewQty.value,
        size: 'small',
        style: { width: '100px' },
        'onUpdate:value': (v: number | null) => { if (v != null) editTagNewQty.value = v },
      }),
      h(NButton, {
        size: 'tiny', type: 'primary',
        onClick: () => handleUpdateTagQuantity(),
      }, { default: () => '确定' }),
      h(NButton, {
        size: 'tiny', type: 'error', secondary: true,
        onClick: () => handleDeleteTag(),
      }, { default: () => '删除' }),
    ]),
  })
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

function handleTagPageChange(p: number) { tagCurrentPage.value = p; lastClickedIndex.value = -1 }

// ── row props: multi-select with Ctrl/Shift, highlight selected ──
const lastClickedIndex = ref(-1)

function rowProps(row: any) {
  const selected = checkedProductIds.value.includes(row.id)
  return {
    class: selected ? 'row-selected' : '',
    style: { cursor: 'pointer' },
    onClick: (e: MouseEvent) => {
      if ((e.target as HTMLElement).closest('.n-checkbox')) {
        const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
        if (idx >= 0) lastClickedIndex.value = idx
        return
      }
      // Ctrl+Shift: additive range select
      if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
        const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
        if (lastClickedIndex.value >= 0 && idx >= 0) {
          const lo = Math.min(lastClickedIndex.value, idx)
          const hi = Math.max(lastClickedIndex.value, idx)
          const rangeIds = visibleTagProducts.value.slice(lo, hi + 1).map((p: any) => p.id)
          checkedProductIds.value = [...new Set([...checkedProductIds.value, ...rangeIds])]
        }
        return
      }
      // Ctrl: toggle single row
      if (e.ctrlKey || e.metaKey) {
        const id = row.id
        const idx = visibleTagProducts.value.findIndex((p: any) => p.id === id)
        if (idx >= 0) lastClickedIndex.value = idx
        if (checkedProductIds.value.includes(id)) {
          checkedProductIds.value = checkedProductIds.value.filter(x => x !== id)
        } else {
          checkedProductIds.value = [...checkedProductIds.value, id]
        }
        return
      }
      // Shift: replacement range select
      if (e.shiftKey) {
        const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
        if (lastClickedIndex.value >= 0 && idx >= 0) {
          const lo = Math.min(lastClickedIndex.value, idx)
          const hi = Math.max(lastClickedIndex.value, idx)
          checkedProductIds.value = visibleTagProducts.value.slice(lo, hi + 1).map((p: any) => p.id)
        }
        return
      }
      // Plain click: select single row
      const idx = visibleTagProducts.value.findIndex((p: any) => p.id === row.id)
      if (idx >= 0) lastClickedIndex.value = idx
      checkedProductIds.value = [row.id]
    },
  }
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
        row.coverImage
          ? h('img', {
            src: '/local-images/' + row.coverImage,
            class: 'w-10 h-10 rounded object-cover',
            style: { cursor: 'pointer' },
            onClick: (e: MouseEvent) => { e.stopPropagation(); openProductDrawer(row) },
          })
          : h('div', { class: 'w-10 h-10 rounded bg-gray-100' }),
    },
    {
      title: '商品名', key: 'name', minWidth: 120,
      render: (row: any) => clampedText(row.name),
    },
    {
      title: '身份 Tag', key: 'levelTags', minWidth: 160,
      render: (row: any) => h(NFlex, { size: 'small', wrap: true }, {
        default: () => (row.tags as TagInfo[])
          .filter((t: TagInfo) => t.tagType === 'level')
          .sort((a, b) => a.platform.localeCompare(b.platform) || a.tagName.localeCompare(b.tagName))
          .map((t: TagInfo) => renderTagChip(row, t)),
      }),
    },
    {
      title: '用户 Tag', key: 'userTags', minWidth: 160,
      render: (row: any) => h(NFlex, { size: 'small', wrap: true }, {
        default: () => (row.tags as TagInfo[])
          .filter((t: TagInfo) => t.tagType === 'user')
          .sort((a, b) => a.platform.localeCompare(b.platform) || a.tagName.localeCompare(b.tagName))
          .map((t: TagInfo) => renderTagChip(row, t)),
      }),
    },
  ]
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
      platform: item.platform, tags: (item.tags as any as TagInfo[]), coverImage: (item as any).coverImage || '',
    }))
  } catch (e) { console.error('加载商品标签失败', e) }
  finally { isTagLoading.value = false }
}

// ── batch operations ──

async function handleBatchAddLevelTag() {
  if (!selectedLevelTag.value || checkedProductIds.value.length === 0) return
  const [platform, tagName] = selectedLevelTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) {
      await upsertLevelTag(productId, platform, tagName, levelTagQuantity.value)
    }
    message.success(`已为 ${checkedProductIds.value.length} 件商品打上 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleBatchRemoveLevelTag() {
  if (!selectedLevelTag.value || checkedProductIds.value.length === 0) return
  const [platform, tagName] = selectedLevelTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) {
      await removeLevelTag(productId, platform, tagName)
    }
    message.success(`已为 ${checkedProductIds.value.length} 件商品移除 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

// ── user tag batch ──
const waveMembers = ref<MemberItem[]>([])

const memberOptions = computed(() =>
  waveMembers.value.map(m => ({
    label: `${m.platform} · ${m.latestNickname} (${m.platformUid})`,
    value: m.id,
  }))
)

async function loadWaveMembers() {
  if (!waveId.value) return
  try { waveMembers.value = await listWaveMembers(waveId.value) }
  catch (e) { console.error('加载波次会员失败', e) }
}

async function handleBatchAddUserTag() {
  if (!selectedUserTagMember.value || checkedProductIds.value.length === 0) return
  const waveMemberId = Number(selectedUserTagMember.value)
  const quantity = userTagQuantity.value
  try {
    for (const productId of checkedProductIds.value) {
      await upsertUserTag(productId, waveMemberId, quantity)
    }
    message.success(`已为 ${checkedProductIds.value.length} 件商品添加用户 Tag`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleBatchRemoveUserTag() {
  if (!selectedUserTagMember.value || checkedProductIds.value.length === 0) return
  const waveMemberId = Number(selectedUserTagMember.value)
  try {
    for (const productId of checkedProductIds.value) {
      await removeUserTag(productId, waveMemberId)
    }
    message.success(`已为 ${checkedProductIds.value.length} 件商品移除用户 Tag`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

// ── product drawer ──
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
  await loadWaveMembers()
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

    <div class="shrink-0 px-1 space-y-2">
      <div v-if="waveLevelTags.length > 0">
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

      <!-- Level tag row -->
      <NFlex :size="'small'" :wrap="true" class="items-center">
        <span class="text-xs shrink-0 font-medium" style="width: 52px">身份 Tag</span>
        <NSelect v-model:value="selectedLevelTag" :options="batchTagOptions" placeholder="选择等级" size="small"
          style="width: 180px" clearable />
        <NInputNumber v-model:value="levelTagQuantity" :min="-999" :max="999" size="small" style="width: 80px" />
        <NButton size="small" type="primary" :disabled="!selectedLevelTag || checkedProductIds.length === 0"
          @click="handleBatchAddLevelTag">添加</NButton>
        <NButton size="small" type="error" secondary :disabled="!selectedLevelTag || checkedProductIds.length === 0"
          @click="handleBatchRemoveLevelTag">删除</NButton>
      </NFlex>

      <!-- User tag row -->
      <NFlex :size="'small'" :wrap="true" class="items-center">
        <span class="text-xs shrink-0 font-medium" style="width: 52px">用户 Tag</span>
        <NSelect v-model:value="selectedUserTagMember" :options="memberOptions" placeholder="搜索会员" size="small"
          style="width: 180px" clearable filterable />
        <NInputNumber v-model:value="userTagQuantity" :min="-999" :max="999" size="small" style="width: 80px" />
        <NButton size="small" type="primary" :disabled="!selectedUserTagMember || checkedProductIds.length === 0"
          @click="handleBatchAddUserTag">添加</NButton>
        <NButton size="small" type="error" secondary :disabled="!selectedUserTagMember || checkedProductIds.length === 0"
          @click="handleBatchRemoveUserTag">删除</NButton>
      </NFlex>
    </div>

    <div class="flex-1 min-h-0 flex flex-col overflow-hidden px-1">
      <div ref="tagTableWrapper" class="flex-1 min-h-0 overflow-hidden">
        <NDataTable :columns="tagColumns" :data="visibleTagProducts" :loading="isTagLoading" :bordered="false"
          :row-key="(row: any) => row.id" v-model:checked-row-keys="checkedProductIds"
          :pagination="false" size="medium"
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
              <NTag v-for="tag in drawerProduct.tags" :key="tag.tagName + tag.tagType" size="small" round
                :color="platformTagColor(tag.platform).color">
                {{ tag.quantity === 1 ? tag.tagName : `${tag.tagName}:${tag.quantity}` }}
              </NTag>
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
</style>
