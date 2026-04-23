import { createRouter, createWebHashHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/app/AppLayout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/pages/dashboard/ui/DashboardPage.vue'),
        },
        {
          path: 'orders',
          name: 'orders',
          component: () => import('@/pages/orders/ui/OrderCenterPage.vue'),
        },
        {
          path: 'members',
          name: 'members',
          component: () => import('@/pages/members/ui/MemberCrmPage.vue'),
        },
        {
          path: 'products',
          name: 'products',
          component: () => import('@/pages/products/ui/ProductLibraryPage.vue'),
        },
        {
          path: 'templates',
          name: 'templates',
          component: () => import('@/pages/templates/ui/TemplatesPage.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/pages/settings/ui/SettingsPage.vue'),
        },
      ],
    },
  ],
})
