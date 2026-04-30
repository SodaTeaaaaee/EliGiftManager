<script setup lang="ts">
import { AddOutline, RefreshOutline, TrashOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NDataTable, NDrawer, NDrawerContent, NEmpty, NForm, NFormItem, NIcon, NInput, NModal, NStep, NSteps, NTabPane, NTabs, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { createWave, deleteWave, isWailsRuntimeAvailable, listTemplates, listWaves, WAILS_PREVIEW_MESSAGE, type TemplateItem, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const router = useRouter()
const route = useRoute()

const waves = ref<WaveItem[]>([])
const templates = ref<TemplateItem[]>([])
const showWaveDrawer = ref(false)
const showCreateTaskModal = ref(false)
const newTaskName = ref('')
const isLoading = ref(false)
const isCreating = ref(false)
const errorMessage = ref('')
const waveStatusFilter = ref('active')

const selectedWaveId = computed(() => {
  const id = Number(route.params.waveId)
  return id > 0 ? id : null
})
const selectedWave = computed(() => waves.value.find(w => w.id === selectedWaveId.value) ?? null)
const currentStep = computed(() => {
  const name = route.name
  if (name === 'waves-step-import') return 1
  if (name === 'waves-step-tags') return 2
  if (name === 'waves-step-preview') return 3
  if (name === 'waves-step-export') return 4
  return 1
})

const filteredWaves = computed(() => {
  if (waveStatusFilter.value === 'active') return waves.value.filter(w => w.status !== 'exported')
  if (waveStatusFilter.value === 'done') return waves.value.filter(w => w.status === 'exported')
  return waves.value
})

function statusLabel(status: string) {
  return ({ draft: '草稿', allocating: '分配中', pending_address: '待补全', exported: '已导出' } as Record<string, string>)[status] ?? status
}
function statusTagType(status: string) {
  return status === 'exported' ? 'success' : status === 'pending_address' ? 'warning' : status === 'allocating' ? 'info' : 'default'
}

const waveColumns: DataTableColumns<WaveItem> = [
  { title: '任务编号', key: 'waveNo', minWidth: 120 },
  { title: '任务名称', key: 'name', minWidth: 180 },
  { title: '状态', key: 'status', width: 120, render: (row) => h(NTag, { type: statusTagType(row.status), size: 'small', round: true }, { default: () => statusLabel(row.status) }) },
  { title: '记录', key: 'totalRecords', width: 70 },
  { title: 'ID', key: 'id', width: 50 },
  {
    title: '', key: 'actions', width: 50, render: (_row, index) =>
      h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: (e: MouseEvent) => { e.stopPropagation(); handleDeleteWave(filteredWaves.value[index].id) } }, {
        icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
      }),
  },
]

async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false }
  return true
}

async function loadAll() {
  if (!(await guardRuntime())) return
  isLoading.value = true
  errorMessage.value = ''
  try {
    const [waveRows, templateRows] = await Promise.all([listWaves(), listTemplates()])
    waves.value = waveRows
    templates.value = templateRows
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载发货任务失败。'
  } finally {
    isLoading.value = false
  }
}

function openWave(waveId: number) {
  router.push({ name: 'waves-step-import', params: { waveId: String(waveId) } })
}
function closeWave() {
  router.push({ name: 'waves-welcome' })
}

function handleSelectWave(row: WaveItem) {
  showWaveDrawer.value = false
  openWave(row.id)
}

async function handleCreate() {
  isCreating.value = true
  try {
    const wave = await createWave(newTaskName.value || '新发货任务')
    message.success(`已创建发货任务：${wave.waveNo}`)
    newTaskName.value = ''
    showCreateTaskModal.value = false
    await loadAll()
    openWave(wave.id)
  } catch (error) {
    message.error(String(error))
  } finally {
    isCreating.value = false
  }
}

async function handleDeleteWave(waveId: number) {
  try {
    await deleteWave(waveId)
    message.success('已删除发货任务')
    if (selectedWaveId.value === waveId) {
      closeWave()
    }
    await loadAll()
  } catch (error) {
    message.error(String(error))
  }
}

onMounted(loadAll)
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Dispatch Task</p>
        <h1 class="app-title mt-2">发货任务管理</h1>
        <p class="app-copy mt-2">创建发货任务，导入外部名单，按 Tag 匹配自动分配并导出发货清单。</p>
      </div>
      <div class="flex gap-2">
        <NButton @click="showWaveDrawer = true">发货任务列表</NButton>
        <NButton type="primary" @click="showCreateTaskModal = true">
          <template #icon><NIcon><AddOutline /></NIcon></template>
          新建发货任务
        </NButton>
        <NButton :loading="isLoading" secondary strong @click="loadAll">
          <template #icon><NIcon><RefreshOutline /></NIcon></template>
          刷新
        </NButton>
      </div>
    </header>

    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>

    <!-- Wave Drawer -->
    <NDrawer v-model:show="showWaveDrawer" :width="840">
      <NDrawerContent title="发货任务列表" closable>
        <NTabs v-model:value="waveStatusFilter" size="small" class="mb-3">
          <NTabPane name="active" tab="进行中" />
          <NTabPane name="done" tab="已完成" />
          <NTabPane name="all" tab="全部" />
        </NTabs>
        <NDataTable :columns="waveColumns" :data="filteredWaves" :loading="isLoading" :bordered="false"
          :row-key="(row: WaveItem) => row.id"
          :row-props="(row: WaveItem) => ({ class: row.id === selectedWaveId ? 'cursor-pointer bg-slate-50 dark:bg-slate-800/60' : 'cursor-pointer', onClick: () => handleSelectWave(row) })" />
      </NDrawerContent>
    </NDrawer>

    <div v-if="selectedWave" class="space-y-4">
      <div class="flex flex-wrap items-center gap-3 p-3 bg-slate-50 dark:bg-slate-800/60 rounded-lg">
        <span class="font-semibold text-lg min-w-0 break-all">{{ selectedWave.waveNo }} · {{ selectedWave.name }}</span>
        <NTag :type="statusTagType(selectedWave.status)" size="small" round class="shrink-0">{{ statusLabel(selectedWave.status) }}</NTag>
        <div class="ml-auto shrink-0">
          <NButton size="small" secondary @click="closeWave">关闭任务</NButton>
        </div>
      </div>

      <NSteps :current="currentStep" status="process" class="mb-3">
        <NStep title="导入数据" />
        <NStep title="Tag 管理与分配" />
        <NStep title="导出预览与编辑" />
        <NStep title="异常检查与导出" />
      </NSteps>

      <RouterView />
    </div>

    <RouterView v-if="!selectedWave" />

    <NEmpty v-if="!selectedWave && !route.name" description="点击「发货任务列表」选择或新建一个发货任务" />

    <NModal v-model:show="showCreateTaskModal" preset="card" title="新建发货任务" style="max-width: 520px">
      <NForm label-placement="top">
        <NFormItem label="任务名称">
          <NInput v-model:value="newTaskName" placeholder="例如：4月24日直播礼物发货" @keyup.enter="handleCreate" />
        </NFormItem>
        <NButton type="primary" block :loading="isCreating" @click="handleCreate">创建发货任务</NButton>
      </NForm>
    </NModal>
  </section>
</template>
