<template>
  <div class="app-sidebar">
    <div class="sidebar-header">
      <div class="app-kicker">{{ t("app.workspace") }}</div>
      <div class="sidebar-title">{{ t("app.name") }}</div>
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
import { GridOutline, DownloadOutline, LayersOutline, CubeOutline, SettingsOutline, TicketOutline } from "@vicons/ionicons5";
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
  padding: 20px 14px 16px;
  gap: 18px;
  background:
    radial-gradient(circle at top right, rgba(37, 99, 235, 0.16), transparent 32%),
    linear-gradient(180deg, var(--surface-strong) 0%, var(--surface-muted) 100%);
}

.sidebar-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 6px 10px 10px;
}

.sidebar-title {
  color: var(--text);
  font-size: 1rem;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.sidebar-footer {
  margin-top: auto;
}
</style>
