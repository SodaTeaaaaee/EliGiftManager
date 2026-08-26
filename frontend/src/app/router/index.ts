import { createRouter, createWebHashHistory } from 'vue-router'
import { useRouteProgressStore } from '@/shared/model/route-progress'

declare module 'vue-router' {
  interface RouteMeta {
    navTitleKey?: string
  }
}

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/home/HomePage.vue'),
      meta: { navTitleKey: 'nav.home' },
    },
    {
      path: '/inbox',
      name: 'inbox',
      component: () => import('@/pages/inbox/InboxPage.vue'),
      meta: { navTitleKey: 'nav.inbox' },
    },
    {
      path: '/waves',
      name: 'waves',
      component: () => import('@/pages/waves/WavesPage.vue'),
      meta: { navTitleKey: 'nav.waves' },
    },
    {
      path: '/waves/:id',
      component: () => import('@/pages/waves/workspace/WaveWorkspaceShell.vue'),
      meta: { navTitleKey: 'nav.waves' },
      children: [
        {
          path: '',
          redirect: (to) => ({ path: `/waves/${to.params.id}/rules` }),
        },
        {
          path: 'rules',
          name: 'wave-rules',
          component: () => import('@/pages/waves/workspace/WaveRulesPage.vue'),
          meta: { navTitleKey: 'wave.rules' },
        },
        {
          path: 'results',
          name: 'wave-results',
          component: () => import('@/pages/waves/workspace/WaveResultsPage.vue'),
          meta: { navTitleKey: 'wave.results' },
        },
      ],
    },
    {
      path: '/library',
      redirect: '/library/customers',
    },
    {
      path: '/library',
      component: () => import('@/pages/library/LibraryLayout.vue'),
      meta: { navTitleKey: 'nav.library' },
      children: [
        {
          path: 'customers',
          name: 'library-customers',
          component: () => import('@/pages/library/customers/CustomersPage.vue'),
          meta: { navTitleKey: 'nav.libraryCustomers' },
        },
        {
          path: 'customers/:id',
          name: 'customer-detail',
          component: () => import('@/pages/library/customers/CustomerDetailPage.vue'),
          props: true,
          meta: { navTitleKey: 'nav.libraryCustomers' },
        },
        {
          path: 'products',
          name: 'library-products',
          component: () => import('@/pages/library/products/ProductsPage.vue'),
          meta: { navTitleKey: 'nav.libraryProducts' },
        },
        {
          path: 'templates',
          name: 'library-templates',
          component: () => import('@/pages/library/templates/TemplatesPage.vue'),
          meta: { navTitleKey: 'nav.libraryTemplates' },
        },
      ],
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/pages/settings/SettingsPage.vue'),
      meta: { navTitleKey: 'nav.settings' },
    },
    {
      path: '/design-lab',
      name: 'design-lab',
      component: () => import('@/pages/design-lab/DesignLabPage.vue'),
    },
  ],
})

router.beforeEach(() => {
  useRouteProgressStore().start()
})

router.afterEach(() => {
  useRouteProgressStore().finish()
})

router.onError(() => {
  useRouteProgressStore().finish()
})

export { router }
