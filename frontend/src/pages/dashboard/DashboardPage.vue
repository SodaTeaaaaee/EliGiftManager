<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import {
  NCard,
  NDataTable,
  NAlert,
  NButton,
  NTag,
  NSpace,
  NStatistic,
  NGrid,
  NGridItem,
  NEmpty,
  type DataTableColumns,
} from 'naive-ui'
import { createWave, listWaveDashboardRows } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'

const router = useRouter()
const rows = ref<dto.WaveDashboardRowDTO[]>([])
const loading = ref(false)
const error = ref('')
const creating = ref(false)

const stageTagType: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  intake: 'info',
  allocation: 'info',
  review: 'warning',
  execution: 'warning',
  syncing_back: 'info',
  awaiting_manual_closure: 'error',
  closed: 'default',
}

const stageLabel: Record<string, string> = {
  intake: '需求录入',
  allocation: '分配中',
  review: '审核中',
  execution: '执行中',
  syncing_back: '回填中',
  awaiting_manual_closure: '待人工闭环',
  closed: '已关闭',
}

interface DashboardRow {
  id: number
  waveNo: string
  name: string
  projectedLifecycleStage: string
  createdAt: string
}

const dashboardRows = computed<DashboardRow[]>(() =>
  rows.value.map((r) => ({
    id: r.id,
    waveNo: r.waveNo,
    name: r.name,
    projectedLifecycleStage: r.projectedLifecycleStage || 'intake',
    createdAt: r.createdAt,
  })),
)

const activeCount = computed(() =>
  dashboardRows.value.filter((r) => r.projectedLifecycleStage !== 'closed').length,
)

const closedCount = computed(() =>
  dashboardRows.value.filter((r) => r.projectedLifecycleStage === 'closed').length,
)

const columns: DataTableColumns<DashboardRow> = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '波次号', key: 'waveNo', width: 160 },
  { title: '名称', key: 'name', ellipsis: { tooltip: true } },
  {
    title: '阶段',
    key: 'projectedLifecycleStage',
    width: 120,
    render(row) {
      const stage = row.projectedLifecycleStage || 'intake'
      return h(NTag, {
        type: stageTagType[stage] || 'default',
        size: 'small',
        round: true,
      }, { default: () => stageLabel[stage] || stage })
    },
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 160,
    render(row) {
      if (!row.createdAt) return '-'
      return h('span', new Date(row.createdAt).toLocaleDateString('zh-CN'))
    },
  },
]

function handleRowClick(row: DashboardRow) {
  router.push(`/waves/${row.id}`)
}

async function handleCreateWave() {
  creating.value = true
  try {
    const wave = await createWave(`波次 ${Date.now()}`)
    router.push(`/waves/${wave.id}`)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    creating.value = false
  }
}

async function loadWaves() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await listWaveDashboardRows()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadWaves)
</script>

<template>
  <div class="dashboard-page p-4">
    <div class="flex items-center justify-between mb-4">
      <h1 class="text-xl font-medium">仪表盘</h1>
      <n-button type="primary" :loading="creating" @click="handleCreateWave">
        新建波次
      </n-button>
    </div>

    <n-alert v-if="error" type="error" :title="error" class="mb-4" closable />

    <n-grid :cols="3" :x-gap="16" class="mb-4">
      <n-grid-item>
        <n-card size="small">
          <n-statistic label="波次总数" :value="rows.length" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small">
          <n-statistic label="进行中" :value="activeCount" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small">
          <n-statistic label="已关闭" :value="closedCount" />
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card title="波次列表">
      <n-empty v-if="!loading && rows.length === 0" description="暂无波次，点击右上角创建" />
      <n-data-table
        v-else
        :columns="columns"
        :data="dashboardRows"
        :loading="loading"
        :pagination="false"
        size="small"
        :row-props="(row: DashboardRow) => ({
          style: 'cursor: pointer',
          onClick: () => handleRowClick(row),
        })"
      />
    </n-card>
  </div>
</template>
