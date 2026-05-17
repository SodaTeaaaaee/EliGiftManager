<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NGrid, NGridItem, NStatistic, NTag } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { createWave, listWaveDashboardRows } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const router = useRouter();
const { t, locale } = useI18n();
const rows = ref<dto.WaveDashboardRowDTO[]>([]);
const loading = ref(false);
const error = ref("");
const creating = ref(false);

const stageTagType: Record<string, "default" | "info" | "success" | "warning" | "error"> = {
  intake: "info",
  allocation: "info",
  review: "warning",
  execution: "warning",
  syncing_back: "info",
  awaiting_manual_closure: "error",
  closed: "default",
};

const activeCount = computed(() =>
  rows.value.filter((row) => row.projectedLifecycleStage !== "closed").length,
);
const closureCount = computed(() =>
  rows.value.filter((row) => row.projectedLifecycleStage === "awaiting_manual_closure").length,
);
const driftCount = computed(() =>
  rows.value.filter((row) => row.projectedLifecycleStage === "syncing_back").length,
);
const recentChangeCount = computed(() => rows.value.slice(0, 5).length);

const columns = computed<DataTableColumns<dto.WaveDashboardRowDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Wave", key: "waveNo", width: 180 },
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
    title: t("dashboard.createdAt"),
    key: "createdAt",
    width: 180,
    render(row) {
      return row.createdAt
        ? new Date(row.createdAt).toLocaleDateString(locale.value)
        : "—";
    },
  },
]);

async function loadRows() {
  loading.value = true;
  error.value = "";
  try {
    rows.value = await listWaveDashboardRows();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function handleCreateWave() {
  creating.value = true;
  try {
    const wave = await createWave(`Wave ${Date.now()}`);
    router.push(`/waves/${wave.id}`);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    creating.value = false;
  }
}

onMounted(loadRows);
</script>

<template>
  <div class="dashboard-page">
    <div class="dashboard-hero">
      <div>
        <div class="app-kicker">{{ t("nav.dashboard") }}</div>
        <h1 class="app-title mt-2">{{ t("dashboard.title") }}</h1>
        <p class="app-copy mt-3">{{ t("dashboard.subtitle") }}</p>
      </div>
      <div class="flex items-center gap-3">
        <NButton secondary @click="router.push('/waves')">
          {{ t("dashboard.openWaves") }}
        </NButton>
        <NButton type="primary" :loading="creating" @click="handleCreateWave">
          {{ t("dashboard.createWave") }}
        </NButton>
      </div>
    </div>

    <NAlert v-if="error" type="error" :title="error" class="mb-4" />

    <NGrid :cols="4" :x-gap="16" :y-gap="16" class="mb-5">
      <NGridItem>
        <NCard>
          <NStatistic :label="t('dashboard.activeWaves')" :value="activeCount" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic :label="t('dashboard.awaitingClosure')" :value="closureCount" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic :label="t('dashboard.driftedBasis')" :value="driftCount" />
        </NCard>
      </NGridItem>
      <NGridItem>
        <NCard>
          <NStatistic :label="t('dashboard.recentChanges')" :value="recentChangeCount" />
        </NCard>
      </NGridItem>
    </NGrid>

    <NCard :title="t('dashboard.waveQueue')">
      <NEmpty v-if="!loading && rows.length === 0" :description="t('dashboard.noWaves')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="rows.slice(0, 6)"
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
