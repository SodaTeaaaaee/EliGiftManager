<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NDescriptions, NDescriptionsItem, NEmpty, NTag, NSpace, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { exportSupplierOrder, getSupplierOrderByWave, listLinesBySupplierOrder } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const loading = ref(false);
const exporting = ref(false);
const orders = ref<dto.SupplierOrderDTO[]>([]);
const orderLines = ref<Map<number, dto.SupplierOrderLineDTO[]>>(new Map());
const error = ref("");

const hasOrders = computed(() => orders.value.length > 0);
const hasDraft = computed(() => orders.value.some((o) => o.status === "draft"));

function statusText(status: string) {
  const map: Record<string, string> = {
    draft: t("execution.statusOptions.draft"),
    submitted: t("execution.statusOptions.submitted"),
    accepted: t("execution.statusOptions.accepted"),
    partially_shipped: t("execution.statusOptions.partiallyShipped"),
    shipped: t("execution.statusOptions.shipped"),
    canceled: t("execution.statusOptions.canceled"),
  };
  return map[status] || status;
}

const columns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { title: t("execution.columns.line"), key: "supplierLineNo", width: 80 },
  { title: t("execution.columns.supplierSku"), key: "supplierSku", width: 180 },
  { title: t("execution.columns.submitted"), key: "submittedQuantity", width: 120 },
  { title: t("execution.columns.accepted"), key: "acceptedQuantity", width: 120 },
  { 
    title: t("execution.status"), 
    key: "status", 
    width: 120,
    render(row) {
      const type = row.status === "accepted" ? "success" : row.status === "submitted" ? "info" : "default";
      return h(NTag, { type, size: "small", bordered: false }, { default: () => row.status || "draft" });
    }
  },
  { title: t("execution.columns.fulfillmentLine"), key: "fulfillmentLineId", width: 140 },
];

async function loadOrder() {
  loading.value = true;
  error.value = "";
  try {
    const result = await getSupplierOrderByWave(waveId.value);
    if (result && result.length > 0) {
      orders.value = result;
      const linesMap = new Map<number, dto.SupplierOrderLineDTO[]>();
      for (const o of result) {
        linesMap.set(o.id, await listLinesBySupplierOrder(o.id));
      }
      orderLines.value = linesMap;
    } else {
      orders.value = [];
      orderLines.value = new Map();
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function handleExport() {
  exporting.value = true;
  error.value = "";
  try {
    await exportSupplierOrder(waveId.value);
    await loadOrder();
    message.success(t("execution.export"));
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    exporting.value = false;
  }
}

onMounted(loadOrder);
</script>

<template>
  <div class="wave-export-step flex flex-col gap-5">
    <div class="mb-2">
      <div class="app-kicker">{{ t("wave.execution") }}</div>
      <h2 class="app-title mt-2">{{ t("execution.title") }}</h2>
      <p class="app-copy mt-2">{{ t("execution.subtitle") }}</p>
    </div>

    <NAlert v-if="hasDraft" type="warning">
      {{ t("execution.draftExists") }}
    </NAlert>
    <NAlert v-if="error" type="error" :title="error" />

    <!-- Action panel -->
    <NCard class="glow-card">
      <div class="flex items-center justify-between gap-6">
        <div>
          <div class="app-heading-sm">{{ t("execution.title") }}</div>
          <p class="app-copy mt-2">
            {{ hasOrders ? t("execution.reexport") : t("execution.noOrder") }}
          </p>
        </div>
        <NSpace>
          <NButton type="primary" :loading="exporting" @click="handleExport">
            {{ hasOrders ? t("execution.reexport") : t("execution.export") }}
          </NButton>
          <NButton secondary @click="router.push(`/waves/${waveId}/shipment`)">
            {{ t("wave.nextStep") }}
          </NButton>
        </NSpace>
      </div>
    </NCard>

    <!-- Orders details list -->
    <template v-if="hasOrders">
      <NCard v-for="order in orders" :key="order.id" class="glow-card" :title="`Supplier Order #${order.id}`">
        <template #header-extra>
          <NTag :type="order.status === 'draft' ? 'warning' : 'success'" size="small" round :bordered="false">
            {{ statusText(order.status) }}
          </NTag>
        </template>

        <NDescriptions bordered :column="3" class="mb-4" label-placement="left" size="small">
          <NDescriptionsItem :label="t('execution.supplierPlatform')">
            <span class="font-bold">{{ order.supplierPlatform || "—" }}</span>
          </NDescriptionsItem>
          <NDescriptionsItem :label="t('execution.batch')">{{ order.batchNo || "—" }}</NDescriptionsItem>
          <NDescriptionsItem :label="t('execution.externalOrderNo')">{{ order.externalOrderNo || "—" }}</NDescriptionsItem>
          <NDescriptionsItem label="Template Key">{{ order.templateID || "—" }}</NDescriptionsItem>
          <NDescriptionsItem label="Submission Mode">{{ order.submissionMode || "—" }}</NDescriptionsItem>
          <NDescriptionsItem :label="t('execution.basis')">
            <NTag size="tiny" :bordered="false">Node #{{ order.basisHistoryNodeID || "—" }}</NTag>
          </NDescriptionsItem>
        </NDescriptions>

        <!-- Paginated Order Lines Table -->
        <NDataTable
          :columns="columns"
          :data="orderLines.get(order.id) || []"
          :loading="loading"
          :pagination="{ pageSize: 10 }"
          size="small"
          :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
        />
      </NCard>
    </template>
    <NEmpty v-else-if="!loading" :description="t('execution.noOrder')" />

    <div class="flex justify-between mt-4">
      <NButton @click="router.push(`/waves/${waveId}/adjustment-review`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
      </NSpace>
    </div>
  </div>
</template>

<style scoped>
.glow-card {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
}

:root[data-theme='dark'] .glow-card {
  border: 1px solid rgba(255, 255, 255, 0.05);
  background: #111827;
}
</style>
