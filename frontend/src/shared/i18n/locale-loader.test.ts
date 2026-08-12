import { describe, expect, it, vi } from 'vitest'
import {
  createLocaleRuntime,
  type LocaleApplyReason,
  type LocaleMessageLoader,
  type LocaleMessages,
  type SupportedLocale,
} from './locale-loader'

const messages = {} as LocaleMessages

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function createHarness(
  initialLocale: SupportedLocale,
  loaders: Record<SupportedLocale, LocaleMessageLoader>,
) {
  const installs: SupportedLocale[] = []
  const applies: Array<{ locale: SupportedLocale; reason: LocaleApplyReason }> = []
  const errors: SupportedLocale[] = []
  const runtime = createLocaleRuntime({
    initialLocale,
    fallbackLocale: 'zh-CN',
    loaders,
    installMessages: (locale) => installs.push(locale),
    applyLocale: (locale, reason) => applies.push({ locale, reason }),
    reportLoadError: (locale) => errors.push(locale),
  })
  return { runtime, installs, applies, errors }
}

describe('locale runtime', () => {
  it('loads only the initial locale and keeps repeated setup/same-locale calls idempotent', async () => {
    const zhLoader = vi.fn(async () => messages)
    const enLoader = vi.fn(async () => messages)
    const { runtime, installs, applies } = createHarness('zh-CN', {
      'zh-CN': zhLoader,
      'en-US': enLoader,
    })

    await runtime.initialize()
    await runtime.initialize()
    await runtime.setLocale('zh-CN')

    expect(zhLoader).toHaveBeenCalledTimes(1)
    expect(enLoader).not.toHaveBeenCalled()
    expect(installs).toEqual(['zh-CN'])
    expect(applies).toEqual([{ locale: 'zh-CN', reason: 'initialize' }])
  })

  it('loads and applies the fallback when the persisted initial bundle fails', async () => {
    const { runtime, installs, applies, errors } = createHarness('en-US', {
      'zh-CN': async () => messages,
      'en-US': async () => Promise.reject(new Error('missing bundle')),
    })

    await expect(runtime.initialize()).resolves.toBe('zh-CN')

    expect(runtime.getCurrentLocale()).toBe('zh-CN')
    expect(installs).toEqual(['zh-CN'])
    expect(applies).toEqual([{ locale: 'zh-CN', reason: 'fallback' }])
    expect(errors).toEqual(['en-US'])
  })

  it('keeps the active locale and skips persistence callbacks when a switch fails', async () => {
    const { runtime, applies, errors } = createHarness('zh-CN', {
      'zh-CN': async () => messages,
      'en-US': async () => Promise.reject(new Error('missing bundle')),
    })
    await runtime.initialize()

    await expect(runtime.setLocale('en-US')).resolves.toBe(false)

    expect(runtime.getCurrentLocale()).toBe('zh-CN')
    expect(applies).toEqual([{ locale: 'zh-CN', reason: 'initialize' }])
    expect(errors).toEqual(['en-US'])
  })

  it('lets the latest rapid switch win even when an older load finishes later', async () => {
    const pendingEnglish = deferred<LocaleMessages>()
    const { runtime, applies } = createHarness('zh-CN', {
      'zh-CN': async () => messages,
      'en-US': () => pendingEnglish.promise,
    })
    await runtime.initialize()

    const staleSwitch = runtime.setLocale('en-US')
    const latestSwitch = runtime.setLocale('zh-CN')
    await expect(latestSwitch).resolves.toBe(true)
    pendingEnglish.resolve(messages)
    await expect(staleSwitch).resolves.toBe(false)

    expect(runtime.getCurrentLocale()).toBe('zh-CN')
    expect(applies).toEqual([{ locale: 'zh-CN', reason: 'initialize' }])
  })

  it('deduplicates concurrent requests for the same locale bundle', async () => {
    const pendingEnglish = deferred<LocaleMessages>()
    const enLoader = vi.fn(() => pendingEnglish.promise)
    const { runtime, installs, applies } = createHarness('zh-CN', {
      'zh-CN': async () => messages,
      'en-US': enLoader,
    })
    await runtime.initialize()

    const firstSwitch = runtime.setLocale('en-US')
    const latestSwitch = runtime.setLocale('en-US')
    pendingEnglish.resolve(messages)

    await expect(firstSwitch).resolves.toBe(false)
    await expect(latestSwitch).resolves.toBe(true)
    expect(enLoader).toHaveBeenCalledTimes(1)
    expect(installs).toEqual(['zh-CN', 'en-US'])
    expect(applies).toEqual([
      { locale: 'zh-CN', reason: 'initialize' },
      { locale: 'en-US', reason: 'switch' },
    ])
  })
})
