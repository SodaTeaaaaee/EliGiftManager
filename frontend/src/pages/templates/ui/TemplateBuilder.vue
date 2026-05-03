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
import BasicMapper from './BasicMapper.vue'
import AdvancedEditor from './AdvancedEditor.vue'
import type { DynamicTemplateRules, DynamicFieldMapping } from './types'

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
        const rules = JSON.parse(t.mappingRules)
        Object.assign(templateConfig, rules)
      }
    } catch (e) {
      message.error('加载模板失败')
    }
  }
})

const platformOptions = [
  { label: 'BILIBILI', value: 'BILIBILI' },
  { label: 'DOUYIN', value: 'DOUYIN' },
  { label: 'KUAISHOU', value: 'KUAISHOU' },
  { label: 'XIAOHONGSHU', value: 'XIAOHONGSHU' },
  { label: 'WEIBO', value: 'WEIBO' },
  { label: 'ACFUN', value: 'ACFUN' },
  { label: 'YOUTUBE', value: 'YOUTUBE' },
  { label: 'TWITCH', value: 'TWITCH' },
  { label: 'OTHER', value: 'OTHER' },
]

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
    if (editingId.value) {
      await updateTemplate(
        editingId.value,
        templatePlatform.value,
        'import_dispatch_record',
        templateName.value,
        JSON.stringify(templateConfig),
      )
    } else {
      await createTemplate(
        templatePlatform.value,
        'import_dispatch_record',
        templateName.value,
        JSON.stringify(templateConfig),
      )
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

    <div class="flex items-center gap-3">
      <NInput v-model:value="templateName" placeholder="模板名称" style="max-width: 240px" />
      <NSelect
        v-model:value="templatePlatform"
        :options="platformOptions"
        placeholder="选择平台"
        style="max-width: 180px"
      />
    </div>

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

    <div class="flex gap-2 pt-2">
      <NButton type="primary" @click="handleSave">保存模板</NButton>
      <NButton secondary @click="handleCancel">取消</NButton>
    </div>
  </section>
</template>
