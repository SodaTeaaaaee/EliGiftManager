<script setup lang="ts">
import { AddOutline, TrashOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NDataTable, NEmpty, NForm, NFormItem, NIcon, NInput, NModal, NTabPane, NTabs, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { createWave, deleteWave, isWailsRuntimeAvailable, listWaves, WAILS_PREVIEW_MESSAGE, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const router = useRouter()

const waves = ref<WaveItem[]>([])
const showCreateModal = ref(false)
const newTaskName = ref('')
const isLoading = ref(false)
const isCreating = ref(false)
const errorMessage = ref('')
const waveStatusFilter = ref('active')

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

const columns: DataTableColumns<WaveItem> = [
  { title: '任务编号', key: 'waveNo', minWidth: 100 },
  { title: '任务名称', key: 'name', minWidth: 160 },
  { title: '状态', key: 'status', width: 120, render: (row) => h(NTag, { type: statusTagType(row.status), size: 'small', round: true }, { default: () => statusLabel(row.status) }) },
  { title: '记录', key: 'totalRecords', width: 70 },
  { title: 'ID', key: 'id', width: 50 },
  { title: '', key: 'actions', width: 50, render: (_row, index) =>
    h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: (e: MouseEvent) => { e.stopPropagation(); handleDelete(filteredWaves.value[index].id) } }, {
      icon: () => h(NIcon, null, { default: () => h(TrashOutline) }),
    }),
  },
]

async function load() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return }
  isLoading.value = true; errorMessage.value = ''
  try { waves.value = await listWaves() }
  catch (e) { console.error(e); errorMessage.value = '加载失败' }
  finally { isLoading.value = false }
}

function openWave(row: WaveItem) {
  router.push({ name: 'waves-step-import', params: { waveId: String(row.id) } })
}

async function handleCreate() {
  isCreating.value = true
  try {
    const wave = await createWave(newTaskName.value || '新发货任务')
    message.success(`已创建：${wave.waveNo}`)
    newTaskName.value = ''; showCreateModal.value = false
    await load(); openWave(wave)
  } catch (e) { message.error(String(e)) }
  finally { isCreating.value = false }
}

async function handleDelete(waveId: number) {
  try { await deleteWave(waveId); message.success('已删除'); await load() }
  catch (e) { message.error(String(e)) }
}

onMounted(load)
</script>
<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between shrink-0 mb-3">
      <NTabs v-model:value="waveStatusFilter" size="small">
        <NTabPane name="active" tab="进行中" />
        <NTabPane name="done" tab="已完成" />
        <NTabPane name="all" tab="全部" />
      </NTabs>
      <NButton type="primary" size="small" @click="showCreateModal = true">
        <template #icon><NIcon><AddOutline /></NIcon></template>
        新建任务
      </NButton>
    </div>

    <div v-if="errorMessage" class="shrink-0 mb-3">
      <NEmpty :description="errorMessage" />
    </div>

    <NDataTable
      v-if="filteredWaves.length"
      :columns="columns"
      :data="filteredWaves"
      :loading="isLoading"
      :bordered="false"
      :row-key="(row: WaveItem) => row.id"
      :row-props="(row: WaveItem) => ({ class: 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50', onClick: () => openWave(row) })"
      :pagination="{ pageSize: 15 }"
      class="flex-1"
    />

    <NEmpty v-if="!isLoading && !errorMessage && !filteredWaves.length" description="暂无发货任务，点击右上角「新建任务」开始" />

    <NModal v-model:show="showCreateModal" preset="card" title="新建发货任务" style="max-width: 480px">
      <NForm label-placement="top">
        <NFormItem label="任务名称">
          <NInput v-model:value="newTaskName" placeholder="例如：4月24日直播礼物发货" @keyup.enter="handleCreate" />
        </NFormItem>
        <NButton type="primary" block :loading="isCreating" @click="handleCreate">创建任务</NButton>
      </NForm>
    </NModal>
  </div>
</template>
