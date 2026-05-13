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
        // TODO(V2): 完整实现 — 添加 AllocationReview, ExportPage 路由
      ],
    },
  ],
});

export { router };
