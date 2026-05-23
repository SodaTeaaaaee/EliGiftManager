<template>
  <div class="app-sidebar">
    <div class="sidebar-header">
      <div class="sidebar-brand">EliGiftManager</div>
      <div class="sidebar-subtitle">Gift Ops Console</div>
    </div>

    <n-menu
      :value="activeKey"
      :options="menuOptions"
      @update:value="onMenuSelect"
    />

    <div class="sidebar-footer">
      <n-menu
        :value="footerKey"
        :options="footerOptions"
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

const router = useRouter();
const route = useRoute();
const { t } = useI18n();

const menuOptions = computed<MenuOption[]>(() => [
  {
    label: t("nav.dashboard"),
    key: "dashboard",
    icon: () => h(NIcon, null, { default: () => h(GridOutline) }),
  },
  {
    label: t("nav.waves"),
    key: "waves",
    icon: () => h(NIcon, null, { default: () => h(TicketOutline) }),
  },
  {
    label: t("nav.demandIntake"),
    key: "demand-intake",
    icon: () => h(NIcon, null, { default: () => h(DownloadOutline) }),
  },
  {
    label: t("nav.profiles"),
    key: "profiles",
    icon: () => h(NIcon, null, { default: () => h(LayersOutline) }),
  },
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
  gap: 24px;
  background: var(--surface-strong);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-right: 1px solid rgba(255, 255, 255, 0.08);
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

.sidebar-footer {
  margin-top: auto;
}
</style>
