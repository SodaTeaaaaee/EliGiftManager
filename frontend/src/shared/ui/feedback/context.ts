import { inject, type InjectionKey } from 'vue'
import type { FeedbackApi } from './types'

export const FEEDBACK_INJECTION_KEY: InjectionKey<FeedbackApi> = Symbol('eligiftmanager:feedback-api')

/**
 * The feedback kit's only public composable. Must be called from a
 * component subtree mounted under `<FeedbackProvider>` — it is the app's
 * ONLY feedback path (no direct `NMessage`/`NNotification` usage).
 *
 * Throws a descriptive error outside a provider, rather than silently
 * no-op'ing, so a missing `<FeedbackProvider>` fails loudly in dev instead
 * of quietly swallowing toasts/receipts.
 */
export function useFeedback(): FeedbackApi {
  const api = inject(FEEDBACK_INJECTION_KEY, null)
  if (!api) {
    throw new Error(
      '[useFeedback] no <FeedbackProvider> found in the component tree. ' +
        'Mount one near the app root before calling useFeedback().',
    )
  }
  return api
}
