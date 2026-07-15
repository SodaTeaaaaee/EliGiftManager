/**
 * vue-i18n setup — the single entry point for the app's locale system.
 *
 * - `i18n`: the vue-i18n instance (composition/`legacy: false` mode).
 * - `setupI18n(app)`: installs `i18n` on the Vue app and syncs
 *   `document.documentElement.lang` to the resolved initial locale.
 * - `useLocaleStore()` / `useAppLocale()`: the Pinia-backed locale store,
 *   persisted under the `eligiftmanager:locale` localStorage key, falling
 *   back to the browser's language on first run.
 */
import type { App } from 'vue'
import { computed, ref } from 'vue'
import { defineStore, storeToRefs } from 'pinia'
import { createI18n } from 'vue-i18n'
import { zhCN } from './locales/zh-CN'
import { enUS } from './locales/en-US'

export type SupportedLocale = 'zh-CN' | 'en-US'

export interface LocaleOption {
  label: string
  value: SupportedLocale
}

export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['zh-CN', 'en-US']

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

/** The shared vue-i18n instance. `legacy: false` = Composition API mode throughout. */
export const i18n = createI18n({
  legacy: false,
  locale: resolveInitialLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
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

function getCurrentLocale(): SupportedLocale {
  const value = i18n.global.locale.value
  return isSupportedLocale(value) ? value : 'zh-CN'
}

/**
 * Pinia store holding the active locale. Call `setLocale` to switch — it
 * persists to localStorage, updates `document.documentElement.lang`, and
 * flips the live vue-i18n locale in one step.
 */
export const useLocaleStore = defineStore('locale', () => {
  const locale = ref<SupportedLocale>(getCurrentLocale())

  function setLocale(value: SupportedLocale): void {
    locale.value = value
    persistLocale(value)
    applyDocumentLang(value)
    i18n.global.locale.value = value
  }

  return { locale, setLocale }
})

/**
 * Installs vue-i18n on the app and syncs the initial `<html lang>` attribute.
 * Call once during bootstrap (see `main.ts`'s `[P0-INTEGRATION]` marker),
 * before `app.mount(...)`.
 */
export function setupI18n(app: App): void {
  app.use(i18n)
  applyDocumentLang(getCurrentLocale())
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
