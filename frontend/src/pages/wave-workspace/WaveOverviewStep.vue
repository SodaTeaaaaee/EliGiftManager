<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  NCard,
  NDivider,
  NAlert,
  NStatistic,
  NGrid,
  NGridItem,
  NTag,
} from 'naive-ui'
import { getWaveOverview } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'

const route = useRoute()
const waveId = ref(Number(route.params.waveId) || 0)
const overview = ref<dto.WaveOverviewDTO | null>(null)
const error = ref('')

async function loadOverview() {
  try {
    overview.value = await getWaveOverview(waveId.value)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  }
}

onMounted(loadOverview)
</script>

<template>
  <div>
    <n-alert v-if="error" type="error" :title="error" class="mb-4" />

    <n-card v-if="overview" title="波次状态概览">
      <n-tag type="info" size="large" class="mb-4">
        预测阶段：{{ overview.projectedLifecycleStage || '-' }}
      </n-tag>

      <n-grid :cols="4" :x-gap="12" :y-gap="12" class="mt-4">
        <n-grid-item>
          <n-statistic label="需求行" :value="overview.demandCount" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="履约行" :value="overview.fulfillmentCount" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="供应商订单" :value="overview.supplierOrderCount" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="发货单" :value="overview.shipmentCount ?? 0" />
        </n-grid-item>
      </n-grid>

      <n-divider title-placement="left">渠道同步</n-divider>

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
          <n-statistic
            label="部分成功"
            :value="overview.channelSyncPartialSuccessCount ?? 0"
          />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="失败" :value="overview.channelSyncFailedCount ?? 0" />
        </n-grid-item>
      </n-grid>
    </n-card>
  </div>
</template>
