import { i18n } from '@/shared/i18n'
import type { DataGridDateFormat } from './types'

/**
 * Locale-aware date formatting for `DataGrid`'s `date` column type. A plain
 * function (not a composable) so it can be called freely from column
 * `render()` closures, which run outside any component's setup context.
 * Reads the active locale off the global i18n instance directly — same
 * pattern as `useGlossary()` in `@/shared/i18n/glossary`.
 */

const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

function toDate(value: Date | string | number): Date {
  return value instanceof Date ? value : new Date(value)
}

function formatRelative(date: Date, now: number): string {
  const diff = Math.max(0, now - date.getTime())

  if (diff < MINUTE_MS) return i18n.global.t('uiKit.dataGrid.relativeTime.justNow')
  if (diff < HOUR_MS) {
    return i18n.global.t('uiKit.dataGrid.relativeTime.minutesAgo', { n: Math.floor(diff / MINUTE_MS) })
  }
  if (diff < DAY_MS) {
    return i18n.global.t('uiKit.dataGrid.relativeTime.hoursAgo', { n: Math.floor(diff / HOUR_MS) })
  }
  if (diff < 7 * DAY_MS) {
    return i18n.global.t('uiKit.dataGrid.relativeTime.daysAgo', { n: Math.floor(diff / DAY_MS) })
  }

  return new Intl.DateTimeFormat(i18n.global.locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

/** Returns `''` for null/undefined/invalid input — callers render that as the empty placeholder. */
export function formatGridDate(
  value: Date | string | number | null | undefined,
  format: DataGridDateFormat = 'date',
  now: number = Date.now(),
): string {
  if (value == null || value === '') return ''

  const date = toDate(value)
  if (Number.isNaN(date.getTime())) return ''

  if (format === 'relative') return formatRelative(date, now)

  const locale = i18n.global.locale.value
  const options: Intl.DateTimeFormatOptions =
    format === 'datetime'
      ? { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
      : { year: 'numeric', month: '2-digit', day: '2-digit' }

  return new Intl.DateTimeFormat(locale, options).format(date)
}
