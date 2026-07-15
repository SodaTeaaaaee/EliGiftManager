<script setup lang="ts">
import { computed, h, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NGrid,
  NGridItem,
  NIcon,
  NSpace,
  NTag,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  CheckmarkDoneOutline,
  DocumentTextOutline,
  LocationOutline,
  MapOutline,
  SyncOutline,
} from "@vicons/ionicons5";
import {
  listDemandInboxRows,
  listWaveDashboardRows,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const router = useRouter();
const { t, locale } = useI18n();

const waveRows = ref<dto.WaveDashboardRowDTO[]>([]);
const inboxRows = ref<dto.DemandInboxRowDTO[]>([]);
const loading = ref(false);
const error = ref("");

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const [waves, inbox] = await Promise.all([
      listWaveDashboardRows(),
      listDemandInboxRows({ assignment: "unassigned", demandKind: "" }),
    ]);
    waveRows.value = waves || [];
    inboxRows.value = inbox || [];
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

onMounted(load);

// ── Bucketed counts ──
const pendingDemandCount = computed(() => inboxRows.value.length);

const driftReviewWaveCount = computed(() =>
  // The dashboard rows DTO doesn't currently expose drift signals; we use lifecycle stage as a heuristic.
  waveRows.value.filter((w) => w.projectedLifecycleStage === "syncing_back")
    .length,
);

const awaitingClosureCount = computed(
  () =>
    waveRows.value.filter(
      (w) => w.projectedLifecycleStage === "awaiting_manual_closure",
    ).length,
);

const addressMissingWaveCount = computed(() =>
  waveRows.value.filter((w) => w.projectedLifecycleStage === "address_blocked")
    .length,
);

const mappingBlockedWaveCount = computed(() =>
  waveRows.value.filter((w) => w.projectedLifecycleStage === "allocating")
    .length,
);

const activeWavesAll = computed(() =>
  waveRows.value.filter((w) => w.projectedLifecycleStage !== "closed"),
);

// ── Action items ──
interface ActionItem {
  key: string;
  label: string;
  desc: string;
  count: number;
  type: "info" | "success" | "warning" | "error" | "default";
  icon: any;
  onClick: () => void;
}

const actionItems = computed<ActionItem[]>(() => [
  {
    key: "pending-demands",
    label: t("actionCenter.pendingDemands"),
    desc: t("actionCenter.pendingDemandsDesc"),
    count: pendingDemandCount.value,
    type: pendingDemandCount.value > 0 ? "info" : "default",
    icon: DocumentTextOutline,
    onClick: () => router.push("/demand-inbox?assignment=unassigned"),
  },
  {
    key: "drift-review",
    label: t("actionCenter.driftRequiredReviews"),
    desc: t("actionCenter.driftRequiredReviewsDesc"),
    count: driftReviewWaveCount.value,
    type: driftReviewWaveCount.value > 0 ? "warning" : "default",
    icon: SyncOutline,
    onClick: () => router.push("/waves"),
  },
  {
    key: "awaiting-closure",
    label: t("actionCenter.awaitingClosure"),
    desc: t("actionCenter.awaitingClosureDesc"),
    count: awaitingClosureCount.value,
    type: awaitingClosureCount.value > 0 ? "error" : "default",
    icon: CheckmarkDoneOutline,
    onClick: () => router.push("/waves"),
  },
  {
    key: "address-missing",
    label: t("actionCenter.addressMissing"),
    desc: t("actionCenter.addressMissingDesc"),
    count: addressMissingWaveCount.value,
    type: addressMissingWaveCount.value > 0 ? "warning" : "default",
    icon: LocationOutline,
    onClick: () => router.push("/waves"),
  },
  {
    key: "mapping-blocked",
    label: t("actionCenter.mappingBlocked"),
    desc: t("actionCenter.mappingBlockedDesc"),
    count: mappingBlockedWaveCount.value,
    type: mappingBlockedWaveCount.value > 0 ? "warning" : "default",
    icon: MapOutline,
    onClick: () => router.push("/waves"),
  },
]);

const totalActionsRequired = computed(() =>
  actionItems.value.reduce((sum, item) => sum + item.count, 0),
);

// ── Active waves table ──
const stageTagType: Record<string, "default" | "info" | "success" | "warning" | "error"> = {
  intake: "info",
  draft: "info",
  allocating: "info",
  review: "warning",
  ready_to_submit: "success",
  execution: "warning",
  syncing_back: "info",
  awaiting_manual_closure: "error",
  closed: "default",
  address_blocked: "warning",
  partially_shipped: "info",
  shipped: "success",
};

const waveColumns = computed<DataTableColumns<dto.WaveDashboardRowDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Wave", key: "waveNo", width: 160 },
  { title: "Name", key: "name", ellipsis: { tooltip: true } },
  {
    title: t("dashboard.stage"),
    key: "projectedLifecycleStage",
    width: 200,
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
    width: 160,
    render(row) {
      return row.createdAt
        ? new Date(row.createdAt).toLocaleDateString(locale.value)
        : "—";
    },
  },
]);
</script>

<template>
  <div class="dashboard-page pb-12">
    <PageHeader
      :title="t('actionCenter.title')"
      :description="t('actionCenter.subtitle')"
      :kicker="t('nav.dashboard')"
    />

    <NAlert type="info" size="small" class="mb-4" :show-icon="false">
      Read-only overview. Use the
      <a class="link-text" href="javascript:void(0)" @click="router.push('/waves')">Waves</a>
      page to create or open a wave; use the
      <a class="link-text" href="javascript:void(0)" @click="router.push('/demand-inbox')">Demand Inbox</a>
      to manage routing.
    </NAlert>

    <NAlert
      v-if="error"
      type="error"
      :title="error"
      class="mb-6"
      closable
      @close="error = ''"
    />

    <!-- Action Required panel -->
    <section class="mb-6">
      <div class="section-heading">
        <span>{{ t("actionCenter.actionsRequired") }}</span>
        <NTag
          :type="totalActionsRequired > 0 ? 'error' : 'success'"
          size="small"
          round
          :bordered="false"
        >
          {{ totalActionsRequired }}
        </NTag>
      </div>

      <NGrid :cols="5" :x-gap="14" :y-gap="14" responsive="screen" item-responsive>
        <NGridItem
          v-for="item in actionItems"
          :key="item.key"
          span="5 m:1"
        >
          <button
            type="button"
            class="action-tile"
            :class="`tile-${item.type}`"
            :disabled="item.count === 0"
            @click="item.onClick"
          >
            <NIcon size="20" class="tile-icon">
              <component :is="item.icon" />
            </NIcon>
            <div class="tile-num">{{ item.count }}</div>
            <div class="tile-label">{{ item.label }}</div>
            <div class="tile-desc">{{ item.desc }}</div>
          </button>
        </NGridItem>
      </NGrid>
    </section>

    <!-- Active Waves -->
    <section class="mb-6">
      <div class="section-heading">
        <span>{{ t("actionCenter.activeWaves") }}</span>
        <NTag size="small" :bordered="false" round>
          {{ activeWavesAll.length }}
        </NTag>
      </div>

      <GlassCard>
        <NEmpty
          v-if="!loading && activeWavesAll.length === 0"
          :description="t('actionCenter.nothingToDo')"
          class="my-8"
        />
        <NDataTable
          v-else
          :columns="waveColumns"
          :data="activeWavesAll.slice(0, 8)"
          :loading="loading"
          :pagination="false"
          size="medium"
        />
      </GlassCard>
    </section>
  </div>
</template>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
}

.section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
  margin-bottom: 12px;
}

.action-tile {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 16px 18px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: var(--surface-strong);
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  color: inherit;
  transition: all 0.2s ease;
}

.action-tile:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.action-tile:not(:disabled):hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
}

.tile-icon {
  color: var(--muted);
}

.tile-info {
  border-color: rgba(59, 130, 246, 0.3);
  background: rgba(59, 130, 246, 0.05);
}
.tile-info .tile-icon,
.tile-info .tile-num {
  color: rgba(59, 130, 246, 1);
}

.tile-warning {
  border-color: rgba(234, 179, 8, 0.3);
  background: rgba(234, 179, 8, 0.05);
}
.tile-warning .tile-icon,
.tile-warning .tile-num {
  color: rgba(234, 179, 8, 1);
}

.tile-error {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.05);
}
.tile-error .tile-icon,
.tile-error .tile-num {
  color: rgba(239, 68, 68, 1);
}

.tile-success {
  border-color: rgba(34, 197, 94, 0.3);
  background: rgba(34, 197, 94, 0.05);
}
.tile-success .tile-icon,
.tile-success .tile-num {
  color: rgba(34, 197, 94, 1);
}

.tile-default {
  /* uses defaults */
}

.tile-num {
  font-size: 1.8rem;
  font-weight: 800;
  color: var(--text);
  line-height: 1;
}

.tile-label {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text);
}

.tile-desc {
  font-size: 0.72rem;
  color: var(--muted);
  line-height: 1.4;
}

.see-more {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}

.my-8 {
  margin: 32px 0;
}

.mb-6 {
  margin-bottom: 24px;
}

.pb-12 {
  padding-bottom: 48px;
}
</style>
