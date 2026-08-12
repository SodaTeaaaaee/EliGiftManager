import { describe, expect, it } from 'vitest'
import i18nSource from './index.ts?raw'
import localeLoaderSource from './locale-loader.ts?raw'

describe('locale lazy-loading contract', () => {
  it('does not statically import a locale bundle into the application entry graph', () => {
    const staticLocaleImport = /^\s*import\s+(?!type\b)[^\n]*['"]\.\/locales\//m

    expect(i18nSource).not.toMatch(staticLocaleImport)
    expect(localeLoaderSource).not.toMatch(staticLocaleImport)
    expect(localeLoaderSource).toContain("import('./locales/zh-CN')")
    expect(localeLoaderSource).toContain("import('./locales/en-US')")
  })
})
