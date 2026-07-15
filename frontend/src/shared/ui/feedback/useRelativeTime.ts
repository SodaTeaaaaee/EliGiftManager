import { useI18n } from 'vue-i18n'

const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

/**
 * Locale-aware "N minutes ago" style formatting for the receipt tray.
 * Buckets: <60s -> "just now", <60min -> minutes, <24h -> hours,
 * otherwise -> days, falling back to a locale-formatted absolute date/time
 * for anything older than a week (receipts are session-scoped and capped at
 * 12 entries, so this is a defensive fallback rather than the common case).
 */
export function useRelativeTime() {
  const { t, locale } = useI18n()

  function format(at: number, now: number = Date.now()): string {
    const diff = Math.max(0, now - at)

    if (diff < MINUTE_MS) return t('feedback.time.justNow')
    if (diff < HOUR_MS) return t('feedback.time.minutesAgo', { n: Math.floor(diff / MINUTE_MS) })
    if (diff < DAY_MS) return t('feedback.time.hoursAgo', { n: Math.floor(diff / HOUR_MS) })
    if (diff < 7 * DAY_MS) return t('feedback.time.daysAgo', { n: Math.floor(diff / DAY_MS) })

    return new Intl.DateTimeFormat(locale.value, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(at))
  }

  return { format }
}
