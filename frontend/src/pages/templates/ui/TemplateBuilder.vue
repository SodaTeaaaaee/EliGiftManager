<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { NButton, NInput, NSelect, NSwitch, useMessage } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import {
  createTemplate,
  listTemplates,
  pickCSVFile,
  previewCSVSample,
  updateTemplate,
} from '@/shared/lib/wails/app'
import { usePlatformCatalog } from '@/shared/composables/usePlatformCatalog'
import BasicMapper from './BasicMapper.vue'
import AdvancedEditor from './AdvancedEditor.vue'
import type {
  DynamicExportRules,
  DynamicTemplateRules,
  ExportColumnMapping,
  DynamicFieldMapping,
} from './types'

const message = useMessage()
const router = useRouter()
const route = useRoute()
const editingId = ref<number | null>(null)

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

async function handleSave() {
  if (!templateName.value.trim()) {
    message.warning('请输入模板名称')
    return
  }
  if (!templatePlatform.value.trim()) {
    message.warning('请选择平台')
    return
  }
  try {
    const mappingRules =
      templateType.value === 'export_order'
        ? JSON.stringify(exportConfig)
        : JSON.stringify(templateConfig)
    const typeParam = templateType.value
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
        :options="platformOptions"
        placeholder="选择平台"
        filterable
        style="max-width: 160px"
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

    <div class="flex items-center gap-3">
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
        @upload="handleUploadCSV"
      />
      <AdvancedEditor v-else :template-config="templateConfig" :sample-rows="sampleRows" />
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

    <div class="flex gap-2 pt-2">
      <NButton type="primary" @click="handleSave">保存模板</NButton>
      <NButton secondary @click="handleCancel">取消</NButton>
    </div>
  </section>
</template>
