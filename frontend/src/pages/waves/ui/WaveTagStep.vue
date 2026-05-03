<script setup lang="ts">
import { PlayOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NDivider,
  NEmpty,
  NFlex,
  NIcon,
  NInput,
  NInputNumber,
  NPopover,
  NPagination,
  NTag,
  NTooltip,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  getProductImages,
  isWailsRuntimeAvailable,
  listProductsWithTags,
  listWaveMembers,
  listWaves,
  removeLevelTag,
  removeUserTag,
  upsertLevelTag,
  upsertUserTag,
  WAILS_PREVIEW_MESSAGE,
  type MemberItem,
  type WaveItem,
} from '@/shared/lib/wails/app'
import { useContextMenu } from '@/shared/composables/useContextMenu'
import { useAdaptiveTable } from '@/shared/composables/useAdaptiveTable'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

// ── types ──
type TagInfo = {
  tagName: string
  quantity: number
  tagType: string
  platform: string
  waveMemberId: number
}

// ── state ──
const wave = ref<WaveItem | null>(null)
const allTagProducts = ref<
  {
    id: number
    name: string
    factorySku: string
    platform: string
    tags: TagInfo[]
    coverImage: string
  }[]
>([])
const checkedProductIds = ref<number[]>([])
const selectedLevelTags = ref<string[]>([])
const levelTagQuantity = ref(1)
const showLevelPanel = ref(false)
const levelSearch = ref('')

const selectedUserTags = ref<string[]>([])
const userTagQuantity = ref(1)
const showUserPanel = ref(false)
const userSearch = ref('')
const isTagLoading = ref(false)
const errorMessage = ref('')

const showProductDrawer = ref(false)
const drawerProduct = ref<any>(null)
const drawerProductImages = ref<
  { id: number; path: string; sortOrder: number; sourceDir: string }[]
>([])

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
      await (tag.tagType === 'level'
        ? removeLevelTag(row.id, tag.platform, tag.tagName)
        : removeUserTag(row.id, tag.waveMemberId))
    } else {
      await (tag.tagType === 'level'
        ? upsertLevelTag(row.id, tag.platform, tag.tagName, newQty)
        : upsertUserTag(row.id, tag.waveMemberId, newQty))
    }
    await loadTagProducts()
    showTagPopover.value = false
    message.success('标签数量已更新')
  } catch (e) {
    message.error(String(e))
  }
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
  } catch (e) {
    message.error(String(e))
  }
}

// ── wave level tags ──
type LevelTag = { platform: string; tagName: string }
const waveLevelTags = computed<LevelTag[]>(() => {
  if (!wave.value?.levelTags) return []
  try {
    return JSON.parse(wave.value.levelTags) as LevelTag[]
  } catch {
    return []
  }
})

const batchTagOptions = computed(() =>
  waveLevelTags.value.map((t) => ({
    label: `${t.platform}·${t.tagName}`,
    value: `${t.platform}|${t.tagName}`,
  })),
)

const filteredLevelOptions = computed(() => {
  const q = levelSearch.value.toLowerCase()
  if (!q) return batchTagOptions.value
  return batchTagOptions.value.filter((o) => o.label.toLowerCase().includes(q))
})

function platformTagColor(platform: string): { color: string; textColor: string } {
  const colors: Record<string, { color: string; textColor: string }> = {
    BILIBILI: { color: '#00A1D633', textColor: '#00A1D6' },
    DOUYIN: { color: '#FE2C5533', textColor: '#FE2C55' },
  }
  return colors[platform] ?? { color: '#99999933', textColor: '#999999' }
}

const NEG_RED = '#EF4444'

function tagColors(tag: TagInfo) {
  const p = platformTagColor(tag.platform)
  const neg = tag.quantity < 0
  return {
    bg: { color: p.color },
    text: 'var(--text)',
    accent: p.textColor,
    number: neg ? NEG_RED : p.textColor,
    border: neg ? `2.5px solid ${NEG_RED}` : '',
  }
}

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
  const displayName =
    tag.tagType === 'user' ? wmNicknameMap.value.get(tag.waveMemberId) || tag.tagName : tag.tagName
  const t = tagColors(tag)
  const content =
    tag.quantity === 1
      ? h('span', { style: { color: t.text, fontWeight: 500 } }, displayName)
      : h('span', { style: { display: 'inline-flex', alignItems: 'baseline', gap: '1px' } }, [
          h('span', { style: { color: t.text, fontWeight: 500 } }, displayName),
          h('span', { style: { color: t.accent } }, ': '),
          h('span', { style: { color: t.number, fontWeight: 600 } }, String(tag.quantity)),
        ])
  return h(
    NPopover,
    {
      trigger: 'click',
      show:
        showTagPopover.value &&
        editTagProduct.value?.id === row.id &&
        editTagInfo.value?.tagName === tag.tagName &&
        editTagInfo.value?.tagType === tag.tagType,
      'onUpdate:show': (v: boolean) => {
        if (!v) showTagPopover.value = false
      },
      placement: 'bottom',
    },
    {
      trigger: () =>
        h(
          NTag,
          {
            size: 'medium',
            round: true,
            color: t.bg,
            style: { cursor: 'pointer', border: t.border },
            onClick: (e: MouseEvent) => {
              e.stopPropagation()
              openTagEdit(row, tag)
            },
          },
          { default: () => content },
        ),
      default: () =>
        h('div', { style: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px' } }, [
          h(NInputNumber, {
            value: editTagNewQty.value,
            size: 'small',
            style: { width: '100px' },
            'onUpdate:value': (v: number | null) => {
              if (v != null) editTagNewQty.value = v
            },
          }),
          h(
            NButton,
            {
              size: 'tiny',
              type: 'primary',
              onClick: () => handleUpdateTagQuantity(),
            },
            { default: () => '确定' },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              type: 'error',
              secondary: true,
              onClick: () => handleDeleteTag(),
            },
            { default: () => '删除' },
          ),
        ]),
    },
  )
}

// ══════════════════════════════════════════════
// adaptive paging — tag products table
// ══════════════════════════════════════════════

const lastClickedIndex = ref(-1)

const tableParentRef = ref<HTMLElement | null>(null)
const tableWrapperRef = ref<HTMLElement | null>(null)
const paginationRef = ref<HTMLElement | null>(null)
const indicatorRef = ref<HTMLElement | null>(null)

const {
  currentPage,
  totalPages,
  visibleItems,
  scrollMode,
  indicatorFontSize,
  indicatorLeft,
  indicatorRight,
  handlePageChange: rawHandlePageChange,
  remeasure,
  setupIndicatorObserver,
  teardown,
  init,
} = useAdaptiveTable(allTagProducts, {
  tableParentRef,
  tableWrapperRef,
  paginationRef,
  indicatorRef,
})

function handlePageChange(p: number) {
  rawHandlePageChange(p)
  lastClickedIndex.value = -1
}

// ── row props: multi-select with Ctrl/Shift, highlight selected ──

function rowProps(row: any) {
  const selected = checkedProductIds.value.includes(row.id)
  const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
  const isAnchor = selected && idx >= 0 && idx === lastClickedIndex.value
  return {
    class: [selected ? 'row-selected' : '', isAnchor ? 'row-anchor' : ''].filter(Boolean).join(' '),
    style: { cursor: 'pointer' },
    'data-contextmenu': 'tag-row',
    'data-product-id': row.id,
    onClick: (e: MouseEvent) => {
      if ((e.target as HTMLElement).closest('.n-checkbox')) {
        const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
        if (idx >= 0) lastClickedIndex.value = idx
        return
      }
      // Ctrl+Shift: additive range select
      if ((e.ctrlKey || e.metaKey) && e.shiftKey) {
        const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
        if (lastClickedIndex.value >= 0 && idx >= 0) {
          const lo = Math.min(lastClickedIndex.value, idx)
          const hi = Math.max(lastClickedIndex.value, idx)
          const rangeIds = visibleItems.value.slice(lo, hi + 1).map((p: any) => p.id)
          checkedProductIds.value = [...new Set([...checkedProductIds.value, ...rangeIds])]
        }
        return
      }
      // Ctrl: toggle single row
      if (e.ctrlKey || e.metaKey) {
        const id = row.id
        const idx = visibleItems.value.findIndex((p: any) => p.id === id)
        if (idx >= 0) lastClickedIndex.value = idx
        if (checkedProductIds.value.includes(id)) {
          checkedProductIds.value = checkedProductIds.value.filter((x) => x !== id)
        } else {
          checkedProductIds.value = [...checkedProductIds.value, id]
        }
        return
      }
      // Shift: replacement range select
      if (e.shiftKey) {
        const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
        if (lastClickedIndex.value >= 0 && idx >= 0) {
          const lo = Math.min(lastClickedIndex.value, idx)
          const hi = Math.max(lastClickedIndex.value, idx)
          checkedProductIds.value = visibleItems.value.slice(lo, hi + 1).map((p: any) => p.id)
        }
        return
      }
      // Plain click: select single row
      const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
      if (idx >= 0) lastClickedIndex.value = idx
      checkedProductIds.value = [row.id]
    },
    onContextmenu: (_e: MouseEvent) => {
      // Windows Explorer behavior: if target is not in selection, select it alone.
      if (!checkedProductIds.value.includes(row.id)) {
        const idx = visibleItems.value.findIndex((p: any) => p.id === row.id)
        if (idx >= 0) lastClickedIndex.value = idx
        checkedProductIds.value = [row.id]
      }
      // If already selected, keep the multi-selection as-is.
    },
  }
}

// ── selection buttons ──
const allSelected = computed(
  () =>
    allTagProducts.value.length > 0 &&
    checkedProductIds.value.length === allTagProducts.value.length,
)
const pageAllSelected = computed(() => {
  if (visibleItems.value.length === 0) return false
  return visibleItems.value.every((p) => checkedProductIds.value.includes(p.id))
})

function handleSelectAll() {
  if (allSelected.value) {
    checkedProductIds.value = []
    lastClickedIndex.value = -1
  } else {
    checkedProductIds.value = allTagProducts.value.map((p) => p.id)
  }
}
function handleSelectPage() {
  const pageIds = visibleItems.value.map((p) => p.id)
  if (pageAllSelected.value) {
    checkedProductIds.value = checkedProductIds.value.filter((id) => !pageIds.includes(id))
  } else {
    checkedProductIds.value = [...new Set([...checkedProductIds.value, ...pageIds])]
  }
}
function handleInvertPage() {
  const pageIds = new Set(visibleItems.value.map((p) => p.id))
  const toAdd = visibleItems.value
    .filter((p) => !checkedProductIds.value.includes(p.id))
    .map((p) => p.id)
  checkedProductIds.value = checkedProductIds.value.filter((id) => !pageIds.has(id)).concat(toAdd)
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
      title: '#',
      key: '__index',
      width: 50,
      render: (row: any) =>
        h('span', { style: { color: '#999' } }, String(productIndexMap.value.get(row.id) ?? '')),
    },
    {
      title: '',
      key: 'coverImage',
      width: 56,
      render: (row: any) =>
        row.coverImage
          ? h('div', { class: 'thumb-cell' }, [
              h('img', {
                src: '/local-images/' + row.coverImage,
                class: 'thumb-img rounded',
                onClick: (e: MouseEvent) => {
                  e.stopPropagation()
                  openProductDrawer(row)
                },
              }),
            ])
          : h('div', { class: 'thumb-cell' }, [h('div', { class: 'thumb-placeholder rounded' })]),
    },
    {
      title: '商品名',
      key: 'name',
      minWidth: 120,
      render: (row: any) => clampedText(row.name),
    },
    {
      title: '身份 Tag',
      key: 'levelTags',
      minWidth: 160,
      render: (row: any) =>
        h(
          NFlex,
          { size: 'small', wrap: true },
          {
            default: () =>
              (row.tags as TagInfo[])
                .filter((t: TagInfo) => t.tagType === 'level')
                .sort(
                  (a, b) =>
                    a.platform.localeCompare(b.platform) || a.tagName.localeCompare(b.tagName),
                )
                .map((t: TagInfo) => renderTagChip(row, t)),
          },
        ),
    },
    {
      title: '用户 Tag',
      key: 'userTags',
      minWidth: 160,
      render: (row: any) =>
        h(
          NFlex,
          { size: 'small', wrap: true },
          {
            default: () =>
              (row.tags as TagInfo[])
                .filter((t: TagInfo) => t.tagType === 'user')
                .sort(
                  (a, b) =>
                    a.platform.localeCompare(b.platform) || a.tagName.localeCompare(b.tagName),
                )
                .map((t: TagInfo) => renderTagChip(row, t)),
          },
        ),
    },
  ]
  return cols
})

// ── data loading ──
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

async function loadTagProducts() {
  if (!waveId.value) return
  isTagLoading.value = true
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 10000)
    allTagProducts.value = result.items.map((item) => ({
      id: item.id,
      name: item.name,
      factorySku: item.factorySku,
      platform: item.platform,
      tags: item.tags as any as TagInfo[],
      coverImage: (item as any).coverImage || '',
    }))
  } catch (e) {
    console.error('加载商品标签失败', e)
  } finally {
    isTagLoading.value = false
  }
}

// ── panel toggles ──

function toggleLevelPanel() {
  showLevelPanel.value = !showLevelPanel.value
  showUserPanel.value = false
}

function toggleUserPanel() {
  showUserPanel.value = !showUserPanel.value
  showLevelPanel.value = false
}

function toggleLevelTagSelection(key: string) {
  const idx = selectedLevelTags.value.indexOf(key)
  if (idx >= 0) selectedLevelTags.value.splice(idx, 1)
  else selectedLevelTags.value.push(key)
}

function toggleUserTagSelection(key: string) {
  const idx = selectedUserTags.value.indexOf(key)
  if (idx >= 0) selectedUserTags.value.splice(idx, 1)
  else selectedUserTags.value.push(key)
}

function setUserTagChecked(key: string) {
  if (!selectedUserTags.value.includes(key)) selectedUserTags.value.push(key)
  else selectedUserTags.value = selectedUserTags.value.filter((x) => x !== key)
}

// ── batch operations ──

async function handleBatchAddLevelTag() {
  if (selectedLevelTags.value.length === 0 || checkedProductIds.value.length === 0) return
  try {
    for (const key of selectedLevelTags.value) {
      const [platform, tagName] = key.split('|')
      for (const productId of checkedProductIds.value) {
        await upsertLevelTag(productId, platform, tagName, levelTagQuantity.value)
      }
    }
    message.success(
      `已为 ${checkedProductIds.value.length} 件商品添加 ${selectedLevelTags.value.length} 个身份 Tag`,
    )
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleBatchRemoveLevelTag() {
  if (selectedLevelTags.value.length === 0 || checkedProductIds.value.length === 0) return
  try {
    for (const key of selectedLevelTags.value) {
      const [platform, tagName] = key.split('|')
      for (const productId of checkedProductIds.value) {
        await removeLevelTag(productId, platform, tagName)
      }
    }
    message.success(
      `已为 ${checkedProductIds.value.length} 件商品移除 ${selectedLevelTags.value.length} 个身份 Tag`,
    )
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

// ── user tag batch ──
const waveMembers = ref<MemberItem[]>([])

const memberOptions = computed(() =>
  waveMembers.value.map((m) => ({
    label: `${m.platform} · ${m.latestNickname} (${m.platformUid})`,
    value: m.id,
  })),
)

const filteredMemberOptions = computed(() => {
  const q = userSearch.value.toLowerCase()
  if (!q) return memberOptions.value
  return memberOptions.value.filter((o) => o.label.toLowerCase().includes(q))
})

async function loadWaveMembers() {
  if (!waveId.value) return
  try {
    waveMembers.value = await listWaveMembers(waveId.value)
  } catch (e) {
    console.error('加载波次会员失败', e)
  }
}

async function handleBatchAddUserTag() {
  if (selectedUserTags.value.length === 0 || checkedProductIds.value.length === 0) return
  try {
    for (const id of selectedUserTags.value) {
      for (const productId of checkedProductIds.value) {
        await upsertUserTag(productId, Number(id), userTagQuantity.value)
      }
    }
    message.success(
      `已为 ${checkedProductIds.value.length} 件商品添加 ${selectedUserTags.value.length} 个用户 Tag`,
    )
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleBatchRemoveUserTag() {
  if (selectedUserTags.value.length === 0 || checkedProductIds.value.length === 0) return
  try {
    for (const id of selectedUserTags.value) {
      for (const productId of checkedProductIds.value) {
        await removeUserTag(productId, Number(id))
      }
    }
    message.success(
      `已为 ${checkedProductIds.value.length} 件商品移除 ${selectedUserTags.value.length} 个用户 Tag`,
    )
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

// ── clear all tags for checked products ──

async function handleClearAllTags() {
  if (checkedProductIds.value.length === 0) return message.warning('请先选择商品')
  try {
    for (const pid of checkedProductIds.value) {
      const product = allTagProducts.value.find((p) => p.id === pid)
      if (!product) continue
      for (const t of product.tags) {
        if (t.tagType === 'level') await removeLevelTag(pid, t.platform, t.tagName)
        else await removeUserTag(pid, t.waveMemberId)
      }
    }
    message.success(`已清空 ${checkedProductIds.value.length} 件商品的全部 Tag`)
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleClearLevelTags() {
  if (checkedProductIds.value.length === 0) return message.warning('请先选择商品')
  try {
    for (const pid of checkedProductIds.value) {
      const product = allTagProducts.value.find((p) => p.id === pid)
      if (!product) continue
      for (const t of product.tags) {
        if (t.tagType === 'level') await removeLevelTag(pid, t.platform, t.tagName)
      }
    }
    message.success(`已清空 ${checkedProductIds.value.length} 件商品的身份 Tag`)
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

async function handleClearUserTags() {
  if (checkedProductIds.value.length === 0) return message.warning('请先选择商品')
  try {
    for (const pid of checkedProductIds.value) {
      const product = allTagProducts.value.find((p) => p.id === pid)
      if (!product) continue
      for (const t of product.tags) {
        if (t.tagType === 'user') await removeUserTag(pid, t.waveMemberId)
      }
    }
    message.success(`已清空 ${checkedProductIds.value.length} 件商品的用户 Tag`)
    await loadTagProducts()
  } catch (e) {
    message.error(String(e))
  }
}

// ── product drawer ──
async function openProductDrawer(product: (typeof allTagProducts.value)[0]) {
  drawerProduct.value = product
  showProductDrawer.value = true
  try {
    drawerProductImages.value = await getProductImages(product.id)
  } catch {
    drawerProductImages.value = []
  }
}

function goPrev() {
  router.push({ name: 'waves-step-import', params: { waveId: String(waveId.value) } })
}
function goNext() {
  router.push({ name: 'waves-step-preview', params: { waveId: String(waveId.value) } })
}

// ── context menu ──
const { register } = useContextMenu()
let unregisterCtxMenu: (() => void) | null = null

// ── lifecycle ──
onMounted(async () => {
  await loadWave()
  await loadTagProducts()
  await loadWaveMembers()
  await nextTick()

  unregisterCtxMenu = register('tag-row', (_event: MouseEvent) => {
    const target = _event.target as HTMLElement | null
    if (!target) return []
    const tr = target.closest<HTMLElement>('[data-product-id]')
    if (!tr) return []
    const productId = Number(tr.dataset.productId)
    if (!productId) return []
    const product = allTagProducts.value.find((p) => p.id === productId)
    if (!product) return []
    const levelTags = product.tags.filter((t: TagInfo) => t.tagType === 'level')
    const userTags = product.tags.filter((t: TagInfo) => t.tagType === 'user')
    const items: Array<{ label: string; key: string; action: () => void; divider?: boolean }> = []
    if (levelTags.length > 0 || userTags.length > 0) {
      items.push({
        label: '清空全部 Tag',
        key: 'clear-all',
        action: async () => {
          for (const t of levelTags) {
            await removeLevelTag(product.id, t.platform, t.tagName)
          }
          for (const t of userTags) {
            await removeUserTag(product.id, t.waveMemberId)
          }
          await loadTagProducts()
          message.success('已清空全部 Tag')
        },
      })
    }
    if (levelTags.length > 0) {
      items.push({
        label: '清空身份 Tag',
        key: 'clear-level',
        action: async () => {
          for (const t of levelTags) {
            await removeLevelTag(product.id, t.platform, t.tagName)
          }
          await loadTagProducts()
          message.success('已清空身份 Tag')
        },
        divider: items.length > 0,
      })
    }
    if (userTags.length > 0) {
      items.push({
        label: '清空用户 Tag',
        key: 'clear-user',
        action: async () => {
          for (const t of userTags) {
            await removeUserTag(product.id, t.waveMemberId)
          }
          await loadTagProducts()
          message.success('已清空用户 Tag')
        },
        divider: items.length > 0,
      })
    }
    return items
  })

  await init()
})

watch([() => allTagProducts.value.length], async () => {
  await nextTick()
  await remeasure()
})

watch(scrollMode, async (v) => {
  if (!v) {
    await nextTick()
    setupIndicatorObserver()
  }
})

onUnmounted(() => {
  teardown()
  if (unregisterCtxMenu) unregisterCtxMenu()
})
</script>
<template>
  <div class="h-full flex flex-col">
    <!-- Row 1: selection + clear actions -->
    <div class="shrink-0 px-1 py-2">
      <NFlex :size="'small'" :wrap="true" class="items-center justify-between">
        <NFlex :size="'small'" :wrap="true" class="items-center">
          <span class="text-xs shrink-0 font-medium">选择</span>
          <NButton size="tiny" secondary @click="handleSelectAll">{{
            allSelected ? '取消全选' : '全选所有'
          }}</NButton>
          <NButton size="tiny" secondary @click="handleSelectPage">{{
            pageAllSelected ? '取消本页' : '本页全选'
          }}</NButton>
          <NButton size="tiny" secondary @click="handleInvertPage">本页反选</NButton>
          <NTag size="small" round :bordered="false"
            >已选 {{ checkedProductIds.length }} / {{ allTagProducts.length }}
          </NTag>
        </NFlex>
        <NFlex :size="'small'" :wrap="true">
          <NButton
            size="tiny"
            type="error"
            secondary
            :disabled="checkedProductIds.length === 0"
            @click="handleClearAllTags"
            >清空全部</NButton
          >
          <NButton
            size="tiny"
            type="warning"
            secondary
            :disabled="checkedProductIds.length === 0"
            @click="handleClearLevelTags"
            >清空身份</NButton
          >
          <NButton
            size="tiny"
            type="warning"
            secondary
            :disabled="checkedProductIds.length === 0"
            @click="handleClearUserTags"
            >清空用户</NButton
          >
        </NFlex>
      </NFlex>
    </div>
    <!-- Row 2: Tag operations (tabbed panels) -->
    <div class="shrink-0 px-1">
      <!-- Tab bar: trigger buttons in one row -->
      <NFlex :size="'small'" :wrap="true" class="items-center tab-bar">
        <!-- Level tag tab -->
        <NFlex
          :size="'small'"
          :wrap="true"
          class="items-center tab-trigger"
          :class="{ 'tab-trigger-active': showLevelPanel }"
        >
          <NButton
            size="small"
            :type="showLevelPanel ? 'primary' : 'default'"
            :secondary="!showLevelPanel"
            @click="toggleLevelPanel"
          >
            {{ selectedLevelTags.length ? `已选 ${selectedLevelTags.length} 项 ▾` : '身份 Tag ▾' }}
          </NButton>
          <NInputNumber
            v-model:value="levelTagQuantity"
            :min="-999"
            :max="999"
            size="small"
            style="width: 72px"
          />
          <NButton
            size="small"
            type="primary"
            :disabled="selectedLevelTags.length === 0 || checkedProductIds.length === 0"
            @click="handleBatchAddLevelTag"
            >添加</NButton
          >
          <NButton
            size="small"
            type="error"
            secondary
            :disabled="selectedLevelTags.length === 0 || checkedProductIds.length === 0"
            @click="handleBatchRemoveLevelTag"
            >删除</NButton
          >
          <NButton
            v-if="selectedLevelTags.length"
            size="tiny"
            quaternary
            @click="selectedLevelTags = []"
            >清空</NButton
          >
        </NFlex>

        <!-- User tag tab -->
        <NFlex
          :size="'small'"
          :wrap="true"
          class="items-center tab-trigger"
          :class="{ 'tab-trigger-active': showUserPanel }"
        >
          <NButton
            size="small"
            :type="showUserPanel ? 'primary' : 'default'"
            :secondary="!showUserPanel"
            @click="toggleUserPanel"
          >
            {{ selectedUserTags.length ? `已选 ${selectedUserTags.length} 人 ▾` : '用户 Tag ▾' }}
          </NButton>
          <NInputNumber
            v-model:value="userTagQuantity"
            :min="-999"
            :max="999"
            size="small"
            style="width: 72px"
          />
          <NButton
            size="small"
            type="primary"
            :disabled="selectedUserTags.length === 0 || checkedProductIds.length === 0"
            @click="handleBatchAddUserTag"
            >添加</NButton
          >
          <NButton
            size="small"
            type="error"
            secondary
            :disabled="selectedUserTags.length === 0 || checkedProductIds.length === 0"
            @click="handleBatchRemoveUserTag"
            >删除</NButton
          >
          <NButton
            v-if="selectedUserTags.length"
            size="tiny"
            quaternary
            @click="selectedUserTags = []"
            >清空</NButton
          >
        </NFlex>
      </NFlex>

      <!-- Panels: rendered below the tab bar, only one visible at a time -->
      <!-- Level tag panel -->
      <div v-if="showLevelPanel" class="tab-panel">
        <NInput
          v-if="batchTagOptions.length > 12"
          v-model:value="levelSearch"
          size="tiny"
          placeholder="过滤..."
          class="mb-2"
          clearable
        />
        <NFlex :size="'small'" :wrap="true">
          <NTag
            v-for="opt in filteredLevelOptions"
            :key="opt.value"
            size="medium"
            round
            :color="platformTagColor((opt.value as string).split('|')[0])"
            :style="{
              cursor: 'pointer',
              border: selectedLevelTags.includes(opt.value as string)
                ? '2.5px solid ' + platformTagColor((opt.value as string).split('|')[0]).textColor
                : '2.5px solid transparent',
            }"
            @click="toggleLevelTagSelection(opt.value as string)"
          >
            <span
              :style="{
                color: 'var(--text)',
                fontWeight: 500,
              }"
            >
              {{ opt.label }}
            </span>
          </NTag>
        </NFlex>
        <NFlex :size="'small'" class="mt-2 items-center justify-between">
          <span class="text-xs" :style="{ color: 'var(--muted)' }"
            >已选 {{ selectedLevelTags.length }} 项</span
          >
          <NButton size="tiny" @click="showLevelPanel = false">收起</NButton>
        </NFlex>
      </div>

      <!-- User tag panel -->
      <div v-if="showUserPanel" class="tab-panel">
        <NInput
          v-model:value="userSearch"
          size="tiny"
          placeholder="搜索会员名或 UID..."
          class="mb-2"
          clearable
        />
        <div class="max-h-48 overflow-y-auto">
          <NFlex :size="'small'" :wrap="true">
            <NTooltip
              v-for="opt in filteredMemberOptions"
              :key="opt.value"
              trigger="hover"
              :delay="500"
              :keep-alive-on-hover="false"
            >
              <template #trigger>
                <NTag
                  size="medium"
                  round
                  :color="platformTagColor(opt.label.split(' · ')[0] || '')"
                  :style="{
                    cursor: 'pointer',
                    border: selectedUserTags.includes(String(opt.value))
                      ? '2.5px solid ' + platformTagColor(opt.label.split(' · ')[0] || '').textColor
                      : '2.5px solid transparent',
                  }"
                  @click="toggleUserTagSelection(String(opt.value))"
                >
                  <span
                    :style="{
                      color: 'var(--text)',
                      fontWeight: 500,
                    }"
                  >
                    {{ opt.label.split(' · ')[1]?.split(' (')[0] || opt.label }}
                  </span>
                </NTag>
              </template>
              {{ opt.label }}
            </NTooltip>
          </NFlex>
        </div>
        <NFlex :size="'small'" class="mt-2 items-center justify-between">
          <span class="text-xs" :style="{ color: 'var(--muted)' }"
            >已选 {{ selectedUserTags.length }} 人</span
          >
          <NButton size="tiny" @click="showUserPanel = false">收起</NButton>
        </NFlex>
      </div>
    </div>

    <div ref="tableParentRef" class="flex-1 min-h-0 flex flex-col overflow-hidden px-1">
      <div
        ref="tableWrapperRef"
        :class="scrollMode ? 'overflow-y-auto flex-1 min-h-0' : 'overflow-hidden'"
      >
        <NDataTable
          :columns="tagColumns"
          :data="visibleItems"
          :loading="isTagLoading"
          :bordered="false"
          :row-key="(row: any) => row.id"
          v-model:checked-row-keys="checkedProductIds"
          :pagination="false"
          size="medium"
          :row-props="rowProps"
        />
      </div>
      <div
        v-if="!scrollMode"
        ref="indicatorRef"
        class="flex-1 flex justify-center items-center select-none"
        :style="{
          fontSize: indicatorFontSize + 'px',
          lineHeight: 1,
          fontFamily: 'monospace',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          marginBottom: '12px',
        }"
      >
        <span style="color: rgba(96, 165, 250, 0.1)">{{ indicatorLeft }}</span
        ><span style="color: rgba(251, 191, 36, 0.1)">{{ indicatorRight }}</span>
      </div>
      <div
        v-if="!scrollMode"
        ref="paginationRef"
        class="flex justify-center mt-0 mb-6 shrink-0"
        style="transform: scale(1.5); transform-origin: top center"
      >
        <NPagination
          :page="currentPage"
          :page-count="totalPages"
          size="small"
          @update:page="handlePageChange"
        />
      </div>
    </div>

    <div
      class="flex justify-between shrink-0 pt-3 pb-1 px-1 border-t border-gray-100 dark:border-gray-700"
    >
      <NButton @click="goPrev">上一步</NButton>
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>

    <!-- Product Detail Drawer -->
    <NDrawer
      :show="showProductDrawer"
      :width="560"
      @update:show="
        (v: boolean) => {
          if (!v) {
            showProductDrawer = false
            drawerProduct = null
            drawerProductImages = []
          }
        }
      "
    >
      <NDrawerContent title="商品详情" closable>
        <template v-if="drawerProduct">
          <div class="space-y-3">
            <img
              v-if="drawerProduct.coverImage"
              :src="'/local-images/' + drawerProduct.coverImage"
              class="w-full rounded-lg object-contain"
              style="max-height: 50vh"
            />
            <h2 class="text-xl font-semibold">{{ drawerProduct.name }}</h2>
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-sm text-gray-500">{{ drawerProduct.factorySku }}</span>
              <NTag size="small" round>{{ drawerProduct.platform }}</NTag>
            </div>
            <template
              v-for="group in [
                {
                  label: '身份 Tag',
                  tags: drawerProduct.tags.filter((t: TagInfo) => t.tagType === 'level'),
                },
                {
                  label: '用户 Tag',
                  tags: drawerProduct.tags.filter((t: TagInfo) => t.tagType === 'user'),
                },
              ]"
              :key="group.label"
            >
              <div
                v-if="group.tags.length"
                class="text-xs text-gray-500 mb-1"
                :class="{ 'mt-2': group.label === '用户 Tag' }"
              >
                {{ group.label }}
              </div>
              <NFlex v-if="group.tags.length" :size="'small'" :wrap="true">
                <NTag
                  v-for="tag in group.tags"
                  :key="tag.tagName + tag.tagType"
                  size="medium"
                  round
                  :color="tagColors(tag).bg"
                  :style="{ boxShadow: tagColors(tag).border }"
                >
                  <template v-if="tag.quantity === 1">
                    <span :style="{ color: tagColors(tag).text, fontWeight: 500 }">
                      {{
                        tag.tagType === 'user'
                          ? wmNicknameMap.get(tag.waveMemberId) || tag.tagName
                          : tag.tagName
                      }}
                    </span>
                  </template>
                  <template v-else>
                    <span :style="{ color: tagColors(tag).text, fontWeight: 500 }">
                      {{
                        tag.tagType === 'user'
                          ? wmNicknameMap.get(tag.waveMemberId) || tag.tagName
                          : tag.tagName
                      }}
                    </span>
                    <span :style="{ color: tagColors(tag).accent }">: </span>
                    <span :style="{ color: tagColors(tag).number, fontWeight: 600 }">
                      {{ tag.quantity }}
                    </span>
                  </template>
                </NTag>
              </NFlex>
            </template>
          </div>
          <template v-if="drawerProductImages.length">
            <NDivider>详情图片</NDivider>
            <div class="space-y-3">
              <img
                v-for="img in drawerProductImages"
                :key="img.id"
                :src="'/local-images/' + img.path"
                class="w-full rounded-lg object-contain"
              />
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
  outline: 2px solid rgba(32, 128, 240, 0.55);
  outline-offset: -2px;
}

.thumb-cell {
  display: flex;
  align-items: center;
  height: 40px;
}

.thumb-img {
  max-height: 100%;
  max-width: 100%;
  object-fit: contain;
  cursor: pointer;
}

.thumb-placeholder {
  width: 40px;
  height: 40px;
  background: #e5e7eb;
}

/* Panel tooltips must not capture mouse — user may want to click a tag behind them */
.tab-panel .n-popover {
  pointer-events: none;
}

.tab-bar {
  position: relative;
  z-index: 1;
}

.tab-trigger {
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 2px 6px;
  transition: border-color 0.15s;
}

.tab-trigger-active {
  border: 1px solid var(--muted);
  border-bottom-color: var(--surface-strong);
  background: var(--surface-strong);
  border-radius: 6px 6px 0 0;
  padding: 2px 6px;
  position: relative;
  z-index: 2;
}

.tab-panel {
  border: 1px solid var(--muted);
  border-radius: 0 6px 6px 6px;
  padding: 8px;
  background: var(--surface-strong);
  margin-top: -1px;
  position: relative;
  z-index: 1;
}
</style>
