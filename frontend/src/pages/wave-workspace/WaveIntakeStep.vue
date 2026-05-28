<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NIcon,
  NSpace,
  NTag,
  useMessage,
} from "naive-ui";
import type { DataTableColumns, DataTableRowKey } from "naive-ui";
import {
  ArrowForwardOutline,
  AddCircleOutline,
  InformationCircleOutline,
} from "@vicons/ionicons5";
import {
  assignDemandToWave,
  listDemandInboxRows,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const message = useMessage();

const waveId = computed(() => {
  const id = Number(route.params.waveId);
  return Number.isFinite(id) ? id : null;
});

const loading = ref(false);
const error = ref("");
const allRows = ref<dto.DemandInboxRowDTO[]>([]);
const selectedToPullKeys = ref<DataTableRowKey[]>([]);
const pulling = ref(false);

// 已接手到本波次的 demand documents（只读视图）
const assignedToThisWave = computed(() =>
  allRows.value.filter((row) => row.assignedWaveId === waveId.value),
);

// 可接手的 demand documents：未分配 + 至少有 1 条 ready accepted line
// 这就是"等着被波次拉进来处理"的部分
const availableToPull = computed(() =>
  allRows.value.filter(
    (row) => !row.assigned && (row.readyAcceptedCount ?? 0) > 0,
  ),
);

// ── Common columns ──
const baseColumns = computed<DataTableColumns<dto.DemandInboxRowDTO>>(() => [
  { title: "ID", key: "demandDocumentId", width: 70 },
  { title: t("demandIntake.demandKind"), key: "kind", width: 160, ellipsis: true },
  {
    title: "Profile",
    key: "integrationProfileLabel",
    width: 200,
    ellipsis: { tooltip: true },
  },
  { title: "Source", key: "sourceDocumentNo", width: 160, ellipsis: true },
  {
    title: t("demandIntake.routing.disposition"),
    key: "routing",
    width: 220,
    render(row) {
      const ready = row.readyAcceptedCount ?? 0;
      const waiting = row.waitingInputCount ?? 0;
      const deferred = row.deferredCount ?? 0;
      const excluded = row.excludedCount ?? 0;
      const total = row.totalLineCount ?? 0;
      const tags: any[] = [
        h(
          NTag,
          { size: "tiny", type: "success", round: true, bordered: false },
          { default: () => `r ${ready}` },
        ),
      ];
      if (waiting > 0)
        tags.push(
          h(
            NTag,
            { size: "tiny", type: "warning", round: true, bordered: false },
            { default: () => `w ${waiting}` },
          ),
        );
      if (deferred > 0)
        tags.push(
          h(
            NTag,
            { size: "tiny", round: true, bordered: false },
            { default: () => `d ${deferred}` },
          ),
        );
      if (excluded > 0)
        tags.push(
          h(
            NTag,
            { size: "tiny", round: true, bordered: false },
            { default: () => `x ${excluded}` },
          ),
        );
      tags.push(
        h(
          "span",
          { style: "color: var(--muted); font-size: 11px;" },
          ` /${total}`,
        ),
      );
      return h(
        "div",
        { style: "display:flex;gap:4px;flex-wrap:wrap;align-items:center;" },
        tags,
      );
    },
  },
]);

// 已接手列：附 "查看 / 编辑 routing" 链接到 demand inbox
const assignedColumns = computed<DataTableColumns<dto.DemandInboxRowDTO>>(
  () => [
    ...baseColumns.value,
    {
      title: "Action",
      key: "actions",
      width: 200,
      render(row) {
        return h(
          NButton,
          {
            size: "small",
            quaternary: true,
            onClick: () => goToInboxDetail(row.demandDocumentId),
          },
          {
            default: () => "View / Edit Routing",
            icon: () =>
              h(NIcon, null, { default: () => h(ArrowForwardOutline) }),
          },
        );
      },
    },
  ],
);

// 可接手列：选中行（多选） + 单行 "Pull into this wave" 按钮
const availableColumns = computed<DataTableColumns<dto.DemandInboxRowDTO>>(
  () => [
    { type: "selection", width: 40 },
    ...baseColumns.value,
    {
      title: "Action",
      key: "actions",
      width: 220,
      render(row) {
        return h(NSpace, { size: 4 }, () => [
          h(
            NButton,
            {
              size: "tiny",
              type: "primary",
              loading: pulling.value,
              onClick: () => pullSingle(row.demandDocumentId),
            },
            { default: () => "Pull into this wave" },
          ),
          h(
            NButton,
            {
              size: "tiny",
              quaternary: true,
              onClick: () => goToInboxDetail(row.demandDocumentId),
            },
            { default: () => "Detail" },
          ),
        ]);
      },
    },
  ],
);

async function loadInbox() {
  loading.value = true;
  error.value = "";
  try {
    allRows.value = await listDemandInboxRows({
      assignment: "all",
      demandKind: "",
    });
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function pullSingle(docId: number) {
  if (!waveId.value) return;
  pulling.value = true;
  try {
    await assignDemandToWave(waveId.value, docId);
    message.success(t("demandInbox.assignSuccess"));
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    pulling.value = false;
  }
}

async function pullSelected() {
  if (!waveId.value || selectedToPullKeys.value.length === 0) return;
  pulling.value = true;
  try {
    // Sequential — backend doesn't currently expose batch assign.
    for (const k of selectedToPullKeys.value) {
      await assignDemandToWave(waveId.value, k as number);
    }
    message.success(
      `Pulled ${selectedToPullKeys.value.length} demand document(s) into this wave`,
    );
    selectedToPullKeys.value = [];
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    pulling.value = false;
  }
}

function goToInboxDetail(demandDocumentId: number) {
  router.push({
    path: "/demand-inbox",
    query: { docId: String(demandDocumentId) },
  });
}

function openInbox() {
  router.push("/demand-inbox");
}

watch(
  waveId,
  () => {
    if (waveId.value) void loadInbox();
  },
  { immediate: true },
);

onMounted(() => {
  if (waveId.value) void loadInbox();
});
</script>

<template>
  <div class="wave-intake-step">
    <PageHeader
      :title="t('waveSidebar.intakeView')"
      description="Pull accepted demands from the global Demand Inbox into this wave. Routing decisions are managed in the Inbox."
    >
      <template #actions>
        <NButton secondary @click="loadInbox">Refresh</NButton>
        <NButton secondary @click="openInbox">
          <template #icon>
            <NIcon><ArrowForwardOutline /></NIcon>
          </template>
          {{ t("nav.demandInbox") }}
        </NButton>
      </template>
    </PageHeader>

    <NAlert type="info" class="mb-4" :show-icon="true">
      <template #icon>
        <NIcon><InformationCircleOutline /></NIcon>
      </template>
      Routing / input state edits are upstream-truth-layer concerns and live in
      the global
      <a class="link-text" href="javascript:void(0)" @click="openInbox">
        Demand Inbox
      </a>. Here, the wave decides which already-accepted demands it will
      handle.
    </NAlert>

    <NAlert v-if="error" type="error" class="mb-4" :title="error" closable @close="error = ''" />

    <!-- ── Section 1: Already accepted into this wave ── -->
    <div class="section-heading">
      <span>Accepted into this wave</span>
      <NTag size="small" :bordered="false" round>
        {{ assignedToThisWave.length }}
      </NTag>
    </div>
    <GlassCard class="mb-6">
      <NEmpty
        v-if="!loading && assignedToThisWave.length === 0"
        description="No demand documents have been pulled into this wave yet. Use the table below to pull from the global inbox."
        class="empty-block"
      />
      <NDataTable
        v-else
        :columns="assignedColumns"
        :data="assignedToThisWave"
        :loading="loading"
        :pagination="false"
        :row-key="(row: dto.DemandInboxRowDTO) => row.demandDocumentId"
        size="small"
      />
    </GlassCard>

    <!-- ── Section 2: Available demands to pull ── -->
    <div class="section-heading">
      <span>Available to pull (accepted &amp; unassigned)</span>
      <NTag size="small" :bordered="false" round>
        {{ availableToPull.length }}
      </NTag>
    </div>

    <div v-if="selectedToPullKeys.length > 0" class="bulk-bar mb-3">
      <NSpace align="center">
        <span class="bulk-label">
          {{ selectedToPullKeys.length }} selected
        </span>
        <NButton
          size="small"
          type="primary"
          :loading="pulling"
          @click="pullSelected"
        >
          <template #icon>
            <NIcon><AddCircleOutline /></NIcon>
          </template>
          Pull selected into this wave
        </NButton>
        <NButton size="small" quaternary @click="selectedToPullKeys = []">
          Clear
        </NButton>
      </NSpace>
    </div>

    <GlassCard>
      <NEmpty
        v-if="!loading && availableToPull.length === 0"
        description="No demand documents are currently waiting for intake."
        class="empty-block"
      >
        <template #extra>
          <NButton type="primary" secondary @click="openInbox">
            Open Demand Inbox
          </NButton>
        </template>
      </NEmpty>
      <NDataTable
        v-else
        v-model:checked-row-keys="selectedToPullKeys"
        :columns="availableColumns"
        :data="availableToPull"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
        :row-key="(row: dto.DemandInboxRowDTO) => row.demandDocumentId"
        size="small"
      />
    </GlassCard>
  </div>
</template>

<style scoped>
.wave-intake-step {
  display: flex;
  flex-direction: column;
}

.section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 10px;
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
}

.bulk-bar {
  padding: 10px 14px;
  border-radius: 10px;
  background: rgba(99, 102, 241, 0.06);
  border: 1px solid rgba(99, 102, 241, 0.18);
}

.bulk-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text);
}

.empty-block {
  padding: 32px 0;
}

.link-text {
  color: var(--accent);
  text-decoration: underline;
  cursor: pointer;
}

.mb-3 {
  margin-bottom: 12px;
}

.mb-4 {
  margin-bottom: 16px;
}

.mb-6 {
  margin-bottom: 24px;
}
</style>
