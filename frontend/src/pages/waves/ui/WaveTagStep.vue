<script setup lang="ts">
import { PlayOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NButton, NDataTable, NEmpty, NIcon, NSelect, NTag, NFlex, useMessage, type DataTableColumns } from 'naive-ui'
import { allocateByTags, assignProductTag, isWailsRuntimeAvailable, listProductsWithTags, listWaves, removeProductTag, WAILS_PREVIEW_MESSAGE, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const wave = ref<WaveItem | null>(null)
const tagProducts = ref<{ id: number; name: string; factorySku: string; platform: string; tags: string[] }[]>([])
const tagProductTotal = ref(0)
const tagProductPage = ref(1)
const checkedProductIds = ref<number[]>([])
const selectedBatchTag = ref<string | null>(null)
const isTagLoading = ref(false)
const errorMessage = ref('')

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

const productColumns: DataTableColumns = [
  { type: 'selection' as const },
  { title: '商品名', key: 'name', minWidth: 160 },
  { title: 'FactorySKU', key: 'factorySku', minWidth: 140 },
  {
    title: 'Tags', key: 'tags', minWidth: 200, render: (row) => h(NFlex, { size: 'small', wrap: true }, {
      default: () => (row.tags as string[]).map(tag => h(NTag, {
        size: 'small', round: true, closable: true,
        onClose: () => handleRemoveTag(row.id as number, row.platform as string, tag),
      }, { default: () => tag }))
    })
  },
]

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
    const result = await listProductsWithTags(waveId.value, '', tagProductPage.value, 50)
    tagProducts.value = result.items.map(item => ({
      id: item.id, name: item.name, factorySku: item.factorySku,
      platform: item.platform, tags: item.tags,
    }))
    tagProductTotal.value = result.total
  } catch (e) { console.error('加载商品标签失败', e) }
  finally { isTagLoading.value = false }
}

async function handleAssignTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await assignProductTag(productId, platform, tagName) }
    message.success(`已为 ${checkedProductIds.value.length} 件商品打上 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleBatchRemoveTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await removeProductTag(productId, platform, tagName) }
    message.success(`已为 ${checkedProductIds.value.length} 件商品移除 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (e) { message.error(String(e)) }
}

async function handleRemoveTag(productId: number, platform: string, tagName: string) {
  try { await removeProductTag(productId, platform, tagName); await loadTagProducts() }
  catch (e) { message.error(String(e)) }
}

async function handleAllocateByTags() {
  if (!waveId.value) return
  try { const count = await allocateByTags(waveId.value); message.success(`Tag 分配完成，共 ${count} 条记录`); await loadTagProducts() }
  catch (e) { message.error(String(e)) }
}

function goPrev() {
  router.push({ name: 'waves-step-import', params: { waveId: String(waveId.value) } })
}
function goNext() {
  router.push({ name: 'waves-step-preview', params: { waveId: String(waveId.value) } })
}

onMounted(async () => {
  await loadWave()
  await loadTagProducts()
})
</script>
<template>
  <NCard size="small">
    <template #header>
      <span class="flex items-center gap-2">
        <NIcon><PlayOutline /></NIcon>步骤二：Tag 管理与分配
      </span>
    </template>
    <div class="space-y-3">
      <div v-if="waveLevelTags.length > 0">
        <span class="text-xs text-gray-500 block mb-2">可选 Tag：</span>
        <NFlex :size="'small'" :wrap="true">
          <NTag v-for="tag in waveLevelTags" :key="`${tag.platform}|${tag.tagName}`" size="small" round
            :color="platformTagColor(tag.platform)">{{ tag.platform }}·{{ tag.tagName }}</NTag>
        </NFlex>
      </div>
      <NEmpty v-else description="当前波次无等级 Tag，导入会员数据后将自动提取" size="small" />

      <div class="flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <span class="text-xs text-gray-500 shrink-0">批量操作：</span>
        <NSelect v-model:value="selectedBatchTag" :options="batchTagOptions" placeholder="勾选 tag" size="small"
          style="width: 180px" clearable />
        <NButton size="small" type="primary" @click="handleAssignTag"
          :disabled="!selectedBatchTag || checkedProductIds.length === 0">打标</NButton>
        <NButton size="small" type="warning" @click="handleBatchRemoveTag"
          :disabled="!selectedBatchTag || checkedProductIds.length === 0">取消打标</NButton>
      </div>

      <NDataTable :columns="productColumns" :data="tagProducts" :loading="isTagLoading" :bordered="false"
        :row-key="(row: any) => row.id" v-model:checked-row-keys="checkedProductIds"
        :pagination="{ pageSize: 50 }" />

      <NButton block type="success" @click="handleAllocateByTags" :disabled="!waveId">
        <template #icon><NIcon><PlayOutline /></NIcon></template>
        一键分配
      </NButton>
    </div>
    <div class="flex justify-between mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
      <NButton @click="goPrev">上一步</NButton>
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>
  </NCard>
</template>
