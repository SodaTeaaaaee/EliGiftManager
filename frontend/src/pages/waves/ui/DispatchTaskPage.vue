<!--
  此文件已拆分为以下组件（2026-05-01）：
  - DispatchTaskShell.vue  壳组件（波次列表 + 抽屉 + 创建弹窗 + 步骤导航 + RouterView）
  - WaveWelcomePage.vue    欢迎页 / 空状态
  - WaveImportStep.vue     步骤一：导入数据
  - WaveTagStep.vue        步骤二：Tag 管理与分配
  - WavePreviewStep.vue    步骤三：导出预览与编辑
  - WaveExportStep.vue     步骤四：异常检查与导出

  路由配置见 frontend/src/app/router/index.ts
-->
<script setup lang="ts">
import { AddOutline, CloudUploadOutline, DownloadOutline, PlayOutline, RefreshOutline, TrashOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NDrawer, NDrawerContent, NEmpty, NFlex, NForm, NFormItem, NIcon, NInput, NInputNumber, NModal, NSelect, NStep, NSteps, NTabPane, NTabs, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { allocateByTags, assignProductTag, bindDefaultAddresses, createWave, deleteWave, exportOrderCSV, importDispatchWave, importToWave, isWailsRuntimeAvailable, listDispatchRecords, listProductsWithTags, listTemplates, listWaveMembers, listWaves, pickCSVFile, pickZIPFile, previewExport, removeProductTag, WAILS_PREVIEW_MESSAGE, type DispatchRecordItem, type MemberItem, type TemplateItem, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const waves = ref<WaveItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const templates = ref<TemplateItem[]>([])
const selectedWaveId = ref<number | null>(null)
const showWaveDrawer = ref(false)
const showCreateTaskModal = ref(false)
const newTaskName = ref('')
const csvPath = ref('')
const productCsvPath = ref('')
const importTemplateId = ref<number | null>(null)
const productTemplateId = ref<number | null>(null)
const exportTemplateId = ref<number | null>(null)
const isBindingAddresses = ref(false)
const isLoading = ref(false)
const isCreating = ref(false)
const errorMessage = ref('')
const waveStatusFilter = ref('active')

// Step 2 — Tag management
const tagProducts = ref<{ id: number; name: string; factorySku: string; platform: string; tags: string[] }[]>([])
const tagProductTotal = ref(0)
const tagProductPage = ref(1)
const checkedProductIds = ref<number[]>([])
const selectedBatchTag = ref<string | null>(null)
const isTagLoading = ref(false)

// Wave member list
const waveMembers = ref<MemberItem[]>([])
const isMembersLoading = ref(false)

// Step 3 — Export preview & edit
const quantityEdits = ref<Record<number, number>>({})
const previewExportResult = ref<{ totalRecords: number; missingAddressCount: number } | null>(null)
const isPreviewLoading = ref(false)

const selectedWave = computed(() => waves.value.find((wave) => wave.id === selectedWaveId.value) ?? null)
const pendingAddressCount = computed(() => selectedWave.value?.pendingAddressRecords ?? 0)
const importTemplates = computed(() => templates.value.filter((template) => template.type.startsWith('import_')).map(toOption))
const productTemplates = computed(() => templates.value.filter((template) => template.type === 'import_product').map(toOption))
const dispatchTemplates = computed(() => templates.value.filter((template) => template.type === 'import_dispatch_record').map(toOption))
const exportTemplates = computed(() => templates.value.filter((template) => template.type === 'export_order').map(toOption))
const currentStep = computed(() =>
  selectedWave.value?.status === 'exported' ? 4 :
    selectedWave.value?.status === 'pending_address' ? 3 :
      selectedWave.value?.status === 'allocating' ? 2 : 1)
const filteredWaves = computed(() => {
  if (waveStatusFilter.value === 'active') return waves.value.filter(w => w.status !== 'exported')
  if (waveStatusFilter.value === 'done') return waves.value.filter(w => w.status === 'exported')
  return waves.value
})

type LevelTag = { platform: string; tagName: string }
const waveLevelTags = computed<LevelTag[]>(() => {
  if (!selectedWave.value?.levelTags) return []
  try { return JSON.parse(selectedWave.value.levelTags) as LevelTag[] }
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

const waveColumns: DataTableColumns<WaveItem> = [
  { title: '任务编号', key: 'waveNo', minWidth: 120 },
  { title: '任务名称', key: 'name', minWidth: 180 },
  { title: '状态', key: 'status', width: 120, render: (row) => h(NTag, { type: statusTagType(row.status), size: 'small', round: true }, { default: () => statusLabel(row.status) }) },
  { title: '记录', key: 'totalRecords', width: 70 },
  { title: 'ID', key: 'id', width: 50 },
  {
    title: '', key: 'actions', width: 50, render: (_row, index) =>
      h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: (e: MouseEvent) => { e.stopPropagation(); handleDeleteWave(filteredWaves.value[index].id) } }, {
        icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
      }),
  },
]

const recordColumns: DataTableColumns<DispatchRecordItem> = [
  { title: '会员', key: 'memberNickname', minWidth: 160, render: (row) => row.memberNickname || row.platformUid },
  { title: '平台', key: 'platform', width: 100 },
  { title: '礼物', key: 'productName', minWidth: 180 },
  { title: '数量', key: 'quantity', width: 80 },
  { title: '地址', key: 'hasAddress', width: 110, render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => row.hasAddress ? '已绑定' : '待补全' }) },
  { title: '收件信息', key: 'address', minWidth: 260, ellipsis: { tooltip: true }, render: (row) => row.address || '-' },
]

const previewColumns: DataTableColumns<DispatchRecordItem> = [
  { title: '订单号', key: 'platformUid', minWidth: 160 },
  { title: '收件人', key: 'recipientName', minWidth: 120, render: (row) => row.recipientName || row.memberNickname || '-' },
  { title: '礼物', key: 'productName', minWidth: 180 },
  {
    title: '数量', key: 'quantity', width: 130, render: (row) => {
      const recordId = row.id
      const currentValue = quantityEdits.value[recordId] !== undefined ? quantityEdits.value[recordId] : row.quantity
      return h(NInputNumber, { value: currentValue, size: 'small', min: 1, onUpdateValue: (v: number | null) => { if (v !== null) quantityEdits.value[recordId] = v }, onClick: (e: MouseEvent) => e.stopPropagation() })
    }
  },
  { title: '地址', key: 'hasAddress', width: 80, render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => row.hasAddress ? '✓' : '✗' }) },
]

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

const memberColumns: DataTableColumns<MemberItem> = [
  { title: '昵称', key: 'latestNickname', minWidth: 140, render: (row) => row.latestNickname || row.platformUid },
  { title: '平台', key: 'platform', width: 100 },
  { title: 'UID', key: 'platformUid', minWidth: 140 },
  {
    title: '等级', key: 'extraData', width: 120, render: (row) => {
      try { const ed = JSON.parse(row.extraData); return ed.giftLevel || '-' }
      catch { return '-' }
    }
  },
  { title: '地址数', key: 'activeAddressCount', width: 80 },
]

function toOption(template: TemplateItem) { return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id } }
function statusLabel(status: string) { return ({ draft: '草稿', allocating: '分配中', pending_address: '待补全', exported: '已导出' } as Record<string, string>)[status] ?? status }
function statusTagType(status: string) { return status === 'exported' ? 'success' : status === 'pending_address' ? 'warning' : status === 'allocating' ? 'info' : 'default' }
async function guardRuntime() { if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false } return true }

async function loadAll() { if (!(await guardRuntime())) return; isLoading.value = true; errorMessage.value = ''; try { const [waveRows, templateRows] = await Promise.all([listWaves(), listTemplates()]); waves.value = waveRows; templates.value = templateRows; if (!selectedWaveId.value && waveRows.length > 0) { selectedWaveId.value = waveRows[0].id; showWaveDrawer.value = false } await loadRecords(); if (selectedWaveId.value) await loadTagProducts() } catch (error) { console.error(error); errorMessage.value = '加载发货任务失败。' } finally { isLoading.value = false } }
async function loadRecords() { records.value = selectedWaveId.value ? await listDispatchRecords(selectedWaveId.value) : [] }

async function loadTagProducts() {
  if (!selectedWaveId.value) return
  isTagLoading.value = true
  try {
    const result = await listProductsWithTags(selectedWaveId.value!, '', tagProductPage.value, 50)
    tagProducts.value = result.items.map(item => ({
      id: item.id, name: item.name, factorySku: item.factorySku,
      platform: item.platform, tags: item.tags,
    }))
    tagProductTotal.value = result.total
  } catch (error) { console.error('加载商品标签失败', error) }
  finally { isTagLoading.value = false }
}

async function loadWaveMembers() {
  if (!selectedWaveId.value) return
  isMembersLoading.value = true
  try {
    waveMembers.value = await listWaveMembers(selectedWaveId.value)
  } catch (error) { console.error('加载波次会员失败', error) }
  finally { isMembersLoading.value = false }
}

function handleSelectWave(row: WaveItem) {
  selectedWaveId.value = row.id
  showWaveDrawer.value = false
  loadRecords()
  loadTagProducts()
  loadWaveMembers()
}

async function handleCreate() {
  isCreating.value = true
  try {
    const wave = await createWave(newTaskName.value || '新发货任务')
    message.success(`已创建发货任务：${wave.waveNo}`)
    records.value = []
    productCsvPath.value = ''
    csvPath.value = ''
    newTaskName.value = ''; showCreateTaskModal.value = false; await loadAll()
    selectedWaveId.value = wave.id; await loadRecords(); await loadTagProducts()
  } catch (error) { message.error(String(error)) }
  finally { isCreating.value = false }
}

function templateFormat(templateId: number | null): string {
  if (!templateId) return 'csv'
  const t = templates.value.find(t => t.id === templateId)
  if (!t) return 'csv'
  try { const rules = JSON.parse(t.mappingRules); return rules.format || 'csv' }
  catch { return 'csv' }
}
const productFileExt = computed(() => templateFormat(productTemplateId.value) === 'zip' ? 'ZIP' : 'CSV')
async function handlePickCSV() { const p = await pickCSVFile(); if (p) csvPath.value = p }
async function handlePickProductFile() {
  const fmt = templateFormat(productTemplateId.value)
  const p = fmt === 'zip' ? await pickZIPFile() : await pickCSVFile()
  if (p) productCsvPath.value = p
}

async function handleImportProduct() {
  if (!selectedWaveId.value || !productCsvPath.value || !productTemplateId.value) return message.warning('请选择商品 CSV 文件和导入模板')
  try { await importToWave(selectedWaveId.value, productCsvPath.value, productTemplateId.value); message.success('商品导入完成'); await loadAll() }
  catch (error) { message.error(String(error)) }
}

async function handleImport() {
  if (!selectedWaveId.value || !importTemplateId.value) return message.warning('请选择发货任务和导入模板')
  try { await importToWave(selectedWaveId.value, csvPath.value, importTemplateId.value); message.success('任务数据导入完成'); await loadAll() }
  catch (error) { message.error(String(error)) }
}

async function handleImportDispatch() {
  if (!selectedWaveId.value || !csvPath.value || !importTemplateId.value) return message.warning('请选择发货任务、填写 CSV 路径、并选择导入模板')
  try { await importDispatchWave(selectedWaveId.value, csvPath.value, importTemplateId.value); message.success('发货数据导入完成'); await loadWaveMembers(); await loadAll() }
  catch (error) { message.error(String(error)) }
}

async function handleAssignTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await assignProductTag(productId, platform, tagName) }
    message.success(`已为 ${checkedProductIds.value.length} 件商品打上 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (error) { message.error(String(error)) }
}

async function handleBatchRemoveTag() {
  if (!selectedBatchTag.value || checkedProductIds.value.length === 0) return message.warning('请选择商品和 Tag')
  const [platform, tagName] = selectedBatchTag.value.split('|')
  try {
    for (const productId of checkedProductIds.value) { await removeProductTag(productId, platform, tagName) }
    message.success(`已为 ${checkedProductIds.value.length} 件商品移除 ${platform}·${tagName} 标签`)
    await loadTagProducts(); checkedProductIds.value = []
  } catch (error) { message.error(String(error)) }
}

async function handleRemoveTag(productId: number, platform: string, tagName: string) {
  try { await removeProductTag(productId, platform, tagName); await loadTagProducts() }
  catch (error) { message.error(String(error)) }
}

async function handleAllocateByTags() {
  if (!selectedWaveId.value) return
  try { const count = await allocateByTags(selectedWaveId.value); message.success(`Tag 分配完成，共 ${count} 条记录`); await loadAll() }
  catch (error) { message.error(String(error)) }
}

async function handlePreviewExport() {
  if (!selectedWaveId.value) return message.warning('请选择发货任务')
  isPreviewLoading.value = true
  try {
    previewExportResult.value = await previewExport(selectedWaveId.value)
    await loadRecords()
  } catch (error) { message.error(String(error)) }
  finally { isPreviewLoading.value = false }
}

async function handleBindAddresses() {
  if (!selectedWaveId.value) return message.warning('请先选择发货任务')
  isBindingAddresses.value = true
  try { const result = await bindDefaultAddresses(selectedWaveId.value); message.success('补全完成：' + result.updated + ' 条已绑定默认地址，' + result.skipped + ' 条无默认地址跳过'); await loadAll(); await loadRecords() }
  catch (error) { message.error(String(error)) }
  finally { isBindingAddresses.value = false }
}

async function handleExport() {
  if (!selectedWaveId.value || !exportTemplateId.value) return message.warning('请选择导出模板')
  try {
    const preview = await previewExport(selectedWaveId.value)
    if (preview.missingAddressCount > 0) { message.warning('仍有 ' + preview.missingAddressCount + ' 条记录缺失地址，请先补全后再导出'); return }
    const path = await exportOrderCSV(selectedWaveId.value, exportTemplateId.value); message.success(`清单已导出：${path}`); await loadAll()
  } catch (error) { message.error(String(error)) }
}
async function handleDeleteWave(waveId: number) {
  try {
    await deleteWave(waveId)
    message.success('已删除发货任务')
    if (selectedWaveId.value === waveId) {
      selectedWaveId.value = null
      records.value = []
      waveMembers.value = []
      tagProducts.value = []
    }
    await loadAll()
  } catch (error) { message.error(String(error)) }
}

onMounted(loadAll)
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Dispatch Task</p>
        <h1 class="app-title mt-2">发货任务管理</h1>
        <p class="app-copy mt-2">创建发货任务，导入外部名单，按 Tag 匹配自动分配并导出发货清单。</p>
      </div>
      <div class="flex gap-2">
        <NButton @click="showWaveDrawer = true">发货任务列表</NButton>
        <NButton type="primary" @click="showCreateTaskModal = true"><template #icon>
            <NIcon>
              <AddOutline />
            </NIcon>
          </template>新建发货任务
        </NButton>
        <NButton :loading="isLoading" secondary strong @click="loadAll"><template #icon>
            <NIcon>
              <RefreshOutline />
            </NIcon>
          </template>刷新</NButton>
      </div>
    </header>

    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>

    <!-- Wave Drawer -->
    <NDrawer v-model:show="showWaveDrawer" :width="840">
      <NDrawerContent title="发货任务列表" closable>
        <NTabs v-model:value="waveStatusFilter" size="small" class="mb-3">
          <NTabPane name="active" tab="进行中" />
          <NTabPane name="done" tab="已完成" />
          <NTabPane name="all" tab="全部" />
        </NTabs>
        <NDataTable :columns="waveColumns" :data="filteredWaves" :loading="isLoading" :bordered="false"
          :row-key="(row: WaveItem) => row.id"
          :row-props="(row: WaveItem) => ({ class: row.id === selectedWaveId ? 'cursor-pointer bg-slate-50 dark:bg-slate-800/60' : 'cursor-pointer', onClick: () => handleSelectWave(row) })" />
      </NDrawerContent>
    </NDrawer>

    <div v-if="selectedWave" class="space-y-4">
      <div class="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800/60 rounded-lg">
        <span class="font-semibold text-lg">{{ selectedWave.waveNo }} · {{ selectedWave.name }}</span>
        <NTag :type="statusTagType(selectedWave.status)" size="small" round>{{ statusLabel(selectedWave.status) }}
        </NTag>
      </div>

      <NSteps :current="currentStep" status="process" class="mb-3">
        <NStep title="导入数据" />
        <NStep title="Tag 管理与分配" />
        <NStep title="导出预览与编辑" />
        <NStep title="异常检查与导出" />
      </NSteps>

      <!-- Step 1: Import -->
      <NCard size="small">
        <template #header><span class="flex items-center gap-2">
            <NIcon>
              <CloudUploadOutline />
            </NIcon>步骤一：导入数据
          </span></template>
        <div class="grid gap-4 md:grid-cols-2">
          <div class="p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
            <span class="text-xs text-gray-500 block mb-3 font-medium">商品导入（工厂平台 {{ productFileExt }}）</span>
            <NFlex :wrap="false" class="mb-2">
              <NButton size="small" secondary @click="handlePickProductFile">选择 {{ productFileExt }}</NButton><span
                class="text-xs text-gray-400 self-center truncate max-w-[200px]">{{ productCsvPath || '未选择文件' }}</span>
            </NFlex>
            <NSelect v-model:value="productTemplateId" :options="productTemplates" placeholder="选择商品导入模板"
              class="mb-2" />
            <NButton block secondary @click="handleImportProduct">导入商品</NButton>
          </div>
          <div class="p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
            <span class="text-xs text-gray-500 block mb-3 font-medium">发货数据导入（会员来源 CSV）</span>
            <NFlex :wrap="false" class="mb-2">
              <NButton size="small" secondary @click="handlePickCSV">选择 CSV</NButton><span
                class="text-xs text-gray-400 self-center truncate max-w-[200px]">{{ csvPath || '未选择文件' }}</span>
            </NFlex>
            <NSelect v-model:value="importTemplateId" :options="dispatchTemplates" placeholder="选择发货导入模板"
              class="mb-2" />
            <NButton block type="primary" @click="handleImportDispatch">导入发货数据</NButton>
          </div>
        </div>
        <div v-if="waveMembers.length > 0" class="mt-4">
          <span class="text-xs text-gray-500 block mb-2 font-medium">本次导入的会员（{{ waveMembers.length }} 人）</span>
          <NDataTable :columns="memberColumns" :data="waveMembers" :loading="isMembersLoading" :bordered="false"
            :pagination="{ pageSize: 10 }" size="small" />
        </div>
      </NCard>

      <!-- Step 2: Tag Management -->
      <NCard size="small">
        <template #header><span class="flex items-center gap-2">
            <NIcon>
              <PlayOutline />
            </NIcon>步骤二：Tag 管理与分配
          </span></template>
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

          <NButton block type="success" @click="handleAllocateByTags" :disabled="!selectedWaveId">
            <template #icon>
              <NIcon>
                <PlayOutline />
              </NIcon>
            </template>
            一键分配
          </NButton>
        </div>
      </NCard>

      <!-- Step 3: Export Preview & Edit -->
      <NCard size="small">
        <template #header><span class="flex items-center gap-2">
            <NIcon>
              <DownloadOutline />
            </NIcon>步骤三：导出预览与编辑
          </span></template>
        <div class="space-y-3">
          <div class="flex items-center gap-3 p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
            <NSelect v-model:value="exportTemplateId" :options="exportTemplates" placeholder="选择导出模板"
              style="width: 240px" />
            <NButton type="primary" :loading="isPreviewLoading" @click="handlePreviewExport">
              <template #icon>
                <NIcon>
                  <PlayOutline />
                </NIcon>
              </template>
              预览导出数据
            </NButton>
          </div>
          <NAlert v-if="previewExportResult" type="info" :show-icon="false">
            共 {{ previewExportResult.totalRecords }} 条记录
            <span v-if="previewExportResult.missingAddressCount > 0">，{{ previewExportResult.missingAddressCount }}
              条缺失地址</span>
          </NAlert>
          <NDataTable :columns="previewColumns" :data="records" :bordered="false" :pagination="{ pageSize: 10 }" />
        </div>
      </NCard>

      <!-- Step 4: Export & Records -->
      <NCard size="small">
        <template #header><span class="flex items-center gap-2"><span>步骤四：异常检查与导出</span>
            <NTag v-if="pendingAddressCount > 0" type="warning" size="small" round>{{ pendingAddressCount }} 条待补全</NTag>
          </span></template>
        <template #header-extra>
          <NButton v-if="selectedWave && pendingAddressCount > 0" size="small" type="warning"
            :loading="isBindingAddresses" @click="handleBindAddresses">一键补全默认地址</NButton>
        </template>
        <NDataTable :columns="recordColumns" :data="records" :bordered="false" :pagination="{ pageSize: 10 }"
          class="mb-4" />
        <div class="p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
          <span class="text-xs text-gray-500 block mb-2 font-medium">导出发货清单</span>
          <NSelect v-model:value="exportTemplateId" :options="exportTemplates" placeholder="选择导出模板" class="mb-2" />
          <NButton block type="success" @click="handleExport">
            <template #icon>
              <NIcon>
                <DownloadOutline />
              </NIcon>
            </template>
            生成发货清单
          </NButton>
        </div>
      </NCard>
    </div>

    <NEmpty v-else description="点击「发货任务列表」选择或新建一个发货任务" />

    <NModal v-model:show="showCreateTaskModal" preset="card" title="新建发货任务" style="max-width: 520px">
      <NForm label-placement="top">
        <NFormItem label="任务名称">
          <NInput v-model:value="newTaskName" placeholder="例如：4月24日直播礼物发货" @keyup.enter="handleCreate" />
        </NFormItem>
        <NButton type="primary" block :loading="isCreating" @click="handleCreate">创建发货任务</NButton>
      </NForm>
    </NModal>
  </section>
</template>
