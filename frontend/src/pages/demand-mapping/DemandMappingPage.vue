<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NTag, NSpace, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { generateParticipants, listAssignedDemandsByWave, listDemandLines, mapDemandLines } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const docs = ref<dto.DemandDocumentDTO[]>([]);
const loading = ref(false);
const applying = ref(false);
const participantsGenerated = ref(false);
const lineCache = ref<Record<number, dto.DemandLineDTO[]>>({});
const expandedKeys = ref<number[]>([]);
const blockedSummary = ref<string>("");

const columns = computed<DataTableColumns<dto.DemandDocumentDTO>>(() => [
  { type: "expand" },
  { title: "ID", key: "id", width: 60 },
  { title: "Kind", key: "kind", width: 180 },
  { title: "Source", key: "sourceDocumentNo", width: 180 },
  { title: "Profile", key: "customerProfileId", width: 100 },
  { title: "Channel", key: "sourceChannel", width: 100 },
]);

const lineColumns: DataTableColumns<dto.DemandLineDTO> = [
  { title: t("mapping.columns.line"), key: "sourceLineNo", width: 70 },
  { title: t("mapping.columns.type"), key: "lineType", width: 120 },
  { title: t("mapping.columns.title"), key: "externalTitle" },
  { title: t("mapping.columns.disposition"), key: "routingDisposition", width: 140 },
  { title: t("mapping.columns.input"), key: "recipientInputState", width: 140 },
  { title: t("mapping.columns.qty"), key: "requestedQuantity", width: 70 },
];

async function loadDocs() {
  loading.value = true;
  try {
    docs.value = await listAssignedDemandsByWave(waveId.value);
  } finally {
    loading.value = false;
  }
}

async function loadLines(docId: number) {
  if (lineCache.value[docId]) return;
  lineCache.value[docId] = await listDemandLines(docId);
}

function renderExpand(row: dto.DemandDocumentDTO) {
  const lines = lineCache.value[row.id] || [];
  return h(NDataTable, {
    columns: lineColumns,
    data: lines,
    size: "small",
    bordered: false,
    pagination: false,
  });
}

async function handleGenerateParticipants() {
  try {
    const count = await generateParticipants(waveId.value);
    participantsGenerated.value = true;
    message.success(`${t("mapping.generateParticipants")}: ${count}`);
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  }
}

async function handleMap() {
  applying.value = true;
  blockedSummary.value = "";
  try {
    const result = await mapDemandLines(waveId.value);
    if (result.blockedLines?.length) {
      blockedSummary.value = result.blockedLines
        .map((line) => line.demandLineTitle || `#${line.demandLineId}`)
        .join(", ");
    }
    message.success(t("mapping.mapDemandOk"));
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  } finally {
    applying.value = false;
  }
}

onMounted(loadDocs);
</script>

<template>
  <div class="demand-mapping-page">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.mapping") }}</div>
      <h2 class="app-title mt-2">{{ t("mapping.title") }}</h2>
      <p class="app-copy mt-3">{{ t("mapping.subtitle") }}</p>
    </div>

    <NCard class="mb-4" :title="t('mapping.currentRhythm')">
      <NSpace vertical :size="10">
        <div>1. {{ t("mapping.instructionsStep1") }}</div>
        <div>2. {{ t("mapping.instructionsStep2") }}</div>
        <div>3. {{ t("mapping.instructionsStep3") }}</div>
      </NSpace>
    </NCard>

    <NAlert v-if="blockedSummary" type="warning" class="mb-4">
      {{ blockedSummary }}
    </NAlert>

    <NCard :title="t('mapping.assigned')">
      <template #header-extra>
        <NSpace>
          <NButton size="small" @click="handleGenerateParticipants">
            {{ t("mapping.generateParticipants") }}
          </NButton>
          <NButton
            size="small"
            type="primary"
            :disabled="!participantsGenerated"
            :loading="applying"
            @click="handleMap"
          >
            {{ t("mapping.mapDemand") }}
          </NButton>
        </NSpace>
      </template>

      <NEmpty v-if="!loading && docs.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="docs"
        :loading="loading"
        :pagination="false"
        size="small"
        :expanded-row-keys="expandedKeys"
        :render-expand="renderExpand"
        @update:expanded-row-keys="(keys) => {
          expandedKeys = keys as number[]
          for (const key of keys as number[]) {
            void loadLines(key)
          }
        }"
      />
    </NCard>

    <div class="mt-4 flex justify-between">
      <NButton @click="router.push(`/waves/${waveId}`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}`)">{{ t("wave.nextStep") }}</NButton>
      </NSpace>
    </div>
  </div>
</template>
