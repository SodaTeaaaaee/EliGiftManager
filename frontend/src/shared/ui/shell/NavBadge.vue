<script setup lang="ts">
/**
 * NavBadge — the small count pill attached to a `SideNav`/`WorkspaceNav`
 * item (plan 2.1: "侧边栏项带实时工作量徽标… 导航本身就是状态板"). Caps the
 * display at "99+" so a runaway backlog count never blows out the fixed-width
 * icon rail. Purely decorative from an accessibility standpoint — the count
 * is folded into the parent nav item's `aria-label` — so this renders
 * `aria-hidden`.
 */
import { computed } from 'vue'
import type { StatusTone } from '@/shared/i18n/glossary'

const props = withDefaults(
  defineProps<{
    count: number
    tone?: StatusTone
  }>(),
  {
    tone: 'neutral',
  },
)

const display = computed(() => (props.count > 99 ? '99+' : String(Math.max(0, Math.trunc(props.count)))))
</script>

<template>
  <span class="nav-badge" :class="`nav-badge--${tone}`" aria-hidden="true">{{ display }}</span>
</template>

<style scoped>
.nav-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 var(--space-1);
  border-radius: var(--statusbadge-radius);
  border: 1px solid transparent;
  font-family: var(--font-body);
  font-size: 0.6875rem;
  font-weight: var(--font-weight-semibold);
  line-height: 1;
  white-space: nowrap;
}

.nav-badge--success {
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-color: var(--status-success-border);
}
.nav-badge--warning {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}
.nav-badge--error {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-color: var(--status-error-border);
}
.nav-badge--info {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border-color: var(--status-info-border);
}
.nav-badge--progress {
  color: var(--status-progress-fg);
  background: var(--status-progress-bg);
  border-color: var(--status-progress-border);
}
.nav-badge--neutral {
  color: var(--status-neutral-fg);
  background: var(--status-neutral-bg);
  border-color: var(--status-neutral-border);
}
</style>
