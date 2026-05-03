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
          path: 'waves',
          name: 'waves',
          redirect: { name: 'waves-welcome' },
          component: () => import('@/pages/waves/ui/DispatchTaskShell.vue'),
          children: [
            {
              path: '',
              name: 'waves-welcome',
              component: () => import('@/pages/waves/ui/WaveWelcomePage.vue'),
            },
            {
              path: ':waveId/step/1',
              name: 'waves-step-import',
              component: () => import('@/pages/waves/ui/WaveImportStep.vue'),
            },
            {
              path: ':waveId/step/2',
              name: 'waves-step-tags',
              component: () => import('@/pages/waves/ui/WaveTagStep.vue'),
            },
            {
              path: ':waveId/step/3',
              name: 'waves-step-preview',
              component: () => import('@/pages/waves/ui/WavePreviewStep.vue'),
            },
            {
              path: ':waveId/step/4',
              name: 'waves-step-export',
              component: () => import('@/pages/waves/ui/WaveExportStep.vue'),
            },
          ],
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
