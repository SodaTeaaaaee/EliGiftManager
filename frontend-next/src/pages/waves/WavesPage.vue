<script setup lang="ts">
/**
 * WavesPage — the wave list (plan 3.2): create/rename/close a wave, filter
 * by keyword + lifecycle stage (client-side — `listWavesFiltered`'s
 * `PaginationInput` carries no filter fields yet), and deep-link into the
 * wave workspace on row click.
 */
import { computed, h, onBeforeMount, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { NAlert, NButton } from "naive-ui";
import { PageHeader } from "@/shared/ui/shell";
import { FilterBar, useUrlFilters, type FilterSchema } from "@/shared/ui/filter-bar";
import { DataGrid, createColumns, type DataGridColumnSpec } from "@/shared/ui/data-grid";
import { useFeedback } from "@/shared/ui/feedback";
import { listWavesFiltered } from "@/shared/api/bridge";
import { registerRefreshTarget } from "@/shared/lib/view-hotkeys";
import type { dto } from "../../../wailsjs/go/models";
import CreateWaveDialog from "./components/CreateWaveDialog.vue";
import RenameWaveDialog from "./components/RenameWaveDialog.vue";
import CloseWaveDialog from "./components/CloseWaveDialog.vue";

const { t } = useI18n({ useScope: "global" });
const router = useRouter();
const feedback = useFeedback();

const WAVE_LIST_PAGE_SIZE = 200;

const allWaves = ref<dto.WaveDTO[]>([]);
const loading = ref(true);
const totalWaveCount = ref(0);

// `listWavesFiltered`'s PaginationInput has no filter fields (5.4 note) — the
// wave count is small enough that client-side filtering below is sufficient.
async function loadWaves(): Promise<void> {
  loading.value = true;
  try {
    const page = await listWavesFiltered({
      page: 1,
      pageSize: WAVE_LIST_PAGE_SIZE,
      sortBy: "updatedAt",
      sortDesc: true,
    });
    allWaves.value = page.items;
    totalWaveCount.value = page.pagination.totalCount;
  } catch (err) {
    feedback.error(t("feedback.error"), err instanceof Error ? err.message : String(err));
  } finally {
    loading.value = false;
  }
}

onMounted(loadWaves);

let unregisterRefresh: (() => void) | undefined;
onBeforeMount(() => {
  unregisterRefresh = registerRefreshTarget(loadWaves);
});
onBeforeUnmount(() => unregisterRefresh?.());

// ── Filters (URL-synced; applied client-side over `allWaves`) ──

const schema = [
  { key: "keyword", type: "keyword" },
  { key: "lifecycleStage", type: "enum-multi", dimension: "lifecycleStage" },
] as const satisfies FilterSchema;

const filters = useUrlFilters(schema);

const filteredRows = computed<dto.WaveDTO[]>(() => {
  const keyword = filters.state.keyword.trim().toLowerCase();
  const stages = filters.state.lifecycleStage;
  return allWaves.value.filter((wave) => {
    if (stages.length > 0 && !stages.includes(wave.lifecycleStage)) return false;
    if (keyword.length > 0 && !`${wave.name} ${wave.waveNo}`.toLowerCase().includes(keyword)) return false;
    return true;
  });
});

const isWaveListTruncated = computed(() => totalWaveCount.value > allWaves.value.length);

// ── Navigation ──

function openWorkspace(wave: dto.WaveDTO): void {
  router.push({ name: "wave-workspace", params: { id: wave.id } });
}

// ── Dialogs ──

const showCreate = ref(false);
const renameTarget = ref<dto.WaveDTO | null>(null);
const closeTarget = ref<dto.WaveDTO | null>(null);

function onRenameDialogVisibility(visible: boolean): void {
  if (!visible) renameTarget.value = null;
}

function onCloseDialogVisibility(visible: boolean): void {
  if (!visible) closeTarget.value = null;
}

function onCreated(wave: dto.WaveDTO): void {
  showCreate.value = false;
  void loadWaves();
  feedback.success(t("wavesList.feedback.created"));
  feedback.receipt({ kind: "action", summary: `${t("wavesList.feedback.created")} · ${wave.name}` });
}

function onRenamed(wave: dto.WaveDTO): void {
  renameTarget.value = null;
  void loadWaves();
  feedback.success(t("wavesList.feedback.renamed"));
  feedback.receipt({ kind: "action", summary: `${t("wavesList.feedback.renamed")} · ${wave.name}` });
}

function onClosed(result: dto.CloseWaveResult): void {
  closeTarget.value = null;
  void loadWaves();
  const message =
    result.forced && result.residualItemCount > 0
      ? t("wavesList.feedback.closeForced", { count: result.residualItemCount })
      : t("wavesList.feedback.closed");
  feedback.success(message);
  feedback.receipt({ kind: "action", summary: `${message} · ${result.wave.name}` });
}

// ── Grid columns ──

const columns = computed(() => {
  const specs: DataGridColumnSpec<dto.WaveDTO>[] = [
    { type: "text", key: "waveNo", title: t("wavesList.columns.waveNo"), width: 140 },
    { type: "text", key: "name", title: t("wavesList.columns.name"), minWidth: 200 },
    {
      type: "text",
      key: "waveType",
      title: t("wavesList.columns.type"),
      width: 120,
      sortable: false,
      getValue: (row) => t(`wavesList.waveType.${row.waveType}`),
    },
    {
      type: "status",
      key: "lifecycleStage",
      title: t("wavesList.columns.stage"),
      dimension: "lifecycleStage",
      width: 170,
      showDot: true,
    },
    { type: "date", key: "createdAt", title: t("wavesList.columns.createdAt"), width: 130 },
    { type: "date", key: "updatedAt", title: t("wavesList.columns.updatedAt"), width: 130 },
    {
      type: "actions",
      key: "actions",
      title: t("wavesList.columns.actions"),
      width: 240,
      render: (row) =>
        h("div", { class: "waves-page__row-actions" }, [
          h(
            NButton,
            {
              size: "tiny",
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation();
                openWorkspace(row);
              },
            },
            { default: () => t("wavesList.rowActions.open") },
          ),
          h(
            NButton,
            {
              size: "tiny",
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation();
                renameTarget.value = row;
              },
            },
            { default: () => t("wavesList.rowActions.rename") },
          ),
          h(
            NButton,
            {
              size: "tiny",
              quaternary: true,
              onClick: (event: MouseEvent) => {
                event.stopPropagation();
                closeTarget.value = row;
              },
            },
            { default: () => t("wavesList.rowActions.close") },
          ),
        ]),
    },
  ];
  return createColumns<dto.WaveDTO>(specs);
});
</script>

<template>
  <div class="waves-page">
    <PageHeader :title="t('wavesList.title')" :description="t('wavesList.subtitle')">
      <template #actions>
        <NButton type="primary" @click="showCreate = true">{{ t("wavesList.create") }}</NButton>
      </template>
    </PageHeader>

    <FilterBar :filters="filters" />

    <NAlert
      v-if="isWaveListTruncated"
      type="info"
      :show-icon="false"
      class="waves-page__truncation-notice"
    >
      {{ t("wavesList.truncationNotice", { shown: allWaves.length, total: totalWaveCount }) }}
    </NAlert>

    <DataGrid
      :columns="columns"
      :rows="filteredRows"
      row-key="id"
      :loading="loading"
      pagination="client"
      :empty="{ title: t('wavesList.empty.title'), description: t('wavesList.empty.description') }"
      @row-click="openWorkspace"
    />

    <CreateWaveDialog v-model:show="showCreate" @created="onCreated" />
    <RenameWaveDialog
      v-if="renameTarget"
      :show="true"
      :wave="renameTarget"
      @update:show="onRenameDialogVisibility"
      @renamed="onRenamed"
    />
    <CloseWaveDialog
      v-if="closeTarget"
      :show="true"
      :wave="closeTarget"
      @update:show="onCloseDialogVisibility"
      @closed="onClosed"
    />
  </div>
</template>

<style scoped>
.waves-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
</style>

<style>
/* Unscoped: `createColumns`' `actions` render() runs outside this SFC's
   scoped subtree (same reasoning as DataGrid's skeleton-bar class). */
.waves-page__row-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}
</style>
