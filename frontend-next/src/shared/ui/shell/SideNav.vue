<script setup lang="ts">
/**
 * SideNav — the top-level app nav (plan 2.1 / 4.3). Token-driven, custom
 * (NOT `NMenu`): brand area with a skin-decoration slot, grouped items with
 * icon + i18n label + optional count badge, active state derived from the
 * live route via `RouterLink`, collapse to a 56–64px icon rail with
 * persisted state, and Settings pinned at the bottom INSIDE this same
 * component — fixing the old tree's two-`NMenu` hack
 * (`frontend/src/shared/ui/AppSidebar.vue`'s separate footer `<n-menu>`).
 *
 * Keyboard model: every item is a native `<RouterLink>` (real anchor), so
 * Tab/Shift+Tab and Enter/Space work with zero extra wiring. ArrowUp/
 * ArrowDown/Home/End additionally roam focus across all enabled items
 * (including the pinned Settings entry) for fast keyboard-only travel,
 * matching common nav/menu expectations without reimplementing full ARIA
 * `menu` semantics (this is a navigation list, not a `menu` widget — plain
 * `nav` + `RouterLink`s is the correct, simpler ARIA shape here).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import { useSideNavStore } from './nav-state'
import NavBadge from './NavBadge.vue'
import type { NavGroupSpec, NavItemSpec } from './types'

const props = defineProps<{
  groups: NavGroupSpec[]
  /** Rendered pinned at the bottom, inside this same nav — never a second menu. */
  settingsItem?: NavItemSpec
}>()

const { t } = useI18n({ useScope: 'global' })
const sideNav = useSideNavStore()

const navRef = ref<HTMLElement | null>(null)

function itemAriaLabel(item: NavItemSpec): string {
  const label = t(item.labelKey)
  return item.badge && item.badge.count > 0
    ? t('shellKit.sideNav.itemAriaLabelWithCount', { label, count: item.badge.count })
    : label
}

const toggleLabel = computed(() =>
  sideNav.collapsed ? t('shellKit.sideNav.expand') : t('shellKit.sideNav.collapse'),
)

function focusableLinks(): HTMLAnchorElement[] {
  const root = navRef.value
  if (!root) return []
  return Array.from(root.querySelectorAll<HTMLAnchorElement>('a.side-nav__link:not([aria-disabled="true"])'))
}

function handleRoving(event: KeyboardEvent): void {
  const keys = ['ArrowUp', 'ArrowDown', 'Home', 'End']
  if (!keys.includes(event.key)) return
  const links = focusableLinks()
  if (links.length === 0) return
  const currentIndex = links.indexOf(document.activeElement as HTMLAnchorElement)
  let nextIndex = currentIndex
  if (event.key === 'ArrowDown') nextIndex = currentIndex < 0 ? 0 : (currentIndex + 1) % links.length
  else if (event.key === 'ArrowUp') nextIndex = currentIndex < 0 ? links.length - 1 : (currentIndex - 1 + links.length) % links.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = links.length - 1
  event.preventDefault()
  links[nextIndex]?.focus()
}
</script>

<template>
  <nav
    ref="navRef"
    class="side-nav"
    :class="{ 'side-nav--collapsed': sideNav.collapsed }"
    :aria-label="t('shellKit.sideNav.rootAriaLabel')"
    @keydown="handleRoving"
  >
    <div class="side-nav__brand">
      <slot name="brand">
        <span class="side-nav__brand-mark" aria-hidden="true">EG</span>
        <span v-if="!sideNav.collapsed" class="side-nav__brand-text">{{ t('shellKit.sideNav.brandName') }}</span>
      </slot>
    </div>

    <div class="side-nav__scroll">
      <div v-for="group in props.groups" :key="group.key" class="side-nav__group" role="group" :aria-label="group.labelKey ? t(group.labelKey) : undefined">
        <p v-if="group.labelKey && !sideNav.collapsed" class="side-nav__group-label">{{ t(group.labelKey) }}</p>
        <ul role="list" class="side-nav__list">
          <li v-for="item in group.items" :key="item.key">
            <RouterLink
              :to="item.to"
              custom
              v-slot="{ href, navigate, isExactActive }"
            >
              <a
                :href="href"
                class="side-nav__link"
                :class="{ 'side-nav__link--active': isExactActive, 'side-nav__link--disabled': item.disabled }"
                :aria-current="isExactActive ? 'page' : undefined"
                :aria-disabled="item.disabled ? 'true' : undefined"
                :aria-label="itemAriaLabel(item)"
                :title="sideNav.collapsed ? t(item.labelKey) : undefined"
                @click="item.disabled ? $event.preventDefault() : navigate($event)"
              >
                <span class="side-nav__icon" aria-hidden="true">
                  <component :is="item.icon" v-if="item.icon" />
                </span>
                <span v-if="!sideNav.collapsed" class="side-nav__label">{{ t(item.labelKey) }}</span>
                <NavBadge
                  v-if="item.badge && item.badge.count > 0"
                  class="side-nav__badge"
                  :count="item.badge.count"
                  :tone="item.badge.tone"
                />
              </a>
            </RouterLink>
          </li>
        </ul>
      </div>
    </div>

    <div class="side-nav__footer">
      <RouterLink v-if="props.settingsItem" :to="props.settingsItem.to" custom v-slot="{ href, navigate, isExactActive }">
        <a
          :href="href"
          class="side-nav__link side-nav__link--settings"
          :class="{ 'side-nav__link--active': isExactActive }"
          :aria-current="isExactActive ? 'page' : undefined"
          :aria-label="itemAriaLabel(props.settingsItem)"
          :title="sideNav.collapsed ? t(props.settingsItem.labelKey) : undefined"
          @click="navigate($event)"
        >
          <span class="side-nav__icon" aria-hidden="true">
            <component :is="props.settingsItem.icon" v-if="props.settingsItem.icon" />
          </span>
          <span v-if="!sideNav.collapsed" class="side-nav__label">{{ t(props.settingsItem.labelKey) }}</span>
        </a>
      </RouterLink>

      <button
        type="button"
        class="side-nav__collapse-toggle"
        :aria-label="toggleLabel"
        :aria-pressed="sideNav.collapsed"
        @click="sideNav.toggle()"
      >
        <ChevronBackOutline v-if="!sideNav.collapsed" class="side-nav__collapse-icon" />
        <ChevronForwardOutline v-else class="side-nav__collapse-icon" />
        <span v-if="!sideNav.collapsed" class="side-nav__collapse-text">{{ toggleLabel }}</span>
      </button>
    </div>
  </nav>
</template>

<style scoped>
.side-nav {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: var(--nav-width);
  flex-shrink: 0;
  background: var(--nav-bg);
  border-right: 1px solid var(--color-border);
  transition: width var(--duration-base) var(--ease-out);
  overflow: hidden;
}

.side-nav--collapsed {
  width: var(--nav-width-collapsed);
}

.side-nav__brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-5) var(--space-4) var(--space-3);
  flex-shrink: 0;
  min-width: 0;
}

.side-nav__brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: var(--radius-md);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
}

.side-nav__brand-text {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.side-nav__scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.side-nav__group-label {
  margin: var(--space-2) var(--space-2) var(--space-1);
  font-family: var(--font-body);
  font-size: 0.6875rem;
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.side-nav__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
}

.side-nav__link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
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

.side-nav--collapsed .side-nav__link {
  justify-content: center;
  padding: 0;
}

.side-nav__link:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.side-nav__link:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: -2px;
}

.side-nav__link--active {
  background: var(--nav-item-active-bg);
  color: var(--nav-item-active-fg);
  font-weight: var(--font-weight-semibold);
}

.side-nav__link--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.side-nav__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.side-nav__icon :deep(svg) {
  width: 20px;
  height: 20px;
}

.side-nav__label {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.side-nav__badge {
  flex-shrink: 0;
}

.side-nav__footer {
  flex-shrink: 0;
  border-top: 1px solid var(--color-border);
  padding: var(--space-2) var(--space-3) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.side-nav__collapse-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  height: var(--nav-item-height);
  padding: 0 var(--space-3);
  border-radius: var(--nav-item-radius);
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition:
    background-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.side-nav--collapsed .side-nav__collapse-toggle {
  justify-content: center;
  padding: 0;
}

.side-nav__collapse-toggle:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.side-nav__collapse-toggle:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: -2px;
}

.side-nav__collapse-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.side-nav__collapse-text {
  white-space: nowrap;
}
</style>
