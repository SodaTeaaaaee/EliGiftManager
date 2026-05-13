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
} from "naive-ui";
import {
  listWaves,
  createWave,
  applyAllocationRules,
  exportSupplierOrder,
} from "@/shared/lib/wails/app";

const waves = ref<any[]>([]);
const loading = ref(false);
const creating = ref(false);
const newWaveName = ref("");
const actionMsg = ref("");
const actionErr = ref("");

const columns = [
  { title: "ID", key: "id", width: 60 },
  { title: "波次号", key: "waveNo", width: 180 },
  { title: "名称", key: "name" },
  { title: "阶段", key: "lifecycleStage", width: 100 },
  {
    title: "操作",
    key: "actions",
    width: 240,
    render(row: any) {
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

onMounted(() => {
  loadWaves();
});
</script>
