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
const order = ref<dto.SupplierOrderDTO | null>(null);
const lines = ref<dto.SupplierOrderLineDTO[]>([]);
const error = ref("");

const hasOrder = computed(() => !!(order.value && order.value.id > 0));
const isDraft = computed(() => hasOrder.value && order.value?.status === "draft");

const columns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { title: "Line", key: "supplierLineNo", width: 80 },
  { title: "Supplier SKU", key: "supplierSku", width: 160 },
  { title: "Submitted", key: "submittedQuantity", width: 100 },
  { title: "Accepted", key: "acceptedQuantity", width: 100 },
  { title: "Status", key: "status", width: 100 },
  { title: "Fulfillment Line", key: "fulfillmentLineId", width: 120 },
];

async function loadOrder() {
  loading.value = true;
  error.value = "";
  try {
    const result = await getSupplierOrderByWave(waveId.value);
    if (result && result.id > 0) {
      order.value = result;
      lines.value = await listLinesBySupplierOrder(result.id);
    } else {
      order.value = null;
      lines.value = [];
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
  <div class="wave-export-step">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.execution") }}</div>
      <h2 class="app-title mt-2">{{ t("execution.title") }}</h2>
      <p class="app-copy mt-3">{{ t("execution.subtitle") }}</p>
    </div>

    <NAlert v-if="isDraft" type="warning" class="mb-4">
      {{ t("execution.draftExists") }}
    </NAlert>
    <NAlert v-if="error" type="error" class="mb-4" :title="error" />

    <NCard class="mb-4">
      <div class="flex items-start justify-between gap-6">
        <div>
          <div class="app-heading-sm">{{ t("execution.title") }}</div>
          <p class="app-copy mt-2">
            {{ hasOrder ? t("execution.reexport") : t("execution.noOrder") }}
          </p>
        </div>
        <NSpace vertical>
          <NButton type="primary" :loading="exporting" @click="handleExport">
            {{ hasOrder ? t("execution.reexport") : t("execution.export") }}
          </NButton>
          <NButton secondary @click="router.push(`/waves/${waveId}/shipment`)">
            {{ t("wave.nextStep") }}
          </NButton>
        </NSpace>
      </div>
    </NCard>

    <template v-if="hasOrder && order">
      <NDescriptions bordered :column="2" class="mb-4" label-placement="left">
        <NDescriptionsItem label="ID">{{ order.id }}</NDescriptionsItem>
        <NDescriptionsItem label="Status">
          <NTag :type="order.status === 'draft' ? 'warning' : 'success'" size="small" round>
            {{ order.status }}
          </NTag>
        </NDescriptionsItem>
        <NDescriptionsItem label="Supplier Platform">{{ order.supplierPlatform }}</NDescriptionsItem>
        <NDescriptionsItem :label="t('execution.basis')">{{ order.basisHistoryNodeId || "—" }}</NDescriptionsItem>
        <NDescriptionsItem label="Batch">{{ order.batchNo || "—" }}</NDescriptionsItem>
        <NDescriptionsItem label="External Order No">{{ order.externalOrderNo || "—" }}</NDescriptionsItem>
      </NDescriptions>

      <NCard title="Supplier Order Lines">
        <NDataTable
          :columns="columns"
          :data="lines"
          :loading="loading"
          :pagination="false"
          size="small"
          :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
        />
      </NCard>
    </template>
    <NEmpty v-else-if="!loading" :description="t('execution.noOrder')" />
  </div>
</template>
