/**
 * Shared prop contracts for the shell/navigation kit (`SideNav` /
 * `WorkspaceNav`). Kept in one file so both nav levels describe items with
 * the same shape (icon + i18n label + optional badge/status), even though
 * they render at different visual weights.
 *
 * Every label is an i18n message KEY, not a resolved string — `SideNav` /
 * `WorkspaceNav` resolve it internally via `useI18n()` (same convention as
 * `FunnelBar`'s `labelKey`), so callers never hand-roll translated text and
 * the components stay reusable across locales without re-instantiation.
 */
import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import type { StatusTone } from '@/shared/i18n/glossary'

/** A count badge attached to a nav item. Rendered via `NavBadge`. */
export interface NavBadgeSpec {
  count: number
  tone?: StatusTone
}

/** One top-level navigation entry (plan 2.1's 6 sections + Settings). */
export interface NavItemSpec {
  key: string
  labelKey: string
  icon?: Component
  to: RouteLocationRaw
  badge?: NavBadgeSpec
  disabled?: boolean
}

/** A labeled (or label-less) cluster of top-level nav items. */
export interface NavGroupSpec {
  key: string
  /** Omit for an ungrouped cluster (no visible group heading). */
  labelKey?: string
  items: NavItemSpec[]
}

/**
 * One wave-workspace second-level nav entry (plan 3.3's step list). `tone`
 * drives a small status dot (advisory state, e.g. "blocked"/"ready") instead
 * of the old tree's fake padlock — plan 3.3.3 explicitly retires hard nav
 * locks in favor of consultative gating.
 */
export interface WorkspaceNavItemSpec {
  key: string
  labelKey: string
  icon?: Component
  to: RouteLocationRaw
  /** Omit for a step with no status signal yet (renders no dot). */
  tone?: StatusTone
  count?: number
  disabled?: boolean
}

/** A labeled (or label-less) cluster of workspace-nav items (e.g. "准备/审查/执行"). */
export interface WorkspaceNavGroupSpec {
  key: string
  labelKey?: string
  items: WorkspaceNavItemSpec[]
}
