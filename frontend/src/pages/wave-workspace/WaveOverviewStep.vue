<script setup lang="ts">
import { computed, inject } from "vue";
import { useRouter } from "vue-router";
import { NAlert, NButton, NCard, NEmpty, NGrid, NGridItem, NList, NListItem, NStatistic, NSpace, NTag, NTimeline, NTimelineItem } from "naive-ui";
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

const stepKeyDisplayMap: Record<string, string> = {
  demand_intake: "demand_intake",
  membership_allocation: "membership_allocation",
  demand_mapping: "demand_mapping",
  wave_overview: "wave_overview",
  adjustment_review: "adjustment_review",
  supplier_execution: "supplier_execution",
  shipment_intake: "shipment_intake",
  channel_sync: "channel_sync",
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
</script>

<template>
  <div class="wave-overview-page">
    <NEmpty v-if="!snapshot" :description="t('common.loading')" />

    <template v-else>
      <!-- Next Step Guidance card -->
      <NCard v-if="suggestedNextStep && suggestedNextStep !== 'wave_overview'" class="mb-4" style="border-left: 4px solid var(--n-color-target, #18a058);">
        <div class="flex items-center justify-between gap-4">
          <div>
            <div class="app-kicker">{{ t("wave.overviewDetail.suggestedNext") }}</div>
            <div class="app-heading-sm mt-1">{{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}</div>
            <p v-if="nextStepReasonText(nextStepReason)" class="app-copy mt-1" style="opacity:0.75;">
              {{ nextStepReasonText(nextStepReason) }}
            </p>
          </div>
          <NButton type="primary" size="small" @click="goTo(suggestedNextStep)">
            {{ t("wave.overviewDetail.goToStep", { step: stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }) }}
          </NButton>
        </div>
      </NCard>

      <!-- Blocking Issues alert -->
      <NAlert v-if="blockingIssues.length" type="warning" class="mb-4" :title="t('wave.overviewDetail.blockingIssues')">
        <ul style="margin: 4px 0; padding-left: 1.2em;">
          <li v-for="issue in blockingIssues" :key="issue">{{ blockingIssueText(issue) }}</li>
        </ul>
      </NAlert>

      <NCard class="mb-4">
        <div class="flex items-start justify-between gap-6">
          <div>
            <div class="app-kicker">{{ lifecycleText(snapshot.projectedLifecycleStage) }}</div>
            <h2 class="app-title mt-2">{{ t("wave.previewDecision") }}</h2>
            <p class="app-copy mt-3">{{ t("wave.previewDescription") }}</p>
          </div>
          <NSpace vertical>
            <NButton type="primary" @click="goTo('supplier_execution')">
              {{ t("wave.continueToExecution") }}
            </NButton>
            <NButton secondary @click="goTo('adjustment_review')">
              {{ t("wave.adjustment") }}
            </NButton>
          </NSpace>
        </div>
      </NCard>

      <NAlert v-if="guidance.length" type="warning" class="mb-4">
        <NSpace vertical :size="10">
          <div class="app-heading-sm">{{ t("wave.nextAction") }}</div>
          <div
            v-for="item in guidance"
            :key="item.code"
            class="flex items-center justify-between gap-4"
          >
            <span>{{ t(`wave.guidance.${item.code}`) || item.code }} ({{ item.count }})</span>
            <NButton size="small" @click="goTo(item.targetStepKey)">
              {{ item.targetStepKey }}
            </NButton>
          </div>
        </NSpace>
      </NAlert>

      <NGrid :cols="3" :x-gap="16" :y-gap="16" class="mb-5">
        <NGridItem>
          <NCard :title="t('wave.exceptionsGroup')">
            <NSpace vertical :size="10">
              <div class="flex items-center justify-between gap-4">
                <span>{{ t("wave.allocation") }}</span>
                <NButton size="small" @click="goTo('membership_allocation')">
                  {{ t("wave.allocation") }}
                </NButton>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span>{{ t("wave.mapping") }}</span>
                <NButton size="small" @click="goTo('demand_mapping')">
                  {{ t("wave.mapping") }}
                </NButton>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span>{{ t("wave.adjustment") }}</span>
                <NButton size="small" @click="goTo('adjustment_review')">
                  {{ t("wave.adjustment") }}
                </NButton>
              </div>
            </NSpace>
          </NCard>
        </NGridItem>

        <NGridItem>
          <NCard :title="t('wave.routingGroup')">
            <NList bordered>
              <NListItem>{{ summaryText("ready") }}: {{ overview?.acceptedReadyOrNotRequired ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("waiting_for_input") }}: {{ overview?.acceptedWaitingForInput ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("deferred") }}: {{ overview?.deferredCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("excluded") }}: {{ (overview?.excludedManualCount ?? 0) + (overview?.excludedDuplicateCount ?? 0) + (overview?.excludedRevokedCount ?? 0) }}</NListItem>
            </NList>
          </NCard>
        </NGridItem>

        <NGridItem>
          <NCard :title="t('wave.executionGroup')">
            <NList bordered>
              <NListItem>{{ summaryText("supplier_orders") }}: {{ overview?.supplierOrderCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("shipments") }}: {{ overview?.shipmentCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("pending_sync") }}: {{ overview?.channelSyncPendingCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("manual_closure_candidates") }}: {{ overview?.manualClosureCandidateCount ?? 0 }}</NListItem>
            </NList>
          </NCard>
        </NGridItem>
      </NGrid>

      <NGrid :cols="4" :x-gap="16" :y-gap="16" class="mb-5">
        <NGridItem><NCard><NStatistic :label="summaryText('demand')" :value="overview?.demandCount ?? 0" /></NCard></NGridItem>
        <NGridItem><NCard><NStatistic :label="summaryText('fulfillment')" :value="overview?.fulfillmentCount ?? 0" /></NCard></NGridItem>
        <NGridItem><NCard><NStatistic :label="summaryText('supplier_orders')" :value="overview?.supplierOrderCount ?? 0" /></NCard></NGridItem>
        <NGridItem><NCard><NStatistic :label="summaryText('shipments')" :value="overview?.shipmentCount ?? 0" /></NCard></NGridItem>
      </NGrid>

      <!-- Fulfillment Breakdown -->
      <NCard :title="t('wave.overviewDetail.fulfillmentBreakdown')" class="mb-5">
        <NGrid :cols="3" :x-gap="16" :y-gap="16">
          <NGridItem>
            <NSpace vertical :size="8">
              <div class="app-kicker">Allocation</div>
              <NGrid :cols="2" :x-gap="12">
                <NGridItem><NStatistic label="Draft" :value="overview?.fulfillmentDraftCount ?? 0" /></NGridItem>
                <NGridItem><NStatistic label="Ready" :value="overview?.fulfillmentReadyCount ?? 0" /></NGridItem>
              </NGrid>
            </NSpace>
          </NGridItem>
          <NGridItem>
            <NSpace vertical :size="8">
              <div class="app-kicker">Address</div>
              <NGrid :cols="3" :x-gap="8">
                <NGridItem>
                  <NStatistic label="Missing" :value="overview?.addressMissingCount ?? 0">
                    <template v-if="(overview?.addressMissingCount ?? 0) > 0" #prefix>
                      <NTag size="tiny" type="warning" :bordered="false" style="margin-right:4px;">!</NTag>
                    </template>
                  </NStatistic>
                </NGridItem>
                <NGridItem><NStatistic label="Ready" :value="overview?.addressReadyCount ?? 0" /></NGridItem>
                <NGridItem>
                  <NStatistic label="Invalid" :value="overview?.addressInvalidCount ?? 0">
                    <template v-if="(overview?.addressInvalidCount ?? 0) > 0" #prefix>
                      <NTag size="tiny" type="error" :bordered="false" style="margin-right:4px;">!</NTag>
                    </template>
                  </NStatistic>
                </NGridItem>
              </NGrid>
            </NSpace>
          </NGridItem>
          <NGridItem>
            <NSpace vertical :size="8">
              <div class="app-kicker">Supplier</div>
              <NGrid :cols="3" :x-gap="8">
                <NGridItem><NStatistic label="Pending" :value="overview?.supplierNotSubmittedCount ?? 0" /></NGridItem>
                <NGridItem><NStatistic label="Submitted" :value="overview?.supplierSubmittedCount ?? 0" /></NGridItem>
                <NGridItem><NStatistic label="Shipped" :value="overview?.supplierShippedCount ?? 0" /></NGridItem>
              </NGrid>
            </NSpace>
          </NGridItem>
        </NGrid>
      </NCard>

      <!-- Adjustments Summary -->
      <NCard :title="t('wave.overviewDetail.adjustmentSummary')" class="mb-5">
        <NGrid :cols="5" :x-gap="16">
          <NGridItem><NStatistic label="Total" :value="overview?.adjustmentCount ?? 0" /></NGridItem>
          <NGridItem><NStatistic :label="t('adjustment.add')" :value="overview?.adjustmentAddCount ?? 0" /></NGridItem>
          <NGridItem><NStatistic :label="t('adjustment.reduce')" :value="overview?.adjustmentReduceCount ?? 0" /></NGridItem>
          <NGridItem><NStatistic :label="t('adjustment.replace')" :value="overview?.adjustmentReplaceCount ?? 0" /></NGridItem>
          <NGridItem><NStatistic :label="t('adjustment.remove')" :value="overview?.adjustmentRemoveCount ?? 0" /></NGridItem>
        </NGrid>
      </NCard>

      <NGrid :cols="2" :x-gap="16" :y-gap="16">
        <NGridItem>
          <NCard :title="t('wave.executionGroup')">
            <NList bordered>
              <NListItem>{{ summaryText("channel_sync_jobs") }}: {{ overview?.channelSyncJobCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("pending_sync") }}: {{ overview?.channelSyncPendingCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("failed_sync") }}: {{ overview?.channelSyncFailedCount ?? 0 }}</NListItem>
              <NListItem>{{ summaryText("manual_closure_candidates") }}: {{ overview?.manualClosureCandidateCount ?? 0 }}</NListItem>
            </NList>
          </NCard>
        </NGridItem>

        <NGridItem>
          <NCard :title="t('wave.basis')">
            <NSpace vertical :size="12">
              <div class="flex items-center gap-2">
                <NTag size="small" :type="snapshot.basisSummary.hasDriftedBasis ? 'warning' : 'default'">
                  {{ t("wave.drifted") }}
                </NTag>
                <span>{{ snapshot.basisSummary.driftedCount }}</span>
              </div>
              <div class="flex items-center gap-2">
                <NTag size="small" :type="snapshot.basisSummary.hasRequiredReview ? 'error' : 'default'">
                  {{ t("wave.reviewRequired") }}
                </NTag>
                <span>{{ snapshot.basisSummary.requiredReviewCount }}</span>
              </div>
            </NSpace>
          </NCard>
        </NGridItem>
      </NGrid>

      <NCard class="mt-5" :title="t('wave.history')">
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
        <NTimeline v-else>
          <NTimelineItem
            v-for="node in recentHistory"
            :key="node.id"
            :title="node.commandSummary"
            :time="node.createdAt"
          >
            <NSpace align="center" :size="8">
              <NTag size="tiny" :bordered="false">{{ historyCommandText(node.commandKind) }}</NTag>
              <span v-if="node.parentNodeId">{{ t("wave.historyMeta.parent") }} #{{ node.parentNodeId }}</span>
              <span v-if="node.preferredRedoChildId">{{ t("wave.historyMeta.redo") }} #{{ node.preferredRedoChildId }}</span>
              <span v-if="node.checkpointHint">{{ t("wave.historyMeta.checkpoint") }}</span>
            </NSpace>
          </NTimelineItem>
        </NTimeline>
      </NCard>
    </template>
  </div>
</template>
