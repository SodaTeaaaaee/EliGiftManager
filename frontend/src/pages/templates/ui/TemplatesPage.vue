<script setup lang="ts">
import { AddOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NTag,
  type DataTableColumns,
} from 'naive-ui'
import {
  isWailsRuntimeAvailable,
  listTemplates,
  WAILS_PREVIEW_MESSAGE,
  type TemplateItem,
} from '@/shared/lib/wails/app'

const router = useRouter()
const templates = ref<TemplateItem[]>([])
const errorMessage = ref('')

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
  const typeOptions: Record<string, string> = {
    import_product: '礼物导入模板',
    import_dispatch_record: '会员数据导入模板',
    export_order: '发货清单导出模板',
  }
  return typeOptions[type] ?? type
}
function openCreateModal(platform?: string) {
  router.push({ name: 'templates-builder' })
}
function openEditModal(row: TemplateItem) {
  router.push({ name: 'templates-builder', query: { id: row.id } })
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
onMounted(async () => {
  await loadTemplates()
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
      <div class="flex gap-2">
        <NButton type="primary" @click="openCreateModal()"
          ><template #icon
            ><NIcon><AddOutline /></NIcon></template
          >新建模板</NButton
        >
        <NButton secondary @click="router.push({ name: 'templates-builder' })"
          >自定义构建器</NButton
        >
      </div>
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
  </section>
</template>
