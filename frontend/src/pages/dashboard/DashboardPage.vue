<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { NAlert, NButton, NDataTable, NEmpty, NGrid, NGridItem, NTag, NIcon } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { DocumentTextOutline, WarningOutline, SyncOutline, TimeOutline } from "@vicons/ionicons5";
import { createWave, listWaveDashboardRows } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

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
          bordered: false,
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
  <div class="dashboard-page pb-12">
    <PageHeader 
      :title="t('dashboard.title')" 
      :description="t('dashboard.subtitle')" 
      :kicker="t('nav.dashboard')"
    >
      <template #actions>
        <NButton secondary @click="router.push('/waves')" size="large" round>
          {{ t("dashboard.openWaves") }}
        </NButton>
        <NButton type="primary" :loading="creating" @click="handleCreateWave" size="large" round>
          {{ t("dashboard.createWave") }}
        </NButton>
      </template>
    </PageHeader>

    <NAlert v-if="error" type="error" :title="error" class="mb-6 rounded-lg" />

    <NGrid :cols="4" :x-gap="20" :y-gap="20" class="mb-8">
      <NGridItem>
        <GlassCard hoverable>
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-full bg-blue-500/10 flex items-center justify-center text-blue-500">
              <NIcon size="24"><DocumentTextOutline /></NIcon>
            </div>
            <div>
              <div class="text-3xl font-bold text-slate-800 dark:text-slate-100">{{ activeCount }}</div>
              <div class="text-sm font-medium text-slate-500 mt-1">{{ t('dashboard.activeWaves') }}</div>
            </div>
          </div>
        </GlassCard>
      </NGridItem>
      <NGridItem>
        <GlassCard hoverable>
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500">
              <NIcon size="24"><WarningOutline /></NIcon>
            </div>
            <div>
              <div class="text-3xl font-bold text-slate-800 dark:text-slate-100">{{ closureCount }}</div>
              <div class="text-sm font-medium text-slate-500 mt-1">{{ t('dashboard.awaitingClosure') }}</div>
            </div>
          </div>
        </GlassCard>
      </NGridItem>
      <NGridItem>
        <GlassCard hoverable>
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-full bg-emerald-500/10 flex items-center justify-center text-emerald-500">
              <NIcon size="24"><SyncOutline /></NIcon>
            </div>
            <div>
              <div class="text-3xl font-bold text-slate-800 dark:text-slate-100">{{ driftCount }}</div>
              <div class="text-sm font-medium text-slate-500 mt-1">{{ t('dashboard.driftedBasis') }}</div>
            </div>
          </div>
        </GlassCard>
      </NGridItem>
      <NGridItem>
        <GlassCard hoverable>
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-full bg-purple-500/10 flex items-center justify-center text-purple-500">
              <NIcon size="24"><TimeOutline /></NIcon>
            </div>
            <div>
              <div class="text-3xl font-bold text-slate-800 dark:text-slate-100">{{ recentChangeCount }}</div>
              <div class="text-sm font-medium text-slate-500 mt-1">{{ t('dashboard.recentChanges') }}</div>
            </div>
          </div>
        </GlassCard>
      </NGridItem>
    </NGrid>

    <GlassCard>
      <div class="mb-4 flex items-center justify-between">
        <h2 class="app-heading-md">{{ t('dashboard.waveQueue') }}</h2>
      </div>
      <NEmpty v-if="!loading && rows.length === 0" :description="t('dashboard.noWaves')" class="my-12" />
      <NDataTable
        v-else
        :columns="columns"
        :data="rows.slice(0, 6)"
        :loading="loading"
        :pagination="false"
        size="large"
        :row-props="(row: dto.WaveDashboardRowDTO) => ({
          style: 'cursor:pointer',
          onClick: () => router.push(`/waves/${row.id}`),
        })"
      />
    </GlassCard>
  </div>
</template>

<style scoped>
/* Scoped styles can be minimized thanks to utility classes and shared components */
</style>
