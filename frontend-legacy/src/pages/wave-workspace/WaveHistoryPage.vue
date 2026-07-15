<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NIcon,
  NPopconfirm,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  ArrowBackOutline,
  PinOutline,
  TrashOutline,
} from "@vicons/ionicons5";
import {
  getHistoryGraph,
  runHistoryGC,
  type HistoryGraphDTO,
  type HistoryGraphNodeDTO,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";

import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const message = useMessage();

const waveId = computed(() => {
  const id = Number(route.params.waveId);
  return Number.isFinite(id) ? id : null;
});

const graph = ref<HistoryGraphDTO | null>(null);
const loading = ref(false);
const error = ref("");
const activeTab = ref("nodes");

async function load() {
  if (!waveId.value) return;
  loading.value = true;
  error.value = "";
  try {
    graph.value = await getHistoryGraph(waveId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function handleGC() {
  if (!waveId.value) return;
  try {
    const n = await runHistoryGC(waveId.value);
    message.success(t("waveHistory.cleanUpDone").replace("{n}", String(n)));
    await load();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

watch(waveId, () => {
  void load();
});

onMounted(load);

function backToWorkspace() {
  if (waveId.value) router.push(`/waves/${waveId.value}`);
}

// ── Build flat-tree (pre-order traversal with depth) for outline view ──
interface FlatTreeRow {
  node: HistoryGraphNodeDTO;
  depth: number;
}

const flatTree = computed<FlatTreeRow[]>(() => {
  const nodes = graph.value?.nodes || [];
  const childrenById = new Map<number, HistoryGraphNodeDTO[]>();
  const rootNodes: HistoryGraphNodeDTO[] = [];
  for (const n of nodes) {
    if (n.parentNodeId && nodes.find((x) => x.id === n.parentNodeId)) {
      const arr = childrenById.get(n.parentNodeId) || [];
      arr.push(n);
      childrenById.set(n.parentNodeId, arr);
    } else {
      rootNodes.push(n);
    }
  }

  const result: FlatTreeRow[] = [];
  function visit(node: HistoryGraphNodeDTO, depth: number) {
    result.push({ node, depth });
    const kids = (childrenById.get(node.id) || []).sort(
      (a, b) => a.id - b.id,
    );
    for (const k of kids) visit(k, depth + 1);
  }
  for (const r of rootNodes.sort((a, b) => a.id - b.id)) visit(r, 0);
  return result;
});

const checkpoints = computed(() =>
  (graph.value?.nodes || []).filter((n) => n.checkpointHint),
);

const pinnedNodes = computed(() =>
  (graph.value?.nodes || []).filter((n) => n.isPinned),
);

const branchCount = computed(() =>
  (graph.value?.nodes || []).filter((n) => n.childCount > 1).length,
);

// ── Node table columns ──
const nodeColumns = computed<DataTableColumns<HistoryGraphNodeDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  {
    title: "",
    key: "marker",
    width: 100,
    render(row) {
      const tags: any[] = [];
      if (row.isCurrentHead)
        tags.push(
          h(
            NTag,
            { size: "tiny", type: "success", round: true, bordered: false },
            { default: () => "HEAD" },
          ),
        );
      if (row.isPinned)
        tags.push(
          h(
            NTag,
            { size: "tiny", type: "warning", round: true, bordered: false },
            { default: () => "pin" },
          ),
        );
      if (row.checkpointHint)
        tags.push(
          h(
            NTag,
            { size: "tiny", type: "info", round: true, bordered: false },
            { default: () => "ckpt" },
          ),
        );
      return h(
        "div",
        { style: "display:flex;gap:4px;flex-wrap:wrap" },
        tags,
      );
    },
  },
  { title: "Parent", key: "parentNodeId", width: 80 },
  { title: "Command", key: "commandKind", width: 200 },
  {
    title: "Summary",
    key: "commandSummary",
    ellipsis: { tooltip: true },
  },
  {
    title: "Children",
    key: "childCount",
    width: 100,
    render(row) {
      if (row.childCount > 1) {
        return h(
          NTag,
          {
            size: "tiny",
            type: "warning",
            round: true,
            bordered: false,
          },
          { default: () => `branch (${row.childCount})` },
        );
      }
      return h("span", null, String(row.childCount));
    },
  },
  {
    title: "Created By",
    key: "createdBy",
    width: 120,
    ellipsis: { tooltip: true },
  },
  {
    title: "When",
    key: "createdAt",
    width: 160,
    render: (r) =>
      r.createdAt ? new Date(r.createdAt).toLocaleString() : "—",
  },
]);
</script>

<template>
  <div class="wave-history-page">
    <!-- Header -->
    <div class="page-header">
      <NButton text @click="backToWorkspace">
        <template #icon>
          <NIcon><ArrowBackOutline /></NIcon>
        </template>
        {{ t("waveHistory.backToWorkspace") }}
      </NButton>

      <div class="page-header-main">
        <div class="app-kicker">{{ t("waveSidebar.historyTree") }}</div>
        <h1 class="page-title">{{ t("waveHistory.title") }}</h1>
        <p class="page-subtitle">{{ t("waveHistory.subtitle") }}</p>
      </div>

      <NSpace>
        <NButton secondary @click="load">Refresh</NButton>
        <NPopconfirm @positive-click="handleGC">
          <template #trigger>
            <NButton secondary type="warning">
              <template #icon>
                <NIcon><TrashOutline /></NIcon>
              </template>
              {{ t("waveHistory.cleanUp") }}
            </NButton>
          </template>
          Clean unused, non-pinned old history nodes?
        </NPopconfirm>
      </NSpace>
    </div>

    <NAlert
      v-if="error"
      type="error"
      class="mb-4"
      :title="error"
      closable
      @close="error = ''"
    />

    <NSpin :show="loading">
      <!-- Quick metrics -->
      <div class="metric-bar mb-4">
        <div class="metric-pill">
          <span class="metric-label">Nodes</span>
          <span class="metric-num">{{ graph?.nodes?.length ?? 0 }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">Current HEAD</span>
          <span class="metric-num">#{{ graph?.currentHeadId ?? "—" }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">Branches</span>
          <span class="metric-num">{{ branchCount }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">Checkpoints</span>
          <span class="metric-num">{{ checkpoints.length }}</span>
        </div>
        <div class="metric-pill">
          <span class="metric-label">Pinned</span>
          <span class="metric-num">{{ pinnedNodes.length }}</span>
        </div>
      </div>

      <GlassCard>
        <NTabs v-model:value="activeTab" type="line" animated>
          <!-- Tab: Node List -->
          <NTabPane name="nodes" :tab="t('waveHistory.nodeList')">
            <NEmpty
              v-if="!graph || graph.nodes.length === 0"
              :description="t('waveHistory.noNodes')"
              class="empty-block"
            />
            <NDataTable
              v-else
              :columns="nodeColumns"
              :data="graph.nodes"
              :pagination="{ pageSize: 50 }"
              size="small"
            />
          </NTabPane>

          <!-- Tab: Tree View (linearized outline) -->
          <NTabPane name="tree" :tab="t('waveHistory.treeView')">
            <NEmpty
              v-if="flatTree.length === 0"
              :description="t('waveHistory.noNodes')"
              class="empty-block"
            />
            <ul v-else class="tree-list">
              <li
                v-for="row in flatTree"
                :key="row.node.id"
                class="tree-row"
                :class="{ 'is-head': row.node.isCurrentHead }"
                :style="{ paddingLeft: 12 + row.depth * 20 + 'px' }"
              >
                <span class="tree-id">#{{ row.node.id }}</span>
                <NTag
                  v-if="row.node.isCurrentHead"
                  size="tiny"
                  type="success"
                  round
                  :bordered="false"
                >HEAD</NTag>
                <NTag
                  v-if="row.node.childCount > 1"
                  size="tiny"
                  type="warning"
                  round
                  :bordered="false"
                >branch x{{ row.node.childCount }}</NTag>
                <NTag
                  v-if="row.node.isPinned"
                  size="tiny"
                  type="warning"
                  round
                  :bordered="false"
                >pin</NTag>
                <NTag
                  v-if="row.node.checkpointHint"
                  size="tiny"
                  type="info"
                  round
                  :bordered="false"
                >ckpt</NTag>
                <span class="tree-cmd">{{ row.node.commandKind }}</span>
                <span class="tree-summary">{{ row.node.commandSummary }}</span>
              </li>
            </ul>
          </NTabPane>

          <!-- Tab: Checkpoints -->
          <NTabPane name="checkpoints" :tab="t('waveHistory.checkpoints')">
            <NEmpty
              v-if="checkpoints.length === 0"
              description="No checkpoints recorded yet."
              class="empty-block"
            />
            <ul v-else class="checkpoint-list">
              <li v-for="cp in checkpoints" :key="cp.id" class="checkpoint-item">
                <NTag size="small" type="info" round :bordered="false">
                  #{{ cp.id }}
                </NTag>
                <span class="ck-summary">{{ cp.commandSummary }}</span>
                <span class="ck-time">
                  {{ cp.createdAt ? new Date(cp.createdAt).toLocaleString() : "—" }}
                </span>
              </li>
            </ul>
          </NTabPane>

          <!-- Tab: Pinned -->
          <NTabPane name="pinned" :tab="t('waveHistory.pinnedNodes')">
            <NEmpty
              v-if="pinnedNodes.length === 0"
              description="No pinned nodes. Pins are created when external objects (SupplierOrder, Shipment, ChannelSync) reference a history node as their basis."
              class="empty-block"
            />
            <ul v-else class="checkpoint-list">
              <li v-for="p in pinnedNodes" :key="p.id" class="checkpoint-item">
                <NIcon class="pin-icon"><PinOutline /></NIcon>
                <NTag size="small" type="warning" round :bordered="false">
                  #{{ p.id }}
                </NTag>
                <span class="ck-summary">{{ p.commandSummary }}</span>
                <span class="ck-time">
                  {{ p.createdAt ? new Date(p.createdAt).toLocaleString() : "—" }}
                </span>
              </li>
            </ul>
          </NTabPane>
        </NTabs>
      </GlassCard>
    </NSpin>

    <!-- Note about advanced operations -->
    <NAlert type="info" class="mt-4" :show-icon="false">
      <strong>Branch switching</strong>: Use <kbd>Ctrl+Z</kbd> /
      <kbd>Ctrl+Shift+Z</kbd> in the Wave Workspace to navigate the active branch.
      Explicit branch switching from this page is not yet available — it will
      appear here once the backend exposes the API.
    </NAlert>
  </div>
</template>

<style scoped>
.wave-history-page {
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}

.page-header-main {
  flex: 1;
  min-width: 0;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--text);
  margin: 4px 0 2px;
  letter-spacing: -0.02em;
}

.page-subtitle {
  color: var(--muted);
  font-size: 0.85rem;
  margin: 0;
}

.metric-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.metric-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 10px;
  background: var(--surface-strong);
  border: 1px solid rgba(148, 163, 184, 0.14);
}

.metric-label {
  font-size: 0.7rem;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.metric-num {
  font-weight: 800;
  font-size: 1rem;
  color: var(--text);
}

.empty-block {
  padding: 32px 0;
}

.tree-list {
  list-style: none;
  padding: 0;
  margin: 0;
  font-family: monospace;
  font-size: 0.8rem;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tree-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
}

.tree-row.is-head {
  background: rgba(34, 197, 94, 0.08);
}

.tree-id {
  font-weight: 700;
  color: var(--accent);
}

.tree-cmd {
  color: var(--muted);
}

.tree-summary {
  color: var(--text);
}

.checkpoint-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.checkpoint-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.04);
  font-size: 0.85rem;
}

.ck-summary {
  flex: 1;
  color: var(--text);
}

.ck-time {
  color: var(--muted);
  font-size: 0.75rem;
}

.pin-icon {
  color: rgba(234, 179, 8, 1);
}

kbd {
  background: rgba(148, 163, 184, 0.16);
  padding: 1px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 0.75rem;
}

.mb-4 {
  margin-bottom: 16px;
}

.mt-4 {
  margin-top: 16px;
}
</style>
