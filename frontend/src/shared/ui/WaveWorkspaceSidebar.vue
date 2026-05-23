<script setup lang="ts">
import { computed, h } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NTag, NIcon, NMenu } from "naive-ui";
import { 
  GridOutline,
  PeopleOutline,
  ListOutline,
  CheckmarkCircleOutline,
  CloudUploadOutline,
  AirplaneOutline,
  SyncOutline,
  LockClosedOutline
} from "@vicons/ionicons5";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const props = defineProps<{
  snapshot?: dto.WaveWorkspaceSnapshotDTO | null
}>()

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const waveId = computed(() => route.params.waveId as string | undefined);
const demandKinds = computed(() => props.snapshot?.overview?.demandKinds || []);

const stepStateMap = computed(() => {
  const map = new Map<string, dto.WaveStepStateDTO>();
  for (const step of props.snapshot?.stepStates || []) {
    map.set(step.stepKey, step);
  }
  return map;
});

function getStepStatus(key: string): 'current' | 'available' | 'active' | 'idle' {
  if (key === 'demand_intake') return 'available'; 
  const state = stepStateMap.value.get(key);
  return (state?.status as any) || 'idle';
}

function renderIcon(icon: any, status: string) {
  if (status === 'idle') {
    return () => h(NIcon, null, { default: () => h(LockClosedOutline) });
  }
  return () => h(NIcon, null, { default: () => h(icon) });
}

function renderLabel(label: string, count?: number) {
  if (count !== undefined && count > 0) {
    return () => h('div', { class: 'flex justify-between items-center w-full pr-2' }, [
      h('span', null, label),
      h(NTag, { size: 'tiny', round: true, type: 'info', bordered: false }, { default: () => count })
    ]);
  }
  return () => h('span', null, label);
}

const menuOptions = computed(() => {
  const options: any[] = [
    {
      label: renderLabel(t("wave.overview")),
      key: "wave_overview",
      icon: renderIcon(GridOutline, "available"),
      path: ""
    },
    {
      label: renderLabel("Intake Demands"),
      key: "demand_intake",
      icon: renderIcon(CloudUploadOutline, "available"),
      path: "intake"
    }
  ];

  // Allocation with children based on demandKinds
  const allocStatus = getStepStatus("membership_allocation");
  const allocChildren = demandKinds.value.length > 0 
    ? demandKinds.value.map(k => ({
        label: renderLabel(k.charAt(0).toUpperCase() + k.slice(1).replace('_', ' ')),
        key: `alloc_${k}`,
        path: `allocation/${k}`
      }))
    : undefined;

  options.push({
    label: renderLabel(t("wave.allocation"), stepStateMap.value.get("membership_allocation")?.primaryCount),
    key: "membership_allocation",
    icon: renderIcon(PeopleOutline, allocStatus),
    path: allocChildren ? undefined : "allocation",
    children: allocChildren,
    disabled: allocStatus === 'idle'
  });

  // Demand Mapping with children based on demandKinds
  const mapStatus = getStepStatus("demand_mapping");
  const mapChildren = demandKinds.value.length > 0 
    ? demandKinds.value.map(k => ({
        label: renderLabel(k.charAt(0).toUpperCase() + k.slice(1).replace('_', ' ')),
        key: `map_${k}`,
        path: `demand-mapping/${k}`
      }))
    : undefined;

  options.push({
    label: renderLabel(t("wave.mapping"), stepStateMap.value.get("demand_mapping")?.primaryCount),
    key: "demand_mapping",
    icon: renderIcon(ListOutline, mapStatus),
    path: mapChildren ? undefined : "demand-mapping",
    children: mapChildren,
    disabled: mapStatus === 'idle'
  });

  // The rest
  options.push({
    label: renderLabel(t("wave.adjustment"), stepStateMap.value.get("adjustment_review")?.primaryCount),
    key: "adjustment_review",
    icon: renderIcon(CheckmarkCircleOutline, getStepStatus("adjustment_review")),
    path: "adjustment-review",
    disabled: getStepStatus("adjustment_review") === 'idle'
  });

  options.push({
    label: renderLabel(t("wave.execution"), stepStateMap.value.get("supplier_execution")?.primaryCount),
    key: "supplier_execution",
    icon: renderIcon(CloudUploadOutline, getStepStatus("supplier_execution")),
    path: "export",
    disabled: getStepStatus("supplier_execution") === 'idle'
  });

  options.push({
    label: renderLabel(t("wave.shipment"), stepStateMap.value.get("shipment_intake")?.primaryCount),
    key: "shipment_intake",
    icon: renderIcon(AirplaneOutline, getStepStatus("shipment_intake")),
    path: "shipment",
    disabled: getStepStatus("shipment_intake") === 'idle'
  });

  options.push({
    label: renderLabel(t("wave.sync"), stepStateMap.value.get("channel_sync")?.primaryCount),
    key: "channel_sync",
    icon: renderIcon(SyncOutline, getStepStatus("channel_sync")),
    path: "channel-sync",
    disabled: getStepStatus("channel_sync") === 'idle'
  });

  return options;
});

const currentMenuKey = computed(() => {
  const name = route.name as string;
  const kind = route.params.demandKind as string;
  if (name === "wave-overview-step") return "wave_overview";
  if (name === "wave-intake") return "demand_intake";
  if (name === "wave-allocation") return kind ? `alloc_${kind}` : "membership_allocation";
  if (name === "wave-demand-mapping") return kind ? `map_${kind}` : "demand_mapping";
  if (name === "wave-adjustment-review") return "adjustment_review";
  if (name === "wave-export") return "supplier_execution";
  if (name === "wave-shipment") return "shipment_intake";
  if (name === "wave-channel-sync") return "channel_sync";
  return "wave_overview";
});

const expandedKeys = computed(() => {
  return ["membership_allocation", "demand_mapping"];
});

function handleMenuUpdateValue(key: string, item: any) {
  if (!waveId.value) return;
  
  if (item.path !== undefined) {
    const base = `/waves/${waveId.value}`;
    router.push(item.path ? `${base}/${item.path}` : base);
  } else if (item.children && item.children.length > 0) {
    // Navigate to first child
    const base = `/waves/${waveId.value}`;
    router.push(`${base}/${item.children[0].path}`);
  }
}
</script>

<template>
  <div class="wave-workspace-sidebar">
    <div class="wave-sidebar-header">
      <div class="app-kicker">Wave Workspace</div>
      <div class="wave-title" :title="snapshot?.wave?.name">{{ snapshot?.wave?.name || 'Loading...' }}</div>
      <div class="wave-number">{{ snapshot?.wave?.waveNo }}</div>
    </div>
    
    <div class="wave-menu-container">
      <NMenu
        :options="menuOptions"
        :value="currentMenuKey"
        :default-expanded-keys="expandedKeys"
        @update:value="handleMenuUpdateValue"
        class="wave-sidebar-menu"
        :indent="24"
      />
    </div>
  </div>
</template>

<style scoped>
.wave-workspace-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--surface-strong);
  border-right: 1px solid rgba(148, 163, 184, 0.12);
  padding: 16px 0;
}

:root[data-theme='dark'] .wave-workspace-sidebar {
  background: rgba(15, 23, 42, 0.6);
  border-right: 1px solid rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
}

.wave-sidebar-header {
  padding: 0 20px 24px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
  margin-bottom: 12px;
}

:root[data-theme='dark'] .wave-sidebar-header {
  border-bottom-color: rgba(255, 255, 255, 0.05);
}

.wave-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  margin-top: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.wave-number {
  font-size: 12px;
  color: var(--muted);
  font-family: monospace;
  margin-top: 2px;
}

.wave-menu-container {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px;
}

/* Customizing NMenu to fit our aesthetics */
:deep(.n-menu-item-content) {
  border-radius: 8px;
  margin-bottom: 4px;
}

:deep(.n-menu-item-content.n-menu-item-content--selected) {
  background: var(--accent-surface);
  color: var(--accent);
  font-weight: 600;
}
</style>
