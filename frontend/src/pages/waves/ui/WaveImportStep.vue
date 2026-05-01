<script setup lang="ts">
import { CloudUploadOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
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

const memberColumns: DataTableColumns<MemberItem> = [
  { title: '昵称', key: 'latestNickname', minWidth: 100, render: (row) => row.latestNickname || row.platformUid },
  { title: '平台', key: 'platform', width: 100 },
  { title: 'UID', key: 'platformUid', minWidth: 100 },
  { title: '等级', key: 'extraData', width: 100, render: (row) => {
      try { const ed = JSON.parse(row.extraData); return ed.giftLevel || '-' }
      catch { return '-' }
    }
  },
  { title: '地址数', key: 'activeAddressCount', width: 70 },
]

const waveProducts = ref<{ id: number; name: string; factorySku: string }[]>([])
const productTotal = ref(0)
const productPage = ref(1)
const productPageSize = ref(10)
const isProductLoading = ref(false)

const productDataColumns: DataTableColumns = [
  { title: '商品名', key: 'name', minWidth: 140 },
  { title: 'SKU', key: 'factorySku', minWidth: 120 },
]

async function loadWaveProducts() {
  if (!waveId.value) return
  isProductLoading.value = true
  try {
    const result = await listProductsWithTags(waveId.value, '', productPage.value, productPageSize.value)
    waveProducts.value = result.items.map(item => ({ id: item.id, name: item.name, factorySku: item.factorySku || '' }))
    productTotal.value = result.total
  } catch (e) { console.error('加载波次商品失败', e) }
  finally { isProductLoading.value = false }
}

function handleProductPageChange(p: number) { productPage.value = p; loadWaveProducts() }

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
  try { await importToWave(waveId.value, productCsvPath.value, productTemplateId.value); message.success('商品导入完成'); productPage.value = 1; await loadWaveProducts(); await loadWaveMembers() }
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

onMounted(async () => {
  await loadTemplates()
  await loadWaveMembers()
  await loadWaveProducts()
})
</script>
<template>
  <div class="h-full flex flex-col">
    <!-- 标题栏 -->
    <div class="flex items-center gap-2 shrink-0 px-1 py-2">
      <NIcon size="18"><CloudUploadOutline /></NIcon>
      <span class="font-semibold text-sm">步骤一：导入数据</span>
    </div>

    <!-- 主内容区 — 双列，填满剩余高度 -->
    <div class="flex-1 min-h-0 grid gap-4 md:grid-cols-2">
      <!-- 商品导入面板 -->
      <div class="border border-gray-100 dark:border-gray-700 rounded-lg flex flex-col min-h-0">
        <div class="p-3 pb-0 shrink-0">
          <span class="text-xs text-gray-500 block mb-3 font-medium">商品导入（工厂平台 {{ productFileExt }}）</span>
          <NFlex :wrap="false" class="mb-2">
            <NButton v-if="productTemplateId" size="small" secondary @click="handlePickProductFile">选择 {{ productFileExt }}</NButton>
            <span class="text-xs text-gray-400 self-center truncate max-w-[200px]">{{ productCsvPath || '未选择文件' }}</span>
          </NFlex>
          <NSelect v-model:value="productTemplateId" :options="productTemplates" placeholder="选择商品导入模板" class="mb-2" />
          <NButton block secondary @click="handleImportProduct">导入商品</NButton>
        </div>
        <div v-if="waveProducts.length" class="flex-1 min-h-0 overflow-auto px-3 pb-3">
          <NDataTable :columns="productDataColumns" :data="waveProducts" :loading="isProductLoading" :bordered="false" :pagination="false" size="small" class="mt-2" />
          <div class="flex justify-center mt-2">
            <NPagination :page="productPage" :page-size="productPageSize" :item-count="productTotal" size="small" @update:page="handleProductPageChange" />
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
        <div v-if="waveMembers.length" class="flex-1 min-h-0 overflow-auto px-3 pb-3">
          <NDataTable :columns="memberColumns" :data="waveMembers" :loading="isMembersLoading" :bordered="false" :pagination="{ pageSize: 10 }" size="small" class="mt-2" />
        </div>
        <div v-else class="flex-1" />
      </div>
    </div>

    <!-- 底部导航 — 永远右下角 -->
    <div class="flex justify-end shrink-0 pt-3 pb-1">
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>
  </div>
</template>
