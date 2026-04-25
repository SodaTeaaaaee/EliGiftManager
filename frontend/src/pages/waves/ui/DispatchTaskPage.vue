<script setup lang="ts">
import { AddOutline, CloudUploadOutline, DownloadOutline, PlayOutline, RefreshOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NEmpty, NForm, NFormItem, NIcon, NInput, NModal, NSelect, NStep, NSteps, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { autoAllocateWave, createWave, exportWaveByPlatform, importToWave, isWailsRuntimeAvailable, listDispatchRecords, listTemplates, listWaves, WAILS_PREVIEW_MESSAGE, type DispatchRecordItem, type TemplateItem, type WaveItem } from '@/shared/lib/wails/app'

const message = useMessage()
const waves = ref<WaveItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const templates = ref<TemplateItem[]>([])
const selectedWaveId = ref<number | null>(null)
const showCreateTaskModal = ref(false)
const newTaskName = ref('')
const csvPath = ref('')
const importTemplateId = ref<number | null>(null)
const allocationTemplateId = ref<number | null>(null)
const exportTemplateId = ref<number | null>(null)
const exportPlatform = ref('')
const isLoading = ref(false)
const isCreating = ref(false)
const errorMessage = ref('')

const selectedWave = computed(() => waves.value.find((wave) => wave.id === selectedWaveId.value) ?? null)
const importTemplates = computed(() => templates.value.filter((template) => template.type.startsWith('import_')).map(toOption))
const allocationTemplates = computed(() => templates.value.filter((template) => template.type === 'allocation').map(toOption))
const exportTemplates = computed(() => templates.value.filter((template) => template.type === 'export_order').map(toOption))
const platformOptions = computed(() => Array.from(new Set(templates.value.map((template) => template.platform).filter(Boolean))).map((platform) => ({ label: platform, value: platform })))
const currentStep = computed(() => selectedWave.value?.status === 'exported' ? 4 : selectedWave.value?.status === 'pending_address' ? 3 : selectedWave.value?.status === 'allocating' ? 2 : 1)

const waveColumns: DataTableColumns<WaveItem> = [
  { title: '任务编号', key: 'waveNo', minWidth: 160 },
  { title: '任务名称', key: 'name', minWidth: 180 },
  { title: '状态', key: 'status', width: 120, render: (row) => h(NTag, { type: statusTagType(row.status), size: 'small', round: true }, { default: () => statusLabel(row.status) }) },
  { title: '记录', key: 'totalRecords', width: 90 },
]
const recordColumns: DataTableColumns<DispatchRecordItem> = [
  { title: '会员', key: 'memberNickname', minWidth: 160, render: (row) => row.memberNickname || row.platformUid },
  { title: '平台', key: 'platform', width: 100 },
  { title: '礼物', key: 'productName', minWidth: 180 },
  { title: '数量', key: 'quantity', width: 80 },
  { title: '地址', key: 'hasAddress', width: 110, render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => row.hasAddress ? '已绑定' : '待补全' }) },
  { title: '收件信息', key: 'address', minWidth: 260, ellipsis: { tooltip: true }, render: (row) => row.address || '-' },
]

function toOption(template: TemplateItem) { return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id } }
function statusLabel(status: string) { return ({ draft: '草稿', allocating: '分配中', pending_address: '待补全', exported: '已导出' } as Record<string, string>)[status] ?? status }
function statusTagType(status: string) { return status === 'exported' ? 'success' : status === 'pending_address' ? 'warning' : status === 'allocating' ? 'info' : 'default' }
async function guardRuntime() { if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false } return true }
async function loadAll() { if (!(await guardRuntime())) return; isLoading.value = true; errorMessage.value = ''; try { const [waveRows, templateRows] = await Promise.all([listWaves(), listTemplates()]); waves.value = waveRows; templates.value = templateRows; if (!selectedWaveId.value && waveRows.length > 0) selectedWaveId.value = waveRows[0].id; await loadRecords() } catch (error) { console.error(error); errorMessage.value = '加载发货任务失败。' } finally { isLoading.value = false } }
async function loadRecords() { records.value = selectedWaveId.value ? await listDispatchRecords(selectedWaveId.value) : [] }
async function handleCreate() { isCreating.value = true; try { const wave = await createWave(newTaskName.value || '新发货任务'); message.success(`已创建发货任务：${wave.waveNo}`); newTaskName.value = ''; showCreateTaskModal.value = false; await loadAll(); selectedWaveId.value = wave.id; await loadRecords() } catch (error) { message.error(String(error)) } finally { isCreating.value = false } }
async function handleImport() { if (!selectedWaveId.value || !importTemplateId.value) return message.warning('请选择发货任务和导入模板'); try { await importToWave(selectedWaveId.value, csvPath.value, importTemplateId.value); message.success('任务数据导入完成'); await loadAll() } catch (error) { message.error(String(error)) } }
async function handleAllocate() { if (!selectedWaveId.value || !allocationTemplateId.value) return message.warning('请选择匹配规则模板'); try { await autoAllocateWave(selectedWaveId.value, allocationTemplateId.value); message.success('自动分配完成'); await loadAll() } catch (error) { message.error(String(error)) } }
async function handleExport() { if (!selectedWaveId.value || !exportPlatform.value || !exportTemplateId.value) return message.warning('请选择平台和导出模板'); try { const path = await exportWaveByPlatform(selectedWaveId.value, exportPlatform.value, exportTemplateId.value); message.success(`清单已导出：${path}`); await loadAll() } catch (error) { message.error(String(error)) } }

onMounted(loadAll)
</script>
<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div><p class="app-kicker">Dispatch Task</p><h1 class="app-title mt-2">发货任务管理</h1><p class="app-copy mt-2">创建发货任务，导入外部名单，按模板自动分配并导出发货清单。</p></div>
      <div class="flex gap-2"><NButton type="primary" @click="showCreateTaskModal = true"><template #icon><NIcon><AddOutline /></NIcon></template>新建发货任务</NButton><NButton :loading="isLoading" secondary strong @click="loadAll"><template #icon><NIcon><RefreshOutline /></NIcon></template>刷新</NButton></div>
    </header>
    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>
    <div class="grid gap-4 xl:grid-cols-[0.95fr_1.55fr]">
      <NCard title="近期发货任务" size="medium"><NDataTable :columns="waveColumns" :data="waves" :loading="isLoading" :bordered="false" :row-key="(row) => row.id" :row-props="(row) => ({ class: row.id === selectedWaveId ? 'cursor-pointer bg-slate-50 dark:bg-slate-800/60' : 'cursor-pointer', onClick: async () => { selectedWaveId = row.id; await loadRecords() } })" /></NCard>
      <div class="space-y-4">
        <NCard :title="selectedWave ? `${selectedWave.waveNo} · ${selectedWave.name}` : '任务详情'" size="medium">
          <NSteps :current="currentStep" status="process" class="mb-5"><NStep title="导入外部名单" /><NStep title="选择模板自动分配" /><NStep title="异常检查" /><NStep title="按平台导出清单" /></NSteps>
          <div v-if="selectedWave" class="grid gap-3 lg:grid-cols-3"><NCard embedded size="small"><template #header><span class="flex items-center gap-2"><NIcon><CloudUploadOutline /></NIcon>步骤一：导入外部名单</span></template><NInput v-model:value="csvPath" placeholder="CSV 文件路径" class="mb-2" /><NSelect v-model:value="importTemplateId" :options="importTemplates" placeholder="选择导入模板" class="mb-2" /><NButton block secondary @click="handleImport">导入任务数据</NButton></NCard><NCard embedded size="small"><template #header><span class="flex items-center gap-2"><NIcon><PlayOutline /></NIcon>步骤二：选择模板自动分配</span></template><NSelect v-model:value="allocationTemplateId" :options="allocationTemplates" placeholder="选择匹配规则模板" class="mb-2" /><NButton block type="primary" @click="handleAllocate">一键分配</NButton></NCard><NCard embedded size="small"><template #header><span class="flex items-center gap-2"><NIcon><DownloadOutline /></NIcon>步骤四：按平台导出清单</span></template><NSelect v-model:value="exportPlatform" :options="platformOptions" placeholder="选择平台" class="mb-2" /><NSelect v-model:value="exportTemplateId" :options="exportTemplates" placeholder="选择导出模板" class="mb-2" /><NButton block type="success" @click="handleExport">生成发货清单</NButton></NCard></div><NEmpty v-else description="请选择或新建一个发货任务" />
        </NCard>
        <NCard title="步骤三：异常检查与明细调整" size="medium"><NDataTable :columns="recordColumns" :data="records" :bordered="false" :scroll-x="960" :pagination="{ pageSize: 10 }" /></NCard>
      </div>
    </div>
    <NModal v-model:show="showCreateTaskModal" preset="card" title="新建发货任务" style="max-width: 520px"><NForm label-placement="top"><NFormItem label="任务名称"><NInput v-model:value="newTaskName" placeholder="例如：4月24日直播礼物发货" @keyup.enter="handleCreate" /></NFormItem><NButton type="primary" block :loading="isCreating" @click="handleCreate">创建发货任务</NButton></NForm></NModal>
  </section>
</template>
