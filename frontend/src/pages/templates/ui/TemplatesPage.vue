<script setup lang="ts">
import { AddOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NEmpty, NForm, NFormItem, NIcon, NInput, NModal, NRadio, NRadioGroup, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { createTemplate, isWailsRuntimeAvailable, listDefaultTemplates, listTemplates, updateTemplate, WAILS_PREVIEW_MESSAGE, type TemplateItem } from '@/shared/lib/wails/app'

const message = useMessage()
const templates = ref<TemplateItem[]>([])
const errorMessage = ref('')
const activePlatform = ref('all')
const showCreateModal = ref(false)
const createMode = ref<'preset' | 'custom'>('preset')
const defaultTemplates = ref<TemplateItem[]>([])
const presetIndex = ref<number | null>(null)
const isSaving = ref(false)
const editingTemplate = ref<TemplateItem | null>(null)
const modalTitle = computed(() => editingTemplate.value ? '编辑模板' : '新建模板')
const form = reactive({ platform: '', type: 'import_member', name: '', mappingRules: '{\n  \n}' })
const typeOptions = [
  { label: '会员导入模板', value: 'import_member' },
  { label: '礼物导入模板', value: 'import_product' },
  { label: '派发记录导入模板', value: 'import_dispatch_record' },
  { label: '发货清单导出模板', value: 'export_order' },
  { label: '匹配规则模板', value: 'allocation' },
]
const presetOptions = computed(() => defaultTemplates.value.map((t, i) => ({ label: `${t.platform || '通用'} / ${t.name}`, value: i })))
const selectedPreset = computed(() => presetIndex.value !== null ? defaultTemplates.value[presetIndex.value] ?? null : null)
const platforms = computed(() => ['all', ...Array.from(new Set(templates.value.map((template) => template.platform || '通用')))])
const filteredTemplates = computed(() => activePlatform.value === 'all' ? templates.value : templates.value.filter((template) => (template.platform || '通用') === activePlatform.value))
const allocationTemplates = computed(() => filteredTemplates.value.filter((template) => template.type === 'allocation'))
const platformTemplates = (p: string) => templates.value.filter(t => (t.platform || '通用') === p)
const hasImportProduct = (p: string) => platformTemplates(p).some(t => t.type === 'import_product')
const hasExportOrder = (p: string) => platformTemplates(p).some(t => t.type === 'export_order')
const columns: DataTableColumns<TemplateItem> = [
  { title: '名称', key: 'name', minWidth: 180 },
  { title: '平台', key: 'platform', width: 110, render: (row) => row.platform || '通用' },
  { title: '类型', key: 'type', width: 170, render: (row) => h(NTag, { type: row.type === 'allocation' ? 'info' : 'default', size: 'small', round: true }, { default: () => typeLabel(row.type) }) },
  { title: '', key: 'actions', width: 60, render: (row) =>
    h(NButton, { size: 'tiny', secondary: true, onClick: (e: MouseEvent) => { e.stopPropagation(); openEditModal(row) } }, { default: () => '编辑' })
  },
]
function typeLabel(type: string) { return typeOptions.find((item) => item.value === type)?.label ?? type }
function openCreateModal(platform?: string) {
  editingTemplate.value = null
  createMode.value = 'preset'
  presetIndex.value = null
  form.platform = platform || ''
  form.type = 'import_member'
  form.name = ''
  form.mappingRules = '{\n  \n}'
  showCreateModal.value = true
}

function openEditModal(template: TemplateItem) {
  editingTemplate.value = template
  form.platform = template.platform
  form.type = template.type
  form.name = template.name
  form.mappingRules = template.mappingRules
  createMode.value = 'custom'
  showCreateModal.value = true
}
async function loadTemplates() { if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return } try { templates.value = await listTemplates() } catch (error) { console.error(error); errorMessage.value = '加载模板失败。' } }
async function loadDefaultTemplates() { try { defaultTemplates.value = await listDefaultTemplates() } catch (error) { console.error(error) } }
async function handleSaveTemplate() {
  isSaving.value = true
  try {
    if (editingTemplate.value) {
      await updateTemplate(editingTemplate.value.id, form.platform, form.type, form.name, form.mappingRules)
      message.success('模板已更新')
    } else if (createMode.value === 'preset') {
      const preset = selectedPreset.value
      if (!preset) { message.error('请先选择一个预设模板'); return }
      await createTemplate(preset.platform, preset.type, preset.name, preset.mappingRules)
      message.success(`已创建模板：${preset.name}`)
    } else {
      const template = await createTemplate(form.platform, form.type, form.name, form.mappingRules)
      message.success(`已创建模板：${template.name}`)
    }
    showCreateModal.value = false
    await loadTemplates()
  } catch (error) { message.error(String(error)) }
  finally { isSaving.value = false }
}
onMounted(async () => { await loadTemplates(); await loadDefaultTemplates() })
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Templates</p>
        <h1 class="app-title mt-2">模板设置</h1>
        <p class="app-copy mt-2">模板必须绑定平台；匹配规则模板用于建立“外部业务字段 -> 内部产品 ID”。</p>
      </div>
      <NButton type="primary" @click="openCreateModal"><template #icon>
          <NIcon>
            <AddOutline />
          </NIcon>
        </template>新建模板
      </NButton>
    </header>
    <NEmpty v-if="errorMessage" :description="errorMessage" />
    <div v-else-if="platforms.filter(p => p !== 'all').length" class="grid gap-4 xl:grid-cols-2">
      <NCard v-for="platform in platforms.filter(p => p !== 'all')" :key="platform" size="small">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-semibold">{{ platform }}</span>
            <NButton size="small" type="primary" @click="openCreateModal(platform)">添加模板</NButton>
          </div>
        </template>

        <!-- 缺失导出模板警告 -->
        <NAlert
          v-if="hasImportProduct(platform) && !hasExportOrder(platform)"
          type="warning"
          :show-icon="false"
          class="mb-3"
        >
          此平台有商品导入模板但缺少订单导出模板，建议补全。
        </NAlert>

        <NDataTable
          v-if="platformTemplates(platform).length"
          :columns="columns"
          :data="platformTemplates(platform)"
          :bordered="false"
          :pagination="false"
          size="small"
        />
        <NEmpty v-else description="暂无模板" />
      </NCard>
    </div>
    <NEmpty v-else description="暂无模板，点击上方「新建模板」开始" />
    <NModal v-model:show="showCreateModal" preset="card" :title="modalTitle" style="max-width: 620px">
      <NForm label-placement="top">
        <NFormItem label="创建方式">
          <NRadioGroup v-model:value="createMode">
            <NRadio value="preset">从预设添加</NRadio>
            <NRadio value="custom">自定义</NRadio>
          </NRadioGroup>
        </NFormItem><template v-if="createMode === 'preset'">
          <NFormItem label="选择预设模板">
            <NSelect v-model:value="presetIndex" :options="presetOptions" placeholder="选择预设模板" />
          </NFormItem>
          <NFormItem v-if="selectedPreset" label="模板预览">
            <NCard size="small">
              <pre
                class="text-xs whitespace-pre-wrap">{{ JSON.stringify(JSON.parse(selectedPreset.mappingRules), null, 2) }}</pre>
            </NCard>
          </NFormItem>
        </template><template v-else>
          <NFormItem label="平台">
            <NInput v-model:value="form.platform" placeholder="例如：抖音 / 快手" />
          </NFormItem>
          <NFormItem label="模板类型">
            <NSelect v-model:value="form.type" :options="typeOptions" />
          </NFormItem>
          <NFormItem label="模板名称">
            <NInput v-model:value="form.name" placeholder="输入模板名称" />
          </NFormItem>
          <NFormItem label="映射规则 JSON">
            <NInput v-model:value="form.mappingRules" type="textarea" :autosize="{ minRows: 8 }"
              placeholder='例如：{"platform_uid":"用户ID"}' />
          </NFormItem>
        </template>
        <NButton type="primary" block :loading="isSaving" @click="handleSaveTemplate">{{ editingTemplate ? '保存修改' : '创建模板' }}</NButton>
      </NForm>
    </NModal>
  </section>
</template>
