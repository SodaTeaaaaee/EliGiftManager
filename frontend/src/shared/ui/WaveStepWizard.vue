<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NTag, NTooltip } from "naive-ui";
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
  { key: "wave_overview", title: t("wave.overview"), path: "", icon: "overview" },
  { key: "membership_allocation", title: t("wave.allocation"), path: "allocation", icon: "allocation" },
  { key: "demand_mapping", title: t("wave.mapping"), path: "demand-mapping", icon: "mapping" },
  { key: "adjustment_review", title: t("wave.adjustment"), path: "adjustment-review", icon: "adjustment" },
  { key: "supplier_execution", title: t("wave.execution"), path: "export", icon: "export" },
  { key: "shipment_intake", title: t("wave.shipment"), path: "shipment", icon: "shipment" },
  { key: "channel_sync", title: t("wave.sync"), path: "channel-sync", icon: "sync" },
]);

const currentStepKey = computed(() => {
  const name = route.name as string;
  if (name === "wave-overview-step") return "wave_overview";
  if (name === "wave-allocation") return "membership_allocation";
  if (name === "wave-demand-mapping") return "demand_mapping";
  if (name === "wave-adjustment-review") return "adjustment_review";
  if (name === "wave-export") return "supplier_execution";
  if (name === "wave-shipment") return "shipment_intake";
  if (name === "wave-channel-sync") return "channel_sync";
  return "wave_overview";
});

const currentStepIndex = computed(() => {
  return stepDefs.value.findIndex(s => s.key === currentStepKey.value);
});

const stepStateMap = computed(() => {
  const map = new Map<string, dto.WaveStepStateDTO>();
  for (const step of props.snapshot?.stepStates || []) {
    map.set(step.stepKey, step);
  }
  return map;
});

function navigateTo(stepKey: string) {
  const step = stepDefs.value.find(s => s.key === stepKey);
  if (!step || !waveId.value) return;
  
  // Only allow navigation to active/available/current steps
  const state = stepStateMap.value.get(stepKey);
  if (!state || state.status === "idle") {
    // If it's overview, always allow
    if (stepKey !== "wave_overview") return;
  }
  
  const base = `/waves/${waveId.value}`;
  router.push(step.path ? `${base}/${step.path}` : base);
}

function getStepStatus(key: string): 'current' | 'available' | 'active' | 'idle' {
  if (key === currentStepKey.value) return 'current';
  const state = stepStateMap.value.get(key);
  return (state?.status as any) || 'idle';
}

function getStepClass(key: string, index: number) {
  const status = getStepStatus(key);
  const isPast = index < currentStepIndex.value;
  return {
    'wave-step-item': true,
    [`is-${status}`]: true,
    'is-past': isPast,
    'is-clickable': status !== 'idle' || key === 'wave_overview',
  };
}

function countBadgeType(key: string) {
  const status = getStepStatus(key);
  if (status === 'current') return 'primary';
  if (status === 'available') return 'success';
  return 'default';
}
</script>

<template>
  <div class="wave-workflow-shell">
    <div class="wave-steps-flow">
      <template v-for="(step, idx) in stepDefs" :key="step.key">
        <!-- Step Node -->
        <div 
          :class="getStepClass(step.key, idx)" 
          @click="navigateTo(step.key)"
        >
          <div class="wave-step-node">
            <!-- Dynamic SVG Icons -->
            <svg v-if="step.icon === 'overview'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"/>
            </svg>
            <svg v-else-if="step.icon === 'allocation'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5s-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/>
            </svg>
            <svg v-else-if="step.icon === 'mapping'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>
            </svg>
            <svg v-else-if="step.icon === 'adjustment'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
            </svg>
            <svg v-else-if="step.icon === 'export'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM17 13l-5 5-5-5h3V9h4v4h3z"/>
            </svg>
            <svg v-else-if="step.icon === 'shipment'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M20 8h-3V4H3c-1.1 0-2 .9-2 2v11h2c0 1.66 1.34 3 3 3s3-1.34 3-3h6c0 1.66 1.34 3 3 3s3-1.34 3-3h2v-5l-3-4zM6 18.5c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5zm12.5-9l2.25 3H17V9.5h1.5zM18 18.5c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"/>
            </svg>
            <svg v-else-if="step.icon === 'sync'" viewBox="0 0 24 24" class="w-5 h-5 icon-svg">
              <path d="M12 4V1L8 5l4 4V6c3.31 0 6 2.69 6 6 0 1.01-.25 1.97-.7 2.8l1.46 1.46C19.54 15.03 20 13.57 20 12c0-4.42-3.58-8-8-8zm-6 8c0-1.01.25-1.97.7-2.8L5.24 7.74C4.46 8.97 4 10.43 4 12c0 4.42 3.58 8 8 8v3l4-4-4-4v3c-3.31 0-6-2.69-6-6z"/>
            </svg>

            <!-- Status Indicator Overlay -->
            <div class="node-status-dot"></div>
          </div>
          
          <div class="wave-step-info">
            <div class="wave-step-title">{{ step.title }}</div>
            <div class="wave-step-badge-wrap" v-if="stepStateMap.get(step.key)">
              <NTag 
                size="tiny" 
                round
                :type="countBadgeType(step.key)"
                :bordered="false"
              >
                {{ stepStateMap.get(step.key)?.primaryCount ?? 0 }}
              </NTag>
            </div>
          </div>
        </div>

        <!-- Connector Line -->
        <div 
          v-if="idx < stepDefs.length - 1" 
          :class="{
            'wave-step-connector': true,
            'is-active': idx < currentStepIndex,
            'is-next-available': getStepStatus(stepDefs[idx + 1].key) !== 'idle'
          }"
        ></div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.wave-workflow-shell {
  margin-bottom: 24px;
  padding: 16px 20px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.08) 0%, rgba(255, 255, 255, 0.02) 100%);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.03);
}

:root[data-theme='dark'] .wave-workflow-shell {
  background: linear-gradient(135deg, rgba(15, 23, 42, 0.6) 0%, rgba(15, 23, 42, 0.25) 100%);
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.wave-steps-flow {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.wave-step-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  position: relative;
  user-select: none;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.wave-step-node {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-strong);
  border: 2px solid var(--border-color, rgba(148, 163, 184, 0.2));
  color: var(--muted);
  position: relative;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.icon-svg {
  fill: currentColor;
  transition: transform 0.3s ease;
}

.wave-step-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.wave-step-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
  transition: color 0.3s ease;
  white-space: nowrap;
}

.wave-step-badge-wrap {
  transform: scale(0.85);
  margin-top: -2px;
}

.node-status-dot {
  position: absolute;
  bottom: -2px;
  right: -2px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: transparent;
  border: 2px solid var(--surface-strong);
  transition: all 0.3s ease;
}

/* Step Status: Idle (Locked) */
.wave-step-item.is-idle .wave-step-node {
  opacity: 0.5;
  cursor: not-allowed;
}
.wave-step-item.is-idle .wave-step-title {
  opacity: 0.5;
}

/* Step Status: Active / Available */
.wave-step-item.is-available .wave-step-node,
.wave-step-item.is-active .wave-step-node {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--accent-surface);
}
.wave-step-item.is-available .wave-step-title,
.wave-step-item.is-active .wave-step-title {
  color: var(--text);
}

.wave-step-item.is-available .node-status-dot,
.wave-step-item.is-active .node-status-dot {
  background: var(--success-color, #16a34a);
}

/* Step Status: Current (Active selection) */
.wave-step-item.is-current .wave-step-node {
  border-color: var(--accent);
  color: #ffffff;
  background: var(--accent);
  box-shadow: 0 0 12px rgba(37, 99, 235, 0.4);
  transform: scale(1.1);
}
:root[data-theme='dark'] .wave-step-item.is-current .wave-step-node {
  box-shadow: 0 0 16px rgba(96, 165, 250, 0.5);
  color: #0f172a;
}
.wave-step-item.is-current .wave-step-title {
  color: var(--accent);
  font-weight: 700;
}
.wave-step-item.is-current .node-status-dot {
  background: var(--accent);
  transform: scale(1.2);
}

/* Hover effects */
.wave-step-item.is-clickable:hover {
  cursor: pointer;
}
.wave-step-item.is-clickable:hover .wave-step-node {
  transform: translateY(-2px);
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.15);
}
.wave-step-item.is-clickable:hover .icon-svg {
  transform: scale(1.1);
}
.wave-step-item.is-clickable:hover .wave-step-title {
  color: var(--accent);
}

/* Past Steps */
.wave-step-item.is-past .node-status-dot {
  background: var(--success-color, #16a34a);
}

/* Connectors */
.wave-step-connector {
  flex: 1;
  height: 2px;
  background: rgba(148, 163, 184, 0.15);
  margin: 0 12px;
  margin-top: -24px;
  position: relative;
  border-radius: 1px;
  transition: all 0.3s ease;
}

.wave-step-connector.is-next-available {
  background: var(--accent-surface);
}

.wave-step-connector.is-active {
  background: var(--accent);
}
</style>

