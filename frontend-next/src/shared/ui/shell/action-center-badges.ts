/**
 * Cross-cutting nav-badge state, sourced from `ActionCenterController.GetActionCenterSummary`
 * (plan 3.1's `navBadges` — "侧边栏项带实时工作量徽标"). A tiny Pinia store
 * (mirrors `nav-state.ts`'s pattern) rather than a component-local `ref` on
 * `App.vue`, so any page that changes badge-relevant state (e.g. the task
 * center's manual refresh button, or closing a wave) can trigger `refresh()`
 * without prop-drilling through `AppShell`.
 *
 * `navKey` values come straight off the backend DTO and are matched 1:1
 * against `App.vue`'s nav item `key`s (`home` / `waves` / `inbox` /
 * `customers` / `products` / `integrations`) — no translation layer needed.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getActionCenterSummary } from '@/shared/api/bridge'

export const useActionCenterBadgesStore = defineStore('shell-action-center-badges', () => {
  const countsByNavKey = ref<Record<string, number>>({})
  const loaded = ref(false)

  async function refresh(): Promise<void> {
    try {
      const summary = await getActionCenterSummary()
      const next: Record<string, number> = {}
      for (const badge of summary.navBadges) {
        next[badge.navKey] = badge.count
      }
      countsByNavKey.value = next
      loaded.value = true
    } catch {
      // Defensive: a transient bridge failure leaves prior counts in place
      // rather than blanking every nav badge.
    }
  }

  return { countsByNavKey, loaded, refresh }
})
