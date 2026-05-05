<script setup lang="ts">
import { DownloadOutline, LibraryOutline, PersonOutline } from '@vicons/ionicons5'
import { computed, onMounted, ref, watch } from 'vue'
import {
  NButton,
  NIcon,
  NModal,
  NSelect,
  NTag,
  NTabs,
  NTabPane,
  useMessage,
} from 'naive-ui'
import {
  addFromPreset,
  getPresetContent,
  listBuiltinPresets,
  listTemplates,
  listUserPresets,
  type PresetContent,
  type PresetInfo,
  type TemplateItem,
} from '@/shared/lib/wails/app'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': (v: boolean) => void; added: [] }>()

const message = useMessage()
const builtinPresets = ref<PresetInfo[]>([])
const userPresets = ref<PresetInfo[]>([])
const templates = ref<TemplateItem[]>([])
const loading = ref(false)

async function refresh() {
  try {
    const [bp, up, t] = await Promise.all([
      listBuiltinPresets(),
      listUserPresets(),
      listTemplates(),
    ])
    builtinPresets.value = bp
    userPresets.value = up
    templates.value = t
  } catch (e) {
    console.error(e)
  }
}

onMounted(refresh)
watch(() => props.visible, (v) => { if (v) refresh() })

function samePlatform(t: TemplateItem, p: PresetInfo) {
  return (t.platform || '通用') === p.platform
}

function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true
  if (typeof a !== typeof b) return false
  if (a === null || b === null) return false
  if (typeof a !== 'object' || typeof b !== 'object') return false

  const ka = Object.keys(a as Record<string, unknown>)
  const kb = Object.keys(b as Record<string, unknown>)
  if (ka.length !== kb.length) return false

  for (const key of ka) {
    if (!kb.includes(key)) return false
    if (!deepEqual(
      (a as Record<string, unknown>)[key],
      (b as Record<string, unknown>)[key],
    )) return false
  }
  return true
}

async function handleAdd(source: string, preset: PresetInfo) {
  loading.value = true
  try {
    // Fetch full content for dedup
    let content: PresetContent
    try {
      content = await getPresetContent(source, preset.id)
    } catch {
      message.error('读取预设内容失败')
      return
    }

    const samePlatformTemplates = templates.value.filter((t) => samePlatform(t, preset))

    // Find name match
    const nameMatch = samePlatformTemplates.find(
      (t) => t.type === preset.type && t.name === preset.name,
    )

    // Find content match — compare mappingRules by deep-equal
    let contentMatchTitle = ''
    for (const t of samePlatformTemplates) {
      try {
        const existingRules = JSON.parse(t.mappingRules)
        if (deepEqual(existingRules, content.mappingRules)) {
          contentMatchTitle = t.name
          break
        }
      } catch {
        // ignore malformed
      }
    }
    const nameMatchTitle = nameMatch?.name || ''

    // Dedup strategy
    if (nameMatchTitle && contentMatchTitle && nameMatchTitle === contentMatchTitle) {
      message.warning('该预设已添加过（名称与内容均一致），无需重复')
      return
    }
    if (nameMatchTitle) {
      message.warning(`已存在同名模板「${nameMatchTitle}」，请修改名称或使用编辑功能`)
      return
    }
    if (contentMatchTitle) {
      message.warning(`已存在功能一致的模板「${contentMatchTitle}」，内容完全相同`)
      return
    }

    await addFromPreset(source, preset.id)
    message.success(`已添加模板：${preset.name}`)
    await refresh()
    emit('added')
  } catch (e) {
    message.error(String(e))
  } finally {
    loading.value = false
  }
}

const groupOrder = ['BILIBILI', 'DOUYIN', 'YOUTUBE', '柔造']
function platformOrder(p: string) {
  const idx = groupOrder.indexOf(p)
  return idx >= 0 ? idx : 999
}

// sorted presets grouped by platform
function grouped(list: PresetInfo[]) {
  const sorted = [...list].sort(
    (a, b) => platformOrder(a.platform) - platformOrder(b.platform),
  )
  const groups: { platform: string; items: PresetInfo[] }[] = []
  for (const p of sorted) {
    let g = groups[groups.length - 1]
    if (!g || g.platform !== p.platform) {
      g = { platform: p.platform, items: [] }
      groups.push(g)
    }
    g.items.push(p)
  }
  return groups
}

function typeLabel(type: string) {
  const m: Record<string, string> = {
    import_product: '礼物导入',
    import_dispatch_record: '会员导入',
    export_order: '订单导出',
  }
  return m[type] ?? type
}
function typeColor(type: string) {
  const m: Record<string, string> = {
    import_product: 'info',
    import_dispatch_record: 'success',
    export_order: 'warning',
  }
  return m[type] ?? 'default'
}

const show = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})
</script>

<template>
  <NModal v-model:show="show" title="从预设添加模板" preset="card" style="max-width: 600px">
    <NTabs type="line">
      <NTabPane name="builtin" tab="内置预设">
        <template #tab>
          <NIcon size="16"><LibraryOutline /></NIcon>
          <span class="ml-1">内置预设</span>
        </template>
        <div v-if="!builtinPresets.length" class="text-sm text-gray-400 py-4">暂无内置预设</div>
        <div v-for="group in grouped(builtinPresets)" :key="'b-' + group.platform" class="mb-3">
          <div class="flex items-center gap-2 mb-1">
            <NIcon size="14"><PersonOutline /></NIcon>
            <span class="text-sm font-semibold">{{ group.platform }}</span>
          </div>
          <div class="space-y-1 pl-5">
            <div
              v-for="p in group.items"
              :key="p.id"
              class="flex items-center justify-between py-1"
            >
              <div class="flex items-center gap-2">
                <span class="text-sm">{{ p.name }}</span>
                <NTag :type="typeColor(p.type)" size="tiny" round>
                  {{ typeLabel(p.type) }}
                </NTag>
              </div>
              <NButton size="tiny" type="primary" :loading="loading" @click="handleAdd('builtin', p)">
                <template #icon><NIcon size="14"><DownloadOutline /></NIcon></template>
                添加
              </NButton>
            </div>
          </div>
        </div>
      </NTabPane>
      <NTabPane name="user" tab="我的预设">
        <template #tab>
          <NIcon size="16"><PersonOutline /></NIcon>
          <span class="ml-1">我的预设</span>
        </template>
        <div v-if="!userPresets.length" class="text-sm text-gray-400 py-4">
          暂无自定义预设。将 JSON 文件放入 data/presets/user/ 即可显示。
        </div>
        <div v-for="group in grouped(userPresets)" :key="'u-' + group.platform" class="mb-3">
          <div class="flex items-center gap-2 mb-1">
            <NIcon size="14"><PersonOutline /></NIcon>
            <span class="text-sm font-semibold">{{ group.platform }}</span>
          </div>
          <div class="space-y-1 pl-5">
            <div
              v-for="p in group.items"
              :key="p.id"
              class="flex items-center justify-between py-1"
            >
              <div class="flex items-center gap-2">
                <span class="text-sm">{{ p.name }}</span>
                <NTag :type="typeColor(p.type)" size="tiny" round>
                  {{ typeLabel(p.type) }}
                </NTag>
              </div>
              <NButton size="tiny" type="primary" :loading="loading" @click="handleAdd('user', p)">
                <template #icon><NIcon size="14"><DownloadOutline /></NIcon></template>
                添加
              </NButton>
            </div>
          </div>
        </div>
      </NTabPane>
    </NTabs>
  </NModal>
</template>
