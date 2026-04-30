<script setup lang="ts">
import { AddOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NCard, NDataTable, NEmpty, NForm, NFormItem, NIcon, NInput, NModal, NRadio, NRadioGroup, NSelect, NTabPane, NTabs, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { createTemplate, isWailsRuntimeAvailable, listDefaultTemplates, listTemplates, WAILS_PREVIEW_MESSAGE, type TemplateItem } from '@/shared/lib/wails/app'

const message = useMessage()
const templates = ref<TemplateItem[]>([])
const errorMessage = ref('')
const activePlatform = ref('all')
const showCreateModal = ref(false)
const createMode = ref<'preset' | 'custom'>('preset')
const defaultTemplates = ref<TemplateItem[]>([])
const presetIndex = ref<number | null>(null)
const isSaving = ref(false)
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
const columns: DataTableColumns<TemplateItem> = [
  { title: '名称', key: 'name', minWidth: 180 },
  { title: '平台', key: 'platform', width: 110, render: (row) => row.platform || '通用' },
  { title: '类型', key: 'type', width: 170, render: (row) => h(NTag, { type: row.type === 'allocation' ? 'info' : 'default', size: 'small', round: true }, { default: () => typeLabel(row.type) }) },
  { title: '映射规则', key: 'mappingRules', minWidth: 320, ellipsis: { tooltip: true } },
]
function typeLabel(type: string) { return typeOptions.find((item) => item.value === type)?.label ?? type }
function openCreateModal() { createMode.value = 'preset'; presetIndex.value = null; form.platform = activePlatform.value !== 'all' && activePlatform.value !== '通用' ? activePlatform.value : ''; form.type = 'import_member'; form.name = ''; form.mappingRules = '{\n  \n}'; showCreateModal.value = true }
async function loadTemplates() { if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return } try { templates.value = await listTemplates() } catch (error) { console.error(error); errorMessage.value = '加载模板失败。' } }
async function loadDefaultTemplates() { try { defaultTemplates.value = await listDefaultTemplates() } catch (error) { console.error(error) } }
async function handleCreateTemplate() {
  isSaving.value = true
  try {
    let template: TemplateItem
    if (createMode.value === 'preset') {
      const preset = selectedPreset.value
      if (!preset) { message.error('请先选择一个预设模板'); return }
      template = await createTemplate(preset.platform, preset.type, preset.name, preset.mappingRules)
    } else {
      template = await createTemplate(form.platform, form.type, form.name, form.mappingRules)
    }
    message.success(`已创建模板：${template.name}`)
    showCreateModal.value = false
    await loadTemplates()
    activePlatform.value = template.platform || 'all'
  } catch (error) { message.error(String(error)) } finally { isSaving.value = false }
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
    <NCard v-else size="medium">
      <NTabs v-model:value="activePlatform" type="segment">
        <NTabPane v-for="platform in platforms" :key="platform" :name="platform"
          :tab="platform === 'all' ? '全部平台' : platform">
          <div class="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
            <NDataTable :columns="columns" :data="filteredTemplates" :bordered="false" :scroll-x="900"
              :pagination="{ pageSize: 10 }" />
            <NCard embedded title="匹配规则模板" size="small">
              <div v-if="allocationTemplates.length" class="space-y-3">
                <div v-for="template in allocationTemplates" :key="template.id"
                  class="rounded-xl border border-slate-200 p-3 dark:border-slate-700">
                  <div class="flex items-center justify-between"><strong>{{ template.name }}</strong>
                    <NTag size="small" type="info" round>{{ template.platform }}</NTag>
                  </div>
                  <NInput class="mt-3" type="textarea" readonly :value="template.mappingRules"
                    :autosize="{ minRows: 5 }" />
                </div>
              </div>
              <NEmpty v-else description="当前平台暂无匹配规则模板" />
            </NCard>
          </div>
        </NTabPane>
      </NTabs>
    </NCard>
    <NModal v-model:show="showCreateModal" preset="card" title="新建模板" style="max-width: 620px">
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
        <NButton type="primary" block :loading="isSaving" @click="handleCreateTemplate">创建模板</NButton>
      </NForm>
    </NModal>
  </section>
</template>
