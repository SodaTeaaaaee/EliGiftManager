<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { NAlert, NButton, NInput, NSelect, NSwitch, useDialog, useMessage } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import {
  createTemplate,
  listTemplates,
  pickCSVFile,
  pickDataFile,
  pickFolder,
  previewArchive,
  previewCSVSample,
  updateTemplate,
  validateTemplate,
  type TemplateValidationResult,
} from '@/shared/lib/wails/app'
import { usePlatformCatalog } from '@/shared/composables/usePlatformCatalog'
import BasicMapper, { type FieldGroup } from './BasicMapper.vue'
import AdvancedEditor from './AdvancedEditor.vue'
import type {
  DynamicExportRules,
  DynamicTemplateRules,
  DynamicFieldMapping,
} from './types'

const message = useMessage()
const dialog = useDialog()
const router = useRouter()
const route = useRoute()
const editingId = ref<number | null>(null)
const validationResult = ref<TemplateValidationResult | null>(null)

onMounted(async () => {
  const idStr = route.query.id
  if (idStr) {
    editingId.value = Number(idStr)
    try {
      const all = await listTemplates()
      const t = all.find((t) => t.id === editingId.value)
      if (t) {
        templateName.value = t.name
        templatePlatform.value = t.platform
        if (t.type === 'export_order') templateType.value = 'export_order'
        const rules = JSON.parse(t.mappingRules)
        if (templateType.value === 'export_order') {
          Object.assign(exportConfig, rules)
        } else {
          Object.assign(templateConfig, rules)
        }
      }
    } catch (e) {
      message.error('加载模板失败')
    }
  } else if (route.query.platform) {
    templatePlatform.value = String(route.query.platform)
  }
})

const { platformOptions, addPlatform, getPlatform } = usePlatformCatalog()

const platformOptionsTyped = computed(() =>
  platformOptions.value.map((o) => ({
    ...o,
    label: `${o.label} · ${o.type === 'member' ? '会员平台' : '工厂平台'}`,
  })),
)
const showAddPlatform = ref(false)
const newPlatformName = ref('')
const newPlatformType = ref<'member' | 'factory'>('member')

function handleAddPlatform() {
  if (addPlatform(newPlatformName.value, newPlatformType.value)) {
    templatePlatform.value = newPlatformName.value
    newPlatformName.value = ''
    showAddPlatform.value = false
  }
}

const templateConfig = reactive<DynamicTemplateRules>({
  format: 'csv',
  hasHeader: true,
  mapping: {
    platform_uid: { sourceColumn: '', required: true },
    gift_level: { sourceColumn: '', required: true },
    nickname: { sourceColumn: '', required: false },
    recipient_name: { sourceColumn: '', required: false },
    phone: { sourceColumn: '', required: false },
    address: { sourceColumn: '', required: false },
  },
  extraData: { strategy: 'catch_all' },
})

const isAdvanced = ref(false)
const templateName = ref('')
const templatePlatform = ref('')
const templateType = ref<'import_product' | 'import_dispatch_record' | 'export_order'>('import_dispatch_record')

const typeOptions = [
  { label: '礼物导入模板', value: 'import_product' },
  { label: '会员数据导入模板', value: 'import_dispatch_record' },
  { label: '发货清单导出模板', value: 'export_order' },
]

const typeShortLabel: Record<string, string> = {
  import_product: '礼物导入',
  import_dispatch_record: '会员数据导入',
  export_order: '发货清单导出',
}

const memberFieldGroups: FieldGroup[] = [
  {
    label: '核心身份',
    fields: [
      { key: 'platform_uid', label: '平台 UID', required: true },
      { key: 'gift_level', label: '等级', required: true },
      { key: 'nickname', label: '昵称' },
    ],
  },
  {
    label: '发货信息（可选）',
    fields: [
      { key: 'recipient_name', label: '收件人' },
      { key: 'phone', label: '电话' },
      { key: 'address', label: '地址' },
    ],
  },
]

const productFieldGroups: FieldGroup[] = [
  {
    label: '商品信息',
    fields: [
      { key: 'name', label: '商品名称', required: true },
      { key: 'factory_sku', label: 'SKU/编码', required: true },
    ],
  },
  {
    label: '补充信息（可选）',
    fields: [
      { key: 'factory', label: '工厂名' },
      { key: 'cover_image', label: '封面图路径' },
    ],
  },
]

const csvPattern = ref('*.csv')
const imageDir = ref('主图')
const productSampleFile = ref('')
const archivePreview = ref<{ extractDir: string; csvFiles: string[]; dirs: { name: string; fileCount: number }[] } | null>(null)
const selectedArchiveCSV = ref('')
const productFormat = computed<'csv' | 'zip'>(() => {
  if (!productSampleFile.value) return 'zip'
  return productSampleFile.value.toLowerCase().endsWith('.csv') ? 'csv' : 'zip'
})

async function loadArchiveCSV(extractDir: string, csvRelPath: string) {
  const fullPath = extractDir.replace(/[/\\]$/, '') + '/' + csvRelPath
  sampleRows.value = await previewCSVSample(fullPath)
}

async function handlePickProductFile() {
  const path = await pickDataFile()
  if (!path) return
  try {
    const lower = path.toLowerCase()
    if (lower.endsWith('.csv')) {
      sampleRows.value = await previewCSVSample(path)
      archivePreview.value = null
      selectedArchiveCSV.value = ''
    } else {
      const preview = await previewArchive(path)
      archivePreview.value = preview
      selectedArchiveCSV.value = ''
      sampleRows.value = []
      if (preview!.csvFiles.length === 1) {
        await loadArchiveCSV(preview!.extractDir, preview!.csvFiles[0])
        selectedArchiveCSV.value = preview!.csvFiles[0]
      }
    }
    productSampleFile.value = path.split(/[/\\]/).pop() || path
  } catch (e) {
    message.error('读取文件失败: ' + String(e))
  }
}
async function handlePickProductDir() {
  const dir = await pickFolder()
  if (!dir) return
  productSampleFile.value = dir
  archivePreview.value = { extractDir: dir, csvFiles: [], dirs: [] }
  selectedArchiveCSV.value = ''
  sampleRows.value = []
}

async function selectArchiveCSV(csvRelPath: string) {
  if (!archivePreview.value) return
  selectedArchiveCSV.value = csvRelPath
  await loadArchiveCSV(archivePreview.value.extractDir, csvRelPath)
}

// Clear state when platform or type changes
watch([templatePlatform, templateType], () => {
  sampleRows.value = []
  productSampleFile.value = ''
  archivePreview.value = null
  selectedArchiveCSV.value = ''
  csvPattern.value = '*.csv'
  imageDir.value = '主图'
})

// Auto-generate default name: "{platform} {typeShortLabel}"
watch([templatePlatform, templateType], ([plat, type]) => {
  if (!templateName.value && plat && type) {
    templateName.value = `${plat} ${typeShortLabel[type] || type}`
  }
})

const filteredTypeOptions = computed(() => {
  const p = getPlatform(templatePlatform.value)
  if (!p) return typeOptions
  if (p.type === 'member') {
    return typeOptions.filter((o) => o.value === 'import_dispatch_record')
  }
  return typeOptions.filter((o) => o.value === 'import_product' || o.value === 'export_order')
})

watch(templatePlatform, (newPlatform) => {
  if (!newPlatform) return
  const p = getPlatform(newPlatform)
  if (!p) return
  if (p.type === 'member' && templateType.value !== 'import_dispatch_record') {
    templateType.value = 'import_dispatch_record'
  } else if (
    p.type === 'factory' &&
    templateType.value === 'import_dispatch_record'
  ) {
    templateType.value = 'import_product'
  }
})

const valueTypeOptions = [
  { label: '订单号', value: 'order_no' },
  { label: '收件人', value: 'recipient' },
  { label: '手机号', value: 'phone' },
  { label: '收件地址', value: 'address' },
  { label: 'SKU', value: 'sku' },
  { label: '数量', value: 'quantity' },
  { label: '固定值', value: 'static' },
  { label: '会员UID', value: 'member_uid' },
  { label: '会员昵称', value: 'member_nickname' },
]

const exportConfig = reactive<DynamicExportRules>({
  format: 'csv',
  hasHeader: true,
  columns: [
    { headerName: '订单号', valueType: 'order_no', prefix: 'ROUZAO-' },
    { headerName: '收件人', valueType: 'recipient' },
    { headerName: '手机号', valueType: 'phone' },
    { headerName: '收件地址', valueType: 'address' },
    { headerName: 'SKU', valueType: 'sku' },
    { headerName: '数量', valueType: 'quantity' },
  ],
})

// CSV upload state (hoisted so AdvancedEditor can use sampleRows)
const sampleRows = ref<string[][]>([])
const csvHeaders = computed(() => sampleRows.value[0] || [])

async function handleUploadCSV() {
  const path = await pickCSVFile()
  if (!path) return
  try {
    sampleRows.value = await previewCSVSample(path)
  } catch (e) {
    message.error('读取 CSV 失败: ' + String(e))
  }
}

function onAdvancedChange(val: boolean) {
  if (!val && templateConfig.extraData.strategy !== 'catch_all') {
    message.warning('切换回基础模式将丢失高级 ExtraData 设置')
  }
}

function buildMappingRules(): string {
  if (templateType.value === 'export_order') {
    return JSON.stringify(exportConfig)
  } else if (templateType.value === 'import_product' && productFormat.value === 'zip') {
    const zipCfg = { ...templateConfig, format: 'zip', csvPattern: csvPattern.value, imageDir: imageDir.value }
    return JSON.stringify(zipCfg)
  } else {
    templateConfig.format = 'csv'
    return JSON.stringify(templateConfig)
  }
}

async function performSave(typeParam: string, mappingRules: string) {
  try {
    if (editingId.value) {
      await updateTemplate(
        editingId.value,
        templatePlatform.value,
        typeParam,
        templateName.value,
        mappingRules,
      )
    } else {
      await createTemplate(templatePlatform.value, typeParam, templateName.value, mappingRules)
    }
    message.success(editingId.value ? '模板已更新' : '模板已保存')
    router.push({ name: 'templates' })
  } catch (e) {
    message.error(String(e))
  }
}

async function handleSave() {
  if (!templateName.value.trim()) {
    message.warning('请输入模板名称')
    return
  }
  if (!templatePlatform.value.trim()) {
    message.warning('请选择平台')
    return
  }
  const mappingRules = buildMappingRules()
  const typeParam = templateType.value
  validationResult.value = null
  try {
    const result = await validateTemplate(typeParam, mappingRules)
    validationResult.value = result
    if (result.errors?.length) {
      return
    }
    if (result.warnings?.length) {
      dialog.warning({
        title: '模板校验警告',
        content: () =>
          result.warnings
            .map((w: string) => `• ${w}`)
            .join('\n'),
        positiveText: '仍然保存',
        negativeText: '返回修改',
        onPositiveClick: () => {
          validationResult.value = null
          performSave(typeParam, mappingRules)
        },
      })
      return
    }
  } catch (e) {
    message.error('模板校验失败：' + String(e))
    return
  }
  await performSave(typeParam, mappingRules)
}

function handleCancel() {
  router.push({ name: 'templates' })
}

const DEFAULT_MAPPING: Record<string, DynamicFieldMapping> = {
  platform_uid: { sourceColumn: '', required: true },
  gift_level: { sourceColumn: '', required: true },
  nickname: { sourceColumn: '', required: false },
  recipient_name: { sourceColumn: '', required: false },
  phone: { sourceColumn: '', required: false },
  address: { sourceColumn: '', required: false },
}

watch(
  () => templateConfig.mapping,
  (mapping) => {
    for (const key of Object.keys(DEFAULT_MAPPING)) {
      if (!mapping[key]) {
        mapping[key] = { ...DEFAULT_MAPPING[key] }
      }
    }
  },
  { deep: true },
)

watch(
  () => templateConfig.extraData,
  (ed) => {
    if (!ed || !ed.strategy) {
      templateConfig.extraData = { strategy: 'catch_all' }
    }
  },
)
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Templates</p>
      <h1 class="app-title mt-2">自定义模板构建器</h1>
      <p class="app-copy mt-2">
        通过 CSV 示例文件快速建立字段映射，或切换到高级模式直接编辑 JSON 规则。
      </p>
    </header>

    <div class="flex items-center gap-2">
      <NInput v-model:value="templateName" placeholder="模板名称" style="max-width: 240px" />
      <NSelect
        v-model:value="templatePlatform"
        :options="platformOptionsTyped"
        placeholder="选择平台"
        filterable
        style="max-width: 210px"
      />
      <NButton size="small" secondary @click="showAddPlatform = !showAddPlatform">+</NButton>
    </div>
    <div v-if="showAddPlatform" class="flex items-center gap-2">
      <NInput
        v-model:value="newPlatformName"
        placeholder="新平台名称"
        size="small"
        style="max-width: 120px"
        @keyup.enter="handleAddPlatform"
      />
      <NSelect
        v-model:value="newPlatformType"
        :options="[
          { label: '会员平台', value: 'member' },
          { label: '工厂平台', value: 'factory' },
        ]"
        size="small"
        style="width: 110px"
      />
      <NButton size="small" type="primary" @click="handleAddPlatform">添加</NButton>
      <NButton size="small" @click="showAddPlatform = false">取消</NButton>
    </div>

    <div v-if="templatePlatform" class="flex items-center gap-3">
      <span class="text-sm">模板类型：</span>
      <NSelect v-model:value="templateType" :options="filteredTypeOptions" style="width: 200px" />
    </div>

    <div v-if="templateType === 'import_dispatch_record'">
      <div class="flex items-center gap-2">
        <NSwitch v-model:value="isAdvanced" @update:value="onAdvancedChange" />
        <span class="text-sm" :class="isAdvanced ? 'text-[var(--primary)]' : ''">
          {{ isAdvanced ? '高级模式' : '基础模式' }}
        </span>
      </div>

      <BasicMapper
        v-if="!isAdvanced"
        :template-config="templateConfig"
        :csv-headers="csvHeaders"
        :field-groups="memberFieldGroups"
        @upload="handleUploadCSV"
      />
      <AdvancedEditor v-else :template-config="templateConfig" :sample-rows="sampleRows" />
    </div>

    <div v-if="templateType === 'import_product'" class="space-y-4">
      <div class="flex items-center gap-3">
        <NButton @click="handlePickProductFile">选择文件</NButton>
        <NButton @click="handlePickProductDir">选择文件夹</NButton>
        <span v-if="productSampleFile" class="text-xs text-gray-400">{{ productSampleFile }}</span>
      </div>

      <!-- Archive/directory structure: CSV selector + dirs -->
      <div v-if="archivePreview && archivePreview.csvFiles.length > 1" class="border rounded p-3 space-y-2">
        <p class="text-sm font-medium">选择 CSV 文件</p>
        <div v-for="csv in archivePreview.csvFiles" :key="csv"
          class="flex items-center gap-2 py-1 px-2 rounded cursor-pointer hover:bg-gray-50"
          :class="selectedArchiveCSV === csv ? 'bg-blue-50' : ''"
          @click="selectArchiveCSV(csv)">
          <span class="text-sm">{{ csv }}</span>
          <NTag v-if="selectedArchiveCSV === csv" size="tiny" type="info">已选</NTag>
        </div>
        <div v-if="archivePreview.dirs.length" class="pt-2 border-t">
          <p class="text-xs text-gray-500 mb-1">目录结构</p>
          <div v-for="d in archivePreview.dirs" :key="d.name" class="text-xs text-gray-400">
            {{ d.name }}/（{{ d.fileCount }} 个文件）
          </div>
        </div>
      </div>

      <!-- Auto-selected single CSV notice -->
      <div v-else-if="archivePreview && archivePreview.csvFiles.length === 1 && selectedArchiveCSV" class="text-xs text-gray-400">
        已自动选择 CSV: {{ selectedArchiveCSV }}
      </div>

      <BasicMapper
        v-if="csvHeaders.length"
        :template-config="templateConfig"
        :csv-headers="csvHeaders"
        :field-groups="productFieldGroups"
        @upload="handleUploadCSV"
      />

      <div v-if="csvHeaders.length || productSampleFile" class="flex items-center gap-4">
        <div>
          <label class="text-xs text-gray-500 block mb-0.5">CSV 文件名模式</label>
          <NInput v-model:value="csvPattern" size="small" style="width: 160px" />
        </div>
        <div>
          <label class="text-xs text-gray-500 block mb-0.5">图片目录（可选）</label>
          <NSelect
            v-model:value="imageDir"
            :options="archivePreview?.dirs.map(d => ({ label: `${d.name}/（${d.fileCount} 文件）`, value: d.name })) || []"
            placeholder="选择目录"
            size="small"
            clearable
            filterable
            tag
            style="width: 200px"
          />
        </div>
      </div>

      <div v-if="archivePreview" class="border rounded p-2 text-xs space-y-1">
        <div v-for="d in archivePreview.dirs" :key="d.name">
          {{ d.name }}/（{{ d.fileCount }} 个文件）
        </div>
      </div>
    </div>

    <div v-if="templateType === 'export_order'" class="space-y-3">
      <p class="text-sm font-medium">导出列配置（顺序即 CSV 列顺序）</p>
      <div
        v-for="(col, i) in exportConfig.columns"
        :key="i"
        class="flex items-center gap-2 p-2 border rounded"
      >
        <span class="text-xs text-gray-400 w-4">{{ i + 1 }}</span>
        <NInput
          v-model:value="col.headerName"
          placeholder="表头名称"
          size="small"
          style="width: 120px"
        />
        <NSelect
          v-model:value="col.valueType"
          :options="valueTypeOptions"
          size="small"
          style="width: 130px"
        />
        <NInput
          v-if="col.valueType === 'order_no'"
          v-model:value="col.prefix"
          placeholder="前缀"
          size="small"
          style="width: 100px"
        />
        <NInput
          v-if="col.valueType === 'static'"
          v-model:value="col.defaultValue"
          placeholder="固定值"
          size="small"
          style="width: 100px"
        />
        <NButton size="tiny" quaternary type="error" @click="exportConfig.columns.splice(i, 1)"
          >✕</NButton
        >
        <NButton
          size="tiny"
          quaternary
          @click="
            exportConfig.columns.splice(i, 0, {
              headerName: '',
              valueType: 'static',
              defaultValue: '',
            })
          "
        >
          ↑插入
        </NButton>
      </div>
      <NButton
        size="small"
        secondary
        @click="
          exportConfig.columns.push({ headerName: '', valueType: 'static', defaultValue: '' })
        "
      >
        + 添加列
      </NButton>
    </div>

    <NAlert
      v-if="validationResult?.errors.length"
      type="error"
      class="mb-2"
      title="模板校验失败，请修正后重试："
    >
      <ul class="list-disc ml-4">
        <li v-for="(err, i) in validationResult.errors" :key="i">{{ err }}</li>
      </ul>
    </NAlert>
    <div class="flex gap-2 pt-2">
      <NButton type="primary" @click="handleSave">保存模板</NButton>
      <NButton secondary @click="handleCancel">取消</NButton>
    </div>
  </section>
</template>
