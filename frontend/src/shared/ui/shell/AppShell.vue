<script setup lang="ts">
/**
 * AppShell — the two-pane desktop shell (plan 2.1/4.3): `SideNav` on the
 * left, a content zone on the right. ONLY the content zone is wrapped by
 * `ContentErrorBoundary` — the nav is a sibling outside the boundary, so a
 * page-level render crash never takes navigation down with it (plan 2.1:
 * "错误边界只替换内容区，导航永远可用").
 *
 * Slot-based so the router view drops straight in:
 *
 *   <AppShell :groups="navGroups" :settings-item="settingsItem">
 *     <template #brand>…</template>
 *     <RouterView />
 *   </AppShell>
 */
import SideNav from './SideNav.vue'
import ContentErrorBoundary from './ContentErrorBoundary.vue'
import type { NavGroupSpec, NavItemSpec } from './types'

defineProps<{
  groups: NavGroupSpec[]
  settingsItem?: NavItemSpec
}>()
</script>

<template>
  <div class="app-shell">
    <SideNav :groups="groups" :settings-item="settingsItem">
      <template v-if="$slots.brand" #brand>
        <slot name="brand" />
      </template>
    </SideNav>
    <main class="app-shell__content">
      <ContentErrorBoundary>
        <slot />
      </ContentErrorBoundary>
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  align-items: stretch;
  height: 100%;
  min-height: 0;
  background: var(--color-bg-app);
}

.app-shell__content {
  flex: 1 1 auto;
  min-width: 0;
  height: 100%;
  overflow-y: auto;
  padding: var(--space-6);
}
</style>
