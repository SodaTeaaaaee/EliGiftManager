<script setup lang="ts">
/**
 * WorkspaceNav — SKELETON for the wave-workspace second-level nav (plan
 * 3.3). This kit ships the visual language + prop contract only; wiring it
 * to `WaveWorkspaceSnapshotDTO.stepStates` / `ValidateStepAccess` and the
 * real step routes is P2's job (plan section 7).
 *
 * Same look as `SideNav` (same tokens, same item chrome) but always
 * full-width and never collapses to a rail — this nav's job is orientation
 * within one wave, not global wayfinding, so there is no icon-only mode.
 * `tone` on an item drives a small status dot instead of the old tree's
 * fake padlock icon (`WaveWorkspaceSidebar.vue`'s `LockClosedOutline` swap):
 * plan 3.3.3 replaces hard nav locks with consultative/advisory gating, so a
 * step is always clickable — the dot communicates state, not permission.
 */
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import type { WorkspaceNavGroupSpec, WorkspaceNavItemSpec } from './types'

const props = defineProps<{
  groups: WorkspaceNavGroupSpec[]
}>()

const { t } = useI18n({ useScope: 'global' })

function itemAriaLabel(item: WorkspaceNavItemSpec): string {
  const label = t(item.labelKey)
  if (item.count !== undefined && item.count > 0) {
    return t('shellKit.workspaceNav.itemAriaLabelWithCount', { label, count: item.count })
  }
  return label
}
</script>

<template>
  <nav class="workspace-nav" :aria-label="t('shellKit.workspaceNav.rootAriaLabel')">
    <div v-if="$slots.header" class="workspace-nav__header">
      <slot name="header" />
    </div>

    <div class="workspace-nav__scroll">
      <div v-for="group in props.groups" :key="group.key" class="workspace-nav__group" role="group" :aria-label="group.labelKey ? t(group.labelKey) : undefined">
        <p v-if="group.labelKey" class="workspace-nav__group-label">{{ t(group.labelKey) }}</p>
        <ul role="list" class="workspace-nav__list">
          <li v-for="item in group.items" :key="item.key">
            <RouterLink :to="item.to" custom v-slot="{ href, navigate, isExactActive }">
              <a
                :href="href"
                class="workspace-nav__link"
                :class="{ 'workspace-nav__link--active': isExactActive, 'workspace-nav__link--disabled': item.disabled }"
                :aria-current="isExactActive ? 'page' : undefined"
                :aria-disabled="item.disabled ? 'true' : undefined"
                :aria-label="itemAriaLabel(item)"
                @click="item.disabled ? $event.preventDefault() : navigate($event)"
              >
                <span class="workspace-nav__icon" aria-hidden="true">
                  <component :is="item.icon" v-if="item.icon" />
                </span>
                <span class="workspace-nav__label">{{ t(item.labelKey) }}</span>
                <span
                  v-if="item.tone"
                  class="workspace-nav__dot"
                  :class="`workspace-nav__dot--${item.tone}`"
                  aria-hidden="true"
                />
                <span v-if="item.count !== undefined && item.count > 0" class="workspace-nav__count tabular-nums" aria-hidden="true">
                  {{ item.count }}
                </span>
              </a>
            </RouterLink>
          </li>
        </ul>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.workspace-nav {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: var(--nav-width);
  flex-shrink: 0;
  background: var(--nav-bg);
  border-right: 1px solid var(--color-border);
}

.workspace-nav__header {
  flex-shrink: 0;
  padding: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.workspace-nav__scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-2) var(--space-3) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.workspace-nav__group-label {
  margin: var(--space-2) var(--space-2) var(--space-1);
  font-family: var(--font-body);
  font-size: 0.6875rem;
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-muted);
}

.workspace-nav__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
}

.workspace-nav__link {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--nav-item-height);
  padding: 0 var(--space-3);
  border-radius: var(--nav-item-radius);
  color: var(--nav-item-fg);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  white-space: nowrap;
  overflow: hidden;
  min-width: 0;
  transition:
    background-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.workspace-nav__link:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.workspace-nav__link:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: -2px;
}

.workspace-nav__link--active {
  background: var(--nav-item-active-bg);
  color: var(--nav-item-active-fg);
  font-weight: var(--font-weight-semibold);
}

.workspace-nav__link--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.workspace-nav__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.workspace-nav__icon :deep(svg) {
  width: 18px;
  height: 18px;
}

.workspace-nav__label {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.workspace-nav__dot {
  flex-shrink: 0;
  width: var(--statusbadge-dot-size);
  height: var(--statusbadge-dot-size);
  border-radius: var(--radius-full);
}
.workspace-nav__dot--success {
  background: var(--status-success-fg);
}
.workspace-nav__dot--warning {
  background: var(--status-warning-fg);
}
.workspace-nav__dot--error {
  background: var(--status-error-fg);
}
.workspace-nav__dot--info {
  background: var(--status-info-fg);
}
.workspace-nav__dot--progress {
  background: var(--status-progress-fg);
}
.workspace-nav__dot--neutral {
  background: var(--status-neutral-fg);
}

.workspace-nav__count {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
