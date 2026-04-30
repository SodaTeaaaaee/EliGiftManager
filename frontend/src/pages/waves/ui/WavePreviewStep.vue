<script setup lang="ts">
import { DownloadOutline, PlayOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NAlert, NButton, NDataTable, NIcon, NInputNumber, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { isWailsRuntimeAvailable, listDispatchRecords, listTemplates, previewExport, WAILS_PREVIEW_MESSAGE, type DispatchRecordItem, type TemplateItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const templates = ref<TemplateItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const exportTemplateId = ref<number | null>(null)
const quantityEdits = ref<Record<number, number>>({})
const previewExportResult = ref<{ totalRecords: number; missingAddressCount: number } | null>(null)
const isPreviewLoading = ref(false)
const errorMessage = ref('')

const exportTemplates = computed(() => templates.value.filter(t => t.type === 'export_order').map(toOption))

function toOption(template: TemplateItem) {
  return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id }
}

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

async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false }
  return true
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

async function handlePreviewExport() {
  if (!waveId.value) return message.warning('请选择发货任务')
  isPreviewLoading.value = true
  try {
    previewExportResult.value = await previewExport(waveId.value)
    await loadRecords()
  } catch (e) { message.error(String(e)) }
  finally { isPreviewLoading.value = false }
}

function goPrev() {
  router.push({ name: 'waves-step-tags', params: { waveId: String(waveId.value) } })
}
function goNext() {
  router.push({ name: 'waves-step-export', params: { waveId: String(waveId.value) } })
}

onMounted(async () => {
  await loadTemplates()
  await loadRecords()
})
</script>
<template>
  <NCard size="small">
    <template #header>
      <span class="flex items-center gap-2">
        <NIcon><DownloadOutline /></NIcon>步骤三：导出预览与编辑
      </span>
    </template>
    <div class="space-y-3">
      <div class="flex items-center gap-3 p-3 border border-gray-100 dark:border-gray-700 rounded-lg">
        <NSelect v-model:value="exportTemplateId" :options="exportTemplates" placeholder="选择导出模板" style="width: 240px" />
        <NButton type="primary" :loading="isPreviewLoading" @click="handlePreviewExport">
          <template #icon><NIcon><PlayOutline /></NIcon></template>
          预览导出数据
        </NButton>
      </div>
      <NAlert v-if="previewExportResult" type="info" :show-icon="false">
        共 {{ previewExportResult.totalRecords }} 条记录
        <span v-if="previewExportResult.missingAddressCount > 0">，{{ previewExportResult.missingAddressCount }} 条缺失地址</span>
      </NAlert>
      <NDataTable :columns="previewColumns" :data="records" :bordered="false" :pagination="{ pageSize: 10 }" />
    </div>
    <div class="flex justify-between mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
      <NButton @click="goPrev">上一步</NButton>
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>
  </NCard>
</template>
