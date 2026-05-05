<script setup lang="ts">
import { AddOutline, BusinessOutline, LibraryOutline, PeopleOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NInput,
  NModal,
  NSelect,
  NTag,
  type DataTableColumns,
  useMessage,
} from 'naive-ui'
import {
  isWailsRuntimeAvailable,
  listTemplates,
  WAILS_PREVIEW_MESSAGE,
  type TemplateItem,
} from '@/shared/lib/wails/app'
import { usePlatformCatalog, type PlatformEntry } from '@/shared/composables/usePlatformCatalog'
import PresetPicker from './PresetPicker.vue'

const router = useRouter()
const message = useMessage()
const templates = ref<TemplateItem[]>([])
const errorMessage = ref('')

const {
  platforms,
  memberPlatforms,
  factoryPlatforms,
  platformOptions,
  addPlatform,
  updatePlatform,
  removePlatform,
  getPlatform,
} = usePlatformCatalog()

const templateTypes = ['import_product', 'import_dispatch_record', 'export_order'] as const
const platformTemplates = (p: string) => templates.value.filter((t) => (t.platform || '通用') === p)
const platformTypeTemplates = (p: string, type: string) =>
  platformTemplates(p).filter((t) => t.type === type)
const platformTemplateCount = (p: string) => platformTemplates(p).length

function typeLabel(type: string) {
  const typeOptions: Record<string, string> = {
    import_product: '礼物导入模板',
    import_dispatch_record: '会员数据导入模板',
    export_order: '发货清单导出模板',
  }
  return typeOptions[type] ?? type
}
const typeColors: Record<string, string> = {
  import_product: 'info',
  import_dispatch_record: 'success',
  export_order: 'warning',
}
function typeColor(type: string) {
  return (typeColors[type] ?? 'default') as 'info' | 'success' | 'warning' | 'default'
}

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
        { type: typeColor(row.type), size: 'small', round: true },
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
            router.push({ name: 'templates-builder', query: { id: row.id } })
          },
        },
        { default: () => '编辑' },
      ),
  },
]

// ---- new platform modal ----
const showNewPlatform = ref(false)
const newPlatformName = ref('')
const newPlatformType = ref<'member' | 'factory'>('member')
const newPlatformNotes = ref('')

function openNewPlatform() {
  newPlatformName.value = ''
  newPlatformType.value = 'member'
  newPlatformNotes.value = ''
  showNewPlatform.value = true
}

function handleAddPlatform() {
  if (!newPlatformName.value.trim()) {
    message.warning('请输入平台名称')
    return
  }
  if (!addPlatform(newPlatformName.value, newPlatformType.value, newPlatformNotes.value)) {
    message.warning('平台名称重复或无效')
    return
  }
  showNewPlatform.value = false
  message.success('平台已添加')
}

// ---- edit platform modal ----
const showEditPlatform = ref(false)
const editTarget = ref('')
const editName = ref('')
const editType = ref<'member' | 'factory'>('member')
const editNotes = ref('')
const editCanChangeType = ref(true)

function openEditPlatform(name: string) {
  const p = getPlatform(name)
  if (!p) return
  editTarget.value = name
  editName.value = p.name
  editType.value = p.type
  editNotes.value = p.notes || ''
  editCanChangeType.value = platformTemplateCount(name) === 0
  showEditPlatform.value = true
}

function handleUpdatePlatform() {
  if (!editName.value.trim()) {
    message.warning('平台名称不能为空')
    return
  }
  const updates: Partial<PlatformEntry> & { name?: string } = {
    name: editName.value.trim(),
    notes: editNotes.value.trim(),
  }
  if (editCanChangeType.value) {
    updates.type = editType.value
  }
  if (!updatePlatform(editTarget.value, updates)) {
    message.warning('更新失败，名称可能重复')
    return
  }
  showEditPlatform.value = false
  message.success('平台已更新')
}

function handleDeletePlatform() {
  if (!removePlatform(editTarget.value)) {
    message.warning('删除失败')
    return
  }
  showEditPlatform.value = false
  message.success('平台已删除')
}

// ---- template builder ----
function openCreateModal(platform?: string) {
  router.push({ name: 'templates-builder', query: platform ? { platform } : {} })
}

const showPresetPicker = ref(false)
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
        <p class="app-copy mt-2">管理平台和模板：先建平台，再为平台配置导入/导出模板。</p>
      </div>
      <div class="flex gap-2">
        <NButton secondary @click="showPresetPicker = true">
          <template #icon><NIcon><LibraryOutline /></NIcon></template>从预设添加
        </NButton>
        <NButton type="primary" @click="openCreateModal()">
          <template #icon><NIcon><AddOutline /></NIcon></template>新建模板
        </NButton>
        <NButton secondary @click="openNewPlatform()">
          <template #icon><NIcon><AddOutline /></NIcon></template>新建平台
        </NButton>
      </div>
    </header>

    <NEmpty v-if="errorMessage" :description="errorMessage" />

    <!-- No platforms at all -->
    <NEmpty
      v-else-if="!platforms.length"
      description="尚未创建任何平台，请先点击「新建平台」开始"
    />

    <template v-else>
      <!-- 会员平台 -->
      <div v-if="memberPlatforms.length">
        <div class="flex items-center gap-2 mb-3">
          <NIcon size="20"><PeopleOutline /></NIcon>
          <h2 class="text-base font-semibold">会员平台</h2>
          <NTag size="small" :bordered="false">{{ memberPlatforms.length }}</NTag>
        </div>
        <div class="grid gap-4 xl:grid-cols-2">
          <NCard v-for="p in memberPlatforms" :key="p.name" size="small">
            <template #header>
              <div class="flex items-center justify-between">
                <div>
                  <span class="font-semibold">{{ p.name }}</span>
                  <span v-if="p.notes" class="text-xs text-gray-400 ml-2">{{ p.notes }}</span>
                </div>
                <NButton size="tiny" secondary @click="openEditPlatform(p.name)">编辑平台</NButton>
              </div>
            </template>
            <NAlert
              v-if="platformTemplateCount(p.name) === 1
                && platformTemplates(p.name)[0]?.type === 'import_product'"
              type="warning"
              :show-icon="false"
              class="mb-3"
            >
              此平台有商品导入模板但缺少订单导出模板，建议补全。
            </NAlert>
            <template v-if="platformTemplateCount(p.name)">
              <div v-for="t in templateTypes" :key="t" class="mb-3 last:mb-0">
                <NTag :type="typeColor(t)" size="small" round class="mb-1">
                  {{ typeLabel(t) }}
                </NTag>
                <NDataTable
                  v-if="platformTypeTemplates(p.name, t).length"
                  :columns="columns"
                  :data="platformTypeTemplates(p.name, t)"
                  :bordered="false"
                  :pagination="false"
                  size="small"
                />
                <div v-else class="text-xs text-gray-400 py-2 pl-1">暂无</div>
              </div>
            </template>
            <NEmpty v-else description="暂无模板" />
          </NCard>
        </div>
      </div>

      <!-- 工厂平台 -->
      <div v-if="factoryPlatforms.length">
        <div class="flex items-center gap-2 mb-3 mt-5">
          <NIcon size="20"><BusinessOutline /></NIcon>
          <h2 class="text-base font-semibold">工厂平台</h2>
          <NTag size="small" :bordered="false">{{ factoryPlatforms.length }}</NTag>
        </div>
        <div class="grid gap-4 xl:grid-cols-2">
          <NCard v-for="p in factoryPlatforms" :key="p.name" size="small">
            <template #header>
              <div class="flex items-center justify-between">
                <div>
                  <span class="font-semibold">{{ p.name }}</span>
                  <span v-if="p.notes" class="text-xs text-gray-400 ml-2">{{ p.notes }}</span>
                </div>
                <NButton size="tiny" secondary @click="openEditPlatform(p.name)">编辑平台</NButton>
              </div>
            </template>
            <template v-if="platformTemplateCount(p.name)">
              <div v-for="t in templateTypes" :key="t" class="mb-3 last:mb-0">
                <NTag :type="typeColor(t)" size="small" round class="mb-1">
                  {{ typeLabel(t) }}
                </NTag>
                <NDataTable
                  v-if="platformTypeTemplates(p.name, t).length"
                  :columns="columns"
                  :data="platformTypeTemplates(p.name, t)"
                  :bordered="false"
                  :pagination="false"
                  size="small"
                />
                <div v-else class="text-xs text-gray-400 py-2 pl-1">暂无</div>
              </div>
            </template>
            <NEmpty v-else description="暂无模板" />
          </NCard>
        </div>
      </div>
    </template>

    <!-- New platform modal -->
    <NModal v-model:show="showNewPlatform" title="新建平台" preset="card" style="max-width: 400px">
      <div class="space-y-3">
        <div>
          <label class="text-sm">平台名称</label>
          <NInput v-model:value="newPlatformName" placeholder="例如：肉燥工廠" />
        </div>
        <div>
          <label class="text-sm block mb-1">平台类型</label>
          <NSelect
            v-model:value="newPlatformType"
            :options="[
              { label: '会员平台', value: 'member' },
              { label: '工厂平台', value: 'factory' },
            ]"
          />
          <p class="text-xs text-gray-400 mt-1">
            会员平台：仅会员导入模板 &nbsp;|&nbsp; 工厂平台：商品导入 + 订单导出模板
          </p>
        </div>
        <div>
          <label class="text-sm block mb-1">备注（可选）</label>
          <NInput v-model:value="newPlatformNotes" placeholder="例如：每月 15 号导出" />
        </div>
        <div class="flex gap-2 justify-end pt-2">
          <NButton @click="showNewPlatform = false">取消</NButton>
          <NButton type="primary" @click="handleAddPlatform">创建</NButton>
        </div>
      </div>
    </NModal>

    <!-- Edit platform modal -->
    <NModal v-model:show="showEditPlatform" title="编辑平台" preset="card" style="max-width: 400px">
      <div class="space-y-3">
        <div>
          <label class="text-sm">平台名称</label>
          <NInput v-model:value="editName" placeholder="平台名称" />
        </div>
        <div>
          <label class="text-sm block mb-1">平台类型</label>
          <NSelect
            v-model:value="editType"
            :options="[
              { label: '会员平台', value: 'member' },
              { label: '工厂平台', value: 'factory' },
            ]"
            :disabled="!editCanChangeType"
          />
          <p v-if="!editCanChangeType" class="text-xs text-[var(--warning)] mt-1">
            该平台下已有模板，无法更改类型
          </p>
        </div>
        <div>
          <label class="text-sm block mb-1">备注（可选）</label>
          <NInput v-model:value="editNotes" placeholder="备注说明" />
        </div>
        <div class="flex justify-between pt-2">
          <NButton type="error" secondary size="small" @click="handleDeletePlatform">删除平台</NButton>
          <div class="flex gap-2">
            <NButton @click="showEditPlatform = false">取消</NButton>
            <NButton type="primary" @click="handleUpdatePlatform">保存</NButton>
          </div>
        </div>
      </div>
    </NModal>

    <!-- Preset picker -->
    <PresetPicker v-model:visible="showPresetPicker" @added="loadTemplates" />
  </section>
</template>
