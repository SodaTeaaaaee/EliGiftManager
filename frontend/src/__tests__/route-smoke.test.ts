import { describe, it, expect, vi } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { createRouter, createWebHashHistory } from "vue-router";
import { createPinia } from "pinia";
import { nextTick } from "vue";
import App from "@/app/App.vue";

const routes = [
  {
    path: "/",
    component: () => import("@/app/AppLayout.vue"),
    children: [
      { path: "", redirect: "/dashboard" },
      {
        path: "dashboard",
        name: "dashboard",
        component: () => import("@/pages/dashboard/DashboardPage.vue"),
      },
      {
        path: "waves",
        name: "waves",
        component: () => import("@/pages/waves/WavesPage.vue"),
      },
      {
        path: "demand-inbox",
        name: "demand-inbox",
        component: () => import("@/pages/demand-inbox/DemandInboxPage.vue"),
      },
      {
        path: "waves/:waveId",
        component: () => import("@/pages/wave-workspace/WaveWorkspaceLayout.vue"),
        children: [
          {
            path: "",
            name: "wave-overview-step",
            component: () => import("@/pages/wave-workspace/WaveOverviewStep.vue"),
          },
          {
            path: "intake",
            name: "wave-intake",
            component: () => import("@/pages/wave-workspace/WaveIntakeStep.vue"),
          },
          {
            path: "allocation/membership",
            name: "wave-membership-allocation",
            component: () =>
              import("@/pages/membership-allocation/MembershipAllocationPage.vue"),
          },
          {
            path: "allocation/demand",
            name: "wave-demand-mapping",
            component: () => import("@/pages/demand-mapping/DemandMappingPage.vue"),
          },
          {
            path: "adjustment-review",
            name: "wave-adjustment-review",
            component: () =>
              import("@/pages/adjustment-review/AdjustmentReviewPage.vue"),
          },
          {
            path: "readiness",
            name: "wave-readiness",
            component: () => import("@/pages/wave-workspace/WaveReadinessStep.vue"),
          },
          {
            path: "history",
            name: "wave-history",
            component: () => import("@/pages/wave-workspace/WaveHistoryPage.vue"),
          },
        ],
      },
      {
        path: "profiles",
        name: "profiles",
        component: () => import("@/pages/profile/ProfileManagementPage.vue"),
      },
      {
        path: "profiles/:id",
        name: "profile-detail",
        component: () => import("@/pages/profile/ProfileDetailPage.vue"),
      },
      {
        path: "customers",
        name: "customers",
        component: () => import("@/pages/customer/CustomerManagementPage.vue"),
      },
      {
        path: "customers/:id",
        name: "customer-detail",
        component: () => import("@/pages/customer/CustomerDetailPage.vue"),
      },
      {
        path: "products",
        name: "products",
        component: () => import("@/pages/product/ProductManagementPage.vue"),
      },
      {
        path: "settings",
        name: "settings",
        component: () => import("@/pages/settings/SettingsPage.vue"),
      },
    ],
  },
];

function createTestRouter() {
  const router = createRouter({
    history: createWebHashHistory(),
    routes,
  });
  return router;
}

async function mountAtRoute(path: string) {
  const router = createTestRouter();
  const pinia = createPinia();

  const wrapper = mount(App, {
    global: {
      plugins: [router, pinia],
      stubs: {
        NDataTable: { template: "<div class='n-data-table' />" },
        NButton: { template: "<button><slot /></button>" },
        NCard: { template: "<div><slot name='header' /><slot /><slot name='header-extra' /></div>" },
        NAlert: { template: "<div><slot /></div>" },
        NTag: { template: "<span><slot /></span>" },
        NSwitch: { template: "<button />" },
        NRadioGroup: { template: "<div><slot /></div>" },
        NDrawer: { template: "<div><slot /></div>" },
        NDrawerContent: { template: "<div><slot /></div>" },
        NModal: { template: "<div><slot /></div>" },
        NPopconfirm: { template: "<div><slot name='trigger' /><slot /></div>" },
        NPopover: { template: "<div><slot name='trigger' /><slot /></div>" },
        NInput: { template: "<input />" },
        NInputNumber: { template: "<input />" },
        NSelect: { template: "<div />" },
        NForm: { template: "<form><slot /></form>" },
        NFormItem: { template: "<div><slot /></div>" },
        NCollapse: { template: "<div><slot /></div>" },
        NCollapseItem: { template: "<div><slot /></div>" },
        NList: { template: "<div><slot /></div>" },
        NListItem: { template: "<div><slot /></div>" },
        NRadioButton: { template: "<button><slot /></button>" },
        NTabs: { template: "<div><slot /></div>" },
        NTabPane: { template: "<div><slot /></div>" },
        NMenu: { template: "<div />" },
        NLayout: { template: "<div><slot /></div>" },
        NLayoutSider: { template: "<div><slot /></div>" },
        NLayoutContent: { template: "<div><slot /></div>" },
        NSpin: { template: "<div><slot /></div>" },
        NProgress: { template: "<div />" },
        NEmpty: { template: "<div><slot name='extra' /></div>" },
        NSpace: { template: "<div><slot /></div>" },
        NGrid: { template: "<div><slot /></div>" },
        NGridItem: { template: "<div><slot /></div>" },
        NIcon: { template: "<i><slot /></i>" },
        NDivider: { template: "<hr />" },
        NAvatar: { template: "<span><slot /></span>" },
      },
    },
  });

  await router.push(path);
  await router.isReady();
  await flushPromises();
  await nextTick();

  return { wrapper, router };
}

describe("Route smoke tests — mount without crash", () => {
  const routeCases = [
    { path: "/dashboard", name: "Dashboard" },
    { path: "/waves", name: "Waves" },
    { path: "/demand-inbox", name: "DemandInbox" },
    { path: "/profiles", name: "Profiles" },
    { path: "/profiles/1", name: "ProfileDetail" },
    { path: "/customers", name: "Customers" },
    { path: "/customers/1", name: "CustomerDetail" },
    { path: "/products", name: "Products" },
    { path: "/settings", name: "Settings" },
    { path: "/waves/1", name: "WaveOverview" },
    { path: "/waves/1/intake", name: "WaveIntake" },
    { path: "/waves/1/allocation/membership", name: "WaveMembershipAllocation" },
    { path: "/waves/1/allocation/demand", name: "WaveDemandMapping" },
    { path: "/waves/1/adjustment-review", name: "WaveAdjustmentReview" },
    { path: "/waves/1/readiness", name: "WaveReadiness" },
    { path: "/waves/1/history", name: "WaveHistory" },
  ];

  for (const { path, name } of routeCases) {
    it(
      `${name} (${path}) mounts without throwing`,
      async () => {
        const { wrapper } = await mountAtRoute(path);
        expect(wrapper.html()).toBeTruthy();
        expect(wrapper.find(".n-result").exists()).toBe(false);
        wrapper.unmount();
      },
      // Cold-start of the first mounted route can exceed the default 5s,
      // especially in CI; bump the per-test timeout to be safe.
      30000,
    );
  }
});

describe("Route resilience — bridge rejection does not crash", () => {
  it("Dashboard handles bridge rejection gracefully", async () => {
    const bridge = await import("@/shared/lib/wails/app.ts");
    vi.mocked(bridge.listWaveDashboardRows).mockRejectedValueOnce(
      new Error("Wails backend not connected")
    );

    const { wrapper } = await mountAtRoute("/dashboard");
    expect(wrapper.html()).toBeTruthy();
    wrapper.unmount();
  });

  it("WaveOverview handles bridge rejection gracefully", async () => {
    const bridge = await import("@/shared/lib/wails/app.ts");
    vi.mocked(bridge.getWaveOverview).mockRejectedValueOnce(
      new Error("Wails backend not connected")
    );

    const { wrapper } = await mountAtRoute("/waves/1");
    expect(wrapper.html()).toBeTruthy();
    wrapper.unmount();
  });

  it("DemandInbox handles bridge rejection gracefully", async () => {
    const bridge = await import("@/shared/lib/wails/app.ts");
    vi.mocked(bridge.listProfiles).mockRejectedValueOnce(
      new Error("Wails backend not connected")
    );

    const { wrapper } = await mountAtRoute("/demand-inbox");
    expect(wrapper.html()).toBeTruthy();
    wrapper.unmount();
  });
});
