/**
 * useRouteProgressStore — reactive state for the top route-transition
 * progress bar (plan §4.5). Driven entirely by `app/router/index.ts`'s
 * `beforeEach`/`afterEach`/`onError` guards; rendered by
 * `shared/ui/feedback/TopProgressBar.vue`. A Pinia store (not a plain
 * module-level ref) so it's safely readable/writable both from a component
 * (`<script setup>`) and from the router guard callbacks, which run outside
 * any component context.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'

/** How long the bar stays at 100% before fading out (ms) — long enough to read as a deliberate "done" snap, short enough not to feel laggy. */
const FINISH_HOLD_MS = 200

export const useRouteProgressStore = defineStore('route-progress', () => {
  const progress = ref(0)
  const visible = ref(false)

  // Counts overlapping navigations (e.g. a redirect chain fires `beforeEach`
  // again before the previous navigation's `afterEach`/`onError` has run) so
  // the bar only fully finishes once every in-flight navigation has settled.
  let pending = 0
  let hideTimer: ReturnType<typeof setTimeout> | undefined

  function start(): void {
    pending += 1
    if (hideTimer !== undefined) {
      clearTimeout(hideTimer)
      hideTimer = undefined
    }
    if (pending > 1) return
    progress.value = 0
    visible.value = true
    // Next frame: bump toward ~90% so the CSS width transition animates the
    // fill for the duration of the (usually short) lazy-chunk load, without
    // ever visually completing before the navigation actually resolves.
    requestAnimationFrame(() => {
      progress.value = 90
    })
  }

  function finish(): void {
    pending = Math.max(0, pending - 1)
    if (pending > 0) return
    progress.value = 100
    hideTimer = setTimeout(() => {
      visible.value = false
      progress.value = 0
      hideTimer = undefined
    }, FINISH_HOLD_MS)
  }

  return { progress, visible, start, finish }
})
