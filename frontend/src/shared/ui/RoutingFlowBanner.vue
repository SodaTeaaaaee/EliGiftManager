<template>
  <div class="routing-flow-banner">
    <div class="banner-header">
      <div class="banner-title">{{ t("routingFlow.title") }}</div>
      <div class="banner-question">{{ t("routingFlow.intentQuestion") }}</div>
    </div>

    <div class="intent-grid">
      <button
        type="button"
        class="intent-card intent-source"
        @click="goToInbox"
      >
        <div class="intent-num">①</div>
        <div class="intent-text">
          <div class="intent-label">{{ t("routingFlow.intentSourceTruth") }}</div>
          <div class="intent-action">{{ t("routingFlow.intentSourceTruthAction") }}</div>
        </div>
      </button>

      <button
        type="button"
        class="intent-card intent-default"
        @click="goToAllocation"
      >
        <div class="intent-num">②</div>
        <div class="intent-text">
          <div class="intent-label">{{ t("routingFlow.intentDefaultLogic") }}</div>
          <div class="intent-action">{{ t("routingFlow.intentDefaultLogicAction") }}</div>
        </div>
      </button>

      <button
        type="button"
        class="intent-card intent-exception"
        @click="goToAdjustment"
      >
        <div class="intent-num">③</div>
        <div class="intent-text">
          <div class="intent-label">{{ t("routingFlow.intentWaveException") }}</div>
          <div class="intent-action">{{ t("routingFlow.intentWaveExceptionAction") }}</div>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { useI18n } from "@/shared/i18n";

const props = defineProps<{
  waveId: number | null;
  defaultAllocationKind?: "membership" | "demand";
}>();

const { t } = useI18n();
const router = useRouter();

function goToInbox() {
  router.push("/demand-inbox");
}

function goToAllocation() {
  if (!props.waveId) return;
  const kind = props.defaultAllocationKind || "membership";
  router.push(`/waves/${props.waveId}/allocation/${kind}`);
}

function goToAdjustment() {
  if (!props.waveId) return;
  router.push(`/waves/${props.waveId}/adjustment-review`);
}
</script>

<style scoped>
.routing-flow-banner {
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.04) 0%,
    rgba(99, 102, 241, 0.01) 100%
  );
  border: 1px solid rgba(99, 102, 241, 0.18);
  border-radius: 14px;
  padding: 18px 20px;
}

:root[data-theme='dark'] .routing-flow-banner {
  background: linear-gradient(
    135deg,
    rgba(99, 102, 241, 0.1) 0%,
    rgba(99, 102, 241, 0.02) 100%
  );
  border-color: rgba(99, 102, 241, 0.28);
}

.banner-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.banner-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text);
}

.banner-question {
  font-size: 0.8rem;
  color: var(--muted);
}

.intent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.intent-card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.14);
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  font-size: inherit;
  color: inherit;
  transition: all 0.2s ease;
}

:root[data-theme='dark'] .intent-card {
  background: rgba(15, 23, 42, 0.45);
  border-color: rgba(255, 255, 255, 0.06);
}

.intent-card:hover {
  border-color: rgba(99, 102, 241, 0.4);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.08);
}

.intent-num {
  font-size: 1.4rem;
  font-weight: 800;
  color: rgba(99, 102, 241, 0.6);
  line-height: 1;
}

.intent-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.intent-label {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text);
}

.intent-action {
  font-size: 0.75rem;
  color: var(--muted);
}
</style>
