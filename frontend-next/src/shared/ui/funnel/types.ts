import type { StatusTone } from '@/shared/i18n/glossary'

/**
 * `FunnelStage` — extracted from `FunnelBar.vue` into a plain `.ts` module.
 * vue-tsc's ambient `declare module "*.vue"` shim (see `src/vite-env.d.ts`)
 * only declares a default export, so `import type { FunnelStage } from
 * './FunnelBar.vue'` resolves against that wildcard shim instead of the
 * component's real compiled types and fails to find the named export. A
 * plain co-located module is the standard fix and keeps the public API
 * (`import type { FunnelStage } from '@/shared/ui/funnel'`) unchanged.
 */
export interface FunnelStage {
  key: string
  /** i18n message key resolving to the stage's display label. */
  labelKey: string
  count: number
  tone?: StatusTone
}
