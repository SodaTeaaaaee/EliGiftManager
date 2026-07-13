/**
 * view-hotkeys — merged global keydown handler for two desktop-shell
 * hotkeys (plan §4.5):
 *
 * - Ctrl+F / Cmd+F -> focuses the ACTIVE view's FilterBar keyword input.
 *   `FilterBar.vue` calls `registerFilterFocusTarget()` (only when its
 *   schema has a keyword field) and unregisters on unmount.
 * - F5, Ctrl+R / Cmd+R -> re-runs the ACTIVE view's own data refresh in
 *   place, instead of letting the Wails webview do a native full-page
 *   reload (which would blow away all app state — undo history, URL
 *   filters restored from memory, in-flight forms, everything). A small,
 *   fixed set of primary list/grid surfaces call `registerRefreshTarget()`
 *   with their EXISTING refresh function. If nothing is registered, the
 *   keydown is still `preventDefault()`-ed and otherwise no-ops — the
 *   destructive native reload must never run.
 *
 * Both hotkeys share ONE `window` keydown listener (installed once via
 * `useGlobalViewHotkeys()`, called from `App.vue`) rather than one listener
 * per page — adding a new FilterBar/refreshable view is just a
 * register/unregister call, no new listener wiring.
 *
 * REGISTRATION TIMING — read before adding a new call site:
 * Register from `onBeforeMount`, NOT `onMounted`. Vue's `mounted` hook
 * fires child-before-parent, but `beforeMount` fires parent-before-child.
 * Both stacks below are last-registered-wins ("topmost" = most specific /
 * currently active view). Consider the wave workspace: `WaveWorkspaceShell`
 * (parent) registers a workspace-wide fallback refresh; `WaveLinesTab`
 * (child, rendered inside the shell's `<RouterView>`) registers a more
 * specific grid refresh. On a plain tab switch the shell is already
 * mounted, so ordering isn't an issue either way — but on a fresh direct
 * deep link (e.g. opening `/waves/1/lines` straight away) BOTH mount in the
 * same tick, and Vue mounts children before parents. With `onMounted`, the
 * child (grid) would register first and the parent (shell) would register
 * SECOND, landing on top of the stack — backwards, since the shell's
 * generic fallback would then incorrectly win over the tab's specific
 * refresh. `onBeforeMount` fires parent-first, so the shell always
 * registers before the child regardless of mount timing — the child always
 * ends up on top, which is what "most specific view wins" requires. Pair
 * every registration with an `onBeforeUnmount` unregister call (unmount
 * order doesn't matter — unregister splices by reference, not position).
 */
import { onBeforeUnmount, onMounted } from 'vue'

type Unregister = () => void

const filterFocusTargets: Array<() => void> = []
const refreshTargets: Array<() => void | Promise<void>> = []

function pushTarget<T>(stack: T[], target: T): Unregister {
  stack.push(target)
  return () => {
    const index = stack.indexOf(target)
    if (index !== -1) stack.splice(index, 1)
  }
}

/** Registers a callback that focuses this view's filter keyword input. Call from `onBeforeMount`; call the returned fn from `onBeforeUnmount`. */
export function registerFilterFocusTarget(focus: () => void): Unregister {
  return pushTarget(filterFocusTargets, focus)
}

/** Registers this view's existing data-refresh function. Call from `onBeforeMount`; call the returned fn from `onBeforeUnmount`. */
export function registerRefreshTarget(refresh: () => void | Promise<void>): Unregister {
  return pushTarget(refreshTargets, refresh)
}

function isMacPlatform(): boolean {
  return /Mac|iPod|iPhone|iPad/.test(navigator.platform)
}

function handleGlobalKeydown(event: KeyboardEvent): void {
  // F5 has no modifier requirement and must always be caught — check first.
  if (event.key === 'F5') {
    event.preventDefault()
    void refreshTargets[refreshTargets.length - 1]?.()
    return
  }

  const primaryModifier = isMacPlatform() ? event.metaKey : event.ctrlKey
  if (!primaryModifier || event.altKey || event.shiftKey) return

  const key = event.key.toLowerCase()

  if (key === 'f') {
    event.preventDefault()
    filterFocusTargets[filterFocusTargets.length - 1]?.()
    return
  }

  if (key === 'r') {
    event.preventDefault()
    void refreshTargets[refreshTargets.length - 1]?.()
  }
}

/**
 * Installs the single global keydown listener (capture phase — guarantees
 * interception even if a focused input's own keydown handler would
 * otherwise stop propagation before a bubble-phase window listener saw it).
 * Call exactly once, from `App.vue`'s `<script setup>` top level.
 */
export function useGlobalViewHotkeys(): void {
  onMounted(() => window.addEventListener('keydown', handleGlobalKeydown, true))
  onBeforeUnmount(() => window.removeEventListener('keydown', handleGlobalKeydown, true))
}
