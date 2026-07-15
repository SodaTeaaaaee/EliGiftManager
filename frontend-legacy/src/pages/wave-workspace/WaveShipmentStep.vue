<script setup lang="ts">
import { computed, h, onMounted, ref, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NDatePicker, NEmpty, NForm, NFormItem, NInput, NInputNumber, NRadio, NRadioGroup, NSelect, NSpin, NTag, NSpace, NTabs, NTabPane, NUpload, useMessage } from "naive-ui";
import type { DataTableColumns, DataTableRowKey } from "naive-ui";
import { createShipment, getSupplierOrderByWave, importShipments, listIntegrationProfiles, listLinesBySupplierOrder, listShipmentsByWave } from "@/shared/lib/wails/app";
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

// ── Tab Control ──
const activeTab = ref("manual");

// ── Import Wizard State ──
const importStep = ref<number>(1); // 1: upload, 2: mapping, 3: validation & preview
const importSubmitting = ref(false);
const importError = ref("");
const importResult = ref<{ successCount: number; errorCount: number; errors: Array<{ entryIndex: number; reason: string }> } | null>(null);
const importProfileId = ref<number | null>(null);
const importMode = ref<'skip_invalid' | 'reject_all'>('skip_invalid');
const integrationProfiles = ref<dto.IntegrationProfileSummaryDTO[]>([]);
const loadingProfiles = ref(false);

const csvHeaders = ref<string[]>([]);
const csvParsedRows = ref<Record<string, string>[]>([]);
const fileInput = ref<HTMLInputElement | null>(null);
const fileName = ref("");

// Headers Mapping Schema
const mappings = reactive<Record<string, string>>({
  supplierOrderLineId: "",
  fulfillmentLineId: "",
  externalShipmentNo: "",
  carrierCode: "",
  carrierName: "",
  trackingNo: "",
  quantity: "",
  shippedAt: "",
});

const generatedEntries = ref<ImportShipmentEntry[]>([]);

const profileOptions = computed(() =>
  integrationProfiles.value.map((p) => ({
    label: p.profileKey,
    value: p.id,
  })),
);

// CSV parsing utilities
function parseCSV(text: string): Record<string, string>[] {
  const lines = text.split(/\r?\n/);
  if (lines.length === 0) return [];
  
  const headers = parseCSVLine(lines[0]);
  const results: Record<string, string>[] = [];
  
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i].trim();
    if (!line) continue;
    const values = parseCSVLine(line);
    const row: Record<string, string> = {};
    headers.forEach((header, index) => {
      row[header] = values[index] || "";
    });
    results.push(row);
  }
  return results;
}

function parseCSVLine(line: string): string[] {
  const result: string[] = [];
  let current = "";
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (char === '"') {
      inQuotes = !inQuotes;
    } else if (char === ',' && !inQuotes) {
      result.push(current.trim());
      current = "";
    } else {
      current += char;
    }
  }
  result.push(current.trim());
  return result.map(val => val.replace(/^"|"$/g, ""));
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

// Trigger browser file selector
function triggerFileSelect() {
  fileInput.value?.click();
}

async function onFileSelected(event: Event) {
  const target = event.target as HTMLInputElement;
  if (!target.files || target.files.length === 0) return;
  const file = target.files[0];
  fileName.value = file.name;
  importError.value = "";
  
  try {
    const text = await file.text();
    const rows = parseCSV(text);
    if (rows.length === 0) {
      importError.value = "Selected CSV file is empty.";
      return;
    }
    
    csvParsedRows.value = rows;
    csvHeaders.value = Object.keys(rows[0]);
    
    // Auto-map similar names
    autoMapHeaders();
    
    importStep.value = 2; // move to header mapping
  } catch (e: unknown) {
    importError.value = "Failed to read CSV: " + String(e);
  }
}

function autoMapHeaders() {
  const findMatch = (field: string, synonyms: string[]) => {
    const header = csvHeaders.value.find(h => 
      synonyms.includes(h.toLowerCase().trim()) || 
      h.toLowerCase().includes(field.toLowerCase())
    );
    return header || "";
  };

  mappings.supplierOrderLineId = findMatch("supplierOrderLineId", ["supplierorderlineid", "order line id", "line id", "订单行id"]);
  mappings.fulfillmentLineId = findMatch("fulfillmentLineId", ["fulfillmentlineid", "fulfillment id", "履约行id"]);
  mappings.externalShipmentNo = findMatch("externalShipmentNo", ["externalshipmentno", "shipment no", "运单号", "发货单号"]);
  mappings.carrierCode = findMatch("carrierCode", ["carriercode", "carrier code", "承运商代码"]);
  mappings.carrierName = findMatch("carrierName", ["carriername", "carrier name", "快递公司"]);
  mappings.trackingNo = findMatch("trackingNo", ["trackingno", "tracking number", "tracking no", "快递单号", "物流单号"]);
  mappings.quantity = findMatch("quantity", ["quantity", "qty", "数量"]);
  mappings.shippedAt = findMatch("shippedAt", ["shippedat", "shipped date", "shipped time", "发货时间"]);
}

function processMapping() {
  importError.value = "";
  // Check required mappings
  if (!mappings.supplierOrderLineId || !mappings.fulfillmentLineId || !mappings.trackingNo) {
    importError.value = "Supplier Order Line ID, Fulfillment Line ID and Tracking No are required mappings.";
    return;
  }

  const entries: ImportShipmentEntry[] = [];
  csvParsedRows.value.forEach((row) => {
    const solId = Number(row[mappings.supplierOrderLineId]) || 0;
    const flId = Number(row[mappings.fulfillmentLineId]) || 0;
    const qty = Number(row[mappings.quantity]) || 1;
    
    entries.push({
      supplierOrderLineId: solId,
      fulfillmentLineId: flId,
      externalShipmentNo: mappings.externalShipmentNo ? row[mappings.externalShipmentNo] || "" : "",
      carrierCode: mappings.carrierCode ? row[mappings.carrierCode] || "" : "",
      carrierName: mappings.carrierName ? row[mappings.carrierName] || "" : "",
      trackingNo: row[mappings.trackingNo] || "",
      quantity: qty,
      shippedAt: mappings.shippedAt ? row[mappings.shippedAt] || "" : "",
    });
  });

  generatedEntries.value = entries;
  importStep.value = 3; // move to validation preview
}

async function handleImportSubmit() {
  importError.value = "";
  importResult.value = null;

  if (!importProfileId.value) {
    importError.value = t("shipment.importProfile");
    return;
  }
  if (generatedEntries.value.length === 0) {
    importError.value = "No entries parsed to import.";
    return;
  }

  importSubmitting.value = true;
  try {
    const result = await importShipments({
      waveId: waveId.value,
      integrationProfileId: importProfileId.value,
      importMode: importMode.value,
      entries: generatedEntries.value,
    });
    importResult.value = {
      successCount: result.successCount,
      errorCount: result.errorCount,
      errors: result.errors,
    };
    if (result.successCount > 0) {
      message.success(t("shipment.importSuccess").replace("{count}", String(result.successCount)));
      await loadShipments();
      resetImportWizard();
    }
  } catch (e: unknown) {
    importError.value = e instanceof Error ? e.message : String(e);
  } finally {
    importSubmitting.value = false;
  }
}

function resetImportWizard() {
  importStep.value = 1;
  fileName.value = "";
  csvHeaders.value = [];
  csvParsedRows.value = [];
  generatedEntries.value = [];
  importResult.value = null;
  if (fileInput.value) {
    fileInput.value.value = "";
  }
}

// ── Manual creation & listing ──

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
  { title: t("shipment.columns.carrier"), key: "carrierName", width: 140 },
  { title: t("shipment.columns.tracking"), key: "trackingNo", width: 200 },
  {
    title: t("shipment.columns.status"),
    key: "status",
    width: 120,
    render(row) {
      const type = row.status === "delivered" ? "success" : row.status === "exception" ? "error" : "default";
      return h(NTag, { type, size: "small", round: true, bordered: false }, { default: () => shipmentStatusText(row.status) });
    },
  },
  { title: t("shipment.columns.shippedAt"), key: "shippedAt", width: 200 },
];

const lineSelectionColumns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { type: "selection" },
  { title: t("shipment.columns.line"), key: "supplierLineNo", width: 80 },
  { title: t("shipment.columns.supplierSku"), key: "supplierSku", width: 160 },
  { title: t("shipment.columns.submitted"), key: "submittedQuantity", width: 110 },
  { title: t("shipment.columns.fulfillmentLine"), key: "fulfillmentLineId", width: 130 },
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
    supplierOrder.value = order && order.length > 0 ? order[0] : null;
    orderLines.value = supplierOrder.value ? await listLinesBySupplierOrder(supplierOrder.value.id) : [];
  } catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : String(e);
  } finally {
    loadingOrder.value = false;
  }
}

async function handleManualSubmit() {
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
  await loadProfiles();
});
</script>

<template>
  <div class="wave-shipment-step flex flex-col gap-5">
    <div class="mb-2">
      <div class="app-kicker">{{ t("wave.shipment") }}</div>
      <h2 class="app-title mt-2">{{ t("shipment.title") }}</h2>
      <p class="app-copy mt-2">{{ t("shipment.subtitle") }}</p>
    </div>

    <NAlert v-if="listError" type="error" :title="listError" />
    <NAlert v-if="formError" type="error" :title="formError" />

    <!-- Tabs Layout -->
    <NTabs v-model:value="activeTab" type="segment" animated>
      <!-- MANUAL ENTRY TAB -->
      <NTabPane name="manual" tab="Manual Shipment Entry">
        <NGrid :cols="24" :x-gap="16" class="mt-4">
          <!-- Manual Form -->
          <NGridItem :span="10">
            <NCard title="New Shipment details" class="glow-card h-full">
              <NSpin :show="loadingOrder">
                <NForm label-placement="left" label-width="120" size="small">
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
                    <NDatePicker v-model:value="form.shippedAt" type="datetime" clearable style="width:100%" />
                  </NFormItem>
                </NForm>
              </NSpin>
            </NCard>
          </NGridItem>

          <!-- Order Lines selector -->
          <NGridItem :span="14">
            <NCard title="Order lines to include" class="glow-card h-full">
              <NDataTable
                :columns="lineSelectionColumns"
                :data="orderLines"
                :pagination="{ pageSize: 5 }"
                size="small"
                :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
                v-model:checked-row-keys="selectedLineKeys"
              />
              <div class="flex justify-end mt-4">
                <NButton 
                  type="primary" 
                  :loading="submitting" 
                  :disabled="selectedLineKeys.length === 0" 
                  @click="handleManualSubmit"
                >
                  {{ t("shipment.create") }}
                </NButton>
              </div>
            </NCard>
          </NGridItem>
        </NGrid>
      </NTabPane>

      <!-- CSV IMPORT WIZARD TAB -->
      <NTabPane name="csv" tab="Bilingual CSV Import Wizard">
        <NCard class="glow-card mt-4">
          <NAlert v-if="importError" type="error" class="mb-4" :title="importError" />
          
          <!-- Invisible File Input -->
          <input 
            type="file" 
            ref="fileInput" 
            style="display: none" 
            accept=".csv" 
            @change="onFileSelected" 
          />

          <!-- STEP 1: Upload Dropzone -->
          <div v-if="importStep === 1" class="flex flex-col items-center justify-center border-2 border-dashed border-slate-700/20 dark:border-slate-700/40 rounded-xl py-12 px-6 cursor-pointer hover:border-blue-500 transition-colors" @click="triggerFileSelect">
            <svg viewBox="0 0 24 24" class="w-12 h-12 text-slate-400 mb-3">
              <path fill="currentColor" d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"/>
            </svg>
            <div class="text-base font-semibold mb-1">Click to select CSV File</div>
            <div class="text-xs text-slate-400">Supported formats: standard comma-separated factory shipment files</div>
          </div>

          <!-- STEP 2: Header Mappings -->
          <div v-else-if="importStep === 2" class="flex flex-col gap-4">
            <div class="flex items-center justify-between border-b border-slate-700/10 dark:border-slate-700/30 pb-3">
              <h3 class="text-base font-bold">Step 2: Map CSV Columns</h3>
              <NSpace>
                <NButton size="small" secondary @click="resetImportWizard">Back</NButton>
                <NButton size="small" type="primary" @click="processMapping">Map and Preview</NButton>
              </NSpace>
            </div>
            
            <NGrid :cols="2" :x-gap="20" :y-gap="12">
              <NGridItem v-for="(val, field) in mappings" :key="field">
                <NFormItem :label="field" :required="['supplierOrderLineId', 'fulfillmentLineId', 'trackingNo'].includes(field)">
                  <NSelect 
                    v-model:value="mappings[field]" 
                    :options="csvHeaders.map(h => ({ label: h, value: h }))" 
                    placeholder="Select CSV column"
                  />
                </NFormItem>
              </NGridItem>
            </NGrid>
          </div>

          <!-- STEP 3: Preview and Submit -->
          <div v-else-if="importStep === 3" class="flex flex-col gap-4">
            <div class="flex items-center justify-between border-b border-slate-700/10 dark:border-slate-700/30 pb-3">
              <div class="flex flex-col">
                <h3 class="text-base font-bold">Step 3: Preview and Validate</h3>
                <span class="text-xs text-slate-400">{{ generatedEntries.length }} rows ready to import</span>
              </div>
              <NSpace>
                <NButton size="small" secondary @click="importStep = 2">Back</NButton>
                <NButton size="small" secondary @click="resetImportWizard">Reset</NButton>
              </NSpace>
            </div>

            <!-- Profile & Import Configuration -->
            <div class="bg-slate-700/5 dark:bg-slate-700/20 p-4 rounded-lg flex items-center justify-between">
              <NForm label-placement="left" inline class="m-0">
                <NFormItem :label="t('shipment.importProfile')" class="m-0 mr-4">
                  <NSelect
                    v-model:value="importProfileId"
                    :options="profileOptions"
                    :loading="loadingProfiles"
                    placeholder="Choose profile"
                    style="width:240px;"
                  />
                </NFormItem>
                <NFormItem :label="t('shipment.importMode')" class="m-0">
                  <NRadioGroup v-model:value="importMode">
                    <NRadio value="skip_invalid">{{ t('shipment.importModeSkipInvalid') }}</NRadio>
                    <NRadio value="reject_all" style="margin-left:16px;">{{ t('shipment.importModeRejectAll') }}</NRadio>
                  </NRadioGroup>
                </NFormItem>
              </NForm>

              <NButton 
                type="primary"
                :loading="importSubmitting"
                :disabled="!importProfileId"
                @click="handleImportSubmit"
              >
                Start Import
              </NButton>
            </div>

            <NDataTable
              :columns="[
                { title: 'SOL ID', key: 'supplierOrderLineId', width: 90 },
                { title: 'FL ID', key: 'fulfillmentLineId', width: 90 },
                { title: 'External No', key: 'externalShipmentNo', width: 130 },
                { title: 'Carrier Code', key: 'carrierCode', width: 110 },
                { title: 'Carrier Name', key: 'carrierName', width: 120 },
                { title: 'Tracking No', key: 'trackingNo', width: 160 },
                { title: 'Quantity', key: 'quantity', width: 90 },
                { title: 'Shipped At', key: 'shippedAt', width: 160 },
              ]"
              :data="generatedEntries"
              :pagination="{ pageSize: 5 }"
              size="small"
            />
          </div>
        </NCard>
      </NTabPane>
    </NTabs>

    <!-- Existing Shipment List -->
    <NCard :title="t('shipment.list')" class="glow-card">
      <NEmpty v-if="!loadingList && shipments.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="shipmentColumns"
        :data="shipments"
        :loading="loadingList"
        :pagination="{ pageSize: 5 }"
        size="small"
        :row-key="(row: dto.ShipmentDTO) => row.id"
      />
    </NCard>

    <div class="flex justify-between mt-4">
      <NButton @click="router.push(`/waves/${waveId}/export`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}/channel-sync`)">{{ t("wave.nextStep") }}</NButton>
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
