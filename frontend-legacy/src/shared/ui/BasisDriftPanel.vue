<template>
  <div class="basis-drift-panel">
    <div class="panel-header">
      <div class="panel-title">{{ t("basisDrift.title") }}</div>
      <div class="summary-badges" v-if="props.snapshot?.basisSummary">
        <NTag
          v-if="props.snapshot.basisSummary.hasRequiredReview"
          type="error"
          size="small"
          round
          :bordered="false"
        >
          {{ t("basisDrift.reviewRequired") }} ·
          {{ props.snapshot.basisSummary.requiredReviewCount }}
        </NTag>
        <NTag
          v-else-if="props.snapshot.basisSummary.hasDriftedBasis"
          type="warning"
          size="small"
          round
          :bordered="false"
        >
          {{ t("basisDrift.drifted") }} ·
          {{ props.snapshot.basisSummary.driftedCount }}
        </NTag>
        <NTag v-else type="success" size="small" round :bordered="false">
          {{ t("basisDrift.inSync") }}
        </NTag>
      </div>
    </div>

    <div class="basis-grid">
      <div
        v-for="kind in BASIS_KINDS"
        :key="kind.code"
        class="basis-tile"
        :class="getTileClass(kind.code)"
      >
        <div class="tile-label">{{ kindLabel(kind.code) }}</div>
        <div class="tile-status">
          <NTag
            :type="driftTagType(kind.code)"
            size="tiny"
            round
            :bordered="false"
          >
            {{ driftLabel(kind.code) }}
          </NTag>
          <NTag
            :type="reviewTagType(kind.code)"
            size="tiny"
            round
            :bordered="false"
          >
            {{ reviewLabel(kind.code) }}
          </NTag>
        </div>
        <div v-if="reasonsFor(kind.code).length" class="tile-reasons">
          <span v-for="r in reasonsFor(kind.code)" :key="r" class="reason-pill">
            {{ r }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NTag } from "naive-ui";
import type { dto } from "@/../wailsjs/go/models";
import { useI18n } from "@/shared/i18n";

const { t } = useI18n();

const props = defineProps<{
  snapshot: dto.WaveWorkspaceSnapshotDTO | null;
}>();

const BASIS_KINDS = [
  { code: "supplier_order_basis", labelKey: "basisDrift.exportBasis" },
  { code: "shipment_basis", labelKey: "basisDrift.shipmentBasis" },
  { code: "channel_sync_basis", labelKey: "basisDrift.syncBasis" },
  { code: "adjustment_basis", labelKey: "basisDrift.adjustmentBasis" },
] as const;

const signalMap = computed(() => {
  const map = new Map<string, dto.BasisDriftSignalDTO>();
  for (const sig of props.snapshot?.overview?.basisDriftSignals || []) {
    map.set(sig.basisKind, sig);
  }
  return map;
});

function kindLabel(code: string): string {
  const k = BASIS_KINDS.find((x) => x.code === code);
  return k ? t(k.labelKey) : code;
}

function getTileClass(code: string) {
  const sig = signalMap.value.get(code);
  if (!sig) return "tile-na";
  if (sig.reviewRequirement === "required") return "tile-required";
  if (sig.basisDriftStatus === "drifted") return "tile-drifted";
  return "tile-in-sync";
}

function driftLabel(code: string): string {
  const sig = signalMap.value.get(code);
  if (!sig) return t("basisDrift.notApplicable");
  return sig.basisDriftStatus === "drifted"
    ? t("basisDrift.drifted")
    : t("basisDrift.inSync");
}

function driftTagType(code: string): "default" | "warning" | "success" {
  const sig = signalMap.value.get(code);
  if (!sig) return "default";
  return sig.basisDriftStatus === "drifted" ? "warning" : "success";
}

function reviewLabel(code: string): string {
  const sig = signalMap.value.get(code);
  if (!sig) return "—";
  switch (sig.reviewRequirement) {
    case "required":
      return t("basisDrift.reviewRequired");
    case "recommended":
      return t("basisDrift.reviewRecommended");
    default:
      return t("basisDrift.reviewNone");
  }
}

function reviewTagType(code: string): "default" | "warning" | "error" | "success" {
  const sig = signalMap.value.get(code);
  if (!sig) return "default";
  switch (sig.reviewRequirement) {
    case "required":
      return "error";
    case "recommended":
      return "warning";
    default:
      return "success";
  }
}

function reasonsFor(code: string): string[] {
  return signalMap.value.get(code)?.driftReasonCodes || [];
}
</script>

<style scoped>
.basis-drift-panel {
  background: var(--surface-strong, rgba(255, 255, 255, 0.04));
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 14px;
  padding: 18px 20px;
}

:root[data-theme='dark'] .basis-drift-panel {
  background: rgba(15, 23, 42, 0.45);
  border-color: rgba(255, 255, 255, 0.06);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.panel-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text);
}

.summary-badges {
  display: flex;
  gap: 6px;
}

.basis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.basis-tile {
  border-radius: 10px;
  padding: 12px 14px;
  background: rgba(148, 163, 184, 0.06);
  border: 1px solid rgba(148, 163, 184, 0.12);
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s ease;
}

.tile-required {
  background: rgba(239, 68, 68, 0.06);
  border-color: rgba(239, 68, 68, 0.3);
}

.tile-drifted {
  background: rgba(234, 179, 8, 0.06);
  border-color: rgba(234, 179, 8, 0.3);
}

.tile-in-sync {
  background: rgba(34, 197, 94, 0.04);
  border-color: rgba(34, 197, 94, 0.2);
}

.tile-na {
  opacity: 0.6;
}

.tile-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--muted);
}

.tile-status {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tile-reasons {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 2px;
}

.reason-pill {
  font-size: 0.7rem;
  background: rgba(148, 163, 184, 0.12);
  padding: 2px 8px;
  border-radius: 999px;
  color: var(--muted);
}
</style>
