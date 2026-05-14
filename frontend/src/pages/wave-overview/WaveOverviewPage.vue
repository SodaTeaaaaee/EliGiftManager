<template>
  <div class="wave-overview-page">
    <h1 class="text-xl font-medium mb-4">波次总览</h1>

    <n-space vertical size="large">
      <!-- Create Wave -->
      <n-card title="创建波次">
        <n-space>
          <n-input
            v-model:value="newWaveName"
            placeholder="输入波次名称"
            style="width: 200px"
          />
          <n-button type="primary" @click="handleCreateWave" :loading="creating">
            创建波次
          </n-button>
        </n-space>
      </n-card>

      <!-- Wave List -->
      <n-card title="波次列表">
        <n-data-table
          :columns="columns"
          :data="waves"
          :loading="loading"
          :pagination="false"
          size="small"
        />
        <n-empty v-if="!loading && waves.length === 0" description="暂无波次" />
      </n-card>

      <!-- Wave Overview Summary -->
      <n-card v-if="overview" title="波次汇总">
        <!-- Projected lifecycle stage — prominent -->
        <div class="mb-4">
          <n-tag type="info" size="large">
            预测阶段：{{ overview.projectedLifecycleStage || '-' }}
          </n-tag>
        </div>

        <!-- 基础统计 -->
        <n-divider title-placement="left">基础统计</n-divider>
        <n-space vertical size="small">
          <div><strong>波次号：</strong>{{ overview.wave.waveNo }}</div>
          <div><strong>名称：</strong>{{ overview.wave.name }}</div>
          <div><strong>阶段：</strong>{{ overview.wave.lifecycleStage }}</div>
          <div><strong>需求数：</strong>{{ overview.demandCount }}</div>
          <div><strong>履约行数：</strong>{{ overview.fulfillmentCount }}</div>
          <div><strong>供应商订单数：</strong>{{ overview.supplierOrderCount }}</div>
          <div><strong>发货单数：</strong>{{ overview.shipmentCount ?? 0 }}</div>
          <div><strong>已追踪履约行：</strong>{{ overview.trackedFulfillmentCount ?? 0 }}</div>
        </n-space>

        <!-- 渠道同步状态 -->
        <n-divider title-placement="left">渠道同步状态</n-divider>
        <n-space vertical size="small">
          <div><strong>作业总数：</strong>{{ overview.channelSyncJobCount ?? 0 }}</div>
          <div><strong>待执行：</strong>{{ overview.channelSyncPendingCount ?? 0 }}</div>
          <div><strong>执行中：</strong>{{ overview.channelSyncRunningCount ?? 0 }}</div>
          <div><strong>成功：</strong>{{ overview.channelSyncSuccessCount ?? 0 }}</div>
          <div><strong>部分成功：</strong>{{ overview.channelSyncPartialSuccessCount ?? 0 }}</div>
          <div><strong>失败：</strong>{{ overview.channelSyncFailedCount ?? 0 }}</div>
        </n-space>

        <!-- 手动闭环决策 -->
        <n-divider title-placement="left">手动闭环决策</n-divider>
        <n-space vertical size="small">
          <div><strong>决策总数：</strong>{{ overview.manualClosureDecisionCount ?? 0 }}</div>
          <div><strong>不支持：</strong>{{ overview.manualUnsupportedCount ?? 0 }}</div>
          <div><strong>已跳过：</strong>{{ overview.manualSkippedCount ?? 0 }}</div>
          <div><strong>手动完成：</strong>{{ overview.manualCompletedCount ?? 0 }}</div>
        </n-space>

        <!-- 基线偏移检测 -->
        <n-divider title-placement="left">基线偏移检测</n-divider>
        <n-alert
          v-if="overview.hasDriftedBasis"
          type="warning"
          title="检测到基线偏移，请检查供应商订单与发货数据的一致性"
        />
        <n-text v-else depth="3">无偏移信号</n-text>
      </n-card>

      <n-alert v-if="actionMsg" type="info" :title="actionMsg" />
      <n-alert v-if="actionErr" type="error" :title="actionErr" />
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from "vue";
import {
  NCard,
  NButton,
  NInput,
  NSpace,
  NDataTable,
  NEmpty,
  NAlert,
  NTag,
  NDivider,
  NText,
} from "naive-ui";
import {
  listWaves,
  createWave,
  applyAllocationRules,
  exportSupplierOrder,
  getWaveOverview,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

const waves = ref<dto.WaveDTO[]>([]);
const loading = ref(false);
const creating = ref(false);
const newWaveName = ref("");
const actionMsg = ref("");
const actionErr = ref("");
const overview = ref<dto.WaveOverviewDTO | null>(null);

const columns = [
  { title: "ID", key: "id", width: 60 },
  { title: "波次号", key: "waveNo", width: 180 },
  { title: "名称", key: "name" },
  { title: "阶段", key: "lifecycleStage", width: 100 },
  {
    title: "操作",
    key: "actions",
    width: 300,
    render(row: dto.WaveDTO) {
      return h(NSpace, {}, {
        default: () => [
          h(
            NButton,
            { size: "small", onClick: () => handleAllocate(row.id) },
            { default: () => "分配" },
          ),
          h(
            NButton,
            { size: "small", onClick: () => handleExport(row.id) },
            { default: () => "导出" },
          ),
          h(
            NButton,
            { size: "small", onClick: () => handleOverview(row.id) },
            { default: () => "查看汇总" },
          ),
        ],
      });
    },
  },
];

async function loadWaves() {
  loading.value = true;
  try {
    waves.value = await listWaves();
  } catch (e) {
    /* noop — guards handle the offline case */
  } finally {
    loading.value = false;
  }
}

async function handleCreateWave() {
  creating.value = true;
  try {
    actionErr.value = "";
    const w = await createWave(newWaveName.value || undefined);
    actionMsg.value = `波次 "${w.waveNo}" 创建成功 (ID: ${w.id})`;
    newWaveName.value = "";
    await loadWaves();
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e);
  } finally {
    creating.value = false;
  }
}

async function handleAllocate(waveId: number) {
  actionMsg.value = "";
  actionErr.value = "";
  try {
    const lines = await applyAllocationRules(waveId);
    actionMsg.value = `分配完成：生成了 ${lines?.length ?? 0} 条 FulfillmentLine`;
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e);
  }
}

async function handleExport(waveId: number) {
  actionMsg.value = "";
  actionErr.value = "";
  try {
    const order = await exportSupplierOrder(waveId);
    actionMsg.value = `导出完成：SupplierOrder ID ${order.id}, 状态 ${order.status}`;
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e);
  }
}

async function handleOverview(waveId: number) {
  actionMsg.value = "";
  actionErr.value = "";
  overview.value = null;
  try {
    overview.value = await getWaveOverview(waveId);
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e);
  }
}

onMounted(() => {
  loadWaves();
});
</script>
