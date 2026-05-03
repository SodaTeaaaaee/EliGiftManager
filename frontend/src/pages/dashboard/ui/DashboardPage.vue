<script setup lang="ts">
import { AlbumsOutline, AlertCircleOutline, GiftOutline, PeopleOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NIcon,
  NProgress,
  NStatistic,
  NTag,
  type DataTableColumns,
} from 'naive-ui'
import {
  getDashboard,
  isWailsRuntimeAvailable,
  WAILS_PREVIEW_MESSAGE,
  type DashboardPayload,
  type WaveItem,
} from '@/shared/lib/wails/app'

const dashboard = ref<DashboardPayload | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')
const stats = computed(() => [
  {
    label: '发货任务',
    value: dashboard.value?.waveCount ?? 0,
    icon: AlbumsOutline,
    detail: '当前工作流容器',
  },
  {
    label: '会员',
    value: dashboard.value?.memberCount ?? 0,
    icon: PeopleOutline,
    detail: '按平台 + UID 唯一',
  },
  {
    label: '礼物',
    value: dashboard.value?.productCount ?? 0,
    icon: GiftOutline,
    detail: '按平台隔离',
  },
  {
    label: '待补地址',
    value: dashboard.value?.pendingAddresses ?? 0,
    icon: AlertCircleOutline,
    detail: '导出前需处理',
  },
])
const columns: DataTableColumns<WaveItem> = [
  { title: '任务编号', key: 'waveNo', minWidth: 160 },
  { title: '任务名称', key: 'name', minWidth: 180 },
  {
    title: '状态',
    key: 'status',
    width: 110,
    render: (row) =>
      h(
        NTag,
        {
          type:
            row.status === 'exported'
              ? 'success'
              : row.status === 'pending_address'
                ? 'warning'
                : row.status === 'allocating'
                  ? 'info'
                  : 'default',
          size: 'small',
          round: true,
        },
        {
          default: () =>
            (
              ({
                draft: '草稿',
                allocating: '分配中',
                pending_address: '待补全',
                exported: '已导出',
              }) as Record<string, string>
            )[row.status] ?? row.status,
        },
      ),
  },
  {
    title: '进度',
    key: 'totalRecords',
    minWidth: 180,
    render: (row) =>
      h(NProgress, {
        type: 'line',
        percentage: progressOf(row),
        indicatorPlacement: 'inside',
        processing: row.status !== 'exported',
      }),
  },
]
function progressOf(wave: WaveItem) {
  if (wave.status === 'exported') return 100
  if (wave.status === 'pending_address') return 80
  if (wave.status === 'allocating') return 60
  return wave.totalRecords > 0 ? 35 : 10
}
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
    console.error(error)
    errorMessage.value = '加载仪表盘失败。'
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
        <h1 class="app-title mt-2">工作台</h1>
        <p class="app-copy mt-2">聚焦最近活跃发货任务，快速回到未完成分配流程。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadDashboard">刷新数据</NButton>
    </header>
    <NAlert v-if="errorMessage" type="warning" :show-icon="false">{{ errorMessage }}</NAlert>
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <NCard v-for="item in stats" :key="item.label" size="medium">
        <div class="flex items-start justify-between gap-4">
          <NStatistic :value="item.value"
            ><template #label
              ><span class="app-text-muted text-sm">{{ item.label }}</span></template
            >
          </NStatistic>
          <NIcon :size="22" depth="3">
            <component :is="item.icon" />
          </NIcon>
        </div>
        <NTag class="mt-3" size="small" round>{{ item.detail }}</NTag>
      </NCard>
    </div>
    <div class="grid gap-4 xl:grid-cols-[1.35fr_1fr]">
      <NCard title="最近活跃发货任务" size="medium">
        <NEmpty v-if="(dashboard?.recentWaves.length ?? 0) === 0" description="暂无发货任务" />
        <NDataTable
          v-else
          :columns="columns"
          :data="dashboard?.recentWaves ?? []"
          :loading="isLoading"
          :bordered="false"
          :pagination="false"
          :scroll-x="760"
        />
      </NCard>
      <NCard title="风险提示" size="medium">
        <NEmpty v-if="(dashboard?.warnings.length ?? 0) === 0" description="当前没有数据库风险" />
        <div v-else class="space-y-3">
          <NAlert
            v-for="warning in dashboard?.warnings"
            :key="warning.title"
            :type="(warning.type as 'info' | 'success' | 'warning' | 'error' | 'default')"
            :title="warning.title"
            :show-icon="false"
            >{{ warning.detail }}</NAlert
          >
        </div>
        <p class="app-copy mt-4 break-all">{{ dashboard?.databasePath }}</p>
      </NCard>
    </div>
  </section>
</template>
