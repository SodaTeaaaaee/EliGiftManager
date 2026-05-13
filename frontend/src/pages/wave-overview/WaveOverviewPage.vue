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
        <n-empty
          v-if="!loading && waves.length === 0"
          description="暂无波次"
        />
      </n-card>

      <n-card v-if="overview" title="波次汇总">
        <n-space vertical size="small">
          <div><strong>波次号:</strong> {{ overview.wave.waveNo }}</div>
          <div><strong>名称:</strong> {{ overview.wave.name }}</div>
          <div><strong>阶段:</strong> {{ overview.wave.lifecycleStage }}</div>
          <div><strong>需求数:</strong> {{ overview.demandCount }}</div>
          <div><strong>履约行数:</strong> {{ overview.fulfillmentCount }}</div>
          <div><strong>供应商订单数:</strong> {{ overview.supplierOrderCount }}</div>
          <div><strong>发货单数:</strong> {{ overview?.shipmentCount ?? 0 }}</div>
          <div><strong>已追踪履约行:</strong> {{ overview?.trackedFulfillmentCount ?? 0 }}</div>
        </n-space>
      </n-card>

      <n-card title="录入发货信息" v-if="overview">
        <n-form>
          <n-space vertical size="small">
            <n-input-number v-model:value="shipmentInput.supplierOrderId" placeholder="Supplier Order ID" :min="0" />
            <n-input v-model:value="shipmentInput.supplierPlatform" placeholder="供应商平台" />
            <n-input v-model:value="shipmentInput.shipmentNo" placeholder="发货单号" />
            <n-input v-model:value="shipmentInput.trackingNo" placeholder="物流单号" />
            <n-input v-model:value="shipmentInput.carrierCode" placeholder="承运商编码" />
            <n-input v-model:value="shipmentInput.carrierName" placeholder="承运商名称" />
            <div v-for="(line, index) in shipmentInput.lines" :key="index" style="margin-top: 8px">
              <n-space align="center">
                <n-input-number v-model:value="line.supplierOrderLineId" placeholder="Supplier Order Line ID" :min="0" style="width: 160px" />
                <n-input-number v-model:value="line.fulfillmentLineId" placeholder="Fulfillment Line ID" :min="0" style="width: 160px" />
                <n-input-number v-model:value="line.quantity" placeholder="数量" :min="1" style="width: 80px" />
                <n-button size="small" @click="shipmentInput.lines.splice(index, 1)">删除</n-button>
              </n-space>
            </div>
            <n-button size="small" @click="addShipmentLine" style="margin-top: 4px">添加行</n-button>
            <n-button type="primary" @click="handleCreateShipment" :loading="creatingShipment">
              录入发货
            </n-button>
          </n-space>
        </n-form>
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
  NInputNumber,
  NSpace,
  NDataTable,
  NEmpty,
  NAlert,
  NForm,
} from "naive-ui";
import {
  listWaves,
  createWave,
  applyAllocationRules,
  exportSupplierOrder,
  getWaveOverview,
  createShipment,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

const waves = ref<dto.WaveDTO[]>([]);
const loading = ref(false);
const creating = ref(false);
const newWaveName = ref("");
const actionMsg = ref("");
const actionErr = ref("");
const overview = ref<dto.WaveOverviewDTO | null>(null);

const creatingShipment = ref(false)
const shipmentInput = ref({
  supplierOrderId: 0,
  supplierPlatform: '',
  shipmentNo: '',
  externalShipmentNo: '',
  carrierCode: '',
  carrierName: '',
  trackingNo: '',
  status: 'shipped',
  shippedAt: new Date().toISOString().replace('T', ' ').slice(0, 19),
  basisHistoryNodeId: '',
  basisProjectionHash: '',
  basisPayloadSnapshot: '',
  lines: [] as Array<{ supplierOrderLineId: number; fulfillmentLineId: number; quantity: number }>
})

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
    actionMsg.value =
      `波次 "${w.waveNo}" 创建成功 (ID: ${w.id})`;
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
    actionMsg.value =
      `分配完成：生成了 ${lines?.length ?? 0} 条 FulfillmentLine`;
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e);
  }
}

async function handleExport(waveId: number) {
  actionMsg.value = "";
  actionErr.value = "";
  try {
    const order = await exportSupplierOrder(waveId);
    actionMsg.value =
      `导出完成：SupplierOrder ID ${order.id}, 状态 ${order.status}`;
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

function addShipmentLine() {
  shipmentInput.value.lines.push({
    supplierOrderLineId: 0,
    fulfillmentLineId: 0,
    quantity: 1,
  })
}

async function handleCreateShipment() {
  creatingShipment.value = true
  actionErr.value = ''
  actionMsg.value = ''
  try {
    const result = await createShipment({
      ...shipmentInput.value,
      status: shipmentInput.value.status || 'shipped',
      shippedAt: shipmentInput.value.shippedAt || new Date().toISOString(),
    })
    actionMsg.value = `发货单创建成功 (ID: ${result.id})`
    // 刷新 overview 统计
    if (overview.value) {
      await handleOverview(overview.value.wave.id)
    }
  } catch (e: any) {
    actionErr.value = e?.message ?? String(e)
  } finally {
    creatingShipment.value = false
  }
}

onMounted(() => {
  loadWaves();
});
</script>