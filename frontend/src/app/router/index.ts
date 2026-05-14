import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      component: () => import("@/app/AppLayout.vue"),
      children: [
        { path: "", redirect: "/demand-intake" },
        {
          path: "demand-intake",
          name: "demand-intake",
          component: () => import("@/pages/demand-intake/DemandIntakePage.vue"),
        },
        {
          path: "wave-overview",
          name: "wave-overview",
          component: () => import("@/pages/wave-overview/WaveOverviewPage.vue"),
        },
        {
          path: "waves/:waveId/allocation",
          name: "membership-allocation",
          component: () => import("@/pages/membership-allocation/MembershipAllocationPage.vue"),
        },
        {
          path: "waves/:waveId/demand-mapping",
          name: "demand-mapping",
          component: () => import("@/pages/demand-mapping/DemandMappingPage.vue"),
        },
        {
          path: "waves/:waveId/adjustment-review",
          name: "adjustment-review",
          component: () => import("@/pages/adjustment-review/AdjustmentReviewPage.vue"),
        },
        // TODO(V2): 完整实现 — 添加 AllocationReview, ExportPage 路由
      ],
    },
  ],
});

export { router };
