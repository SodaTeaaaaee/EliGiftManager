<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NCollapse, NCollapseItem, NDataTable, NDatePicker, NEmpty, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpin, NTag, NSpace, useMessage } from "naive-ui";
import type { DataTableColumns, DataTableRowKey } from "naive-ui";
import { createShipment, getSupplierOrderByWave, importShipments, listIntegrationProfiles, listLinesBySupplierOrder, listShipmentsByWave, pickCsvFile } from "@/shared/lib/wails/app";
import type { ImportShipmentEntry } from "@/shared/lib/wails/app";
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

// ── Import section state ──

const showImportSection = ref(false);
const importSubmitting = ref(false);
const importError = ref("");
const importResult = ref<{ successCount: number; errorCount: number; errors: Array<{ entryIndex: number; reason: string }> } | null>(null);
const importProfileId = ref<number | null>(null);
const integrationProfiles = ref<dto.IntegrationProfileSummaryDTO[]>([]);
const loadingProfiles = ref(false);

type ImportRow = ImportShipmentEntry & { _key: number }

let _rowKeySeq = 0;
function makeRow(): ImportRow {
  return {
    _key: ++_rowKeySeq,
    supplierOrderLineId: 0,
    fulfillmentLineId: 0,
    externalShipmentNo: "",
    carrierCode: "",
    carrierName: "",
    trackingNo: "",
    quantity: 1,
    shippedAt: "",
  };
}

const importRows = ref<ImportRow[]>([makeRow()]);

function addImportRow() {
  importRows.value.push(makeRow());
}

function removeImportRow(key: number) {
  importRows.value = importRows.value.filter((r) => r._key !== key);
}

async function loadProfiles() {
  if (integrationProfiles.value.length > 0) return;
  loadingProfiles.value = true;
  try {
    integrationProfiles.value = await listIntegrationProfiles();
  } finally {
    loadingProfiles.value = false;
  }
}

function toggleImportSection() {
  showImportSection.value = !showImportSection.value;
  if (showImportSection.value) {
    loadProfiles();
  }
}

async function handlePickCsv() {
  try {
    await pickCsvFile();
    // CSV parsing happens backend-side via template system.
    // File selection confirms the intent; user fills entries manually in the table.
    message.info(t("shipment.importButton"));
  } catch (e: unknown) {
    importError.value = e instanceof Error ? e.message : String(e);
  }
}

async function handleImportSubmit() {
  importError.value = "";
  importResult.value = null;

  if (!importProfileId.value) {
    importError.value = t("shipment.importProfile");
    return;
  }
  const entries = importRows.value.filter(
    (r) => r.supplierOrderLineId > 0 && r.fulfillmentLineId > 0 && r.quantity > 0,
  );
  if (entries.length === 0) {
    importError.value = "No valid entries to import.";
    return;
  }

  importSubmitting.value = true;
  try {
    const result = await importShipments({
      waveId: waveId.value,
      integrationProfileId: importProfileId.value,
      entries: entries.map(({ _key: _k, ...rest }) => rest),
    });
    importResult.value = {
      successCount: result.successCount,
      errorCount: result.errorCount,
      errors: result.errors,
    };
    if (result.successCount > 0) {
      message.success(t("shipment.importSuccess").replace("{count}", String(result.successCount)));
      await loadShipments();
    }
  } catch (e: unknown) {
    importError.value = e instanceof Error ? e.message : String(e);
  } finally {
    importSubmitting.value = false;
  }
}

const profileOptions = computed(() =>
  integrationProfiles.value.map((p) => ({
    label: p.profileKey,
    value: p.id,
  })),
);

const importRowColumns: DataTableColumns<ImportRow> = [
  {
    title: t("shipment.import.supplierOrderLineId"),
    key: "supplierOrderLineId",
    width: 120,
    render(row) {
      return h(NInputNumber, {
        value: row.supplierOrderLineId,
        min: 0,
        placeholder: "0",
        style: "width:100%",
        onUpdateValue: (v: number | null) => { row.supplierOrderLineId = v ?? 0; },
      });
    },
  },
  {
    title: t("shipment.import.fulfillmentLineId"),
    key: "fulfillmentLineId",
    width: 120,
    render(row) {
      return h(NInputNumber, {
        value: row.fulfillmentLineId,
        min: 0,
        placeholder: "0",
        style: "width:100%",
        onUpdateValue: (v: number | null) => { row.fulfillmentLineId = v ?? 0; },
      });
    },
  },
  {
    title: t("shipment.import.externalShipmentNo"),
    key: "externalShipmentNo",
    width: 140,
    render(row) {
      return h(NInput, {
        value: row.externalShipmentNo,
        placeholder: "—",
        onUpdateValue: (v: string) => { row.externalShipmentNo = v; },
      });
    },
  },
  {
    title: t("shipment.import.carrierCode"),
    key: "carrierCode",
    width: 100,
    render(row) {
      return h(NInput, {
        value: row.carrierCode,
        placeholder: "—",
        onUpdateValue: (v: string) => { row.carrierCode = v; },
      });
    },
  },
  {
    title: t("shipment.import.carrierName"),
    key: "carrierName",
    width: 120,
    render(row) {
      return h(NInput, {
        value: row.carrierName,
        placeholder: "—",
        onUpdateValue: (v: string) => { row.carrierName = v; },
      });
    },
  },
  {
    title: t("shipment.import.trackingNo"),
    key: "trackingNo",
    width: 140,
    render(row) {
      return h(NInput, {
        value: row.trackingNo,
        placeholder: "—",
        onUpdateValue: (v: string) => { row.trackingNo = v; },
      });
    },
  },
  {
    title: t("shipment.import.quantity"),
    key: "quantity",
    width: 80,
    render(row) {
      return h(NInputNumber, {
        value: row.quantity,
        min: 1,
        style: "width:100%",
        onUpdateValue: (v: number | null) => { row.quantity = v ?? 1; },
      });
    },
  },
  {
    title: t("shipment.import.shippedAt"),
    key: "shippedAt",
    width: 160,
    render(row) {
      return h(NInput, {
        value: row.shippedAt,
        placeholder: "2024-01-01T00:00:00Z",
        onUpdateValue: (v: string) => { row.shippedAt = v; },
      });
    },
  },
  {
    title: "",
    key: "_actions",
    width: 70,
    render(row) {
      return h(
        NButton,
        {
          size: "small",
          type: "error",
          ghost: true,
          onClick: () => removeImportRow(row._key),
        },
        { default: () => t("shipment.removeRow") },
      );
    },
  },
];

const importErrorColumns: DataTableColumns<{ entryIndex: number; reason: string }> = [
  { title: "#", key: "entryIndex", width: 60 },
  { title: "Reason", key: "reason" },
];

// ── Existing shipment list / create ──

function shipmentStatusText(status: string) {
  const map: Record<string, string> = {
    pending: t("shipment.statusOptions.pending"),
    shipped: t("shipment.statusOptions.shipped"),
    in_transit: t("shipment.statusOptions.inTransit"),
    delivered: t("shipment.statusOptions.delivered"),
    exception: t("shipment.statusOptions.exception"),
    returned: t("shipment.statusOptions.returned"),
  };
  return map[status] || status;
}

const statusOptions = [
  { label: t("shipment.statusOptions.pending"), value: "pending" },
  { label: t("shipment.statusOptions.shipped"), value: "shipped" },
  { label: t("shipment.statusOptions.inTransit"), value: "in_transit" },
  { label: t("shipment.statusOptions.delivered"), value: "delivered" },
  { label: t("shipment.statusOptions.exception"), value: "exception" },
  { label: t("shipment.statusOptions.returned"), value: "returned" },
];

const shipmentColumns: DataTableColumns<dto.ShipmentDTO> = [
  { title: t("shipment.columns.shipmentNo"), key: "shipmentNo", width: 160 },
  { title: t("shipment.columns.carrier"), key: "carrierName", width: 120 },
  { title: t("shipment.columns.tracking"), key: "trackingNo", width: 180 },
  {
    title: t("shipment.columns.status"),
    key: "status",
    width: 100,
    render(row) {
      const type = row.status === "delivered" ? "success" : row.status === "exception" ? "error" : "default";
      return h(NTag, { type, size: "small", round: true }, { default: () => shipmentStatusText(row.status) });
    },
  },
  { title: t("shipment.columns.shippedAt"), key: "shippedAt", width: 180 },
];

const lineSelectionColumns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { type: "selection" },
  { title: t("shipment.columns.line"), key: "supplierLineNo", width: 80 },
  { title: t("shipment.columns.supplierSku"), key: "supplierSku", width: 160 },
  { title: t("shipment.columns.submitted"), key: "submittedQuantity", width: 100 },
  { title: t("shipment.columns.fulfillmentLine"), key: "fulfillmentLineId", width: 120 },
  {
    title: t("shipment.columns.thisShipment"),
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

    <!-- Import Factory Shipments -->
    <NCard class="mb-4" style="border: 1px solid var(--n-border-color); background: var(--n-color-modal);">
      <template #header>
        <NSpace align="center" justify="space-between">
          <span style="font-weight:600;">{{ t("shipment.importTitle") }}</span>
          <NButton size="small" secondary @click="toggleImportSection">
            {{ showImportSection ? t("common.cancel") : t("shipment.importButton") }}
          </NButton>
        </NSpace>
      </template>

      <div v-if="showImportSection">
        <NAlert v-if="importError" type="error" class="mb-3" :title="importError" />

        <NAlert
          v-if="importResult"
          :type="importResult.errorCount === 0 ? 'success' : 'warning'"
          class="mb-3"
          :title="importResult.successCount > 0
            ? t('shipment.importSuccess').replace('{count}', String(importResult.successCount))
            : t('shipment.importErrors').replace('{count}', String(importResult.errorCount))"
        />

        <div v-if="importResult && importResult.errors.length > 0" class="mb-3">
          <NDataTable
            :columns="importErrorColumns"
            :data="importResult.errors"
            :pagination="false"
            size="small"
          />
        </div>

        <NForm label-placement="left" label-width="120" class="mb-3">
          <NFormItem :label="t('shipment.importProfile')">
            <NSelect
              v-model:value="importProfileId"
              :options="profileOptions"
              :loading="loadingProfiles"
              :placeholder="t('shipment.importProfile')"
              style="width:280px;"
            />
          </NFormItem>
        </NForm>

        <NDataTable
          :columns="importRowColumns"
          :data="importRows"
          :pagination="false"
          size="small"
          :row-key="(row: ImportRow) => row._key"
          class="mb-3"
          :scroll-x="1060"
        />

        <NSpace>
          <NButton size="small" dashed @click="addImportRow">+ {{ t("shipment.addRow") }}</NButton>
          <NButton size="small" secondary @click="handlePickCsv">CSV</NButton>
          <NButton
            type="primary"
            size="small"
            :loading="importSubmitting"
            :disabled="!importProfileId || importRows.length === 0"
            @click="handleImportSubmit"
          >
            {{ t("shipment.importButton") }}
          </NButton>
        </NSpace>
      </div>
    </NCard>

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
          <NFormItem :label="t('shipment.supplierOrderId')">
            <NInput :value="supplierOrder ? String(supplierOrder.id) : '—'" readonly />
          </NFormItem>
          <NFormItem :label="t('shipment.shipmentNo')">
            <NInput v-model:value="form.shipmentNo" />
          </NFormItem>
          <NFormItem :label="t('shipment.externalShipmentNo')">
            <NInput v-model:value="form.externalShipmentNo" />
          </NFormItem>
          <NFormItem :label="t('shipment.carrierCode')">
            <NInput v-model:value="form.carrierCode" />
          </NFormItem>
          <NFormItem :label="t('shipment.carrierName')">
            <NInput v-model:value="form.carrierName" />
          </NFormItem>
          <NFormItem :label="t('shipment.trackingNo')">
            <NInput v-model:value="form.trackingNo" />
          </NFormItem>
          <NFormItem :label="t('shipment.status')">
            <NSelect v-model:value="form.status" :options="statusOptions" />
          </NFormItem>
          <NFormItem :label="t('shipment.shippedAt')">
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
