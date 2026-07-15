<script setup lang="ts">
import { computed, onMounted, reactive, ref, h } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NButton, NCard, NDataTable, NDrawer, NDrawerContent, NEmpty, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace, NTag, NGrid, NGridItem, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { listProductsByWave, listWaveFulfillmentRows, recordAdjustment } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const rows = ref<dto.WaveFulfillmentRowDTO[]>([]);
const products = ref<dto.ProductDTO[]>([]);
const loading = ref(false);
const drawerVisible = ref(false);
const selectedRow = ref<dto.WaveFulfillmentRowDTO | null>(null);
const submitting = ref(false);

// Filter & Search State
const searchKeyword = ref("");
const activeFilter = ref<'all' | 'review' | 'drift' | 'address' | 'adjusted'>('all');

const form = reactive({
  targetKind: "fulfillment_line",
  adjustmentKind: "add",
  quantityDelta: 1,
  reasonCode: "",
  note: "",
  evidenceRef: "",
  operatorId: "Operator",
  fromProductId: null as number | null,
  toProductId: null as number | null,
});

const adjustmentOptions = computed(() => [
  { label: t("adjustment.add"), value: "add" },
  { label: t("adjustment.reduce"), value: "reduce" },
  { label: t("adjustment.remove"), value: "remove" },
  { label: t("adjustment.replace"), value: "replace" },
]);

function reviewText(value: string) {
  const map: Record<string, string> = {
    none: t("adjustment.reviewState.none"),
    recommended: t("adjustment.reviewState.recommended"),
    required: t("adjustment.reviewState.required"),
  };
  return map[value] || value;
}

const columns = computed<DataTableColumns<dto.WaveFulfillmentRowDTO>>(() => [
  { title: t("adjustment.columns.id"), key: "fulfillmentLineId", width: 75 },
  { title: t("adjustment.participant"), key: "participantDisplay", width: 180 },
  { title: t("adjustment.columns.product"), key: "productDisplay", width: 220 },
  { title: t("adjustment.columns.source"), key: "demandSourceSummary", width: 200 },
  { title: t("adjustment.quantity"), key: "quantity", width: 80 },
  { 
    title: t("adjustment.columns.supplier"), 
    key: "supplierState", 
    width: 120,
    render(row) {
      const type = row.supplierState === "shipped" ? "success" : row.supplierState === "submitted" ? "info" : "default";
      return h(NTag, { type, size: "small", bordered: false }, { default: () => row.supplierState || "unsubmitted" });
    }
  },
  { 
    title: t("adjustment.columns.sync"), 
    key: "channelSyncState", 
    width: 120,
    render(row) {
      const type = row.channelSyncState === "synced" ? "success" : row.channelSyncState === "failed" ? "error" : "default";
      return h(NTag, { type, size: "small", bordered: false }, { default: () => row.channelSyncState || "pending" });
    }
  },
  {
    title: t("adjustment.columns.review"),
    key: "reviewRequirement",
    width: 130,
    render(row) {
      const type =
        row.reviewRequirement === "required"
          ? "error"
          : row.reviewRequirement === "recommended"
            ? "warning"
            : "default";
      return h(NTag, { type, size: "small", round: true, bordered: false }, { default: () => reviewText(row.reviewRequirement) });
    },
  },
]);

// Statistics for filters
const reviewRequiredCount = computed(() => rows.value.filter(r => r.reviewRequirement === "required").length);
const driftedCount = computed(() => rows.value.filter(r => r.basisDriftStatus === "drifted" || r.reviewRequirement === "recommended").length);
const addressIssuesCount = computed(() => rows.value.filter(r => r.addressState === "missing" || r.addressState === "invalid" || r.addressState === "unverified").length);
const adjustedCount = computed(() => rows.value.filter(r => r.generatedBy === "manual" || r.lineReason !== "").length);

const filteredRows = computed(() => {
  return rows.value.filter((row) => {
    // 1. Text Search Filter
    const matchesKeyword = !searchKeyword.value ||
      row.participantDisplay.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      row.productDisplay.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      (row.demandSourceSummary && row.demandSourceSummary.toLowerCase().includes(searchKeyword.value.toLowerCase()));
      
    if (!matchesKeyword) return false;

    // 2. Status Category Filter
    if (activeFilter.value === "review") return row.reviewRequirement === "required";
    if (activeFilter.value === "drift") return row.basisDriftStatus === "drifted" || row.reviewRequirement === "recommended";
    if (activeFilter.value === "address") return row.addressState === "missing" || row.addressState === "invalid" || row.addressState === "unverified";
    if (activeFilter.value === "adjusted") return row.generatedBy === "manual" || row.lineReason !== "";
    
    return true;
  });
});

async function loadRows() {
  if (!waveId.value) return;
  loading.value = true;
  try {
    const [rowsResult, productsResult] = await Promise.all([
      listWaveFulfillmentRows(waveId.value),
      listProductsByWave(waveId.value),
    ]);
    rows.value = rowsResult;
    products.value = productsResult;
  } finally {
    loading.value = false;
  }
}

function openDrawer(row: dto.WaveFulfillmentRowDTO) {
  selectedRow.value = row;
  form.targetKind = "fulfillment_line";
  form.adjustmentKind = "add";
  form.quantityDelta = 1;
  form.reasonCode = "";
  form.note = "";
  form.evidenceRef = "";
  form.operatorId = "Operator";
  form.fromProductId = null;
  form.toProductId = null;
  drawerVisible.value = true;
}

// Quick apply templates inside editing drawer
function applyPreset(preset: 'add_one' | 'gift' | 'replace' | 'remove') {
  if (preset === 'add_one') {
    form.adjustmentKind = 'add';
    form.quantityDelta = 1;
    form.reasonCode = 'additional_qty';
    form.note = 'Customer requested additional unit.';
  } else if (preset === 'gift') {
    form.adjustmentKind = 'add';
    form.quantityDelta = 1;
    form.reasonCode = 'complementary_gift';
    form.note = 'Added as campaign gift.';
  } else if (preset === 'replace') {
    form.adjustmentKind = 'replace';
    form.reasonCode = 'sku_replacement';
    form.note = 'Replaced due to inventory out of stock.';
    if (selectedRow.value && selectedRow.value.productId) {
      form.fromProductId = selectedRow.value.productId;
    }
  } else if (preset === 'remove') {
    form.adjustmentKind = 'remove';
    form.reasonCode = 'manual_cancellation';
    form.note = 'Customer canceled this fulfillment line.';
  }
}

async function handleSubmit() {
  if (!selectedRow.value) return;
  submitting.value = true;
  try {
    await recordAdjustment({
      waveId: waveId.value,
      targetKind: form.adjustmentKind === "replace" ? "fulfillment_line" : form.targetKind,
      fulfillmentLineId: selectedRow.value.fulfillmentLineId,
      adjustmentKind: form.adjustmentKind,
      quantityDelta: (form.adjustmentKind === "remove" || form.adjustmentKind === "replace") ? 0 : form.quantityDelta,
      reasonCode: form.reasonCode,
      note: form.note,
      evidenceRef: form.evidenceRef,
      operatorId: form.operatorId,
      fromProductId: form.adjustmentKind === "replace" ? form.fromProductId : null,
      toProductId: form.adjustmentKind === "replace" ? form.toProductId : null,
    });
    message.success(t("adjustment.create"));
    drawerVisible.value = false;
    await loadRows();
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  } finally {
    submitting.value = false;
  }
}

onMounted(loadRows);
</script>

<template>
  <div class="adjustment-review-page flex flex-col gap-5">
    <div class="mb-2">
      <div class="app-kicker">{{ t("wave.adjustment") }}</div>
      <h2 class="app-title mt-2">{{ t("adjustment.title") }}</h2>
      <p class="app-copy mt-2">{{ t("adjustment.subtitle") }}</p>
    </div>

    <!-- Quick filter cards -->
    <NGrid :cols="5" :x-gap="12" :y-gap="12" class="mb-2">
      <NGridItem>
        <div 
          :class="['filter-card', activeFilter === 'all' ? 'is-active' : '']"
          @click="activeFilter = 'all'"
        >
          <span class="filter-card__label">All Lines</span>
          <span class="filter-card__value">{{ rows.length }}</span>
        </div>
      </NGridItem>
      <NGridItem>
        <div 
          :class="['filter-card is-error', activeFilter === 'review' ? 'is-active' : '']"
          @click="activeFilter = 'review'"
        >
          <span class="filter-card__label">Requires Review</span>
          <span class="filter-card__value">{{ reviewRequiredCount }}</span>
        </div>
      </NGridItem>
      <NGridItem>
        <div 
          :class="['filter-card is-warning', activeFilter === 'drift' ? 'is-active' : '']"
          @click="activeFilter = 'drift'"
        >
          <span class="filter-card__label">Drifted</span>
          <span class="filter-card__value">{{ driftedCount }}</span>
        </div>
      </NGridItem>
      <NGridItem>
        <div 
          :class="['filter-card is-address', activeFilter === 'address' ? 'is-active' : '']"
          @click="activeFilter = 'address'"
        >
          <span class="filter-card__label">Address Issues</span>
          <span class="filter-card__value">{{ addressIssuesCount }}</span>
        </div>
      </NGridItem>
      <NGridItem>
        <div 
          :class="['filter-card is-adjusted', activeFilter === 'adjusted' ? 'is-active' : '']"
          @click="activeFilter = 'adjusted'"
        >
          <span class="filter-card__label">Adjusted</span>
          <span class="filter-card__value">{{ adjustedCount }}</span>
        </div>
      </NGridItem>
    </NGrid>

    <!-- Main Table Card -->
    <NCard class="glow-card">
      <template #header-extra>
        <NInput 
          v-model:value="searchKeyword" 
          placeholder="Search participant or product..." 
          clearable 
          size="small" 
          style="width: 260px" 
        />
      </template>

      <NEmpty v-if="!loading && rows.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="filteredRows"
        :loading="loading"
        :pagination="{ page: 1, pageSize: 10, showSizePicker: true, pageSizes: [10, 20, 50, 100] }"
        size="small"
        :row-props="(row: dto.WaveFulfillmentRowDTO) => ({
          style: 'cursor:pointer',
          onClick: () => openDrawer(row),
        })"
      />
    </NCard>

    <div class="flex justify-between mt-4">
      <NButton @click="router.push(`/waves/${waveId}/demand-mapping`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}/export`)">{{ t("wave.nextStep") }}</NButton>
      </NSpace>
    </div>

    <!-- Right Drawer for Editing Details & Presets -->
    <NDrawer v-model:show="drawerVisible" :width="500" placement="right">
      <NDrawerContent :title="selectedRow ? `${t('adjustment.line')} #${selectedRow.fulfillmentLineId}` : t('adjustment.noSelection')" closable>
        <template v-if="selectedRow">
          <NSpace vertical :size="16">
            <!-- Selected Item Summary -->
            <NCard size="small" style="background: rgba(148, 163, 184, 0.05);">
              <div class="text-sm font-bold">{{ selectedRow.participantDisplay }}</div>
              <div class="text-xs text-slate-400 mt-1">Current SKU: {{ selectedRow.productDisplay }}</div>
              <div class="text-xs text-slate-400 mt-1">Quantity: {{ selectedRow.quantity }}</div>
              <div class="text-xs text-slate-500 mt-2 italic" v-if="selectedRow.demandSourceSummary">
                Source: {{ selectedRow.demandSourceSummary }}
              </div>
            </NCard>

            <!-- Quick Adjust Presets -->
            <div>
              <div class="text-xs font-bold text-slate-500 uppercase tracking-wide mb-2">Quick Presets</div>
              <NSpace>
                <NButton size="small" secondary @click="applyPreset('add_one')">+1 Qty</NButton>
                <NButton size="small" secondary @click="applyPreset('gift')">Campaign Gift</NButton>
                <NButton size="small" secondary @click="applyPreset('replace')">Replace SKU</NButton>
                <NButton size="small" type="error" secondary @click="applyPreset('remove')">Remove Line</NButton>
              </NSpace>
            </div>

            <!-- Form -->
            <NForm label-placement="top" class="border-t border-slate-700/10 dark:border-slate-700/30 pt-4 mt-2">
              <NFormItem :label="t('adjustment.reason')">
                <NSelect v-model:value="form.adjustmentKind" :options="adjustmentOptions" />
              </NFormItem>
              <NFormItem :label="t('adjustment.quantity')" v-if="form.adjustmentKind !== 'remove' && form.adjustmentKind !== 'replace'">
                <NInputNumber v-model:value="form.quantityDelta" :min="1" class="w-full" />
              </NFormItem>
              <NFormItem :label="t('adjustment.fromProduct')" v-if="form.adjustmentKind === 'replace'">
                <NSelect
                  v-model:value="form.fromProductId"
                  :options="products.map(p => ({ label: p.name, value: p.id }))"
                  clearable
                  class="w-full"
                  filterable
                />
              </NFormItem>
              <NFormItem :label="t('adjustment.toProduct')" v-if="form.adjustmentKind === 'replace'">
                <NSelect
                  v-model:value="form.toProductId"
                  :options="products.map(p => ({ label: p.name, value: p.id }))"
                  clearable
                  class="w-full"
                  filterable
                />
              </NFormItem>
              <NFormItem label="Reason Code" required>
                <NInput v-model:value="form.reasonCode" placeholder="e.g. additional_qty, manual_cancellation" />
              </NFormItem>
              <NFormItem :label="t('adjustment.note')">
                <NInput v-model:value="form.note" type="textarea" :rows="3" placeholder="Enter comments about this adjustment..." />
              </NFormItem>
              <NFormItem :label="t('adjustment.form.evidenceRef')">
                <NInput v-model:value="form.evidenceRef" placeholder="Link to ticket, invoice or discord message" />
              </NFormItem>
              <NFormItem :label="t('adjustment.form.operatorId')">
                <NInput v-model:value="form.operatorId" />
              </NFormItem>
            </NForm>

            <NButton type="primary" :loading="submitting" @click="handleSubmit" class="w-full mt-2">
              {{ t("adjustment.create") }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
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

/* Filter Cards Layout */
.filter-card {
  display: flex;
  flex-direction: column;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--surface-strong);
  border: 1px solid rgba(148, 163, 184, 0.12);
  cursor: pointer;
  user-select: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

:root[data-theme='dark'] .filter-card {
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.filter-card__label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
}

.filter-card__value {
  font-size: 24px;
  font-weight: 800;
  margin-top: 4px;
  color: var(--text);
}

/* Active and Hover States */
.filter-card:hover {
  transform: translateY(-1px);
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.05);
}

.filter-card.is-active {
  border-color: var(--accent);
  background: var(--accent-surface);
  box-shadow: 0 0 0 1px var(--accent);
}

.filter-card.is-active .filter-card__label {
  color: var(--accent);
}

/* Color codes for special cards */
.filter-card.is-error.is-active {
  border-color: var(--error-color, #dc2626);
  background: rgba(220, 38, 38, 0.08);
  box-shadow: 0 0 0 1px var(--error-color, #dc2626);
}
.filter-card.is-error.is-active .filter-card__label,
.filter-card.is-error.is-active .filter-card__value {
  color: var(--error-color, #dc2626);
}

.filter-card.is-warning.is-active {
  border-color: var(--warning-color, #d97706);
  background: rgba(217, 119, 6, 0.08);
  box-shadow: 0 0 0 1px var(--warning-color, #d97706);
}
.filter-card.is-warning.is-active .filter-card__label,
.filter-card.is-warning.is-active .filter-card__value {
  color: var(--warning-color, #d97706);
}

.filter-card.is-address.is-active {
  border-color: #a855f7;
  background: rgba(168, 85, 247, 0.08);
  box-shadow: 0 0 0 1px #a855f7;
}
.filter-card.is-address.is-active .filter-card__label,
.filter-card.is-address.is-active .filter-card__value {
  color: #a855f7;
}

.filter-card.is-adjusted.is-active {
  border-color: #06b6d4;
  background: rgba(6, 182, 212, 0.08);
  box-shadow: 0 0 0 1px #06b6d4;
}
.filter-card.is-adjusted.is-active .filter-card__label,
.filter-card.is-adjusted.is-active .filter-card__value {
  color: #06b6d4;
}
</style>
