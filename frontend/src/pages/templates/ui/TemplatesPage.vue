<script setup lang="ts">
import { AddOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioGroup,
  NSelect,
  NSwitch,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import {
  createTemplate,
  isWailsRuntimeAvailable,
  listDefaultTemplates,
  listTemplates,
  updateTemplate,
  WAILS_PREVIEW_MESSAGE,
  type TemplateItem,
} from '@/shared/lib/wails/app'

const message = useMessage()
const templates = ref<TemplateItem[]>([])
const errorMessage = ref('')
const showCreateModal = ref(false)
const createMode = ref<'preset' | 'custom'>('preset')
const defaultTemplates = ref<TemplateItem[]>([])
const presetIndex = ref<number | null>(null)
const isSaving = ref(false)
const editingTemplate = ref<TemplateItem | null>(null)
const modalTitle = computed(() => (editingTemplate.value ? '编辑模板' : '新建模板'))
const form = reactive({
  platform: '',
  type: 'import_dispatch_record',
  name: '',
  mappingRules: '{\n  \n}',
})
const useGuiEditor = ref(true)
const showCloseConfirm = ref(false)
let formSnapshot = ''

function snapshotForm() {
  formSnapshot = JSON.stringify({
    platform: form.platform,
    type: form.type,
    name: form.name,
    mappingRules: form.mappingRules,
  })
}
function hasFormChanged() {
  if (createMode.value === 'preset') return false
  if (useGuiEditor.value) buildJsonFromGui()
  return (
    JSON.stringify({
      platform: form.platform,
      type: form.type,
      name: form.name,
      mappingRules: form.mappingRules,
    }) !== formSnapshot
  )
}
function requestClose() {
  if (!hasFormChanged()) {
    showCreateModal.value = false
    return
  }
  showCloseConfirm.value = true
}
function discardAndClose() {
  showCloseConfirm.value = false
  showCreateModal.value = false
}
async function saveAndClose() {
  showCloseConfirm.value = false
  await handleSaveTemplate()
}

// ---- GUI editor state ----
const guiHasHeader = ref(false)
const guiMemberFields = ref<{ name: string; col: number; required: boolean }[]>([])
const guiProductFormat = ref<'csv' | 'zip'>('csv')
const guiCsvPattern = ref('*.csv')
const guiImageDir = ref('')
const guiProductFields = ref<{ name: string; header: string }[]>([])
const guiExportPrefix = ref('')
const guiExportHeaders = ref<string[]>([])

function syncGuiFromJson() {
  try {
    JSON.parse(form.mappingRules)
  } catch {
    return
  }
  const rules = JSON.parse(form.mappingRules)
  if (form.type === 'import_dispatch_record') {
    guiHasHeader.value = !!rules.hasHeader
    const mapping = rules.mapping || {}
    guiMemberFields.value = Object.entries(mapping as Record<string, any>).map(([name, v]) => ({
      name,
      col: v?.columnIndex ?? 0,
      required: !!v?.required,
    }))
  } else if (form.type === 'import_product') {
    guiProductFormat.value = rules.format === 'zip' ? 'zip' : 'csv'
    guiCsvPattern.value = rules.csvPattern || '*.csv'
    guiImageDir.value = rules.imageDir || ''
    const mapping = rules.mapping || {}
    guiProductFields.value = Object.entries(mapping as Record<string, string>).map(
      ([name, header]) => ({
        name,
        header: String(header),
      }),
    )
  } else if (form.type === 'export_order') {
    guiExportPrefix.value = rules.prefix || ''
    guiExportHeaders.value = Array.isArray(rules.headers) ? rules.headers : []
  }
}

function buildJsonFromGui() {
  if (form.type === 'import_dispatch_record') {
    const mapping: Record<string, any> = {}
    for (const f of guiMemberFields.value) {
      mapping[f.name] = f.required ? { columnIndex: f.col, required: true } : { columnIndex: f.col }
    }
    form.mappingRules = JSON.stringify({ hasHeader: guiHasHeader.value, mapping }, null, 2)
  } else if (form.type === 'import_product') {
    const mapping: Record<string, string> = {}
    for (const f of guiProductFields.value) mapping[f.name] = f.header
    const obj: Record<string, any> = { format: guiProductFormat.value, mapping }
    if (guiProductFormat.value === 'zip') {
      obj.csvPattern = guiCsvPattern.value
      obj.imageDir = guiImageDir.value
    }
    form.mappingRules = JSON.stringify(obj, null, 2)
  } else if (form.type === 'export_order') {
    form.mappingRules = JSON.stringify(
      { prefix: guiExportPrefix.value, headers: guiExportHeaders.value },
      null,
      2,
    )
  }
}

function onTypeChange() {
  form.mappingRules = '{}'
  syncGuiFromJson()
}
// ---- end GUI editor ----

const typeOptions = [
  { label: '礼物导入模板', value: 'import_product' },
  { label: '会员数据导入模板', value: 'import_dispatch_record' },
  { label: '发货清单导出模板', value: 'export_order' },
]
const memberFieldOptions = [
  { label: 'giftName（礼物等级）', value: 'giftName' },
  { label: 'platformUid（平台UID）', value: 'platformUid' },
  { label: 'nickname（昵称）', value: 'nickname' },
  { label: 'platform（平台）', value: 'platform' },
]
const productFieldOptions = [
  { label: 'name（商品名）', value: 'name' },
  { label: 'factorySku（商家编码）', value: 'factorySku' },
  { label: 'platform（平台）', value: 'platform' },
  { label: 'factory（工厂）', value: 'factory' },
  { label: 'coverImage（主图）', value: 'cover_image' },
]
const presetOptions = computed(() =>
  defaultTemplates.value.map((t, i) => ({
    label: `${t.platform || '通用'} / ${t.name}`,
    value: i,
  })),
)
const selectedPreset = computed(() =>
  presetIndex.value !== null ? (defaultTemplates.value[presetIndex.value] ?? null) : null,
)
const platforms = computed(() => [
  'all',
  ...Array.from(new Set(templates.value.map((template) => template.platform || '通用'))),
])
const platformTemplates = (p: string) => templates.value.filter((t) => (t.platform || '通用') === p)
const hasImportProduct = (p: string) =>
  platformTemplates(p).some((t) => t.type === 'import_product')
const hasExportOrder = (p: string) => platformTemplates(p).some((t) => t.type === 'export_order')
const columns: DataTableColumns<TemplateItem> = [
  { title: '名称', key: 'name', minWidth: 120 },
  { title: '平台', key: 'platform', width: 110, render: (row) => row.platform || '通用' },
  {
    title: '类型',
    key: 'type',
    width: 170,
    render: (row) =>
      h(
        NTag,
        { type: 'default', size: 'small', round: true },
        { default: () => typeLabel(row.type) },
      ),
  },
  {
    title: '',
    key: 'actions',
    width: 60,
    render: (row) =>
      h(
        NButton,
        {
          size: 'tiny',
          secondary: true,
          onClick: (e: MouseEvent) => {
            e.stopPropagation()
            openEditModal(row)
          },
        },
        { default: () => '编辑' },
      ),
  },
]
function typeLabel(type: string) {
  return typeOptions.find((item) => item.value === type)?.label ?? type
}
function openCreateModal(platform?: string) {
  editingTemplate.value = null
  createMode.value = 'preset'
  presetIndex.value = null
  form.platform = platform || ''
  form.type = 'import_dispatch_record'
  form.name = ''
  form.mappingRules = '{\n  \n}'
  useGuiEditor.value = true
  showCreateModal.value = true
  setTimeout(snapshotForm, 0)
}
function openEditModal(template: TemplateItem) {
  editingTemplate.value = template
  form.platform = template.platform
  form.type = template.type
  form.name = template.name
  form.mappingRules = template.mappingRules
  useGuiEditor.value = true
  syncGuiFromJson()
  createMode.value = 'custom'
  showCreateModal.value = true
  setTimeout(snapshotForm, 0)
}
async function loadTemplates() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }
  try {
    templates.value = await listTemplates()
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载模板失败。'
  }
}
async function loadDefaultTemplates() {
  try {
    defaultTemplates.value = await listDefaultTemplates()
  } catch (error) {
    console.error(error)
  }
}
async function handleSaveTemplate() {
  if (useGuiEditor.value && createMode.value === 'custom') buildJsonFromGui()
  isSaving.value = true
  try {
    if (editingTemplate.value) {
      await updateTemplate(
        editingTemplate.value.id,
        form.platform,
        form.type,
        form.name,
        form.mappingRules,
      )
      message.success('模板已更新')
    } else if (createMode.value === 'preset') {
      const preset = selectedPreset.value
      if (!preset) {
        message.error('请先选择一个预设模板')
        return
      }
      await createTemplate(preset.platform, preset.type, preset.name, preset.mappingRules)
      message.success(`已创建模板：${preset.name}`)
    } else {
      const template = await createTemplate(form.platform, form.type, form.name, form.mappingRules)
      message.success(`已创建模板：${template.name}`)
    }
    showCreateModal.value = false
    await loadTemplates()
  } catch (error) {
    message.error(String(error))
  } finally {
    isSaving.value = false
  }
}
onMounted(async () => {
  await loadTemplates()
  await loadDefaultTemplates()
})
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Templates</p>
        <h1 class="app-title mt-2">模板设置</h1>
        <p class="app-copy mt-2">
          模板必须绑定平台；匹配规则模板用于建立"外部业务字段 -> 内部产品 ID"。
        </p>
      </div>
      <NButton type="primary" @click="openCreateModal()"
        ><template #icon
          ><NIcon><AddOutline /></NIcon></template
        >新建模板</NButton
      >
    </header>
    <NEmpty v-if="errorMessage" :description="errorMessage" />
    <div v-else-if="platforms.filter((p) => p !== 'all').length" class="grid gap-4 xl:grid-cols-2">
      <NCard v-for="platform in platforms.filter((p) => p !== 'all')" :key="platform" size="small">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-semibold">{{ platform }}</span>
            <NButton size="small" type="primary" @click="openCreateModal(platform)"
              >添加模板</NButton
            >
          </div>
        </template>
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

    <!-- ====== 模板编辑弹窗 ====== -->
    <NModal
      v-model:show="showCreateModal"
      preset="card"
      :title="modalTitle"
      :mask-closable="false"
      style="max-width: 860px; width: 90vw"
    >
      <NForm label-placement="top">
        <NFormItem label="创建方式">
          <NRadioGroup v-model:value="createMode">
            <NRadio value="preset">从预设添加</NRadio>
            <NRadio value="custom">自定义</NRadio>
          </NRadioGroup>
        </NFormItem>

        <!-- 预设模式 -->
        <template v-if="createMode === 'preset'">
          <NFormItem label="选择预设模板">
            <NSelect
              v-model:value="presetIndex"
              :options="presetOptions"
              placeholder="选择预设模板"
            />
          </NFormItem>
          <NFormItem v-if="selectedPreset" label="模板预览">
            <NCard size="small">
              <pre class="text-xs whitespace-pre-wrap">{{
                JSON.stringify(JSON.parse(selectedPreset.mappingRules), null, 2)
              }}</pre>
            </NCard>
          </NFormItem>
        </template>

        <!-- 自定义模式 -->
        <template v-else>
          <NFormItem label="平台">
            <NInput v-model:value="form.platform" placeholder="例如：抖音 / 快手" />
          </NFormItem>
          <NFormItem label="模板类型">
            <NSelect
              v-model:value="form.type"
              :options="typeOptions"
              @update:value="onTypeChange"
            />
          </NFormItem>
          <NFormItem label="模板名称">
            <NInput v-model:value="form.name" placeholder="输入模板名称" />
          </NFormItem>

          <!-- 映射规则：开关 + GUI / Raw 二选一 -->
          <div class="mb-4">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium">映射规则</span>
              <div class="flex items-center gap-2">
                <span
                  class="text-xs"
                  :class="useGuiEditor ? 'text-[var(--primary)]' : 'text-gray-400'"
                  >GUI</span
                >
                <NSwitch v-model:value="useGuiEditor" size="small" />
                <span
                  class="text-xs"
                  :class="!useGuiEditor ? 'text-[var(--primary)]' : 'text-gray-400'"
                  >Raw</span
                >
              </div>
            </div>

            <!-- GUI 编辑器 -->
            <div
              v-if="useGuiEditor"
              class="border rounded-lg p-3 space-y-3 bg-gray-50/50 dark:bg-gray-800/30"
            >
              <!-- 会员导入 / 发货记录导入 -->
              <template v-if="form.type === 'import_dispatch_record'">
                <NCheckbox
                  :checked="guiHasHeader"
                  @update:checked="(v) => (guiHasHeader = v)"
                  size="small"
                  >CSV 包含表头行</NCheckbox
                >
                <div>
                  <div class="text-xs text-gray-500 mb-1">字段映射</div>
                  <div class="space-y-1">
                    <div v-for="(f, i) in guiMemberFields" :key="i" class="flex items-center gap-1">
                      <NSelect
                        v-model:value="f.name"
                        :options="memberFieldOptions"
                        size="small"
                        style="width: 130px"
                      />
                      <span class="text-xs text-gray-400 shrink-0">第</span>
                      <NInputNumber
                        v-model:value="f.col"
                        size="small"
                        :min="0"
                        style="width: 64px"
                      />
                      <span class="text-xs text-gray-400 shrink-0">列</span>
                      <NCheckbox
                        :checked="f.required"
                        @update:checked="(v) => (f.required = v)"
                        size="small"
                        class="shrink-0"
                        >必填</NCheckbox
                      >
                      <NButton
                        size="tiny"
                        quaternary
                        type="error"
                        @click="guiMemberFields.splice(i, 1)"
                        >✕</NButton
                      >
                    </div>
                  </div>
                  <NButton
                    size="tiny"
                    quaternary
                    class="mt-1"
                    @click="
                      guiMemberFields.push({
                        name: '',
                        col: guiMemberFields.length,
                        required: false,
                      })
                    "
                    >+ 添加字段</NButton
                  >
                </div>
              </template>

              <!-- 商品导入 -->
              <template v-else-if="form.type === 'import_product'">
                <div class="flex items-center gap-3">
                  <span class="text-xs text-gray-500">文件格式</span>
                  <NRadioGroup v-model:value="guiProductFormat" size="small">
                    <NRadio value="csv">CSV</NRadio>
                    <NRadio value="zip">ZIP</NRadio>
                  </NRadioGroup>
                </div>
                <div v-if="guiProductFormat === 'zip'" class="flex gap-2">
                  <div class="flex-1">
                    <div class="text-xs text-gray-500 mb-0.5">CSV 匹配</div>
                    <NInput v-model:value="guiCsvPattern" size="small" />
                  </div>
                  <div class="flex-1">
                    <div class="text-xs text-gray-500 mb-0.5">图片目录</div>
                    <NInput v-model:value="guiImageDir" size="small" />
                  </div>
                </div>
                <div>
                  <div class="text-xs text-gray-500 mb-1">字段映射</div>
                  <div class="space-y-1">
                    <div
                      v-for="(f, i) in guiProductFields"
                      :key="i"
                      class="flex items-center gap-1"
                    >
                      <NSelect
                        v-model:value="f.name"
                        :options="productFieldOptions"
                        size="small"
                        style="width: 130px"
                      />
                      <span class="text-xs text-gray-400">→</span>
                      <NInput
                        v-model:value="f.header"
                        size="small"
                        class="flex-1"
                        placeholder="CSV 表头名"
                      />
                      <NButton
                        size="tiny"
                        quaternary
                        type="error"
                        @click="guiProductFields.splice(i, 1)"
                        >✕</NButton
                      >
                    </div>
                  </div>
                  <NButton
                    size="tiny"
                    quaternary
                    class="mt-1"
                    @click="guiProductFields.push({ name: '', header: '' })"
                    >+ 添加字段</NButton
                  >
                </div>
              </template>

              <!-- 订单导出 -->
              <template v-else-if="form.type === 'export_order'">
                <div>
                  <div class="text-xs text-gray-500 mb-0.5">订单号前缀</div>
                  <NInput v-model:value="guiExportPrefix" size="small" />
                </div>
                <div>
                  <div class="text-xs text-gray-500 mb-1">导出表头（按顺序）</div>
                  <div class="space-y-1">
                    <div
                      v-for="(h, i) in guiExportHeaders"
                      :key="i"
                      class="flex items-center gap-1"
                    >
                      <span class="text-xs text-gray-400 w-4 shrink-0">{{ i + 1 }}</span>
                      <NInput v-model:value="guiExportHeaders[i]" size="small" class="flex-1" />
                      <NButton
                        size="tiny"
                        quaternary
                        type="error"
                        @click="guiExportHeaders.splice(i, 1)"
                        >✕</NButton
                      >
                    </div>
                  </div>
                  <NButton size="tiny" quaternary class="mt-1" @click="guiExportHeaders.push('')"
                    >+ 添加表头</NButton
                  >
                </div>
              </template>

              <!-- allocation / 其他 → 降级提示 -->
              <template v-else>
                <p class="text-xs text-gray-400">
                  此模板类型暂无 GUI 编辑器，请切换到 Raw 模式编辑。
                </p>
              </template>
            </div>

            <!-- Raw JSON -->
            <NInput
              v-else
              v-model:value="form.mappingRules"
              type="textarea"
              :autosize="{ minRows: 6 }"
              placeholder='例如：{"platform_uid":"用户ID"}'
            />
          </div>
        </template>

        <div class="flex gap-2 mt-2">
          <NButton secondary @click="requestClose">关闭</NButton>
          <NButton type="primary" :loading="isSaving" class="flex-1" @click="handleSaveTemplate">{{
            editingTemplate ? '保存修改' : '创建模板'
          }}</NButton>
        </div>
      </NForm>
    </NModal>

    <!-- 未保存确认 -->
    <NModal
      v-model:show="showCloseConfirm"
      preset="card"
      title="未保存的更改"
      :mask-closable="false"
      style="max-width: 400px"
    >
      <p class="app-copy mb-4">当前有未保存的修改，如何处理？</p>
      <div class="flex gap-2">
        <NButton size="small" secondary @click="showCloseConfirm = false">取消</NButton>
        <NButton size="small" type="warning" @click="discardAndClose">忽略修改</NButton>
        <NButton size="small" type="primary" @click="saveAndClose">保存并关闭</NButton>
      </div>
    </NModal>
  </section>
</template>
