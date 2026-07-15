<script setup lang="ts">
import { computed, inject, ref } from "vue";
import { useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NEmpty,
  NGrid,
  NGridItem,
  NTag,
  NIcon,
  NProgress,
} from "naive-ui";
import {
  ArrowForwardOutline,
  CheckmarkCircleOutline,
  WarningOutline,
} from "@vicons/ionicons5";
import { dto } from "@/../wailsjs/go/models";
import { useI18n } from "@/shared/i18n";
import { waveWorkspaceSnapshotKey } from "@/shared/model/wave-injection-keys";
import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";
import BasisDriftPanel from "@/shared/ui/BasisDriftPanel.vue";
import RoutingFlowBanner from "@/shared/ui/RoutingFlowBanner.vue";
const snapshot = inject(waveWorkspaceSnapshotKey, ref(null))

const router = useRouter();
const { t } = useI18n();

const overview = computed(() => snapshot.value?.overview);
const guidance = computed(() => snapshot.value?.guidance || []);
const blockingIssues = computed(() => overview.value?.blockingIssues ?? []);
const suggestedNextStep = computed(
  () => overview.value?.suggestedNextStep ?? "",
);
const nextStepReason = computed(() => overview.value?.nextStepReason ?? "");
const stepStates = computed(() => snapshot.value?.stepStates || []);

const demandKinds = computed(() => overview.value?.demandKinds || []);
const isMembership = computed(() =>
  demandKinds.value.includes("membership_entitlement"),
);
const isRetail = computed(() => demandKinds.value.includes("retail_order"));
const defaultAllocationKind = computed<"membership" | "demand">(() =>
  isRetail.value && !isMembership.value ? "demand" : "membership",
);

const stepKeyDisplayMap: Record<string, string> = {
  demand_intake: t("waveSidebar.intakeView"),
  membership_allocation: t("wave.allocation"),
  demand_mapping: t("wave.mapping"),
  wave_overview: t("wave.overview"),
  adjustment_review: t("wave.adjustment"),
  execution_readiness: t("readiness.title"),
  supplier_execution: t("wave.execution"),
  shipment_intake: t("wave.shipment"),
  channel_sync: t("wave.sync"),
};

function nextStepReasonText(reason: string): string {
  const map: Record<string, string> = {
    no_demands_assigned: t("wave.overviewDetail.noDemandsAssigned"),
    no_fulfillment_lines: t("wave.summary.fulfillment"),
    not_exported: t("wave.overviewDetail.notExported"),
    no_shipments: t("wave.summary.shipments"),
    pending_sync: t("wave.summary.pendingSync"),
    all_steps_progressed: "",
  };
  return map[reason] ?? reason;
}

function blockingIssueText(issue: string): string {
  const map: Record<string, string> = {
    address_missing: t("wave.overviewDetail.addressMissing"),
    basis_drifted: t("wave.overviewDetail.basisDrifted"),
    review_required: t("wave.overviewDetail.reviewRequired"),
    mapping_blocked: t("wave.guidance.mapping_blocked"),
  };
  return map[issue] ?? issue;
}

function stepRoutePath(stepKey: string): string {
  const waveId = snapshot.value?.wave?.id;
  if (!waveId) return "/waves";
  const map: Record<string, string> = {
    demand_intake: `/waves/${waveId}/intake`,
    membership_allocation: `/waves/${waveId}/allocation/membership`,
    demand_mapping: `/waves/${waveId}/allocation/demand`,
    wave_overview: `/waves/${waveId}`,
    adjustment_review: `/waves/${waveId}/adjustment-review`,
    execution_readiness: `/waves/${waveId}/readiness`,
    supplier_execution: `/waves/${waveId}/export`,
    shipment_intake: `/waves/${waveId}/shipment`,
    channel_sync: `/waves/${waveId}/channel-sync`,
  };
  return map[stepKey] || `/waves/${waveId}`;
}

function goTo(stepKey: string) {
  router.push(stepRoutePath(stepKey));
}

const allocationTotal = computed(() => overview.value?.fulfillmentCount || 0);
const allocationReadyPct = computed(() => {
  if (allocationTotal.value === 0) return 0;
  return Math.round(
    ((overview.value?.fulfillmentReadyCount || 0) / allocationTotal.value) *
      100,
  );
});

// 三桶分类（路由处置 + 输入采集）
const acceptedReady = computed(
  () => overview.value?.acceptedReadyOrNotRequired ?? 0,
);
const waitingInput = computed(
  () => overview.value?.acceptedWaitingForInput ?? 0,
);
const deferredCount = computed(() => overview.value?.deferredCount ?? 0);
const excludedManualCount = computed(
  () => overview.value?.excludedManualCount ?? 0,
);

// 8 阶段进度展示（按文档定义的顺序）
const stageOrder = [
  { key: "demand_intake", label: t("waveSidebar.sectionDemandIntake") },
  {
    key: "membership_allocation",
    label: t("waveSidebar.sectionInitialAllocation"),
  },
  {
    key: "adjustment_review",
    label: t("waveSidebar.sectionAdjustment"),
  },
  {
    key: "execution_readiness",
    label: t("waveSidebar.sectionReadiness"),
  },
  {
    key: "supplier_execution",
    label: t("waveSidebar.sectionSupplierExecution"),
  },
  {
    key: "shipment_intake",
    label: t("waveSidebar.sectionShipmentIntake"),
  },
  { key: "channel_sync", label: t("waveSidebar.sectionChannelSync") },
];

const stageProgress = computed(() => {
  const stateMap = new Map<string, dto.WaveStepStateDTO>();
  for (const s of stepStates.value) stateMap.set(s.stepKey, s);
  return stageOrder.map((stage) => {
    const s = stateMap.get(stage.key);
    return {
      ...stage,
      status: (s?.status as string) || "idle",
      count: s?.primaryCount ?? 0,
    };
  });
});

function stageBadgeType(
  status: string,
): "default" | "info" | "success" | "warning" {
  switch (status) {
    case "current":
      return "info";
    case "available":
    case "active":
      return "success";
    default:
      return "default";
  }
}
</script>

<template>
  <div class="wave-overview-readonly">
    <NEmpty
      v-if="!snapshot"
      :description="t('common.loading')"
      class="mt-20"
    />

    <template v-else>
      <PageHeader
        :title="t('waveSidebar.overview')"
        :description="t('wave.previewDescription')"
      />

      <!-- ── 三桶分类（路由处置） ── -->
      <h3 class="section-heading">{{ t("wave.routingGroup") }}</h3>
      <NGrid :cols="4" :x-gap="16" :y-gap="16" class="mb-6" responsive="screen" item-responsive>
        <NGridItem span="4 m:1">
          <GlassCard class="metric-card metric-success">
            <div class="metric-label">{{ t("wave.summary.ready") }}</div>
            <div class="metric-value">{{ acceptedReady }}</div>
          </GlassCard>
        </NGridItem>
        <NGridItem span="4 m:1">
          <GlassCard class="metric-card metric-warning">
            <div class="metric-label">{{ t("wave.summary.waitingInput") }}</div>
            <div class="metric-value">{{ waitingInput }}</div>
          </GlassCard>
        </NGridItem>
        <NGridItem span="4 m:1">
          <GlassCard class="metric-card metric-muted">
            <div class="metric-label">{{ t("wave.summary.deferred") }}</div>
            <div class="metric-value">{{ deferredCount }}</div>
          </GlassCard>
        </NGridItem>
        <NGridItem span="4 m:1">
          <GlassCard class="metric-card metric-muted">
            <div class="metric-label">
              {{ t("wave.summary.excluded") }}
              <span class="metric-sublabel">(manual)</span>
            </div>
            <div class="metric-value">{{ excludedManualCount }}</div>
          </GlassCard>
        </NGridItem>
      </NGrid>

      <!-- ── Basis Drift Panel ── -->
      <BasisDriftPanel :snapshot="snapshot" class="mb-6" />

      <!-- ── 8 阶段进度 ── -->
      <h3 class="section-heading">{{ t("wave.executionGroup") }}</h3>
      <div class="stage-progress mb-6">
        <div
          v-for="(stage, idx) in stageProgress"
          :key="stage.key"
          class="stage-pill"
          :class="`stage-${stage.status}`"
        >
          <span class="stage-num">{{ idx + 1 }}</span>
          <span class="stage-label">{{ stage.label }}</span>
          <NTag
            v-if="stage.count > 0"
            size="tiny"
            round
            :type="stageBadgeType(stage.status)"
            :bordered="false"
          >
            {{ stage.count }}
          </NTag>
        </div>
      </div>

      <!-- ── Suggested Next Step ── -->
      <GlassCard
        v-if="suggestedNextStep && suggestedNextStep !== 'wave_overview'"
        class="next-step-card mb-6"
      >
        <div class="next-step-grid">
          <div class="next-step-text">
            <div class="app-kicker">{{ t("wave.overviewDetail.suggestedNext") }}</div>
            <h2 class="next-step-title">
              {{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}
            </h2>
            <p class="next-step-reason">
              {{ nextStepReasonText(nextStepReason) || t("wave.previewDescription") }}
            </p>
          </div>
          <NButton
            type="primary"
            size="medium"
            round
            @click="goTo(suggestedNextStep)"
          >
            <template #icon>
              <NIcon><ArrowForwardOutline /></NIcon>
            </template>
            {{ t("waveSidebar.overview") }} → {{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}
          </NButton>
        </div>
      </GlassCard>

      <!-- ── 三问分流 Banner ── -->
      <RoutingFlowBanner
        :wave-id="snapshot.wave?.id ?? null"
        :default-allocation-kind="defaultAllocationKind"
        class="mb-6"
      />

      <!-- ── Blocking Issues ── -->
      <NAlert
        v-if="blockingIssues.length"
        type="error"
        class="mb-6"
        :title="t('wave.overviewDetail.blockingIssues')"
      >
        <ul class="blocking-list">
          <li v-for="issue in blockingIssues" :key="issue">
            <NIcon class="mr-1"><WarningOutline /></NIcon>
            <strong>{{ blockingIssueText(issue) }}</strong>
          </li>
        </ul>
      </NAlert>

      <!-- ── Guidance（弱提示） ── -->
      <NAlert
        v-if="guidance.length"
        type="warning"
        class="mb-6"
        :title="t('wave.nextAction')"
      >
        <ul class="guidance-list">
          <li v-for="item in guidance" :key="item.code">
            <span>
              {{ t(`wave.guidance.${item.code}`) || item.code }}
              <NTag size="tiny" :bordered="false">{{ item.count }}</NTag>
            </span>
            <NButton
              size="tiny"
              type="warning"
              secondary
              @click="goTo(item.targetStepKey)"
            >
              {{ stepKeyDisplayMap[item.targetStepKey] ?? item.targetStepKey }}
            </NButton>
          </li>
        </ul>
      </NAlert>

      <!-- ── 履约 / 执行 总览（只读） ── -->
      <h3 class="section-heading">{{ t("wave.summary.fulfillment") }}</h3>
      <NGrid :cols="3" :x-gap="16" :y-gap="16" class="mb-6" responsive="screen" item-responsive>
        <NGridItem span="3 m:1">
          <GlassCard hoverable>
            <div class="app-kicker mb-3">{{ t("wave.summary.ready") }}</div>
            <div class="metric-value-line">
              <span class="metric-value-num">{{ allocationReadyPct }}%</span>
              <span class="metric-value-suffix">ready</span>
            </div>
            <NProgress
              type="line"
              status="success"
              :percentage="allocationReadyPct"
              :show-indicator="false"
              class="mt-3"
            />
            <div class="metric-row mt-3">
              <span class="text-muted">draft: {{ overview?.fulfillmentDraftCount ?? 0 }}</span>
              <span class="text-success">
                ready: {{ overview?.fulfillmentReadyCount ?? 0 }} / {{ allocationTotal }}
              </span>
            </div>
          </GlassCard>
        </NGridItem>
        <NGridItem span="3 m:1">
          <GlassCard hoverable>
            <div class="app-kicker mb-3">{{ t("wave.summary.demand") }}</div>
            <div class="kv-list">
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.summary.demand") }}</span>
                <span class="kv-num">{{ overview?.demandCount ?? 0 }}</span>
              </div>
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.summary.fulfillment") }}</span>
                <span class="kv-num">{{ overview?.fulfillmentCount ?? 0 }}</span>
              </div>
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.adjustment") }}</span>
                <NTag size="small" :bordered="false">{{ overview?.adjustmentCount ?? 0 }}</NTag>
              </div>
            </div>
          </GlassCard>
        </NGridItem>
        <NGridItem span="3 m:1">
          <GlassCard hoverable>
            <div class="app-kicker mb-3">{{ t("wave.execution") }}</div>
            <div class="kv-list">
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.summary.supplierOrders") }}</span>
                <span class="kv-num">{{ overview?.supplierOrderCount ?? 0 }}</span>
              </div>
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.summary.shipments") }}</span>
                <span class="kv-num text-success">
                  <NIcon class="mr-1"><CheckmarkCircleOutline /></NIcon>
                  {{ overview?.shipmentCount ?? 0 }}
                </span>
              </div>
              <div class="kv-row">
                <span class="text-muted">{{ t("wave.summary.channelSyncJobs") }}</span>
                <span class="kv-num">{{ overview?.channelSyncJobCount ?? 0 }}</span>
              </div>
            </div>
          </GlassCard>
        </NGridItem>
      </NGrid>
    </template>
  </div>
</template>

<style scoped>
.wave-overview-readonly {
  display: flex;
  flex-direction: column;
  padding-bottom: 32px;
}

.section-heading {
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
  margin: 16px 0 12px;
}

.metric-card {
  padding: 14px 16px;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--surface-strong);
  border: 1px solid rgba(148, 163, 184, 0.12);
}

.metric-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.metric-sublabel {
  font-weight: 400;
  text-transform: none;
  font-size: 0.7rem;
  margin-left: 4px;
}

.metric-value {
  font-size: 1.8rem;
  font-weight: 800;
  color: var(--text);
  line-height: 1;
}

.metric-success {
  border-color: rgba(34, 197, 94, 0.25);
}
.metric-warning {
  border-color: rgba(234, 179, 8, 0.25);
}
.metric-muted {
  border-color: rgba(148, 163, 184, 0.18);
}

.stage-progress {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.stage-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(148, 163, 184, 0.04);
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted);
}

.stage-pill.stage-current {
  border-color: rgba(99, 102, 241, 0.5);
  background: rgba(99, 102, 241, 0.1);
  color: var(--accent);
}

.stage-pill.stage-available,
.stage-pill.stage-active {
  border-color: rgba(34, 197, 94, 0.4);
  background: rgba(34, 197, 94, 0.08);
  color: rgba(34, 197, 94, 1);
}

.stage-num {
  font-family: monospace;
  font-size: 0.75rem;
  opacity: 0.7;
}

.stage-label {
  white-space: nowrap;
}

.next-step-card {
  padding: 18px 20px;
  border-left: 3px solid var(--accent);
}

.next-step-grid {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.next-step-text {
  flex: 1;
  min-width: 240px;
}

.next-step-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--text);
  margin: 4px 0;
}

.next-step-reason {
  color: var(--muted);
  margin: 0;
}

.blocking-list,
.guidance-list {
  list-style: none;
  padding: 0;
  margin: 8px 0 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.blocking-list li,
.guidance-list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.metric-value-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.metric-value-num {
  font-size: 2rem;
  font-weight: 800;
  color: var(--text);
}

.metric-value-suffix {
  font-size: 0.85rem;
  color: var(--muted);
}

.metric-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.8rem;
}

.kv-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kv-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.kv-num {
  font-weight: 700;
  color: var(--text);
}

.text-muted {
  color: var(--muted);
}
.text-success {
  color: rgba(34, 197, 94, 1);
}

.mt-3 {
  margin-top: 12px;
}
.mr-1 {
  margin-right: 4px;
}
.mb-3 {
  margin-bottom: 12px;
}
.mb-6 {
  margin-bottom: 24px;
}
.mt-20 {
  margin-top: 80px;
}
</style>
