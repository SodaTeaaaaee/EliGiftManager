import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
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
          component: () =>
            import("@/pages/demand-intake/DemandIntakePage.vue"),
        },
        // Wave workspace — nested layout with step wizard.
        // Defined before the legacy flat routes so the wizard is always shown
        // for /waves/:waveId/* paths.
        {
          path: "waves/:waveId",
          component: () =>
            import("@/pages/wave-workspace/WaveWorkspaceLayout.vue"),
          children: [
            {
              path: "",
              name: "wave-overview-step",
              component: () =>
                import("@/pages/wave-workspace/WaveOverviewStep.vue"),
            },
            {
              path: "demand-mapping",
              name: "wave-demand-mapping",
              component: () =>
                import("@/pages/demand-mapping/DemandMappingPage.vue"),
            },
            {
              path: "allocation",
              name: "wave-allocation",
              component: () =>
                import(
                  "@/pages/membership-allocation/MembershipAllocationPage.vue"
                ),
            },
            {
              path: "adjustment-review",
              name: "wave-adjustment-review",
              component: () =>
                import(
                  "@/pages/adjustment-review/AdjustmentReviewPage.vue"
                ),
            },
          ],
        },
        // Legacy standalone routes — redirects to wizard workspace.
        {
          path: "wave-overview",
          name: "wave-overview",
          component: () =>
            import("@/pages/wave-overview/WaveOverviewPage.vue"),
        },
        {
          path: "waves/:waveId/allocation",
          redirect: (to: any) => ({ name: "wave-allocation", params: to.params }),
        },
        {
          path: "waves/:waveId/demand-mapping",
          redirect: (to: any) => ({ name: "wave-demand-mapping", params: to.params }),
        },
        {
          path: "waves/:waveId/adjustment-review",
          redirect: (to: any) => ({ name: "wave-adjustment-review", params: to.params }),
        },
        // TODO(V2): 完整实现 — 添加 AllocationReview, ExportPage 路由
      ],
    },
  ],
});

export { router };
