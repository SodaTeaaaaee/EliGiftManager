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
      <NCard class="mb-4">
        <div class="flex items-start justify-between gap-6">
          <div>
            <div class="app-kicker">{{ snapshot.projectedLifecycleStage }}</div>
            <h2 class="app-title mt-2">{{ t("wave.previewDecision") }}</h2>
            <p class="app-copy mt-3">
              在这里集中判断当前波次该继续执行、回前置修正，还是进入共享调整层。
            </p>
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
            <span>{{ item.code }} ({{ item.count }})</span>
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
                <span>Back to Membership Allocation</span>
                <NButton size="small" @click="goTo('membership_allocation')">
                  {{ t("wave.allocation") }}
                </NButton>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span>Back to Demand Mapping</span>
                <NButton size="small" @click="goTo('demand_mapping')">
                  {{ t("wave.mapping") }}
                </NButton>
              </div>
              <div class="flex items-center justify-between gap-4">
                <span>Handle final fulfillment exceptions</span>
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
              <NListItem>Ready: {{ overview?.acceptedReadyOrNotRequired ?? 0 }}</NListItem>
              <NListItem>Waiting Input: {{ overview?.acceptedWaitingForInput ?? 0 }}</NListItem>
              <NListItem>Deferred: {{ overview?.deferredCount ?? 0 }}</NListItem>
              <NListItem>Excluded: {{ (overview?.excludedManualCount ?? 0) + (overview?.excludedDuplicateCount ?? 0) + (overview?.excludedRevokedCount ?? 0) }}</NListItem>
            </NList>
          </NCard>
        </NGridItem>

        <NGridItem>
          <NCard :title="t('wave.executionGroup')">
            <NList bordered>
              <NListItem>Supplier Orders: {{ overview?.supplierOrderCount ?? 0 }}</NListItem>
              <NListItem>Shipments: {{ overview?.shipmentCount ?? 0 }}</NListItem>
              <NListItem>Pending Sync: {{ overview?.channelSyncPendingCount ?? 0 }}</NListItem>
              <NListItem>Manual Closure Candidates: {{ overview?.manualClosureCandidateCount ?? 0 }}</NListItem>
            </NList>
          </NCard>
        </NGridItem>
      </NGrid>

      <NGrid :cols="4" :x-gap="16" :y-gap="16" class="mb-5">
        <NGridItem>
          <NCard>
            <NStatistic label="Demand" :value="overview?.demandCount ?? 0" />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard>
            <NStatistic label="Fulfillment" :value="overview?.fulfillmentCount ?? 0" />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard>
            <NStatistic label="Supplier Orders" :value="overview?.supplierOrderCount ?? 0" />
          </NCard>
        </NGridItem>
        <NGridItem>
          <NCard>
            <NStatistic label="Shipments" :value="overview?.shipmentCount ?? 0" />
          </NCard>
        </NGridItem>
      </NGrid>

      <NGrid :cols="2" :x-gap="16" :y-gap="16">
        <NGridItem>
          <NCard :title="t('wave.executionGroup')">
            <NList bordered>
              <NListItem>Channel Sync Jobs: {{ overview?.channelSyncJobCount ?? 0 }}</NListItem>
              <NListItem>Pending Sync: {{ overview?.channelSyncPendingCount ?? 0 }}</NListItem>
              <NListItem>Failed Sync: {{ overview?.channelSyncFailedCount ?? 0 }}</NListItem>
              <NListItem>Manual Closure Candidates: {{ overview?.manualClosureCandidateCount ?? 0 }}</NListItem>
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
        <NTimeline>
          <NTimelineItem
            v-for="node in snapshot.recentHistory"
            :key="node.id"
            :title="node.commandSummary"
            :time="node.createdAt"
          >
            <NTag size="tiny" :bordered="false">{{ node.commandKind }}</NTag>
          </NTimelineItem>
        </NTimeline>
      </NCard>
    </template>
  </div>
</template>
