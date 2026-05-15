import { describe, it, expect, vi, beforeEach } from "vitest";
import { mount, flushPromises } from "@vue/test-utils";
import { createRouter, createWebHashHistory } from "vue-router";
import { createPinia } from "pinia";
import { defineComponent, h, nextTick } from "vue";

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
        path: "demand-intake",
        name: "demand-intake",
        component: () => import("@/pages/demand-intake/DemandIntakePage.vue"),
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
            path: "demand-mapping",
            name: "wave-demand-mapping",
            component: () => import("@/pages/demand-mapping/DemandMappingPage.vue"),
          },
          {
            path: "allocation",
            name: "wave-allocation",
            component: () => import("@/pages/membership-allocation/MembershipAllocationPage.vue"),
          },
          {
            path: "adjustment-review",
            name: "wave-adjustment-review",
            component: () => import("@/pages/adjustment-review/AdjustmentReviewPage.vue"),
          },
        ],
      },
      {
        path: "profiles",
        name: "profiles",
        component: () => import("@/pages/profile/ProfileManagementPage.vue"),
      },
    ],
  },
];

function createTestRouter(initialRoute: string) {
  const router = createRouter({
    history: createWebHashHistory(),
    routes,
  });
  return router;
}

async function mountAtRoute(path: string) {
  const router = createTestRouter(path);
  const pinia = createPinia();

  const wrapper = mount(
    defineComponent({
      template: `<router-view />`,
    }),
    {
      global: {
        plugins: [router, pinia],
        stubs: {
          NConfigProvider: { template: "<div><slot /></div>" },
          NGlobalStyle: { template: "<div />" },
          NMessageProvider: { template: "<div><slot /></div>" },
          NDialogProvider: { template: "<div><slot /></div>" },
          NResult: { template: "<div class='n-result'><slot /></div>" },
          NButton: { template: "<button><slot /></button>" },
          NDataTable: { template: "<div class='n-data-table' />" },
          NSpace: { template: "<div><slot /></div>" },
          NTag: { template: "<span><slot /></span>" },
          NModal: { template: "<div><slot /></div>" },
          NPopconfirm: { template: "<div><slot /></div>" },
          NSelect: { template: "<div />" },
          NInput: { template: "<div />" },
          NInputNumber: { template: "<div />" },
          NForm: { template: "<div><slot /></div>" },
          NFormItem: { template: "<div><slot /></div>" },
          NCard: { template: "<div><slot /></div>" },
          NGrid: { template: "<div><slot /></div>" },
          NGi: { template: "<div><slot /></div>" },
          NStatistic: { template: "<div />" },
          NAlert: { template: "<div><slot /></div>" },
          NCollapse: { template: "<div><slot /></div>" },
          NCollapseItem: { template: "<div><slot /></div>" },
          NList: { template: "<div><slot /></div>" },
          NListItem: { template: "<div><slot /></div>" },
          NSteps: { template: "<div><slot /></div>" },
          NStep: { template: "<div><slot /></div>" },
          NSwitch: { template: "<div />" },
          NRadioGroup: { template: "<div><slot /></div>" },
          NRadio: { template: "<div><slot /></div>" },
          NCheckbox: { template: "<div />" },
          NDivider: { template: "<div />" },
          NIcon: { template: "<span />" },
          NTooltip: { template: "<div><slot /></div>" },
          NEmpty: { template: "<div />" },
          NSpin: { template: "<div><slot /></div>" },
          NPageHeader: { template: "<div><slot /></div>" },
          NBreadcrumb: { template: "<div><slot /></div>" },
          NBreadcrumbItem: { template: "<div><slot /></div>" },
          ContextMenu: { template: "<div />" },
        },
      },
    }
  );

  await router.push(path);
  await router.isReady();
  await flushPromises();
  await nextTick();

  return { wrapper, router };
}

describe("Route smoke tests — mount without crash", () => {
  const routeCases = [
    { path: "/dashboard", name: "Dashboard" },
    { path: "/demand-intake", name: "DemandIntake" },
    { path: "/profiles", name: "Profiles" },
    { path: "/waves/1", name: "WaveOverview" },
    { path: "/waves/1/demand-mapping", name: "WaveDemandMapping" },
    { path: "/waves/1/allocation", name: "WaveAllocation" },
    { path: "/waves/1/adjustment-review", name: "WaveAdjustmentReview" },
  ];

  for (const { path, name } of routeCases) {
    it(`${name} (${path}) mounts without throwing`, async () => {
      const { wrapper } = await mountAtRoute(path);
      expect(wrapper.html()).toBeTruthy();
      expect(wrapper.find(".n-result").exists()).toBe(false);
      wrapper.unmount();
    });
  }
});

describe("Route resilience — bridge rejection does not crash", () => {
  it("WaveOverview handles bridge rejection gracefully", async () => {
    const bridge = await import("@/shared/lib/wails/app.ts");
    vi.mocked(bridge.getWaveOverview).mockRejectedValueOnce(
      new Error("Wails backend not connected")
    );

    const { wrapper } = await mountAtRoute("/waves/1");
    expect(wrapper.html()).toBeTruthy();
    wrapper.unmount();
  });

  it("DemandIntake handles bridge rejection gracefully", async () => {
    const bridge = await import("@/shared/lib/wails/app.ts");
    vi.mocked(bridge.listProfiles).mockRejectedValueOnce(
      new Error("Wails backend not connected")
    );

    const { wrapper } = await mountAtRoute("/demand-intake");
    expect(wrapper.html()).toBeTruthy();
    wrapper.unmount();
  });
});
