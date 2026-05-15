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
            {
              path: "export",
              name: "wave-export",
              component: () =>
                import("@/pages/wave-workspace/WaveExportStep.vue"),
            },
            {
              path: "shipment",
              name: "wave-shipment",
              component: () =>
                import("@/pages/wave-workspace/WaveShipmentStep.vue"),
            },
            {
              path: "channel-sync",
              name: "wave-channel-sync",
              component: () =>
                import("@/pages/wave-workspace/WaveChannelSyncStep.vue"),
            },
          ],
        },
        {
          path: "profiles",
          name: "profiles",
          component: () =>
            import("@/pages/profile/ProfileManagementPage.vue"),
        },
        {
          path: "products",
          name: "products",
          component: () =>
            import("@/pages/product/ProductManagementPage.vue"),
        },
        // Legacy wave-overview route — redirect to dashboard (consolidated into workspace wizard)
        {
          path: "wave-overview",
          redirect: "/dashboard",
        },

      ],
    },
  ],
});

export { router };
