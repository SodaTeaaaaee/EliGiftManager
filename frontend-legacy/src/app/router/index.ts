import { createRouter, createWebHashHistory } from "vue-router";

// NOTE: createWebHashHistory is correct for Wails desktop (file:// protocol).
// If the app ever targets web/SSR, switch to createWebHistory.
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      component: () => import("@/app/AppLayout.vue"),
      children: [
        { path: "", redirect: "/dashboard" },

        // ── Action Center ──
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

        // ── Wave Workspace (波次工作区) ──
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
              path: "intake",
              name: "wave-intake",
              component: () =>
                import("@/pages/wave-workspace/WaveIntakeStep.vue"),
            },
            // Initial Allocation 折叠分组：两个独立子项
            {
              path: "allocation/membership",
              name: "wave-membership-allocation",
              component: () =>
                import(
                  "@/pages/membership-allocation/MembershipAllocationPage.vue"
                ),
            },
            {
              path: "allocation/demand",
              name: "wave-demand-mapping",
              component: () =>
                import("@/pages/demand-mapping/DemandMappingPage.vue"),
            },
            // Legacy compatibility — 旧 demandKind 参数路径
            {
              path: "allocation/:demandKind?",
              name: "wave-allocation-legacy",
              component: () =>
                import(
                  "@/pages/membership-allocation/MembershipAllocationPage.vue"
                ),
            },
            {
              path: "demand-mapping/:demandKind?",
              name: "wave-demand-mapping-legacy",
              component: () =>
                import("@/pages/demand-mapping/DemandMappingPage.vue"),
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
              path: "readiness",
              name: "wave-readiness",
              component: () =>
                import("@/pages/wave-workspace/WaveReadinessStep.vue"),
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
            {
              path: "history",
              name: "wave-history",
              component: () =>
                import("@/pages/wave-workspace/WaveHistoryPage.vue"),
            },
          ],
        },

        // ── Demand Inbox (全局需求收件箱) ──
        {
          path: "demand-inbox",
          name: "demand-inbox",
          component: () =>
            import("@/pages/demand-inbox/DemandInboxPage.vue"),
        },

        // ── Profiles (集成配置 master-detail) ──
        {
          path: "profiles",
          name: "profiles",
          component: () =>
            import("@/pages/profile/ProfileManagementPage.vue"),
        },
        {
          path: "profiles/:id",
          name: "profile-detail",
          component: () =>
            import("@/pages/profile/ProfileDetailPage.vue"),
        },

        // ── Customers (客户档案 CRM) ──
        {
          path: "customers",
          name: "customers",
          component: () =>
            import("@/pages/customer/CustomerManagementPage.vue"),
        },
        {
          path: "customers/:id",
          name: "customer-detail",
          component: () =>
            import("@/pages/customer/CustomerDetailPage.vue"),
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
          path: "demand-intake",
          redirect: "/demand-inbox",
        },
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
          redirect: "/customers",
        },
        {
          path: "merge",
          redirect: "/customers",
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
