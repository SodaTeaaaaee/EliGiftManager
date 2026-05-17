<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NCollapse, NCollapseItem, NDataTable, NDatePicker, NEmpty, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpin, NTag, NSpace, useMessage } from "naive-ui";
import type { DataTableColumns, DataTableRowKey } from "naive-ui";
import { createShipment, getSupplierOrderByWave, listLinesBySupplierOrder, listShipmentsByWave } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const loadingList = ref(false);
const loadingOrder = ref(false);
const submitting = ref(false);
const listError = ref("");
const formError = ref("");
const shipments = ref<dto.ShipmentDTO[]>([]);
const supplierOrder = ref<dto.SupplierOrderDTO | null>(null);
const orderLines = ref<dto.SupplierOrderLineDTO[]>([]);
const selectedLineKeys = ref<DataTableRowKey[]>([]);
const lineQuantities = ref<Record<number, number>>({});

const form = ref({
  shipmentNo: "",
  externalShipmentNo: "",
  carrierCode: "",
  carrierName: "",
  trackingNo: "",
  status: "pending",
  shippedAt: null as number | null,
});

const statusOptions = [
  { label: "pending", value: "pending" },
  { label: "shipped", value: "shipped" },
  { label: "in_transit", value: "in_transit" },
  { label: "delivered", value: "delivered" },
  { label: "exception", value: "exception" },
  { label: "returned", value: "returned" },
];

const shipmentColumns: DataTableColumns<dto.ShipmentDTO> = [
  { title: "Shipment No", key: "shipmentNo", width: 160 },
  { title: "Carrier", key: "carrierName", width: 120 },
  { title: "Tracking", key: "trackingNo", width: 180 },
  {
    title: "Status",
    key: "status",
    width: 100,
    render(row) {
      const type = row.status === "delivered" ? "success" : row.status === "exception" ? "error" : "default";
      return h(NTag, { type, size: "small", round: true }, { default: () => row.status });
    },
  },
  { title: "Shipped At", key: "shippedAt", width: 180 },
];

const lineSelectionColumns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { type: "selection" },
  { title: "Line", key: "supplierLineNo", width: 80 },
  { title: "Supplier SKU", key: "supplierSku", width: 160 },
  { title: "Submitted", key: "submittedQuantity", width: 100 },
  { title: "Fulfillment Line", key: "fulfillmentLineId", width: 120 },
  {
    title: "This Shipment",
    key: "qty",
    width: 120,
    render(row) {
      return h(NInputNumber, {
        value: lineQuantities.value[row.id] ?? row.submittedQuantity,
        min: 1,
        max: row.submittedQuantity,
        onUpdateValue: (value: number | null) => {
          lineQuantities.value[row.id] = value ?? 1;
        },
      });
    },
  },
];

async function loadShipments() {
  loadingList.value = true;
  listError.value = "";
  try {
    shipments.value = await listShipmentsByWave(waveId.value);
  } catch (e: unknown) {
    listError.value = e instanceof Error ? e.message : String(e);
  } finally {
    loadingList.value = false;
  }
}

async function loadSupplierOrder() {
  loadingOrder.value = true;
  formError.value = "";
  try {
    const order = await getSupplierOrderByWave(waveId.value);
    supplierOrder.value = order && order.id > 0 ? order : null;
    orderLines.value = supplierOrder.value ? await listLinesBySupplierOrder(supplierOrder.value.id) : [];
  } catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : String(e);
  } finally {
    loadingOrder.value = false;
  }
}

async function handleSubmit() {
  if (!supplierOrder.value) return;
  submitting.value = true;
  formError.value = "";
  try {
    const selectedLines = orderLines.value
      .filter((line) => selectedLineKeys.value.includes(line.id))
      .map((line) => ({
        supplierOrderLineId: line.id,
        fulfillmentLineId: line.fulfillmentLineId,
        quantity: lineQuantities.value[line.id] ?? line.submittedQuantity,
      }));

    await createShipment({
      supplierOrderId: supplierOrder.value.id,
      supplierPlatform: supplierOrder.value.supplierPlatform,
      shipmentNo: form.value.shipmentNo,
      externalShipmentNo: form.value.externalShipmentNo,
      carrierCode: form.value.carrierCode,
      carrierName: form.value.carrierName,
      trackingNo: form.value.trackingNo,
      status: form.value.status,
      shippedAt: form.value.shippedAt ? new Date(form.value.shippedAt).toISOString() : "",
      basisPayloadSnapshot: "",
      lines: selectedLines,
    });

    message.success(t("shipment.create"));
    await loadShipments();
    form.value = {
      shipmentNo: "",
      externalShipmentNo: "",
      carrierCode: "",
      carrierName: "",
      trackingNo: "",
      status: "pending",
      shippedAt: null,
    };
    selectedLineKeys.value = [];
    lineQuantities.value = {};
  } catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : String(e);
  } finally {
    submitting.value = false;
  }
}

onMounted(async () => {
  await loadShipments();
  await loadSupplierOrder();
});
</script>

<template>
  <div class="wave-shipment-step">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.shipment") }}</div>
      <h2 class="app-title mt-2">{{ t("shipment.title") }}</h2>
      <p class="app-copy mt-3">{{ t("shipment.subtitle") }}</p>
    </div>

    <NAlert v-if="listError" type="error" class="mb-4" :title="listError" />
    <NAlert v-if="formError" type="error" class="mb-4" :title="formError" />

    <NCard :title="t('shipment.list')" class="mb-4">
      <NEmpty v-if="!loadingList && shipments.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="shipmentColumns"
        :data="shipments"
        :loading="loadingList"
        :pagination="false"
        size="small"
        :row-key="(row: dto.ShipmentDTO) => row.id"
      />
    </NCard>

    <NCard :title="t('shipment.create')">
      <NSpin :show="loadingOrder">
        <NForm label-placement="left" label-width="120">
          <NFormItem label="Supplier Order ID">
            <NInput :value="supplierOrder ? String(supplierOrder.id) : '—'" readonly />
          </NFormItem>
          <NFormItem label="Shipment No">
            <NInput v-model:value="form.shipmentNo" />
          </NFormItem>
          <NFormItem label="External Shipment No">
            <NInput v-model:value="form.externalShipmentNo" />
          </NFormItem>
          <NFormItem label="Carrier Code">
            <NInput v-model:value="form.carrierCode" />
          </NFormItem>
          <NFormItem label="Carrier Name">
            <NInput v-model:value="form.carrierName" />
          </NFormItem>
          <NFormItem label="Tracking No">
            <NInput v-model:value="form.trackingNo" />
          </NFormItem>
          <NFormItem label="Status">
            <NSelect v-model:value="form.status" :options="statusOptions" />
          </NFormItem>
          <NFormItem label="Shipped At">
            <NDatePicker v-model:value="form.shippedAt" type="datetime" clearable />
          </NFormItem>
        </NForm>

        <NDataTable
          class="mt-4"
          :columns="lineSelectionColumns"
          :data="orderLines"
          :pagination="false"
          size="small"
          :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
          v-model:checked-row-keys="selectedLineKeys"
        />

        <div class="mt-4 flex justify-between">
          <NButton @click="router.push(`/waves/${waveId}/export`)">{{ t("wave.prevStep") }}</NButton>
          <NSpace>
            <NButton type="primary" :loading="submitting" :disabled="selectedLineKeys.length === 0" @click="handleSubmit">
              {{ t("shipment.create") }}
            </NButton>
            <NButton secondary @click="router.push(`/waves/${waveId}/channel-sync`)">{{ t("wave.nextStep") }}</NButton>
          </NSpace>
        </div>
      </NSpin>
    </NCard>
  </div>
</template>
