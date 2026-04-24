<script setup lang="ts">
import { CodeSlashOutline, LayersOutline, SearchOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NCard, NDataTable, NEmpty, NIcon, NInput, NTag, type DataTableColumns } from 'naive-ui'
import { WAILS_PREVIEW_MESSAGE, isWailsRuntimeAvailable, listTemplates, type TemplateItem } from '@/shared/lib/wails/app'

const templates = ref<TemplateItem[]>([])
const keyword = ref('')
const isLoading = ref(false)
const errorMessage = ref('')

const typeLabels: Record<string, string> = {
  import_member: '会员导入',
  import_product: '商品导入',
  import_dispatch_record: '派发导入',
  export_order: '订单导出',
}

const filteredTemplates = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return templates.value
  return templates.value.filter((template) =>
    [template.name, template.type, template.mappingRules].some((value) => value.toLowerCase().includes(text)),
  )
})

const typeCount = computed(() => new Set(templates.value.map((template) => template.type)).size)

const columns: DataTableColumns<TemplateItem> = [
  { title: '模板名称', key: 'name', minWidth: 180 },
  {
    title: '类型',
    key: 'type',
    minWidth: 150,
    render: (row) => h(NTag, { type: 'info', size: 'small', round: true }, { default: () => typeLabels[row.type] ?? row.type }),
  },
  { title: 'Mapping Rules', key: 'mappingRules', minWidth: 360, ellipsis: { tooltip: true } },
  { title: '更新时间', key: 'updatedAt', minWidth: 170, render: (row) => new Date(row.updatedAt).toLocaleString() },
]

async function loadTemplates() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''
  try {
    templates.value = await listTemplates()
  } catch (error) {
    console.error('加载模板失败', error)
    errorMessage.value = '加载模板配置失败，请查看后端日志。'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadTemplates)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Templates</p>
        <h1 class="app-title mt-2">导入导出模板</h1>
        <p class="app-copy mt-2">直接展示 template_configs 表，模板类型与后端常量保持一致。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadTemplates">刷新模板</NButton>
    </header>

    <div class="grid gap-4 md:grid-cols-3">
      <NCard><p class="app-copy">模板总数</p><p class="mt-1 text-2xl font-semibold">{{ templates.length }}</p></NCard>
      <NCard><p class="app-copy">覆盖类型</p><p class="mt-1 text-2xl font-semibold">{{ typeCount }}</p></NCard>
      <NCard><p class="app-copy">标准类型</p><p class="mt-1 text-2xl font-semibold">4</p></NCard>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1fr_1.4fr]">
      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2"><NIcon><LayersOutline /></NIcon><span>模板类型</span></div>
        </template>
        <div class="space-y-3">
          <div v-for="(label, type) in typeLabels" :key="type" class="flex items-center justify-between rounded-xl border border-slate-200/70 p-3 dark:border-slate-700/70">
            <div>
              <p class="font-semibold">{{ label }}</p>
              <p class="app-copy">{{ type }}</p>
            </div>
            <NTag :type="templates.some((template) => template.type === type) ? 'success' : 'default'" round>
              {{ templates.filter((template) => template.type === type).length }} 个
            </NTag>
          </div>
        </div>
      </NCard>

      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2"><NIcon><CodeSlashOutline /></NIcon><span>模板配置</span></div>
        </template>
        <template #header-extra>
          <NInput v-model:value="keyword" clearable placeholder="搜索模板或字段映射" style="width: 260px">
            <template #prefix><NIcon><SearchOutline /></NIcon></template>
          </NInput>
        </template>
        <NEmpty v-if="errorMessage" :description="errorMessage" />
        <NDataTable v-else :columns="columns" :data="filteredTemplates" :loading="isLoading" :bordered="false" :scroll-x="860" :pagination="{ pageSize: 8 }" />
      </NCard>
    </div>
  </section>
</template>
