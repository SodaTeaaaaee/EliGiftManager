<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  NCard,
  NDivider,
  NAlert,
  NStatistic,
  NGrid,
  NGridItem,
  NTag,
  NText,
  NSpace,
  NTimeline,
  NTimelineItem,
  NEmpty,
} from 'naive-ui'
import { getWaveOverview, listRecentHistory } from '@/shared/lib/wails/app'
import type { HistoryNodeDTO } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'

const route = useRoute()
const waveId = computed(() => Number(route.params.waveId) || 0)
const overview = ref<dto.WaveOverviewDTO | null>(null)
const recentHistory = ref<HistoryNodeDTO[]>([])
const loading = ref(false)
const error = ref('')

const stageLabel: Record<string, string> = {
  planning: '规划中',
  demand_intake: '需求录入',
  allocation: '分配中',
  fulfillment: '履约中',
  export: '导出中',
  shipping: '物流中',
  channel_sync: '回填中',
  closed: '已关闭',
}

const stageTagType: Record<string, 'default' | 'info' | 'success' | 'warning'> = {
  planning: 'default',
  demand_intake: 'info',
  allocation: 'info',
  fulfillment: 'warning',
  export: 'warning',
  shipping: 'success',
  channel_sync: 'success',
  closed: 'default',
}

const nextStepGuidance = computed(() => {
  if (!overview.value) return ''
  const o = overview.value
  if (o.demandCount === 0) return '下一步：前往「需求映射」录入需求文档'
  if (o.fulfillmentCount === 0) return '下一步：前往「分配规则」配置并执行分配'
  if (o.supplierOrderCount === 0) return '下一步：前往「导出」生成供应商订单'
  if (o.shipmentCount === 0) return '下一步：前往「物流」录入发货信息'
  if (o.channelSyncJobCount === 0) return '下一步：前往「回填」创建渠道同步任务'
  return '所有主要步骤已完成'
})

async function loadOverview() {
  loading.value = true
  error.value = ''
  try {
    overview.value = await getWaveOverview(waveId.value)
    recentHistory.value = await listRecentHistory(waveId.value, 10)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
</script>

<template>
  <div>
    <n-alert v-if="error" type="error" :title="error" class="mb-4" closable />

    <template v-if="overview">
      <n-card class="mb-4">
        <template #header>
          <n-space align="center">
            <span class="text-lg font-medium">{{ overview.wave?.name || '波次概览' }}</span>
            <n-tag
              :type="stageTagType[overview.projectedLifecycleStage] || 'default'"
              size="medium"
              round
            >
              {{ stageLabel[overview.projectedLifecycleStage] || overview.projectedLifecycleStage || '未知' }}
            </n-tag>
          </n-space>
        </template>

        <n-alert v-if="nextStepGuidance" type="info" class="mb-4">
          {{ nextStepGuidance }}
        </n-alert>

        <n-grid :cols="4" :x-gap="16" :y-gap="16">
          <n-grid-item>
            <n-statistic label="需求行" :value="overview.demandCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="履约行" :value="overview.fulfillmentCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="供应商订单" :value="overview.supplierOrderCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="发货单" :value="overview.shipmentCount ?? 0" />
          </n-grid-item>
        </n-grid>
      </n-card>

      <n-card title="渠道同步" class="mb-4" size="small">
        <n-grid :cols="5" :x-gap="12">
          <n-grid-item>
            <n-statistic label="待执行" :value="overview.channelSyncPendingCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="执行中" :value="overview.channelSyncRunningCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="成功" :value="overview.channelSyncSuccessCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="部分成功" :value="overview.channelSyncPartialSuccessCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="失败" :value="overview.channelSyncFailedCount ?? 0" />
          </n-grid-item>
        </n-grid>
      </n-card>

      <n-card title="手动闭环决策" class="mb-4" size="small">
        <n-grid :cols="4" :x-gap="12">
          <n-grid-item>
            <n-statistic label="决策总数" :value="overview.manualClosureDecisionCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="不支持" :value="overview.manualUnsupportedCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="已跳过" :value="overview.manualSkippedCount ?? 0" />
          </n-grid-item>
          <n-grid-item>
            <n-statistic label="手动完成" :value="overview.manualCompletedCount ?? 0" />
          </n-grid-item>
        </n-grid>
      </n-card>

      <n-card title="基线偏移检测" size="small">
        <n-alert
          v-if="overview.hasDriftedBasis"
          type="warning"
          title="检测到基线偏移"
        >
          供应商订单或发货数据与当前工作区状态存在偏差，请检查一致性。
        </n-alert>
        <n-alert
          v-else-if="overview.hasRequiredReviewBasis"
          type="info"
          title="存在待审查基线"
        >
          部分基线引用需要人工确认。
        </n-alert>
        <n-text v-else depth="3">无偏移信号</n-text>
      </n-card>

      <n-card title="最近操作" class="mt-4" size="small">
        <n-empty v-if="recentHistory.length === 0" description="暂无操作记录" />
        <n-timeline v-else>
          <n-timeline-item
            v-for="node in recentHistory"
            :key="node.id"
            :title="node.commandSummary"
            :time="node.createdAt ? new Date(node.createdAt).toLocaleString('zh-CN') : ''"
          >
            <n-tag size="tiny" :bordered="false">{{ node.commandKind }}</n-tag>
          </n-timeline-item>
        </n-timeline>
      </n-card>
    </template>
  </div>
</template>
