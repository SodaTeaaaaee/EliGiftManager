import type { AppMessageSchema } from './locales/zh-CN'

export type SupportedLocale = 'zh-CN' | 'en-US'
export type LocaleMessages = AppMessageSchema

export const SUPPORTED_LOCALES: readonly SupportedLocale[] = ['zh-CN', 'en-US']

export type LocaleMessageLoader = () => Promise<LocaleMessages>

export type LocaleApplyReason = 'initialize' | 'fallback' | 'switch'

export interface LocaleRuntimeOptions {
  initialLocale: SupportedLocale
  fallbackLocale: SupportedLocale
  loaders?: Record<SupportedLocale, LocaleMessageLoader>
  installMessages: (locale: SupportedLocale, messages: LocaleMessages) => void
  applyLocale: (locale: SupportedLocale, reason: LocaleApplyReason) => void
  reportLoadError?: (locale: SupportedLocale, error: unknown) => void
}

export interface LocaleRuntime {
  initialize: () => Promise<SupportedLocale>
  setLocale: (locale: SupportedLocale) => Promise<boolean>
  getCurrentLocale: () => SupportedLocale
}

/**
 * Static import expressions are deliberately kept inside these functions.
 * Vite turns them into local, hashed chunks which Wails embeds with the rest
 * of `dist`; no locale is fetched from a network or evaluated dynamically.
 */
export const localeMessageLoaders: Record<SupportedLocale, LocaleMessageLoader> = {
  'zh-CN': () => import('./locales/zh-CN').then((module) => module.zhCN),
  'en-US': () => import('./locales/en-US').then((module) => module.enUS),
}

/**
 * Coordinates locale bundle loading independently of Vue/Pinia. Loads are
 * deduplicated, a failed switch leaves the current locale intact, and every
 * request invalidates older pending switches so the latest selection wins.
 */
export function createLocaleRuntime(options: LocaleRuntimeOptions): LocaleRuntime {
  const loaders = options.loaders ?? localeMessageLoaders
  const loaded = new Set<SupportedLocale>()
  const pending = new Map<SupportedLocale, Promise<void>>()

  let currentLocale = options.initialLocale
  let initialization: Promise<SupportedLocale> | null = null
  let initialized = false
  let switchSequence = 0

  function ensureMessages(locale: SupportedLocale): Promise<void> {
    if (loaded.has(locale)) return Promise.resolve()

    const existing = pending.get(locale)
    if (existing) return existing

    const request = Promise.resolve()
      .then(() => loaders[locale]())
      .then((messages) => {
        options.installMessages(locale, messages)
        loaded.add(locale)
      })
      .finally(() => {
        if (pending.get(locale) === request) {
          pending.delete(locale)
        }
      })

    pending.set(locale, request)
    return request
  }

  async function performInitialization(): Promise<SupportedLocale> {
    let resolvedLocale = options.initialLocale
    let reason: LocaleApplyReason = 'initialize'

    try {
      await ensureMessages(resolvedLocale)
    } catch (error) {
      options.reportLoadError?.(resolvedLocale, error)
      if (resolvedLocale === options.fallbackLocale) throw error

      resolvedLocale = options.fallbackLocale
      reason = 'fallback'
      try {
        await ensureMessages(resolvedLocale)
      } catch (fallbackError) {
        options.reportLoadError?.(resolvedLocale, fallbackError)
        throw fallbackError
      }
    }

    currentLocale = resolvedLocale
    initialized = true
    options.applyLocale(resolvedLocale, reason)
    return resolvedLocale
  }

  function initialize(): Promise<SupportedLocale> {
    if (initialized) return Promise.resolve(currentLocale)
    if (initialization) return initialization

    initialization = performInitialization().catch((error) => {
      initialization = null
      throw error
    })
    return initialization
  }

  async function setLocale(locale: SupportedLocale): Promise<boolean> {
    await initialize()

    const requestSequence = ++switchSequence
    if (locale === currentLocale) return true

    try {
      await ensureMessages(locale)
    } catch (error) {
      if (requestSequence === switchSequence) {
        options.reportLoadError?.(locale, error)
      }
      return false
    }

    if (requestSequence !== switchSequence) return false

    currentLocale = locale
    options.applyLocale(locale, 'switch')
    return true
  }

  return {
    initialize,
    setLocale,
    getCurrentLocale: () => currentLocale,
  }
}
