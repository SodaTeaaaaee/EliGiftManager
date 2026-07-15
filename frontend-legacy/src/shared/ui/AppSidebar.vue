<template>
  <div class="app-sidebar" :class="{ 'is-collapsed': props.collapsed }">
    <div class="sidebar-header">
      <template v-if="!props.collapsed">
        <div class="sidebar-brand">EliGiftManager</div>
        <div class="sidebar-subtitle">Gift Ops Console</div>
      </template>
      <div v-else class="sidebar-brand-mini">EG</div>
    </div>

    <n-menu
      :value="activeKey"
      :options="menuOptions"
      :collapsed="props.collapsed"
      :collapsed-width="64"
      :collapsed-icon-size="20"
      :indent="props.collapsed ? 0 : 32"
      @update:value="onMenuSelect"
    />

    <div class="sidebar-footer">
      <n-menu
        :value="footerKey"
        :options="footerOptions"
        :collapsed="props.collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="20"
        @update:value="onFooterSelect"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NMenu, NIcon, type MenuOption } from "naive-ui";
import {
  GridOutline,
  DownloadOutline,
  LayersOutline,
  CubeOutline,
  SettingsOutline,
  TicketOutline,
  PeopleOutline,
} from "@vicons/ionicons5";
import { h } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useI18n } from "@/shared/i18n";

const props = defineProps<{
  collapsed?: boolean;
}>();

const router = useRouter();
const route = useRoute();
const { t } = useI18n();

// 顶层导航分段（按 V2 文档：行动中心 / 履约工作流 / 主数据）
const menuOptions = computed<MenuOption[]>(() => [
  // ── 行动中心 ──
  {
    type: "group",
    label: t("nav.sectionActionCenter"),
    key: "section-action-center",
    children: [
      {
        label: t("nav.dashboard"),
        key: "dashboard",
        icon: () => h(NIcon, null, { default: () => h(GridOutline) }),
      },
    ],
  },
  // ── 履约工作流 ──
  {
    type: "group",
    label: t("nav.sectionFulfillment"),
    key: "section-fulfillment",
    children: [
      {
        label: t("nav.waves"),
        key: "waves",
        icon: () => h(NIcon, null, { default: () => h(TicketOutline) }),
      },
      {
        label: t("nav.demandInbox"),
        key: "demand-inbox",
        icon: () => h(NIcon, null, { default: () => h(DownloadOutline) }),
      },
    ],
  },
  // ── 主数据 ──
  {
    type: "group",
    label: t("nav.sectionMasterData"),
    key: "section-master-data",
    children: [
      {
        label: t("nav.customers"),
        key: "customers",
        icon: () => h(NIcon, null, { default: () => h(PeopleOutline) }),
      },
      {
        label: t("nav.products"),
        key: "products",
        icon: () => h(NIcon, null, { default: () => h(CubeOutline) }),
      },
      {
        label: t("nav.profiles"),
        key: "profiles",
        icon: () => h(NIcon, null, { default: () => h(LayersOutline) }),
      },
    ],
  },
]);

const footerOptions = computed<MenuOption[]>(() => [
  {
    label: t("nav.settings"),
    key: "settings",
    icon: () => h(NIcon, null, { default: () => h(SettingsOutline) }),
  },
]);

const activeKey = computed(() => {
  const name = route.name;
  if (typeof name !== "string") return null;
  if (name.startsWith("wave-")) return "waves";
  if (name === "customer-detail") return "customers";
  if (name === "profile-detail") return "profiles";
  if (name === "settings") return null;
  return name;
});

const footerKey = computed(() => (route.name === "settings" ? "settings" : null));

function onMenuSelect(key: string) {
  router.push({ name: key });
}

function onFooterSelect(key: string) {
  router.push({ name: key });
}
</script>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 24px 16px 20px;
  gap: 16px;
  background: var(--surface-strong);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: 1px solid rgba(255, 255, 255, 0.08);
}

.app-sidebar.is-collapsed {
  padding-left: 0;
  padding-right: 0;
}

.app-sidebar.is-collapsed .sidebar-header {
  padding: 0;
  text-align: center;
}

/* Naive UI does not hide group titles when collapsed — suppress them so the
   64px rail shows icons only (otherwise the labels wrap / overflow). */
:deep(.n-menu--collapsed .n-menu-item-group-title) {
  display: none;
}

:root[data-theme='dark'] .app-sidebar {
  background: rgba(24, 24, 27, 0.6);
  border-right: 1px solid rgba(255, 255, 255, 0.04);
}

.sidebar-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 0 8px;
}

.sidebar-brand {
  color: var(--text);
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  background: linear-gradient(135deg, var(--text) 0%, var(--muted) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.sidebar-subtitle {
  color: var(--muted);
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.sidebar-brand-mini {
  color: var(--text);
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  text-align: center;
}

.sidebar-footer {
  margin-top: auto;
}
</style>
