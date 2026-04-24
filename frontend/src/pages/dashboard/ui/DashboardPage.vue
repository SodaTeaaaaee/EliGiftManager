<script setup lang="ts">
import {
  AlertCircleOutline,
  AlbumsOutline,
  CubeOutline,
  FileTrayFullOutline,
  LayersOutline,
  PeopleOutline,
  TicketOutline,
  WarningOutline,
} from '@vicons/ionicons5'
import { computed, h, onMounted, ref, type Component } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NStatistic,
  NTag,
  type DataTableColumns,
} from 'naive-ui'
import {
  WAILS_PREVIEW_MESSAGE,
  getDashboard,
  isWailsRuntimeAvailable,
  type DispatchRecordItem,
  type DashboardPayload,
} from '@/shared/lib/wails/app'

interface StatItem {
  label: string
  value: number
  detail: string
  icon: Component
  tone: 'default' | 'info' | 'warning' | 'error' | 'success'
}

const dashboard = ref<DashboardPayload | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')

const stats = computed<StatItem[]>(() => {
  const data = dashboard.value
  return [
    { label: '会员', value: data?.memberCount ?? 0, detail: `${data?.addressCount ?? 0} 条有效地址`, icon: PeopleOutline, tone: 'info' },
    { label: '商品', value: data?.productCount ?? 0, detail: '来自 products 表', icon: CubeOutline, tone: 'success' },
    { label: '派发记录', value: data?.dispatchCount ?? 0, detail: `${data?.batchCount ?? 0} 个批次`, icon: TicketOutline, tone: 'default' },
    { label: '待补地址', value: data?.pendingAddresses ?? 0, detail: `${data?.missingAddresses ?? 0} 位会员缺地址`, icon: WarningOutline, tone: (data?.pendingAddresses ?? 0) > 0 ? 'warning' : 'success' },
  ]
})

const columns: DataTableColumns<DispatchRecordItem> = [
  { title: '批次', key: 'batchName', minWidth: 140 },
  {
    title: '会员',
    key: 'memberNickname',
    minWidth: 140,
    render: (row) => row.memberNickname || row.platformUid,
  },
  { title: '商品', key: 'productName', minWidth: 180 },
  { title: '数量', key: 'quantity', width: 80 },
  {
    title: '地址',
    key: 'hasAddress',
    width: 110,
    render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => (row.hasAddress ? '已绑定' : '待补全') }),
  },
]

async function loadDashboard() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''

  try {
    dashboard.value = await getDashboard()
  } catch (error) {
    console.error('加载工作台失败', error)
    errorMessage.value = '加载数据库看板失败，请查看后端日志。'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Dashboard</p>
        <h1 class="app-title mt-2">数据库工作台</h1>
        <p class="app-copy mt-2">实时汇总 SQLite 中的会员、商品、派发批次和地址风险。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadDashboard">刷新数据</NButton>
    </header>

    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <NCard v-for="item in stats" :key="item.label" size="medium">
        <div class="flex items-start justify-between gap-4">
          <NStatistic :value="item.value">
            <template #label><span class="app-text-muted text-sm">{{ item.label }}</span></template>
          </NStatistic>
          <NIcon :size="22" depth="3"><component :is="item.icon" /></NIcon>
        </div>
        <NTag class="mt-3" :type="item.tone" size="small" round>{{ item.detail }}</NTag>
      </NCard>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1.35fr_1fr]">
      <NCard size="medium" title="最近派发记录">
        <NDataTable
          :columns="columns"
          :data="dashboard?.recentDispatches ?? []"
          :loading="isLoading"
          :bordered="false"
          :pagination="false"
          :scroll-x="640"
        />
      </NCard>

      <div class="space-y-4">
        <NCard size="medium">
          <template #header>
            <div class="flex items-center gap-2"><NIcon><AlbumsOutline /></NIcon><span>批次概览</span></div>
          </template>
          <NEmpty v-if="(dashboard?.batches.length ?? 0) === 0" description="数据库中暂无派发批次" />
          <div v-else class="space-y-3">
            <div v-for="batch in dashboard?.batches" :key="batch.batchName" class="rounded-xl border border-slate-200/70 p-3 dark:border-slate-700/70">
              <div class="flex items-center justify-between gap-3">
                <strong>{{ batch.batchName }}</strong>
                <NTag :type="batch.pendingAddressRecords > 0 ? 'warning' : 'success'" size="small" round>
                  {{ batch.pendingAddressRecords }} 待补
                </NTag>
              </div>
              <p class="app-copy mt-1">{{ batch.totalRecords }} 条记录 / {{ batch.totalQuantity }} 件礼物</p>
            </div>
          </div>
        </NCard>

        <NCard size="medium">
          <template #header>
            <div class="flex items-center gap-2"><NIcon><AlertCircleOutline /></NIcon><span>风险提示</span></div>
          </template>
          <NEmpty v-if="(dashboard?.warnings.length ?? 0) === 0" description="当前没有数据库风险" />
          <div v-else class="space-y-3">
            <NAlert v-for="warning in dashboard?.warnings" :key="warning.title" :type="warning.type" :title="warning.title" :show-icon="false">
              {{ warning.detail }}
            </NAlert>
          </div>
        </NCard>

        <NCard size="medium">
          <template #header>
            <div class="flex items-center gap-2"><NIcon><LayersOutline /></NIcon><span>数据库文件</span></div>
          </template>
          <p class="app-copy break-all">{{ dashboard?.databasePath ?? '等待连接 Wails 后端' }}</p>
          <NTag class="mt-3" type="info" size="small" round>{{ dashboard?.templateCount ?? 0 }} 个模板配置</NTag>
          <NIcon class="ml-2" depth="3"><FileTrayFullOutline /></NIcon>
        </NCard>
      </div>
    </div>
  </section>
</template>
