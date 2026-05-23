<script setup lang="ts">
import { computed, inject } from "vue";
import { useRouter } from "vue-router";
import { NAlert, NButton, NCard, NEmpty, NGrid, NGridItem, NList, NListItem, NStatistic, NSpace, NTag, NTimeline, NTimelineItem, NProgress } from "naive-ui";
import { dto } from "@/../wailsjs/go/models";
import { useI18n } from "@/shared/i18n";

const snapshot = inject("waveWorkspaceSnapshot", computed(() => null)) as { value: dto.WaveWorkspaceSnapshotDTO | null };
const router = useRouter();
const { t } = useI18n();

const overview = computed(() => snapshot.value?.overview);
const guidance = computed(() => snapshot.value?.guidance || []);
const recentHistory = computed(() => snapshot.value?.recentHistory || []);

const blockingIssues = computed(() => overview.value?.blockingIssues ?? []);
const suggestedNextStep = computed(() => overview.value?.suggestedNextStep ?? "");
const nextStepReason = computed(() => overview.value?.nextStepReason ?? "");

function lifecycleText(key: string) {
  const map: Record<string, string> = {
    intake: t("wave.lifecycle.intake"),
    allocation: t("wave.lifecycle.allocation"),
    review: t("wave.lifecycle.review"),
    execution: t("wave.lifecycle.execution"),
    syncing_back: t("wave.lifecycle.syncing_back"),
    awaiting_manual_closure: t("wave.lifecycle.awaiting_manual_closure"),
    closed: t("wave.lifecycle.closed"),
  };
  return map[key] || key;
}

// Map technical keys to localization keys.
const stepKeyDisplayMap: Record<string, string> = {
  demand_intake: t("wave.overview"),
  membership_allocation: t("wave.allocation"),
  demand_mapping: t("wave.mapping"),
  wave_overview: t("wave.overview"),
  adjustment_review: t("wave.adjustment"),
  supplier_execution: t("wave.execution"),
  shipment_intake: t("wave.shipment"),
  channel_sync: t("wave.sync"),
};

function summaryText(key: string) {
  const map: Record<string, string> = {
    ready: t("wave.summary.ready"),
    waiting_for_input: t("wave.summary.waitingInput"),
    deferred: t("wave.summary.deferred"),
    excluded: t("wave.summary.excluded"),
    supplier_orders: t("wave.summary.supplierOrders"),
    shipments: t("wave.summary.shipments"),
    pending_sync: t("wave.summary.pendingSync"),
    failed_sync: t("wave.summary.failedSync"),
    manual_closure_candidates: t("wave.summary.manualClosureCandidates"),
    demand: t("wave.summary.demand"),
    fulfillment: t("wave.summary.fulfillment"),
    channel_sync_jobs: t("wave.summary.channelSyncJobs"),
  };
  return map[key] || key;
}

function historyCommandText(key: string) {
  const map: Record<string, string> = {
    system_baseline: t("wave.commandKinds.system_baseline"),
    create_rule: t("wave.commandKinds.create_rule"),
    update_rule: t("wave.commandKinds.update_rule"),
    delete_rule: t("wave.commandKinds.delete_rule"),
    reconcile_wave: t("wave.commandKinds.reconcile_wave"),
    generate_participants: t("wave.commandKinds.generate_participants"),
    assign_demand: t("wave.commandKinds.assign_demand"),
    map_demand_lines: t("wave.commandKinds.map_demand_lines"),
    record_adjustment: t("wave.commandKinds.record_adjustment"),
    export_supplier_order: t("wave.commandKinds.export_supplier_order"),
    create_shipment: t("wave.commandKinds.create_shipment"),
    create_channel_sync_job: t("wave.commandKinds.create_channel_sync_job"),
    execute_channel_sync_job: t("wave.commandKinds.execute_channel_sync_job"),
    retry_channel_sync_job: t("wave.commandKinds.retry_channel_sync_job"),
    record_closure_decision: t("wave.commandKinds.record_closure_decision"),
    snapshot: t("wave.commandKinds.snapshot"),
  };
  return map[key] || key;
}

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

function goTo(stepKey: string) {
  const waveId = snapshot.value?.wave?.id;
  if (!waveId) return;
  const targetMap: Record<string, string> = {
    demand_intake: "/demand-intake",
    membership_allocation: `/waves/${waveId}/allocation`,
    demand_mapping: `/waves/${waveId}/demand-mapping`,
    wave_overview: `/waves/${waveId}`,
    adjustment_review: `/waves/${waveId}/adjustment-review`,
    supplier_execution: `/waves/${waveId}/export`,
    shipment_intake: `/waves/${waveId}/shipment`,
    channel_sync: `/waves/${waveId}/channel-sync`,
  };
  router.push(targetMap[stepKey] || `/waves/${waveId}`);
}

// Compute percentage for address readiness
const addressTotal = computed(() => {
  if (!overview.value) return 0;
  return (overview.value.addressReadyCount || 0) + (overview.value.addressMissingCount || 0) + (overview.value.addressInvalidCount || 0);
});

const addressReadyPct = computed(() => {
  if (addressTotal.value === 0) return 0;
  return Math.round(((overview.value?.addressReadyCount || 0) / addressTotal.value) * 100);
});

// Compute percentage for allocation state
const allocationTotal = computed(() => {
  return overview.value?.fulfillmentCount || 0;
});

const allocationReadyPct = computed(() => {
  if (allocationTotal.value === 0) return 0;
  return Math.round(((overview.value?.fulfillmentReadyCount || 0) / allocationTotal.value) * 100);
});
</script>

<template>
  <div class="wave-overview-page">
    <NEmpty v-if="!snapshot" :description="t('common.loading')" />

    <template v-else>
      <!-- Top Grid: Suggested Step + Stage -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" class="mb-5">
        <!-- Next Step Card -->
        <NGridItem :span="suggestedNextStep && suggestedNextStep !== 'wave_overview' ? 16 : 24">
          <NCard 
            class="glow-card h-full"
            style="border-left: 5px solid var(--accent); background: linear-gradient(135deg, var(--surface-strong) 0%, var(--surface-muted) 100%);"
          >
            <div class="flex items-start justify-between gap-6">
              <div>
                <div class="app-kicker">{{ t("wave.overviewDetail.suggestedNext") }}</div>
                <h2 class="app-title mt-2">
                  {{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}
                </h2>
                <p v-if="nextStepReasonText(nextStepReason)" class="app-copy mt-2">
                  {{ nextStepReasonText(nextStepReason) }}
                </p>
                <p v-else class="app-copy mt-2">{{ t("wave.previewDescription") }}</p>
              </div>
              <NSpace align="center">
                <NButton 
                  v-if="suggestedNextStep && suggestedNextStep !== 'wave_overview'" 
                  type="primary" 
                  @click="goTo(suggestedNextStep)"
                >
                  {{ t("wave.overviewDetail.goToStep", { step: stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }) }}
                </NButton>
                <NButton secondary @click="goTo('adjustment_review')">
                  {{ t("wave.adjustment") }}
                </NButton>
              </NSpace>
            </div>
          </NCard>
        </NGridItem>

        <!-- Current Lifecycle Stage Card -->
        <NGridItem :span="8" v-if="suggestedNextStep && suggestedNextStep !== 'wave_overview'">
          <NCard class="glow-card h-full flex flex-col justify-between">
            <div>
              <div class="app-kicker">Current Stage</div>
              <h3 class="app-heading-md mt-2" style="font-size: 20px;">
                {{ lifecycleText(snapshot.projectedLifecycleStage) }}
              </h3>
            </div>
            <div class="mt-4">
              <NProgress
                type="line"
                status="success"
                :percentage="allocationReadyPct"
                :show-indicator="true"
                processing
              />
              <div class="text-xs text-slate-400 mt-1">
                Fulfillment lines ready: {{ overview?.fulfillmentReadyCount ?? 0 }} / {{ overview?.fulfillmentCount ?? 0 }}
              </div>
            </div>
          </NCard>
        </NGridItem>
      </NGrid>

      <!-- Blocking Issues and Guidance Alerts -->
      <NAlert v-if="blockingIssues.length" type="error" class="mb-4" :title="t('wave.overviewDetail.blockingIssues')">
        <NSpace vertical :size="8">
          <ul style="margin: 4px 0; padding-left: 1.2em; line-height: 1.6;">
            <li v-for="issue in blockingIssues" :key="issue">
              <strong>{{ blockingIssueText(issue) }}</strong> — Requires attention before final export.
            </li>
          </ul>
          <NButton size="tiny" secondary type="error" @click="goTo('adjustment_review')" style="margin-top: 4px;">
            Go to Adjustment Review
          </NButton>
        </NSpace>
      </NAlert>

      <NAlert v-if="guidance.length" type="warning" class="mb-4" :title="t('wave.nextAction')">
        <NSpace vertical :size="10">
          <div
            v-for="item in guidance"
            :key="item.code"
            class="flex items-center justify-between gap-4 border-b border-dashed border-slate-700/20 pb-2"
          >
            <span class="text-sm font-medium text-amber-700 dark:text-amber-300">
              {{ t(`wave.guidance.${item.code}`) || item.code }} ({{ item.count }})
            </span>
            <NButton size="small" type="warning" secondary @click="goTo(item.targetStepKey)">
              Resolve in {{ stepKeyDisplayMap[item.targetStepKey] ?? item.targetStepKey }}
            </NButton>
          </div>
        </NSpace>
      </NAlert>

      <!-- Core Statistics Cards Grid -->
      <NGrid :cols="4" :x-gap="16" :y-gap="16" class="mb-5">
        <NGridItem>
          <NCard class="glow-card compact-card">
            <NStatistic :label="summaryText('demand')" :value="overview?.demandCount ?? 0" />
            <div class="text-xs text-slate-400 mt-2">Incoming raw demands</div>
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard class="glow-card compact-card">
            <NStatistic :label="summaryText('fulfillment')" :value="overview?.fulfillmentCount ?? 0" />
            <div class="text-xs text-slate-400 mt-2">Fulfillment items mapped</div>
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard class="glow-card compact-card">
            <NStatistic :label="summaryText('supplier_orders')" :value="overview?.supplierOrderCount ?? 0" />
            <div class="text-xs text-slate-400 mt-2">Factory orders created</div>
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard class="glow-card compact-card">
            <NStatistic :label="summaryText('shipments')" :value="overview?.shipmentCount ?? 0" />
            <div class="text-xs text-slate-400 mt-2">Tracking records loaded</div>
          </NCard>
        </NGridItem>
      </NGrid>

      <!-- Diagnosis breakdown card -->
      <NCard :title="t('wave.overviewDetail.fulfillmentBreakdown')" class="mb-5 glow-card">
        <NGrid :cols="3" :x-gap="20" :y-gap="20">
          <!-- Allocation State Column -->
          <NGridItem class="border-r border-slate-700/10 dark:border-slate-700/30 pr-5">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm font-bold tracking-wide uppercase text-slate-500">Allocation Status</span>
              <NTag size="small" type="primary" :bordered="false">{{ allocationReadyPct }}% Ready</NTag>
            </div>
            <NProgress
              type="line"
              status="info"
              :percentage="allocationReadyPct"
              class="mb-4"
              processing
            />
            <NList size="small" :bordered="false">
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Draft</span>
                <span class="font-bold">{{ overview?.fulfillmentDraftCount ?? 0 }}</span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Ready</span>
                <span class="font-bold text-emerald-600 dark:text-emerald-400">
                  {{ overview?.fulfillmentReadyCount ?? 0 }}
                </span>
              </NListItem>
            </NList>
          </NGridItem>

          <!-- Address Validation Column -->
          <NGridItem class="border-r border-slate-700/10 dark:border-slate-700/30 pr-5">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm font-bold tracking-wide uppercase text-slate-500">Address Validation</span>
              <NTag size="small" :type="overview?.addressMissingCount || overview?.addressInvalidCount ? 'warning' : 'success'" :bordered="false">
                {{ addressReadyPct }}% Valid
              </NTag>
            </div>
            <NProgress
              type="line"
              :status="overview?.addressMissingCount || overview?.addressInvalidCount ? 'warning' : 'success'"
              :percentage="addressReadyPct"
              class="mb-4"
            />
            <NList size="small" :bordered="false">
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Missing Address</span>
                <span :class="['font-bold', (overview?.addressMissingCount ?? 0) > 0 ? 'text-amber-500' : '']">
                  {{ overview?.addressMissingCount ?? 0 }}
                </span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Invalid / Unverified</span>
                <span :class="['font-bold', (overview?.addressInvalidCount ?? 0) > 0 ? 'text-red-500' : '']">
                  {{ overview?.addressInvalidCount ?? 0 }}
                </span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Ready & Verified</span>
                <span class="font-bold text-emerald-500">{{ overview?.addressReadyCount ?? 0 }}</span>
              </NListItem>
            </NList>
          </NGridItem>

          <!-- Factory Submission Column -->
          <NGridItem>
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm font-bold tracking-wide uppercase text-slate-500">Supplier Execution</span>
              <NTag size="small" :type="overview?.supplierShippedCount ? 'success' : 'default'" :bordered="false">
                {{ overview?.supplierShippedCount ?? 0 }} Shipped
              </NTag>
            </div>
            <NList size="small" :bordered="false">
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Unsubmitted Lines</span>
                <span class="font-bold text-amber-500">{{ overview?.supplierNotSubmittedCount ?? 0 }}</span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Submitted to Factory</span>
                <span class="font-bold text-blue-500">{{ overview?.supplierSubmittedCount ?? 0 }}</span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-400">Shipped by Factory</span>
                <span class="font-bold text-emerald-500">{{ overview?.supplierShippedCount ?? 0 }}</span>
              </NListItem>
            </NList>
          </NGridItem>
        </NGrid>
      </NCard>

      <!-- Wave Details, Adjustments, and Sync status -->
      <NGrid :cols="24" :x-gap="16" :y-gap="16" class="mb-5">
        <!-- Routing summary -->
        <NGridItem :span="8">
          <NCard :title="t('wave.routingGroup')" class="glow-card h-full">
            <NList size="small">
              <NListItem class="flex justify-between">
                <span>{{ summaryText("ready") }}</span>
                <NTag size="small" type="success" :bordered="false">{{ overview?.acceptedReadyOrNotRequired ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span>{{ summaryText("waiting_for_input") }}</span>
                <NTag size="small" type="warning" :bordered="false">{{ overview?.acceptedWaitingForInput ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span>{{ summaryText("deferred") }}</span>
                <NTag size="small" :bordered="false">{{ overview?.deferredCount ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span>{{ summaryText("excluded") }}</span>
                <NTag size="small" type="error" :bordered="false">
                  {{ (overview?.excludedManualCount ?? 0) + (overview?.excludedDuplicateCount ?? 0) + (overview?.excludedRevokedCount ?? 0) }}
                </NTag>
              </NListItem>
            </NList>
          </NCard>
        </NGridItem>

        <!-- Adjustments summary -->
        <NGridItem :span="8">
          <NCard :title="t('wave.overviewDetail.adjustmentSummary')" class="glow-card h-full">
            <NList size="small">
              <NListItem class="flex justify-between">
                <span>Total Adjustments</span>
                <span class="font-bold">{{ overview?.adjustmentCount ?? 0 }}</span>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-emerald-600 dark:text-emerald-400">{{ t('adjustment.add') }}</span>
                <NTag size="small" type="success" :bordered="false">+{{ overview?.adjustmentAddCount ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-rose-600 dark:text-rose-400">{{ t('adjustment.reduce') }}</span>
                <NTag size="small" type="error" :bordered="false">-{{ overview?.adjustmentReduceCount ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-blue-600 dark:text-blue-400">{{ t('adjustment.replace') }}</span>
                <NTag size="small" type="info" :bordered="false">{{ overview?.adjustmentReplaceCount ?? 0 }}</NTag>
              </NListItem>
              <NListItem class="flex justify-between">
                <span class="text-slate-500">{{ t('adjustment.remove') }}</span>
                <NTag size="small" :bordered="false">{{ overview?.adjustmentRemoveCount ?? 0 }}</NTag>
              </NListItem>
            </NList>
          </NCard>
        </NGridItem>

        <!-- Basis and Sync State -->
        <NGridItem :span="8">
          <NCard :title="t('wave.basis')" class="glow-card h-full">
            <NSpace vertical :size="12" class="mb-4">
              <div class="flex items-center justify-between">
                <span class="text-slate-400">{{ t("wave.drifted") }}</span>
                <NTag size="small" :type="snapshot.basisSummary.hasDriftedBasis ? 'warning' : 'default'">
                  {{ snapshot.basisSummary.driftedCount }} drifted
                </NTag>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-slate-400">{{ t("wave.reviewRequired") }}</span>
                <NTag size="small" :type="snapshot.basisSummary.hasRequiredReview ? 'error' : 'default'">
                  {{ snapshot.basisSummary.requiredReviewCount }} required
                </NTag>
              </div>
            </NSpace>
            <div class="border-t border-slate-700/10 dark:border-slate-700/30 pt-3">
              <div class="text-xs font-bold text-slate-500 uppercase tracking-wide mb-2">Channel Sync Progress</div>
              <NList size="small">
                <NListItem class="flex justify-between py-1">
                  <span class="text-slate-400 text-xs">Sync Jobs</span>
                  <span class="font-bold text-xs">{{ overview?.channelSyncJobCount ?? 0 }}</span>
                </NListItem>
                <NListItem class="flex justify-between py-1">
                  <span class="text-slate-400 text-xs">Failed Sync</span>
                  <span :class="['font-bold', 'text-xs', (overview?.channelSyncFailedCount ?? 0) > 0 ? 'text-red-500' : '']">
                    {{ overview?.channelSyncFailedCount ?? 0 }}
                  </span>
                </NListItem>
              </NList>
            </div>
          </NCard>
        </NGridItem>
      </NGrid>

      <!-- History Feed -->
      <NCard :title="t('wave.history')" class="glow-card">
        <template #header-extra>
          <NSpace align="center" :size="8">
            <NTag size="small" round :type="snapshot.historyHeadNodeId ? 'info' : 'default'">
              {{ t("wave.historyMeta.head") }} #{{ snapshot.historyHeadNodeId || "0" }}
            </NTag>
            <NTag v-if="snapshot.historyHeadProjectionHash" size="small" round>
              {{ snapshot.historyHeadProjectionHash.slice(0, 8) }}
            </NTag>
          </NSpace>
        </template>
        <NAlert v-if="recentHistory.length" class="mb-4" type="info">
          {{ t("wave.historyMeta.currentHead") }}：{{ recentHistory[0].commandSummary }}
        </NAlert>
        <NEmpty v-if="recentHistory.length === 0" :description="t('common.empty')" />
        <NTimeline v-else horizontal>
          <NTimelineItem
            v-for="(node, index) in recentHistory.slice(0, 5)"
            :key="node.id"
            :type="index === 0 ? 'info' : 'default'"
            :title="node.commandSummary"
            :time="node.createdAt.split('T')[1]?.slice(0, 8) || node.createdAt"
          >
            <div class="mt-1">
              <NTag size="tiny" :bordered="false">{{ historyCommandText(node.commandKind) }}</NTag>
            </div>
          </NTimelineItem>
        </NTimeline>
      </NCard>
    </template>
  </div>
</template>

<style scoped>
.wave-overview-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.glow-card {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

:root[data-theme='dark'] .glow-card {
  border: 1px solid rgba(255, 255, 255, 0.05);
  background: #111827;
}

.glow-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.04);
}

.compact-card :deep(.n-card__content) {
  padding: 16px 20px;
}
</style>
