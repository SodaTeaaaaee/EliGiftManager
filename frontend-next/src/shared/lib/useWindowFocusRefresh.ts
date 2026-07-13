/**
 * `useWindowFocusRefresh` — runs an async callback whenever the window
 * regains focus (and, optionally, whenever the document becomes visible
 * again via the Page Visibility API — useful for the Wails desktop shell,
 * where minimizing/restoring doesn't always fire a `focus` event on every
 * platform). No precedent existed for this pattern before plan 3.1 (task
 * center "刷新任务中心" on window re-focus), so this is a clean, reusable
 * composable — any future page needing focus-triggered refresh should use
 * this rather than re-wiring its own listeners.
 *
 * Registration/cleanup happens in `onMounted`/`onBeforeUnmount` so it is
 * safe to call from any `<script setup>` component.
 */
import { onBeforeUnmount, onMounted } from 'vue'

export interface UseWindowFocusRefreshOptions {
  /** Also refresh on `document.visibilitychange` -> `visible`. Default `true`. */
  refreshOnVisibilityChange?: boolean
}

export function useWindowFocusRefresh(
  callback: () => void | Promise<void>,
  options: UseWindowFocusRefreshOptions = {},
): void {
  const { refreshOnVisibilityChange = true } = options

  function handleFocus(): void {
    void callback()
  }

  function handleVisibilityChange(): void {
    if (document.visibilityState === 'visible') {
      void callback()
    }
  }

  onMounted(() => {
    window.addEventListener('focus', handleFocus)
    if (refreshOnVisibilityChange) {
      document.addEventListener('visibilitychange', handleVisibilityChange)
    }
  })

  onBeforeUnmount(() => {
    window.removeEventListener('focus', handleFocus)
    if (refreshOnVisibilityChange) {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  })
}
