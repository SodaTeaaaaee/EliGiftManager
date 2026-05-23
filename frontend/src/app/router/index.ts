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

        // ── Waves (波次列表) ──
        {
          path: "waves",
          name: "waves",
          component: () => import("@/pages/waves/WavesPage.vue"),
        },

        // ── Wave Workspace (波次工作区 — 7步向导) ──
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

        // ── Demand Intake (需求录入) ──
        {
          path: "demand-intake",
          name: "demand-intake",
          component: () =>
            import("@/pages/demand-intake/DemandIntakePage.vue"),
        },

        // ── Profiles (集成配置 — 整合 Template / Address / Merge) ──
        {
          path: "profiles",
          name: "profiles",
          component: () =>
            import("@/pages/profile/ProfileManagementPage.vue"),
        },

        // ── Products (商品库) ──
        {
          path: "products",
          name: "products",
          component: () =>
            import("@/pages/product/ProductManagementPage.vue"),
        },

        // ── Settings (设置) ──
        {
          path: "settings",
          name: "settings",
          component: () =>
            import("@/pages/settings/SettingsPage.vue"),
        },

        // Legacy redirects — keep old routes functional
        {
          path: "wave-overview",
          redirect: "/dashboard",
        },
        {
          path: "templates",
          redirect: "/profiles",
        },
        {
          path: "templates/bindings",
          redirect: "/profiles",
        },
        {
          path: "templates/csv-import",
          redirect: "/profiles",
        },
        {
          path: "addresses",
          redirect: "/profiles",
        },
        {
          path: "merge",
          redirect: "/profiles",
        },
      ],
    },
  ],
});

router.onError((error) => {
  // Chunk load failures from dynamic imports (network issues, stale deployment, etc.)
  const msg = error?.message ?? '';
  const isChunkError =
    msg.includes('Failed to fetch dynamically imported module') ||
    msg.includes('Importing a module script failed') ||
    msg.includes('Unable to preload CSS') ||
    msg.includes('error loading dynamically imported module');
  if (isChunkError) {
    window.dispatchEvent(new CustomEvent('router-chunk-error', { detail: error }));
  }
});

export { router };
