<script setup lang="ts">
import { computed, inject } from "vue";
import { useRouter } from "vue-router";
import { NAlert, NButton, NEmpty, NGrid, NGridItem, NTag, NIcon, NProgress } from "naive-ui";
import { ArrowForwardOutline, CheckmarkCircleOutline, WarningOutline } from "@vicons/ionicons5";
import { dto } from "@/../wailsjs/go/models";
import { useI18n } from "@/shared/i18n";
import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const snapshot = inject("waveWorkspaceSnapshot", computed(() => null)) as { value: dto.WaveWorkspaceSnapshotDTO | null };
const router = useRouter();
const { t } = useI18n();

const overview = computed(() => snapshot.value?.overview);
const guidance = computed(() => snapshot.value?.guidance || []);
const blockingIssues = computed(() => overview.value?.blockingIssues ?? []);
const suggestedNextStep = computed(() => overview.value?.suggestedNextStep ?? "");
const nextStepReason = computed(() => overview.value?.nextStepReason ?? "");

const stepKeyDisplayMap: Record<string, string> = {
  demand_intake: "Intake Demands",
  membership_allocation: t("wave.allocation"),
  demand_mapping: t("wave.mapping"),
  wave_overview: t("wave.overview"),
  adjustment_review: t("wave.adjustment"),
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

function goTo(stepKey: string) {
  const waveId = snapshot.value?.wave?.id;
  if (!waveId) return;
  const targetMap: Record<string, string> = {
    demand_intake: `/waves/${waveId}/intake`,
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

const allocationTotal = computed(() => overview.value?.fulfillmentCount || 0);
const allocationReadyPct = computed(() => {
  if (allocationTotal.value === 0) return 0;
  return Math.round(((overview.value?.fulfillmentReadyCount || 0) / allocationTotal.value) * 100);
});
</script>

<template>
  <div class="wave-action-center pb-12">
    <NEmpty v-if="!snapshot" :description="t('common.loading')" class="mt-20" />

    <template v-else>
      <PageHeader 
        title="Command Center" 
        description="Monitor status and execute your next action." 
      />

      <!-- Action Required Section -->
      <GlassCard class="mb-8" style="border-left: 4px solid var(--accent);">
        <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
          <div>
            <div class="app-kicker mb-2 text-accent">{{ t("wave.overviewDetail.suggestedNext") }}</div>
            <h2 class="text-3xl font-bold text-slate-800 dark:text-slate-100">
              {{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}
            </h2>
            <p class="text-slate-500 mt-2">
              {{ nextStepReasonText(nextStepReason) || t("wave.previewDescription") }}
            </p>
          </div>
          <div>
            <NButton 
              v-if="suggestedNextStep && suggestedNextStep !== 'wave_overview'" 
              type="primary" 
              size="large"
              round
              icon-placement="right"
              @click="goTo(suggestedNextStep)"
              class="shadow-lg shadow-blue-500/30"
            >
              <template #icon><NIcon><ArrowForwardOutline /></NIcon></template>
              Go to {{ stepKeyDisplayMap[suggestedNextStep] ?? suggestedNextStep }}
            </NButton>
          </div>
        </div>
      </GlassCard>

      <!-- Blocking Issues -->
      <NAlert v-if="blockingIssues.length" type="error" class="mb-8 rounded-lg shadow-sm" :title="t('wave.overviewDetail.blockingIssues')">
        <ul class="list-disc pl-5 mt-2 text-red-600 dark:text-red-400 space-y-1">
          <li v-for="issue in blockingIssues" :key="issue">
            <strong>{{ blockingIssueText(issue) }}</strong> — Requires attention before final export.
          </li>
        </ul>
        <NButton size="small" secondary type="error" @click="goTo('adjustment_review')" class="mt-4">
          Resolve in Adjustment Review
        </NButton>
      </NAlert>

      <NAlert v-if="guidance.length" type="warning" class="mb-8 rounded-lg shadow-sm" :title="t('wave.nextAction')">
        <div class="flex flex-col gap-3 mt-2">
          <div
            v-for="item in guidance"
            :key="item.code"
            class="flex items-center justify-between border-b border-amber-500/20 pb-2 last:border-0 last:pb-0"
          >
            <span class="text-sm font-medium text-amber-700 dark:text-amber-400">
              <NIcon class="mr-1"><WarningOutline /></NIcon>
              {{ t(`wave.guidance.${item.code}`) || item.code }} ({{ item.count }} items)
            </span>
            <NButton size="small" type="warning" secondary @click="goTo(item.targetStepKey)">
              Resolve in {{ stepKeyDisplayMap[item.targetStepKey] ?? item.targetStepKey }}
            </NButton>
          </div>
        </div>
      </NAlert>

      <!-- Core Status Board -->
      <h3 class="app-heading-md mb-4 mt-8">Fulfillment Status</h3>
      <NGrid :cols="3" :x-gap="20" :y-gap="20">
        <NGridItem>
          <GlassCard hoverable>
            <div class="app-kicker mb-4">Allocation Readiness</div>
            <div class="flex items-end gap-3 mb-2">
              <span class="text-4xl font-bold text-slate-800 dark:text-slate-100">{{ allocationReadyPct }}%</span>
              <span class="text-sm text-slate-500 mb-1">Ready</span>
            </div>
            <NProgress type="line" status="success" :percentage="allocationReadyPct" :show-indicator="false" class="mb-3" />
            <div class="flex justify-between text-sm">
              <span class="text-slate-500">Draft: {{ overview?.fulfillmentDraftCount ?? 0 }}</span>
              <span class="text-emerald-600 font-medium">Ready: {{ overview?.fulfillmentReadyCount ?? 0 }} / {{ allocationTotal }}</span>
            </div>
          </GlassCard>
        </NGridItem>
        <NGridItem>
          <GlassCard hoverable>
            <div class="app-kicker mb-4">Demands & Lines</div>
            <div class="space-y-3">
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Incoming Demands</span>
                <span class="font-bold text-slate-800 dark:text-slate-100">{{ overview?.demandCount ?? 0 }}</span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Mapped Lines</span>
                <span class="font-bold text-slate-800 dark:text-slate-100">{{ overview?.fulfillmentCount ?? 0 }}</span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Adjustments</span>
                <NTag size="small" type="info" :bordered="false">{{ overview?.adjustmentCount ?? 0 }} rules</NTag>
              </div>
            </div>
          </GlassCard>
        </NGridItem>
        <NGridItem>
          <GlassCard hoverable>
            <div class="app-kicker mb-4">Execution & Sync</div>
            <div class="space-y-3">
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Factory Orders</span>
                <span class="font-bold text-slate-800 dark:text-slate-100">{{ overview?.supplierOrderCount ?? 0 }}</span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Tracking Loaded</span>
                <span class="font-bold text-emerald-600 dark:text-emerald-400">
                  <NIcon class="mr-1 align-text-bottom"><CheckmarkCircleOutline /></NIcon>
                  {{ overview?.shipmentCount ?? 0 }}
                </span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-slate-500">Sync Jobs</span>
                <span class="font-bold text-blue-600 dark:text-blue-400">{{ overview?.channelSyncJobCount ?? 0 }}</span>
              </div>
            </div>
          </GlassCard>
        </NGridItem>
      </NGrid>
    </template>
  </div>
</template>

<style scoped>
.wave-action-center {
  display: flex;
  flex-direction: column;
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
