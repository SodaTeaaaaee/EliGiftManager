<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { NButton, NCard, NDataTable, NEmpty, NTabPane, NTabs, NTag } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { createWave, listWaveDashboardRows } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const router = useRouter();
const { t } = useI18n();

const rows = ref<dto.WaveDashboardRowDTO[]>([]);
const loading = ref(false);
const creating = ref(false);
const filter = ref("active");

const stageTagType: Record<string, "default" | "info" | "success" | "warning" | "error"> = {
  intake: "info",
  allocation: "info",
  review: "warning",
  execution: "warning",
  syncing_back: "info",
  awaiting_manual_closure: "error",
  closed: "default",
};

const filteredRows = computed(() => {
  switch (filter.value) {
    case "active":
      return rows.value.filter((row) => row.projectedLifecycleStage !== "closed");
    case "awaiting_closure":
      return rows.value.filter((row) => row.projectedLifecycleStage === "awaiting_manual_closure");
    case "closed":
      return rows.value.filter((row) => row.projectedLifecycleStage === "closed");
    default:
      return rows.value;
  }
});

const columns = computed<DataTableColumns<dto.WaveDashboardRowDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Wave", key: "waveNo", width: 160 },
  { title: "Name", key: "name" },
  {
    title: t("dashboard.stage"),
    key: "projectedLifecycleStage",
    width: 180,
    render(row) {
      return h(
        NTag,
        {
          type: stageTagType[row.projectedLifecycleStage] || "default",
          size: "small",
          round: true,
        },
        { default: () => row.projectedLifecycleStage },
      );
    },
  },
  {
    title: "",
    key: "actions",
    width: 120,
    render(row) {
      return h(
        NButton,
        {
          size: "small",
          type: "primary",
          onClick: () => router.push(`/waves/${row.id}`),
        },
        { default: () => t("waves.openWave") },
      );
    },
  },
]);

async function loadRows() {
  loading.value = true;
  try {
    rows.value = await listWaveDashboardRows();
  } finally {
    loading.value = false;
  }
}

async function handleCreateWave() {
  creating.value = true;
  try {
    const wave = await createWave(`Wave ${Date.now()}`);
    router.push(`/waves/${wave.id}`);
  } finally {
    creating.value = false;
  }
}

onMounted(loadRows);
</script>

<template>
  <div class="waves-page">
    <div class="dashboard-hero">
      <div>
        <div class="app-kicker">{{ t("nav.waves") }}</div>
        <h1 class="app-title mt-2">{{ t("waves.title") }}</h1>
        <p class="app-copy mt-3">{{ t("waves.subtitle") }}</p>
      </div>
      <NButton type="primary" :loading="creating" @click="handleCreateWave">
        {{ t("waves.createWave") }}
      </NButton>
    </div>

    <NCard>
      <NTabs v-model:value="filter" size="small" class="mb-4">
        <NTabPane name="active" :tab="t('waves.active')" />
        <NTabPane name="awaiting_closure" :tab="t('waves.awaitingClosure')" />
        <NTabPane name="closed" :tab="t('waves.closed')" />
        <NTabPane name="all" :tab="t('waves.all')" />
      </NTabs>

      <NEmpty v-if="!loading && filteredRows.length === 0" :description="t('waves.noWaves')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="filteredRows"
        :loading="loading"
        :pagination="false"
        size="small"
        :row-props="(row: dto.WaveDashboardRowDTO) => ({
          style: 'cursor:pointer',
          onClick: () => router.push(`/waves/${row.id}`),
        })"
      />
    </NCard>
  </div>
</template>

<style scoped>
.dashboard-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;
}
</style>
