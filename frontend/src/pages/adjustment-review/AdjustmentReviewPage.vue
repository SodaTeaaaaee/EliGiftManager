<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NButton, NCard, NDataTable, NDrawer, NDrawerContent, NEmpty, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace, NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { listWaveFulfillmentRows, recordAdjustment } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const rows = ref<dto.WaveFulfillmentRowDTO[]>([]);
const loading = ref(false);
const drawerVisible = ref(false);
const selectedRow = ref<dto.WaveFulfillmentRowDTO | null>(null);
const submitting = ref(false);

const form = reactive({
  targetKind: "fulfillment_line",
  adjustmentKind: "add",
  quantityDelta: 1,
  reasonCode: "",
  note: "",
  evidenceRef: "",
  operatorId: "",
});

const adjustmentOptions = computed(() => [
  { label: t("adjustment.add"), value: "add" },
  { label: t("adjustment.reduce"), value: "reduce" },
  { label: t("adjustment.remove"), value: "remove" },
]);

const columns = computed<DataTableColumns<dto.WaveFulfillmentRowDTO>>(() => [
  { title: "ID", key: "fulfillmentLineId", width: 70 },
  { title: t("adjustment.participant"), key: "participantDisplay", width: 180 },
  { title: "Product", key: "productDisplay", width: 200 },
  { title: "Source", key: "demandSourceSummary", width: 180 },
  { title: t("adjustment.quantity"), key: "quantity", width: 80 },
  { title: "Supplier", key: "supplierState", width: 110 },
  { title: "Sync", key: "channelSyncState", width: 110 },
  {
    title: "Review",
    key: "reviewRequirement",
    width: 120,
    render(row) {
      const type =
        row.reviewRequirement === "required"
          ? "error"
          : row.reviewRequirement === "recommended"
            ? "warning"
            : "default";
      return h(NTag, { type, size: "small", round: true }, { default: () => row.reviewRequirement });
    },
  },
]);

async function loadRows() {
  if (!waveId.value) return;
  loading.value = true;
  try {
    rows.value = await listWaveFulfillmentRows(waveId.value);
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
  form.operatorId = "";
  drawerVisible.value = true;
}

async function handleSubmit() {
  if (!selectedRow.value) return;
  submitting.value = true;
  try {
    await recordAdjustment({
      waveId: waveId.value,
      targetKind: form.targetKind,
      fulfillmentLineId: selectedRow.value.fulfillmentLineId,
      adjustmentKind: form.adjustmentKind,
      quantityDelta: form.adjustmentKind === "remove" ? 0 : form.quantityDelta,
      reasonCode: form.reasonCode,
      note: form.note,
      evidenceRef: form.evidenceRef,
      operatorId: form.operatorId,
    });
    message.success(t("adjustment.create"));
    drawerVisible.value = false;
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  } finally {
    submitting.value = false;
  }
}

onMounted(loadRows);
</script>

<template>
  <div class="adjustment-review-page">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.adjustment") }}</div>
      <h2 class="app-title mt-2">{{ t("adjustment.title") }}</h2>
      <p class="app-copy mt-3">{{ t("adjustment.subtitle") }}</p>
    </div>

    <NCard class="mb-4" title="Editing Intent">
      <NSpace vertical :size="10">
        <div>Use this step only for wave-local final fulfillment exceptions.</div>
        <div>Go back to Membership Allocation or Demand Mapping if you need to change default generation logic.</div>
      </NSpace>
    </NCard>

    <NCard>
      <NEmpty v-if="!loading && rows.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="rows"
        :loading="loading"
        :pagination="false"
        size="small"
        :row-props="(row: dto.WaveFulfillmentRowDTO) => ({
          style: 'cursor:pointer',
          onClick: () => openDrawer(row),
        })"
      />
    </NCard>

    <div class="mt-4 flex justify-between">
      <NButton @click="router.push(`/waves/${waveId}`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
      </NSpace>
    </div>

    <NDrawer v-model:show="drawerVisible" :width="480" placement="right">
      <NDrawerContent :title="selectedRow ? `${t('adjustment.line')} #${selectedRow.fulfillmentLineId}` : t('adjustment.noSelection')" closable>
        <template v-if="selectedRow">
          <NSpace vertical :size="16">
            <NCard size="small">
              <div><strong>{{ selectedRow.participantDisplay }}</strong></div>
              <div>{{ selectedRow.productDisplay }}</div>
              <div class="app-copy mt-2">{{ selectedRow.demandSourceSummary }}</div>
            </NCard>

            <NForm label-placement="top">
              <NFormItem :label="t('adjustment.reason')">
                <NSelect v-model:value="form.adjustmentKind" :options="adjustmentOptions" />
              </NFormItem>
              <NFormItem :label="t('adjustment.quantity')" v-if="form.adjustmentKind !== 'remove'">
                <NInputNumber v-model:value="form.quantityDelta" :min="1" class="w-full" />
              </NFormItem>
              <NFormItem :label="t('adjustment.reason')">
                <NInput v-model:value="form.reasonCode" />
              </NFormItem>
              <NFormItem :label="t('adjustment.note')">
                <NInput v-model:value="form.note" type="textarea" :rows="3" />
              </NFormItem>
              <NFormItem label="Evidence Ref">
                <NInput v-model:value="form.evidenceRef" />
              </NFormItem>
              <NFormItem label="Operator ID">
                <NInput v-model:value="form.operatorId" />
              </NFormItem>
            </NForm>

            <NButton type="primary" :loading="submitting" @click="handleSubmit">
              {{ t("adjustment.create") }}
            </NButton>
          </NSpace>
        </template>
      </NDrawerContent>
    </NDrawer>
  </div>
</template>
