<script setup lang="ts">
import { DownloadOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NButton, NDataTable, NIcon, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { bindDefaultAddresses, exportOrderCSV, isWailsRuntimeAvailable, listDispatchRecords, listTemplates, listWaves, previewExport, WAILS_PREVIEW_MESSAGE, type DispatchRecordItem, type TemplateItem, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const wave = ref<WaveItem | null>(null)
const templates = ref<TemplateItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const exportTemplateId = ref<number | null>(null)
const isBindingAddresses = ref(false)
const errorMessage = ref('')

const exportTemplates = computed(() => templates.value.filter(t => t.type === 'export_order').map(toOption))
const pendingAddressCount = computed(() => wave.value?.pendingAddressRecords ?? 0)

function toOption(template: TemplateItem) {
  return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id }
}

const recordColumns: DataTableColumns<DispatchRecordItem> = [
  { title: '会员', key: 'memberNickname', minWidth: 160, render: (row) => row.memberNickname || row.platformUid },
  { title: '平台', key: 'platform', width: 100 },
  { title: '礼物', key: 'productName', minWidth: 180 },
  { title: '数量', key: 'quantity', width: 80 },
  { title: '地址', key: 'hasAddress', width: 110, render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => row.hasAddress ? '已绑定' : '待补全' }) },
  { title: '收件信息', key: 'address', minWidth: 260, ellipsis: { tooltip: true }, render: (row) => row.address || '-' },
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

async function loadTemplates() {
  if (!(await guardRuntime())) return
  try { templates.value = await listTemplates() }
  catch (e) { console.error('加载模板失败', e) }
}

async function loadRecords() {
  if (!waveId.value) return
  try { records.value = await listDispatchRecords(waveId.value) }
  catch (e) { console.error('加载发货记录失败', e) }
}

async function handleBindAddresses() {
  if (!waveId.value) return message.warning('请先选择发货任务')
  isBindingAddresses.value = true
  try {
    const result = await bindDefaultAddresses(waveId.value)
    message.success('补全完成：' + result.updated + ' 条已绑定默认地址，' + result.skipped + ' 条无默认地址跳过')
    await loadWave()
    await loadRecords()
  } catch (e) { message.error(String(e)) }
  finally { isBindingAddresses.value = false }
}

async function handleExport() {
  if (!waveId.value || !exportTemplateId.value) return message.warning('请选择导出模板')
  try {
    const preview = await previewExport(waveId.value)
    if (preview.missingAddressCount > 0) {
      message.warning('仍有 ' + preview.missingAddressCount + ' 条记录缺失地址，请先补全后再导出')
      return
    }
    const path = await exportOrderCSV(waveId.value, exportTemplateId.value)
    message.success(`清单已导出：${path}`)
    await loadWave()
  } catch (e) { message.error(String(e)) }
}

function goPrev() {
  router.push({ name: 'waves-step-preview', params: { waveId: String(waveId.value) } })
}

onMounted(async () => {
  await loadWave()
  await loadTemplates()
  await loadRecords()
})
</script>
<template>
  <NCard size="small">
    <template #header>
      <span class="flex items-center gap-2">
        <span>步骤四：异常检查与导出</span>
        <NTag v-if="pendingAddressCount > 0" type="warning" size="small" round>{{ pendingAddressCount }} 条待补全</NTag>
      </span>
    </template>
    <template #header-extra>
      <NButton v-if="wave && pendingAddressCount > 0" size="small" type="warning"
        :loading="isBindingAddresses" @click="handleBindAddresses">一键补全默认地址</NButton>
    </template>
    <NDataTable :columns="recordColumns" :data="records" :bordered="false" :pagination="{ pageSize: 10 }" class="mb-4" />
    <div class="p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
      <span class="text-xs text-gray-500 block mb-2 font-medium">导出发货清单</span>
      <NSelect v-model:value="exportTemplateId" :options="exportTemplates" placeholder="选择导出模板" class="mb-2" />
      <NButton block type="success" @click="handleExport">
        <template #icon><NIcon><DownloadOutline /></NIcon></template>
        生成发货清单
      </NButton>
    </div>
    <div class="flex justify-between mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
      <NButton @click="goPrev">上一步</NButton>
      <div></div>
    </div>
  </NCard>
</template>
