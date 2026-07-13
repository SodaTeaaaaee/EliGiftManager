<script setup lang="ts">
/**
 * App root — NConfigProvider (theme/tokens bridge) wraps FeedbackProvider
 * (the app's only toast/receipt path) wraps AppShell (nav + content zone,
 * itself the only place a content-crash error boundary lives). `abstract`
 * on NConfigProvider skips its default wrapper `<div>` so AppShell's own
 * `height: 100%` flex chain reaches all the way up to `#app` uninterrupted
 * (see `shared/styles/reset.css`'s `#app { height: 100% }`).
 */
import { computed, onMounted } from "vue";
import { RouterView } from "vue-router";
import { NConfigProvider } from "naive-ui";
import {
  GridOutline,
  TicketOutline,
  DownloadOutline,
  PeopleOutline,
  CubeOutline,
  GitNetworkOutline,
  SettingsOutline,
  FlaskOutline,
} from "@vicons/ionicons5";
import { AppShell, useActionCenterBadgesStore, type NavGroupSpec, type NavItemSpec } from "@/shared/ui/shell";
import { FeedbackProvider, DisconnectedBanner, TopProgressBar } from "@/shared/ui/feedback";
import { useNaiveTheme } from "@/shared/theme/naive-bridge";
import { useGlobalViewHotkeys } from "@/shared/lib/view-hotkeys";

const { theme, themeOverrides } = useNaiveTheme();
useGlobalViewHotkeys();

// Plan 3.1's nav badges (ActionCenterSummaryDTO.navBadges) are fetched once
// on mount and kept in a small cross-cutting store (see
// shared/ui/shell/action-center-badges.ts) so any later page can trigger a
// fresh count (e.g. after closing a wave) without prop-drilling through
// AppShell. An empty/failed summary yields an empty countsByNavKey map, so
// every item below simply renders with no badge — defensive by construction.
const actionCenterBadges = useActionCenterBadgesStore();
onMounted(() => {
  void actionCenterBadges.refresh();
});

function badgeFor(navKey: string): NavItemSpec["badge"] {
  const count = actionCenterBadges.countsByNavKey[navKey] ?? 0;
  return count > 0 ? { count, tone: "warning" } : undefined;
}

// Plan 2.1's 6 top-level sections + a dev-only Design Lab entry. Real
// destinations that don't have a rebuilt page yet route to a distinct
// placeholder route (see app/router/index.ts) rather than all piling onto
// "/", so SideNav's active-item highlighting stays correct.
const navGroups = computed<NavGroupSpec[]>(() => [
  {
    key: "primary",
    items: [
      { key: "home", labelKey: "nav.home", icon: GridOutline, to: { name: "home" }, badge: badgeFor("home") },
      { key: "waves", labelKey: "nav.waves", icon: TicketOutline, to: { name: "waves" }, badge: badgeFor("waves") },
      { key: "inbox", labelKey: "nav.inbox", icon: DownloadOutline, to: { name: "inbox" }, badge: badgeFor("inbox") },
      {
        key: "customers",
        labelKey: "nav.customers",
        icon: PeopleOutline,
        to: { name: "customers" },
        badge: badgeFor("customers"),
      },
      {
        key: "products",
        labelKey: "nav.products",
        icon: CubeOutline,
        to: { name: "products" },
        badge: badgeFor("products"),
      },
      {
        key: "integrations",
        labelKey: "nav.integrations",
        icon: GitNetworkOutline,
        to: { name: "integrations" },
        badge: badgeFor("integrations"),
      },
    ],
  },
  {
    key: "dev",
    labelKey: "nav.devSectionLabel",
    items: [{ key: "design-lab", labelKey: "designLab.title", icon: FlaskOutline, to: { name: "design-lab" } }],
  },
]);

const settingsItem: NavItemSpec = {
  key: "settings",
  labelKey: "nav.settings",
  icon: SettingsOutline,
  to: { name: "settings" },
};
</script>

<template>
  <NConfigProvider abstract :theme="theme" :theme-overrides="themeOverrides">
    <TopProgressBar />
    <FeedbackProvider>
      <AppShell :groups="navGroups" :settings-item="settingsItem">
        <DisconnectedBanner />
        <RouterView />
      </AppShell>
    </FeedbackProvider>
  </NConfigProvider>
</template>
