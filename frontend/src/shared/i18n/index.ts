/**
 * vue-i18n setup — the single entry point for the app's locale system.
 *
 * - `i18n`: the vue-i18n instance (composition/`legacy: false` mode).
 * - `setupI18n(app)`: loads the initial local message bundle, installs `i18n`,
 *   and syncs `document.documentElement.lang` before the app mounts.
 * - `useLocaleStore()` / `useAppLocale()`: the Pinia-backed locale store,
 *   persisted under the `eligiftmanager:locale` localStorage key, falling
 *   back to the browser's language on first run.
 */
import type { App } from 'vue'
import { computed, ref } from 'vue'
import { defineStore, storeToRefs } from 'pinia'
import { createI18n } from 'vue-i18n'
import {
  createLocaleRuntime,
  SUPPORTED_LOCALES,
  type LocaleApplyReason,
  type SupportedLocale,
} from './locale-loader'

export { SUPPORTED_LOCALES, type SupportedLocale } from './locale-loader'

export interface LocaleOption {
  label: string
  value: SupportedLocale
}

const LOCALE_STORAGE_KEY = 'eligiftmanager:locale'

const localeLabelKeys: Record<SupportedLocale, string> = {
  'zh-CN': 'common.locales.zhCN',
  'en-US': 'common.locales.enUS',
}

function isSupportedLocale(value: string | null): value is SupportedLocale {
  return value === 'zh-CN' || value === 'en-US'
}

function persistLocale(value: SupportedLocale): void {
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, value)
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('[i18n] failed to persist locale', error)
    }
  }
}

function readPersistedLocale(): SupportedLocale | null {
  if (typeof window === 'undefined') return null

  try {
    const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY)
    return isSupportedLocale(stored) ? stored : null
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('[i18n] failed to read persisted locale', error)
    }
    return null
  }
}

/** Best-effort locale guess from `navigator.languages` / `navigator.language`. */
function resolveBrowserLocale(): SupportedLocale {
  if (typeof navigator === 'undefined') return 'zh-CN'

  const langs = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const lang of langs) {
    if (!lang) continue

    const lower = lang.toLowerCase()
    if (lower.startsWith('zh')) return 'zh-CN'
    if (lower.startsWith('en')) return 'en-US'
  }

  return 'zh-CN'
}

function resolveInitialLocale(): SupportedLocale {
  return readPersistedLocale() ?? resolveBrowserLocale()
}

function applyDocumentLang(value: SupportedLocale): void {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = value
  }
}

const initialLocale = resolveInitialLocale()

/** The shared vue-i18n instance. Messages are installed before app mount. */
export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'zh-CN',
  messages: {},
  // We report missing keys ourselves via `missing` below; avoid vue-i18n's
  // own console noise duplicating it.
  missingWarn: false,
  fallbackWarn: false,
  missing: (locale, key) => {
    if (import.meta.env.DEV) {
      console.warn(`[i18n] missing message key "${key}" for locale "${locale}"`)
    }
    return key
  },
})

const activeLocale = ref<SupportedLocale>(initialLocale)

function applyLocale(value: SupportedLocale, reason: LocaleApplyReason): void {
  activeLocale.value = value
  i18n.global.locale.value = value
  applyDocumentLang(value)

  // Normal bootstrap preserves the existing browser/persisted resolution.
  // Explicit switches and a successful startup fallback record the locale
  // that was actually applied, never a bundle that failed to load.
  if (reason !== 'initialize') persistLocale(value)
}

const localeRuntime = createLocaleRuntime({
  initialLocale,
  fallbackLocale: 'zh-CN',
  installMessages: (locale, messages) => {
    i18n.global.setLocaleMessage(locale, messages)
  },
  applyLocale,
  reportLoadError: (locale, error) => {
    if (import.meta.env.DEV) {
      console.warn(`[i18n] failed to load locale bundle "${locale}"`, error)
    }
  },
})

/**
 * Pinia store holding the active locale. A switch commits only after its
 * embedded message bundle loads; callers may await the boolean result.
 */
export const useLocaleStore = defineStore('locale', () => {
  async function setLocale(value: SupportedLocale): Promise<boolean> {
    return localeRuntime.setLocale(value)
  }

  return { locale: activeLocale, setLocale }
})

/**
 * Loads the initial message bundle and installs vue-i18n before app mount, so
 * route components never render untranslated keys or flash another language.
 */
export async function setupI18n(app: App): Promise<void> {
  await localeRuntime.initialize()
  app.use(i18n)
}

/**
 * The public composable for reading/switching the active locale.
 * Prefer this over reaching into `useLocaleStore()` directly.
 */
export function useAppLocale() {
  const store = useLocaleStore()
  const { locale } = storeToRefs(store)
  const localeOptions = computed<LocaleOption[]>(() =>
    SUPPORTED_LOCALES.map((value) => ({
      label: i18n.global.t(localeLabelKeys[value]),
      value,
    })),
  )
  const localeLabel = computed(() => i18n.global.t(localeLabelKeys[locale.value]))

  return {
    locale,
    localeLabel,
    localeOptions,
    setLocale: store.setLocale,
  }
}
