<script setup lang="ts">
import { computed, h, type Component } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NTag, NIcon, NMenu, type MenuOption } from "naive-ui";
import {
  GridOutline,
  PeopleOutline,
  ListOutline,
  CheckmarkCircleOutline,
  CloudUploadOutline,
  AirplaneOutline,
  SyncOutline,
  LockClosedOutline,
  ShieldCheckmarkOutline,
  GitNetworkOutline,
  CheckmarkDoneOutline,
} from "@vicons/ionicons5";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const props = defineProps<{
  snapshot?: dto.WaveWorkspaceSnapshotDTO | null;
  collapsed?: boolean;
}>();

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const waveId = computed(() => route.params.waveId as string | undefined);

const demandKinds = computed(
  () => props.snapshot?.overview?.demandKinds || [],
);

const isMembership = computed(() =>
  demandKinds.value.includes("membership_entitlement"),
);
const isRetail = computed(() => demandKinds.value.includes("retail_order"));
const isMixed = computed(() => isMembership.value && isRetail.value);

const stepStateMap = computed(() => {
  const map = new Map<string, dto.WaveStepStateDTO>();
  for (const step of props.snapshot?.stepStates || []) {
    map.set(step.stepKey, step);
  }
  return map;
});

type StepStatus = 'current' | 'available' | 'active' | 'idle'

const VALID_STATUSES: ReadonlySet<string> = new Set<StepStatus>(['current', 'available', 'active', 'idle'])

function getStepStatus(key: string): StepStatus {
  const raw = stepStateMap.value.get(key)?.status
  return VALID_STATUSES.has(raw ?? '') ? (raw as StepStatus) : 'idle'
}

function renderIcon(icon: Component, status: StepStatus) {
  if (status === 'idle') {
    return () => h(NIcon, null, { default: () => h(LockClosedOutline) })
  }
  return () => h(NIcon, null, { default: () => h(icon) })
}

function renderLabelWithCount(
  label: string,
  count?: number,
  badge?: string,
) {
  return () =>
    h(
      "div",
      {
        style:
          "display: flex; justify-content: space-between; align-items: center; width: 100%; padding-right: 8px; gap: 6px;",
      },
      [
        h("span", { style: "flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;" }, label),
        ...(badge
          ? [
              h(
                NTag,
                {
                  size: "tiny",
                  round: true,
                  bordered: false,
                  type: "default",
                },
                { default: () => badge },
              ),
            ]
          : []),
        ...(count !== undefined && count > 0
          ? [
              h(
                NTag,
                {
                  size: "tiny",
                  round: true,
                  bordered: false,
                  type: "info",
                },
                { default: () => count },
              ),
            ]
          : []),
      ],
    );
}

const allocationBadge = computed(() => {
  if (isMixed.value) return t("waveSidebar.mixedBadge");
  if (isMembership.value) return t("waveSidebar.membersOnly");
  if (isRetail.value) return t("waveSidebar.retailOnly");
  return undefined;
});

const menuOptions = computed(() => {
  const opts: any[] = []

  // ── 总览 ──
  opts.push({
    label: renderLabelWithCount(t('waveSidebar.overview')),
    key: 'wave_overview',
    icon: renderIcon(GridOutline, 'available'),
  })

  // ── ① Demand Intake (波次内只读视图) ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionDemandIntake'),
    key: 'section_intake',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.intakeView'),
          stepStateMap.value.get('demand_intake')?.primaryCount,
        ),
        key: 'demand_intake',
        icon: renderIcon(CloudUploadOutline, 'available'),
      },
    ],
  })

  // ── ② Initial Allocation (折叠分组) ──
  const allocChildren: any[] = []
  if (isMembership.value || !isRetail.value) {
    allocChildren.push({
      label: renderLabelWithCount(
        t('waveSidebar.membershipAllocation'),
        stepStateMap.value.get('membership_allocation')?.primaryCount,
      ),
      key: 'membership_allocation',
      icon: renderIcon(PeopleOutline, getStepStatus('membership_allocation')),
    })
  }
  if (isRetail.value || !isMembership.value) {
    allocChildren.push({
      label: renderLabelWithCount(
        t('waveSidebar.demandMapping'),
        stepStateMap.value.get('demand_mapping')?.primaryCount,
      ),
      key: 'demand_mapping',
      icon: renderIcon(ListOutline, getStepStatus('demand_mapping')),
    })
  }
  if (allocChildren.length === 0) {
    allocChildren.push({
      label: renderLabelWithCount(t('waveSidebar.membershipAllocation')),
      key: 'membership_allocation',
      icon: renderIcon(PeopleOutline, 'available'),
    })
  }

  opts.push({
    type: 'group',
    label: () =>
      h(
        'div',
        { style: 'display: flex; align-items: center; gap: 6px;' },
        [
          h('span', null, t('waveSidebar.sectionInitialAllocation')),
          ...(allocationBadge.value
            ? [
                h(
                  NTag,
                  {
                    size: 'tiny',
                    round: true,
                    bordered: false,
                    type: 'default',
                  },
                  { default: () => allocationBadge.value },
                ),
              ]
            : []),
        ],
      ),
    key: 'section_allocation',
    children: allocChildren,
  })

  // ── ③ Adjustment Review ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionAdjustment'),
    key: 'section_adjustment',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.adjustmentReview'),
          stepStateMap.value.get('adjustment_review')?.primaryCount,
        ),
        key: 'adjustment_review',
        icon: renderIcon(
          CheckmarkCircleOutline,
          getStepStatus('adjustment_review'),
        ),
      },
    ],
  })

  // ── ④ Execution Readiness ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionReadiness'),
    key: 'section_readiness',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.readiness'),
          stepStateMap.value.get('execution_readiness')?.primaryCount,
        ),
        key: 'execution_readiness',
        icon: renderIcon(
          ShieldCheckmarkOutline,
          getStepStatus('execution_readiness') === 'idle'
            ? 'available'
            : getStepStatus('execution_readiness'),
        ),
      },
    ],
  })

  // ── ⑤ Supplier Execution ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionSupplierExecution'),
    key: 'section_execution',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.exportNow'),
          stepStateMap.value.get('supplier_execution')?.primaryCount,
        ),
        key: 'supplier_execution',
        icon: renderIcon(
          CloudUploadOutline,
          getStepStatus('supplier_execution'),
        ),
      },
    ],
  })

  // ── ⑥ Shipment Intake ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionShipmentIntake'),
    key: 'section_shipment',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.shipmentIntake'),
          stepStateMap.value.get('shipment_intake')?.primaryCount,
        ),
        key: 'shipment_intake',
        icon: renderIcon(AirplaneOutline, getStepStatus('shipment_intake')),
      },
    ],
  })

  // ── ⑦ Channel Sync / Closure ──
  opts.push({
    type: 'group',
    label: t('waveSidebar.sectionChannelSync'),
    key: 'section_sync',
    children: [
      {
        label: renderLabelWithCount(
          t('waveSidebar.channelSync'),
          stepStateMap.value.get('channel_sync')?.primaryCount,
        ),
        key: 'channel_sync',
        icon: renderIcon(SyncOutline, getStepStatus('channel_sync')),
      },
    ],
  })

  // ── 高级 ──
  opts.push({
    type: 'divider',
    key: 'divider_advanced',
  })

  opts.push({
    label: renderLabelWithCount(t('waveSidebar.historyTree')),
    key: 'history_tree',
    icon: renderIcon(GitNetworkOutline, 'available'),
  })

  return opts as MenuOption[]
})

// 当前激活 menu key
const currentMenuKey = computed(() => {
  const name = route.name as string;
  if (name === "wave-overview-step") return "wave_overview";
  if (name === "wave-intake") return "demand_intake";
  if (
    name === "wave-membership-allocation" ||
    name === "wave-allocation-legacy"
  )
    return "membership_allocation";
  if (
    name === "wave-demand-mapping" ||
    name === "wave-demand-mapping-legacy"
  )
    return "demand_mapping";
  if (name === "wave-adjustment-review") return "adjustment_review";
  if (name === "wave-readiness") return "execution_readiness";
  if (name === "wave-export") return "supplier_execution";
  if (name === "wave-shipment") return "shipment_intake";
  if (name === "wave-channel-sync") return "channel_sync";
  if (name === "wave-history") return "history_tree";
  return "wave_overview";
});

// 默认全部展开（视觉上 group 不需要展开/收起，但 NMenu 的 group 类型本身就是常展开的）
const expandedKeys = computed<string[]>(() => [
  "section_intake",
  "section_allocation",
  "section_adjustment",
  "section_readiness",
  "section_execution",
  "section_shipment",
  "section_sync",
]);

// menu key → route path
const KEY_TO_PATH: Record<string, string> = {
  wave_overview: "",
  demand_intake: "intake",
  membership_allocation: "allocation/membership",
  demand_mapping: "allocation/demand",
  adjustment_review: "adjustment-review",
  execution_readiness: "readiness",
  supplier_execution: "export",
  shipment_intake: "shipment",
  channel_sync: "channel-sync",
  history_tree: "history",
};

function handleMenuUpdateValue(key: string) {
  if (!waveId.value) return;
  const path = KEY_TO_PATH[key];
  if (path === undefined) return;
  const base = `/waves/${waveId.value}`;
  router.push(path ? `${base}/${path}` : base);
}

const stageTagType = computed(() => {
  switch (props.snapshot?.projectedLifecycleStage) {
    case "awaiting_manual_closure":
      return "error" as const;
    case "syncing_back":
      return "warning" as const;
    case "closed":
      return "success" as const;
    case "execution":
      return "info" as const;
    default:
      return "default" as const;
  }
});
</script>

<template>
  <div class="wave-workspace-sidebar" :class="{ 'is-collapsed': props.collapsed }">
    <div v-if="!props.collapsed" class="wave-sidebar-header">
      <div class="app-kicker">Wave Workspace</div>
      <div class="wave-title" :title="snapshot?.wave?.name">
        {{ snapshot?.wave?.name || "Loading..." }}
      </div>
      <div class="wave-meta">
        <span class="wave-number">{{ snapshot?.wave?.waveNo }}</span>
        <NTag
          v-if="snapshot?.projectedLifecycleStage"
          :type="stageTagType"
          size="tiny"
          round
          :bordered="false"
        >
          {{ snapshot.projectedLifecycleStage }}
        </NTag>
        <NTag
          v-if="snapshot?.basisSummary?.hasRequiredReview"
          type="error"
          size="tiny"
          round
          :bordered="false"
        >
          <NIcon size="10" style="margin-right: 2px;">
            <CheckmarkDoneOutline />
          </NIcon>
          review
        </NTag>
        <NTag
          v-else-if="snapshot?.basisSummary?.hasDriftedBasis"
          type="warning"
          size="tiny"
          round
          :bordered="false"
        >
          drift
        </NTag>
      </div>
    </div>

    <div class="wave-menu-container">
      <NMenu
        :options="menuOptions"
        :value="currentMenuKey"
        :default-expanded-keys="expandedKeys"
        :collapsed="props.collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        @update:value="handleMenuUpdateValue"
        class="wave-sidebar-menu"
        :indent="props.collapsed ? 0 : 20"
      />
    </div>
  </div>
</template>

<style scoped>
.wave-workspace-sidebar {
  display: flex;
  flex-direction: column;
  flex: 1 1 0;
  min-height: 0;
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
  padding: 0 20px 16px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
  margin-bottom: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

:root[data-theme='dark'] .wave-sidebar-header {
  border-bottom-color: rgba(255, 255, 255, 0.05);
}

.app-kicker {
  color: var(--muted);
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.wave-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.wave-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 2px;
}

.wave-number {
  font-size: 11px;
  color: var(--muted);
  font-family: monospace;
}

.wave-menu-container {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px;
}

.wave-workspace-sidebar.is-collapsed .wave-menu-container {
  padding: 0;
  overflow-x: hidden;
}

/* Naive UI does not hide group titles when collapsed — suppress them so the
   64px rail shows icons only (otherwise the labels wrap / overflow). */
:deep(.n-menu--collapsed .n-menu-item-group-title) {
  display: none;
}

:deep(.n-menu-item-content) {
  border-radius: 8px;
  margin-bottom: 2px;
}

:deep(.n-menu-item-content.n-menu-item-content--selected) {
  color: var(--accent);
  font-weight: 600;
}

/* Recolor Naive UI's own inset selection pill instead of stacking a second,
   full-width background on the content element (which produced two
   differently-sized selection layers). */
:deep(.n-menu-item-content.n-menu-item-content--selected::before) {
  background-color: var(--accent-surface) !important;
}

:deep(.n-menu-item-group-title) {
  font-size: 0.7rem !important;
  font-weight: 700 !important;
  letter-spacing: 0.04em;
  color: var(--muted) !important;
  text-transform: uppercase;
  padding-top: 12px;
}
</style>
