<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NStep, NSteps, NTag } from "naive-ui";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const props = defineProps<{
  snapshot?: dto.WaveWorkspaceSnapshotDTO | null
}>()

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const waveId = computed(() => route.params.waveId as string | undefined);

const stepDefs = computed(() => [
  { key: "wave_overview", title: t("wave.overview"), path: "" },
  { key: "membership_allocation", title: t("wave.allocation"), path: "allocation" },
  { key: "demand_mapping", title: t("wave.mapping"), path: "demand-mapping" },
  { key: "adjustment_review", title: t("wave.adjustment"), path: "adjustment-review" },
  { key: "supplier_execution", title: t("wave.execution"), path: "export" },
  { key: "shipment_intake", title: t("wave.shipment"), path: "shipment" },
  { key: "channel_sync", title: t("wave.sync"), path: "channel-sync" },
]);

const currentStep = computed(() => {
  const name = route.name as string;
  if (name === "wave-overview-step") return 1;
  if (name === "wave-allocation") return 2;
  if (name === "wave-demand-mapping") return 3;
  if (name === "wave-adjustment-review") return 4;
  if (name === "wave-export") return 5;
  if (name === "wave-shipment") return 6;
  if (name === "wave-channel-sync") return 7;
  return 1;
});

const stepStateMap = computed(() => {
  const map = new Map<string, dto.WaveStepStateDTO>();
  for (const step of props.snapshot?.stepStates || []) {
    map.set(step.stepKey, step);
  }
  return map;
});

const stepStatusTextMap: Record<string, string> = {
  idle: t("wave.stepStatus.idle"),
  active: t("wave.stepStatus.active"),
  available: t("wave.stepStatus.available"),
  current: t("wave.stepStatus.current"),
};

function navigateTo(index: number) {
  const step = stepDefs.value[index - 1];
  if (!step || !waveId.value) return;
  const base = `/waves/${waveId.value}`;
  router.push(step.path ? `${base}/${step.path}` : base);
}

function statusText(key: string) {
  const status = stepStateMap.value.get(key)?.status || "idle";
  return stepStatusTextMap[status] || status;
}

function stepSummaryTitle(key: string) {
  const map: Record<string, string> = {
    wave_overview: t("wave.overview"),
    membership_allocation: t("wave.allocation"),
    demand_mapping: t("wave.mapping"),
    adjustment_review: t("wave.adjustment"),
    supplier_execution: t("wave.execution"),
    shipment_intake: t("wave.shipment"),
    channel_sync: t("wave.sync"),
  };
  return map[key] || key;
}
</script>

<template>
  <div class="wave-step-shell">
    <NSteps :current="currentStep" status="process" @update:current="navigateTo">
      <NStep
        v-for="step in stepDefs"
        :key="step.key"
        :title="step.title"
        :description="statusText(step.key)"
      />
    </NSteps>

    <div class="wave-step-summary">
      <div
        v-for="step in stepDefs"
        :key="step.key"
        class="wave-step-summary__item"
      >
        <span>{{ stepSummaryTitle(step.key) }}</span>
        <NTag size="small" :bordered="false">
          {{ stepStateMap.get(step.key)?.primaryCount ?? 0 }}
        </NTag>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wave-step-shell {
  margin-bottom: 16px;
  padding: 14px 16px 12px;
  border-radius: 18px;
  background: linear-gradient(180deg, var(--surface-strong) 0%, var(--surface-muted) 100%);
  border: 1px solid rgba(148, 163, 184, 0.18);
}

.wave-step-summary {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}

.wave-step-summary__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 12px;
  background: rgba(148, 163, 184, 0.08);
  color: var(--text);
  font-size: 12px;
}
</style>
