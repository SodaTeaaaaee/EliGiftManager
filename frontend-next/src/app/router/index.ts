import { createRouter, createWebHashHistory } from "vue-router";
import { useRouteProgressStore } from "@/shared/model/route-progress";

// Module augmentation: `meta.navTitleKey` is an i18n message key (not a
// resolved string) read by `PlaceholderPage.vue` to render its title.
declare module "vue-router" {
  interface RouteMeta {
    navTitleKey?: string;
  }
}

// NOTE: createWebHashHistory is required for Wails desktop (file:// protocol).
// If the app ever targets web/SSR, switch to createWebHistory.
//
// `meta.navTitleKey` on the placeholder routes below is an i18n message KEY
// (not a resolved string) consumed by `PlaceholderPage.vue` — every one of
// the plan 2.1 top-level nav destinations gets a real, distinctly-routable
// page so `SideNav`'s active-item highlighting is correct even before the
// real page for that section is built. Swapping a placeholder for its real
// page later is a one-line `component:` change here, nothing else moves.
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("@/pages/home/HomePage.vue"),
    },
    {
      path: "/waves",
      name: "waves",
      component: () => import("@/pages/waves/WavesPage.vue"),
      meta: { navTitleKey: "nav.waves" },
    },
    {
      // Parent shell route (plan 3.3): `WaveWorkspaceShell` owns the header +
      // `WorkspaceNav` rail and provides the single `useWaveWorkspace()`
      // context; each child renders inside its `<RouterView/>`. The
      // empty-path child KEEPS the name `wave-workspace` so P1's
      // `buildWaveFilterLink({name:'wave-workspace',params:{id},query})`
      // deep link keeps resolving unchanged.
      path: "/waves/:id",
      component: () => import("@/pages/waves/workspace/WaveWorkspaceShell.vue"),
      meta: { navTitleKey: "nav.waves" },
      children: [
        {
          path: "",
          name: "wave-workspace",
          component: () => import("@/pages/waves/workspace/tabs/WaveOverviewTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "intake",
          name: "wave-workspace-intake",
          component: () => import("@/pages/waves/workspace/tabs/WaveIntakeTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "allocation",
          name: "wave-workspace-allocation",
          component: () => import("@/pages/waves/workspace/tabs/WaveAllocationTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "lines",
          name: "wave-workspace-lines",
          component: () => import("@/pages/waves/workspace/tabs/WaveLinesTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "readiness",
          name: "wave-workspace-readiness",
          component: () => import("@/pages/waves/workspace/tabs/WaveLinesTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "factory",
          name: "wave-workspace-factory",
          component: () => import("@/pages/waves/workspace/tabs/WaveFactoryTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "shipments",
          name: "wave-workspace-shipments",
          component: () => import("@/pages/waves/workspace/tabs/WaveShipmentsTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
        {
          path: "closure",
          name: "wave-workspace-closure",
          component: () => import("@/pages/waves/workspace/tabs/WaveClosureTab.vue"),
          meta: { navTitleKey: "nav.waves" },
        },
      ],
    },
    {
      path: "/inbox",
      name: "inbox",
      component: () => import("@/pages/inbox/InboxPage.vue"),
      meta: { navTitleKey: "nav.inbox" },
    },
    {
      path: "/customers",
      name: "customers",
      component: () => import("@/pages/customers/CustomersPage.vue"),
      meta: { navTitleKey: "nav.customers" },
    },
    {
      // Unified customer detail: single edit surface (profile fields,
      // identities, addresses, fulfillment history, merge preview). `id` is
      // passed as a prop (see `props: true` below) rather than read from
      // `route.params` inside the component, matching Vue Router's typed
      // prop-passing convention.
      path: "/customers/:id",
      name: "customer-detail",
      component: () => import("@/pages/customers/CustomerDetailPage.vue"),
      props: true,
      meta: { navTitleKey: "nav.customers" },
    },
    {
      path: "/products",
      name: "products",
      component: () => import("@/pages/products/ProductsPage.vue"),
      meta: { navTitleKey: "nav.products" },
    },
    {
      path: "/integrations",
      name: "integrations",
      component: () => import("@/pages/integrations/IntegrationsPage.vue"),
      meta: { navTitleKey: "nav.integrations" },
    },
    {
      path: "/settings",
      name: "settings",
      component: () => import("@/pages/settings/SettingsPage.vue"),
      meta: { navTitleKey: "nav.settings" },
    },
    {
      path: "/design-lab",
      name: "design-lab",
      component: () => import("@/pages/design-lab/DesignLabPage.vue"),
    },
  ],
});

router.beforeEach(() => {
  useRouteProgressStore().start();
});

router.afterEach(() => {
  useRouteProgressStore().finish();
});

router.onError(() => {
  useRouteProgressStore().finish();
});

export { router };
